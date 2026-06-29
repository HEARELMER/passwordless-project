package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"passwordless-backend/internal/domain/entities"
	"passwordless-backend/internal/domain/ports"
)

// SessionToken es el token de acceso que se emite tras un login exitoso.
type SessionToken struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	ExpiresIn int    `json:"expires_in"` // segundos
}

// WebAuthnAppService orquesta los 4 use cases del protocolo WebAuthn.
type WebAuthnAppService struct {
	userRepo    ports.UserRepository
	credRepo    ports.CredentialRepository
	sessionRepo ports.SessionRepository
	wa          *webauthn.WebAuthn
	jwtSecret   []byte
}

func NewWebAuthnAppService(
	userRepo ports.UserRepository,
	credRepo ports.CredentialRepository,
	sessionRepo ports.SessionRepository,
	wa *webauthn.WebAuthn,
	jwtSecret []byte,
) *WebAuthnAppService {
	return &WebAuthnAppService{
		userRepo:    userRepo,
		credRepo:    credRepo,
		sessionRepo: sessionRepo,
		wa:          wa,
		jwtSecret:   jwtSecret,
	}
}

// ─── REGISTRO ────────────────────────────────────────────────────────────────

// BeginRegistration genera el desafío de registro y lo guarda en la BD.
// Crea el usuario si no existe todavía.
// Paso A del flujo de registro de la app móvil.
func (s *WebAuthnAppService) BeginRegistration(ctx context.Context, username, email string) (*protocol.CredentialCreation, error) {
	// 1. Buscar o crear usuario.
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		// Usuario no existe → lo creamos.
		user, err = s.userRepo.CreateUser(ctx, username, email, nil)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
	}

	// 2. Cargar credenciales existentes para excluirlas en la opción excludeCredentials.
	existingCreds, err := s.credRepo.GetCredentialsByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("load credentials: %w", err)
	}

	waUser := newWebAuthnUser(user, existingCreds)

	// 3. Generar opciones de creación con go-webauthn.
	creation, sessionData, err := s.wa.BeginRegistration(waUser)
	if err != nil {
		return nil, fmt.Errorf("begin registration: %w", err)
	}

	// 4. Serializar SessionData y guardar en BD.
	if err := s.saveSession(ctx, user.ID, "REGISTRATION", sessionData); err != nil {
		return nil, err
	}

	return creation, nil
}

// FinishRegistration valida la respuesta del autenticador y guarda la credencial.
// Paso C del flujo de registro (después de la biometría).
func (s *WebAuthnAppService) FinishRegistration(ctx context.Context, username string, parsed *protocol.ParsedCredentialCreationData) error {
	// 1. Recuperar usuario y sesión.
	user, waUser, session, err := s.loadUserAndSession(ctx, username, "REGISTRATION")
	if err != nil {
		return err
	}

	// 2. Validar la respuesta criptográfica con go-webauthn.
	credential, err := s.wa.CreateCredential(waUser, *session, parsed)
	if err != nil {
		return fmt.Errorf("finish registration: %w", err)
	}

	// 3. Persistir la nueva credencial en la BD.
	now := time.Now().UTC()
	domainCred := &entities.Credential{
		ID:              credential.ID,
		UserID:          user.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		AAGUID:          credential.Authenticator.AAGUID,
		SignCount:        credential.Authenticator.SignCount,
		LastUsedAt:      &now,
		ClientInfo:      map[string]any{},
		CreatedAt:       now,
	}
	if err := s.credRepo.SaveCredential(ctx, domainCred); err != nil {
		return fmt.Errorf("save credential: %w", err)
	}

	// 4. Eliminar la sesión temporal.
	return s.sessionRepo.DeleteSession(ctx, session.ID) //nolint:wrapcheck
}

// ─── AUTENTICACIÓN ───────────────────────────────────────────────────────────

// BeginLogin genera el desafío de autenticación para el usuario.
// Paso A del flujo de login de la app móvil.
func (s *WebAuthnAppService) BeginLogin(ctx context.Context, username string) (*protocol.CredentialAssertion, error) {
	// 1. Cargar usuario y sus credenciales.
	user, waUser, err := s.loadUser(ctx, username)
	if err != nil {
		return nil, err
	}
	if len(waUser.credentials) == 0 {
		return nil, errors.New("user has no registered credentials")
	}

	// 2. Generar el challenge con go-webauthn.
	assertion, sessionData, err := s.wa.BeginLogin(waUser)
	if err != nil {
		return nil, fmt.Errorf("begin login: %w", err)
	}

	// 3. Guardar la sesión en BD.
	if err := s.saveSession(ctx, user.ID, "AUTHENTICATION", sessionData); err != nil {
		return nil, err
	}

	return assertion, nil
}

