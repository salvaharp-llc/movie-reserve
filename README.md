# movie-reserve

A backend REST API for reserving seats at movie screenings, built in Go.

---

## Description

movie-reserve handles user authentication, seat reservations, and provides a full management interface for admins to control movies, genres, rooms, screenings, and seats. It exposes a clean REST API with role-based access control — differentiating between public, authenticated, and admin-only endpoints. The backend also integrates a retrieval-augmented generation (RAG) service to enable intelligent, context-aware queries about movies, helping users discover and learn about available films.

---

## Motivation

This repository is primarily an exploration of modern backend technologies and how they can be implemented together as an integrated system. The project uses the familiar domain of rooms, screenings, seats, and reservations as a model to exercise and demonstrate system-level concerns: API design, transactional state, concurrency control, persistence, authentication, and cross-language service composition.

The goal is to provide a compact, practical playground for experimenting with Go-based HTTP services, PostgreSQL-backed data modeling and migrations, JWT-based stateless authentication, and a Python-based retrieval-augmented generation (RAG) microservice for advanced query capabilities. By wiring these pieces into a coherent system, the project surfaces trade-offs, integration patterns, and reusable components that are applicable to many domains beyond movie booking.

---

## Quick Start

### Prerequisites

- [Go](https://golang.org/dl/)
- [PostgreSQL](https://www.postgresql.org/)
- [Goose](https://github.com/pressly/goose) (for database migrations)
- [Python 3.12+](https://www.python.org/downloads/)
- [uv](https://docs.astral.sh/uv/) (for the `rag_server` Python service)

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

# Email settings
MAIL_HOST=""
MAIL_PORT=""
MAIL_USERNAME=""
MAIL_PASSWORD=""

# RAG service integration
RAG_SERVER_URL="http://localhost:8000/rag"
RAG_API_KEY=""

# Seed admin credentials (used on first run)
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
```

The RAG service is a separate FastAPI app located in `./rag_server`. Set the same value for `API_KEY` in `rag_server/.env` and `RAG_API_KEY` in the project root `.env` so the Go server can authenticate to it:

```dotenv
OPENROUTER_API_KEY=""
API_KEY=""
DB_URL=""
TRANSFORMER_MODEL="clip-ViT-B-32"
```

### 3. Start the RAG service

```bash
cd rag_server
uv sync
uv run python run.py
# Uvicorn running on http://0.0.0.0:8000
```

### 4. Start the Go API server

```bash
cd ..
go run .
# 2026/03/01 19:56:08 Serving files from ./app on port: 8080
```

---

## Usage

The full API is documented via OpenAPI. You can view the interactive documentation in Swagger UI:

[**View API Docs**](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/salvaharp-llc/movie-reserve/refs/heads/main/openapi.yaml)

This provides a complete, interactive guide to all endpoints, including request and response schemas.

### Endpoint Overview

The API is organized into three main categories:

#### Public Endpoints (no auth required)

- **Health & Auth**: Health check, login, token refresh, token revocation
- **Users**: Register, verify email, resend verification, request password reset, reset password
- **Discovery**: List/get movies, genres, rooms, screenings, seats

#### User Endpoints (authentication required)

- **Account**: Update password, change email
- **Reservations**: Create, list, view, and cancel reservations
- **RAG**: Query movie information using retrieval-augmented generation

#### Admin Endpoints (admin role required)

- **Movies**: Create, update, delete movies; upload posters
- **Genres**: Create, update, delete genres
- **Rooms**: Create, update, delete rooms
- **Screenings**: Create, update, delete screenings; list all screenings (including past)
- **Seats**: Create, update, delete seats
- **Reservations**: View all reservations with advanced filtering

### Dev / test endpoints (only available when `PLATFORM=dev`)

| Method | Endpoint     | Description        |
|--------|----------    |-------------       |
| POST   | `/dev/reset` | Reset the database |

### Static files

| Path       | Description                 |
|------      |-------------                |
| `/app/`    | Served from `FILEPATH_ROOT` |
| `/assets/` | Served from `ASSETS_ROOT`   |
