package cashtrack

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUBSAccountParser_Parse(t *testing.T) {
	parser := NewUBSAccountParser()
	data := mustReadTestFile(t, "ubs_account_transactions.csv")

	report, err := parser.Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if report.ParserName != parser.Name() {
		t.Fatalf("expected parser name %q, got %q", parser.Name(), report.ParserName)
	}
	if got := len(report.Transactions); got != 22 {
		t.Fatalf("expected 22 transactions, got %d", got)
	}

	first := report.Transactions[0]
	if first.EntryType != EntryTypeDebit {
		t.Fatalf("expected entry type debit, got %q", first.EntryType)
	}
	if first.Currency != "CHF" {
		t.Fatalf("expected currency CHF, got %q", first.Currency)
	}
	if first.Amount != "-1500.00" {
		t.Fatalf("expected amount -1500.00, got %q", first.Amount)
	}
	if first.TransactionID != "9930023GK2701888" {
		t.Fatalf("expected transaction id 9930023GK2701888, got %q", first.TransactionID)
	}
	if first.SourceAccountNumber != "0230 00826810.40" {
		t.Fatalf("expected account number, got %q", first.SourceAccountNumber)
	}

	expectedDate := time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC)
	if !sameDate(first.PostedDate, expectedDate) {
		t.Fatalf("expected posted date %v, got %v", expectedDate, first.PostedDate)
	}

	if first.Description == "" {
		t.Fatalf("expected description to be present")
	}
}

func mustReadTestFile(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func sameDate(value time.Time, expected time.Time) bool {
	return value.Year() == expected.Year() && value.Month() == expected.Month() && value.Day() == expected.Day()
}
