-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: MakeAdmin :one
UPDATE users SET role = 'admin', updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET hashed_password = $2, updated_at = NOW()
WHERE id = $1;

-- name: VerifyUser :exec
UPDATE users SET is_active = true, updated_at = NOW()
WHERE id = $1;