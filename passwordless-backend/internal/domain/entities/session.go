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
	SessionData map[string]any
	ExpiresAt   time.Time
}
