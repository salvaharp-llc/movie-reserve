-- name: CreateEmailVerification :one
INSERT INTO email_verifications (id, user_id, user_email, code, created_at, expires_at)
VALUES (
    gen_random_uuid(),
    $1,
    $2,
    $3,
    NOW(),
    $4
)
RETURNING *;

-- name: GetEmailVerificationByEmail :one
SELECT *
FROM email_verifications
WHERE user_email = $1
ORDER BY created_at DESC
LIMIT 1;