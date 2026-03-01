# movie-reserve

A backend REST API for reserving seats at movie screenings. Built in Go, it handles user authentication, seat reservations, and provides a full management interface for admins to control movies, genres, rooms, screenings, and seats.

---

## Prerequisites

- [Go](https://golang.org/dl/)
- [PostgreSQL](https://www.postgresql.org/)
- [Goose](https://github.com/pressly/goose) (for running database migrations)

---

## Getting Started

### 1. Set up the database

Run the schema migrations using Goose from the `sql/schema` directory:

```bash
goose postgres "$DB_URL" up
```

### 2. Configure environment variables

Create a `.env` file in the project root:

```dotenv
DB_URL=""         # PostgreSQL connection string
PLATFORM=""       # Set to "dev" to enable the /dev/reset endpoint
JWT_SECRET=""     # Secret key for signing JWT tokens
FILEPATH_ROOT="./app"
ASSETS_ROOT="./assets"
PORT=""           # e.g. 8080

# Seed admin credentials (used on first run)
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
```

### 3. Run the server

```bash
go run .
2026/03/01 19:56:08 Serving files from ./app on port: 8080
```

---

## API Endpoints

### Public (no authentication required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/healthz` | Health check |
| POST | `/api/users` | Register a new user |
| POST | `/api/login` | Login and receive tokens |
| POST | `/api/refresh` | Refresh access token |
| POST | `/api/revoke` | Revoke refresh token |
| GET | `/api/genres` | List all genres |
| GET | `/api/genres/{genreID}` | Get a genre by ID |
| GET | `/api/movies` | List all movies |
| GET | `/api/movies/{movieID}` | Get a movie by ID |
| GET | `/api/movies/current` | List currently showing movies |
| GET | `/api/rooms` | List all rooms |
| GET | `/api/rooms/{roomID}` | Get a room by ID |
| GET | `/api/screenings` | List upcoming screenings |
| GET | `/api/screenings/{screeningID}` | Get a screening by ID |
| GET | `/api/seats/{seatID}` | Get a seat by ID |

### Authenticated (valid JWT required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| PUT | `/api/users` | Update current user |
| POST | `/api/reservations` | Create a reservation |
| GET | `/api/reservations` | List current user's reservations |
| GET | `/api/reservations/{reservationID}` | Get a reservation by ID |
| DELETE | `/api/reservations/{reservationID}` | Cancel a reservation |

### Admin only

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/movies` | Create a movie |
| PUT | `/api/movies/{movieID}` | Update a movie |
| DELETE | `/api/movies/{movieID}` | Delete a movie |
| POST | `/api/poster_upload/{movieID}` | Upload a movie poster |
| POST | `/api/genres` | Create a genre |
| PUT | `/api/genres/{genreID}` | Update a genre |
| DELETE | `/api/genres/{genreID}` | Delete a genre |
| POST | `/api/rooms` | Create a room |
| PUT | `/api/rooms/{roomID}` | Update a room |
| DELETE | `/api/rooms/{roomID}` | Delete a room |
| POST | `/api/screenings` | Create a screening |
| GET | `/api/screenings/all` | List all screenings (no date limit) |
| PUT | `/api/screenings/{screeningID}` | Update a screening |
| DELETE | `/api/screenings/{screeningID}` | Delete a screening |
| POST | `/api/seats` | Create a seat |
| PUT | `/api/seats/{seatID}` | Update a seat |
| DELETE | `/api/seats/{seatID}` | Delete a seat |
| GET | `/api/reservations/all` | List all reservations |
| PUT | `/api/reservations/{reservationID}` | Update a reservation |

### Dev / Test (only available when `PLATFORM=dev`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/dev/reset` | Reset the database |

---

## Static Files

| Path | Description |
|------|-------------|
| `/app/` | Served from `FILEPATH_ROOT` |
| `/assets/` | Served from `ASSETS_ROOT` |

---

## Futuren Additions

- implement sql transactions for atomicity
- improve error responses
- tweak pagination
- add query parameter do decide resource format (detail/summary)
- external login method eg. Google
- payment method (webhook whith external service)
- add forntend

## License

MIT
