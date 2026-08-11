package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	apperrors "kindred_server/internal/platform/errors"
	"kindred_server/internal/user"
	"kindred_server/pkg/mailer"
	"kindred_server/pkg/security"
	"kindred_server/pkg/sms"
)

var (
	ErrInvalidToken       = apperrors.Unauthorized("invalid or missing token")
	ErrInvalidCredentials = apperrors.Unauthorized("email or password is incorrect")
	ErrAccountLocked      = apperrors.Unauthorized("account temporarily locked due to too many failed login attempts")
	ErrEmailNotVerified   = apperrors.Unauthorized("email address is not verified")
)

// AccountLockedError is returned by Login when the account is currently
// locked due to too many failed attempts. It carries the remaining lockout
// duration so the handler can emit a Retry-After header. The wrapped
// AppError preserves the existing wire-level code/message/status.
type AccountLockedError struct {
	RetryAfter time.Duration
	*apperrors.AppError
}

func (e *AccountLockedError) Error() string { return e.AppError.Error() }
func (e *AccountLockedError) Unwrap() error { return e.AppError }

// Config controls runtime behavior of the auth service.
type Config struct {
	TokenTTL                    time.Duration
	RefreshTokenIdleTTL         time.Duration
	RefreshTokenAbsoluteTTL     time.Duration
	VerificationTokenTTL        time.Duration
	PasswordResetTokenTTL       time.Duration
	PhoneVerificationTTL        time.Duration
	ReturnPhoneVerificationCode bool
	MaxFailedLoginAttempts      int
	LockoutDuration             time.Duration
	RequireVerifiedEmail        bool
}

type Service struct {
	repo      user.Repository
	mailer    mailer.Mailer
	sms       sms.Sender
	cfg       Config
	userSvc   *user.Service
	signer    *Signer
	verifier  *Verifier
	consents  ConsentRecorder
	lifecycle AccountLifecycle
}

type ConsentRecorder interface {
	SetInitialConsents(ctx context.Context, userID string, consents map[string]bool) error
}

type PhoneVerificationCommitter interface {
	VerifyPhoneAndGrantSignupBonus(ctx context.Context, u user.User, normalizedPhone string, expectedCodeHash string, bonusPoints int) (bool, error)
}

type PhoneChangeCommitter interface {
	CommitPhoneChange(ctx context.Context, u user.User, oldPhone string, normalizedPhone string, expectedCodeHash string) error
}

type AccountLifecycle interface {
	CloseAccount(ctx context.Context, userID string, now time.Time) error
}

type Option func(*Service)

func WithConsentRecorder(recorder ConsentRecorder) Option {
	return func(s *Service) {
		s.consents = recorder
	}
}

func WithAccountLifecycle(lifecycle AccountLifecycle) Option {
	return func(s *Service) {
		s.lifecycle = lifecycle
	}
}

const signupBonusPoints = 40

var consentPurposes = []string{
	"analytics",
	"personalization",
	"marketing",
	"precise_location",
	"heatmap",
	"messaging_metadata",
}

