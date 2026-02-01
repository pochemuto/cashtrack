-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN language VARCHAR(10) NOT NULL DEFAULT 'en';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN language;
-- +goose StatementEnd
