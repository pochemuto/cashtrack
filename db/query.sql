-- name: ListTodosByUser :many
SELECT id, title FROM todo WHERE user_id = $1 ORDER BY id;

-- name: AddTodo :exec
INSERT INTO todo (title, user_id) VALUES ($1, $2);

-- name: RemoveTodo :exec
DELETE FROM todo WHERE id = $1 AND user_id = $2;

-- name: AddTodosBatch :exec
INSERT INTO todo (title, user_id)
SELECT unnest(sqlc.arg(titles)::text[]) AS title, sqlc.arg(user_id) AS user_id;

-- name: GetUserByUsername :one
SELECT id, username
FROM users
WHERE username = $1;

-- name: CreateUser :one
INSERT INTO users (username, password)
VALUES ($1, $2)
RETURNING id, username;

-- name: CreateSession :one
INSERT INTO sessions (user_id, expires)
VALUES ($1, $2)
RETURNING id::text;

-- name: DeleteSession :execrows
DELETE FROM sessions
WHERE id = $1;

-- name: GetUserBySession :one
SELECT u.id, u.username, s.expires
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.id = $1;

-- name: CreateReport :exec
INSERT INTO financial_reports (user_id, filename, content_type, data, status)
VALUES ($1, $2, $3, $4, $5);

-- name: ListReportsByUser :many
SELECT id,
       filename,
       octet_length(data) AS size_bytes,
       status,
       uploaded_at,
       status_description
FROM financial_reports
WHERE user_id = $1
ORDER BY uploaded_at DESC, id DESC;

-- name: GetReportByID :one
SELECT filename, content_type, data
FROM financial_reports
WHERE id = $1 AND user_id = $2;

-- name: GetExchangeRate :one
SELECT rate
FROM exchange_rates
WHERE rate_date = $1
  AND base_currency = $2
  AND target_currency = $3;

-- name: UpsertExchangeRate :exec
INSERT INTO exchange_rates (rate_date, base_currency, target_currency, rate)
VALUES ($1, $2, $3, $4)
ON CONFLICT (rate_date, base_currency, target_currency)
DO UPDATE SET rate = EXCLUDED.rate;

-- name: ListCategoriesByUser :many
SELECT id, name, color, created_at
FROM categories
WHERE user_id = $1
ORDER BY name;

-- name: CreateCategory :one
INSERT INTO categories (user_id, name, color)
VALUES ($1, $2, $3)
RETURNING id, name, color, created_at;

-- name: UpdateCategory :execrows
UPDATE categories
SET name = $1,
    color = $2
WHERE id = $3 AND user_id = $4;

-- name: DeleteCategory :execrows
DELETE FROM categories
WHERE id = $1 AND user_id = $2;

-- name: CategoryExists :one
SELECT EXISTS(
    SELECT 1
    FROM categories
    WHERE id = $1 AND user_id = $2
);

-- name: ListCategoryRulesByUser :many
SELECT id, category_id, description_contains, position, created_at
FROM category_rules
WHERE user_id = $1
ORDER BY position, id;

-- name: CreateCategoryRule :one
INSERT INTO category_rules (user_id, category_id, description_contains, position)
VALUES (
    $1,
    $2,
    $3,
    COALESCE((SELECT MAX(position) FROM category_rules WHERE user_id = $1), 0) + 1
)
RETURNING id, category_id, description_contains, position, created_at;

-- name: UpdateCategoryRule :execrows
UPDATE category_rules
SET category_id = $1,
    description_contains = $2
WHERE id = $3 AND user_id = $4;

-- name: UpdateCategoryRulePosition :execrows
UPDATE category_rules
SET position = $1
WHERE id = $2 AND user_id = $3;

-- name: DeleteCategoryRule :execrows
DELETE FROM category_rules
WHERE id = $1 AND user_id = $2;

-- name: UpdateTransactionCategory :execrows
UPDATE transactions
SET category_id = $1,
    category_source = $2
WHERE id = $3 AND user_id = $4;

-- name: ListPendingReports :many
SELECT id, user_id, filename, data
FROM financial_reports
WHERE status = 'pending'
ORDER BY uploaded_at ASC, id ASC;

-- name: UpdateReportStatus :exec
UPDATE financial_reports
SET status = $1
WHERE id = $2 AND user_id = $3;

-- name: UpdateReportStatusWithError :exec
UPDATE financial_reports
SET status = $1,
    status_description = $2
