-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3
)
RETURNING *;

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens
SET updated_at = NOW(), revoked_at = NOW()
FROM users
WHERE refresh_tokens.token = $1
  AND refresh_tokens.revoked_at IS NULL
  AND refresh_tokens.expires_at > NOW()
  AND users.id = refresh_tokens.user_id
RETURNING refresh_tokens.user_id, users.is_admin;