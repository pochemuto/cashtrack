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
}

type TransactionFilters struct {
	FromDate            *time.Time
	ToDate              *time.Time
	SourceFileID        *int64
	EntryType           string
	SearchText          string
	SourceAccountNumber string
	SourceCardNumber    string
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
		LimitCount:          int32OrDefault(filters.Limit, 500),
		OffsetCount:         int32OrDefault(filters.Offset, 0),
	}

	rows, err := s.db.Queries.ListTransactions(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}

	entries := make([]TransactionEntry, 0)
	for _, row := range rows {
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
		}
		entries = append(entries, entry)
	}

	return entries, nil
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
