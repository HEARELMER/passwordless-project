package application

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const tokenTTL = 24 * time.Hour

type tokenHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type tokenClaims struct {
	Sub string `json:"sub"` // user UUID
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

// GenerateToken crea un JWT HS256 firmado usando únicamente la librería estándar.
// No requiere dependencias externas.
func GenerateToken(userID uuid.UUID, secret []byte) (string, error) {
	now := time.Now().UTC()

	hdr, err := json.Marshal(tokenHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}

	cls, err := json.Marshal(tokenClaims{
		Sub: userID.String(),
		Exp: now.Add(tokenTTL).Unix(),
		Iat: now.Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	h64 := base64.RawURLEncoding.EncodeToString(hdr)
	c64 := base64.RawURLEncoding.EncodeToString(cls)
	payload := h64 + "." + c64

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return payload + "." + sig, nil
}

// VerifyToken valida la firma y la expiración, y retorna el UUID del usuario.
func VerifyToken(token string, secret []byte) (uuid.UUID, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return uuid.Nil, fmt.Errorf("malformed token")
	}

	payload := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return uuid.Nil, fmt.Errorf("invalid token signature")
	}

	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, fmt.Errorf("decode claims: %w", err)
	}

	var claims tokenClaims
	if err := json.Unmarshal(claimBytes, &claims); err != nil {
		return uuid.Nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	if time.Now().UTC().Unix() > claims.Exp {
		return uuid.Nil, fmt.Errorf("token expired")
	}

	id, err := uuid.Parse(claims.Sub)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse user id: %w", err)
	}

	return id, nil
}
