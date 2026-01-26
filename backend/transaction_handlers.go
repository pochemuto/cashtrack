package cashtrack

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type TransactionsListHandler Handler

type TransactionItem struct {
	ID                  int64     `json:"id"`
	SourceFileID        int64     `json:"source_file_id"`
	SourceFileRow       int       `json:"source_file_row"`
	ParserName          string    `json:"parser_name"`
	PostedDate          time.Time `json:"posted_date"`
	Description         string    `json:"description"`
	Amount              string    `json:"amount"`
	Currency            string    `json:"currency"`
	TransactionID       string    `json:"transaction_id"`
	EntryType           string    `json:"entry_type"`
	SourceAccountNumber string    `json:"source_account_number"`
	SourceCardNumber    string    `json:"source_card_number"`
	CreatedAt           time.Time `json:"created_at"`
	CategoryID          *int64    `json:"category_id"`
}

type TransactionsResponse struct {
	Items   []TransactionItem  `json:"items"`
	Summary TransactionSummary `json:"summary"`
}

func NewTransactionsListHandler(db *Db, service *TransactionsService) *TransactionsListHandler {
	return &TransactionsListHandler{
		Path: "/api/transactions",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			user, ok := userFromRequest(r.Context(), db, r.Header)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			filters, err := parseTransactionFilters(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			items, err := service.ListWithCategories(r.Context(), user.ID, filters)
			if err != nil {
				http.Error(w, "failed to load transactions", http.StatusInternalServerError)
				return
			}

			summary, err := service.Summary(r.Context(), user.ID, filters)
			if err != nil {
				http.Error(w, "failed to load summary", http.StatusInternalServerError)
				return
			}

			response := make([]TransactionItem, 0, len(items))
			for _, entry := range items {
				response = append(response, TransactionItem{
					ID:                  entry.ID,
					SourceFileID:        entry.SourceFileID,
					SourceFileRow:       entry.SourceFileRow,
					ParserName:          entry.ParserName,
					PostedDate:          entry.PostedDate,
					Description:         entry.Description,
					Amount:              entry.Amount,
					Currency:            entry.Currency,
					TransactionID:       entry.TransactionID,
					EntryType:           entry.EntryType,
					SourceAccountNumber: entry.SourceAccountNumber,
					SourceCardNumber:    entry.SourceCardNumber,
					CreatedAt:           entry.CreatedAt,
					CategoryID:          entry.CategoryID,
				})
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(TransactionsResponse{Items: response, Summary: summary}); err != nil {
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
			}
		}),
	}
}

func parseTransactionFilters(r *http.Request) (TransactionFilters, error) {
	query := r.URL.Query()

	filters := TransactionFilters{}
	if from := query.Get("from"); from != "" {
		value, err := time.Parse("2006-01-02", from)
		if err != nil {
			return filters, err
		}
		filters.FromDate = &value
	}
	if to := query.Get("to"); to != "" {
		value, err := time.Parse("2006-01-02", to)
		if err != nil {
			return filters, err
		}
		filters.ToDate = &value
	}
	if sourceFile := query.Get("source_file_id"); sourceFile != "" {
		value, err := strconv.ParseInt(sourceFile, 10, 64)
		if err != nil {
			return filters, err
		}
		filters.SourceFileID = &value
	}
	if entryType := query.Get("entry_type"); entryType != "" {
		filters.EntryType = entryType
	}
	if search := query.Get("search"); search != "" {
		filters.SearchText = search
	}
	if category := query.Get("category_id"); category != "" {
		value, err := strconv.ParseInt(category, 10, 64)
		if err != nil {
			return filters, err
		}
		filters.CategoryID = &value
	}
	if account := query.Get("account_number"); account != "" {
		filters.SourceAccountNumber = account
	}
	if card := query.Get("card_number"); card != "" {
		filters.SourceCardNumber = card
	}
	if limit := query.Get("limit"); limit != "" {
		value, err := strconv.Atoi(limit)
		if err != nil {
			return filters, err
		}
		filters.Limit = value
	}
	if offset := query.Get("offset"); offset != "" {
		value, err := strconv.Atoi(offset)
		if err != nil {
			return filters, err
		}
		filters.Offset = value
	}

	return filters, nil
}
