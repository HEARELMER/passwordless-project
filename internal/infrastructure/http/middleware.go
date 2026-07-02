package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"passwordless-backend/internal/application"
)

type contextKey string

const ctxUserID contextKey = "user_id"

func userIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxUserID).(uuid.UUID)
	return id, ok
}

func AuthMiddleware(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				httpError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			userID, err := application.VerifyToken(token, jwtSecret)
			if err != nil {
				httpError(w, http.StatusUnauthorized, "invalid token: "+err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), ctxUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
