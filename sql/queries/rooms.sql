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

-- name: GetRoomByID :one
SELECT * FROM rooms
WHERE id = $1;

-- name: GetRooms :many
SELECT * FROM rooms
ORDER BY name;