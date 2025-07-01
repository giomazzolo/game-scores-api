package main

import (
	"context"
	"log"
	"os"

	"game-scores/ent"
	"game-scores/ent/user"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	log.Println("Starting database seeder...")

	// --- Connect to the Database ---
	connStr := os.Getenv("DB_SOURCE")
	client, err := ent.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	adminUsername := "admin"

	// --- Check if admin user already exists ---
	exists, err := client.User.Query().Where(user.UsernameEQ(adminUsername)).Exist(ctx)
	if err != nil {
		log.Fatalf("failed checking for admin user: %v", err)
	}

	if exists {
		log.Println("Admin user already exists. Seeder finished.")
		return
	}

	// --- Create Admin User ---
	// WARNING: Not production ready: hardcoded password for seeding purposes
	adminPassword := "admin123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	_, err = client.User.Create().
		SetUsername(adminUsername).
		SetEmail("admin@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetRole(user.RoleAdmin). // Set the role to admin
		Save(ctx)

	if err != nil {
		log.Fatalf("failed creating admin user: %v", err)
	}

	log.Println("Admin user created successfully.")
}
