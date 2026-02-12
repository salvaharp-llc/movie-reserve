-- +goose Up
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE rooms (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE screenings (
    id UUID PRIMARY KEY,
    movie_id UUID NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE RESTRICT,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT valid_time_range CHECK (start_time < end_time),
    CONSTRAINT no_overlapping_screenings
        EXCLUDE USING gist (
            room_id WITH =,
            tsrange(start_time, end_time, '[)') WITH &&
        )
);

-- +goose Down
DROP TABLE screenings;
DROP TABLE rooms;