func NewService(repo user.Repository, m mailer.Mailer, smsSender sms.Sender, cfg Config, signer *Signer, verifier *Verifier, opts ...Option) *Service {
	if cfg.VerificationTokenTTL == 0 {
		cfg.VerificationTokenTTL = 24 * time.Hour
	}
	if cfg.PasswordResetTokenTTL == 0 {
		cfg.PasswordResetTokenTTL = time.Hour
	}
	if cfg.PhoneVerificationTTL == 0 {
		cfg.PhoneVerificationTTL = 10 * time.Minute
	}
	if cfg.RefreshTokenIdleTTL == 0 {
		cfg.RefreshTokenIdleTTL = 30 * 24 * time.Hour
	}
	if cfg.RefreshTokenAbsoluteTTL == 0 {
		cfg.RefreshTokenAbsoluteTTL = 90 * 24 * time.Hour
	}
	if cfg.MaxFailedLoginAttempts == 0 {
		cfg.MaxFailedLoginAttempts = 5
	}
	if cfg.LockoutDuration == 0 {
		cfg.LockoutDuration = 15 * time.Minute
	}
	svc := &Service{
		repo:     repo,
		mailer:   m,
		sms:      smsSender,
		cfg:      cfg,
		userSvc:  user.NewService(repo),
		signer:   signer,
		verifier: verifier,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

func (s *Service) StartPhoneVerification(ctx context.Context, userID string) (StartPhoneVerificationResponse, error) {
	found, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return StartPhoneVerificationResponse{}, ErrInvalidToken
		}
		return StartPhoneVerificationResponse{}, apperrors.Internal(err)
	}
	if strings.TrimSpace(found.Phone) == "" {
		return StartPhoneVerificationResponse{}, apperrors.BadRequest("phone number is required")
	}
	if found.PhoneVerified {
		return StartPhoneVerificationResponse{}, apperrors.Conflict("phone number is already verified")
	}
	code, err := verificationCode()
	if err != nil {
		return StartPhoneVerificationResponse{}, apperrors.Internal(err)
	}
	now := time.Now().UTC()
	found.PhoneVerificationCodeHash = security.HashToken(code)
	found.PhoneVerificationExpiresAt = now.Add(s.cfg.PhoneVerificationTTL)
	found.UpdatedAt = now
	if err := s.repo.Update(ctx, found); err != nil {
		return StartPhoneVerificationResponse{}, apperrors.Internal(err)
	}
	if s.sms != nil {
		if err := s.sms.SendPhoneVerification(ctx, found.Phone, code); err != nil {
			return StartPhoneVerificationResponse{}, apperrors.Internal(err)
		}
	}
	out := StartPhoneVerificationResponse{ExpiresAt: found.PhoneVerificationExpiresAt}
	if s.cfg.ReturnPhoneVerificationCode {
		out.VerificationCode = code
	}
	return out, nil
}

func (s *Service) VerifyPhone(ctx context.Context, userID string, req VerifyPhoneRequest) error {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return apperrors.BadRequest("code is required")
	}
	found, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return ErrInvalidToken
		}
		return apperrors.Internal(err)
	}
	now := time.Now().UTC()
	if found.PhoneVerificationCodeHash == "" || now.After(found.PhoneVerificationExpiresAt) {
		return apperrors.BadRequest("verification code expired, request a new one")
	}
	if !security.CompareTokenHashes(security.HashToken(code), found.PhoneVerificationCodeHash) {
		return apperrors.BadRequest("invalid verification code")
	}
	expectedCodeHash := found.PhoneVerificationCodeHash
	normalizedPhone := normalizePhone(found.Phone)
	if normalizedPhone == "" {
		return apperrors.BadRequest("phone number is required")
	}
	found.PhoneVerified = true
	found.Phone = normalizedPhone
	found.PhoneVerificationCodeHash = ""
	found.PhoneVerificationExpiresAt = time.Time{}
	found.UpdatedAt = now
	if committer, ok := s.repo.(PhoneVerificationCommitter); ok {
		if _, err := committer.VerifyPhoneAndGrantSignupBonus(ctx, found, normalizedPhone, expectedCodeHash, signupBonusPoints); err != nil {
			if errors.Is(err, user.ErrAlreadyExists) {
				return apperrors.Conflict("phone number is already verified")
			}
			return apperrors.Internal(err)
		}
		return nil
	}
	if err := s.repo.Update(ctx, found); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) StartPhoneChange(ctx context.Context, userID string, req StartPhoneChangeRequest) (StartPhoneVerificationResponse, error) {
	newPhone := normalizePhone(req.NewPhone)
	if newPhone == "" {
		return StartPhoneVerificationResponse{}, apperrors.BadRequest("newPhone is required")
	}
	found, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return StartPhoneVerificationResponse{}, ErrInvalidToken
		}
		return StartPhoneVerificationResponse{}, apperrors.Internal(err)
	}
	if found.Status() != user.AccountStatusActive {
		return StartPhoneVerificationResponse{}, ErrInvalidToken
	}
	if !security.VerifyPassword(req.Password, found.PasswordHash) {
		return StartPhoneVerificationResponse{}, ErrInvalidCredentials
	}
	if normalizePhone(found.Phone) == newPhone {
		return StartPhoneVerificationResponse{}, apperrors.Conflict("new phone must differ from current")
	}
	code, err := verificationCode()
	if err != nil {
		return StartPhoneVerificationResponse{}, apperrors.Internal(err)
	}
	now := time.Now().UTC()
	found.PendingPhone = newPhone
	found.PendingPhoneVerificationCodeHash = security.HashToken(code)
	found.PendingPhoneVerificationExpiresAt = now.Add(s.cfg.PhoneVerificationTTL)
	found.UpdatedAt = now
	if err := s.repo.Update(ctx, found); err != nil {
		return StartPhoneVerificationResponse{}, apperrors.Internal(err)
	}
	if s.sms != nil {
		if err := s.sms.SendPhoneVerification(ctx, newPhone, code); err != nil {
			return StartPhoneVerificationResponse{}, apperrors.Internal(err)
		}
	}
	out := StartPhoneVerificationResponse{ExpiresAt: found.PendingPhoneVerificationExpiresAt}
	if s.cfg.ReturnPhoneVerificationCode {
		out.VerificationCode = code
	}
	return out, nil
}

