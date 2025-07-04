package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// --- Configuration Constants ---
const (
	apiURL = "http://localhost:8080"

	// Admin configuration
	adminUsername = "admin"
	adminPassword = "admin123!"
	adminEmail    = "admin@example.com"

	// Player simulation configuration
	numPlayers      = 100 // Total number of players to register
	registerBatch   = 5   // Number of players to register concurrently
	joinGameBatch   = 10  // Number of players to join games concurrently
	updateTestCount = 5   // Number of random players to select for the final score update test
)

// --- Helper Structs for API Responses ---

// LoginResponse captures the token from the /login endpoint.
type LoginResponse struct {
	Token string `json:"token"`
}

// Game captures the data for a single game from the /games endpoint.
type Game struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Player holds the token and joined game IDs for a simulated player.
type Player struct {
	Username string
	Token    string
	GameIDs  []int
}

// --- Main Test Execution ---

func main() {
	log.Println("--- Starting API Load Test ---")
	// Seed the random number generator to ensure different results each run.
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// --- Step 1: Register and Login as Admin ---
	// The admin account is required to create the games for players to join.
	log.Println("🔹 [Phase 1] Admin Setup")
	adminToken := registerAndLogin(adminUsername, adminEmail, adminPassword)
	if adminToken == "" {
		log.Fatal("❌ Failed to get admin token. Aborting test.")
	}
	log.Println("✅ Admin logged in successfully.")

	// --- Step 2: Admin Creates Games ---
	// The admin creates a list of games that players will interact with.
	gameIDs := createGames(adminToken)
	if len(gameIDs) == 0 {
		log.Fatal("❌ Failed to create games. Aborting test.")
	}
	log.Printf("✅ Admin created %d games.\n", len(gameIDs))

	// --- Step 3: Concurrently Register Players ---
	// We simulate 100 players registering in batches of 5 to create load.
	log.Println("🔹 [Phase 2] Player Registration (100 players, 5 at a time)")
	players := registerPlayers()
	log.Printf("✅ Successfully registered and logged in %d players.\n", len(players))

	// --- Step 4: Concurrently Join Games ---
	// Players will now join a random number of games (1-4) in batches of 10.
	log.Println("🔹 [Phase 3] Game Joining (10 players at a time)")
	joinGames(players, gameIDs)
	log.Printf("✅ All %d players have joined their games.\n", len(players))

	// --- Step 5: Simulate Score Updates and Reads ---
	// A random subset of 5 players will update scores and read data concurrently.
	log.Println("🔹 [Phase 4] Score Updates & Reads (5 random players)")
	simulatePlayerActivity(players)

	log.Println("\n--- ✅ Load Test Finished Successfully ---")
}

// --- Test Logic Functions ---

// registerPlayers creates 100 player accounts by running registration goroutines in batches.
func registerPlayers() []*Player {
	// A slice to hold the created players. We pre-allocate to avoid resizing.
	players := make([]*Player, 0, numPlayers)
	var wg sync.WaitGroup
	// A channel to use as a semaphore to limit concurrency to the batch size.
	sem := make(chan struct{}, registerBatch)
	// A mutex to safely append to the players slice from multiple goroutines.
	var mu sync.Mutex

	for i, gamertag := range gamertags {
		wg.Add(1)
		sem <- struct{}{} // Acquire a slot
		go func(playerNum int, username string) {
			defer wg.Done()
			defer func() { <-sem }() // Release the slot

			email := fmt.Sprintf("%s@example.com", username)
			password := "playerpass123"

			playerToken := registerAndLogin(username, email, password)
			if playerToken == "" {
				// The registerAndLogin function already logs the specific error
				return
			}
			log.Printf("▶️ Player %d (%s) registered and logged in.", playerNum, username)

			// Safely append the new player to the slice
			mu.Lock()
			players = append(players, &Player{Username: username, Token: playerToken})
			mu.Unlock()
		}(i, gamertag)
	}

	wg.Wait()
	return players
}

