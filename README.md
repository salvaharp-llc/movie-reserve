# movie-reserve

A backend REST API for reserving seats at movie screenings, built in Go.

---

## Description

movie-reserve handles user authentication, seat reservations, and provides a full management interface for admins to control movies, genres, rooms, screenings, and seats. It exposes a clean REST API with role-based access control — differentiating between public, authenticated, and admin-only endpoints.

---

## Motivation

Booking a seat at the movies involves a surprisingly complex set of moving parts: managing rooms, scheduling screenings, tracking seat availability, and handling concurrent reservations. This project was built to explore those challenges in a real-world Go backend, using PostgreSQL for persistence and JWTs for stateless auth.

---

## Quick Start

### Prerequisites

- [Go](https://golang.org/dl/)
- [PostgreSQL](https://www.postgresql.org/)
- [Goose](https://github.com/pressly/goose) (for database migrations)

### Installation

```bash
git clone https://github.com/salvaharp-llc/movie-reserve@latest
cd movie-reserve
```

### 1. Run database migrations

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

### 3. Start the server

```bash
go run .
# 2026/03/01 19:56:08 Serving files from ./app on port: 8080
```

---

## API Reference

The full API is documented via OpenAPI. You can view the interactive documentation in Swagger UI:

[**View API Docs**](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/salvaharp-llc/movie-reserve/refs/heads/main/openapi.yaml)

This provides a complete, interactive guide to all endpoints, including request and response schemas.

### Dev / test endpoints (only available when `PLATFORM=dev`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/dev/reset` | Reset the database |

### Static files

| Path | Description |
|------|-------------|
| `/app/` | Served from `FILEPATH_ROOT` |
| `/assets/` | Served from `ASSETS_ROOT` |

---

## Contributing

Contributions are welcome! Here are some areas actively being worked on:

- SQL transactions for atomicity
- Improved error responses
- Pagination tweaks
- Query parameter to control resource format (detail vs. summary)
- External login (e.g. Google OAuth)
- Payment integration via webhook
- Frontend

To contribute, fork the repository, create a feature branch, and open a pull request. Please keep changes focused and include a clear description of what you've added or fixed.

---

MIT License