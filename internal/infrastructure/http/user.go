package http

import (
	"encoding/base64"
	"net/http"
	"strings"
)

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r.Context())
	if !ok {
		httpError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.svc.GetUserByID(r.Context(), userID)
	if err != nil {
		httpError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}

func (h *Handler) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r.Context())
	if !ok {
		httpError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	creds, err := h.svc.GetUserCredentials(r.Context(), userID)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to load credentials")
		return
	}

	type credentialResponse struct {
		ID              string `json:"id"`
		AttestationType string `json:"attestation_type"`
		SignCount       uint32 `json:"sign_count"`
		IsActive        bool   `json:"is_active"`
		LastUsedAt      any    `json:"last_used_at"`
		LastLoginIP     string `json:"last_login_ip"`
		ClientInfo      any    `json:"client_info"`
		CreatedAt       any    `json:"created_at"`
	}

	result := make([]credentialResponse, 0, len(creds))
	for _, c := range creds {
		result = append(result, credentialResponse{
			ID:              base64.RawURLEncoding.EncodeToString(c.ID),
			AttestationType: c.AttestationType,
			SignCount:       c.SignCount,
			IsActive:        c.IsActive,
			LastUsedAt:      c.LastUsedAt,
			LastLoginIP:     c.LastLoginIP,
			ClientInfo:      c.ClientInfo,
			CreatedAt:       c.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleRevokeCredential(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r.Context())
	if !ok {
		httpError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rawID := strings.TrimPrefix(r.URL.Path, "/api/credentials/")
	if rawID == "" {
		httpError(w, http.StatusBadRequest, "credential id is required")
		return
	}

	credentialID, err := base64.RawURLEncoding.DecodeString(rawID)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid credential id format")
		return
	}

	if err := h.svc.RevokeCredential(r.Context(), userID, credentialID); err != nil {
		if err.Error() == "forbidden: credential does not belong to user" {
			httpError(w, http.StatusForbidden, err.Error())
			return
		}
		httpError(w, http.StatusInternalServerError, "failed to revoke credential")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
