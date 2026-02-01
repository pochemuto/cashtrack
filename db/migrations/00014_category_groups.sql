-- +goose Up
ALTER TABLE public.categories
    ADD COLUMN parent_id bigint REFERENCES public.categories(id) ON DELETE SET NULL,
    ADD COLUMN is_group boolean NOT NULL DEFAULT false;

CREATE INDEX categories_parent_id_idx ON public.categories USING btree (parent_id);

-- +goose Down
DROP INDEX IF EXISTS categories_parent_id_idx;
ALTER TABLE public.categories
    DROP COLUMN IF EXISTS parent_id,
    DROP COLUMN IF EXISTS is_group;
