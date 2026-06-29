package application

import (
	"github.com/go-webauthn/webauthn/webauthn"

	"passwordless-backend/internal/domain/entities"
)

// webAuthnUser adapts entities.User to satisfy the webauthn.User interface
// required by the go-webauthn library. Lives in Application because it bridges
// the Domain entity with an external library contract.
type webAuthnUser struct {
	user        *entities.User
	credentials []webauthn.Credential
}

func newWebAuthnUser(user *entities.User, creds []*entities.Credential) *webAuthnUser {
	waCreds := make([]webauthn.Credential, 0, len(creds))
	for _, c := range creds {
		waCreds = append(waCreds, domainCredToWA(c))
	}
	return &webAuthnUser{user: user, credentials: waCreds}
}

// WebAuthnID returns UUID bytes as the FIDO2 user handle.
func (u *webAuthnUser) WebAuthnID() []byte {
	b, _ := u.user.ID.MarshalBinary()
	return b
}

func (u *webAuthnUser) WebAuthnName() string        { return u.user.Username }
func (u *webAuthnUser) WebAuthnDisplayName() string { return u.user.Username }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// domainCredToWA converts a domain Credential entity into the webauthn.Credential
// struct used by go-webauthn for signature verification.
func domainCredToWA(c *entities.Credential) webauthn.Credential {
	return webauthn.Credential{
		ID:              c.ID,
		PublicKey:       c.PublicKey,
		AttestationType: c.AttestationType,
		Authenticator: webauthn.Authenticator{
			AAGUID:    c.AAGUID,
			SignCount: c.SignCount,
		},
	}
}
