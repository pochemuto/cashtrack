-- +goose Up
CREATE TABLE transactions (
    id bigserial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_file_id bigint NOT NULL REFERENCES financial_reports(id) ON DELETE CASCADE,
    source_file_row integer NOT NULL,
    parser_name varchar(64) NOT NULL,
    posted_date date NOT NULL,
    description text NOT NULL,
    amount numeric(18, 2) NOT NULL,
    currency varchar(3) NOT NULL,
    transaction_id text,
    entry_type varchar(16) NOT NULL,
    source_account_number varchar(64),
    source_card_number varchar(64),
    parser_meta jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX transactions_user_id_idx ON transactions(user_id);
CREATE INDEX transactions_source_file_id_idx ON transactions(source_file_id);
CREATE INDEX transactions_posted_date_idx ON transactions(posted_date);
CREATE INDEX transactions_entry_type_idx ON transactions(entry_type);
CREATE INDEX transactions_source_account_number_idx ON transactions(source_account_number);
CREATE INDEX transactions_source_card_number_idx ON transactions(source_card_number);
CREATE INDEX transactions_description_tsv_idx ON transactions USING GIN (to_tsvector('simple', description));

-- +goose Down
DROP TABLE transactions;
