-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE UNIQUE INDEX users_username_idx ON users (username);

CREATE TABLE sessions (
    id uuid NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    user_id integer REFERENCES users(id) ON DELETE CASCADE,
    expires timestamp with time zone NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
DROP TABLE sessions;
-- +goose StatementEnd
