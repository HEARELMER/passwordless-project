package ports

import (
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type WebAuthnPort interface {
	BeginRegistration(user webauthn.User, opts ...webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error)
	CreateCredential(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error)
	BeginLogin(user webauthn.User, opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	ValidateLogin(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error)
}
