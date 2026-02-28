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

-- name: GetScreeningDetailByID :one
SELECT
    sc.*,
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'id',           s.id,
                'room_id',      s.room_id,
                'row_label',    s.row_label,
                'seat_number',  s.seat_number,
                'created_at',   s.created_at,
                'updated_at',   s.updated_at,
                'available',    (res.id IS NULL)
            )
        ) FILTER (WHERE s.id IS NOT NULL),
        '[]'
    ) AS seats
FROM screenings sc
LEFT JOIN rooms r ON sc.room_id = r.id
LEFT JOIN seats s ON r.id = s.room_id
LEFT JOIN reservations res ON s.id = res.seat_id AND res.screening_id = sc.id
WHERE sc.id = $1
GROUP BY sc.id;

-- name: GetScreeningsSummary :many
SELECT s.id, s.movie_id, s.room_id, s.start_time, s.end_time FROM screenings s
WHERE (sqlc.narg('movie_id')::uuid IS NULL OR s.movie_id = sqlc.narg('movie_id'))
  AND (sqlc.narg('room_id')::uuid IS NULL OR s.room_id = sqlc.narg('room_id'))
  AND (sqlc.narg('from')::timestamp IS NULL OR s.start_time >= sqlc.narg('from'))
  AND (sqlc.narg('to')::timestamp IS NULL OR s.start_time <= sqlc.narg('to'))
ORDER BY start_time
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');