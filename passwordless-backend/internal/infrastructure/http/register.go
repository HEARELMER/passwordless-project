package http

import (
	"errors"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
)

type beginRegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// handleRegisterBegin — Paso A del registro.
// La app envía { "username": "alice", "email": "alice@example.com" }
// El servidor responde con PublicKeyCredentialCreationOptions (contiene el challenge).
func (h *Handler) handleRegisterBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req beginRegisterRequest
	if err := readJSON(r, &req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Username == "" {
		httpError(w, http.StatusBadRequest, "username is required")
		return
	}
	if req.Email == "" {
		httpError(w, http.StatusBadRequest, "email is required")
		return
	}

	options, err := h.svc.BeginRegistration(r.Context(), req.Username, req.Email)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "begin registration failed: "+err.Error())
		return
	}

	// Responde con el JSON estándar de WebAuthn que el autenticador espera.
	writeJSON(w, http.StatusOK, options)
}

// handleRegisterFinish — Paso C del registro (tras la biometría).
// La app envía el AttestationResponse firmado por el Enclave Seguro del celular.
// El servidor verifica, guarda la llave pública y responde { "status": "ok" }.
func (h *Handler) handleRegisterFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		httpError(w, http.StatusBadRequest, "username query param is required")
		return
	}

	// ParseCredentialCreationResponseBody desempaqueta y valida el formato CBOR/JSON
	// de la respuesta del autenticador antes de pasarla al servicio.
	parsed, err := protocol.ParseCredentialCreationResponseBody(r.Body)
	if err != nil {
		var pErr *protocol.Error
		if errors.As(err, &pErr) {
			httpError(w, http.StatusBadRequest, pErr.Error())
			return
		}
		httpError(w, http.StatusBadRequest, "parse attestation response: "+err.Error())
		return
	}

	if err := h.svc.FinishRegistration(r.Context(), username, parsed); err != nil {
		httpError(w, http.StatusUnauthorized, "finish registration failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}