// FinishLogin verifica la firma del autenticador y emite el token de acceso.
// Paso C del flujo de login (después de la biometría).
func (s *WebAuthnAppService) FinishLogin(ctx context.Context, username string, parsed *protocol.ParsedCredentialAssertionData) (*SessionToken, error) {
	// 1. Recuperar usuario y sesión.
	user, waUser, session, err := s.loadUserAndSession(ctx, username, "AUTHENTICATION")
	if err != nil {
		return nil, err
	}

	// 2. Verificar la firma ECDSA criptográficamente.
	credential, err := s.wa.ValidateLogin(waUser, *session, parsed)
	if err != nil {
		return nil, fmt.Errorf("finish login: %w", err)
	}

	// 3. Actualizar sign_count (prevención de clonación del dispositivo).
	if err := s.credRepo.UpdateSignCount(ctx, credential.ID, credential.Authenticator.SignCount); err != nil {
		return nil, fmt.Errorf("update sign count: %w", err)
	}

	// 4. Registrar auditoría de último acceso.
	now := time.Now().UTC()
	if err := s.credRepo.UpdateCredentialAudit(ctx, credential.ID, "", map[string]any{}, &now); err != nil {
		return nil, fmt.Errorf("update audit: %w", err)
	}

	// 5. Eliminar la sesión temporal.
	if err := s.sessionRepo.DeleteSession(ctx, session.ID); err != nil {
		return nil, fmt.Errorf("delete session: %w", err)
	}

	// 6. Emitir el token de acceso (JWT HS256 con stdlib).
	token, err := GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &SessionToken{
		Token:     token,
		UserID:    user.ID.String(),
		ExpiresIn: int(tokenTTL.Seconds()),
	}, nil
}

// ─── HELPERS PRIVADOS ─────────────────────────────────────────────────────────

func (s *WebAuthnAppService) loadUser(ctx context.Context, username string) (*entities.User, *webAuthnUser, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, nil, fmt.Errorf("user not found: %w", err)
	}
	creds, err := s.credRepo.GetCredentialsByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("load credentials: %w", err)
	}
	return user, newWebAuthnUser(user, creds), nil
}

// loadUserAndSession recupera el usuario, sus credenciales y la sesión WebAuthn
// activa del tipo indicado ('REGISTRATION' o 'AUTHENTICATION').
func (s *WebAuthnAppService) loadUserAndSession(ctx context.Context, username, sessionType string) (*entities.User, *webAuthnUser, *webAuthnSession, error) {
	user, waUser, err := s.loadUser(ctx, username)
	if err != nil {
		return nil, nil, nil, err
	}

	rawSession, err := s.sessionRepo.GetSessionByUserIDAndType(ctx, user.ID, sessionType)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("session not found: %w", err)
	}

	// Verificar que la sesión no haya expirado.
	if time.Now().UTC().After(rawSession.ExpiresAt) {
		_ = s.sessionRepo.DeleteSession(ctx, rawSession.ID)
		return nil, nil, nil, fmt.Errorf("session expired")
	}

	// Re-serializar el mapa a JSON y deserializar en webauthn.SessionData.
	sessionBytes, err := json.Marshal(rawSession.SessionData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("re-encode session: %w", err)
	}
	var waSessionData webauthn.SessionData
	if err := json.Unmarshal(sessionBytes, &waSessionData); err != nil {
		return nil, nil, nil, fmt.Errorf("decode session: %w", err)
	}

	return user, waUser, &webAuthnSession{
		ID:          rawSession.ID,
		SessionData: waSessionData,
	}, nil
}

// saveSession serializa webauthn.SessionData y lo persiste en la BD.
// Elimina primero cualquier sesión anterior del mismo tipo para evitar duplicados.
func (s *WebAuthnAppService) saveSession(ctx context.Context, userID uuid.UUID, sessionType string, data *webauthn.SessionData) error {
	// Intentar eliminar sesión anterior del mismo tipo (ignoramos "not found").
	if old, err := s.sessionRepo.GetSessionByUserIDAndType(ctx, userID, sessionType); err == nil {
		_ = s.sessionRepo.DeleteSession(ctx, old.ID)
	}

	// Serializar SessionData a map[string]any para almacenarlo como JSONB.
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	var sessionMap map[string]any
	if err := json.Unmarshal(raw, &sessionMap); err != nil {
		return fmt.Errorf("decode session map: %w", err)
	}

	session := &entities.WebAuthnSession{
		ID:          uuid.New(),
		UserID:      userID,
		Challenge:   data.Challenge,
		Type:        sessionType,
		SessionData: sessionMap,
		ExpiresAt:   data.Expires,
	}
	if err := s.sessionRepo.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

// webAuthnSession envuelve webauthn.SessionData junto al UUID de BD.
type webAuthnSession struct {
	ID          uuid.UUID
	SessionData webauthn.SessionData
}
