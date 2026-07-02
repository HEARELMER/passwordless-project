package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/joho/godotenv"

	"passwordless-backend/internal/application"
	"passwordless-backend/internal/infrastructure/database"
	httpapi "passwordless-backend/internal/infrastructure/http"
	webauthn_adapter "passwordless-backend/internal/infrastructure/webauthn"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := getEnv("HTTP_ADDR", ":"+port)
	databaseURL   := mustEnv("DATABASE_URL")
	jwtSecretStr  := mustEnv("JWT_SECRET")
	rpID          := getEnv("RP_ID", "localhost")
	rpOrigin      := getEnv("RP_ORIGIN", "http://localhost:8080")
	rpDisplayName := getEnv("RP_DISPLAY_NAME", "Passwordless App")

	if len(jwtSecretStr) < 32 {
		log.Fatalf("JWT_SECRET must be at least 32 characters long (got %d)", len(jwtSecretStr))
	}
	jwtSecret := []byte(jwtSecretStr)

	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	if err := database.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("db migrations failed: %v", err)
	}

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

	userRepo    := database.NewUserRepository(pool)
	credRepo    := database.NewCredentialRepository(pool)
	sessionRepo := database.NewSessionRepository(pool)

	waAdapter := webauthn_adapter.NewAdapter(wa)
	svc       := application.NewWebAuthnAppService(userRepo, credRepo, sessionRepo, waAdapter, jwtSecret)
	handler   := httpapi.NewHandler(svc, jwtSecret)

	application.StartSessionCleanupWorker(ctx, pool, 1*time.Minute)

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

	go func() {
		log.Printf("server listening on %s (RP_ID=%s)", addr, rpID)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("server stopped")
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
