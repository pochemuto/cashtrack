-- +goose Up
ALTER TABLE categories
ADD COLUMN color character varying(7);

-- +goose Down
ALTER TABLE categories
DROP COLUMN IF EXISTS color;
