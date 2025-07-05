package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"game-scores/ent"
	"game-scores/ent/game"
	"game-scores/ent/score"
	"game-scores/ent/user"
	"game-scores/internal/decoder"
	auth_middleware "game-scores/internal/middleware"

	"github.com/go-chi/chi/v5"
)

// GameScoresHandler holds dependencies for game-related handlers.
type GameScoresHandler struct {
	Database *ent.Client
}

// AddGameRequest defines the shape of the request body for adding a score to a game.
type UpdateScoreRequest struct {
	Score string `json:"score"`
}

// GameScoreResponse defines the shape of the scores returned in the response.
type GameScoreResponse struct {
	Username string `json:"username"`
	Score    string `json:"score"`
}

type ScoreUpdateResponse struct {
	Score string `json:"score"`
}

type GameStatisticsResponse struct {
	Mean   string   `json:"mean"`
	Median string   `json:"median"`
	Mode   []string `json:"mode"`
}

// ListGameScores retrieves a game's scores from the database and returns them as a JSON response.
func (h *GameScoresHandler) ListGameScores(w http.ResponseWriter, r *http.Request) {

	// Get the game ID from the URL parameter.
	gameIDStr := chi.URLParam(r, "gameID")
	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, "Invalid game ID format", http.StatusBadRequest)
		return
	}

	exists, err := h.Database.Game.
		Query().
		Where(game.ID(gameID)).
		Exist(r.Context())

	if err != nil {
		log.Printf("Failed to check for game %d: %v", gameID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// Query the database for all scores related to this game ID.
	scores, err := h.Database.Score.
		Query().
		Where(score.HasGameWith(game.ID(gameID))). // Filter scores by the game's ID
		WithUser().                                // DB Optimization: Eager load the user who made the score
		Order(ent.Desc(score.FieldValue)).         // Sort scores in descending order by value
		All(r.Context())

	if err != nil {
		log.Printf("Failed to retrieve scores for game %d: %v", gameID, err)
		http.Error(w, "Failed to retrieve scores", http.StatusInternalServerError)
		return
	}

	// Add the scores to the response.
	scoreResponses := make([]GameScoreResponse, len(scores))
	for i, s := range scores {
		scoreResponses[i] = GameScoreResponse{
			Username: s.Edges.User.Username,
			Score:    strconv.FormatInt(s.Value, 10), // Convert int64 score to string
		}
	}

	// Send the response.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scoreResponses)
}

// JoinGame creates an initial score of 0 for the logged-in user and a specific game.
func (h *GameScoresHandler) JoinGame(w http.ResponseWriter, r *http.Request) {
	// 1. Get the User ID from the JWT claims.
	claims, ok := auth_middleware.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Could not retrieve user claims", http.StatusInternalServerError)
		return
	}
	userID := claims.UserID

	// 2. Decode the Game ID from the request body.
	gameIDStr := chi.URLParam(r, "gameID")
	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, "Invalid game ID format", http.StatusBadRequest)
		return
	}

	// 3. Check if the user has already joined this game to prevent duplicates.
	exists, err := h.Database.Score.
		Query().
		Where(
			score.HasUserWith(user.ID(userID)),
			score.HasGameWith(game.ID(gameID)),
		).
		Exist(r.Context())

	if err != nil {
		log.Printf("Failed to check for existing score: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "User has already joined this game", http.StatusConflict)
		return
	}

	// Create the new score record, value is by default set to 0.
	newScore, err := h.Database.Score.
		Create().
		SetUserID(userID).
		SetGameID(gameID).
		Save(r.Context())

	if err != nil {
		log.Printf("Failed to create score (join game): %v", err)
		http.Error(w, "Failed to join game", http.StatusInternalServerError)
		return
	}

	// Respond with a success message.
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Successfully joined game",
		"score":   newScore.Value,
	})
}

