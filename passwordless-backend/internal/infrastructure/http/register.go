package http

import (
	"errors"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type beginRegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

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

	options, sessionID, err := h.svc.BeginRegistration(r.Context(), req.Username, req.Email, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "begin registration failed: "+err.Error())
		return
	}

	response := struct {
		SessionID string `json:"session_id"`
		*protocol.CredentialCreation
	}{
		SessionID:          sessionID.String(),
		CredentialCreation: options,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleRegisterFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionHeader := r.Header.Get("X-Session-ID")
	if sessionHeader == "" {
		httpError(w, http.StatusUnauthorized, "missing X-Session-ID header")
		return
	}

	sessionID, err := uuid.Parse(sessionHeader)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid session id")
		return
	}

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

	if err := h.svc.FinishRegistration(r.Context(), sessionID, parsed); err != nil {
		httpError(w, http.StatusUnauthorized, "finish registration failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}