WHERE id = $3 AND user_id = $4;

-- name: DeleteReportByID :exec
DELETE FROM financial_reports
WHERE id = $1 AND user_id = $2;

-- name: DeleteTransactionsBySource :exec
DELETE FROM transactions
WHERE source_file_id = $1 AND user_id = $2;

-- name: CreateTransaction :exec
INSERT INTO transactions (
    user_id,
    source_file_id,
    source_file_row,
    parser_name,
    posted_date,
    description,
    amount,
    currency,
    transaction_id,
    entry_type,
    source_account_number,
    source_card_number,
    category_id,
    category_source,
    parser_meta
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14,
    $15
);

-- name: ListTransactionsForRuleApply :many
SELECT id,
       description,
       category_id,
       category_source
FROM transactions
WHERE user_id = $1
  AND ($2::boolean OR category_source IS DISTINCT FROM 'manual');

-- name: SummaryTransactions :one
SELECT
    COUNT(*) AS count,
    COALESCE(SUM(amount), 0::numeric)::text AS total_amount,
    COALESCE(AVG(amount), 0::numeric)::text AS average_amount,
    COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount), 0::numeric)::text AS median_amount
FROM transactions
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(from_date)::date IS NULL OR posted_date >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date IS NULL OR posted_date <= sqlc.narg(to_date))
  AND (sqlc.narg(source_file_id)::bigint IS NULL OR source_file_id = sqlc.narg(source_file_id))
  AND (sqlc.narg(entry_type)::text IS NULL OR entry_type = sqlc.narg(entry_type))
  AND (sqlc.narg(source_account_number)::text IS NULL OR source_account_number = sqlc.narg(source_account_number))
  AND (sqlc.narg(source_card_number)::text IS NULL OR source_card_number = sqlc.narg(source_card_number))
  AND (sqlc.narg(search_text)::text IS NULL OR to_tsvector('simple', description) @@ plainto_tsquery('simple', sqlc.narg(search_text)))
  AND (sqlc.narg(category_id)::bigint IS NULL OR category_id = sqlc.narg(category_id));

-- name: ListTransactionsSummaryRows :many
SELECT posted_date, amount, currency
FROM transactions
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(from_date)::date IS NULL OR posted_date >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date IS NULL OR posted_date <= sqlc.narg(to_date))
  AND (sqlc.narg(source_file_id)::bigint IS NULL OR source_file_id = sqlc.narg(source_file_id))
  AND (sqlc.narg(entry_type)::text IS NULL OR entry_type = sqlc.narg(entry_type))
  AND (sqlc.narg(source_account_number)::text IS NULL OR source_account_number = sqlc.narg(source_account_number))
  AND (sqlc.narg(source_card_number)::text IS NULL OR source_card_number = sqlc.narg(source_card_number))
  AND (sqlc.narg(search_text)::text IS NULL OR to_tsvector('simple', description) @@ plainto_tsquery('simple', sqlc.narg(search_text)))
  AND (sqlc.narg(category_id)::bigint IS NULL OR category_id = sqlc.narg(category_id));

-- name: ListTransactions :many
SELECT id,
       source_file_id,
       source_file_row,
       parser_name,
       posted_date,
       description,
       amount,
       currency,
       transaction_id,
       entry_type,
       source_account_number,
       source_card_number,
       category_id,
       parser_meta,
       created_at
FROM transactions
WHERE user_id = $1
  AND (sqlc.narg(from_date)::date IS NULL OR posted_date >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date IS NULL OR posted_date <= sqlc.narg(to_date))
  AND (sqlc.narg(source_file_id)::bigint IS NULL OR source_file_id = sqlc.narg(source_file_id))
  AND (sqlc.narg(entry_type)::text IS NULL OR entry_type = sqlc.narg(entry_type))
  AND (sqlc.narg(source_account_number)::text IS NULL OR source_account_number = sqlc.narg(source_account_number))
  AND (sqlc.narg(source_card_number)::text IS NULL OR source_card_number = sqlc.narg(source_card_number))
  AND (sqlc.narg(search_text)::text IS NULL OR to_tsvector('simple', description) @@ plainto_tsquery('simple', sqlc.narg(search_text)))
  AND (sqlc.narg(category_id)::bigint IS NULL OR category_id = sqlc.narg(category_id))
ORDER BY posted_date DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);
