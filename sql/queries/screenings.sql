-- name: CreateScreening :one
INSERT INTO screenings (id, created_at, updated_at, movie_id, room_id, start_time, end_time)
VALUES (    
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: UpdateScreening :one
UPDATE screenings
SET updated_at = NOW(), movie_id = $2, room_id = $3, start_time = $4, end_time = $5
WHERE id = $1
RETURNING *;

-- name: DeleteScreening :exec
DELETE FROM screenings
WHERE id = $1;

-- name: GetScreeningByID :one
SELECT * FROM screenings
WHERE id = $1;

-- name: GetScreeningsPaginated :many
SELECT * FROM screenings
ORDER BY start_time
LIMIT $1 OFFSET $2;

-- name: GetScreeningsByMovieID :many
SELECT * FROM screenings
WHERE movie_id = $1
ORDER BY start_time;

-- name: GetUpcomingScreeningsByMovieID :many
SELECT * FROM screenings
WHERE movie_id = $1 AND start_time >= NOW()
ORDER BY start_time;

-- name: GetScreeningsByDateRange :many
SELECT * FROM screenings
WHERE start_time >= $1 AND start_time <= $2
ORDER BY start_time;

-- name: GetUpcomingScreeningsByDateRange :many
SELECT * FROM screenings
WHERE start_time >= NOW() AND start_time >= $1 AND start_time <= $2
ORDER BY start_time;

-- name: GetScreeningsByMovieIDAndDateRange :many
SELECT * FROM screenings
WHERE movie_id = $1 AND start_time >= $2 AND start_time <= $3
ORDER BY start_time;

-- name: GetUpcomingScreeningsByMovieIDAndDateRange :many
SELECT * FROM screenings
WHERE movie_id = $1 AND start_time >= NOW() AND start_time >= $2 AND start_time <= $3
ORDER BY start_time;