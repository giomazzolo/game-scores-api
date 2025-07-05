# 🎮 Game Scores API

A Go-based API for tracking game scores, built with PostgreSQL and containerized with Docker. This project is designed to be run with a hybrid approach, where the database runs in Docker for consistency, and the Go code can be run locally for rapid development.

![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)
![Docker](https://img.shields.io/badge/Docker-Compose-blue)
![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL-blue.svg)

---

## 🚀 Getting Started

This guide will walk you through setting up and running the project.

### Prerequisites

Before you begin, ensure you have the following installed on your system:
* **Go** (version 1.24.4 or later)
* **Docker** and **Docker Compose**

---

## 💻 Docker Containers Set-up

This is the recommended workflow for active development. It provides the stability of a containerized database while allowing you to run and debug your Go code locally for maximum speed.

### Step 1: Start the Containers

This command builds the Server, the PostgreSQL database, the Prometheus and Grafana services and runs them in the background. The server will start listening on `http://localhost:8080`. All system and environmental variables are configured in `docker-composer.yml`.

**Run:**
```powershell
docker-compose build --build-arg CACHE_BUSTER=$(Get-Date -UFormat %s); docker-compose up
```

### Step 2: Run Database Migrations and Seed the Database

Now you need to execute the migration and seeder scripts. Both scripts run from your local machine and connect to the Dockerized database. The migration script creates all the necessary tables, the seeder script adds an admin user to the database which can be used for testing.

First, we need to set up the BD_SOURCE env variable to let our local machine know how to connect to the dockerized database. The configs used are the same as in the docker-compose.

**1 - On a new terminal, run:**

> **On Windows (PowerShell):**
> ```powershell
> $env:DB_SOURCE="postgresql://user:mysecretpassword@localhost:5433/mydatabase?sslmode=disable"; go run ./cmd/migrate; go run ./cmd/seeder
> ```

> **On macOS / Linux:**
> ```bash
> export DB_SOURCE="postgresql://user:mysecretpassword@localhost:5433/mydatabase?sslmode=disable"; go run ./cmd/migrate; go run ./cmd/seeder
> ```

> **Note:** We are using port `5433` to avoid conflicts with any local PostgreSQL installations.


After this, everything will be setup correctly and the API will be available at `http://localhost:8080`.

## 🔁 Stop and Restart

**To stop the containers run:**

```powershell
docker-compose stop
```

Alternatively, press `ctrl + c` on the terminal where it is running, or stop the containers with the Docker application.

**To re-run the containers, use the same command as in Step 1**:
```powershell
docker-compose build --build-arg CACHE_BUSTER=$(Get-Date -UFormat %s); docker-compose up
```

## 🧪 Running the Integration Tests

The project includes a full integration test suite that automatically sets up and tears down its own environment.

**To run all tests, all containers must be running.**
Execute the following command from the project root:
```powershell
go clean -testcache; go test -v -timeout 5m ./cmd/api
```

The expected test results output should look like this:

```
--- PASS: TestAPIFlow (2.95s)
    --- PASS: TestAPIFlow/Admin_Login (0.08s)
    --- PASS: TestAPIFlow/Register_API (0.93s)
        --- PASS: TestAPIFlow/Register_API/Re-register_random_existing_user (0.06s)
        --- PASS: TestAPIFlow/Register_API/Register_with_short_password (0.00s)
        --- PASS: TestAPIFlow/Register_API/Register_with_short_username (0.00s)
    --- PASS: TestAPIFlow/Login_API (0.84s)
        --- PASS: TestAPIFlow/Login_API/Login_with_wrong_password (0.06s)
        --- PASS: TestAPIFlow/Login_API/Login_with_non-existent_user (0.00s)
    --- PASS: TestAPIFlow/Add_Game_API (0.05s)
        --- PASS: TestAPIFlow/Add_Game_API/Non-admin_cannot_add_game (0.00s)
    --- PASS: TestAPIFlow/List_Games_API (0.00s)
    --- PASS: TestAPIFlow/Join_Game_API (0.45s)
    --- PASS: TestAPIFlow/Update_Score_API (0.54s)
    --- PASS: TestAPIFlow/List_Scores_API (0.03s)
    --- PASS: TestAPIFlow/List_Statistics_API (0.02s)
PASS
ok      game-scores/cmd/api     4.881s
```

API coverage: `login`, `register`, `add_game`, `list_games`, `join_games`, `update_score`, `list_scores`, `list_statistics`.

The API's `ping` and `metrics` are not included in testing.

---

## 🛑 Removing the Environment

To stop all running containers and **completely remove** the containers and volumes use the following command:

```powershell
docker-compose down -v
```

To start again, go back to the set-up section.

---
## 🛠️ Technology Stack

### Database: PostgreSQL with EntGo

* **PostgreSQL:** The api implemetation uses the PostgresSQL database, an open-source object-relational database system known for its reliability and performance.
* **EntGo:** A modern entity framework for Go. We use Ent's "schema-as-code" approach to define our database tables and relationships directly in Go. This provides type-safety for all database queries, catching errors at compile time instead of at runtime. The schema is located in the `/ent/schema` directory.

### Database Schemas

The schemas defined in the database are:

* **Games:** Holds information about the game name and description
* **Users:** Holds username, email, password and role
* **Scores:** Relates a User to a Game and holds all the scores of all Users for any game they have joined.

```mermaid
erDiagram
    GAMES ||--o{ SCORES : "has"
    USERS ||--o{ SCORES : "has"

    GAMES {
        int id PK
        string name
        string description
    }

    SCORES {
        int id PK
        datetime created_at
        int game_scores
        int user_scores
    }

    USERS {
        int id PK
        string username
        string email
        string password_hash
        string role
    }
```

### HTTP Router: go-chi

Package [chi](https://github.com/go-chi/chi) is a small, idiomatic and composable router for building HTTP services. We utilize this router for it's simplicity and cleanliness.


---

## 📊 Telemetry & Monitoring with Grafana

The project includes a full observability stack to monitor the API's health and performance.

### How to Access

1.  **Start all services:** All services and containers must be running (See Setup)
2.  **Navigate to Grafana:** Open your browser to `http://localhost:3000`.
3.  **Login:** Use the default credentials:
    * **Username:** `admin`
    * **Password:** `admin`
4.  **Add Data Source:**
    * On the left menu, go to **Connections** > **Data sources**.
    * Click **"Add new data source"** and select **Prometheus**.
    * For the **"Prometheus server URL"**, enter `http://prometheus:9090`.
    * Click **Save & test**, you should see a pop-up `"Datasource updated"`
5.  **Import Dashboard**
    * On the left menu, go to **Dashboards**
    * Click **New** and select **Import**.
    * On the project root, to to `configs/grafana/` and select the file `dashboard.json`.
    * The dashboard should load with pre-configured visualizations and metrics.

You can now create dashboards to visualize the metrics exposed by the API at the `/metrics` endpoint.

---

# API Documentation

This section outlines all the available endpoints for the Game Scores API, including usage, request bodies, and authorization requirements.

---

## 🔑 Authentication Endpoints

These endpoints are used for user registration and login. They are public and do not require a JWT.

### `POST /register` - Register a New User

Creates a new user account with the default "player" role. The user's data is stored in the database, with the password being hashed for security. A unique UUID is generated for each registered user.

* **Authorization:** Public

* **Request Body:**
    ```json
    {
        "username": "new_player",         // must be between 3 and 64 characters
        "email": "player@example.com",    // optional, must be a valid email format
        "password": "a_strong_password"   // 8 characters minimum
    }
    ```

**Success Response:**

* **Code:** `201 Created`
* **Body:**
    ```json
    {
        "message": "User registered successfully"
    }
    ```

---
### `POST /login` - User Login

Authenticates a user with their username and password. If successful, it returns a JSON Web Token (JWT) used for accessing protected routes.

* **Authorization:** Public

* **Request Body:**
    ```json
    {
        "username": "new_player",
        "password": "a_strong_password"
    }
    ```

**Success Response:**

* **Code:** `200 OK`
* **Body:**
    ```json
    {
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoi..."
    }
    ```

---
## 🎲 Game Endpoints

Endpoints for managing and viewing games.

### `GET /games` - List All Games

Retrieves a list of all available games in the database.

* **Authorization:** Public

* **Request Body:** None

**Success Response:**

* **Code:** `200 OK`
* **Body:**
    ```json
    [
        {
            "id": 1,
            "name": "Starship Commander",
            "description": "A test game."
        },
        {
            "id": 2,
            "name": "Dungeon Crawler X",
            "description": "A test game."
        }
    ]
    ```

---
### `POST /games` - Add a New Game

Creates a new game entry in the database. This action is restricted to admin users.

* **Authorization:** **Admin only** (Requires a valid JWT with the "admin" role)

* **Request Body:**
    ```json
    {
        "game_name": "Pixel Racer",         // must not be empty
        "description": "A retro racing game." // optional
    }
    ```

**Success Response:**

* **Code:** `201 Created`
* **Body:**
    ```json
    {
        "message": "Game added successfully"
    }
    ```

---
## 🏆 Score & Statistics Endpoints

Endpoints for managing player scores and viewing game statistics.

### `GET /games/{gameID}/scores` - List All Scores for a Game

Retrieves a sorted leaderboard (highest score first) of all scores for a specific game.

* **Authorization:** Public

* **Request Body:** None

**Success Response:**

* **Code:** `200 OK`
* **Body:**
    ```json
    [
        {
            "username": "ShadowStriker",
            "score": "9500"
        },
        {
            "username": "CyberNinja",
            "score": "8200"
        }
    ]
    ```

---
### `GET /games/{gameID}/statistics` - Get Game Statistics

Retrieves the mean, median, and mode of all scores for a specific game.

* **Authorization:** Public

* **Request Body:** None

**Success Response:**

* **Code:** `200 OK`
* **Body:**
    ```json
    {
        "mean": "7540",
        "median": "7800",
        "mode": ["8200"]
    }
    ```

---
### `POST /games/{gameID}/join` - Join a Game

Creates an initial score of 0 for the logged-in player, effectively "joining" them to the specified game.

* **Authorization:** **Player** (Requires a valid JWT)
* **Request Body:** None

**Success Response:**

* **Code:** `201 Created`
* **Body:**
    ```json
    {
        "message": "Successfully joined game",
        "score": 0
    }
    ```

---
### `PUT /games/{gameID}/scores` - Update a Score

Updates the score for the logged-in player in a specific game. The new score must be higher than the current score for it to be updated.

* **Authorization:** **Player** (Requires a valid JWT)

* **Request Body:**
    ```json
    {
        "score": "12000" // The new score value as a string
    }
    ```

**Success Response:**

* **Code:** `200 OK`
* **Body:**
    ```json
    {
        "score": "12000"
    }
    ```

---
## ⚙️ System Endpoints

### `GET /ping` - Health Check

A simple endpoint to check if the server is running and reachable.

* **Authorization:** Public
* **Code:** `200 OK`
* **Response:** A plain text response `Server is running!`

---
### `GET /metrics` - Prometheus Metrics

Exposes application metrics in the Prometheus format for monitoring and telemetry.

* **Authorization:** Public
* **Response:** A text-based exposition of all collected metrics.