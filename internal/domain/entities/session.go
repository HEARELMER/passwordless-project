package entities

import (
	"time"

	"github.com/google/uuid"
)

type WebAuthnSession struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Challenge   string
	Type        string
	Status      string
	SessionData map[string]any
	UserAgent   string
	IPAddress   string
	ExpiresAt   time.Time
}
