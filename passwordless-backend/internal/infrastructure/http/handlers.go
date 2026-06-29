package http

import (
	"encoding/json"
	"log"
	"net/http"

	"passwordless-backend/internal/application"
)

// Handler encapsula el servicio de aplicación y expone los endpoints REST
// que consume la app móvil autenticadora.
type Handler struct {
	svc *application.WebAuthnAppService
}

func NewHandler(svc *application.WebAuthnAppService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes monta los 4 endpoints del protocolo WebAuthn.
//
// Flujo Registro:
//
//	POST /api/webauthn/register/begin   ← app envía {username, email}
//	POST /api/webauthn/register/finish  ← app envía AttestationResponse
//
// Flujo Login (lo que describió el usuario — Paso A/B/C):
//
//	POST /api/webauthn/login/begin      ← app envía {username}  (Paso A)
//	                                    → servidor responde con challenge (Paso B)
//	POST /api/webauthn/login/finish     ← app envía firma tras biometría (Paso C)
//	                                    → servidor responde con JWT
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/webauthn/register/begin", h.handleRegisterBegin)
	mux.HandleFunc("/api/webauthn/register/finish", h.handleRegisterFinish)
	mux.HandleFunc("/api/webauthn/login/begin", h.handleLoginBegin)
	mux.HandleFunc("/api/webauthn/login/finish", h.handleLoginFinish)
	mux.HandleFunc("/health", h.handleHealth)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

func readJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
