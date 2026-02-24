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

-- name: GetMovieDetailByID :one
SELECT
    m.*,
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'id',           g.id,
                'created_at',   g.created_at,
                'updated_at',   g.updated_at,
                'name',         g.name
            )
        ) FILTER (WHERE g.id IS NOT NULL),
        '[]'
    ) AS genres
FROM movies m
LEFT JOIN movie_genre mg ON m.id = mg.movie_id
LEFT JOIN genres       g ON mg.genre_id = g.id
WHERE m.id = $1
GROUP BY m.id;

-- name: GetMoviesSummary :many
SELECT
    m.id, m.title, m.slug, m.poster_url
FROM movies m
LEFT JOIN movie_genre mg ON m.id = mg.movie_id
WHERE (sqlc.narg('genre_id')::uuid IS NULL OR mg.genre_id = sqlc.narg('genre_id'))
    AND (sqlc.narg('title')::text IS NULL OR m.title ILIKE '%' || sqlc.narg('title') || '%')
    AND (sqlc.narg('release_date_from')::date IS NULL OR m.release_date >= sqlc.narg('release_date_from')::date)
    AND (sqlc.narg('release_date_to')::date IS NULL OR m.release_date <= sqlc.narg('release_date_to')::date)
    AND (sqlc.narg('runtime_min')::int IS NULL OR m.runtime_minutes >= sqlc.narg('runtime_min')::int)
    AND (sqlc.narg('runtime_max')::int IS NULL OR m.runtime_minutes <= sqlc.narg('runtime_max')::int)
GROUP BY m.id
ORDER BY m.title ASC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: GetCurrentMoviesSummary :many
SELECT
    m.id, m.title, m.slug, m.poster_url
FROM movies m
JOIN screenings s ON s.movie_id = m.id
WHERE s.start_time >= NOW()
GROUP BY m.id
ORDER BY m.title ASC;

-- name: DeleteMovie :exec
DELETE FROM movies
WHERE id = $1;

-- name: AssignGenresToMovie :exec
INSERT INTO movie_genre (movie_id, genre_id, created_at, updated_at)
VALUES (
    $1,
    unnest(sqlc.arg('genre_ids')::uuid[]),
    NOW(),
    NOW()
);

-- name: UploadMoviePoster :exec
UPDATE movies
SET poster_url = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteMovieGenres :exec
DELETE FROM movie_genre
WHERE movie_id = $1;