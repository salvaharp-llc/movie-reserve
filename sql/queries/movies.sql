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
SET updated_at = NOW(), title = $2, slug = $3, description = $4, runtime_minutes = $5, release_date = $6, poster_url = $7
WHERE id = $1
RETURNING *;

-- name: GetMovieBySlug :one
SELECT * FROM movies
WHERE slug = $1;

-- name: GetMovieByID :one
SELECT * FROM movies
WHERE id = $1;

-- name: GetMoviesByGenre :many
SELECT DISTINCT m.*
FROM movies m
INNER JOIN movie_genre mg ON m.id = mg.movie_id
INNER JOIN genres g ON mg.genre_id = g.id
WHERE g.id = $1
ORDER BY m.created_at DESC;

-- name: DeleteMovie :exec
DELETE FROM movies
WHERE id = $1;

-- name: AssignGenresToMovie :exec
INSERT INTO movie_genre (movie_id, genre_id, created_at, updated_at)
SELECT $1, unnest($2::uuid[]), NOW(), NOW()
ON CONFLICT DO NOTHING;

-- name: DeleteMovieGenres :exec
DELETE FROM movie_genre
WHERE movie_id = $1;