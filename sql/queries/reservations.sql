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

-- name: GetReservationByID :one
SELECT * FROM reservations 
WHERE id = $1;

-- name: GetReservationsByUserID :many
SELECT * FROM reservations
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetReservationsAdmin :many
SELECT r.* FROM reservations r
JOIN screenings s ON r.screening_id = s.id
WHERE (sqlc.narg('user_id')::uuid IS NULL OR r.user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('screening_id')::uuid IS NULL OR r.screening_id = sqlc.narg('screening_id'))
  AND (sqlc.narg('movie_id')::uuid IS NULL OR s.movie_id = sqlc.narg('movie_id'))
  AND (sqlc.narg('room_id')::uuid IS NULL OR r.room_id = sqlc.narg('room_id'))
ORDER BY r.created_at DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: UpdateReservation :one
UPDATE reservations
SET updated_at = NOW(), user_id = $2, screening_id = $3, room_id = s.room_id, seat_id = $4
FROM screenings s
WHERE reservations.id = $1
  AND s.id = $3
RETURNING reservations.*;

-- name: DeleteReservation :exec
DELETE FROM reservations
WHERE id = $1;