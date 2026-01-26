-- +goose Up
CREATE TABLE public.exchange_rates (
    id bigserial PRIMARY KEY,
    rate_date date NOT NULL,
    base_currency character varying(3) NOT NULL,
    target_currency character varying(3) NOT NULL,
    rate numeric(18, 8) NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX exchange_rates_unique_idx ON public.exchange_rates (rate_date, base_currency, target_currency);

-- +goose Down
DROP INDEX IF EXISTS exchange_rates_unique_idx;
DROP TABLE IF EXISTS public.exchange_rates;
