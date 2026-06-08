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

-- name: GetRefreshToken :one
SELECT rt.*, u.role
FROM refresh_tokens rt
JOIN users u ON rt.user_id = u.id
WHERE rt.token = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET updated_at = NOW(), revoked_at = NOW()
WHERE token = $1;

-- name: RevokeRefreshTokens :exec
UPDATE refresh_tokens
SET updated_at = NOW(), revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;