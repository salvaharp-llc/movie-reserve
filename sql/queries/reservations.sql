-- name: CreateReservation :one
INSERT INTO reservations (id, created_at, updated_at, user_id, screening_id, room_id, seat_id)
SELECT
    gen_random_uuid(), 
    NOW(), 
    NOW(),
    $1,
    $2,
    s.room_id,
    $3
FROM screenings s
WHERE s.id = $2
RETURNING *;

-- name: GetReservationDetailByID :one
SELECT 
r.id, 
r.created_at,
r.updated_at,
u.id AS user_id,
u.email AS user_email,
sc.id AS screening_id, 
sc.start_time AS screening_start_time, 
sc.end_time AS screening_end_time,  
m.id AS movie_id,
m.title AS movie_title,
m.slug AS movie_slug,
m.poster_url AS movie_poster_url,
rm.id AS room_id,
rm.name AS room_name,
s.id AS seat_id,
s.row_label AS seat_row_label,
s.seat_number AS seat_number
FROM reservations r
JOIN screenings sc ON r.screening_id = sc.id
JOIN movies m ON sc.movie_id = m.id
JOIN seats s ON r.seat_id = s.id
JOIN users u ON r.user_id = u.id
JOIN rooms rm ON r.room_id = rm.id
WHERE r.id = $1;

-- name: GetReservationMetaByID :one
SELECT r.*, sc.start_time AS screening_start_time
FROM reservations r
JOIN screenings sc ON r.screening_id = sc.id
WHERE r.id = $1;

-- name: GetReservationsSummary :many
SELECT 
r.id, 
sc.id AS screening_id, 
sc.start_time AS screening_start_time, 
sc.end_time AS screening_end_time,  
m.id AS movie_id,
m.title AS movie_title,
m.slug AS movie_slug,
m.poster_url AS movie_poster_url,
s.id AS seat_id,
s.row_label AS seat_row_label,
s.seat_number AS seat_number
FROM reservations r
JOIN screenings sc ON r.screening_id = sc.id
JOIN movies m ON sc.movie_id = m.id
JOIN seats s ON r.seat_id = s.id
WHERE (sqlc.narg('user_id')::uuid IS NULL OR r.user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('screening_id')::uuid IS NULL OR r.screening_id = sqlc.narg('screening_id'))
  AND (sqlc.narg('movie_id')::uuid IS NULL OR sc.movie_id = sqlc.narg('movie_id'))
  AND (sqlc.narg('room_id')::uuid IS NULL OR r.room_id = sqlc.narg('room_id'))
ORDER BY r.created_at DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: DeleteReservation :exec
DELETE FROM reservations
WHERE id = $1;