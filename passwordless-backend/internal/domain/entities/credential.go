package entities

import (
	"time"

	"github.com/google/uuid"
)

type Credential struct {
	ID                  []byte
	UserID              uuid.UUID
	PublicKey           []byte
	AttestationType     string
	AAGUID              []byte
	SignCount           uint32
	LastUsedAt          *time.Time
	LastLoginIP         string
	ClientInfo          map[string]any
	RawRegistrationData map[string]any
	CreatedAt           time.Time
}
