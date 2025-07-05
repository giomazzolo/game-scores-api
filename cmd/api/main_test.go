package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	handler "game-scores/internal/handlers"

	"game-scores/internal/auth"

	"github.com/anandvarma/namegen"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// --- Test Suite Setup (TestMain) ---

// TestMain is the entry point for the entire test suite.
// Docker must be running with the all the containers
// It runs database migrations if needed, cleans up any previous test data, and runs the tests
// After it runs the tests, it leaves the database with the test data intact for manual inspection.
// After all tests are done, it will exit with the appropriate exit code
func TestMain(m *testing.M) {
	log.Println("--- 🚀 Setting up Test Environment ---")

	// Define the database source environment variable for the next commands
	dbSourceEnv := "DB_SOURCE=postgresql://user:mysecretpassword@localhost:5433/mydatabase?sslmode=disable"

	// Run the database migration to create tables
	log.Println("Running database migrations...")
	migrateCmd := exec.Command("go", "run", "../migrate")
	migrateCmd.Env = append(os.Environ(), dbSourceEnv)
	if output, err := migrateCmd.CombinedOutput(); err != nil {
		log.Fatalf("❌ Could not run database migrations: %v\nOutput: %s", err, string(output))
	}

	// Clean up any previous test data
	log.Println("Cleaning up previous test data...")
	resetCmd := exec.Command("go", "run", "../reset")
	resetCmd.Env = append(os.Environ(), dbSourceEnv)
	if output, err := resetCmd.CombinedOutput(); err != nil {
		log.Fatalf("❌ Could not reset the database: %v\nOutput: %s", err, string(output))
	}

	// Run the seeder to ensure the admin account exists
	log.Println("Seeding the database with the admin account...")
	seederCmd := exec.Command("go", "run", "../seeder")
	seederCmd.Env = append(os.Environ(), dbSourceEnv)
	if output, err := seederCmd.CombinedOutput(); err != nil {
		log.Fatalf("❌ Could not run database seeder: %v\nOutput: %s", err, string(output))
	}

	// Run the actual tests
	log.Println("--- Running Tests ---")
	exitCode := m.Run()

	os.Exit(exitCode)
}

// --- Configuration ---
const (
	apiURL           = "http://localhost:8080"
	adminUsername    = "admin"
	adminPassword    = "admin123!"
	adminEmail       = "admin@example.com"
	numPlayersToTest = 100
	maxScore         = 100 // Maximum score a player can have
	concurrency      = 10  // How many requests to run in parallel
)

// --- Helper Structs ---
type TestState struct {
	AdminToken       string
	Games            []handler.GameResponse
	Players          []*Player
	CreatedGameNames []string
}

type Player struct {
	Username string
	Email    string
	Password string
	Token    string
	UserID   uuid.UUID
	GameIDs  []int
}

var gamertags = make([]string, numPlayersToTest)
var totalGameJoins int32 = 0 // Total number of game registrations across all players

// --- Main Test Function ---

// TestAPIFlow runs all integration tests in a specific order.
func TestAPIFlow(t *testing.T) {
	state := &TestState{}
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate unique gamertags for each player
	schemas := [][]namegen.DictType{
		{namegen.Adjectives, namegen.Colors, namegen.Animals},
		{namegen.Adjectives, namegen.Animals},
		{namegen.Colors, namegen.Animals},
	}
	ngen1 := namegen.NewWithPostfixId(schemas[0], namegen.Numeric, 4)
	ngen2 := namegen.NewWithPostfixId(schemas[1], namegen.Numeric, 4)
	ngen3 := namegen.NewWithPostfixId(schemas[2], namegen.Numeric, 4)
	for i := range numPlayersToTest {
		switch rand.Intn(3) {
		case 0:
			gamertags[i] = ngen1.Get()
		case 1:
			gamertags[i] = ngen2.Get()
		case 2:
			gamertags[i] = ngen3.Get()
		}
	}

	// Each API test is now a sub-test, which provides clear output.
	t.Run("Admin Login", func(t *testing.T) { testAdminLogin(t, state) })
	t.Run("Register API", func(t *testing.T) { testRegisterAPI(t, state) })
	t.Run("Login API", func(t *testing.T) { testLoginAPI(t, state) })
	t.Run("Add Game API", func(t *testing.T) { testAddGameAPI(t, state) })
	t.Run("List Games API", func(t *testing.T) { testListGamesAPI(t, state) })
	t.Run("Join Game API", func(t *testing.T) { testJoinGameAPI(t, state) })
	t.Run("Update Score API", func(t *testing.T) { testUpdateScoreAPI(t, state) })
	t.Run("List Scores API", func(t *testing.T) { testListScoresAPI(t, state) })
	t.Run("List Statistics API", func(t *testing.T) { testListStatisticsAPI(t, state) })
}

