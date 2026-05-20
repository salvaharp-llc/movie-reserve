-- name: CreateEmailVerification :one
INSERT INTO email_verifications (id, user_id, code, created_at, expires_at)
VALUES (
    gen_random_uuid(),
    $1,
    $2,
    NOW(),
    $3
)
RETURNING *;