// joinGames has players concurrently join 1-4 random games.
func joinGames(players []*Player, gameIDs []int) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, joinGameBatch)

	for _, player := range players {
		wg.Add(1)
		sem <- struct{}{}
		go func(p *Player) {
			defer wg.Done()
			defer func() { <-sem }()

			numGamesToJoin := rand.Intn(4) + 1 // Join 1 to 4 games
			// Use a map to prevent joining the same game twice
			joined := make(map[int]bool)
			for i := 0; i < numGamesToJoin; i++ {
				gameID := gameIDs[rand.Intn(len(gameIDs))]
				if !joined[gameID] {
					joinGame(p.Token, gameID)
					p.GameIDs = append(p.GameIDs, gameID) // Track joined games
					joined[gameID] = true
				}
			}
			log.Printf("  ▶️ Player %s joined %d games.", p.Username, len(p.GameIDs))
		}(player)
	}
	wg.Wait()
}

// simulatePlayerActivity selects 5 random players to update and read scores.
func simulatePlayerActivity(players []*Player) {
	var wg sync.WaitGroup
	// Shuffle players to pick 5 random ones for the test.
	rand.Shuffle(len(players), func(i, j int) { players[i], players[j] = players[j], players[i] })

	for i := 0; i < updateTestCount && i < len(players); i++ {
		wg.Add(1)
		go func(player *Player) {
			defer wg.Done()
			log.Printf("    ▶️ Player %s starting activity...", player.Username)
			for _, gameID := range player.GameIDs {
				// Roll twice for the score and take the smaller value.
				score1 := rand.Int63n(10000)
				score2 := rand.Int63n(10000)
				newScore := min(score1, score2)

				// Update the score for the game.
				updateScore(player.Token, gameID, newScore)
				log.Printf("      ▶️ Player %s updated score in game %d to %d", player.Username, gameID, newScore)

				// After updating, read the full leaderboard for that game.
				getGameScores(player.Token, gameID)
				log.Printf("      ▶️ Player %s read leaderboard for game %d", player.Username, gameID)

				// Then, read the game statistics.
				getGameStats(player.Token, gameID)
				log.Printf("      ▶️ Player %s read stats for game %d", player.Username, gameID)
			}
		}(players[i])
	}
	wg.Wait()
}

// --- HTTP Helper Functions ---

// registerAndLogin handles the full registration and login flow for a single user.
func registerAndLogin(username, email, password string) string {
	// Register user
	regBodyMap := map[string]string{"username": username, "email": email, "password": password}
	regBody, _ := json.Marshal(regBodyMap)
	resp, err := makeRequest("POST", apiURL+"/register", bytes.NewBuffer(regBody), "")
	if err != nil {
		log.Printf("❌ Error during registration for user %s: %v", username, err)
		return ""
	}
	// We ignore specific status codes like 409 Conflict if the user already exists.
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("❌ Registration failed for user %s with status: %s, response: %s", username, resp.Status, string(respBody))
	}
	resp.Body.Close()

	// Login user to get the token.
	loginBodyMap := map[string]string{"username": username, "password": password}
	loginBody, _ := json.Marshal(loginBodyMap)
	resp, err = makeRequest("POST", apiURL+"/login", bytes.NewBuffer(loginBody), "")
	if err != nil {
		log.Printf("❌ Error during login for user %s: %v", username, err)
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("❌ Login failed for user %s with status: %s, response: %s", username, resp.Status, string(respBody))
		return ""
	}
	defer resp.Body.Close()

	var loginResp LoginResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	return loginResp.Token
}

// createGames has the admin create a predefined list of 10 games.
func createGames(token string) []int {
	gameNames := []string{
		"Starship Commander", "Dungeon Crawler X", "Pixel Racer", "Cyber Glitch", "Astro Colony",
		"Rogue Planet", "Kingdoms of Ether", "Mech Warriors Arena", "Chronos Trigger", "Void Runner",
	}
	var gameIDs []int

	for _, name := range gameNames {
		// CORRECTED: The API expects "game_name" as the key, not "name".
		gameBodyMap := map[string]string{"game_name": name, "description": "A test game created by admin."}
		gameBody, _ := json.Marshal(gameBodyMap)
		resp, err := makeRequest("POST", apiURL+"/games", bytes.NewBuffer(gameBody), token)
		if err != nil {
			log.Printf("❌ Error creating game '%s': %v", name, err)
			continue
		}
		// We ignore "Conflict" errors in case games exist from a previous run.
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
			respBody, _ := io.ReadAll(resp.Body)
			log.Printf("❌ Failed to create game '%s', status: %s, response: %s", name, resp.Status, string(respBody))
		}
		resp.Body.Close()
	}

	// Get all games to retrieve their IDs for later use.
	resp, err := makeRequest("GET", apiURL+"/games", nil, "")
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Println("❌ Failed to retrieve the list of games.")
		return nil
	}
	defer resp.Body.Close()

	var games []Game
	json.NewDecoder(resp.Body).Decode(&games)
	for _, g := range games {
		gameIDs = append(gameIDs, g.ID)
	}
	return gameIDs
}

