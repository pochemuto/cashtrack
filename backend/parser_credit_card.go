package cashtrack

import (
	"crypto/sha1"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"
)

type CreditCardParser struct{}

func NewCreditCardParser() *CreditCardParser {
	return &CreditCardParser{}
}

func (p *CreditCardParser) Name() string {
	return "credit_card_transactions"
}

func (p *CreditCardParser) CanParse(sample string, filename string) bool {
	trimmed := strings.TrimSpace(sample)
	return strings.HasPrefix(trimmed, "sep=") || strings.HasPrefix(trimmed, "Account number;")
}

func (p *CreditCardParser) Parse(data []byte) (ParsedReport, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	record, err := reader.Read()
	if err != nil {
		return ParsedReport{}, fmt.Errorf("read csv header: %w", err)
	}
	if len(record) > 0 && strings.HasPrefix(strings.TrimSpace(record[0]), "sep=") {
		record, err = reader.Read()
		if err != nil {
			return ParsedReport{}, fmt.Errorf("read csv header: %w", err)
		}
	}

	headers := headerIndex(record)
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

		rowNumber++
		purchaseDateRaw := fieldByHeader(headers, record, "Purchase date")
		if strings.TrimSpace(purchaseDateRaw) == "" {
			continue
		}
		postedDate, err := time.Parse("02.01.2006", purchaseDateRaw)
		if err != nil {
			return ParsedReport{}, fmt.Errorf("parse purchase date %q: %w", purchaseDateRaw, err)
		}

		debitRaw := fieldByHeader(headers, record, "Debit")
		creditRaw := fieldByHeader(headers, record, "Credit")
		entryType, amountRaw := resolveEntryType(debitRaw, creditRaw)
		if strings.TrimSpace(amountRaw) == "" {
			amountRaw = fieldByHeader(headers, record, "Amount")
			if entryType == "" {
				entryType = EntryTypeDebit
			}
		}
		amount, err := normalizeAmount(amountRaw, entryType)
		if err != nil {
			return ParsedReport{}, fmt.Errorf("parse amount %q: %w", amountRaw, err)
		}

		description := fieldByHeader(headers, record, "Booking text")
		accountNumber := fieldByHeader(headers, record, "Account number")
		cardNumber := fieldByHeader(headers, record, "Card number")

		transactionID := buildCardTransactionID(accountNumber, cardNumber, purchaseDateRaw, description, amount)
		transactions = append(transactions, ParsedTransaction{
			PostedDate:          postedDate,
			Description:         description,
			Amount:              amount,
			Currency:            fieldByHeader(headers, record, "Currency"),
			TransactionID:       transactionID,
			EntryType:           entryType,
			SourceAccountNumber: accountNumber,
			SourceCardNumber:    cardNumber,
			SourceFileRow:       rowNumber,
			ParserName:          p.Name(),
		})
	}

	return ParsedReport{ParserName: p.Name(), Transactions: transactions}, nil
}

func buildCardTransactionID(accountNumber string, cardNumber string, purchaseDate string, description string, amount string) string {
	hash := sha1.New()
	hash.Write([]byte(accountNumber))
	hash.Write([]byte("|"))
	hash.Write([]byte(cardNumber))
	hash.Write([]byte("|"))
	hash.Write([]byte(purchaseDate))
	hash.Write([]byte("|"))
	hash.Write([]byte(description))
	hash.Write([]byte("|"))
	hash.Write([]byte(amount))
	return "cc-" + hex.EncodeToString(hash.Sum(nil))
}
