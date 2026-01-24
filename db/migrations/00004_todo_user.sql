-- +goose Up
-- +goose StatementBegin
INSERT INTO users (id, username, password) OVERRIDING SYSTEM VALUE
VALUES (1, 'legacy', 'legacy')
ON CONFLICT DO NOTHING;

ALTER TABLE todo ADD COLUMN user_id integer;
UPDATE todo SET user_id = 1 WHERE user_id IS NULL;

ALTER TABLE todo ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE todo
    ADD CONSTRAINT todo_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX todo_user_id_idx ON todo (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS todo_user_id_idx;
ALTER TABLE todo DROP CONSTRAINT IF EXISTS todo_user_id_fkey;
ALTER TABLE todo DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
