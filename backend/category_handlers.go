package cashtrack

import (
	dbgen "cashtrack/backend/gen/db"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type CategoryRule struct {
	ID                  int64     `json:"id"`
	CategoryID          int64     `json:"category_id"`
	DescriptionContains string    `json:"description_contains"`
	CreatedAt           time.Time `json:"created_at"`
}

type CategoriesHandler Handler

type CategoryHandler Handler

type CategoryRulesHandler Handler

type CategoryRuleHandler Handler

type TransactionCategoryHandler Handler

type categoryPayload struct {
	Name string `json:"name"`
}

type categoryRulePayload struct {
	CategoryID          int64  `json:"category_id"`
	DescriptionContains string `json:"description_contains"`
}

type transactionCategoryPayload struct {
	CategoryID *int64 `json:"category_id"`
}

func NewCategoriesHandler(db *Db) *CategoriesHandler {
	return &CategoriesHandler{
		Path: "/api/categories",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				user, ok := userFromRequest(r.Context(), db, r.Header)
				if !ok {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				categories, err := listCategories(r.Context(), db, user.ID)
				if err != nil {
					http.Error(w, "failed to load categories", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(categories); err != nil {
					http.Error(w, "failed to encode response", http.StatusInternalServerError)
				}
			case http.MethodPost:
				user, ok := userFromRequest(r.Context(), db, r.Header)
				if !ok {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				payload, err := decodeCategoryPayload(r)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				category, err := createCategory(r.Context(), db, user.ID, payload.Name)
				if err != nil {
					http.Error(w, "failed to create category", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(category); err != nil {
					http.Error(w, "failed to encode response", http.StatusInternalServerError)
				}
			default:
				w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}),
	}
}

func NewCategoryHandler(db *Db) *CategoryHandler {
	return &CategoryHandler{
		Path: "/api/categories/",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := parseIDFromPath(r.URL.Path, "/api/categories/")
			if !ok {
				http.NotFound(w, r)
				return
			}

			user, ok := userFromRequest(r.Context(), db, r.Header)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			switch r.Method {
			case http.MethodPatch:
				payload, err := decodeCategoryPayload(r)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := updateCategory(r.Context(), db, user.ID, id, payload.Name); err != nil {
					if errors.Is(err, errNotFound) {
						http.NotFound(w, r)
						return
					}
					http.Error(w, "failed to update category", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			case http.MethodDelete:
				if err := deleteCategory(r.Context(), db, user.ID, id); err != nil {
					if errors.Is(err, errNotFound) {
						http.NotFound(w, r)
						return
					}
					http.Error(w, "failed to delete category", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			default:
				w.Header().Set("Allow", strings.Join([]string{http.MethodPatch, http.MethodDelete}, ", "))
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}),
	}
}

func NewCategoryRulesHandler(db *Db) *CategoryRulesHandler {
	return &CategoryRulesHandler{
		Path: "/api/category-rules",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				user, ok := userFromRequest(r.Context(), db, r.Header)
				if !ok {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				rules, err := listCategoryRules(r.Context(), db, user.ID)
				if err != nil {
					http.Error(w, "failed to load rules", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(rules); err != nil {
					http.Error(w, "failed to encode response", http.StatusInternalServerError)
				}
			case http.MethodPost:
				user, ok := userFromRequest(r.Context(), db, r.Header)
				if !ok {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				payload, err := decodeCategoryRulePayload(r)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if !categoryExists(r.Context(), db, user.ID, payload.CategoryID) {
					http.Error(w, "category not found", http.StatusBadRequest)
					return
				}
				rule, err := createCategoryRule(r.Context(), db, user.ID, payload)
				if err != nil {
					http.Error(w, "failed to create rule", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(rule); err != nil {
					http.Error(w, "failed to encode response", http.StatusInternalServerError)
				}
			default:
				w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}),
	}
}

func NewCategoryRuleHandler(db *Db) *CategoryRuleHandler {
	return &CategoryRuleHandler{
		Path: "/api/category-rules/",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := parseIDFromPath(r.URL.Path, "/api/category-rules/")
			if !ok {
				http.NotFound(w, r)
				return
			}

			user, ok := userFromRequest(r.Context(), db, r.Header)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			switch r.Method {
			case http.MethodPatch:
				payload, err := decodeCategoryRulePayload(r)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if !categoryExists(r.Context(), db, user.ID, payload.CategoryID) {
					http.Error(w, "category not found", http.StatusBadRequest)
					return
				}
				if err := updateCategoryRule(r.Context(), db, user.ID, id, payload); err != nil {
					if errors.Is(err, errNotFound) {
						http.NotFound(w, r)
						return
					}
					http.Error(w, "failed to update rule", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			case http.MethodDelete:
				if err := deleteCategoryRule(r.Context(), db, user.ID, id); err != nil {
					if errors.Is(err, errNotFound) {
						http.NotFound(w, r)
						return
					}
					http.Error(w, "failed to delete rule", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			default:
				w.Header().Set("Allow", strings.Join([]string{http.MethodPatch, http.MethodDelete}, ", "))
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}),
	}
}

func NewTransactionCategoryHandler(db *Db) *TransactionCategoryHandler {
	return &TransactionCategoryHandler{
		Path: "/api/transactions/",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch {
				w.Header().Set("Allow", http.MethodPatch)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			id, ok := parseTransactionCategoryPath(r.URL.Path)
			if !ok {
				http.NotFound(w, r)
				return
			}

			user, ok := userFromRequest(r.Context(), db, r.Header)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			payload := transactionCategoryPayload{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "invalid payload", http.StatusBadRequest)
				return
			}

			if payload.CategoryID != nil {
				if !categoryExists(r.Context(), db, user.ID, *payload.CategoryID) {
					http.Error(w, "category not found", http.StatusBadRequest)
					return
				}
			}

			if err := updateTransactionCategory(r.Context(), db, user.ID, id, payload.CategoryID); err != nil {
				if errors.Is(err, errNotFound) {
					http.NotFound(w, r)
					return
				}
				http.Error(w, "failed to update category", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}),
	}
}

var errNotFound = errors.New("not found")

func listCategories(ctx context.Context, db *Db, userID int32) ([]Category, error) {
	rows, err := db.Queries.ListCategoriesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	categories := make([]Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, Category{
			ID:        row.ID,
			Name:      row.Name,
			CreatedAt: row.CreatedAt.Time,
		})
	}
	return categories, nil
}

func createCategory(ctx context.Context, db *Db, userID int32, name string) (Category, error) {
	row, err := db.Queries.CreateCategory(ctx, dbgen.CreateCategoryParams{
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return Category{}, err
	}
	return Category{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func updateCategory(ctx context.Context, db *Db, userID int32, id int64, name string) error {
	affected, err := db.Queries.UpdateCategory(ctx, dbgen.UpdateCategoryParams{
		Name:   name,
		ID:     id,
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

func deleteCategory(ctx context.Context, db *Db, userID int32, id int64) error {
	affected, err := db.Queries.DeleteCategory(ctx, dbgen.DeleteCategoryParams{
		ID:     id,
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

func listCategoryRules(ctx context.Context, db *Db, userID int32) ([]CategoryRule, error) {
	rows, err := db.Queries.ListCategoryRulesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	rules := make([]CategoryRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, CategoryRule{
			ID:                  row.ID,
			CategoryID:          row.CategoryID,
			DescriptionContains: row.DescriptionContains,
			CreatedAt:           row.CreatedAt.Time,
		})
	}
	return rules, nil
}

func createCategoryRule(ctx context.Context, db *Db, userID int32, payload categoryRulePayload) (CategoryRule, error) {
	row, err := db.Queries.CreateCategoryRule(ctx, dbgen.CreateCategoryRuleParams{
		UserID:              userID,
		CategoryID:          payload.CategoryID,
		DescriptionContains: payload.DescriptionContains,
	})
	if err != nil {
		return CategoryRule{}, err
	}
	return CategoryRule{
		ID:                  row.ID,
		CategoryID:          row.CategoryID,
		DescriptionContains: row.DescriptionContains,
		CreatedAt:           row.CreatedAt.Time,
	}, nil
}

func updateCategoryRule(ctx context.Context, db *Db, userID int32, id int64, payload categoryRulePayload) error {
	affected, err := db.Queries.UpdateCategoryRule(ctx, dbgen.UpdateCategoryRuleParams{
		CategoryID:          payload.CategoryID,
		DescriptionContains: payload.DescriptionContains,
		ID:                  id,
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

func deleteCategoryRule(ctx context.Context, db *Db, userID int32, id int64) error {
	affected, err := db.Queries.DeleteCategoryRule(ctx, dbgen.DeleteCategoryRuleParams{
		ID:     id,
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

func categoryExists(ctx context.Context, db *Db, userID int32, id int64) bool {
	exists, err := db.Queries.CategoryExists(ctx, dbgen.CategoryExistsParams{
		ID:     id,
		UserID: userID,
	})
	return err == nil && exists
}

func updateTransactionCategory(ctx context.Context, db *Db, userID int32, transactionID int64, categoryID *int64) error {
	var category pgtype.Int8
	if categoryID != nil {
		category = pgtype.Int8{Int64: *categoryID, Valid: true}
	}
	affected, err := db.Queries.UpdateTransactionCategory(ctx, dbgen.UpdateTransactionCategoryParams{
		CategoryID: category,
		ID:         transactionID,
		UserID:     userID,
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}

func parseIDFromPath(path string, prefix string) (int64, bool) {
	if !strings.HasPrefix(path, prefix) {
		return 0, false
	}
	idPart := strings.TrimPrefix(path, prefix)
	if idPart == "" {
		return 0, false
	}
	if strings.Contains(idPart, "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func parseTransactionCategoryPath(path string) (int64, bool) {
	if !strings.HasPrefix(path, "/api/transactions/") {
		return 0, false
	}
	remainder := strings.TrimPrefix(path, "/api/transactions/")
	parts := strings.Split(remainder, "/")
	if len(parts) != 2 || parts[1] != "category" {
		return 0, false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func decodeCategoryPayload(r *http.Request) (categoryPayload, error) {
	payload := categoryPayload{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return payload, errors.New("invalid payload")
	}
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		return payload, errors.New("name is required")
	}
	return payload, nil
}

func decodeCategoryRulePayload(r *http.Request) (categoryRulePayload, error) {
	payload := categoryRulePayload{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return payload, errors.New("invalid payload")
	}
	payload.DescriptionContains = strings.TrimSpace(payload.DescriptionContains)
	if payload.DescriptionContains == "" {
		return payload, errors.New("description_contains is required")
	}
	return payload, nil
}
