package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	SessionIDKey contextKey = "session_id"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Try to get token from query parameter
			token := r.URL.Query().Get("token")
			if token == "" {
				http.Error(w, "Authorization token required", http.StatusUnauthorized)
				return
			}
			authHeader = "Bearer " + token
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		token := parts[1]
		claims, err := ValidateJWT(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, SessionIDKey, claims.SessionID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func GetUserID(ctx context.Context) string {
	userID, _ := ctx.Value(UserIDKey).(string)
	return userID
}

func GetSessionID(ctx context.Context) string {
	sessionID, _ := ctx.Value(SessionIDKey).(string)
	return sessionID
}
