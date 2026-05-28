-- name: CreatePasswordResetToken :one
INSERT INTO pw_reset_tokens (hashed_token, created_at, updated_at, user_id, expires_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3
)
RETURNING *;

-- name: GetPasswordResetToken :one
SELECT *
FROM pw_reset_tokens
WHERE hashed_token = $1;

-- name: RevokePasswordResetToken :exec
UPDATE pw_reset_tokens
SET updated_at = NOW(), revoked_at = NOW()
WHERE hashed_token = $1;