// --- Test Phase Implementations ---

func testAdminLogin(t *testing.T, state *TestState) {
	token := loginUser(t, adminUsername, adminPassword)
	if token == "" {
		t.Fatal("❌ Could not log in as admin.")
	}
	state.AdminToken = token
	log.Println("✅ Admin user logged in.")
}

func testRegisterAPI(t *testing.T, _ *TestState) {
	var successCount int32
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for i := 0; i < numPlayersToTest; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			username := gamertags[i]
			email := fmt.Sprintf("%s@example.com", username)
			password := "playerpass123"
			if registerUser(t, username, email, password) {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}
	wg.Wait()

	if successCount != numPlayersToTest {
		t.Fatalf("❌ Verification failed: Expected %d successful registrations, but got %d.", numPlayersToTest, successCount)
	}
	log.Printf("✅ Successfully sent %d registration requests.", successCount)

	// Test edge cases
	t.Run("Re-register random existing user", func(t *testing.T) {
		username := gamertags[rand.Intn(numPlayersToTest)]
		email := fmt.Sprintf("%s@example.com", username)
		reqBody, _ := json.Marshal(handler.RegisterRequest{Username: username, Email: email, Password: "playerpass123"})
		resp, err := makeRequest(t, "POST", apiURL+"/register", bytes.NewBuffer(reqBody), "")
		if err != nil {
			t.Fatalf("Request failed unexpectedly: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("❌ Edge case failed: Expected status 409 Conflict, but got %d", resp.StatusCode)
		}
	})

	t.Run("Register with short password", func(t *testing.T) {
		reqBody, _ := json.Marshal(handler.RegisterRequest{Username: "randomUser1", Email: "random1@example.com", Password: "123"})
		resp, err := makeRequest(t, "POST", apiURL+"/register", bytes.NewBuffer(reqBody), "")
		if err != nil {
			t.Fatalf("Request failed unexpectedly: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("❌ Edge case failed: Expected status 400 Bad Request, but got %d", resp.StatusCode)
		}
	})

	t.Run("Register with short username", func(t *testing.T) {
		reqBody, _ := json.Marshal(handler.RegisterRequest{Username: "No", Email: "random1@example.com", Password: "password123"})
		resp, err := makeRequest(t, "POST", apiURL+"/register", bytes.NewBuffer(reqBody), "")
		if err != nil {
			t.Fatalf("Request failed unexpectedly: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("❌ Edge case failed: Expected status 400 Bad Request, but got %d", resp.StatusCode)
		}
	})
	log.Println("✅ Edge cases passed.")
}

func testLoginAPI(t *testing.T, state *TestState) {
	var successCount int32
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex

	for i := 0; i < numPlayersToTest; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			username := gamertags[i]
			email := fmt.Sprintf("%s@example.com", username)
			password := "playerpass123"
			token := loginUser(t, username, password)
			if token != "" {
				atomic.AddInt32(&successCount, 1)
				claims, _ := parseJWT(token)
				mu.Lock()
				state.Players = append(state.Players, &Player{Username: username, Email: email, Password: password, Token: token, UserID: claims.UserID})
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	if int(successCount) != len(gamertags) {
		t.Fatalf("❌ Verification failed: Expected %d successful logins, but got %d.", len(gamertags), successCount)
	}
	log.Printf("✅ Successfully logged in %d players.", successCount)

	// Test edge cases
	t.Run("Login with wrong password", func(t *testing.T) {
		if loginUser(t, gamertags[0], "wrongpassword") != "" {
			t.Error("❌ Edge case failed: Login with wrong password should not return a token.")
		}
	})
	t.Run("Login with non-existent user", func(t *testing.T) {
		if loginUser(t, "nonexistentuser", "somepassword") != "" {
			t.Error("❌ Edge case failed: Login with non-existent user should not return a token.")
		}
	})
	log.Println("✅ Edge cases passed.")
}

func testAddGameAPI(t *testing.T, state *TestState) {
	gameNames := []string{"Starship Commander", "Dungeon Crawler X", "Pixel Racer", "Cyber Glitch", "Astro Colony", "Rogue Planet", "Kingdoms of Ether", "Mech Warriors Arena", "Chronos Trigger", "Void Runner"}
	// Store the created game names in the test state for later verification
	state.CreatedGameNames = gameNames

	for _, name := range gameNames {
		gameBody, _ := json.Marshal(map[string]string{"game_name": name, "description": "A test game."})
		resp, _ := makeRequest(t, "POST", apiURL+"/games", bytes.NewBuffer(gameBody), state.AdminToken)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
			t.Fatalf("❌ Failed to create game '%s', status: %s", name, resp.Status)
		}
		resp.Body.Close()
	}
	log.Println("✅ Successfully sent requests to create 10 games.")

	// Test edge case: non-admin cannot add a game.
	t.Run("Non-admin cannot add game", func(t *testing.T) {
		if len(state.Players) == 0 {
			t.Fatal("Cannot run non-admin test, no players were logged in.")
		}
		playerToken := state.Players[rand.Intn(numPlayersToTest)].Token
		gameBody, _ := json.Marshal(handler.AddGameRequest{Name: "Unauthorized Game", Description: "This should not be allowed."})
		resp, err := makeRequest(t, "POST", apiURL+"/games", bytes.NewBuffer(gameBody), playerToken)
		if err != nil {
			t.Fatalf("Request failed unexpectedly: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("❌ Edge case failed: Expected status 403 Forbidden, but got %d", resp.StatusCode)
		}
	})
	log.Println("✅ Edge cases passed.")
}

func testListGamesAPI(t *testing.T, state *TestState) {
	log.Println("Listing all available games and verifying content...")
	resp, err := makeRequest(t, "GET", apiURL+"/games", nil, "")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("❌ Failed to list games, status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&state.Games); err != nil {
		t.Fatalf("❌ Failed to decode game list response: %v", err)
	}

	// Verification: Check that the returned body contains the games we created.
	if len(state.Games) < len(state.CreatedGameNames) {
		t.Fatalf("❌ Verification failed: Expected at least %d games, but got %d.", len(state.CreatedGameNames), len(state.Games))
	}

	returnedGamesMap := make(map[string]bool)
	for _, game := range state.Games {
		returnedGamesMap[game.Name] = true
	}

	for _, expectedName := range state.CreatedGameNames {
		if !returnedGamesMap[expectedName] {
			t.Errorf("❌ Verification failed: Expected to find game '%s', but it was missing from the response.", expectedName)
		}
	}

	log.Printf("✅ Successfully listed and verified %d games.", len(state.Games))
}

func testJoinGameAPI(t *testing.T, state *TestState) {
	var wg sync.WaitGroup
	var successJionsCount int32
	sem := make(chan struct{}, concurrency)
	for _, player := range state.Players {
		wg.Add(1)
		sem <- struct{}{}
		go func(p *Player) {
			defer wg.Done()
			defer func() { <-sem }()
			numGamesToJoin := rand.Intn(4) + 1
			joined := make(map[int]bool)
			for range numGamesToJoin {
				gameID := state.Games[rand.Intn(len(state.Games))].ID
				if !joined[gameID] {
					url := fmt.Sprintf("%s/games/%d/join", apiURL, gameID)
					resp, _ := makeRequest(t, "POST", url, nil, p.Token)
					if resp.StatusCode == http.StatusCreated {
						p.GameIDs = append(p.GameIDs, gameID)
						atomic.AddInt32(&successJionsCount, 1)
					}
					resp.Body.Close()
					joined[gameID] = true
				}
			}
		}(player)
	}
	wg.Wait()

	// Couny the total number of game joins across all players
	totalGameJoins = 0
	for _, player := range state.Players {
		totalGameJoins += int32(len(player.GameIDs))
	}
	if successJionsCount != totalGameJoins {
		t.Errorf("❌ Verification failed: Expected %d successful game joins, but got %d.", totalGameJoins, successJionsCount)
	}
	log.Println("✅ 'Join game' requests completed. Expected joins: ", totalGameJoins, " - Actual joins: ", successJionsCount)
}

func testUpdateScoreAPI(t *testing.T, state *TestState) {
	var successCount int32
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	for _, player := range state.Players {
		if len(player.GameIDs) == 0 {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(p *Player) {
			defer wg.Done()
			defer func() { <-sem }()
			for _, gameID := range p.GameIDs {
				score := min(rand.Int63n(maxScore), rand.Int63n(maxScore))
				url := fmt.Sprintf("%s/games/%d/scores", apiURL, gameID)
				body, _ := json.Marshal(handler.UpdateScoreRequest{Score: fmt.Sprintf("%d", score)})
				resp, _ := makeRequest(t, "PUT", url, bytes.NewBuffer(body), p.Token)
				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotAcceptable {
					continue
				}
				atomic.AddInt32(&successCount, 1)
				resp.Body.Close()
			}
		}(player)
	}
	wg.Wait()

	if successCount != totalGameJoins {
		t.Errorf("❌ Verification failed: Expected %d successful score updates, but got %d.", totalGameJoins, successCount)
	}
	log.Println("✅ Score update requests completed. Expected updates: ", totalGameJoins, " - Actual updates: ", successCount)
}

func testListScoresAPI(t *testing.T, state *TestState) {
	for _, game := range state.Games {
		url := fmt.Sprintf("%s/games/%d/scores", apiURL, game.ID)
		resp, _ := makeRequest(t, "GET", url, nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("❌ Failed to list scores for game %d, status: %d", game.ID, resp.StatusCode)
			continue
		}
		var scores []handler.GameScoreResponse
		if err := json.NewDecoder(resp.Body).Decode(&scores); err != nil {
			t.Errorf("❌ Failed to decode scores response for game %d: %v", game.ID, err)
		}
		resp.Body.Close()
	}
	log.Println("✅ Successfully listed and decoded scores for all games.")
}

func testListStatisticsAPI(t *testing.T, state *TestState) {
	for _, game := range state.Games {
		url := fmt.Sprintf("%s/games/%d/statistics", apiURL, game.ID)
		resp, _ := makeRequest(t, "GET", url, nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("❌ Failed to list statistics for game %d, status: %d", game.ID, resp.StatusCode)
			continue
		}
		var stats handler.GameStatisticsResponse
		if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
			t.Errorf("❌ Failed to decode statistics response for game %d: %v", game.ID, err)
		}
		resp.Body.Close()
	}
	log.Println("✅ Successfully listed and decoded statistics for all games.")
}

// --- HTTP Helpers ---

func registerUser(t *testing.T, username, email, password string) bool {
	t.Helper()
	reqBody, _ := json.Marshal(handler.RegisterRequest{Username: username, Email: email, Password: password})
	resp, err := makeRequest(t, "POST", apiURL+"/register", bytes.NewBuffer(reqBody), "")
	if err != nil {
		t.Errorf("❌ Registration request failed for %s: %v", username, err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return true
	}
	t.Errorf("❌ Registration failed for %s with status: %s", username, resp.Status)
	return false
}

func loginUser(t *testing.T, username, password string) string {
	t.Helper()
	loginBody, _ := json.Marshal(handler.LoginRequest{Username: username, Password: password})
	resp, err := makeRequest(t, "POST", apiURL+"/login", bytes.NewBuffer(loginBody), "")
	if err != nil {
		t.Errorf("❌ Login request failed for %s: %v", username, err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var loginResp handler.LoginResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	return loginResp.Token
}

func makeRequest(t *testing.T, method, url string, body io.Reader, token string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return http.DefaultClient.Do(req)
}

func parseJWT(tokenString string) (*auth.JWTClaims, error) {
	claims := &auth.JWTClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(tokenString, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// Helper function: min returns the smaller of two int64 values.
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
