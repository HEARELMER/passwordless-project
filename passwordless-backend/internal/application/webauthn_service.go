package application

import (
	"context"
	"errors"

	"github.com/go-webauthn/webauthn/webauthn"

	"passwordless-backend/internal/domain/ports"
)

type CreationOptions struct {
	Data map[string]any
}

type AssertionOptions struct {
	Data map[string]any
}

type SessionToken struct {
	Token string
}

type WebAuthnAppService struct {
	userRepo    ports.UserRepository
	credRepo    ports.CredentialRepository
	sessionRepo ports.SessionRepository
	wa          *webauthn.WebAuthn
}

func NewWebAuthnAppService(
	userRepo ports.UserRepository,
	credRepo ports.CredentialRepository,
	sessionRepo ports.SessionRepository,
	wa *webauthn.WebAuthn,
) *WebAuthnAppService {
	return &WebAuthnAppService{
		userRepo:    userRepo,
		credRepo:    credRepo,
		sessionRepo: sessionRepo,
		wa:          wa,
	}
}

func (s *WebAuthnAppService) BeginRegistration(ctx context.Context, username string) (*CreationOptions, error) {
	return nil, errors.New("not implemented")
}

func (s *WebAuthnAppService) FinishRegistration(ctx context.Context, username string, parsedRequest any) error {
	return errors.New("not implemented")
}

func (s *WebAuthnAppService) BeginLogin(ctx context.Context, username string) (*AssertionOptions, error) {
	return nil, errors.New("not implemented")
}

func (s *WebAuthnAppService) FinishLogin(ctx context.Context, username string, parsedRequest any) (*SessionToken, error) {
	return nil, errors.New("not implemented")
}
