-- +goose Up
ALTER TABLE category_rules
ADD COLUMN position integer NOT NULL DEFAULT 1;

WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at, id) AS rn
    FROM category_rules
)
UPDATE category_rules
SET position = ranked.rn
FROM ranked
WHERE category_rules.id = ranked.id;

ALTER TABLE category_rules
ALTER COLUMN position DROP DEFAULT;

CREATE INDEX category_rules_user_position_idx ON public.category_rules USING btree (user_id, position);

-- +goose Down
DROP INDEX IF EXISTS category_rules_user_position_idx;

ALTER TABLE category_rules
DROP COLUMN IF EXISTS position;