// joinGame sends a request for the authenticated user to join a game.
func joinGame(token string, gameID int) {
	// CORRECTED: The route is /games/{gameID}/join and takes no body.
	url := fmt.Sprintf("%s/games/%d/join", apiURL, gameID)
	resp, err := makeRequest("POST", url, nil, token)
	if err != nil {
		log.Printf("❌ Error joining game %d: %v", gameID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("❌ Failed to join game %d, status: %s, response: %s", gameID, resp.Status, string(respBody))
	}
}

// updateScore sends a request to update the user's score in a specific game.
func updateScore(token string, gameID int, score int64) {
	// CORRECTED: The route is /games/{gameID}/scores (plural).
	// CORRECTED: The API expects the score as a string.
	url := fmt.Sprintf("%s/games/%d/scores", apiURL, gameID)
	body, _ := json.Marshal(map[string]string{"score": fmt.Sprintf("%d", score)})
	resp, err := makeRequest("PUT", url, bytes.NewBuffer(body), token)
	if err != nil {
		log.Printf("❌ Error updating score for game %d: %v", gameID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("❌ Failed to update score for game %d, status: %s, response: %s", gameID, resp.Status, string(respBody))
	}
}

// getGameScores sends a request to get the full leaderboard for a game.
func getGameScores(token string, gameID int) {
	url := fmt.Sprintf("%s/games/%d/scores", apiURL, gameID)
	resp, err := makeRequest("GET", url, nil, "") // This is a public endpoint
	if err != nil {
		log.Printf("❌ Error getting scores for game %d: %v", gameID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("❌ Failed to get scores for game %d, status: %s, response: %s", gameID, resp.Status, string(respBody))
	}
}

// getGameStats sends a request to get the statistics for a game.
func getGameStats(token string, gameID int) {
	// CORRECTED: The route is /games/{gameID}/statistics.
	url := fmt.Sprintf("%s/games/%d/statistics", apiURL, gameID)
	resp, err := makeRequest("GET", url, nil, "") // This is a public endpoint
	if err != nil {
		log.Printf("❌ Error getting stats for game %d: %v", gameID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("❌ Failed to get stats for game %d, status: %s, response: %s", gameID, resp.Status, string(respBody))
	}
}

// makeRequest is a generic helper to create and send HTTP requests.
func makeRequest(method, url string, body io.Reader, token string) (*http.Response, error) {
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

// --- Data & Utilities ---

// A list of 100 gamertags for creating players.
var gamertags = []string{
	"ShadowStriker", "CyberNinja", "VortexViper", "QuantumQuake", "NightHawk", "SolarFlare", "IronClad", "GhostReaper",
	"VenomSpike", "BladeRunner", "ZeroGravity", "WarpRider", "SteelStorm", "BlazeDragon", "FrostBite", "OmegaWolf",
	"RogueSpecter", "ThunderGod", "CrimsonFang", "DarkPhoenix", "Hyperion", "Nemesis", "Vindicator", "Wraith",
	"Avalanche", "Blitz", "Cobra", "Dynamo", "Echo", "Falcon", "Goliath", "Havoc", "Inferno", "Juggernaut", "Kestrel",
	"Leviathan", "Maverick", "Nova", "Obsidian", "Pulsar", "Quasar", "Razor", "Scorpion", "Talon", "Uprising",
	"Vanguard", "Warlock", "Xenon", "Yankee", "Zephyr", "ApexPredator", "BlackMamba", "Cyclone", "DeathWish",
	"Eradicator", "FireFly", "Gunslinger", "HeadHunter", "IceMan", "Jackal", "KnightRider", "LoneWolf", "MadDog",
	"Nightmare", "Outlaw", "Phantom", "QuickSilver", "Rampage", "ShadowHunter", "Terminator", "Undertaker", "Viper",
	"Wolverine", "X-Factor", "YellowJacket", "Zodiac", "Alpha", "Bravo", "Charlie", "Delta", "Foxtrot", "Gamma",
	"Helios", "Icarus", "Jester", "Karma", "Loki", "Midas", "Nero", "Orion", "Phoenix", "Raptor", "Siren", "Titan",
}

// min returns the smaller of two int64 values.
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
