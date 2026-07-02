# SYSTEM CONTEXT & ARCHITECTURE SPECIFICATION
**Project:** FIDO2 / WebAuthn Passwordless Authentication Backend
**Language:** Go (Golang 1.21+)
**Architecture:** Strict Hexagonal Architecture (Domain, Application, Infrastructure)
**Database Tooling:** PostgreSQL + `sqlc` + `pgx/v5`
**Authentication Library:** `github.com/go-webauthn/webauthn`

## 1. ARCHITECTURAL RULES (CRITICAL FOR AI AGENT)
You are an expert Go developer. When generating code for this project, you MUST adhere to the following rules:
1.  **Dependency Rule:** `Domain` has no dependencies. `Application` depends ONLY on `Domain`. `Infrastructure` depends on `Application` and `Domain`.
2.  **Context Propagation:** Every interface method, use case, and repository call MUST take `ctx context.Context` as its first parameter.
3.  **No ORMs:** We use `sqlc` for database interactions. Do not generate GORM code. Write pure SQL queries that `sqlc` will compile into Go code inside the Infrastructure layer.
4.  **Error Handling:** Bubble up errors to the Infrastructure (HTTP) layer. Do not swallow errors in the Application or Domain layers.

---

## 2. FOLDER STRUCTURE
respect de current structura of the project

## 3. LAYER 1: DOMAIN

### A. Entities (internal/domain/entities.go)
Pure Go structs. No JSON or DB tags (tags can be used in DTOs or sqlc generated structs in the infra layer, but keep domain pure or minimal).

```go
type User struct {
	ID        uuid.UUID
	Username  string
	CreatedAt time.Time
}

type Credential struct {
	ID              []byte // FIDO2 Credential ID
	UserID          uuid.UUID
	PublicKey       []byte
	AttestationType string
	AAGUID          []byte
	SignCount       uint32 // CRITICAL: Must be updated on every login
	CreatedAt       time.Time
}

type WebAuthnSession struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Challenge string
	Type      string // "REGISTRATION" | "AUTHENTICATION"
	ExpiresAt time.Time
}
```

### B. Ports (internal/domain/ports.go)
Interfaces that the Application layer uses, which will be implemented by the Infrastructure layer.

```go
type UserRepository interface {
	CreateUser(ctx context.Context, username string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type CredentialRepository interface {
	SaveCredential(ctx context.Context, cred *Credential) error
	GetCredentialsByUserID(ctx context.Context, userID uuid.UUID) ([]*Credential, error)
	UpdateSignCount(ctx context.Context, credentialID []byte, newCount uint32) error
}

type SessionRepository interface {
	SaveSession(ctx context.Context, session *WebAuthnSession) error
	GetSessionByUserIDAndType(ctx context.Context, userID uuid.UUID, sessionType string) (*WebAuthnSession, error)
	DeleteSession(ctx context.Context, id uuid.UUID) error
}
```

## 4. LAYER 2: APPLICATION (USE CASES)
This layer orchestrates the go-webauthn library logic and uses the Domain Ports.

`internal/application/webauthn_service.go`
Exposes 4 main Use Cases. The Service struct should hold references to the Domain Repositories and the webauthn.WebAuthn instance.

```go
type WebAuthnAppService struct {
	userRepo   domain.UserRepository
	credRepo   domain.CredentialRepository
	sessionRepo domain.SessionRepository
	wa         *webauthn.WebAuthn
}

// USE CASES TO IMPLEMENT:
// 1. BeginRegistration(ctx, username) -> (CreationOptions, error)
// 2. FinishRegistration(ctx, username, parsedRequest) -> error
// 3. BeginLogin(ctx, username) -> (AssertionOptions, error)
// 4. FinishLogin(ctx, username, parsedRequest) -> (SessionToken, error)
```

## 5. LAYER 3: INFRASTRUCTURE

### A. Database Adapter (PostgreSQL + sqlc)
The AI agent should write pure SQL in `infrastructure/database/schema.sql` to initialize the database:

```sql
-- schema.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE credentials (
    id BYTEA PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key BYTEA NOT NULL,
    attestation_type VARCHAR(50) NOT NULL,
    aaguid BYTEA,
    sign_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE webauthn_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);
```

The AI agent must write queries in `infrastructure/database/query.sql` representing the methods defined in `domain.ports.go`. For example:

```sql
-- query.sql
-- name: CreateUser :one
INSERT INTO users (username) VALUES ($1) RETURNING *;

-- name: UpdateSignCount :exec
UPDATE credentials SET sign_count = $2 WHERE id = $1;
```

### B. HTTP Delivery Adapter
REST API exposing the Application Use Cases.

**Endpoints:**
- `POST /api/webauthn/register/begin` (Body: `{"username": "..."}`)
- `POST /api/webauthn/register/finish` (Body: FIDO2 Attestation JSON)
- `POST /api/webauthn/login/begin` (Body: `{"username": "..."}`)
- `POST /api/webauthn/login/finish` (Body: FIDO2 Assertion JSON)

**Responsibilities:**
- Parse incoming JSON requests.
- Call the corresponding Application Use Case.
- Handle errors and map them to appropriate HTTP Status Codes (400, 404, 500).
- Return JSON responses to the client. Do NOT place business logic here.
