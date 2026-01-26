-- +goose Up
ALTER TABLE financial_reports
ADD COLUMN status_description text;

-- +goose Down
ALTER TABLE financial_reports
DROP COLUMN status_description;
