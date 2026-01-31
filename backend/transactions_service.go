package cashtrack

import (
	apiv1 "cashtrack/backend/gen/api/v1"
	db "cashtrack/backend/gen/db"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TransactionsService struct {
	db            *Db
	exchangeRates *ExchangeRateService
}

type TransactionFilters struct {
	FromDate            *time.Time
	ToDate              *time.Time
	SourceFileID        *int64
	EntryType           string
	SearchText          string
	SourceAccountNumber string
	SourceCardNumber    string
	CategoryID          *int64
	Limit               int
	Offset              int
}

func NewTransactionsService(db *Db) *TransactionsService {
	return &TransactionsService{
		db:            db,
		exchangeRates: NewExchangeRateService(db),
	}
}

func (s *TransactionsService) ReplaceForSourceTx(ctx context.Context, tx pgx.Tx, userID int32, sourceFileID int64, entries []ParsedTransaction) error {
	txQueries := s.db.Queries.WithTx(tx)
	err := txQueries.DeleteTransactionsBySource(ctx, db.DeleteTransactionsBySourceParams{
		SourceFileID: sourceFileID,
		UserID:       userID,
	})
	if err != nil {
		return fmt.Errorf("delete transactions: %w", err)
	}

	if len(entries) == 0 {
		return nil
	}

	rules, err := s.listCategoryRules(ctx, userID)
	if err != nil {
		return fmt.Errorf("load category rules: %w", err)
	}
	normalizedRules := normalizeRules(rules)

	for _, entry := range entries {
		var meta json.RawMessage
		if entry.ParserMeta != nil {
			payload, err := json.Marshal(entry.ParserMeta)
			if err != nil {
				return fmt.Errorf("encode parser meta: %w", err)
			}
			meta = payload
		}

		amount, err := numericFromString(entry.Amount)
		if err != nil {
			return fmt.Errorf("parse amount %q: %w", entry.Amount, err)
		}

		categoryID := categoryIDFromDescription(entry.Description, normalizedRules)
		categorySource := pgtype.Text{}
		if categoryID.Valid {
			categorySource = pgtype.Text{String: categorySourceRule, Valid: true}
		}

		err = txQueries.CreateTransaction(ctx, db.CreateTransactionParams{
			UserID:              userID,
			SourceFileID:        sourceFileID,
			SourceFileRow:       int32(entry.SourceFileRow),
			ParserName:          entry.ParserName,
			PostedDate:          pgtype.Date{Time: entry.PostedDate, Valid: true},
			Description:         entry.Description,
			Amount:              amount,
			Currency:            entry.Currency,
			TransactionID:       nullableText(entry.TransactionID),
			EntryType:           entry.EntryType,
			SourceAccountNumber: nullableText(entry.SourceAccountNumber),
			SourceCardNumber:    nullableText(entry.SourceCardNumber),
			CategoryID:          categoryID,
			CategorySource:      categorySource,
			ParserMeta:          meta,
		})
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	return nil
}

func (s *TransactionsService) List(ctx context.Context, userID int32, filters TransactionFilters) ([]*apiv1.Transaction, error) {
	params := db.ListTransactionsParams{
		UserID:              userID,
		FromDate:            dateOrNull(filters.FromDate),
		ToDate:              dateOrNull(filters.ToDate),
		SourceFileID:        int64OrNull(filters.SourceFileID),
		EntryType:           textOrNull(filters.EntryType),
		SourceAccountNumber: textOrNull(filters.SourceAccountNumber),
		SourceCardNumber:    textOrNull(filters.SourceCardNumber),
		SearchText:          textOrNull(filters.SearchText),
		CategoryID:          int64OrNull(filters.CategoryID),
		LimitCount:          int32OrDefault(filters.Limit, 500),
		OffsetCount:         int32OrDefault(filters.Offset, 0),
	}

	rows, err := s.db.Queries.ListTransactions(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}

	entries := make([]*apiv1.Transaction, 0)
	for _, row := range rows {
		var categoryID *int32
		if row.CategoryID.Valid {
			value := int32(row.CategoryID.Int64)
			categoryID = &value
		}
		postedDate := ""
		if row.PostedDate.Valid {
			postedDate = row.PostedDate.Time.Format(time.RFC3339Nano)
		}
		createdAt := ""
		if row.CreatedAt.Valid {
			createdAt = row.CreatedAt.Time.Format(time.RFC3339Nano)
		}
		amountCents, err := numericToCents(row.Amount)
		if err != nil {
			return nil, fmt.Errorf("convert amount: %w", err)
		}
		entry := &apiv1.Transaction{
			Id:                  int32(row.ID),
			SourceFileId:        int32(row.SourceFileID),
			SourceFileRow:       row.SourceFileRow,
			ParserName:          row.ParserName,
			PostedDate:          postedDate,
			Description:         row.Description,
			Amount:              amountCents,
			Currency:            row.Currency,
			TransactionId:       row.TransactionID.String,
			EntryType:           row.EntryType,
			SourceAccountNumber: row.SourceAccountNumber.String,
			SourceCardNumber:    row.SourceCardNumber.String,
			CreatedAt:           createdAt,
			CategoryId:          categoryID,
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

type CategoryRuleEntry struct {
	CategoryID          int64
	DescriptionContains string
}

func (s *TransactionsService) Summary(ctx context.Context, userID int32, filters TransactionFilters) (*apiv1.TransactionSummary, error) {
	rows, err := s.db.Queries.ListTransactionsSummaryRows(ctx, db.ListTransactionsSummaryRowsParams{
		UserID:              userID,
		FromDate:            dateOrNull(filters.FromDate),
		ToDate:              dateOrNull(filters.ToDate),
		SourceFileID:        int64OrNull(filters.SourceFileID),
		EntryType:           textOrNull(filters.EntryType),
		SourceAccountNumber: textOrNull(filters.SourceAccountNumber),
		SourceCardNumber:    textOrNull(filters.SourceCardNumber),
		SearchText:          textOrNull(filters.SearchText),
		CategoryID:          int64OrNull(filters.CategoryID),
	})
	if err != nil {
		log.Error().Err(err).Interface("filters", filters).Msg("failed to query transactions summary")
		return nil, err
	}

	if len(rows) == 0 {
		return &apiv1.TransactionSummary{
			Count:          0,
			Total:          0,
			Average:        0,
			Median:         0,
			Currency:       "CHF",
			UniqueAccounts: 0,
			DateRangeStart: "",
			DateRangeEnd:   "",
		}, nil
	}

	debitAmounts := make([]float64, 0, len(rows))
	var total float64
	var debitTotal float64
	uniqueAccounts := make(map[string]struct{})
	var minDate time.Time
	var maxDate time.Time
	hasDate := false
	for _, row := range rows {
		value, err := numericToFloat(row.Amount)
		if err != nil {
			return nil, fmt.Errorf("parse amount: %w", err)
		}
		currency := strings.ToUpper(strings.TrimSpace(row.Currency))
		if currency == "" {
			currency = "CHF"
		}
		if currency != "CHF" {
			rate, err := s.exchangeRates.GetRateToCHF(ctx, currency, row.PostedDate.Time)
			if err != nil {
				log.Error().Err(err).Str("currency", currency).Time("date", row.PostedDate.Time).Msg("failed to convert currency")
				return nil, err
			}
			value = value * rate
		}
		total += value
		if value < 0 {
			debitTotal += value
			debitAmounts = append(debitAmounts, value)
		}

		if row.PostedDate.Valid {
			postedDate := row.PostedDate.Time
			if !hasDate {
				minDate = postedDate
				maxDate = postedDate
				hasDate = true
			} else {
				if postedDate.Before(minDate) {
					minDate = postedDate
				}
				if postedDate.After(maxDate) {
					maxDate = postedDate
				}
			}
		}

		accountKey := ""
		if row.SourceAccountNumber.Valid {
			accountKey = strings.TrimSpace(row.SourceAccountNumber.String)
		}
		if accountKey == "" && row.SourceCardNumber.Valid {
			accountKey = strings.TrimSpace(row.SourceCardNumber.String)
		}
		if accountKey != "" {
			uniqueAccounts[accountKey] = struct{}{}
		}
	}

	sort.Float64s(debitAmounts)
	median := 0.0
	if len(debitAmounts) > 0 {
		middle := len(debitAmounts) / 2
		if len(debitAmounts)%2 == 0 {
			median = (debitAmounts[middle-1] + debitAmounts[middle]) / 2
		} else {
			median = debitAmounts[middle]
		}
	}
	average := 0.0
	if len(debitAmounts) > 0 {
		average = debitTotal / float64(len(debitAmounts))
	}

	dateRangeStart := ""
	dateRangeEnd := ""
	if hasDate {
		dateRangeStart = minDate.Format("2006-01-02")
		dateRangeEnd = maxDate.Format("2006-01-02")
	}

	return &apiv1.TransactionSummary{
		Count:          int32(len(rows)),
		Total:          centsFromFloat(total),
		Average:        centsFromFloat(average),
		Median:         centsFromFloat(median),
		Currency:       "CHF",
		UniqueAccounts: int32(len(uniqueAccounts)),
		DateRangeStart: dateRangeStart,
		DateRangeEnd:   dateRangeEnd,
	}, nil
}

func (s *TransactionsService) ListWithCategories(ctx context.Context, userID int32, filters TransactionFilters) ([]*apiv1.Transaction, error) {
	return s.List(ctx, userID, filters)
}

func (s *TransactionsService) listCategoryRules(ctx context.Context, userID int32) ([]CategoryRuleEntry, error) {
	rows, err := s.db.Queries.ListCategoryRulesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	rules := make([]CategoryRuleEntry, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, CategoryRuleEntry{
			CategoryID:          row.CategoryID,
			DescriptionContains: row.DescriptionContains,
		})
	}
	return rules, nil
}

func (s *TransactionsService) ApplyCategoryRules(ctx context.Context, userID int32, applyToAll bool) (int64, error) {
	rules, err := s.listCategoryRules(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("load category rules: %w", err)
	}
	normalizedRules := normalizeRules(rules)

	rows, err := s.db.Queries.ListTransactionsForRuleApply(ctx, db.ListTransactionsForRuleApplyParams{
		UserID:  userID,
		Column2: applyToAll,
	})
	if err != nil {
		return 0, fmt.Errorf("load transactions: %w", err)
	}

	var updated int64
	for _, row := range rows {
		var nextCategoryID pgtype.Int8
		var nextCategorySource pgtype.Text
		if derivedID := matchCategoryRule(row.Description, normalizedRules); derivedID != nil {
			nextCategoryID = pgtype.Int8{Int64: *derivedID, Valid: true}
			nextCategorySource = pgtype.Text{String: categorySourceRule, Valid: true}
		}

		if sameInt8(row.CategoryID, nextCategoryID) && sameText(row.CategorySource, nextCategorySource) {
			continue
		}

		affected, err := s.db.Queries.UpdateTransactionCategory(ctx, db.UpdateTransactionCategoryParams{
			CategoryID:     nextCategoryID,
			CategorySource: nextCategorySource,
			ID:             row.ID,
			UserID:         userID,
		})
		if err != nil {
			return updated, fmt.Errorf("update transaction %d: %w", row.ID, err)
		}
		updated += affected
	}

	return updated, nil
}

type normalizedRule struct {
	CategoryID int64
	Needle     string
}

func normalizeRules(rules []CategoryRuleEntry) []normalizedRule {
	normalized := make([]normalizedRule, 0, len(rules))
	for _, rule := range rules {
		needle := strings.ToLower(strings.TrimSpace(rule.DescriptionContains))
		if needle == "" {
			continue
		}
		normalized = append(normalized, normalizedRule{CategoryID: rule.CategoryID, Needle: needle})
	}
	return normalized
}

func matchCategoryRule(description string, rules []normalizedRule) *int64 {
	if len(rules) == 0 {
		return nil
	}
	haystack := strings.ToLower(description)
	for _, rule := range rules {
		if strings.Contains(haystack, rule.Needle) {
			value := rule.CategoryID
			return &value
		}
	}
	return nil
}

func categoryIDFromDescription(description string, rules []normalizedRule) pgtype.Int8 {
	if description == "" {
		return pgtype.Int8{}
	}
	if derivedID := matchCategoryRule(description, rules); derivedID != nil {
		return pgtype.Int8{Int64: *derivedID, Valid: true}
	}
	return pgtype.Int8{}
}

func nullableText(value string) pgtype.Text {
	if strings.TrimSpace(value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func textOrNull(value string) pgtype.Text {
	return nullableText(value)
}

func int64OrNull(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}

func sameInt8(left, right pgtype.Int8) bool {
	if left.Valid != right.Valid {
		return false
	}
	if !left.Valid {
		return true
	}
	return left.Int64 == right.Int64
}

func sameText(left, right pgtype.Text) bool {
	if left.Valid != right.Valid {
		return false
	}
	if !left.Valid {
		return true
	}
	return left.String == right.String
}

func dateOrNull(value *time.Time) pgtype.Date {
	if value == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *value, Valid: true}
}

func int32OrDefault(value int, fallback int32) int32 {
	if value == 0 {
		return fallback
	}
	return int32(value)
}
