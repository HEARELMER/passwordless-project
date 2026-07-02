package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"passwordless-backend/internal/domain/entities"
)

type UserRepository interface {
	CreateUser(ctx context.Context, username string, email string, preferences map[string]any) (*entities.User, error)
	GetUserByUsername(ctx context.Context, username string) (*entities.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
}

type CredentialRepository interface {
	SaveCredential(ctx context.Context, cred *entities.Credential) error
	GetCredentialsByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Credential, error)
	GetCredentialsByUserIDPaginated(ctx context.Context, userID uuid.UUID, limit int, offset int) ([]*entities.Credential, error)
	GetCredentialsByUserIDFiltered(ctx context.Context, userID uuid.UUID, filters []Filter, opts *ListOptions) ([]*entities.Credential, error)
	GetCredentialByID(ctx context.Context, credentialID []byte) (*entities.Credential, error)
	UpdateSignCount(ctx context.Context, credentialID []byte, newCount uint32) error
	UpdateCredentialAudit(ctx context.Context, credentialID []byte, lastLoginIP string, clientInfo map[string]any, lastUsedAt *time.Time) error
	DeactivateCredential(ctx context.Context, credentialID []byte, userID uuid.UUID) error
}

type SessionRepository interface {
	SaveSession(ctx context.Context, session *entities.WebAuthnSession) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*entities.WebAuthnSession, error)
	GetSessionByUserIDAndType(ctx context.Context, userID uuid.UUID, sessionType string) (*entities.WebAuthnSession, error)
	UpdateSessionStatus(ctx context.Context, id uuid.UUID, status string) error
}
