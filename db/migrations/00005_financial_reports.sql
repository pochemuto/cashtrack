-- +goose Up
CREATE TABLE financial_reports (
    id bigserial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename varchar(255) NOT NULL,
    content_type varchar(255),
    data bytea NOT NULL,
    uploaded_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX financial_reports_user_id_idx ON financial_reports(user_id);

-- +goose Down
DROP TABLE financial_reports;
