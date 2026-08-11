package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrConflict        = errors.New("conflict")
	ErrInvalidLogin    = errors.New("invalid login")
	ErrAuthUnavailable = errors.New("auth unavailable")
)

type StoreConfig struct {
	TokenIssuer          *TokenService
	AccessTokenTTL       time.Duration
	RefreshTTL           time.Duration
	MemberRefreshIdleTTL time.Duration
	StaffRefreshIdleTTL  time.Duration
	StepUpTTL            time.Duration
}

type Store struct {
	db     *sql.DB
	cfg    StoreConfig
	now    func() time.Time
	params Argon2idParams
}

type RegisterInput struct {
	Name     string
	Surname  string
	Username string
	Email    string
	Phone    string
	Password string
	CityID   *int64
}

type LoginInput struct {
	Username   string
	Password   string
	UserAgent  string
	DeviceID   string
	ClientID   string
	Platform   string
	AppVersion string
}

type RefreshInput struct {
	RefreshToken string
	DeviceID     string
	ClientID     string
}

type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

type UpdateProfileInput struct {
	Name         *string
	Surname      *string
	Username     *string
	Email        *string
	Phone        *string
	About        *string
	BirthDay     *time.Time
	CityID       *int64
	ShowEvent    *bool
	VocationID   *int64
	ShowCity     *bool
	ShowVocation *bool
	ShowBirthDay *bool
}

type AuthUser struct {
	ID             int64    `json:"id"`
	Subject        string   `json:"-"`
	AuthVersion    int64    `json:"-"`
	Name           string   `json:"name"`
	Surname        string   `json:"surname"`
	FullName       string   `json:"fullName"`
	Username       string   `json:"username"`
	Email          string   `json:"email"`
	Phone          string   `json:"phone"`
	City           string   `json:"city"`
	PictureURL     string   `json:"pictureUrl"`
	Role           string   `json:"role"`
	Roles          []string `json:"roles"`
	AuthorityCodes []string `json:"authorityCodes"`
}

type TokenEnvelope struct {
	Token      string     `json:"token"`
	Expiration *time.Time `json:"expiration,omitempty"`
}

type AuthResponse struct {
	AccessToken  TokenEnvelope `json:"accessToken"`
	RefreshToken TokenEnvelope `json:"refreshToken"`
	User         AuthUser      `json:"user"`
}

type SessionInfo struct {
	SessionID         string     `json:"sessionID"`
	Current           bool       `json:"current"`
	ClientID          string     `json:"clientID"`
	Platform          string     `json:"platform"`
	AppVersion        string     `json:"appVersion"`
	DeviceLabel       string     `json:"deviceLabel"`
	CreatedAt         time.Time  `json:"createdAt"`
	LastUsedAt        time.Time  `json:"lastUsedAt"`
	AuthTime          time.Time  `json:"authTime"`
	AbsoluteExpiresAt time.Time  `json:"absoluteExpiresAt"`
	IdleExpiresAt     time.Time  `json:"idleExpiresAt"`
	LastStepUpAt      *time.Time `json:"lastStepUpAt,omitempty"`
	RevokedAt         *time.Time `json:"revokedAt,omitempty"`
}

func NewStore(db *sql.DB, cfg StoreConfig) *Store {
	return &Store{
		db:     db,
		cfg:    cfg,
		now:    func() time.Time { return time.Now().UTC() },
		params: DefaultArgon2idParams,
	}
}

func (s *Store) TokenIssuer() *TokenService {
	if s == nil {
		return nil
	}
	return s.cfg.TokenIssuer
}

