-- name: Reset :exec
TRUNCATE TABLE users, movies RESTART IDENTITY CASCADE;
