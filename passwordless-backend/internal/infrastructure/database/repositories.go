package database

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"passwordless-backend/internal/domain/entities"
	"passwordless-backend/internal/domain/ports"
	shareddb "passwordless-backend/internal/shared/db"
)

var _ ports.UserRepository = (*UserRepository)(nil)
var _ ports.CredentialRepository = (*CredentialRepository)(nil)
var _ ports.SessionRepository = (*SessionRepository)(nil)

type UserRepository struct {
	pool *pgxpool.Pool
}

type CredentialRepository struct {
	pool *pgxpool.Pool
}

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func NewCredentialRepository(pool *pgxpool.Pool) *CredentialRepository {
	return &CredentialRepository{pool: pool}
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *UserRepository) CreateUser(ctx context.Context, username string, email string, preferences map[string]any) (*entities.User, error) {
	prefsJSON, err := encodeJSON(preferences)
	if err != nil {
		return nil, err
	}

	query, args, err := usersTable.Insert(map[string]any{
		"username":    username,
		"email":       email,
		"preferences": prefsJSON,
	}, []string{"id", "username", "email", "preferences", "created_at", "updated_at"})
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, query, args...)
	return scanUser(row)
}

func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*entities.User, error) {
	query, args, err := usersTable.SelectOne(
		[]string{"id", "username", "email", "preferences", "created_at", "updated_at"},
		map[string]any{"username": username},
	)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, query, args...)
	return scanUser(row)
}

func (r *UserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	query, args, err := usersTable.SelectOne(
		[]string{"id", "username", "email", "preferences", "created_at", "updated_at"},
		map[string]any{"id": id},
	)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, query, args...)
	return scanUser(row)
}

func (r *CredentialRepository) SaveCredential(ctx context.Context, cred *entities.Credential) error {
	clientInfoJSON, err := encodeJSON(cred.ClientInfo)
	if err != nil {
		return err
	}

	rawJSON, err := encodeOptionalJSON(cred.RawRegistrationData)
	if err != nil {
		return err
	}

	query, args, err := credentialsTable.Insert(map[string]any{
		"id":                  cred.ID,
		"user_id":             cred.UserID,
		"public_key":          cred.PublicKey,
		"attestation_type":    cred.AttestationType,
		"aaguid":              cred.AAGUID,
		"sign_count":          cred.SignCount,
		"last_used_at":         cred.LastUsedAt,
		"last_login_ip":        nullIP(cred.LastLoginIP),
		"client_info":          clientInfoJSON,
		"raw_registration_data": rawJSON,
	}, nil)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func (r *CredentialRepository) GetCredentialsByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Credential, error) {
	query, args, err := credentialsTable.SelectMany(
		[]string{
			"id",
			"user_id",
			"public_key",
			"attestation_type",
			"aaguid",
			"sign_count",
			"last_used_at",
			"last_login_ip",
			"client_info",
			"raw_registration_data",
			"created_at",
		},
		map[string]any{"user_id": userID},
		nil,
	)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*entities.Credential
	for rows.Next() {
		cred, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, cred)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return results, nil
}

func (r *CredentialRepository) GetCredentialsByUserIDPaginated(ctx context.Context, userID uuid.UUID, limit int, offset int) ([]*entities.Credential, error) {
	opts := buildSelectOptions(limit, offset, nil)
	query, args, err := credentialsTable.SelectMany(
		[]string{
			"id",
			"user_id",
			"public_key",
			"attestation_type",
			"aaguid",
			"sign_count",
			"last_used_at",
			"last_login_ip",
			"client_info",
			"raw_registration_data",
			"created_at",
		},
		map[string]any{"user_id": userID},
		opts,
	)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*entities.Credential
	for rows.Next() {
		cred, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, cred)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return results, nil
}

func (r *CredentialRepository) GetCredentialsByUserIDFiltered(ctx context.Context, userID uuid.UUID, filters []ports.Filter, opts *ports.ListOptions) ([]*entities.Credential, error) {
	sharedFilters, err := mapFilters(userID, filters)
	if err != nil {
		return nil, err
	}

	sharedOpts, err := mapListOptions(opts)
	if err != nil {
		return nil, err
	}

	query, args, err := credentialsTable.SelectManyWithFilters(
		[]string{
			"id",
			"user_id",
			"public_key",
			"attestation_type",
			"aaguid",
			"sign_count",
			"last_used_at",
			"last_login_ip",
			"client_info",
			"raw_registration_data",
			"created_at",
		},
		sharedFilters,
		sharedOpts,
	)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*entities.Credential
	for rows.Next() {
		cred, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, cred)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return results, nil
}

