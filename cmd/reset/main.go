package main

import (
	"context"
	"log"
	"os"

	"game-scores/ent"
	"game-scores/ent/user" // Import the generated user package for queries

	_ "github.com/lib/pq"
)

func main() {
	log.Println("--- Starting Database Reset ---")

	// Get database connection string from environment variable
	connStr := os.Getenv("DB_SOURCE")
	if connStr == "" {
		log.Fatal("DB_SOURCE environment variable not set")
	}

	// Connect to the database
	client, err := ent.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Step 1: Delete all scores
	// This is done first because scores have foreign keys to users and games.
	deletedScores, err := client.Score.Delete().Exec(ctx)
	if err != nil {
		log.Fatalf("failed to delete scores: %v", err)
	}
	log.Printf("✅ Deleted %d scores.", deletedScores)

	// Step 2: Delete all games
	deletedGames, err := client.Game.Delete().Exec(ctx)
	if err != nil {
		log.Fatalf("failed to delete games: %v", err)
	}
	log.Printf("✅ Deleted %d games.", deletedGames)

	// Step 3: Delete all users EXCEPT the admin user
	deletedUsers, err := client.User.
		Delete().
		Where(user.UsernameNEQ("admin")). // Use the NEQ (Not Equal) predicate
		Exec(ctx)
	if err != nil {
		log.Fatalf("failed to delete non-admin users: %v", err)
	}
	log.Printf("✅ Deleted %d non-admin users.", deletedUsers)

	log.Println("--- ✅ Database Reset Complete. Admin account remains. ---")
}
