package main

import (
	"context"
	"log"
	"os"

	"game-scores/ent"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting database migration...")

	// Connect to the database
	connStr, ok := os.LookupEnv("DB_SOURCE")
	if !ok {
		log.Fatal("DB_SOURCE environment variable not set")
	}

	log.Printf("Attempting to connect with DSN: %s", connStr)

	client, err := ent.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer client.Close()

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	log.Println("Database migration completed successfully.")
}
