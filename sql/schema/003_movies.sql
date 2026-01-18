-- +goose Up
CREATE TABLE movies (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    runtime_minutes INT,
    release_date DATE,
    poster_url TEXT
);

CREATE TABLE genres (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE movie_genre (
    movie_id UUID NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    genre_id UUID NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (movie_id, genre_id)
);

-- +goose Down
DROP TABLE movie_genre;
DROP TABLE genres;
DROP TABLE movies;