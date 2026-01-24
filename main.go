package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

type apiConfig struct {
	db        *database.Queries
	jwtSecret string
	platform  string
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}
	JWTSecret := os.Getenv("JWT_SECRET")
	if JWTSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM must be set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Could not open database: %v", err)
	}
	dbQueries := database.New(db)

	apiCfg := apiConfig{
		db:        dbQueries,
		jwtSecret: JWTSecret,
		platform:  platform,
	}

	if err := apiCfg.ensureAdmin(); err != nil {
		log.Fatalf("Could not ensure admin user: %v", err)
	}

	mux := http.NewServeMux()
	fsHandler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/", fsHandler)

	// Public routes (no auth required)
	mux.HandleFunc("GET /api/healthz", handlerReadiness)

	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUsers)
	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)

	// Routes requiring valid auth token
	mux.HandleFunc("POST /api/refresh", apiCfg.RequireAuth(apiCfg.handlerRefresh))
	mux.HandleFunc("POST /api/revoke", apiCfg.RequireAuth(apiCfg.handlerRevoke))

	mux.HandleFunc("PUT /api/users", apiCfg.RequireAuth(apiCfg.handlerUpdateUsers))

	// Routes requiring admin role
	mux.HandleFunc("POST /api/movies", apiCfg.RequireAdmin(apiCfg.handlerCreateMovies))
	mux.HandleFunc("PUT /api/movies/{movieID}", apiCfg.RequireAdmin(apiCfg.handlerUpdateMovies))
	// mux.HandleFunc("DELETE /api/movies/{movieID}", apiCfg.RequireAdmin(apiCfg.handlerDeleteMovies))

	// Dev/test routes
	mux.HandleFunc("POST /dev/reset", apiCfg.handlerReset)

	server := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
