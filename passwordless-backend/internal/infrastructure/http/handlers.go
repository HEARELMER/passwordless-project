package http

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"passwordless-backend/internal/application"
)

const maxBodyBytes = 1 << 20 // 1 MB

type Handler struct {
	svc        *application.WebAuthnAppService
	jwtSecret  []byte
}

func NewHandler(svc *application.WebAuthnAppService, jwtSecret []byte) *Handler {
	return &Handler{svc: svc, jwtSecret: jwtSecret}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	auth := AuthMiddleware(h.jwtSecret)

	mux.Handle("/api/webauthn/register/begin", SecurityHeadersMiddleware(CORSMiddleware(RateLimitMiddleware(http.HandlerFunc(h.handleRegisterBegin)))))
	mux.Handle("/api/webauthn/register/finish", SecurityHeadersMiddleware(CORSMiddleware(RateLimitMiddleware(http.HandlerFunc(h.handleRegisterFinish)))))
	mux.Handle("/api/webauthn/login/begin", SecurityHeadersMiddleware(CORSMiddleware(RateLimitMiddleware(http.HandlerFunc(h.handleLoginBegin)))))
	mux.Handle("/api/webauthn/login/finish", SecurityHeadersMiddleware(CORSMiddleware(RateLimitMiddleware(http.HandlerFunc(h.handleLoginFinish)))))

	mux.Handle("/api/me", SecurityHeadersMiddleware(CORSMiddleware(auth(http.HandlerFunc(h.handleMe)))))
	mux.Handle("/api/credentials", SecurityHeadersMiddleware(CORSMiddleware(auth(http.HandlerFunc(h.handleListCredentials)))))
	mux.Handle("/api/credentials/", SecurityHeadersMiddleware(CORSMiddleware(auth(http.HandlerFunc(h.handleRevokeCredential)))))

	mux.Handle("/health", http.HandlerFunc(h.handleHealth))
	
	// Digital Asset Links para Android Passkeys
	mux.Handle("/.well-known/assetlinks.json", http.HandlerFunc(h.handleAssetLinks))
}

func (h *Handler) handleAssetLinks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`[{
  "relation": ["delegate_permission/common.handle_all_urls", "delegate_permission/common.get_login_creds"],
  "target": {
    "namespace": "android_app",
    "package_name": "com.passwordless.authenticator",
    "sha256_cert_fingerprints": [
      "5B:50:D4:9F:C6:C6:04:2E:AF:A1:7C:0D:98:1D:05:E7:E9:45:E4:50:BC:57:2D:AF:CF:DD:0C:C0:2C:4C:27:EE"
    ]
  }
}]`))
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

func readJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
