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
	"github.com/jackc/pgx/v5"
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

	parentID, err := resolveCategoryParent(ctx, s.db, user.Id, 0, req.ParentId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	category, err := createCategory(ctx, s.db, user.Id, name, color, parentID, req.IsGroup)
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

	parentID, err := resolveCategoryParent(ctx, s.db, user.Id, req.Id, req.ParentId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := updateCategory(ctx, s.db, user.Id, req.Id, name, color, parentID, req.IsGroup); err != nil {
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
	category, err := getCategory(ctx, s.db, user.Id, req.CategoryId)
	if err != nil {
		if errors.Is(err, errNotFound) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if category.IsGroup {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category cannot be a group"))
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
	category, err := getCategory(ctx, s.db, user.Id, req.CategoryId)
	if err != nil {
		if errors.Is(err, errNotFound) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if category.IsGroup {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category cannot be a group"))
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
		var parentID int32
		if row.ParentID.Valid {
			parentID = int32(row.ParentID.Int64)
		}
		categories = append(categories, &apiv1.Category{
			Id:        int32(row.ID),
			Name:      row.Name,
			Color:     color,
			CreatedAt: row.CreatedAt.Time.Format(time.RFC3339Nano),
			ParentId:  parentID,
			IsGroup:   row.IsGroup,
		})
	}
	return categories, nil
}

func createCategory(ctx context.Context, db *Db, userID int32, name string, color pgtype.Text, parentID pgtype.Int8, isGroup bool) (*apiv1.Category, error) {
	row, err := db.Queries.CreateCategory(ctx, dbgen.CreateCategoryParams{
		UserID:   userID,
		Name:     name,
		Color:    color,
		ParentID: parentID,
		IsGroup:  isGroup,
	})
	if err != nil {
		return nil, err
	}
	colorValue := ""
	if row.Color.Valid {
		colorValue = row.Color.String
	}
	var parentIDValue int32
	if row.ParentID.Valid {
		parentIDValue = int32(row.ParentID.Int64)
	}
	return &apiv1.Category{
		Id:        int32(row.ID),
		Name:      row.Name,
		Color:     colorValue,
		CreatedAt: row.CreatedAt.Time.Format(time.RFC3339Nano),
		ParentId:  parentIDValue,
		IsGroup:   row.IsGroup,
	}, nil
}

func updateCategory(ctx context.Context, db *Db, userID int32, id int32, name string, color pgtype.Text, parentID pgtype.Int8, isGroup bool) error {
	affected, err := db.Queries.UpdateCategory(ctx, dbgen.UpdateCategoryParams{
		Name:     name,
		Color:    color,
		ParentID: parentID,
		IsGroup:  isGroup,
		ID:       int64(id),
		UserID:   userID,
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

func getCategory(ctx context.Context, db *Db, userID int32, id int32) (*dbgen.GetCategoryByIDRow, error) {
	row, err := db.Queries.GetCategoryByID(ctx, dbgen.GetCategoryByIDParams{
		ID:     int64(id),
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	return &row, nil
}

func resolveCategoryParent(ctx context.Context, db *Db, userID int32, categoryID int32, parentID int32) (pgtype.Int8, error) {
	if parentID < 0 {
		return pgtype.Int8{}, errors.New("parent_id must be positive")
	}
	if parentID == 0 {
		return pgtype.Int8{}, nil
	}
	if categoryID != 0 && parentID == categoryID {
		return pgtype.Int8{}, errors.New("parent_id must be different from id")
	}
	if _, err := getCategory(ctx, db, userID, parentID); err != nil {
		if errors.Is(err, errNotFound) {
			return pgtype.Int8{}, errors.New("parent category not found")
		}
		return pgtype.Int8{}, err
	}
	if categoryID != 0 {
		rows, err := db.Queries.ListCategoriesByUser(ctx, userID)
		if err != nil {
			return pgtype.Int8{}, err
		}
		if hasCategoryParentCycle(rows, categoryID, parentID) {
			return pgtype.Int8{}, errors.New("category parent creates a cycle")
		}
	}
	return pgtype.Int8{Int64: int64(parentID), Valid: true}, nil
}

func hasCategoryParentCycle(rows []dbgen.ListCategoriesByUserRow, categoryID int32, parentID int32) bool {
	if categoryID == 0 || parentID == 0 {
		return false
	}
	parentByID := make(map[int32]int32, len(rows))
	for _, row := range rows {
		if row.ParentID.Valid {
			parentByID[int32(row.ID)] = int32(row.ParentID.Int64)
		}
	}
	visited := make(map[int32]struct{}, len(rows))
	current := parentID
	for current != 0 {
		if current == categoryID {
			return true
		}
		if _, ok := visited[current]; ok {
			return true
		}
		visited[current] = struct{}{}
		next, ok := parentByID[current]
		if !ok {
			break
		}
		current = next
	}
	return false
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