func (s *Service) VerifyPhoneChange(ctx context.Context, userID string, req VerifyPhoneChangeRequest) (user.Response, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return user.Response{}, apperrors.BadRequest("code is required")
	}
	found, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return user.Response{}, ErrInvalidToken
		}
		return user.Response{}, apperrors.Internal(err)
	}
	now := time.Now().UTC()
	if found.PendingPhone == "" || found.PendingPhoneVerificationCodeHash == "" || now.After(found.PendingPhoneVerificationExpiresAt) {
		return user.Response{}, apperrors.BadRequest("verification code expired, request a new one")
	}
	if !security.CompareTokenHashes(security.HashToken(code), found.PendingPhoneVerificationCodeHash) {
		return user.Response{}, apperrors.BadRequest("invalid verification code")
	}
	expectedCodeHash := found.PendingPhoneVerificationCodeHash
	oldPhone := normalizePhone(found.Phone)
	newPhone := normalizePhone(found.PendingPhone)
	if newPhone == "" {
		return user.Response{}, apperrors.BadRequest("newPhone is required")
	}
	found.Phone = newPhone
	found.PhoneVerified = true
	found.PendingPhone = ""
	found.PendingPhoneVerificationCodeHash = ""
	found.PendingPhoneVerificationExpiresAt = time.Time{}
	found.UpdatedAt = now
	if committer, ok := s.repo.(PhoneChangeCommitter); ok {
		if err := committer.CommitPhoneChange(ctx, found, oldPhone, newPhone, expectedCodeHash); err != nil {
			if errors.Is(err, user.ErrAlreadyExists) {
				return user.Response{}, apperrors.Conflict("phone number is already verified")
			}
			return user.Response{}, apperrors.Internal(err)
		}
		return user.ToResponse(found), nil
	}
	if err := s.repo.Update(ctx, found); err != nil {
		return user.Response{}, apperrors.Internal(err)
	}
	return user.ToResponse(found), nil
}

func normalizePhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	var b strings.Builder
	for i, r := range trimmed {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		if r == '+' && i == 0 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
	if len(req.Password) < 8 {
		return AuthResponse{}, apperrors.BadRequest("password must be at least 8 characters")
	}
	for purpose := range req.Consents {
		if !validConsentPurpose(purpose) {
			return AuthResponse{}, apperrors.BadRequest("unsupported consent purpose")
		}
	}

	token, err := security.GenerateToken(32)
	if err != nil {
		return AuthResponse{}, apperrors.Internal(err)
	}

	created, err := s.userSvc.Create(ctx, user.CreateRequest{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		Phone:       req.Phone,
		City:        req.City,
		BirthYear:   req.BirthYear,
		Gender:      req.Gender,
	}, security.HashPassword(req.Password))
	if err != nil {
		return AuthResponse{}, err
	}

	if s.consents != nil {
		if err := s.consents.SetInitialConsents(ctx, created.ID, req.Consents); err != nil {
			if cleanupErr := s.repo.Delete(ctx, created.ID); cleanupErr != nil {
				return AuthResponse{}, apperrors.Internal(errors.Join(err, cleanupErr))
			}
			return AuthResponse{}, err
		}
	}

	created.EmailVerified = false
	created.EmailVerificationTokenHash = security.HashToken(token)
	created.EmailVerificationExpiresAt = time.Now().UTC().Add(s.cfg.VerificationTokenTTL)
	created.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, created); err != nil {
		return AuthResponse{}, apperrors.Internal(err)
	}

	if err := s.mailer.SendVerificationEmail(ctx, created.Email, token); err != nil {
		return AuthResponse{}, apperrors.Internal(err)
	}

	return s.issue(ctx, created, req.DeviceID)
}

