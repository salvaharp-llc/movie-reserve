-- name: CreateRoom :one
INSERT INTO rooms (id, created_at, updated_at, name)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1
)
RETURNING *;

-- name: UpdateRoom :one
UPDATE rooms
SET updated_at = NOW(), name = $2
WHERE id = $1
RETURNING *;

-- name: DeleteRoom :exec
DELETE FROM rooms
WHERE id = $1;

-- name: GetRoomDetailByID :one
SELECT
    r.*,
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'id',           s.id,
                'room_id',      s.room_id,
                'row_label',    s.row_label,
                'seat_number',  s.seat_number,
                'created_at',   s.created_at,
                'updated_at',   s.updated_at
            )
        ) FILTER (WHERE s.id IS NOT NULL),
        '[]'
    ) AS seats
FROM rooms r
LEFT JOIN seats s ON r.id = s.room_id
WHERE r.id = $1
GROUP BY r.id;

-- name: GetRoomsSummary :many
SELECT 
    r.id,
    r.name
FROM rooms r
ORDER BY r.name;