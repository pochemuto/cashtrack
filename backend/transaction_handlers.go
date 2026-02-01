package cashtrack

import (
	"context"
	"errors"
	"strings"
	"time"

	apiv1 "cashtrack/backend/gen/api/v1"
	"cashtrack/backend/gen/api/v1/apiv1connect"
	dbgen "cashtrack/backend/gen/db"
	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"github.com/jackc/pgx/v5/pgtype"
)

type TransactionService struct {
	db           *Db
	transactions *TransactionsService
}

type TransactionServiceHandler Handler

func NewTransactionServiceHandler(db *Db, transactions *TransactionsService) *TransactionServiceHandler {
	service := &TransactionService{db: db, transactions: transactions}
	path, handler := apiv1connect.NewTransactionServiceHandler(
		service,
		connect.WithInterceptors(validate.NewInterceptor(), NewAuthInterceptor(db)),
	)
	return &TransactionServiceHandler{Path: path, Handler: handler}
}

func (s *TransactionService) ListTransactions(ctx context.Context, req *apiv1.ListTransactionsRequest) (*apiv1.ListTransactionsResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	filters, err := transactionFiltersFromRequest(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	items, err := s.transactions.ListWithCategories(ctx, user.Id, filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	summary, err := s.transactions.Summary(ctx, user.Id, filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &apiv1.ListTransactionsResponse{Items: items, Summary: summary}, nil
}

func (s *TransactionService) UpdateTransactionCategory(ctx context.Context, req *apiv1.UpdateTransactionCategoryRequest) (*apiv1.UpdateTransactionCategoryResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if req.TransactionId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("transaction_id is required"))
	}

	var categoryID *int64
	if req.CategoryId != nil {
		value := int64(*req.CategoryId)
		if value == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category_id must be set or omitted"))
		}
		category, err := getCategory(ctx, s.db, user.Id, int32(value))
		if err != nil {
			if errors.Is(err, errNotFound) {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category not found"))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if category.IsGroup {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category cannot be a group"))
		}
		categoryID = &value
	}

	if err := updateTransactionCategory(ctx, s.db, user.Id, req.TransactionId, categoryID); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.UpdateTransactionCategoryResponse{}, nil
}

func transactionFiltersFromRequest(req *apiv1.ListTransactionsRequest) (TransactionFilters, error) {
	filters := TransactionFilters{}

	fromDate := strings.TrimSpace(req.FromDate)
	if fromDate != "" {
		value, err := time.Parse("2006-01-02", fromDate)
		if err != nil {
			return filters, err
		}
		filters.FromDate = &value
	}

	toDate := strings.TrimSpace(req.ToDate)
	if toDate != "" {
		value, err := time.Parse("2006-01-02", toDate)
		if err != nil {
			return filters, err
		}
		filters.ToDate = &value
	}

	if req.SourceFileId > 0 {
		value := int64(req.SourceFileId)
		filters.SourceFileID = &value
	}

	entryType := strings.TrimSpace(req.EntryType)
	if entryType != "" {
		filters.EntryType = entryType
	}

	searchText := strings.TrimSpace(req.SearchText)
	if searchText != "" {
		filters.SearchText = searchText
	}

	if req.CategoryId > 0 {
		value := int64(req.CategoryId)
		filters.CategoryID = &value
	}

	accountNumber := strings.TrimSpace(req.AccountNumber)
	if accountNumber != "" {
		filters.SourceAccountNumber = accountNumber
	}

	cardNumber := strings.TrimSpace(req.CardNumber)
	if cardNumber != "" {
		filters.SourceCardNumber = cardNumber
	}

	if req.Limit > 0 {
		filters.Limit = int(req.Limit)
	}

	if req.Offset > 0 {
		filters.Offset = int(req.Offset)
	}

	return filters, nil
}

func updateTransactionCategory(ctx context.Context, db *Db, userID int32, transactionID int32, categoryID *int64) error {
	var category pgtype.Int8
	var categorySource pgtype.Text
	if categoryID != nil {
		category = pgtype.Int8{Int64: *categoryID, Valid: true}
		categorySource = pgtype.Text{String: categorySourceManual, Valid: true}
	}
	affected, err := db.Queries.UpdateTransactionCategory(ctx, dbgen.UpdateTransactionCategoryParams{
		CategoryID:     category,
		CategorySource: categorySource,
		ID:             int64(transactionID),
		UserID:         userID,
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}