func validConsentPurpose(purpose string) bool {
	for _, valid := range consentPurposes {
		if purpose == valid {
			return true
		}
	}
	return false
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return AuthResponse{}, apperrors.BadRequest("valid email is required")
	}
	found, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return AuthResponse{}, ErrInvalidCredentials
		}
		return AuthResponse{}, apperrors.Internal(err)
	}

	now := time.Now().UTC()
	if !found.LockedUntil.IsZero() && now.Before(found.LockedUntil) {
		return AuthResponse{}, &AccountLockedError{
			RetryAfter: found.LockedUntil.Sub(now),
			AppError:   ErrAccountLocked,
		}
	}

	if !security.VerifyPassword(req.Password, found.PasswordHash) {
		s.recordFailedLogin(ctx, found, now)
		return AuthResponse{}, ErrInvalidCredentials
	}

	if s.cfg.RequireVerifiedEmail && !found.EmailVerified {
		return AuthResponse{}, ErrEmailNotVerified
	}

	if found.FailedLoginAttempts != 0 || !found.LockedUntil.IsZero() {
		found.FailedLoginAttempts = 0
		found.LockedUntil = time.Time{}
		found.UpdatedAt = now
		_ = s.repo.Update(ctx, found)
	}

	return s.issue(ctx, found, req.DeviceID)
}

func (s *Service) Reactivate(ctx context.Context, req ReactivateRequest) (AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return AuthResponse{}, apperrors.BadRequest("valid email is required")
	}
	found, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return AuthResponse{}, ErrInvalidCredentials
		}
		return AuthResponse{}, apperrors.Internal(err)
	}
	now := time.Now().UTC()
	if !found.LockedUntil.IsZero() && now.Before(found.LockedUntil) {
		return AuthResponse{}, &AccountLockedError{
			RetryAfter: found.LockedUntil.Sub(now),
			AppError:   ErrAccountLocked,
		}
	}
	if !security.VerifyPassword(req.Password, found.PasswordHash) {
		s.recordFailedLogin(ctx, found, now)
		return AuthResponse{}, ErrInvalidCredentials
	}
	if s.cfg.RequireVerifiedEmail && !found.EmailVerified {
		return AuthResponse{}, ErrEmailNotVerified
	}
	status := found.Status()
	if status != user.AccountStatusDeactivated && status != user.AccountStatusDeletionPending {
		return AuthResponse{}, apperrors.Conflict("account is not eligible for reactivation")
	}
	if status == user.AccountStatusDeletionPending && !found.ScheduledPurgeAt.IsZero() && !now.Before(found.ScheduledPurgeAt) {
		return AuthResponse{}, apperrors.Conflict("account is not eligible for reactivation")
	}
	found.AccountStatus = user.AccountStatusActive
	found.DeactivatedAt = time.Time{}
	found.DeletionRequestedAt = time.Time{}
	found.ScheduledPurgeAt = time.Time{}
	found.FailedLoginAttempts = 0
	found.LockedUntil = time.Time{}
	found.UpdatedAt = now
	if err := s.repo.Update(ctx, found); err != nil {
		return AuthResponse{}, apperrors.Internal(err)
	}
	return s.issue(ctx, found, req.DeviceID)
}

func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (AuthResponse, error) {
	raw := strings.TrimSpace(req.RefreshToken)
	if raw == "" {
		return AuthResponse{}, ErrInvalidToken
	}
	userID, token, ok := strings.Cut(raw, ".")
	if !ok || strings.TrimSpace(userID) == "" || strings.TrimSpace(token) == "" {
		return AuthResponse{}, ErrInvalidToken
	}
	found, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return AuthResponse{}, ErrInvalidToken
		}
		return AuthResponse{}, apperrors.Internal(err)
	}

	now := time.Now().UTC()
	if found.RefreshTokenHash == "" ||
		now.After(found.RefreshTokenExpiresAt) ||
		now.After(found.RefreshTokenAbsoluteExpiresAt) {
		s.clearRefreshSession(ctx, found, now)
		return AuthResponse{}, ErrInvalidToken
	}

	if !security.CompareTokenHashes(security.HashToken(raw), found.RefreshTokenHash) {
		// Reuse detection: an already-rotated token was presented. Clear the
		// active session so the legitimate client must authenticate again.
		s.clearRefreshSession(ctx, found, now)
		return AuthResponse{}, ErrInvalidToken
	}

	// Device binding: when a deviceID was recorded for this session, the
	// caller must present the same one. A mismatch is treated like reuse:
	// clear the session so a stolen refresh token alone can't mint tokens.
	if found.RefreshTokenDeviceID != "" && req.DeviceID != found.RefreshTokenDeviceID {
		s.clearRefreshSession(ctx, found, now)
		return AuthResponse{}, ErrInvalidToken
	}

	return s.issueWithAbsoluteExpiry(ctx, found, found.RefreshTokenAbsoluteExpiresAt, found.RefreshTokenDeviceID)
}

