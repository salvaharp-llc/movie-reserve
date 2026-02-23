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

-- name: GetScreenings :many
SELECT * FROM screenings s
WHERE (sqlc.narg('movie_id')::uuid IS NULL OR s.movie_id = sqlc.narg('movie_id'))
  AND (sqlc.narg('room_id')::uuid IS NULL OR s.room_id = sqlc.narg('room_id'))
  AND (sqlc.narg('from')::timestampt IS NULL OR s.start_time >= sqlc.narg('from'))
  AND (sqlc.narg('to')::timestampt IS NULL OR s.start_time <= sqlc.narg('to'))
  AND (sqlc.arg('upcoming')::bool IS FALSE OR s.start_time >= NOW())
ORDER BY start_time
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');