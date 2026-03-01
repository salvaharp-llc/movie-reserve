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
    sc.id,
    sc.start_time,
    sc.end_time,
    sc.created_at,
    sc.updated_at,
    sc.movie_id,
    m.title         AS movie_title,
    m.slug          AS movie_slug,
    m.poster_url    AS movie_poster_url,
    sc.room_id,
    r.name          AS room_name,
    COALESCE(seats_agg.seats, '[]') AS seats
FROM screenings sc
JOIN rooms r ON sc.room_id = r.id
JOIN movies m ON sc.movie_id = m.id
LEFT JOIN (
    SELECT
        s.room_id,
        jsonb_agg(
            jsonb_build_object(
                'id',          s.id,
                'row_label',   s.row_label,
                'seat_number', s.seat_number,
                'available',   (res.id IS NULL)
            )
        ) AS seats
    FROM seats s
    LEFT JOIN reservations res 
        ON s.id = res.seat_id 
        AND res.screening_id = $1
    GROUP BY s.room_id
) seats_agg ON r.id = seats_agg.room_id
WHERE sc.id = $1;

-- name: GetScreeningsSummary :many
SELECT sc.id, sc.start_time, sc.end_time,
sc.movie_id,
m.title AS movie_title,
m.slug AS movie_slug,
m.poster_url AS movie_poster_url
FROM screenings sc
JOIN rooms r ON sc.room_id = r.id
JOIN movies m ON sc.movie_id = m.id
WHERE (sqlc.narg('movie_id')::uuid IS NULL OR sc.movie_id = sqlc.narg('movie_id'))
  AND (sqlc.narg('room_id')::uuid IS NULL OR sc.room_id = sqlc.narg('room_id'))
  AND (sqlc.narg('from')::timestamp IS NULL OR sc.start_time >= sqlc.narg('from'))
  AND (sqlc.narg('to')::timestamp IS NULL OR sc.start_time <= sqlc.narg('to'))
ORDER BY sc.start_time
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');