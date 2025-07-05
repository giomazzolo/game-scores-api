package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"game-scores/ent"

	"game-scores/internal/decoder"
	auth_middleware "game-scores/internal/middleware"
)

// GameHandler holds dependencies for game-related handlers.
type GameHandler struct {
	Database *ent.Client
}

// AddGameRequest defines the shape of the request body for adding a new game.
type AddGameRequest struct {
	Name        string `json:"game_name"`
	Description string `json:"description"`
}

// GameResponse defines the shape of the list of games returned in the response.
type GameResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AddGame handles the addition of a new game to the database.
func (h *GameHandler) AddGame(w http.ResponseWriter, r *http.Request) {

	claims, ok := auth_middleware.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Could not retrieve user claims", http.StatusInternalServerError)
		return
	}

	// Only admins can add games
	if claims.Role != "admin" {
		http.Error(w, "Forbidden: This action requires admin privileges", http.StatusForbidden)
		return
	}

	var req AddGameRequest

	err := decoder.DecodeJSONBody(w, r, &req)
	if err != nil {
		log.Printf("Failed to decode add game request: %v", err)
		return
	}

	if req.Name == "" {
		http.Error(w, "Game name cannot be empty", http.StatusBadRequest)
		return
	}

	// Add game in the database using the Ent client
	newGame, err := h.Database.Game.
		Create().
		SetName(req.Name).
		SetDescription(req.Description).
		Save(r.Context())

	if ent.IsConstraintError(err) {
		log.Printf("Game with this name already exists: %v", err)
		http.Error(w, "Game with this name already exists", http.StatusConflict)
		return
	}
	if err != nil {
		log.Printf("Failed to create game: %v", err)
		http.Error(w, "Failed to create game", http.StatusInternalServerError)
		return
	}

	log.Printf("Game added successfully: %s, ID: %d", newGame.Name, newGame.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Game added successfully"})
}

// ListGames retrieves all games from the database and returns them as a JSON response.
func (h *GameHandler) ListGames(w http.ResponseWriter, r *http.Request) {

	// Get all games from the database
	gamesList, err := h.Database.Game.
		Query().
		All(r.Context())

	if err != nil {
		log.Printf("Failed to retrieve games %v", err)
		http.Error(w, "Failed to retrieve games", http.StatusInternalServerError)
		return
	}

	gameResponses := make([]GameResponse, len(gamesList))
	for i, game := range gamesList {
		gameResponses[i] = GameResponse{
			ID:          game.ID,
			Name:        game.Name,
			Description: game.Description,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gameResponses)
}
