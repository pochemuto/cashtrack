package cashtrack

import (
	db "cashtrack/backend/gen/db"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TransactionsService struct {
	db            *Db
	httpClient    *http.Client
	rateCache     map[string]float64
	rateCacheLock sync.Mutex
}

type TransactionEntry struct {
	ID                  int64
	SourceFileID        int64
	SourceFileRow       int
	ParserName          string
	PostedDate          time.Time
	Description         string
	Amount              string
	Currency            string
	TransactionID       string
	EntryType           string
	SourceAccountNumber string
	SourceCardNumber    string
	ParserMeta          json.RawMessage
	CreatedAt           time.Time
	CategoryID          *int64
}

type TransactionSummary struct {
	Count    int64  `json:"count"`
	Total    string `json:"total"`
	Average  string `json:"average"`
	Median   string `json:"median"`
	Currency string `json:"currency"`
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
		db:         db,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		rateCache:  make(map[string]float64),
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
			ParserMeta:          meta,
		})
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	return nil
}

func (s *TransactionsService) List(ctx context.Context, userID int32, filters TransactionFilters) ([]TransactionEntry, error) {
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

	entries := make([]TransactionEntry, 0)
	for _, row := range rows {
		var categoryID *int64
		if row.CategoryID.Valid {
			value := row.CategoryID.Int64
			categoryID = &value
		}
		entry := TransactionEntry{
			ID:                  row.ID,
			SourceFileID:        row.SourceFileID,
			SourceFileRow:       int(row.SourceFileRow),
			ParserName:          row.ParserName,
			PostedDate:          row.PostedDate.Time,
			Description:         row.Description,
			Amount:              numericToString(row.Amount),
			Currency:            row.Currency,
			TransactionID:       row.TransactionID.String,
			EntryType:           row.EntryType,
			SourceAccountNumber: row.SourceAccountNumber.String,
			SourceCardNumber:    row.SourceCardNumber.String,
			ParserMeta:          row.ParserMeta,
			CreatedAt:           row.CreatedAt.Time,
			CategoryID:          categoryID,
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

type CategoryRuleEntry struct {
	CategoryID          int64
	DescriptionContains string
}

func (s *TransactionsService) Summary(ctx context.Context, userID int32, filters TransactionFilters) (TransactionSummary, error) {
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
		return TransactionSummary{}, err
	}

	if len(rows) == 0 {
		return TransactionSummary{
			Count:    0,
			Total:    "0",
			Average:  "0",
			Median:   "0",
			Currency: "CHF",
		}, nil
	}

	amounts := make([]float64, 0, len(rows))
	var total float64
	for _, row := range rows {
		value, err := numericToFloat(row.Amount)
		if err != nil {
			return TransactionSummary{}, fmt.Errorf("parse amount: %w", err)
		}
		currency := strings.ToUpper(strings.TrimSpace(row.Currency))
		if currency == "" {
			currency = "CHF"
		}
		if currency != "CHF" {
			rate, err := s.getRateToCHF(ctx, currency, row.PostedDate.Time)
			if err != nil {
				log.Error().Err(err).Str("currency", currency).Time("date", row.PostedDate.Time).Msg("failed to convert currency")
				return TransactionSummary{}, err
			}
			value = value * rate
		}
		total += value
		amounts = append(amounts, value)
	}

	sort.Float64s(amounts)
	median := 0.0
	if len(amounts) > 0 {
		middle := len(amounts) / 2
		if len(amounts)%2 == 0 {
			median = (amounts[middle-1] + amounts[middle]) / 2
		} else {
			median = amounts[middle]
		}
	}
	average := 0.0
	if len(amounts) > 0 {
		average = total / float64(len(amounts))
	}

	return TransactionSummary{
		Count:    int64(len(amounts)),
		Total:    formatFloat(total),
		Average:  formatFloat(average),
		Median:   formatFloat(median),
		Currency: "CHF",
	}, nil
}

func (s *TransactionsService) ListWithCategories(ctx context.Context, userID int32, filters TransactionFilters) ([]TransactionEntry, error) {
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

func numericFromString(value string) (pgtype.Numeric, error) {
	var numeric pgtype.Numeric
	if strings.TrimSpace(value) == "" {
		return numeric, fmt.Errorf("amount is empty")
	}
	if err := numeric.Scan(value); err != nil {
		return numeric, err
	}
	return numeric, nil
}

func numericToString(value pgtype.Numeric) string {
	if !value.Valid {
		return ""
	}
	plan := (pgtype.NumericCodec{}).PlanEncode(nil, 0, pgtype.TextFormatCode, value)
	if plan == nil {
		return ""
	}
	buf, err := plan.Encode(value, nil)
	if err != nil {
		return ""
	}
	return string(buf)
}

func numericToFloat(value pgtype.Numeric) (float64, error) {
	if !value.Valid {
		return 0, nil
	}
	raw := numericToString(value)
	if raw == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func (s *TransactionsService) getRateToCHF(ctx context.Context, currency string, date time.Time) (float64, error) {
	if currency == "CHF" {
		return 1, nil
	}
	dateKey := date.Format("2006-01-02")
	cacheKey := currency + "|" + dateKey

	s.rateCacheLock.Lock()
	if rate, ok := s.rateCache[cacheKey]; ok {
		s.rateCacheLock.Unlock()
		return rate, nil
	}
	s.rateCacheLock.Unlock()

	url := fmt.Sprintf("https://api.exchangerate.host/%s?base=%s&symbols=CHF", dateKey, currency)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("rate request failed: %s", resp.Status)
	}
	var payload struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	rate, ok := payload.Rates["CHF"]
	if !ok || rate == 0 {
		return 0, fmt.Errorf("missing CHF rate for %s on %s", currency, dateKey)
	}

	s.rateCacheLock.Lock()
	s.rateCache[cacheKey] = rate
	s.rateCacheLock.Unlock()
	return rate, nil
}
