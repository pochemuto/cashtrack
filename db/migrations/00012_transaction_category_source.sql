-- +goose Up
ALTER TABLE transactions
ADD COLUMN category_source text;

ALTER TABLE transactions
ADD CONSTRAINT transactions_category_source_check
CHECK (category_source IN ('manual', 'rule') OR category_source IS NULL);

UPDATE transactions
SET category_source = 'manual'
WHERE category_id IS NOT NULL;

-- +goose Down
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS transactions_category_source_check;

ALTER TABLE transactions
DROP COLUMN IF EXISTS category_source;
