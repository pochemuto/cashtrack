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

type CategoryService struct {
	db           *Db
	transactions *TransactionsService
}

type CategoryServiceHandler Handler

var errNotFound = errors.New("not found")

func NewCategoryServiceHandler(db *Db, transactions *TransactionsService) *CategoryServiceHandler {
	service := &CategoryService{db: db, transactions: transactions}
	path, handler := apiv1connect.NewCategoryServiceHandler(
		service,
		connect.WithInterceptors(validate.NewInterceptor(), NewAuthInterceptor(db)),
	)
	return &CategoryServiceHandler{Path: path, Handler: handler}
}

func (s *CategoryService) ListCategories(ctx context.Context, req *apiv1.ListCategoriesRequest) (*apiv1.ListCategoriesResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	categories, err := listCategories(ctx, s.db, user.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.ListCategoriesResponse{Categories: categories}, nil
}

func (s *CategoryService) CreateCategory(ctx context.Context, req *apiv1.CreateCategoryRequest) (*apiv1.CreateCategoryResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	colorValue := strings.TrimSpace(req.Color)
	color, err := parseCategoryColor(colorValue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	category, err := createCategory(ctx, s.db, user.Id, name, color)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.CreateCategoryResponse{Category: category}, nil
}

func (s *CategoryService) UpdateCategory(ctx context.Context, req *apiv1.UpdateCategoryRequest) (*apiv1.UpdateCategoryResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	colorValue := strings.TrimSpace(req.Color)
	color, err := parseCategoryColor(colorValue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := updateCategory(ctx, s.db, user.Id, req.Id, name, color); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.UpdateCategoryResponse{}, nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, req *apiv1.DeleteCategoryRequest) (*apiv1.DeleteCategoryResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := deleteCategory(ctx, s.db, user.Id, req.Id); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.DeleteCategoryResponse{}, nil
}

func (s *CategoryService) ListCategoryRules(ctx context.Context, req *apiv1.ListCategoryRulesRequest) (*apiv1.ListCategoryRulesResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := listCategoryRules(ctx, s.db, user.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.ListCategoryRulesResponse{Rules: rules}, nil
}

func (s *CategoryService) CreateCategoryRule(ctx context.Context, req *apiv1.CreateCategoryRuleRequest) (*apiv1.CreateCategoryRuleResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.CategoryId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category_id is required"))
	}
	description := strings.TrimSpace(req.DescriptionContains)
	if description == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("description_contains is required"))
	}
	if !categoryExists(ctx, s.db, user.Id, req.CategoryId) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category not found"))
	}

	rule, err := createCategoryRule(ctx, s.db, user.Id, req.CategoryId, description)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.CreateCategoryRuleResponse{Rule: rule}, nil
}

func (s *CategoryService) UpdateCategoryRule(ctx context.Context, req *apiv1.UpdateCategoryRuleRequest) (*apiv1.UpdateCategoryRuleResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if req.CategoryId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category_id is required"))
	}
	description := strings.TrimSpace(req.DescriptionContains)
	if description == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("description_contains is required"))
	}
	if !categoryExists(ctx, s.db, user.Id, req.CategoryId) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category not found"))
	}

	if err := updateCategoryRule(ctx, s.db, user.Id, req.Id, req.CategoryId, description); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.UpdateCategoryRuleResponse{}, nil
}

func (s *CategoryService) DeleteCategoryRule(ctx context.Context, req *apiv1.DeleteCategoryRuleRequest) (*apiv1.DeleteCategoryRuleResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := deleteCategoryRule(ctx, s.db, user.Id, req.Id); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.DeleteCategoryRuleResponse{}, nil
}

func (s *CategoryService) ApplyCategoryRules(ctx context.Context, req *apiv1.ApplyCategoryRulesRequest) (*apiv1.ApplyCategoryRulesResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	updated, err := s.transactions.ApplyCategoryRules(ctx, user.Id, req.ApplyToAll)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.ApplyCategoryRulesResponse{UpdatedCount: int32(updated)}, nil
}

