-- +goose Up
CREATE TABLE public.categories (
    id bigserial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    name character varying(255) NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX categories_user_id_idx ON public.categories USING btree (user_id);

CREATE TABLE public.category_rules (
    id bigserial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    category_id bigint NOT NULL REFERENCES public.categories(id) ON DELETE CASCADE,
    description_contains text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX category_rules_user_id_idx ON public.category_rules USING btree (user_id);
CREATE INDEX category_rules_category_id_idx ON public.category_rules USING btree (category_id);

ALTER TABLE public.transactions
ADD COLUMN category_id bigint REFERENCES public.categories(id) ON DELETE SET NULL;

CREATE INDEX transactions_category_id_idx ON public.transactions USING btree (category_id);

-- +goose Down
DROP INDEX IF EXISTS transactions_category_id_idx;
ALTER TABLE public.transactions
DROP COLUMN IF EXISTS category_id;

DROP INDEX IF EXISTS category_rules_category_id_idx;
DROP INDEX IF EXISTS category_rules_user_id_idx;
DROP TABLE IF EXISTS public.category_rules;

DROP INDEX IF EXISTS categories_user_id_idx;
DROP TABLE IF EXISTS public.categories;