func (r *CredentialRepository) GetCredentialByID(ctx context.Context, credentialID []byte) (*entities.Credential, error) {
	query, args, err := credentialsTable.SelectOne(
		[]string{
			"id",
			"user_id",
			"public_key",
			"attestation_type",
			"aaguid",
			"sign_count",
			"last_used_at",
			"last_login_ip",
			"client_info",
			"raw_registration_data",
			"created_at",
		},
		map[string]any{"id": credentialID},
	)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, query, args...)
	return scanCredential(row)
}

func (r *CredentialRepository) UpdateSignCount(ctx context.Context, credentialID []byte, newCount uint32) error {
	query, args, err := credentialsTable.Update(
		map[string]any{"sign_count": newCount},
		map[string]any{"id": credentialID},
	)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func (r *CredentialRepository) UpdateCredentialAudit(ctx context.Context, credentialID []byte, lastLoginIP string, clientInfo map[string]any, lastUsedAt *time.Time) error {
	clientInfoJSON, err := encodeJSON(clientInfo)
	if err != nil {
		return err
	}

	query, args, err := credentialsTable.Update(
		map[string]any{
			"last_login_ip": nullIP(lastLoginIP),
			"client_info":   clientInfoJSON,
			"last_used_at":  lastUsedAt,
		},
		map[string]any{"id": credentialID},
	)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func (r *SessionRepository) SaveSession(ctx context.Context, session *entities.WebAuthnSession) error {
	sessionJSON, err := encodeJSON(session.SessionData)
	if err != nil {
		return err
	}

	query, args, err := webauthnSessionsTable.Insert(map[string]any{
		"id":           session.ID,
		"user_id":      session.UserID,
		"challenge":    session.Challenge,
		"type":         session.Type,
		"session_data": sessionJSON,
		"expires_at":   session.ExpiresAt,
	}, nil)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func (r *SessionRepository) GetSessionByUserIDAndType(ctx context.Context, userID uuid.UUID, sessionType string) (*entities.WebAuthnSession, error) {
	query, args, err := webauthnSessionsTable.SelectOne(
		[]string{"id", "user_id", "challenge", "type", "session_data", "expires_at"},
		map[string]any{"user_id": userID, "type": sessionType},
	)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, query, args...)
	return scanSession(row)
}

func (r *SessionRepository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	query, args, err := webauthnSessionsTable.Delete(map[string]any{"id": id})
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func scanUser(row pgx.Row) (*entities.User, error) {
	var user entities.User
	var prefsRaw []byte

	if err := row.Scan(&user.ID, &user.Username, &user.Email, &prefsRaw, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}

	prefs, err := decodeJSON(prefsRaw)
	if err != nil {
		return nil, err
	}
	user.Preferences = prefs
	return &user, nil
}

func scanCredential(row pgx.Row) (*entities.Credential, error) {
	var cred entities.Credential
	var signCount int64
	// pgx/v5 mapea el tipo INET de PostgreSQL a netip.Addr, NO a net.IP.
	// Usar net.IP aquí causa un error de tipo en runtime al hacer Scan.
	var lastLoginIP netip.Addr
	var clientInfoRaw []byte
	var rawRegistrationRaw []byte

	if err := row.Scan(
		&cred.ID,
		&cred.UserID,
		&cred.PublicKey,
		&cred.AttestationType,
		&cred.AAGUID,
		&signCount,
		&cred.LastUsedAt,
		&lastLoginIP,
		&clientInfoRaw,
		&rawRegistrationRaw,
		&cred.CreatedAt,
	); err != nil {
		return nil, err
	}

	if signCount < 0 {
		return nil, errors.New("invalid sign_count")
	}
	cred.SignCount = uint32(signCount)
	// IsValid() es false cuando el campo NULL de la BD devuelve la zero-value de netip.Addr.
	if lastLoginIP.IsValid() {
		cred.LastLoginIP = lastLoginIP.String()
	}

	clientInfo, err := decodeJSON(clientInfoRaw)
	if err != nil {
		return nil, err
	}
	cred.ClientInfo = clientInfo

	rawRegistration, err := decodeJSON(rawRegistrationRaw)
	if err != nil {
		return nil, err
	}
	cred.RawRegistrationData = rawRegistration

	return &cred, nil
}

func scanSession(row pgx.Row) (*entities.WebAuthnSession, error) {
	var session entities.WebAuthnSession
	var sessionRaw []byte

	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.Challenge,
		&session.Type,
		&sessionRaw,
		&session.ExpiresAt,
	); err != nil {
		return nil, err
	}

	sessionData, err := decodeJSON(sessionRaw)
	if err != nil {
		return nil, err
	}
	session.SessionData = sessionData
	return &session, nil
}

func encodeJSON(value map[string]any) ([]byte, error) {
	if value == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(value)
}

func encodeOptionalJSON(value map[string]any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

func decodeJSON(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return map[string]any{}, nil
	}

	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	if value == nil {
		value = map[string]any{}
	}
	return value, nil
}

// nullIP convierte un string de IP a netip.Addr para insertar en una columna INET.
// pgx/v5 acepta netip.Addr como tipo nativo para INET; devuelve zero-value si el
// string está vacío, lo que pgx interpreta como NULL cuando el campo es nullable.
func nullIP(value string) *netip.Addr {
	if value == "" {
		return nil
	}
	addr, err := netip.ParseAddr(value)
	if err != nil {
		return nil
	}
	return &addr
}

func buildSelectOptions(limit int, offset int, orderBy []shareddb.OrderBy) *shareddb.SelectOptions {
	if limit < 0 && offset < 0 && len(orderBy) == 0 {
		return nil
	}

	opts := &shareddb.SelectOptions{}
	if limit >= 0 {
		limitCopy := limit
		opts.Limit = &limitCopy
	}
	if offset >= 0 {
		offsetCopy := offset
		opts.Offset = &offsetCopy
	}
	if len(orderBy) > 0 {
		opts.OrderBy = orderBy
	}
	return opts
}

func mapListOptions(opts *ports.ListOptions) (*shareddb.SelectOptions, error) {
	if opts == nil {
		return nil, nil
	}

	shared := &shareddb.SelectOptions{}
	if opts.Limit != nil {
		limit := *opts.Limit
		shared.Limit = &limit
	}
	if opts.Offset != nil {
		offset := *opts.Offset
		shared.Offset = &offset
	}
	if len(opts.OrderBy) > 0 {
		orderBy, err := mapOrderBy(opts.OrderBy)
		if err != nil {
			return nil, err
		}
		shared.OrderBy = orderBy
	}

	return shared, nil
}

func mapOrderBy(order []ports.OrderBy) ([]shareddb.OrderBy, error) {
	result := make([]shareddb.OrderBy, len(order))
	for i, item := range order {
		direction, err := mapDirection(item.Direction)
		if err != nil {
			return nil, err
		}
		result[i] = shareddb.OrderBy{Column: item.Column, Direction: direction}
	}
	return result, nil
}

func mapDirection(direction ports.OrderDirection) (shareddb.OrderDirection, error) {
	switch direction {
	case ports.OrderAsc:
		return shareddb.OrderAsc, nil
	case ports.OrderDesc:
		return shareddb.OrderDesc, nil
	default:
		return "", errors.New("invalid order direction")
	}
}

func mapFilters(userID uuid.UUID, filters []ports.Filter) ([]shareddb.Filter, error) {
	result := make([]shareddb.Filter, 0, len(filters)+1)
	result = append(result, shareddb.Filter{Column: "user_id", Op: shareddb.OpEq, Value: userID})

	for _, filter := range filters {
		op, err := mapOperator(filter.Op)
		if err != nil {
			return nil, err
		}
		result = append(result, shareddb.Filter{Column: filter.Column, Op: op, Value: filter.Value})
	}

	return result, nil
}

func mapOperator(op ports.Operator) (shareddb.Operator, error) {
	switch op {
	case ports.OpEq:
		return shareddb.OpEq, nil
	case ports.OpNeq:
		return shareddb.OpNeq, nil
	case ports.OpGt:
		return shareddb.OpGt, nil
	case ports.OpGte:
		return shareddb.OpGte, nil
	case ports.OpLt:
		return shareddb.OpLt, nil
	case ports.OpLte:
		return shareddb.OpLte, nil
	case ports.OpIn:
		return shareddb.OpIn, nil
	default:
		return "", errors.New("invalid filter operator")
	}
}
