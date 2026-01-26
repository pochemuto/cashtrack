package cashtrack

import (
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
	rows, err := db.conn.Query(ctx, "SELECT id, name, created_at FROM categories WHERE user_id = $1 ORDER BY name", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var item Category
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, item)
	}
	return categories, rows.Err()
}

func createCategory(ctx context.Context, db *Db, userID int32, name string) (Category, error) {
	var item Category
	err := db.conn.QueryRow(ctx, "INSERT INTO categories (user_id, name) VALUES ($1, $2) RETURNING id, name, created_at", userID, name).Scan(&item.ID, &item.Name, &item.CreatedAt)
	return item, err
}

func updateCategory(ctx context.Context, db *Db, userID int32, id int64, name string) error {
	cmd, err := db.conn.Exec(ctx, "UPDATE categories SET name = $1 WHERE id = $2 AND user_id = $3", name, id, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errNotFound
	}
	return nil
}

func deleteCategory(ctx context.Context, db *Db, userID int32, id int64) error {
	cmd, err := db.conn.Exec(ctx, "DELETE FROM categories WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errNotFound
	}
	return nil
}

func listCategoryRules(ctx context.Context, db *Db, userID int32) ([]CategoryRule, error) {
	rows, err := db.conn.Query(ctx, "SELECT id, category_id, description_contains, created_at FROM category_rules WHERE user_id = $1 ORDER BY id", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []CategoryRule
	for rows.Next() {
		var item CategoryRule
		if err := rows.Scan(&item.ID, &item.CategoryID, &item.DescriptionContains, &item.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, item)
	}
	return rules, rows.Err()
}

func createCategoryRule(ctx context.Context, db *Db, userID int32, payload categoryRulePayload) (CategoryRule, error) {
	var item CategoryRule
	err := db.conn.QueryRow(ctx, "INSERT INTO category_rules (user_id, category_id, description_contains) VALUES ($1, $2, $3) RETURNING id, category_id, description_contains, created_at", userID, payload.CategoryID, payload.DescriptionContains).Scan(&item.ID, &item.CategoryID, &item.DescriptionContains, &item.CreatedAt)
	return item, err
}

func updateCategoryRule(ctx context.Context, db *Db, userID int32, id int64, payload categoryRulePayload) error {
	cmd, err := db.conn.Exec(ctx, "UPDATE category_rules SET category_id = $1, description_contains = $2 WHERE id = $3 AND user_id = $4", payload.CategoryID, payload.DescriptionContains, id, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errNotFound
	}
	return nil
}

func deleteCategoryRule(ctx context.Context, db *Db, userID int32, id int64) error {
	cmd, err := db.conn.Exec(ctx, "DELETE FROM category_rules WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errNotFound
	}
	return nil
}

func categoryExists(ctx context.Context, db *Db, userID int32, id int64) bool {
	var exists bool
	err := db.conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1 AND user_id = $2)", id, userID).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func updateTransactionCategory(ctx context.Context, db *Db, userID int32, transactionID int64, categoryID *int64) error {
	var category pgtype.Int8
	if categoryID != nil {
		category = pgtype.Int8{Int64: *categoryID, Valid: true}
	}

	cmd, err := db.conn.Exec(ctx, "UPDATE transactions SET category_id = $1 WHERE id = $2 AND user_id = $3", category, transactionID, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
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
