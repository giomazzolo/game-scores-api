package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"game-scores/ent"
	"game-scores/ent/user"
	"game-scores/internal/auth"
	"game-scores/internal/decoder"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinimumPasswordLength is the minimum length for user passwords.
	MinimumPasswordLength = 8
	MinumumUsernameLength = 3
	MaximumUsernameLength = 64
)

// UserHandler holds dependencies for user-related handlers.
type UserHandler struct {
	Database  *ent.Client
	JWTSecret []byte
}

// RegisterRequest defines the shape of the registration request body.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest defines the shape of the login request body.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse defines the shape of the successful login response.
type LoginResponse struct {
	Token string `json:"token"`
}

// Register handles user creation.
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	err := decoder.DecodeJSONBody(w, r, &req)
	if err != nil {
		log.Printf("Failed to decode registration request: %v", err)
		return
	}

	// Validate the username length
	if len(req.Username) < MinumumUsernameLength {
		http.Error(w, "Username must be at least "+strconv.Itoa(MinumumUsernameLength)+" characters long", http.StatusBadRequest)
		return
	}
	if len(req.Username) > MaximumUsernameLength {
		http.Error(w, "Username must not exceed "+strconv.Itoa(MaximumUsernameLength)+" characters", http.StatusBadRequest)
		return
	}

	// Validate the password length
	if len(req.Password) < MinimumPasswordLength {
		http.Error(w, "Password must be at least "+strconv.Itoa(MinimumPasswordLength)+" characters long", http.StatusBadRequest)
		return
	}

	// Hash the user's password for security
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create user in the database using the Ent client
	newUser, err := h.Database.User.
		Create().
		SetUsername(req.Username).
		SetEmail(req.Email).
		SetPasswordHash(string(hashedPassword)).
		Save(r.Context())

	if ent.IsConstraintError(err) {
		log.Printf("User already exists: %v", err)
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	log.Printf("User registered successfully: %s", newUser.Username)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

// Login handles user authentication and JWT issuance.
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	err := decoder.DecodeJSONBody(w, r, &req)
	if err != nil {
		log.Printf("Failed to decode login request: %v", err)
		return
	}

	// Find the user by username in the database
	foundUser, err := h.Database.User.
		Query().
		Where(user.UsernameEQ(req.Username)).
		Only(r.Context())

	if err != nil {
		// If user is not found, return a generic unauthorized error
		if ent.IsNotFound(err) {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		log.Printf("Failed to query user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Compare the provided password with the stored hash
	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// If password is correct, generate a JWT
	tokenString, err := auth.GenerateJWT(foundUser, h.JWTSecret)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Send the token back to the client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: tokenString})
}
