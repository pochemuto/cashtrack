package cashtrack

import (
	db "cashtrack/backend/gen/db"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TransactionsService struct {
	db *Db
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
	Count   int64  `json:"count"`
	Total   string `json:"total"`
	Average string `json:"average"`
	Median  string `json:"median"`
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
	return &TransactionsService{db: db}
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
	params := []any{
		userID,
		dateOrNull(filters.FromDate),
		dateOrNull(filters.ToDate),
		int64OrNull(filters.SourceFileID),
		textOrNull(filters.EntryType),
		textOrNull(filters.SourceAccountNumber),
		textOrNull(filters.SourceCardNumber),
		textOrNull(filters.SearchText),
		int64OrNull(filters.CategoryID),
	}

	query := `
		SELECT
			COUNT(*),
			COALESCE(SUM(amount), 0::numeric)::text,
			COALESCE(AVG(amount), 0::numeric)::text,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount), 0::numeric)::text
		FROM transactions
		WHERE user_id = $1
		  AND ($2::date IS NULL OR posted_date >= $2)
		  AND ($3::date IS NULL OR posted_date <= $3)
		  AND ($4::bigint IS NULL OR source_file_id = $4)
		  AND ($5::text IS NULL OR entry_type = $5)
		  AND ($6::text IS NULL OR source_account_number = $6)
		  AND ($7::text IS NULL OR source_card_number = $7)
		  AND ($8::text IS NULL OR to_tsvector('simple', description) @@ plainto_tsquery('simple', $8))
		  AND ($9::bigint IS NULL OR category_id = $9)
	`

	var summary TransactionSummary
	if err := s.db.conn.QueryRow(ctx, query, params...).Scan(&summary.Count, &summary.Total, &summary.Average, &summary.Median); err != nil {
		log.Error().Err(err).Interface("filters", filters).Msg("failed to query transactions summary")
		return summary, err
	}
	if summary.Total == "" {
		summary.Total = "0"
	}
	if summary.Average == "" {
		summary.Average = "0"
	}
	if summary.Median == "" {
		summary.Median = "0"
	}
	return summary, nil
}

func (s *TransactionsService) ListWithCategories(ctx context.Context, userID int32, filters TransactionFilters) ([]TransactionEntry, error) {
	return s.List(ctx, userID, filters)
}

func (s *TransactionsService) listCategoryRules(ctx context.Context, userID int32) ([]CategoryRuleEntry, error) {
	rows, err := s.db.conn.Query(ctx, "SELECT category_id, description_contains FROM category_rules WHERE user_id = $1 ORDER BY id", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []CategoryRuleEntry
	for rows.Next() {
		var rule CategoryRuleEntry
		if err := rows.Scan(&rule.CategoryID, &rule.DescriptionContains); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
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
