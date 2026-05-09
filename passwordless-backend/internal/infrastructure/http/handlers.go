package http

import (
	"net/http"

	"passwordless-backend/internal/application"
)

type Handler struct {
	svc *application.WebAuthnAppService
}

func NewHandler(svc *application.WebAuthnAppService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/webauthn/register/begin", h.handleRegisterBegin)
	mux.HandleFunc("/api/webauthn/register/finish", h.handleRegisterFinish)
	mux.HandleFunc("/api/webauthn/login/begin", h.handleLoginBegin)
	mux.HandleFunc("/api/webauthn/login/finish", h.handleLoginFinish)
}

func (h *Handler) handleRegisterBegin(w http.ResponseWriter, r *http.Request) {
	h.notImplemented(w)
}

func (h *Handler) handleRegisterFinish(w http.ResponseWriter, r *http.Request) {
	h.notImplemented(w)
}

func (h *Handler) handleLoginBegin(w http.ResponseWriter, r *http.Request) {
	h.notImplemented(w)
}

func (h *Handler) handleLoginFinish(w http.ResponseWriter, r *http.Request) {
	h.notImplemented(w)
}

func (h *Handler) notImplemented(w http.ResponseWriter) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
