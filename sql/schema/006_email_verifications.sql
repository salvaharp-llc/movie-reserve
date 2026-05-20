-- +goose Up
ALTER TABLE users
    ADD is_active BOOL NOT NULL DEFAULT false;

CREATE TABLE email_verifications(
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code INT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE email_verifications;

ALTER TABLE users
    DROP is_active;
