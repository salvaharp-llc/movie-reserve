package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

type Config struct {
	DBURL        string `env:"DB_URL,required"`
	JWTSecret    string `env:"JWT_SECRET,required"`
	Platform     string `env:"PLATFORM,required"`
	FilepathRoot string `env:"FILEPATH_ROOT,required"`
	AssetsRoot   string `env:"ASSETS_ROOT,required"`
	Port         string `env:"PORT,required"`
}

type apiConfig struct {
	db         *database.Queries
	jwtSecret  string
	platform   string
	assetsRoot string
	port       string
}

func main() {
	godotenv.Load()

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("Could not open database: %v", err)
	}
	dbQueries := database.New(db)

	apiCfg := apiConfig{
		db:         dbQueries,
		jwtSecret:  cfg.JWTSecret,
		platform:   cfg.Platform,
		assetsRoot: cfg.AssetsRoot,
		port:       cfg.Port,
	}

	if err := apiCfg.ensureAdmin(); err != nil {
		log.Fatalf("Could not ensure admin user: %v", err)
	}

	if err := apiCfg.ensureAssetsDir(); err != nil {
		log.Fatalf("Couldn't create assets directory: %v", err)
	}

	mux := http.NewServeMux()
	fsHandler := http.StripPrefix("/app", http.FileServer(http.Dir(cfg.FilepathRoot)))
	mux.Handle("/app/", fsHandler)

	assetsHandler := http.StripPrefix("/assets", http.FileServer(http.Dir(cfg.AssetsRoot)))
	mux.Handle("/assets/", assetsHandler)

	// Public routes (no auth required)
	mux.HandleFunc("GET /api/healthz", handlerReadiness)

	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUsers)
	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)

	mux.HandleFunc("GET /api/genres/{genreID}", apiCfg.handlerGetGenres)
	mux.HandleFunc("GET /api/genres", apiCfg.handlerRetrieveGenres)

	mux.HandleFunc("GET /api/movies/{movieID}", apiCfg.handlerGetMovies)
	mux.HandleFunc("GET /api/movies", apiCfg.handlerRetrieveMovies)
	mux.HandleFunc("GET /api/movies/current", apiCfg.handlerRetrieveCurrentMovies)

	mux.HandleFunc("GET /api/rooms/{roomID}", apiCfg.handlerGetRooms)
	mux.HandleFunc("GET /api/rooms", apiCfg.handlerRetrieveRooms)

	mux.HandleFunc("GET /api/screenings/{screeningID}", apiCfg.handlerGetScreenings)
	mux.HandleFunc("GET /api/screenings", apiCfg.handlerRetrieveScreenings) // Only upcoming screenings for public

	mux.HandleFunc("GET /api/seats/{seatID}", apiCfg.handlerGetSeats)

	// Routes requiring valid auth token
	mux.HandleFunc("PUT /api/users", apiCfg.RequireAuth(apiCfg.handlerUpdateUsers))

	mux.HandleFunc("POST /api/reservations", apiCfg.RequireAuth(apiCfg.handlerCreateReservations))
	mux.HandleFunc("GET /api/reservations", apiCfg.RequireAuth(apiCfg.handlerRetrieveReservations)) // Only user's reservations for public
	mux.HandleFunc("GET /api/reservations/{reservationID}", apiCfg.RequireAuth(apiCfg.handlerGetReservations))
	mux.HandleFunc("DELETE /api/reservations/{reservationID}", apiCfg.RequireAuth(apiCfg.handlerDeleteReservations))

	// Routes requiring admin role
	mux.HandleFunc("POST /api/movies", apiCfg.RequireAdmin(apiCfg.handlerCreateMovies))
	mux.HandleFunc("PUT /api/movies/{movieID}", apiCfg.RequireAdmin(apiCfg.handlerUpdateMovies))
	mux.HandleFunc("DELETE /api/movies/{movieID}", apiCfg.RequireAdmin(apiCfg.handlerDeleteMovies))
	mux.HandleFunc("POST /api/poster_upload/{movieID}", apiCfg.RequireAdmin(apiCfg.handlerUploadPoster))

	mux.HandleFunc("POST /api/genres", apiCfg.RequireAdmin(apiCfg.handlerCreateGenres))
	mux.HandleFunc("PUT /api/genres/{genreID}", apiCfg.RequireAdmin(apiCfg.handlerUpdateGenres))
	mux.HandleFunc("DELETE /api/genres/{genreID}", apiCfg.RequireAdmin(apiCfg.handlerDeleteGenres))

	mux.HandleFunc("POST /api/rooms", apiCfg.RequireAdmin(apiCfg.handlerCreateRooms))
	mux.HandleFunc("PUT /api/rooms/{roomID}", apiCfg.RequireAdmin(apiCfg.handlerUpdateRooms))
	mux.HandleFunc("DELETE /api/rooms/{roomID}", apiCfg.RequireAdmin(apiCfg.handlerDeleteRooms))

	mux.HandleFunc("POST /api/screenings", apiCfg.RequireAdmin(apiCfg.handlerCreateScreenings))
	mux.HandleFunc("GET /api/screenings/all", apiCfg.RequireAdmin(apiCfg.handlerRetrieveScreeningsAdmin)) // Non-limited dates for admin
	mux.HandleFunc("PUT /api/screenings/{screeningID}", apiCfg.RequireAdmin(apiCfg.handlerUpdateScreenings))
	mux.HandleFunc("DELETE /api/screenings/{screeningID}", apiCfg.RequireAdmin(apiCfg.handlerDeleteScreenings))

	mux.HandleFunc("POST /api/seats", apiCfg.RequireAdmin(apiCfg.handlerCreateSeats))
	mux.HandleFunc("PUT /api/seats/{seatID}", apiCfg.RequireAdmin(apiCfg.handlerUpdateSeats))
	mux.HandleFunc("DELETE /api/seats/{seatID}", apiCfg.RequireAdmin(apiCfg.handlerDeleteSeats))

	mux.HandleFunc("GET /api/reservations/all", apiCfg.RequireAdmin(apiCfg.handlerRetrieveReservationsAdmin)) // Non-limited reservations for admin
	mux.HandleFunc("PUT /api/reservations/{reservationID}", apiCfg.RequireAdmin(apiCfg.handlerUpdateReservations))

	// Dev/test routes
	mux.HandleFunc("POST /dev/reset", apiCfg.handlerReset)

	server := http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", cfg.FilepathRoot, cfg.Port)
	log.Fatal(server.ListenAndServe())
}
