package webauthn_adapter

import (
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"passwordless-backend/internal/domain/ports"
)

type Adapter struct {
	wa *webauthn.WebAuthn
}

func NewAdapter(wa *webauthn.WebAuthn) ports.WebAuthnPort {
	return &Adapter{wa: wa}
}

func (a *Adapter) BeginRegistration(user webauthn.User, opts ...webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	return a.wa.BeginRegistration(user, opts...)
}

func (a *Adapter) CreateCredential(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
	return a.wa.CreateCredential(user, session, response)
}

func (a *Adapter) BeginLogin(user webauthn.User, opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	return a.wa.BeginLogin(user, opts...)
}

func (a *Adapter) ValidateLogin(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error) {
	return a.wa.ValidateLogin(user, session, response)
}
