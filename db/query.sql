-- name: ListTodos :many
SELECT * FROM todo;

-- name: AddTodo :exec
INSERT INTO todo (title) VALUES ($1);

-- name: RemoveTodo :exec
DELETE FROM todo WHERE id = $1;

-- name: AddTodosBatch :exec
INSERT INTO todo (title)
SELECT unnest($1::text[]);