func (s *Service) Logout(ctx context.Context, req LogoutRequest) error {
	raw := strings.TrimSpace(req.RefreshToken)
	if raw == "" {
		return nil
	}
	userID, _, ok := strings.Cut(raw, ".")
	if !ok || userID == "" {
		return nil
	}
	found, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil
		}
		return apperrors.Internal(err)
	}
	now := time.Now().UTC()
	// Bumping TokenVersion invalidates every outstanding access token for
	// this user; the auth middleware rejects on mismatch.
	found.TokenVersion++
	found.RefreshTokenHash = ""
	found.RefreshTokenExpiresAt = time.Time{}
	found.RefreshTokenAbsoluteExpiresAt = time.Time{}
	found.RefreshTokenDeviceID = ""
	found.UpdatedAt = now
	_ = s.repo.Update(ctx, found)
	return nil
}

// recordFailedLogin increments the failure counter and locks the account if
// the threshold is exceeded. Errors here are swallowed because the user-facing
// error is already the invalid-credentials response.
func (s *Service) recordFailedLogin(ctx context.Context, u user.User, now time.Time) {
	u.FailedLoginAttempts++
	if u.FailedLoginAttempts >= s.cfg.MaxFailedLoginAttempts {
		u.LockedUntil = now.Add(s.cfg.LockoutDuration)
		u.FailedLoginAttempts = 0
	}
	u.UpdatedAt = now
	_ = s.repo.Update(ctx, u)
}

func (s *Service) VerifyEmail(ctx context.Context, req VerifyEmailRequest) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return apperrors.BadRequest("valid email is required")
	}
	if strings.TrimSpace(req.Token) == "" {
		return apperrors.BadRequest("token is required")
	}
	found, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return apperrors.BadRequest("invalid verification token")
		}
		return apperrors.Internal(err)
	}
	if found.EmailVerified {
		return nil
	}
	now := time.Now().UTC()
	if found.EmailVerificationTokenHash == "" || now.After(found.EmailVerificationExpiresAt) {
		return apperrors.BadRequest("verification token expired, request a new one")
	}
	if !security.CompareTokenHashes(security.HashToken(req.Token), found.EmailVerificationTokenHash) {
		return apperrors.BadRequest("invalid verification token")
	}
	found.EmailVerified = true
	found.EmailVerificationTokenHash = ""
	found.EmailVerificationExpiresAt = time.Time{}
	found.UpdatedAt = now
	if err := s.repo.Update(ctx, found); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) ResendVerification(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return apperrors.BadRequest("valid email is required")
	}
	found, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil // do not leak existence
		}
		return apperrors.Internal(err)
	}
	if found.EmailVerified {
		return nil
	}
	token, err := security.GenerateToken(32)
	if err != nil {
		return apperrors.Internal(err)
	}
	now := time.Now().UTC()
	found.EmailVerificationTokenHash = security.HashToken(token)
	found.EmailVerificationExpiresAt = now.Add(s.cfg.VerificationTokenTTL)
	found.UpdatedAt = now
	if err := s.repo.Update(ctx, found); err != nil {
		return apperrors.Internal(err)
	}
	return s.mailer.SendVerificationEmail(ctx, found.Email, token)
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return apperrors.BadRequest("valid email is required")
	}
	found, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil // silent to avoid email enumeration
		}
		return apperrors.Internal(err)
	}
	token, err := security.GenerateToken(32)
	if err != nil {
		return apperrors.Internal(err)
	}
	now := time.Now().UTC()
	found.PasswordResetTokenHash = security.HashToken(token)
	found.PasswordResetExpiresAt = now.Add(s.cfg.PasswordResetTokenTTL)
	found.UpdatedAt = now
	if err := s.repo.Update(ctx, found); err != nil {
		return apperrors.Internal(err)
	}
	return s.mailer.SendPasswordResetEmail(ctx, found.Email, token)
}

