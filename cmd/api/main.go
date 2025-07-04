/*
 * main.go
 * Entry point for the Game Scores API.
 *
 * Project: game-scores-api
 * Author: [Giovanni Mazzolo]
 * email: [giovannimazzolo@outlook.com]
 * Created: [30 June 2025]
 */

package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"game-scores/ent"

	handler "game-scores/internal/handlers"
	api_middleware "game-scores/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	/* Logging Init ************************************************************/

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	/* Environmental variable loading ********************************************/

	// Load jwt secret key from environment variable
	jwtSecret, ok := (os.LookupEnv("JWT_SECRET_KEY"))
	if !ok {
		log.Fatal("JWT_SECRET_KEY environment variable not set")
	}
	// Load database connection string from environment variable
	connStr, ok := (os.LookupEnv("DB_SOURCE"))
	if !ok {
		log.Fatal("DB_SOURCE environment variable not set")
	}

	/* Database Init ************************************************************/

	// Connect to the database
	db, err := ent.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Successfully connected to the database!")

	/* Server and Routes Init ************************************************************/
	// API endpoint to check the connection

	r := chi.NewRouter()

	// Add Telemetry middleware (must go before the logger, to wrap everything)
	r.Use(api_middleware.Telemetry)

	// Middleware logger from chi to see simple console logs
	r.Use(middleware.Logger)

	// Initialize handlers with dependencies
	userHandler := &handler.UserHandler{Database: db, JWTSecret: []byte(jwtSecret)}
	gameHandler := &handler.GameHandler{Database: db}
	gameScoresHandler := &handler.GameScoresHandler{Database: db}

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Server is running!"))
	})

	r.Post("/register", userHandler.Register)
	r.Post("/login", userHandler.Login)
	r.Get("/games", gameHandler.ListGames)
	r.Get("/games/{gameID}/scores", gameScoresHandler.ListGameScores)
	r.Get("/games/{gameID}/statistics", gameScoresHandler.ListGameScoreStatistics)

	// Add Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	r.Group(func(r chi.Router) {

		// Protected routes that require authentication
		r.Use(api_middleware.AuthMiddleware([]byte(jwtSecret)))

		r.Post("/games", gameHandler.AddGame)
		r.Put("/games/{gameID}/scores", gameScoresHandler.UpdateGameScore)
		r.Post("/games/{gameID}/join", gameScoresHandler.JoinGame)
	})

	// Start the server and listen on port 8080

	slog.Info("Starting server", "port", 8080)
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
