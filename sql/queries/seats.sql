-- name: CreateSeat :one
INSERT INTO seats (id, room_id, row_label, seat_number, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    $1,
    $2,
    $3,
    NOW(),
    NOW()
)
RETURNING *;

-- name: UpdateSeat :one
UPDATE seats
SET updated_at = NOW(), room_id = $2, row_label = $3, seat_number = $4
WHERE id = $1
RETURNING *;

-- name: DeleteSeat :exec
DELETE FROM seats
WHERE id = $1;

-- name: GetSeatByID :one
SELECT * FROM seats
WHERE id = $1;

-- name: GetSeatsByRoomID :many
SELECT * FROM seats
WHERE room_id = $1
ORDER BY row_label, seat_number;

-- name: GetSeatsForScreening :many
SELECT s.*, (CASE WHEN r.id IS NOT NULL THEN false ELSE true END) AS is_available
FROM seats s
JOIN screenings sc ON sc.room_id = s.room_id
LEFT JOIN reservations r ON r.seat_id = s.id AND r.screening_id = $1
WHERE sc.id = $1
ORDER BY s.row_label, s.seat_number;