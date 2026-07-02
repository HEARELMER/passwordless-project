package entities

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	Username    string
	Email       string
	Preferences map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
