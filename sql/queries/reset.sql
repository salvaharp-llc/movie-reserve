-- name: Reset :exec
TRUNCATE TABLE users, movies, genres RESTART IDENTITY CASCADE;
