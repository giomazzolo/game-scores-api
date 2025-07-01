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
	"context"
	"log"
	"net/http"
	"os"

	"game-scores/ent"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {

	/* Database Init ************************************************************/
	// Get database connection string from environment variable
	connStr, ok := os.LookupEnv("DB_SOURCE")
	if !ok {
		log.Fatal("DB_SOURCE environment variable not set")
	}

	// Connect to the database
	db, err := ent.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Successfully connected to the database!")

	/* DB test ************************************************************/

	// Use a background context for operations
	ctx := context.Background()

	// --- Run Database Migrations ---
	if err := db.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}
	log.Println("Database schema is up-to-date")

	// --- Create a New Game ---
	// This block directly creates a new "Game" entity in the database.
	newGame, err := db.Game.
		Create().
		SetName("Space Invaders").
		SetDescription("Another classic arcade game").
		Save(ctx)

	if err != nil {
		if ent.IsConstraintError(err) {
			log.Printf("Game with this name already exists: %v", err)
		} else {
			log.Fatalf("failed creating game: %v", err)
		}
	}

	// Print the newly created game to the console
	log.Printf("Game created successfully: %+v\n", newGame)

	// --- Read All Games from the Database ---
	games, err := db.Game.
		Query(). // Start a query builder for the Game type
		All(ctx) // Execute the query and get all results

	if err != nil {
		log.Fatalf("failed querying games: %v", err)
	}

	// Print the results to the console
	log.Println("Games found in the database:")
	for _, g := range games {
		log.Printf("- ID: %v, Name: %s, Description: %s", g.ID, g.Name, g.Description)
	}

	/* Router Init ************************************************************/
	// API endpoint to check the connection

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("API is running with go-chi and database connection is successful!"))
	})

	log.Println("Go API server starting on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
