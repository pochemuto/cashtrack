package cashtrack

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"time"
)

type ParsedTransaction struct {
	PostedDate          time.Time
	Description         string
	Amount              string
	Currency            string
	TransactionID       string
	EntryType           string
	SourceAccountNumber string
	SourceCardNumber    string
	SourceFileRow       int
	ParserName          string
	ParserMeta          map[string]any
}

type ParsedReport struct {
	ParserName   string
	Transactions []ParsedTransaction
}

type ReportParser interface {
	Name() string
	CanParse(sample string, filename string) bool
	Parse(data []byte) (ParsedReport, error)
}

type ReportParsingService struct {
	parsers []ReportParser
}

func NewReportParsingService() *ReportParsingService {
	return &ReportParsingService{
		parsers: []ReportParser{
			NewUBSAccountParser(),
			NewCreditCardParser(),
		},
	}
}

func (s *ReportParsingService) Parse(data []byte, filename string) (ParsedReport, error) {
	sample := firstNonEmptyLine(data)
	for _, parser := range s.parsers {
		if parser.CanParse(sample, filename) {
			return parser.Parse(data)
		}
	}
	return ParsedReport{}, errors.New("no parser available")
}

func firstNonEmptyLine(data []byte) string {
	reader := bufio.NewReader(strings.NewReader(string(data)))
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return ""
			}
			return ""
		}
	}
}

const (
	EntryTypeDebit  = "debit"
	EntryTypeCredit = "credit"
)
