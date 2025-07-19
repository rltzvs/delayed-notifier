-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE notify (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    send_at TIMESTAMPTZ NOT NULL,
    message TEXT NOT NULL,
    status TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE notify;
-- +goose StatementEnd
