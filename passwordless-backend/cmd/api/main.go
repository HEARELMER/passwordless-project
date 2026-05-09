package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"passwordless-backend/internal/application"
	"passwordless-backend/internal/infrastructure/database"
	httpapi "passwordless-backend/internal/infrastructure/http"
)

func main() {
	ctx := context.Background()

	addr := getEnv("HTTP_ADDR", ":8080")
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	userRepo := database.NewUserRepository(pool)
	credRepo := database.NewCredentialRepository(pool)
	sessionRepo := database.NewSessionRepository(pool)

	app := application.NewWebAuthnAppService(userRepo, credRepo, sessionRepo, nil)
	handler := httpapi.NewHandler(app)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
