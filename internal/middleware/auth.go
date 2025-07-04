package api_middleware

import (
	"context"
	"net/http"
	"strings"

	"game-scores/internal/auth"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const ClaimsContextKey = contextKey("claims")

// AuthMiddleware verifies the JWT and passes the claims down to the handler.
func AuthMiddleware(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims := &auth.JWTClaims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Token is valid. Add claims to context and call the next handler.
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext is a helper function to get claims from the context.
func ClaimsFromContext(ctx context.Context) (*auth.JWTClaims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*auth.JWTClaims)
	return claims, ok
}
