package cashtrack

import (
	"context"
	"testing"
	"time"
)

func TestTransactionsSummaryHandlesNegativeMedian(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()
	ctx := context.Background()

	createSummaryTables(t, db)
	userID := createUser(t, db, "summary@example.com")

	_, err := db.conn.Exec(ctx, `
		INSERT INTO transactions (
			user_id,
			source_file_id,
			posted_date,
			description,
			amount,
			entry_type,
			category_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, int64(1), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "Test", "-5.90", EntryTypeDebit, int64(4))
	if err != nil {
		t.Fatalf("insert transaction: %v", err)
	}

	service := NewTransactionsService(db)
	summary, err := service.Summary(ctx, userID, TransactionFilters{CategoryID: int64Ptr(4)})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}

	if summary.Count != 1 {
		t.Fatalf("expected count 1, got %d", summary.Count)
	}
	assertSummaryCents(t, summary.Total, -590)
	assertSummaryCents(t, summary.Average, -590)
	assertSummaryCents(t, summary.Median, -590)
}

func TestTransactionsSummaryEmpty(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()
	ctx := context.Background()

	createSummaryTables(t, db)
	userID := createUser(t, db, "summary-empty@example.com")

	service := NewTransactionsService(db)
	summary, err := service.Summary(ctx, userID, TransactionFilters{})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}

	if summary.Count != 0 {
		t.Fatalf("expected count 0, got %d", summary.Count)
	}
	assertSummaryCents(t, summary.Total, 0)
	assertSummaryCents(t, summary.Average, 0)
	assertSummaryCents(t, summary.Median, 0)
}

func createSummaryTables(t *testing.T, db *Db) {
	t.Helper()
	_, err := db.conn.Exec(context.Background(), `
		CREATE TABLE transactions (
			id bigserial PRIMARY KEY,
			user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			source_file_id bigint,
			posted_date date NOT NULL,
			description text NOT NULL,
			amount numeric(18, 2) NOT NULL,
			entry_type varchar(16),
			source_account_number varchar(64),
			source_card_number varchar(64),
			category_id bigint
		);
	`)
	if err != nil {
		t.Fatalf("create summary tables: %v", err)
	}
}

func assertSummaryCents(t *testing.T, raw int64, expected int64) {
	t.Helper()
	if raw != expected {
		t.Fatalf("expected %d, got %d", expected, raw)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
