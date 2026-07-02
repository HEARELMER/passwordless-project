package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"passwordless-backend/internal/domain/entities"
	"passwordless-backend/internal/domain/ports"
)

type SessionToken struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	ExpiresIn int    `json:"expires_in"`
}

type WebAuthnAppService struct {
	userRepo    ports.UserRepository
	credRepo    ports.CredentialRepository
	sessionRepo ports.SessionRepository
	wa          ports.WebAuthnPort
	jwtSecret   []byte
}

func NewWebAuthnAppService(
	userRepo ports.UserRepository,
	credRepo ports.CredentialRepository,
	sessionRepo ports.SessionRepository,
	wa ports.WebAuthnPort,
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

func (s *WebAuthnAppService) BeginRegistration(ctx context.Context, username, email, userAgent, ipAddress string) (*protocol.CredentialCreation, uuid.UUID, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		user, err = s.userRepo.CreateUser(ctx, username, email, nil)
		if err != nil {
			return nil, uuid.Nil, fmt.Errorf("create user: %w", err)
		}
	}

	existingCreds, err := s.credRepo.GetCredentialsByUserID(ctx, user.ID)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("load credentials: %w", err)
	}

	waUser := newWebAuthnUser(user, existingCreds)

	creation, sessionData, err := s.wa.BeginRegistration(waUser)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("begin registration: %w", err)
	}

	sessionID, err := s.saveSession(ctx, user.ID, "REGISTRATION", sessionData, userAgent, ipAddress)
	if err != nil {
		return nil, uuid.Nil, err
	}

	return creation, sessionID, nil
}

func (s *WebAuthnAppService) FinishRegistration(ctx context.Context, sessionID uuid.UUID, parsed *protocol.ParsedCredentialCreationData) error {
	user, waUser, session, err := s.loadUserAndSession(ctx, sessionID, "REGISTRATION")
	if err != nil {
		return err
	}

	credential, err := s.wa.CreateCredential(waUser, session.SessionData, parsed)
	if err != nil {
		return fmt.Errorf("finish registration: %w", err)
	}

	now := time.Now().UTC()
	domainCred := &entities.Credential{
		ID:              credential.ID,
		UserID:          user.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		AAGUID:          credential.Authenticator.AAGUID,
		SignCount:       credential.Authenticator.SignCount,
		IsActive:        true,
		LastUsedAt:      &now,
		ClientInfo:      map[string]any{},
		CreatedAt:       now,
	}
	if err := s.credRepo.SaveCredential(ctx, domainCred); err != nil {
		return fmt.Errorf("save credential: %w", err)
	}

	log.Printf("[SECURITY] register_success user_id=%s", user.ID)

	return s.sessionRepo.UpdateSessionStatus(ctx, session.ID, "COMPLETED") //nolint:wrapcheck
}

func (s *WebAuthnAppService) BeginLogin(ctx context.Context, username, userAgent, ipAddress string) (*protocol.CredentialAssertion, uuid.UUID, error) {
	user, waUser, err := s.loadUser(ctx, username)
	if err != nil {
		log.Printf("[SECURITY] login_failed user=%s reason=user_not_found", username)
		return nil, uuid.Nil, err
	}
	if len(waUser.credentials) == 0 {
		log.Printf("[SECURITY] login_failed user=%s reason=no_credentials", username)
		return nil, uuid.Nil, errors.New("user has no registered credentials")
	}

	assertion, sessionData, err := s.wa.BeginLogin(waUser)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("begin login: %w", err)
	}

	sessionID, err := s.saveSession(ctx, user.ID, "AUTHENTICATION", sessionData, userAgent, ipAddress)
	if err != nil {
		return nil, uuid.Nil, err
	}

	return assertion, sessionID, nil
}

