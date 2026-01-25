-- name: ListTodosByUser :many
SELECT id, title FROM todo WHERE user_id = $1 ORDER BY id;

-- name: AddTodo :exec
INSERT INTO todo (title, user_id) VALUES ($1, $2);

-- name: RemoveTodo :exec
DELETE FROM todo WHERE id = $1 AND user_id = $2;

-- name: AddTodosBatch :exec
INSERT INTO todo (title, user_id)
SELECT unnest(sqlc.arg(titles)::text[]) AS title, sqlc.arg(user_id) AS user_id;

-- name: ListPendingReports :many
SELECT id, user_id, filename, data
FROM financial_reports
WHERE status = 'pending'
ORDER BY uploaded_at ASC, id ASC;

-- name: UpdateReportStatus :exec
UPDATE financial_reports
SET status = $1
WHERE id = $2 AND user_id = $3;

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
    $13
);

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
ORDER BY posted_date DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);
