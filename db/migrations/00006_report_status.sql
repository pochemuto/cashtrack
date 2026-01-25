-- +goose Up
ALTER TABLE financial_reports
ADD COLUMN status varchar(32) NOT NULL DEFAULT 'pending';

-- +goose Down
ALTER TABLE financial_reports
DROP COLUMN status;
