-- name: CreateGenre :one
INSERT INTO genres (id, created_at, updated_at, name)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1
)
RETURNING *;

-- name: UpdateGenre :one
UPDATE genres
SET updated_at = NOW(), name = $2
WHERE id = $1
RETURNING *;

-- name: DeleteGenre :exec
DELETE FROM genres
WHERE id = $1;

-- name: GetGenreByID :one
SELECT * FROM genres
WHERE id = $1;

-- name: GetGenresByIDs :many
SELECT * FROM genres
WHERE id = ANY($1::uuid[])
ORDER BY name;

-- name: GetGenres :many
SELECT * FROM genres
ORDER BY name;

-- name: GetGenresByMovieID :many
SELECT g.*
FROM genres g
INNER JOIN movie_genre mg ON g.id = mg.genre_id
WHERE mg.movie_id = $1
ORDER BY g.name;