package cashtrack

import (
	"strings"
	"testing"
	"time"
)

func TestCreditCardParser_Parse(t *testing.T) {
	parser := NewCreditCardParser()
	data := mustReadTestFile(t, "credit_card_transactions.csv")

	report, err := parser.Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if report.ParserName != parser.Name() {
		t.Fatalf("expected parser name %q, got %q", parser.Name(), report.ParserName)
	}
	if got := len(report.Transactions); got != 47 {
		t.Fatalf("expected 47 transactions, got %d", got)
	}

	first := report.Transactions[0]
	if first.EntryType != EntryTypeDebit {
		t.Fatalf("expected entry type debit, got %q", first.EntryType)
	}
	if first.Currency != "CHF" {
		t.Fatalf("expected currency CHF, got %q", first.Currency)
	}
	if first.Amount != "-28.95" {
		t.Fatalf("expected amount -28.95, got %q", first.Amount)
	}
	if !strings.HasPrefix(first.TransactionID, "cc-") {
		t.Fatalf("expected transaction id to have cc- prefix, got %q", first.TransactionID)
	}
	if first.SourceAccountNumber != "7000 2895 9703" {
		t.Fatalf("expected account number, got %q", first.SourceAccountNumber)
	}
	if first.SourceCardNumber != "4894 33XX XXXX 9396" {
		t.Fatalf("expected card number, got %q", first.SourceCardNumber)
	}

	expectedDate := time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)
	if !sameDate(first.PostedDate, expectedDate) {
		t.Fatalf("expected posted date %v, got %v", expectedDate, first.PostedDate)
	}

	if first.Description == "" {
		t.Fatalf("expected description to be present")
	}
}
