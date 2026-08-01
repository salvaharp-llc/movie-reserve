package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
	"github.com/salvaharp-llc/movie-reserve/internal/email"
)

type Config struct {
	DBURL        string `env:"DB_URL,required"`
	JWTSecret    string `env:"JWT_SECRET,required"`
	Platform     string `env:"PLATFORM,required"`
	FilepathRoot string `env:"FILEPATH_ROOT,required"`
	AssetsRoot   string `env:"ASSETS_ROOT,required"`
	Port         string `env:"PORT,required"`
	MailHost     string `env:"MAIL_HOST,required"`
	MailPort     string `env:"MAIL_PORT,required"`
	MailUsername string `env:"MAIL_USERNAME,required"`
	MailPassword string `env:"MAIL_PASSWORD,required"`
	RagServerURL string `env:"RAG_SERVER_URL,required"`
	RagAPIKey    string `env:"RAG_API_KEY,required"`
}

type apiConfig struct {
	db           *database.DbStore
	emailSender  *email.EmailSender
	jwtSecret    string
	platform     string
	assetsRoot   string
	port         string
	ragServerURL string
	ragAPIKey    string
}

const (
	staticRateLimit float64 = 10
	staticRateBurst int     = 50
	publicRateLimit float64 = 10
	publicRateBurst int     = 20
	userRateLimit   float64 = 5
	userRateBurst   int     = 10
	adminRateLimit  float64 = 2
	adminRateBurst  int     = 5
)

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
	dbStore := database.NewStore(db)

	emailSender, err := email.NewEmailSender(cfg.MailHost, cfg.MailPort, cfg.MailUsername, cfg.MailPassword)
	if err != nil {
		log.Fatalf("Failed to initialize email sender: %v", err)
	}

	apiCfg := apiConfig{
		db:           dbStore,
		emailSender:  emailSender,
		jwtSecret:    cfg.JWTSecret,
		platform:     cfg.Platform,
		assetsRoot:   cfg.AssetsRoot,
		port:         cfg.Port,
		ragServerURL: cfg.RagServerURL,
		ragAPIKey:    cfg.RagAPIKey,
	}

	if err := apiCfg.ensureAdmin(); err != nil {
		log.Fatalf("Could not ensure admin user: %v", err)
	}

	if err := apiCfg.ensureAssetsDir(); err != nil {
		log.Fatalf("Couldn't create assets directory: %v", err)
	}

	mux := http.NewServeMux()
	staticMux := http.NewServeMux()
	publicMux := http.NewServeMux()
	userMux := http.NewServeMux()
	adminMux := http.NewServeMux()

	staticMux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir(cfg.FilepathRoot))))
	staticMux.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir(cfg.AssetsRoot))))

	publicMux.HandleFunc("GET /api/healthz", handlerReadiness)

	publicMux.HandleFunc("POST /api/users", apiCfg.handlerCreateUsers)
	publicMux.HandleFunc("PUT /api/users/verify", apiCfg.handlerVerifyEmail)
	publicMux.HandleFunc("POST /api/resend-verification", apiCfg.handlerResendVerificationEmail)
	publicMux.HandleFunc("POST /api/password-reset", apiCfg.handlerRequestPasswordReset)
	publicMux.HandleFunc("PUT /api/users/password-reset", apiCfg.handlerPasswordReset)
	publicMux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	publicMux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	publicMux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)

	publicMux.HandleFunc("GET /api/genres/{genreID}", apiCfg.handlerGetGenres)
	publicMux.HandleFunc("GET /api/genres", apiCfg.handlerRetrieveGenres)

	publicMux.HandleFunc("GET /api/movies/{movieID}", apiCfg.handlerGetMovies)
	publicMux.HandleFunc("GET /api/movies", apiCfg.handlerRetrieveMovies)
	publicMux.HandleFunc("GET /api/movies/current", apiCfg.handlerRetrieveCurrentMovies)

	publicMux.HandleFunc("POST /api/rag", apiCfg.handlerRAG) // RAG endpoint

	publicMux.HandleFunc("GET /api/rooms/{roomID}", apiCfg.handlerGetRooms)
	publicMux.HandleFunc("GET /api/rooms", apiCfg.handlerRetrieveRooms)

	publicMux.HandleFunc("GET /api/screenings/{screeningID}", apiCfg.handlerGetScreenings)
	publicMux.HandleFunc("GET /api/screenings", apiCfg.handlerRetrieveScreenings) // Only upcoming screenings for public

	publicMux.HandleFunc("GET /api/seats/{seatID}", apiCfg.handlerGetSeats)

	// Routes requiring user login
	userMux.HandleFunc("PUT /api/users/passwords", apiCfg.handlerUpdatePassword)
	userMux.HandleFunc("PUT /api/users/emails", apiCfg.handlerUpdateEmail)

	userMux.HandleFunc("POST /api/reservations", apiCfg.handlerCreateReservations)
	userMux.HandleFunc("GET /api/reservations", apiCfg.handlerRetrieveReservations) // Only user's reservations for public
	userMux.HandleFunc("GET /api/reservations/{reservationID}", apiCfg.handlerGetReservations)
	userMux.HandleFunc("DELETE /api/reservations/{reservationID}", apiCfg.handlerDeleteReservations)

	// Routes requiring admin role
	adminMux.HandleFunc("POST /api/movies", apiCfg.handlerCreateMovies)
	adminMux.HandleFunc("PUT /api/movies/{movieID}", apiCfg.handlerUpdateMovies)
	adminMux.HandleFunc("DELETE /api/movies/{movieID}", apiCfg.handlerDeleteMovies)
	adminMux.HandleFunc("POST /api/poster_upload/{movieID}", apiCfg.handlerUploadPoster)

	adminMux.HandleFunc("POST /api/genres", apiCfg.handlerCreateGenres)
	adminMux.HandleFunc("PUT /api/genres/{genreID}", apiCfg.handlerUpdateGenres)
	adminMux.HandleFunc("DELETE /api/genres/{genreID}", apiCfg.handlerDeleteGenres)

	adminMux.HandleFunc("POST /api/rooms", apiCfg.handlerCreateRooms)
	adminMux.HandleFunc("PUT /api/rooms/{roomID}", apiCfg.handlerUpdateRooms)
	adminMux.HandleFunc("DELETE /api/rooms/{roomID}", apiCfg.handlerDeleteRooms)

	adminMux.HandleFunc("POST /api/screenings", apiCfg.handlerCreateScreenings)
	adminMux.HandleFunc("GET /api/screenings/all", apiCfg.handlerRetrieveScreeningsAdmin) // Non-limited dates for admin
	adminMux.HandleFunc("PUT /api/screenings/{screeningID}", apiCfg.handlerUpdateScreenings)
	adminMux.HandleFunc("DELETE /api/screenings/{screeningID}", apiCfg.handlerDeleteScreenings)

	adminMux.HandleFunc("POST /api/seats", apiCfg.handlerCreateSeats)
	adminMux.HandleFunc("PUT /api/seats/{seatID}", apiCfg.handlerUpdateSeats)
	adminMux.HandleFunc("DELETE /api/seats/{seatID}", apiCfg.handlerDeleteSeats)

	adminMux.HandleFunc("GET /api/reservations/all", apiCfg.handlerRetrieveReservationsAdmin) // Non-limited reservations for admin

	// Dev/test routes
	mux.HandleFunc("POST /dev/reset", apiCfg.handlerReset)

	mux.Handle("/static/",
		rateLimiterMiddleware(
			http.StripPrefix("/static", staticMux),
			staticRateLimit,
			staticRateBurst,
		),
	)

	mux.Handle("/public/",
		rateLimiterMiddleware(
			http.StripPrefix("/public", publicMux),
			publicRateLimit,
			publicRateBurst,
		),
	)
	mux.Handle("/user/",
		rateLimiterMiddleware(
			http.StripPrefix("/user", apiCfg.requireAuthMiddleware(userMux)),
			userRateLimit,
			userRateBurst,
		),
	)
	mux.Handle("/admin/",
		rateLimiterMiddleware(
			http.StripPrefix("/admin", apiCfg.requireAdminMiddleware(adminMux)),
			adminRateLimit,
			adminRateBurst,
		),
	)

	server := http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", cfg.FilepathRoot, cfg.Port)
	log.Fatal(server.ListenAndServe())
}
