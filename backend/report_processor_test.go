package cashtrack

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestReportProcessor_ProcessPendingReports(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()
	ctx := context.Background()

	createReportTables(t, db)
	userID := createUser(t, db, "reports@example.com")

	ubsData := mustReadTestFile(t, "ubs_account_transactions.csv")
	cardData := mustReadTestFile(t, "credit_card_transactions.csv")

	ubsReportID := insertReport(t, db, userID, "transactions.csv", ubsData)
	cardReportID := insertReport(t, db, userID, "transactions (1).csv", cardData)

	processor := NewReportProcessor(db, NewReportParsingService(), NewTransactionsService(db))
	if err := processor.ProcessPendingReports(ctx); err != nil {
		t.Fatalf("process pending reports: %v", err)
	}

	assertReportStatus(t, db, ubsReportID, userID, "processed")
	assertReportStatus(t, db, cardReportID, userID, "processed")

	assertTransactionCount(t, db, ubsReportID, 22)
	assertTransactionCount(t, db, cardReportID, 47)
	assertTotalTransactions(t, db, 69)

	assertTransactionFields(t, db, ubsReportID, 1, transactionExpectation{
		PostedDate:          time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC),
		DescriptionContains: "Debit UBS TWINT",
		Amount:              "-1500.00",
		Currency:            "CHF",
		TransactionIDPrefix: "9930023GK2701888",
		EntryType:           EntryTypeDebit,
		SourceAccountNumber: "0230 00826810.40",
	})
	assertTransactionFields(t, db, cardReportID, 1, transactionExpectation{
		PostedDate:          time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC),
		DescriptionContains: "UBR* PENDING.UBER.COM",
		Amount:              "-28.95",
		Currency:            "CHF",
		TransactionIDPrefix: "cc-",
		EntryType:           EntryTypeDebit,
		SourceAccountNumber: "7000 2895 9703",
		SourceCardNumber:    "4894 33XX XXXX 9396",
	})
}

func TestReportProcessor_ReprocessReplacesRows(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()
	ctx := context.Background()

	createReportTables(t, db)
	userID := createUser(t, db, "reports-reprocess@example.com")

	ubsData := mustReadTestFile(t, "ubs_account_transactions.csv")
	ubsReportID := insertReport(t, db, userID, "transactions.csv", ubsData)

	processor := NewReportProcessor(db, NewReportParsingService(), NewTransactionsService(db))
	if err := processor.ProcessPendingReports(ctx); err != nil {
		t.Fatalf("process pending reports: %v", err)
	}

	insertDummyTransaction(t, db, userID, ubsReportID)
	setReportStatus(t, db, ubsReportID, userID, "pending")

	if err := processor.ProcessPendingReports(ctx); err != nil {
		t.Fatalf("process pending reports: %v", err)
	}

	assertTransactionCount(t, db, ubsReportID, 22)
}

func createReportTables(t *testing.T, db *Db) {
	t.Helper()
	_, err := db.conn.Exec(context.Background(), `
		CREATE TABLE financial_reports (
			id bigserial PRIMARY KEY,
			user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			filename varchar(255) NOT NULL,
			content_type varchar(255),
			data bytea NOT NULL,
			uploaded_at timestamptz NOT NULL DEFAULT now(),
			status varchar(32) NOT NULL DEFAULT 'pending'
		);
		CREATE TABLE transactions (
			id bigserial PRIMARY KEY,
			user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			source_file_id bigint NOT NULL REFERENCES financial_reports(id) ON DELETE CASCADE,
			source_file_row integer NOT NULL,
			parser_name varchar(64) NOT NULL,
			posted_date date NOT NULL,
			description text NOT NULL,
			amount numeric(18, 2) NOT NULL,
			currency varchar(3) NOT NULL,
			transaction_id text,
			entry_type varchar(16) NOT NULL,
			source_account_number varchar(64),
			source_card_number varchar(64),
			parser_meta jsonb,
			created_at timestamptz NOT NULL DEFAULT now()
		);
	`)
	if err != nil {
		t.Fatalf("create report tables: %v", err)
	}
}

