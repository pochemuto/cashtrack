-- name: ListTodos :many
SELECT * FROM todo;

-- name: AddTodo :exec
INSERT INTO todo (title) VALUES ($1);

-- name: RemoveTodo :exec
DELETE FROM todo WHERE id = $1;
