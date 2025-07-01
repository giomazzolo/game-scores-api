# game-scores-api

# 🎮 Game Scores API

A Go-based API for tracking game scores, built with PostgreSQL and containerized with Docker.

---
## 🚀 Getting Started

This guide will walk you through setting up the project for local development or running it entirely within Docker.

### Prerequisites

Before you begin, ensure you have the following installed:

* Go (version 1.24.4 or later)
* Docker and Docker Compose

---
## 💻 Local Development Workflow

### 1. Start the Database

This command starts the PostgreSQL database container in the background.

> ```
> docker-compose up -d db
> ```

### 2. Set the Database Connection String

You must set an environment variable in your terminal so your Go application knows how to connect to the database.

> On Windows (PowerShell):
> ```
> $env:DB_SOURCE="postgresql://user:mysecretpassword@localhost:5433/mydatabase?sslmode=disable"
> ```

> On macOS / Linux:
> ```
> export DB_SOURCE="postgresql://user:mysecretpassword@localhost:5433/mydatabase?sslmode=disable"
> ```

> Note: We are using port 5433 to avoid conflicts with any local PostgreSQL installations.

### 3. Run Database Migrations

This command runs the Ent migration logic to create or update your database tables according to your schema.

> ```
> go run ./cmd/migrate
> ```

### 4. Seed the Database (Optional)

If you need to create initial data (like an admin account), run the seeder script.

> ```
> go run ./cmd/seeder
> ```

### 5. Run the API Server

Finally, run the main application. It will connect to the database and start listening for requests.

> ```
> go run ./cmd/api
> ```

The API will be available at http://localhost:8080.

---
## 🐳 Docker Workflow

This builds and runs the entire application stack (API and Database) inside Docker containers.

### Run the Entire Application

This command builds the Go application image, starts the API and database containers, and connects them. The `--build-arg CACHE_BUSTER` flag is used to ensure Docker doesn't use a stale cache and always includes your latest code changes. This is to avoid having to double build on some systems.


> On Windows (PowerShell):
> ```
> docker-compose build --build-arg CACHE_BUSTER=$(Get-Date -UFormat %s); docker-compose up
> ```

> On macOS / Linux:
> ```
> docker-compose build --build-arg CACHE_BUSTER=$(date +%s) && docker-compose up
> ```

Your API will be available at http://localhost:8080.

To stop all services, press Ctrl + C in the terminal where the containers are running.