func (s *CategoryService) ReorderCategoryRules(ctx context.Context, req *apiv1.ReorderCategoryRulesRequest) (*apiv1.ReorderCategoryRulesResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.RuleIds) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rule_ids is required"))
	}

	rules, err := listCategoryRules(ctx, s.db, user.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if len(req.RuleIds) != len(rules) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rule_ids must include all rules"))
	}

	ruleMap := make(map[int32]struct{}, len(rules))
	for _, rule := range rules {
		ruleMap[rule.Id] = struct{}{}
	}

	seen := make(map[int32]struct{}, len(req.RuleIds))
	for _, id := range req.RuleIds {
		if _, ok := ruleMap[id]; !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unknown rule id"))
		}
		if _, ok := seen[id]; ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("duplicate rule id"))
		}
		seen[id] = struct{}{}
	}

	tx, err := s.db.conn.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := s.db.Queries.WithTx(tx)
	for index, id := range req.RuleIds {
		affected, err := txQueries.UpdateCategoryRulePosition(ctx, dbgen.UpdateCategoryRulePositionParams{
			Position: int32(index + 1),
			ID:       int64(id),
			UserID:   user.Id,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if affected == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rule not found"))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &apiv1.ReorderCategoryRulesResponse{}, nil
}

func listCategories(ctx context.Context, db *Db, userID int32) ([]*apiv1.Category, error) {
	rows, err := db.Queries.ListCategoriesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	categories := make([]*apiv1.Category, 0, len(rows))
	for _, row := range rows {
		color := ""
		if row.Color.Valid {
			color = row.Color.String
		}
		categories = append(categories, &apiv1.Category{
			Id:        int32(row.ID),
			Name:      row.Name,
			Color:     color,
			CreatedAt: row.CreatedAt.Time.Format(time.RFC3339Nano),
		})
	}
	return categories, nil
}

func createCategory(ctx context.Context, db *Db, userID int32, name string, color pgtype.Text) (*apiv1.Category, error) {
	row, err := db.Queries.CreateCategory(ctx, dbgen.CreateCategoryParams{
		UserID: userID,
		Name:   name,
		Color:  color,
	})
	if err != nil {
		return nil, err
	}
	colorValue := ""
	if row.Color.Valid {
		colorValue = row.Color.String
	}
	return &apiv1.Category{
		Id:        int32(row.ID),
		Name:      row.Name,
		Color:     colorValue,
		CreatedAt: row.CreatedAt.Time.Format(time.RFC3339Nano),
	}, nil
}

func updateCategory(ctx context.Context, db *Db, userID int32, id int32, name string, color pgtype.Text) error {
	affected, err := db.Queries.UpdateCategory(ctx, dbgen.UpdateCategoryParams{
		Name:   name,
		Color:  color,
		ID:     int64(id),
		UserID: userID,
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}

func deleteCategory(ctx context.Context, db *Db, userID int32, id int32) error {
	affected, err := db.Queries.DeleteCategory(ctx, dbgen.DeleteCategoryParams{
		ID:     int64(id),
		UserID: userID,
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}

func listCategoryRules(ctx context.Context, db *Db, userID int32) ([]*apiv1.CategoryRule, error) {
	rows, err := db.Queries.ListCategoryRulesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	rules := make([]*apiv1.CategoryRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, &apiv1.CategoryRule{
			Id:                  int32(row.ID),
			CategoryId:          int32(row.CategoryID),
			DescriptionContains: row.DescriptionContains,
			Position:            row.Position,
			CreatedAt:           row.CreatedAt.Time.Format(time.RFC3339Nano),
		})
	}
	return rules, nil
}

func createCategoryRule(ctx context.Context, db *Db, userID int32, categoryID int32, description string) (*apiv1.CategoryRule, error) {
	row, err := db.Queries.CreateCategoryRule(ctx, dbgen.CreateCategoryRuleParams{
		UserID:              userID,
		CategoryID:          int64(categoryID),
		DescriptionContains: description,
	})
	if err != nil {
		return nil, err
	}
	return &apiv1.CategoryRule{
		Id:                  int32(row.ID),
		CategoryId:          int32(row.CategoryID),
		DescriptionContains: row.DescriptionContains,
		Position:            row.Position,
		CreatedAt:           row.CreatedAt.Time.Format(time.RFC3339Nano),
	}, nil
}

func updateCategoryRule(ctx context.Context, db *Db, userID int32, id int32, categoryID int32, description string) error {
	affected, err := db.Queries.UpdateCategoryRule(ctx, dbgen.UpdateCategoryRuleParams{
		CategoryID:          int64(categoryID),
		DescriptionContains: description,
		ID:                  int64(id),
		UserID:              userID,
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}

func deleteCategoryRule(ctx context.Context, db *Db, userID int32, id int32) error {
	affected, err := db.Queries.DeleteCategoryRule(ctx, dbgen.DeleteCategoryRuleParams{
		ID:     int64(id),
		UserID: userID,
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}

func categoryExists(ctx context.Context, db *Db, userID int32, id int32) bool {
	exists, err := db.Queries.CategoryExists(ctx, dbgen.CategoryExistsParams{
		ID:     int64(id),
		UserID: userID,
	})
	return err == nil && exists
}

func parseCategoryColor(value string) (pgtype.Text, error) {
	if strings.TrimSpace(value) == "" {
		return pgtype.Text{}, nil
	}
	color := strings.TrimSpace(value)
	if strings.HasPrefix(color, "#") {
		color = strings.TrimPrefix(color, "#")
	}
	if len(color) != 6 {
		return pgtype.Text{}, errors.New("color must be hex like #RRGGBB")
	}
	for _, ch := range color {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
			return pgtype.Text{}, errors.New("color must be hex like #RRGGBB")
		}
	}
	return pgtype.Text{String: "#" + strings.ToUpper(color), Valid: true}, nil
}