func insertReport(t *testing.T, db *Db, userID int32, filename string, data []byte) int64 {
	t.Helper()
	var reportID int64
	err := db.conn.QueryRow(
		context.Background(),
		`INSERT INTO financial_reports (user_id, filename, content_type, data, status) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		userID,
		filename,
		"text/csv",
		data,
		"pending",
	).Scan(&reportID)
	if err != nil {
		t.Fatalf("insert report: %v", err)
	}
	return reportID
}

func assertReportStatus(t *testing.T, db *Db, reportID int64, userID int32, expected string) {
	t.Helper()
	var status string
	err := db.conn.QueryRow(
		context.Background(),
		`SELECT status FROM financial_reports WHERE id = $1 AND user_id = $2`,
		reportID,
		userID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("load report status: %v", err)
	}
	if status != expected {
		t.Fatalf("expected report status %q, got %q", expected, status)
	}
}

func assertTransactionCount(t *testing.T, db *Db, reportID int64, expected int) {
	t.Helper()
	var count int
	err := db.conn.QueryRow(
		context.Background(),
		`SELECT count(*) FROM transactions WHERE source_file_id = $1`,
		reportID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	if count != expected {
		t.Fatalf("expected %d transactions for report %d, got %d", expected, reportID, count)
	}
}

func assertTotalTransactions(t *testing.T, db *Db, expected int) {
	t.Helper()
	var count int
	err := db.conn.QueryRow(context.Background(), `SELECT count(*) FROM transactions`).Scan(&count)
	if err != nil {
		t.Fatalf("count total transactions: %v", err)
	}
	if count != expected {
		t.Fatalf("expected total %d transactions, got %d", expected, count)
	}
}

func insertDummyTransaction(t *testing.T, db *Db, userID int32, reportID int64) {
	t.Helper()
	_, err := db.conn.Exec(
		context.Background(),
		`INSERT INTO transactions (
			user_id,
			source_file_id,
			source_file_row,
			parser_name,
			posted_date,
			description,
			amount,
			currency,
			transaction_id,
			entry_type
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		userID,
		reportID,
		999,
		"dummy",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		"dummy",
		"-1.00",
		"CHF",
		"dummy",
		EntryTypeDebit,
	)
	if err != nil {
		t.Fatalf("insert dummy transaction: %v", err)
	}
}

func setReportStatus(t *testing.T, db *Db, reportID int64, userID int32, status string) {
	t.Helper()
	_, err := db.conn.Exec(
		context.Background(),
		`UPDATE financial_reports SET status = $1 WHERE id = $2 AND user_id = $3`,
		status,
		reportID,
		userID,
	)
	if err != nil {
		t.Fatalf("update report status: %v", err)
	}
}

type transactionExpectation struct {
	PostedDate          time.Time
	DescriptionContains string
	Amount              string
	Currency            string
	TransactionIDPrefix string
	EntryType           string
	SourceAccountNumber string
	SourceCardNumber    string
}

func assertTransactionFields(t *testing.T, db *Db, reportID int64, sourceRow int, expected transactionExpectation) {
	t.Helper()

	var postedDate pgtype.Date
	var description string
	var amount pgtype.Numeric
	var currency string
	var transactionID pgtype.Text
	var entryType string
	var accountNumber pgtype.Text
	var cardNumber pgtype.Text

	err := db.conn.QueryRow(
		context.Background(),
		`SELECT posted_date, description, amount, currency, transaction_id, entry_type, source_account_number, source_card_number
		FROM transactions
		WHERE source_file_id = $1 AND source_file_row = $2`,
		reportID,
		sourceRow,
	).Scan(&postedDate, &description, &amount, &currency, &transactionID, &entryType, &accountNumber, &cardNumber)
	if err != nil {
		t.Fatalf("load transaction fields: %v", err)
	}

	if !sameDate(postedDate.Time, expected.PostedDate) {
		t.Fatalf("expected posted date %v, got %v", expected.PostedDate, postedDate.Time)
	}
	if expected.DescriptionContains != "" && !strings.Contains(description, expected.DescriptionContains) {
		t.Fatalf("expected description to contain %q, got %q", expected.DescriptionContains, description)
	}
	if got := numericToString(amount); got != expected.Amount {
		t.Fatalf("expected amount %q, got %q", expected.Amount, got)
	}
	if currency != expected.Currency {
		t.Fatalf("expected currency %q, got %q", expected.Currency, currency)
	}
	if expected.TransactionIDPrefix != "" && !strings.HasPrefix(transactionID.String, expected.TransactionIDPrefix) {
		t.Fatalf("expected transaction id prefix %q, got %q", expected.TransactionIDPrefix, transactionID.String)
	}
	if entryType != expected.EntryType {
		t.Fatalf("expected entry type %q, got %q", expected.EntryType, entryType)
	}
	if expected.SourceAccountNumber != "" && accountNumber.String != expected.SourceAccountNumber {
		t.Fatalf("expected account number %q, got %q", expected.SourceAccountNumber, accountNumber.String)
	}
	if expected.SourceCardNumber != "" && cardNumber.String != expected.SourceCardNumber {
		t.Fatalf("expected card number %q, got %q", expected.SourceCardNumber, cardNumber.String)
	}
}
