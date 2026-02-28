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

-- name: GetSeatDetailByID :one
SELECT s.*, r.name AS room_name FROM seats s
JOIN rooms r ON s.room_id = r.id
WHERE s.id = $1;
