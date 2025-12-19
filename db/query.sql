-- name: ListTodos :many
SELECT * FROM todo;

-- name: AddTodo :exec
INSERT INTO todo (id, title) VALUES ($1, $2);

-- name: RemoveTodo :exec
DELETE FROM todo WHERE id = $1;