func (s *Store) Register(ctx context.Context, input RegisterInput) (AuthUser, error) {
	if s == nil || s.db == nil {
		return AuthUser{}, ErrAuthUnavailable
	}
	input.normalize()
	if input.Username == "" {
		input.Username = input.Email
	}
	if input.Username == "" || input.Password == "" {
		return AuthUser{}, ErrInvalidLogin
	}
	passwordHash, err := HashPassword(input.Password, s.params)
	if err != nil {
		return AuthUser{}, err
	}
	subject, err := randomSubject()
	if err != nil {
		return AuthUser{}, err
	}
	raw, _ := json.Marshal(map[string]any{"cityId": input.CityID})
	role := "applicant"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AuthUser{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var user AuthUser
	err = tx.QueryRowContext(ctx, `
		INSERT INTO app.users (username, email, phone, name, surname, role, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
		RETURNING id, username, email, phone, name, surname, role
	`, input.Username, input.Email, input.Phone, input.Name, input.Surname, role, string(raw)).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Phone,
		&user.Name,
		&user.Surname,
		&user.Role,
	)
	if err != nil {
		return AuthUser{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO app.auth_accounts (user_id, subject, username, email, password_hash, role, auth_version)
		VALUES ($1, $2, $3, $4, $5, $6, 1)
	`, user.ID, subject, input.Username, input.Email, passwordHash, user.Role)
	if err != nil {
		return AuthUser{}, classifyStoreError(err)
	}
	if err = tx.Commit(); err != nil {
		return AuthUser{}, err
	}
	user.Subject = subject
	user.AuthVersion = 1
	user.finish()
	return user, nil
}

func (s *Store) Login(ctx context.Context, input LoginInput) (AuthResponse, error) {
	if s == nil || s.db == nil || s.cfg.TokenIssuer == nil {
		return AuthResponse{}, ErrAuthUnavailable
	}
	input.Username = strings.TrimSpace(input.Username)
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	input.ClientID = strings.TrimSpace(input.ClientID)
	if input.Username == "" || input.Password == "" || input.DeviceID == "" || input.ClientID == "" {
		return AuthResponse{}, ErrInvalidLogin
	}

	user, passwordHash, err := s.userByLogin(ctx, input.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuthResponse{}, ErrInvalidLogin
		}
		return AuthResponse{}, err
	}
	ok, err := VerifyPassword(input.Password, passwordHash)
	if err != nil || !ok {
		return AuthResponse{}, ErrInvalidLogin
	}
	return s.issueSession(ctx, user, input)
}

func (s *Store) Refresh(ctx context.Context, input RefreshInput) (AuthResponse, error) {
	if s == nil || s.db == nil || s.cfg.TokenIssuer == nil {
		return AuthResponse{}, ErrAuthUnavailable
	}
	input.RefreshToken = strings.TrimSpace(input.RefreshToken)
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	input.ClientID = strings.TrimSpace(input.ClientID)
	if input.RefreshToken == "" || input.DeviceID == "" || input.ClientID == "" {
		return AuthResponse{}, ErrUnauthorized
	}
	tokenHash := hashRefreshToken(input.RefreshToken)
	deviceSignalHash := hashDeviceSignal(input.DeviceID, input.ClientID)
	now := s.now()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AuthResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var sessionID, familyID, tokenClientID, tokenDeviceHash string
	var sessionRevokedAt, tokenRotatedAt, tokenRevokedAt sql.NullTime
	var tokenExpiresAt, tokenIdleExpiresAt, sessionAbsoluteExpiresAt, sessionIdleExpiresAt, authTime time.Time
	var user AuthUser
	err = tx.QueryRowContext(ctx, `
		SELECT rt.session_id, rt.family_id, rt.client_id, rt.device_signal_hash,
		       rt.expires_at, rt.idle_expires_at, rt.rotated_at, rt.revoked_at,
		       s.revoked_at, s.absolute_expires_at, s.idle_expires_at, s.auth_time,
		       u.id, u.username, u.email, u.phone, u.name, u.surname,
		       COALESCE(a.role, u.role), a.subject, a.auth_version
		FROM app.auth_refresh_tokens rt
		JOIN app.auth_sessions s ON s.id = rt.session_id
		JOIN app.users u ON u.id = s.user_id
		LEFT JOIN app.auth_accounts a ON a.user_id = u.id
		WHERE rt.token_hash = $1
		FOR UPDATE OF rt, s
	`, tokenHash).Scan(
		&sessionID,
		&familyID,
		&tokenClientID,
		&tokenDeviceHash,
		&tokenExpiresAt,
		&tokenIdleExpiresAt,
		&tokenRotatedAt,
		&tokenRevokedAt,
		&sessionRevokedAt,
		&sessionAbsoluteExpiresAt,
		&sessionIdleExpiresAt,
		&authTime,
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Phone,
		&user.Name,
		&user.Surname,
		&user.Role,
		&user.Subject,
		&user.AuthVersion,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuthResponse{}, ErrUnauthorized
		}
		return AuthResponse{}, err
	}
	if tokenRotatedAt.Valid || tokenRevokedAt.Valid {
		_ = s.revokeRefreshFamilyTx(ctx, tx, familyID, sessionID, user.ID, "suspected_replay")
		_ = s.insertSecurityEventTx(ctx, tx, user.ID, sessionID, "refresh_reuse_detected", input.ClientID, map[string]any{"familyId": familyID})
		_ = tx.Commit()
		return AuthResponse{}, ErrUnauthorized
	}
	if sessionRevokedAt.Valid || !now.Before(sessionAbsoluteExpiresAt) || !now.Before(sessionIdleExpiresAt) || !now.Before(tokenExpiresAt) || !now.Before(tokenIdleExpiresAt) {
		return AuthResponse{}, ErrUnauthorized
	}
	if tokenClientID != input.ClientID || tokenDeviceHash != deviceSignalHash {
		return AuthResponse{}, ErrUnauthorized
	}
	if err := s.hydrateStaffPermissions(ctx, &user); err != nil {
		return AuthResponse{}, err
	}

	newRefreshToken, err := randomToken()
	if err != nil {
		return AuthResponse{}, err
	}
	newTokenHash := hashRefreshToken(newRefreshToken)
	refreshExpiresAt := sessionAbsoluteExpiresAt
	idleExpiresAt := minTime(now.Add(s.refreshIdleTTL(user)), sessionAbsoluteExpiresAt)
	if _, err = tx.ExecContext(ctx, `
		UPDATE app.auth_refresh_tokens
		SET rotated_at = $2, used_at = $2, replaced_by_token_hash = $3
		WHERE token_hash = $1
	`, tokenHash, now, newTokenHash); err != nil {
		return AuthResponse{}, err
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO app.auth_refresh_tokens (
			session_id, user_id, family_id, token_hash, client_id, device_signal_hash,
			issued_at, expires_at, idle_expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, sessionID, user.ID, familyID, newTokenHash, input.ClientID, deviceSignalHash, now, refreshExpiresAt, idleExpiresAt); err != nil {
		return AuthResponse{}, err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE app.auth_sessions
		SET refresh_token_hash = $1, idle_expires_at = $2, expires_at = $3, last_used_at = $4
		WHERE id = $5
	`, newTokenHash, idleExpiresAt, refreshExpiresAt, now, sessionID)
	if err != nil {
		return AuthResponse{}, err
	}
	if err = tx.Commit(); err != nil {
		return AuthResponse{}, err
	}
	user.finish()
	return s.authResponse(user, sessionID, input.ClientID, authTime, newRefreshToken, refreshExpiresAt)
}

func (s *Store) Logout(ctx context.Context, user User, refreshToken string) error {
	if s == nil || s.db == nil {
		return ErrAuthUnavailable
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if refreshToken != "" {
		var sessionID string
		var userID int64
		var familyID string
		err := tx.QueryRowContext(ctx, `
			SELECT rt.session_id, rt.user_id, rt.family_id
			FROM app.auth_refresh_tokens rt
			WHERE rt.token_hash = $1
		`, hashRefreshToken(refreshToken)).Scan(&sessionID, &userID, &familyID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := s.revokeRefreshFamilyTx(ctx, tx, familyID, sessionID, userID, "logout"); err != nil {
			return err
		}
		_ = s.insertSecurityEventTx(ctx, tx, userID, sessionID, "logout", user.ClientID, nil)
		return tx.Commit()
	}
	if user.SessionID == "" {
		return nil
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE app.auth_sessions
		SET revoked_at = COALESCE(revoked_at, now()),
		    revoked_by_user_id = $2,
		    revoked_reason = 'logout',
		    last_used_at = now()
		WHERE id = $1 AND user_id = $2
	`, user.SessionID, user.ID)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE app.auth_refresh_tokens
		SET revoked_at = COALESCE(revoked_at, now()), revoked_reason = 'logout'
		WHERE session_id = $1 AND revoked_at IS NULL
	`, user.SessionID); err != nil {
		return err
	}
	_ = s.insertSecurityEventTx(ctx, tx, user.ID, user.SessionID, "logout", user.ClientID, nil)
	return tx.Commit()
}

func (s *Store) ValidateSession(ctx context.Context, user User) (User, bool, error) {
	if s == nil || s.db == nil || user.SessionID == "" || user.Subject == "" {
		return User{}, false, nil
	}
	var validated User
	var lastStepUp sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, COALESCE(a.role, u.role), a.subject, s.id, s.client_id,
		       a.auth_version, s.auth_time, s.last_step_up_at
		FROM app.auth_sessions s
		JOIN app.users u ON u.id = s.user_id
		JOIN app.auth_accounts a ON a.user_id = u.id
		WHERE s.id = $1
		  AND a.subject = $2
		  AND a.auth_version = $3
		  AND s.client_id = $4
		  AND s.revoked_at IS NULL
		  AND s.absolute_expires_at > now()
		  AND s.idle_expires_at > now()
	`, user.SessionID, user.Subject, user.AuthVersion, user.ClientID).Scan(
		&validated.ID,
		&validated.Role,
		&validated.Subject,
		&validated.SessionID,
		&validated.ClientID,
		&validated.AuthVersion,
		&validated.AuthTime,
		&lastStepUp,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	validated.TokenID = user.TokenID
	if lastStepUp.Valid {
		value := lastStepUp.Time.UTC()
		validated.LastStepUpAt = &value
	}
	return validated, true, nil
}

func (s *Store) ListSessions(ctx context.Context, user User) ([]SessionInfo, error) {
	if s == nil || s.db == nil || user.ID <= 0 {
		return nil, ErrAuthUnavailable
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, platform, app_version, created_at, last_used_at,
		       auth_time, absolute_expires_at, idle_expires_at, last_step_up_at, revoked_at
		FROM app.auth_sessions
		WHERE user_id = $1
		ORDER BY COALESCE(last_used_at, created_at) DESC, created_at DESC
	`, user.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []SessionInfo
	for rows.Next() {
		var item SessionInfo
		var lastStepUp, revokedAt sql.NullTime
		if err := rows.Scan(
			&item.SessionID,
			&item.ClientID,
			&item.Platform,
			&item.AppVersion,
			&item.CreatedAt,
			&item.LastUsedAt,
			&item.AuthTime,
			&item.AbsoluteExpiresAt,
			&item.IdleExpiresAt,
			&lastStepUp,
			&revokedAt,
		); err != nil {
			return nil, err
		}
		item.Current = item.SessionID == user.SessionID
		item.DeviceLabel = redactedDeviceLabel(item.Platform, item.ClientID)
		if lastStepUp.Valid {
			value := lastStepUp.Time.UTC()
			item.LastStepUpAt = &value
		}
		if revokedAt.Valid {
			value := revokedAt.Time.UTC()
			item.RevokedAt = &value
		}
		sessions = append(sessions, item)
	}
	return sessions, rows.Err()
}

func (s *Store) LogoutAll(ctx context.Context, user User, includeCurrent bool) error {
	if s == nil || s.db == nil || user.ID <= 0 {
		return ErrAuthUnavailable
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	query := `
		UPDATE app.auth_sessions
		SET revoked_at = COALESCE(revoked_at, now()),
		    revoked_by_user_id = $1,
		    revoked_reason = 'logout_all',
		    last_used_at = now()
		WHERE user_id = $1
		  AND revoked_at IS NULL
		  AND ($2 OR id <> $3)
	`
	if _, err = tx.ExecContext(ctx, query, user.ID, includeCurrent, user.SessionID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE app.auth_refresh_tokens rt
		SET revoked_at = COALESCE(revoked_at, now()), revoked_reason = 'logout_all'
		FROM app.auth_sessions s
		WHERE s.id = rt.session_id
		  AND s.user_id = $1
		  AND ($2 OR s.id <> $3)
		  AND rt.revoked_at IS NULL
	`, user.ID, includeCurrent, user.SessionID); err != nil {
		return err
	}
	_ = s.insertSecurityEventTx(ctx, tx, user.ID, user.SessionID, "logout_all", user.ClientID, map[string]any{"includeCurrent": includeCurrent})
	return tx.Commit()
}

func (s *Store) RevokeSession(ctx context.Context, user User, sessionID string) error {
	if s == nil || s.db == nil || user.ID <= 0 {
		return ErrAuthUnavailable
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ErrUnauthorized
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	var exists bool
	if err := tx.QueryRowContext(ctx, `
		UPDATE app.auth_sessions
		SET revoked_at = COALESCE(revoked_at, now()),
		    revoked_by_user_id = $1,
		    revoked_reason = 'selected_revoke',
		    last_used_at = now()
		WHERE id = $2 AND user_id = $1
		RETURNING true
	`, user.ID, sessionID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return ErrUnauthorized
	} else if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE app.auth_refresh_tokens
		SET revoked_at = COALESCE(revoked_at, now()), revoked_reason = 'selected_revoke'
		WHERE session_id = $1 AND revoked_at IS NULL
	`, sessionID); err != nil {
		return err
	}
	_ = s.insertSecurityEventTx(ctx, tx, user.ID, sessionID, "session_revoked", user.ClientID, nil)
	return tx.Commit()
}

func (s *Store) StepUp(ctx context.Context, user User, password string) (time.Time, error) {
	if s == nil || s.db == nil || user.ID <= 0 || user.SessionID == "" {
		return time.Time{}, ErrAuthUnavailable
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return time.Time{}, ErrInvalidLogin
	}
	var passwordHash string
	if err := s.db.QueryRowContext(ctx, `
		SELECT password_hash
		FROM app.auth_accounts
		WHERE user_id = $1
	`, user.ID).Scan(&passwordHash); err != nil {
		return time.Time{}, err
	}
	ok, err := VerifyPassword(password, passwordHash)
	if err != nil || !ok {
		return time.Time{}, ErrInvalidLogin
	}
	now := s.now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return time.Time{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err = tx.ExecContext(ctx, `
		UPDATE app.auth_sessions
		SET last_step_up_at = $1, last_used_at = $1
		WHERE id = $2 AND user_id = $3 AND revoked_at IS NULL
	`, now, user.SessionID, user.ID); err != nil {
		return time.Time{}, err
	}
	_ = s.insertSecurityEventTx(ctx, tx, user.ID, user.SessionID, "step_up", user.ClientID, nil)
	if err = tx.Commit(); err != nil {
		return time.Time{}, err
	}
	return now.Add(s.stepUpTTL()), nil
}

func (s *Store) ChangePassword(ctx context.Context, user User, input ChangePasswordInput) error {
	if s == nil || s.db == nil || user.ID <= 0 || user.SessionID == "" {
		return ErrAuthUnavailable
	}
	if strings.TrimSpace(input.CurrentPassword) == "" || strings.TrimSpace(input.NewPassword) == "" {
		return ErrInvalidLogin
	}
	var passwordHash string
	if err := s.db.QueryRowContext(ctx, `
		SELECT password_hash
		FROM app.auth_accounts
		WHERE user_id = $1
	`, user.ID).Scan(&passwordHash); err != nil {
		return err
	}
	ok, err := VerifyPassword(input.CurrentPassword, passwordHash)
	if err != nil || !ok {
		return ErrInvalidLogin
	}
	newHash, err := HashPassword(input.NewPassword, s.params)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err = tx.ExecContext(ctx, `
		UPDATE app.auth_accounts
		SET password_hash = $2, auth_version = auth_version + 1, updated_at = now()
		WHERE user_id = $1
	`, user.ID, newHash); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE app.auth_sessions
		SET revoked_at = COALESCE(revoked_at, now()),
		    revoked_by_user_id = $1,
		    revoked_reason = 'password_changed',
		    last_used_at = now()
		WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL
	`, user.ID, user.SessionID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE app.auth_refresh_tokens rt
		SET revoked_at = COALESCE(revoked_at, now()), revoked_reason = 'password_changed'
		FROM app.auth_sessions s
		WHERE s.id = rt.session_id
		  AND s.user_id = $1
		  AND s.id <> $2
		  AND rt.revoked_at IS NULL
	`, user.ID, user.SessionID); err != nil {
		return err
	}
	_ = s.insertSecurityEventTx(ctx, tx, user.ID, user.SessionID, "password_changed", user.ClientID, nil)
	return tx.Commit()
}

func (s *Store) UserByID(ctx context.Context, userID int64) (AuthUser, error) {
	if s == nil || s.db == nil {
		return AuthUser{}, ErrAuthUnavailable
	}
	var user AuthUser
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.email, u.phone, u.name, u.surname,
		       COALESCE(c.name, ''), u.picture_url, COALESCE(a.role, u.role)
		FROM app.users u
		LEFT JOIN app.auth_accounts a ON a.user_id = u.id
		LEFT JOIN app.cities c ON c.id = CASE
			WHEN u.raw->>'cityId' ~ '^[0-9]+$' THEN (u.raw->>'cityId')::BIGINT
			ELSE NULL
		END
		WHERE u.id = $1
	`, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Phone,
		&user.Name,
		&user.Surname,
		&user.City,
		&user.PictureURL,
		&user.Role,
	)
	if err != nil {
		return AuthUser{}, err
	}
	user.finish()
	if err := s.hydrateStaffPermissions(ctx, &user); err != nil {
		return AuthUser{}, err
	}
	return user, nil
}

func (s *Store) UpdateProfile(ctx context.Context, userID int64, input UpdateProfileInput) (AuthUser, error) {
	if s == nil || s.db == nil {
		return AuthUser{}, ErrAuthUnavailable
	}
	rawPatch, err := json.Marshal(input.rawPatch())
	if err != nil {
		return AuthUser{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AuthUser{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `
		UPDATE app.users
		SET name = COALESCE(NULLIF($2, ''), name),
		    surname = COALESCE(NULLIF($3, ''), surname),
		    username = COALESCE(NULLIF($4, ''), username),
		    email = COALESCE(NULLIF($5, ''), email),
		    phone = COALESCE(NULLIF($6, ''), phone),
		    raw = raw || $7::jsonb,
		    updated_at = now()
		WHERE id = $1
	`, userID, trimString(input.Name), trimString(input.Surname), trimString(input.Username),
		trimString(input.Email), trimString(input.Phone), string(rawPatch)); err != nil {
		return AuthUser{}, classifyStoreError(err)
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE app.auth_accounts
		SET username = COALESCE(NULLIF($2, ''), username),
		    email = COALESCE(NULLIF($3, ''), email),
		    updated_at = now()
		WHERE user_id = $1
	`, userID, trimString(input.Username), trimString(input.Email)); err != nil {
		return AuthUser{}, classifyStoreError(err)
	}
	if err = tx.Commit(); err != nil {
		return AuthUser{}, classifyStoreError(err)
	}
	return s.UserByID(ctx, userID)
}

func (s *Store) SearchUsers(ctx context.Context, searchText string, excludeUserID int64) ([]AuthUser, error) {
	if s == nil || s.db == nil {
		return nil, ErrAuthUnavailable
	}
	searchText = strings.TrimSpace(searchText)
	if searchText == "" {
		return []AuthUser{}, nil
	}
	pattern := "%" + strings.ToLower(searchText) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.email, u.phone, u.name, u.surname,
		       COALESCE(c.name, ''), u.picture_url, COALESCE(a.role, u.role)
		FROM app.users u
		LEFT JOIN app.auth_accounts a ON a.user_id = u.id
		LEFT JOIN app.cities c ON c.id = CASE
			WHEN u.raw->>'cityId' ~ '^[0-9]+$' THEN (u.raw->>'cityId')::BIGINT
			ELSE NULL
		END
		WHERE u.id <> $2
		  AND (
			lower(u.username) LIKE $1
			OR lower(u.name) LIKE $1
			OR lower(u.surname) LIKE $1
			OR lower(concat_ws(' ', u.name, u.surname)) LIKE $1
			OR lower(u.email) LIKE $1
		  )
		ORDER BY u.name, u.surname, u.username, u.id
		LIMIT 20
	`, pattern, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []AuthUser
	for rows.Next() {
		var user AuthUser
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Phone,
			&user.Name,
			&user.Surname,
			&user.City,
			&user.PictureURL,
			&user.Role,
		); err != nil {
			return nil, err
		}
		user.finish()
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) userByLogin(ctx context.Context, login string) (AuthUser, string, error) {
	var user AuthUser
	var passwordHash string
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.email, u.phone, u.name, u.surname, a.role,
		       a.password_hash, a.subject, a.auth_version
		FROM app.auth_accounts a
		JOIN app.users u ON u.id = a.user_id
		WHERE lower(a.username) = lower($1) OR lower(a.email) = lower($1)
	`, login).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Phone,
		&user.Name,
		&user.Surname,
		&user.Role,
		&passwordHash,
		&user.Subject,
		&user.AuthVersion,
	)
	user.finish()
	if err == nil {
		err = s.hydrateStaffPermissions(ctx, &user)
	}
	return user, passwordHash, err
}

func (s *Store) hydrateStaffPermissions(ctx context.Context, user *AuthUser) error {
	if s == nil || s.db == nil || user == nil || user.ID <= 0 {
		return nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT permission
		FROM app.staff_permissions
		WHERE user_id = $1
		  AND active = true
		  AND revoked_at IS NULL
		ORDER BY permission
	`, user.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	codes := make([]string, 0)
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return err
		}
		permission = strings.TrimSpace(permission)
		if permission != "" {
			codes = append(codes, permission)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	user.AuthorityCodes = codes
	return nil
}

func (s *Store) issueSession(ctx context.Context, user AuthUser, input LoginInput) (AuthResponse, error) {
	sessionID, err := randomToken()
	if err != nil {
		return AuthResponse{}, err
	}
	refreshToken, err := randomToken()
	if err != nil {
		return AuthResponse{}, err
	}
	familyID, err := randomToken()
	if err != nil {
		return AuthResponse{}, err
	}
	now := s.now()
	authTime := now
	refreshExpiresAt := now.Add(s.refreshTTL())
	idleExpiresAt := minTime(now.Add(s.refreshIdleTTL(user)), refreshExpiresAt)
	refreshTokenHash := hashRefreshToken(refreshToken)
	deviceSignalHash := hashDeviceSignal(input.DeviceID, input.ClientID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AuthResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO app.auth_sessions (
			id, user_id, refresh_token_hash, user_agent, device_id, client_id,
			platform, app_version, device_signal_hash, auth_time,
			absolute_expires_at, idle_expires_at, expires_at
		)
		VALUES ($1, $2, $3, $4, '', $5, $6, $7, $8, $9, $10, $11, $10)
	`, sessionID, user.ID, refreshTokenHash, input.UserAgent, input.ClientID,
		strings.TrimSpace(input.Platform), strings.TrimSpace(input.AppVersion), deviceSignalHash,
		authTime, refreshExpiresAt, idleExpiresAt)
	if err != nil {
		return AuthResponse{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO app.auth_refresh_tokens (
			session_id, user_id, family_id, token_hash, client_id, device_signal_hash,
			issued_at, expires_at, idle_expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, sessionID, user.ID, familyID, refreshTokenHash, input.ClientID, deviceSignalHash, now, refreshExpiresAt, idleExpiresAt)
	if err != nil {
		return AuthResponse{}, err
	}
	_ = s.insertSecurityEventTx(ctx, tx, user.ID, sessionID, "login", input.ClientID, map[string]any{
		"platform":   strings.TrimSpace(input.Platform),
		"appVersion": strings.TrimSpace(input.AppVersion),
	})
	if err = tx.Commit(); err != nil {
		return AuthResponse{}, err
	}
	return s.authResponse(user, sessionID, input.ClientID, authTime, refreshToken, refreshExpiresAt)
}

func (s *Store) authResponse(user AuthUser, sessionID, clientID string, authTime time.Time, refreshToken string, refreshExpiresAt time.Time) (AuthResponse, error) {
	access, err := s.cfg.TokenIssuer.IssueAccessToken(AccessTokenInput{
		Subject:     user.Subject,
		SessionID:   sessionID,
		ClientID:    clientID,
		Role:        user.Role,
		AuthTime:    authTime,
		AuthVersion: user.AuthVersion,
	}, s.accessTTL(), s.now())
	if err != nil {
		return AuthResponse{}, err
	}
	return AuthResponse{
		AccessToken: TokenEnvelope{
			Token:      access.Token,
			Expiration: &access.ExpiresAt,
		},
		RefreshToken: TokenEnvelope{
			Token:      refreshToken,
			Expiration: &refreshExpiresAt,
		},
		User: user,
	}, nil
}

func (s *Store) accessTTL() time.Duration {
	if s.cfg.AccessTokenTTL > 0 {
		return s.cfg.AccessTokenTTL
	}
	return 15 * time.Minute
}

func (s *Store) refreshTTL() time.Duration {
	if s.cfg.RefreshTTL > 0 {
		return s.cfg.RefreshTTL
	}
	return 30 * 24 * time.Hour
}

func (s *Store) refreshIdleTTL(user AuthUser) time.Duration {
	if isStaffRole(user.Role) {
		if s.cfg.StaffRefreshIdleTTL > 0 {
			return s.cfg.StaffRefreshIdleTTL
		}
		return 12 * time.Hour
	}
	if s.cfg.MemberRefreshIdleTTL > 0 {
		return s.cfg.MemberRefreshIdleTTL
	}
	return 14 * 24 * time.Hour
}

func (s *Store) stepUpTTL() time.Duration {
	if s.cfg.StepUpTTL > 0 {
		return s.cfg.StepUpTTL
	}
	return 10 * time.Minute
}

func (input *RegisterInput) normalize() {
	input.Name = strings.TrimSpace(input.Name)
	input.Surname = strings.TrimSpace(input.Surname)
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
}

func (input UpdateProfileInput) rawPatch() map[string]any {
	patch := map[string]any{}
	if input.About != nil {
		patch["about"] = strings.TrimSpace(*input.About)
	}
	if input.BirthDay != nil {
		patch["birthDay"] = input.BirthDay.UTC().Format(time.RFC3339)
	}
	if input.CityID != nil {
		patch["cityId"] = *input.CityID
	}
	if input.ShowEvent != nil {
		patch["showEvent"] = *input.ShowEvent
	}
	if input.VocationID != nil {
		patch["vocationId"] = *input.VocationID
	}
	if input.ShowCity != nil {
		patch["showCity"] = *input.ShowCity
	}
	if input.ShowVocation != nil {
		patch["showVocation"] = *input.ShowVocation
	}
	if input.ShowBirthDay != nil {
		patch["showBirthDay"] = *input.ShowBirthDay
	}
	return patch
}

func trimString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func (user *AuthUser) finish() {
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.TrimSpace(user.Email)
	user.Phone = strings.TrimSpace(user.Phone)
	user.Name = strings.TrimSpace(user.Name)
	user.Surname = strings.TrimSpace(user.Surname)
	user.City = strings.TrimSpace(user.City)
	user.PictureURL = strings.TrimSpace(user.PictureURL)
	user.Role = strings.TrimSpace(user.Role)
	if user.Role == "" {
		user.Role = "member"
	}
	if user.AuthVersion <= 0 {
		user.AuthVersion = 1
	}
	if len(user.Roles) == 0 {
		user.Roles = []string{user.Role}
	}
	user.FullName = strings.TrimSpace(strings.Join([]string{user.Name, user.Surname}, " "))
	if user.FullName == "" {
		user.FullName = user.Username
	}
	if user.AuthorityCodes == nil {
		user.AuthorityCodes = []string{}
	}
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func hashDeviceSignal(deviceID, clientID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(deviceID) + "\x00" + strings.TrimSpace(clientID)))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomSubject() (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	return "usr_" + token, nil
}

func randomToken() (string, error) {
	var data [32]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data[:]), nil
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func isStaffRole(role string) bool {
	normalized := strings.ToLower(strings.TrimSpace(role))
	return normalized != "" && normalized != "member" && normalized != "community"
}

func redactedDeviceLabel(platform, clientID string) string {
	platform = strings.TrimSpace(platform)
	clientID = strings.TrimSpace(clientID)
	switch {
	case platform != "" && clientID != "":
		return platform + " (" + clientID + ")"
	case platform != "":
		return platform
	case clientID != "":
		return clientID
	default:
		return "Unknown device"
	}
}

func (s *Store) revokeRefreshFamilyTx(ctx context.Context, tx *sql.Tx, familyID, sessionID string, userID int64, reason string) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE app.auth_sessions
		SET revoked_at = COALESCE(revoked_at, now()),
		    revoked_by_user_id = NULLIF($2, 0),
		    revoked_reason = $3,
		    last_used_at = now()
		WHERE id = $1
	`, sessionID, userID, strings.TrimSpace(reason)); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE app.auth_refresh_tokens
		SET revoked_at = COALESCE(revoked_at, now()), revoked_reason = $2
		WHERE family_id = $1 AND revoked_at IS NULL
	`, familyID, strings.TrimSpace(reason))
	return err
}

func (s *Store) insertSecurityEventTx(ctx context.Context, tx *sql.Tx, userID int64, sessionID, eventType, clientID string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	var nullableUserID any
	if userID > 0 {
		nullableUserID = userID
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO app.auth_security_events (user_id, session_id, event_type, client_id, metadata)
		VALUES ($1, NULLIF($2, ''), $3, NULLIF($4, ''), $5::jsonb)
	`, nullableUserID, strings.TrimSpace(sessionID), strings.TrimSpace(eventType), strings.TrimSpace(clientID), string(raw))
	return err
}

func classifyStoreError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "duplicate") || strings.Contains(message, "unique") {
		return ErrConflict
	}
	return err
}