func (s *WebAuthnAppService) FinishLogin(ctx context.Context, sessionID uuid.UUID, parsed *protocol.ParsedCredentialAssertionData) (*SessionToken, error) {
	user, waUser, session, err := s.loadUserAndSession(ctx, sessionID, "AUTHENTICATION")
	if err != nil {
		return nil, err
	}

	credential, err := s.wa.ValidateLogin(waUser, session.SessionData, parsed)
	if err != nil {
		log.Printf("[SECURITY] login_failed user=%s reason=invalid_signature", user.Username)
		return nil, fmt.Errorf("finish login: %w", err)
	}

	if err := s.credRepo.UpdateSignCount(ctx, credential.ID, credential.Authenticator.SignCount); err != nil {
		return nil, fmt.Errorf("update sign count: %w", err)
	}

	now := time.Now().UTC()
	rawSession, _ := s.sessionRepo.GetSessionByUserIDAndType(ctx, user.ID, "AUTHENTICATION")
	ip, ua := "", ""
	if rawSession != nil {
		ip = rawSession.IPAddress
		ua = rawSession.UserAgent
	}
	clientInfo := map[string]any{"user_agent": ua}
	if err := s.credRepo.UpdateCredentialAudit(ctx, credential.ID, ip, clientInfo, &now); err != nil {
		return nil, fmt.Errorf("update audit: %w", err)
	}

	if err := s.sessionRepo.UpdateSessionStatus(ctx, session.ID, "COMPLETED"); err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	token, err := GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	log.Printf("[SECURITY] login_success user_id=%s ip=%s", user.ID, ip)

	return &SessionToken{
		Token:     token,
		UserID:    user.ID.String(),
		ExpiresIn: int(tokenTTL.Seconds()),
	}, nil
}

func (s *WebAuthnAppService) GetUserCredentials(ctx context.Context, userID uuid.UUID) ([]*entities.Credential, error) {
	return s.credRepo.GetCredentialsByUserID(ctx, userID)
}

func (s *WebAuthnAppService) RevokeCredential(ctx context.Context, userID uuid.UUID, credentialID []byte) error {
	cred, err := s.credRepo.GetCredentialByID(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("credential not found: %w", err)
	}
	if cred.UserID != userID {
		return errors.New("forbidden: credential does not belong to user")
	}
	log.Printf("[SECURITY] credential_revoked user_id=%s", userID)
	return s.credRepo.DeactivateCredential(ctx, credentialID, userID) //nolint:wrapcheck
}

func (s *WebAuthnAppService) GetUserByID(ctx context.Context, userID uuid.UUID) (*entities.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}

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

func (s *WebAuthnAppService) loadUserAndSession(ctx context.Context, sessionID uuid.UUID, sessionType string) (*entities.User, *webAuthnUser, *webAuthnSession, error) {
	rawSession, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("session not found: %w", err)
	}
	if rawSession.Type != sessionType {
		return nil, nil, nil, fmt.Errorf("invalid session type")
	}

	user, err := s.userRepo.GetUserByID(ctx, rawSession.UserID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("user not found: %w", err)
	}
	creds, err := s.credRepo.GetCredentialsByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load credentials: %w", err)
	}
	waUser := newWebAuthnUser(user, creds)

	if time.Now().UTC().After(rawSession.ExpiresAt) {
		_ = s.sessionRepo.UpdateSessionStatus(ctx, rawSession.ID, "EXPIRED")
		log.Printf("[SECURITY] session_expired user_id=%s", user.ID)
		return nil, nil, nil, fmt.Errorf("session expired")
	}

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

func (s *WebAuthnAppService) saveSession(ctx context.Context, userID uuid.UUID, sessionType string, data *webauthn.SessionData, userAgent, ipAddress string) (uuid.UUID, error) {
	if old, err := s.sessionRepo.GetSessionByUserIDAndType(ctx, userID, sessionType); err == nil {
		_ = s.sessionRepo.UpdateSessionStatus(ctx, old.ID, "EXPIRED")
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal session: %w", err)
	}
	var sessionMap map[string]any
	if err := json.Unmarshal(raw, &sessionMap); err != nil {
		return uuid.Nil, fmt.Errorf("decode session map: %w", err)
	}

	sessionID := uuid.New()
	session := &entities.WebAuthnSession{
		ID:          sessionID,
		UserID:      userID,
		Challenge:   data.Challenge,
		Type:        sessionType,
		Status:      "PENDING",
		SessionData: sessionMap,
		UserAgent:   userAgent,
		IPAddress:   ipAddress,
		ExpiresAt:   data.Expires,
	}
	if err := s.sessionRepo.SaveSession(ctx, session); err != nil {
		return uuid.Nil, fmt.Errorf("save session: %w", err)
	}
	return sessionID, nil
}

type webAuthnSession struct {
	ID          uuid.UUID
	SessionData webauthn.SessionData
}