// UpdateGameScore handles updating a user's score for a specific game.
func (h *GameScoresHandler) UpdateGameScore(w http.ResponseWriter, r *http.Request) {

	// Get the user ID from the JWT
	claims, ok := auth_middleware.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Could not retrieve user claims", http.StatusInternalServerError)
		return
	}
	userID := claims.UserID

	// Get the Game ID from the URL
	gameIDStr := chi.URLParam(r, "gameID")
	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, "Invalid game ID format", http.StatusBadRequest)
		return
	}

	// Decode the new score from the request body.
	var req UpdateScoreRequest

	err = decoder.DecodeJSONBody(w, r, &req)
	if err != nil {
		log.Printf("Failed to decode update score request: %v", err)
		return
	}

	// Find the current game score of the player
	scoreToUpdate, err := h.Database.Score.
		Query().
		Where(
			score.HasUserWith(user.ID(userID)),
			score.HasGameWith(game.ID(gameID)),
		).
		Only(r.Context())

	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Score not found, player must join the game first.", http.StatusNotFound)
			return
		}
		log.Printf("Failed to find score to update: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newScore, err := strconv.ParseInt(req.Score, 10, 64)
	if err != nil {
		log.Printf("Invalid score format: %v", err)
		http.Error(w, "Invalid score format", http.StatusBadRequest)
		return
	}

	// Check if the new score is greater than the current score
	if newScore < scoreToUpdate.Value {
		http.Error(w, "New score is less than the current one, UNACCEPTABLE!", http.StatusNotAcceptable)
		return
	}

	// Update the score with the new value.
	updatedScore, err := scoreToUpdate.Update().
		SetValue(newScore).
		Save(r.Context())

	if err != nil {
		log.Printf("Failed to update score: %v", err)
		http.Error(w, "Failed to update score", http.StatusInternalServerError)
		return
	}

	// 6. Respond with the updated score.
	response := ScoreUpdateResponse{
		Score: strconv.FormatInt(updatedScore.Value, 10),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListGameScores retrieves a game's scores from the database and returns them as a JSON response.
func (h *GameScoresHandler) ListGameScoreStatistics(w http.ResponseWriter, r *http.Request) {

	// Get the game ID from the URL parameter.
	gameIDStr := chi.URLParam(r, "gameID")
	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, "Invalid game ID format", http.StatusBadRequest)
		return
	}

	exists, err := h.Database.Game.
		Query().
		Where(game.ID(gameID)).
		Exist(r.Context())

	if err != nil {
		log.Printf("Failed to check for game %d: %v", gameID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// Query the database for all scores related to this game ID.
	scores, err := h.Database.Score.
		Query().
		Where(score.HasGameWith(game.ID(gameID))). // Filter scores by the game's ID
		Order(ent.Desc(score.FieldValue)).         // Sort scores in descending order by value
		All(r.Context())

	if err != nil {
		log.Printf("Failed to retrieve scores for game %d: %v", gameID, err)
		http.Error(w, "Failed to retrieve scores", http.StatusInternalServerError)
		return
	}

	scoresArray := make([]int64, len(scores))
	for i, s := range scores {
		scoresArray[i] = s.Value // Collect all scores in an array
	}

	var mean, median int64
	var mode []int64

	if len(scoresArray) == 0 {
		mean = 0
		median = 0
		mode = []int64{0}
	} else {
		calculateMean(scoresArray, &mean)
		calculateMedian(scoresArray, &median) // Scores are already sorted in descending order, provided by the query
		calculateMode(scoresArray, &mode)
	}

	// Add the scores statistics to the response.
	scoreStatistics := GameStatisticsResponse{
		Mean:   strconv.FormatInt(mean, 10),
		Median: strconv.FormatInt(median, 10),
		Mode:   make([]string, len(mode)),
	}

	for i, m := range mode {
		scoreStatistics.Mode[i] = strconv.FormatInt(m, 10)
	}

	// Send the response.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scoreStatistics)
}

// calculateMean, calculateMedian, and calculateMode are utility functions to compute statistics on scores.
// They are used to calculate the mean, median, and mode of a slice of int64 scores.
// They assume that scores are non-empty

func calculateMean(scores []int64, mean *int64) {
	sum := int64(0)
	for _, score := range scores {
		sum += score
	}
	*mean = sum / int64(len(scores))
}

// calculateMedian assumes that the scores are sorted
func calculateMedian(scores []int64, median *int64) {
	mid := len(scores) / 2
	if len(scores)%2 == 0 {
		*median = (scores[mid-1] + scores[mid]) / 2
	} else {
		*median = scores[mid]
	}
}

func calculateMode(scores []int64, mode *[]int64) {
	maxFreq := 0
	frequency := make(map[int64]int)
	for _, score := range scores {
		frequency[score]++
		if frequency[score] > maxFreq {
			maxFreq = frequency[score]
		}
	}
	if maxFreq == 1 {
		*mode = []int64{} // No mode if all scores are unique
		return
	}
	for score, freq := range frequency {
		if freq == maxFreq {
			*mode = append(*mode, score)
		}
	}
}
