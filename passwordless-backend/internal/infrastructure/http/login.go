package http

import (
	"errors"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type beginLoginRequest struct {
	Username string `json:"username"`
}

type finishLoginRequest struct {
	Username string `json:"username"`
}

func (h *Handler) handleLoginBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req beginLoginRequest
	if err := readJSON(r, &req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Username == "" {
		httpError(w, http.StatusBadRequest, "username is required")
		return
	}

	assertion, sessionID, err := h.svc.BeginLogin(r.Context(), req.Username, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		httpError(w, http.StatusNotFound, "begin login failed: "+err.Error())
		return
	}

	response := struct {
		SessionID string `json:"session_id"`
		*protocol.CredentialAssertion
	}{
		SessionID:           sessionID.String(),
		CredentialAssertion: assertion,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleLoginFinish(w http.ResponseWriter, r *http.Request) {
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

	parsed, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		var pErr *protocol.Error
		if errors.As(err, &pErr) {
			httpError(w, http.StatusBadRequest, pErr.Error())
			return
		}
		httpError(w, http.StatusBadRequest, "parse assertion response: "+err.Error())
		return
	}

	token, err := h.svc.FinishLogin(r.Context(), sessionID, parsed)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "authentication failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, token)
}