func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return apperrors.BadRequest("valid email is required")
	}
	if strings.TrimSpace(req.Token) == "" {
		return apperrors.BadRequest("token is required")
	}
	if len(req.NewPassword) < 8 {
		return apperrors.BadRequest("password must be at least 8 characters")
	}
	found, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return apperrors.BadRequest("invalid or expired reset token")
		}
		return apperrors.Internal(err)
	}
	now := time.Now().UTC()
	if found.PasswordResetTokenHash == "" || now.After(found.PasswordResetExpiresAt) {
		return apperrors.BadRequest("invalid or expired reset token")
	}
	if !security.CompareTokenHashes(security.HashToken(req.Token), found.PasswordResetTokenHash) {
		return apperrors.BadRequest("invalid or expired reset token")
	}
	found.PasswordHash = security.HashPassword(req.NewPassword)
	found.PasswordResetTokenHash = ""
	found.PasswordResetExpiresAt = time.Time{}
	found.FailedLoginAttempts = 0
	found.LockedUntil = time.Time{}
	s.closeSessions(&found, now)
	if err := s.repo.Update(ctx, found); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

// ChangePasswordRequest is the body of POST /auth/password/change. The
// caller must already be authenticated (we re-verify the OLD password
// here as a step-up) and is expected to compute the new identity-backup
// blob with the new password before calling this endpoint.
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// ChangePassword verifies the old password and replaces the hash with a
// new one. The identity-backup blob is updated by the client in a separate
// request before this endpoint is called.
func (s *Service) ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) error {
	if len(req.NewPassword) < 8 {
		return apperrors.BadRequest("password must be at least 8 characters")
	}
	if req.NewPassword == req.OldPassword {
		return apperrors.BadRequest("new password must differ from current")
	}
	found, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return apperrors.Unauthorized("session invalid")
		}
		return apperrors.Internal(err)
	}
	if !security.VerifyPassword(req.OldPassword, found.PasswordHash) {
		return ErrInvalidCredentials
	}
	found.PasswordHash = security.HashPassword(req.NewPassword)
	found.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, found); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) Deactivate(ctx context.Context, userID string, req PasswordStepUpRequest) error {
	found, err := s.passwordStepUp(ctx, userID, req.Password)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	found.AccountStatus = user.AccountStatusDeactivated
	found.DeactivatedAt = now
	s.closeSessions(&found, now)
	if err := s.closeAccountFlows(ctx, found.ID, now); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, found); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) DeleteAccount(ctx context.Context, userID string, req PasswordStepUpRequest) (DeleteAccountResponse, error) {
	found, err := s.passwordStepUp(ctx, userID, req.Password)
	if err != nil {
		return DeleteAccountResponse{}, err
	}
	now := time.Now().UTC()
	found.AccountStatus = user.AccountStatusDeletionPending
	found.DeletionRequestedAt = now
	found.ScheduledPurgeAt = now.Add(30 * 24 * time.Hour)
	s.closeSessions(&found, now)
	if err := s.revokeConsents(ctx, found.ID); err != nil {
		return DeleteAccountResponse{}, err
	}
	if err := s.closeAccountFlows(ctx, found.ID, now); err != nil {
		return DeleteAccountResponse{}, err
	}
	if err := s.repo.Update(ctx, found); err != nil {
		return DeleteAccountResponse{}, apperrors.Internal(err)
	}
	return DeleteAccountResponse{Status: found.AccountStatus, ScheduledPurgeAt: found.ScheduledPurgeAt}, nil
}

