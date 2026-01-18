-- name: CreateMovie :one
INSERT INTO movies (id, created_at, updated_at, title, slug, description, runtime_minutes, release_date, poster_url)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: UpdateMovie :one
UPDATE movies
SET title = $2, slug = $3, description = $4, runtime_minutes = $5, release_date = $6, poster_url = $7
WHERE id = $1
RETURNING *;

-- name: GetMovieBySlug :one
SELECT * FROM movies
WHERE slug = $1;

-- name: GetMoviesByGenre :many
SELECT DISTINCT m.*
FROM movies m
INNER JOIN movie_genre mg ON m.id = mg.movie_id
INNER JOIN genres g ON mg.genre_id = g.id
WHERE g.id = $1
ORDER BY m.created_at DESC;

-- name: CreateGenre :one
INSERT INTO genres (id, created_at, updated_at, name)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1
)
RETURNING *;