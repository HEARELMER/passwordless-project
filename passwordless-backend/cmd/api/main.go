package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"passwordless-backend/internal/application"
	"passwordless-backend/internal/infrastructure/database"
	httpapi "passwordless-backend/internal/infrastructure/http"
)

func main() {
	ctx := context.Background()

	// ── Variables de entorno ────────────────────────────────────────────────
	addr        := getEnv("HTTP_ADDR", ":8080")
	databaseURL := mustEnv("DATABASE_URL")
	jwtSecret   := []byte(mustEnv("JWT_SECRET"))

	// Configuración WebAuthn (FIDO2 Relying Party).
	// RP_ID     → dominio del servidor (ej: "example.com" o "localhost")
	// RP_ORIGIN → URL completa del cliente (ej: "https://example.com" o "http://localhost:8080")
	rpID          := getEnv("RP_ID", "localhost")
	rpOrigin      := getEnv("RP_ORIGIN", "http://localhost:8080")
	rpDisplayName := getEnv("RP_DISPLAY_NAME", "Passwordless App")

	// ── Base de datos ───────────────────────────────────────────────────────
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	if err := database.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("db migrations failed: %v", err)
	}

	// ── WebAuthn ────────────────────────────────────────────────────────────
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: rpDisplayName,
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    60 * time.Second,
				TimeoutUVD: 60 * time.Second,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    60 * time.Second,
				TimeoutUVD: 60 * time.Second,
			},
		},
	})
	if err != nil {
		log.Fatalf("webauthn init failed: %v", err)
	}

	// ── Repositorios y servicio ─────────────────────────────────────────────
	userRepo    := database.NewUserRepository(pool)
	credRepo    := database.NewCredentialRepository(pool)
	sessionRepo := database.NewSessionRepository(pool)

	svc     := application.NewWebAuthnAppService(userRepo, credRepo, sessionRepo, wa, jwtSecret)
	handler := httpapi.NewHandler(svc)

	// ── Router HTTP ─────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("server listening on %s  (RP_ID=%s)", addr, rpID)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server failed: %v", err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
