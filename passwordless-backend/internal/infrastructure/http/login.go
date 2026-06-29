package http

import (
	"errors"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
)

type beginLoginRequest struct {
	Username string `json:"username"`
}

// handleLoginBegin — Paso A del flujo de login (lo que describió el usuario).
// La app envía: POST /api/webauthn/login/begin  { "username": "alice" }
// El servidor responde inmediatamente con PublicKeyCredentialRequestOptions
// que contiene el challenge aleatorio.  (Paso B)
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

	// BeginLogin:
	// 1. Carga al usuario y sus credenciales de la BD.
	// 2. Genera un challenge aleatorio criptográficamente seguro.
	// 3. Guarda la sesión temporal en webauthn_sessions.
	// 4. Retorna las opciones de aserción que la app mostrará en pantalla.
	assertion, err := h.svc.BeginLogin(r.Context(), req.Username)
	if err != nil {
		httpError(w, http.StatusNotFound, "begin login failed: "+err.Error())
		return
	}

	// Paso B: respuesta inmediata con el challenge.
	// La app lo recibe en milisegundos, lo guarda en RAM y lanza BiometricPrompt.
	writeJSON(w, http.StatusOK, assertion)
}

// handleLoginFinish — Paso C del flujo de login (tras la biometría).
// La app envía la firma digital generada por el Enclave Seguro sobre el challenge.
// El servidor verifica la firma criptográficamente y, si es válida, emite un JWT.
func (h *Handler) handleLoginFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		httpError(w, http.StatusBadRequest, "username query param is required")
		return
	}

	// ParseCredentialRequestResponseBody valida el formato CBOR/JSON de la firma
	// antes de que el servicio haga la verificación criptográfica ECDSA.
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

	token, err := h.svc.FinishLogin(r.Context(), username, parsed)
	if err != nil {
		// Firma inválida → 401 Unauthorized.
		httpError(w, http.StatusUnauthorized, "authentication failed: "+err.Error())
		return
	}

	// Acceso concedido. La app recibe el JWT y lo almacena de forma segura.
	writeJSON(w, http.StatusOK, token)
}
