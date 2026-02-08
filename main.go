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
	db         *database.Queries
	jwtSecret  string
	platform   string
	assetsRoot string
	port       string
}

func main() {
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
	filepathRoot := os.Getenv("FILEPATH_ROOT")
	if filepathRoot == "" {
		log.Fatal("FILEPATH_ROOT environment variable is not set")
	}
	assetsRoot := os.Getenv("ASSETS_ROOT")
	if assetsRoot == "" {
		log.Fatal("ASSETS_ROOT environment variable is not set")
	}
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Could not open database: %v", err)
	}
	dbQueries := database.New(db)

	apiCfg := apiConfig{
		db:         dbQueries,
		jwtSecret:  JWTSecret,
		platform:   platform,
		assetsRoot: assetsRoot,
		port:       port,
	}

	if err := apiCfg.ensureAdmin(); err != nil {
		log.Fatalf("Could not ensure admin user: %v", err)
	}

	if err := apiCfg.ensureAssetsDir(); err != nil {
		log.Fatalf("Couldn't create assets directory: %v", err)
	}

	mux := http.NewServeMux()
	fsHandler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/", fsHandler)

	assetsHandler := http.StripPrefix("/assets", http.FileServer(http.Dir(assetsRoot)))
	mux.Handle("/assets/", assetsHandler)

	// Public routes (no auth required)
	mux.HandleFunc("GET /api/healthz", handlerReadiness)

	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUsers)
	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)

	mux.HandleFunc("GET /api/genres/{genreID}", apiCfg.handlerGetGenres)
	mux.HandleFunc("GET /api/genres", apiCfg.handlerRetrieveGenres)

	mux.HandleFunc("GET /api/movies/{movieID}", apiCfg.handlerGetMovies)

	// Routes requiring valid auth token
	mux.HandleFunc("POST /api/refresh", apiCfg.RequireAuth(apiCfg.handlerRefresh))
	mux.HandleFunc("POST /api/revoke", apiCfg.RequireAuth(apiCfg.handlerRevoke))

	mux.HandleFunc("PUT /api/users", apiCfg.RequireAuth(apiCfg.handlerUpdateUsers))

	// Routes requiring admin role
	mux.HandleFunc("POST /api/movies", apiCfg.RequireAdmin(apiCfg.handlerCreateMovies))
	mux.HandleFunc("PUT /api/movies/{movieID}", apiCfg.RequireAdmin(apiCfg.handlerUpdateMovies))
	mux.HandleFunc("DELETE /api/movies/{movieID}", apiCfg.RequireAdmin(apiCfg.handlerDeleteMovies))
	mux.HandleFunc("POST /api/poster_upload/{movieID}", apiCfg.RequireAdmin(apiCfg.handlerUploadPoster))

	mux.HandleFunc("POST /api/genres", apiCfg.RequireAdmin(apiCfg.handlerCreateGenres))
	mux.HandleFunc("PUT /api/genres/{genreID}", apiCfg.RequireAdmin(apiCfg.handlerUpdateGenres))
	mux.HandleFunc("DELETE /api/genres/{genreID}", apiCfg.RequireAdmin(apiCfg.handlerDeleteGenres))

	// Dev/test routes
	mux.HandleFunc("POST /dev/reset", apiCfg.handlerReset)

	server := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
