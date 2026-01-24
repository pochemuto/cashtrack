-- name: ListTodosByUser :many
SELECT id, title FROM todo WHERE user_id = $1 ORDER BY id;

-- name: AddTodo :exec
INSERT INTO todo (title, user_id) VALUES ($1, $2);

-- name: RemoveTodo :exec
DELETE FROM todo WHERE id = $1 AND user_id = $2;

-- name: AddTodosBatch :exec
INSERT INTO todo (title, user_id)
SELECT unnest(sqlc.arg(titles)::text[]) AS title, sqlc.arg(user_id) AS user_id;