// VerifyToken validates the bearer token's signature and expiry, then looks
// up the user to confirm the token's TokenVersion still matches. The DB
// GetItem on every authenticated request is the cost of true server-side
// revocation (Logout bumps TokenVersion to invalidate outstanding tokens).
// The users table is pk-keyed and warm; this is acceptable.
func (s *Service) VerifyToken(ctx context.Context, token string) (Claims, error) {
	claims, err := s.verifier.Parse(token)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	found, err := s.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return Claims{}, ErrInvalidToken
		}
		return Claims{}, apperrors.Internal(err)
	}
	if claims.TokenVersion != found.TokenVersion {
		return Claims{}, ErrInvalidToken
	}
	if found.Status() != user.AccountStatusActive {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

func (s *Service) issue(ctx context.Context, u user.User, deviceID string) (AuthResponse, error) {
	return s.issueWithAbsoluteExpiry(ctx, u, time.Now().UTC().Add(s.cfg.RefreshTokenAbsoluteTTL), deviceID)
}

func (s *Service) issueWithAbsoluteExpiry(ctx context.Context, u user.User, absoluteExpiresAt time.Time, deviceID string) (AuthResponse, error) {
	if u.Status() != user.AccountStatusActive {
		return AuthResponse{}, ErrInvalidCredentials
	}
	token, expiresAt, err := s.signer.Sign(u.ID, u.Email, deviceID, u.TokenVersion, s.cfg.TokenTTL)
	if err != nil {
		return AuthResponse{}, apperrors.Internal(err)
	}
	refreshToken, refreshExpiresAt, err := s.rotateRefreshToken(ctx, u, absoluteExpiresAt, deviceID)
	if err != nil {
		return AuthResponse{}, err
	}
	return AuthResponse{
		Token:            token,
		ExpiresAt:        expiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
		User:             user.ToResponse(u),
	}, nil
}

func (s *Service) passwordStepUp(ctx context.Context, userID, password string) (user.User, error) {
	found, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return user.User{}, ErrInvalidToken
		}
		return user.User{}, apperrors.Internal(err)
	}
	if found.Status() != user.AccountStatusActive {
		return user.User{}, ErrInvalidToken
	}
	if !security.VerifyPassword(password, found.PasswordHash) {
		return user.User{}, ErrInvalidCredentials
	}
	return found, nil
}

func (s *Service) closeSessions(u *user.User, now time.Time) {
	u.TokenVersion++
	u.RefreshTokenHash = ""
	u.RefreshTokenExpiresAt = time.Time{}
	u.RefreshTokenAbsoluteExpiresAt = time.Time{}
	u.RefreshTokenDeviceID = ""
	u.UpdatedAt = now
}

func (s *Service) closeAccountFlows(ctx context.Context, userID string, now time.Time) error {
	if s.lifecycle == nil {
		return nil
	}
	if err := s.lifecycle.CloseAccount(ctx, userID, now); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) revokeConsents(ctx context.Context, userID string) error {
	if s.consents == nil {
		return nil
	}
	values := make(map[string]bool, len(consentPurposes))
	for _, purpose := range consentPurposes {
		values[purpose] = false
	}
	if err := s.consents.SetInitialConsents(ctx, userID, values); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) rotateRefreshToken(ctx context.Context, u user.User, absoluteExpiresAt time.Time, deviceID string) (string, time.Time, error) {
	secret, err := security.GenerateToken(32)
	if err != nil {
		return "", time.Time{}, apperrors.Internal(err)
	}
	now := time.Now().UTC()
	refreshExpiresAt := now.Add(s.cfg.RefreshTokenIdleTTL)
	if refreshExpiresAt.After(absoluteExpiresAt) {
		refreshExpiresAt = absoluteExpiresAt
	}
	refreshToken := u.ID + "." + secret
	u.RefreshTokenHash = security.HashToken(refreshToken)
	u.RefreshTokenExpiresAt = refreshExpiresAt
	u.RefreshTokenAbsoluteExpiresAt = absoluteExpiresAt
	u.RefreshTokenDeviceID = deviceID
	u.UpdatedAt = now
	if err := s.repo.Update(ctx, u); err != nil {
		return "", time.Time{}, apperrors.Internal(err)
	}
	return refreshToken, refreshExpiresAt, nil
}

func (s *Service) clearRefreshSession(ctx context.Context, u user.User, now time.Time) {
	u.RefreshTokenHash = ""
	u.RefreshTokenExpiresAt = time.Time{}
	u.RefreshTokenAbsoluteExpiresAt = time.Time{}
	u.RefreshTokenDeviceID = ""
	u.UpdatedAt = now
	_ = s.repo.Update(ctx, u)
}

func verificationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
