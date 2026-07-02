package application

import (
	"github.com/go-webauthn/webauthn/webauthn"

	"passwordless-backend/internal/domain/entities"
)

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

func (u *webAuthnUser) WebAuthnID() []byte {
	b, _ := u.user.ID.MarshalBinary()
	return b
}

func (u *webAuthnUser) WebAuthnName() string        { return u.user.Username }
func (u *webAuthnUser) WebAuthnDisplayName() string { return u.user.Username }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

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
