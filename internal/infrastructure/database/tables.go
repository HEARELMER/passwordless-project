package database

import "passwordless-backend/internal/shared/db"

var (
	usersTable = shareddb.NewTable("users", []string{
		"id",
		"username",
		"email",
		"preferences",
		"created_at",
		"updated_at",
	})

	credentialsTable = shareddb.NewTable("credentials", []string{
		"id",
		"user_id",
		"public_key",
		"attestation_type",
		"aaguid",
		"sign_count",
		"is_active",
		"last_used_at",
		"last_login_ip",
		"client_info",
		"raw_registration_data",
		"created_at",
	})

	webauthnSessionsTable = shareddb.NewTable("webauthn_sessions", []string{
		"id",
		"user_id",
		"challenge",
		"type",
		"status",
		"session_data",
		"user_agent",
		"ip_address",
		"expires_at",
	})
)
