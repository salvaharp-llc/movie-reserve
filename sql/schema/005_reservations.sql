-- +goose Up
CREATE TABLE seats (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    row_label TEXT NOT NULL,
    seat_number INT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE (room_id, row_label, seat_number),
    UNIQUE (id, room_id)                        -- allows composite FK later
);

ALTER TABLE screenings
    ADD CONSTRAINT uq_screenings_room UNIQUE (id, room_id); -- allows composite FK later

CREATE TABLE reservations (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    screening_id UUID NOT NULL REFERENCES screenings(id) ON DELETE CASCADE,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE, -- needed for composite FK
    seat_id UUID NOT NULL REFERENCES seats(id) ON DELETE RESTRICT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE (screening_id, seat_id),
    FOREIGN KEY (screening_id, room_id) REFERENCES screenings(id, room_id),
    FOREIGN KEY (seat_id, room_id) REFERENCES seats(id, room_id)
);

-- +goose Down
DROP TABLE reservations;
ALTER TABLE screenings DROP CONSTRAINT uq_screenings_room;
DROP TABLE seats;