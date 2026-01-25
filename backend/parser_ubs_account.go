package cashtrack

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

type UBSAccountParser struct{}

func NewUBSAccountParser() *UBSAccountParser {
	return &UBSAccountParser{}
}

func (p *UBSAccountParser) Name() string {
	return "ubs_account_transactions"
}

func (p *UBSAccountParser) CanParse(sample string, filename string) bool {
	trimmed := strings.TrimSpace(sample)
	return strings.HasPrefix(trimmed, "Account number:") || strings.HasPrefix(trimmed, "\ufeffAccount number:")
}

func (p *UBSAccountParser) Parse(data []byte) (ParsedReport, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	var accountNumber string
	var headers map[string]int
	inDataSection := false
	rowNumber := 0

	transactions := make([]ParsedTransaction, 0)
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return ParsedReport{}, fmt.Errorf("read csv: %w", err)
		}

		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		if !inDataSection {
			first := strings.TrimSpace(stripBOM(record[0]))
			if strings.EqualFold(first, "Account number:") && len(record) > 1 {
				accountNumber = strings.TrimSpace(record[1])
				continue
			}

			if strings.EqualFold(first, "Trade date") {
				headers = headerIndex(record)
				inDataSection = true
			}
			continue
		}

		rowNumber++
		postedDateRaw := fieldByHeader(headers, record, "Booking date")
		postedDate, err := time.Parse("2006-01-02", postedDateRaw)
		if err != nil {
			return ParsedReport{}, fmt.Errorf("parse booking date %q: %w", postedDateRaw, err)
		}

		debitRaw := fieldByHeader(headers, record, "Debit")
		creditRaw := fieldByHeader(headers, record, "Credit")
		entryType, amountRaw := resolveEntryType(debitRaw, creditRaw)
		if entryType == "" {
			continue
		}
		amount, err := normalizeAmount(amountRaw, entryType)
		if err != nil {
			return ParsedReport{}, fmt.Errorf("parse amount %q: %w", amountRaw, err)
		}

		description := joinNonEmpty(
			fieldByHeader(headers, record, "Description1"),
			fieldByHeader(headers, record, "Description2"),
			fieldByHeader(headers, record, "Description3"),
		)

		transactions = append(transactions, ParsedTransaction{
			PostedDate:          postedDate,
			Description:         description,
			Amount:              amount,
			Currency:            fieldByHeader(headers, record, "Currency"),
			TransactionID:       fieldByHeader(headers, record, "Transaction no."),
			EntryType:           entryType,
			SourceAccountNumber: accountNumber,
			SourceFileRow:       rowNumber,
			ParserName:          p.Name(),
		})
	}

	return ParsedReport{ParserName: p.Name(), Transactions: transactions}, nil
}

func headerIndex(headers []string) map[string]int {
	index := make(map[string]int, len(headers))
	for i, header := range headers {
		index[strings.TrimSpace(header)] = i
	}
	return index
}

func fieldByHeader(headers map[string]int, record []string, name string) string {
	idx, ok := headers[name]
	if !ok || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

func resolveEntryType(debitRaw string, creditRaw string) (string, string) {
	debitRaw = strings.TrimSpace(debitRaw)
	creditRaw = strings.TrimSpace(creditRaw)
	if debitRaw != "" {
		return EntryTypeDebit, debitRaw
	}
	if creditRaw != "" {
		return EntryTypeCredit, creditRaw
	}
	return "", ""
}

func joinNonEmpty(parts ...string) string {
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return strings.Join(items, "; ")
}

func normalizeAmount(raw string, entryType string) (string, error) {
	value := strings.TrimSpace(raw)
	value = strings.ReplaceAll(value, " ", "")
	if value == "" {
		return "", nil
	}
	if strings.Contains(value, ",") && !strings.Contains(value, ".") {
		value = strings.ReplaceAll(value, ",", ".")
	}
	if strings.HasPrefix(value, "+") {
		value = strings.TrimPrefix(value, "+")
	}
	if entryType == EntryTypeDebit && !strings.HasPrefix(value, "-") {
		value = "-" + value
	}
	if entryType == EntryTypeCredit && strings.HasPrefix(value, "-") {
		value = strings.TrimPrefix(value, "-")
	}
	return value, nil
}

func stripBOM(value string) string {
	return strings.TrimPrefix(value, "\ufeff")
}
