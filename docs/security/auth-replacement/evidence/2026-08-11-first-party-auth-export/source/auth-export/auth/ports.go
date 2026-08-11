package auth

import (
	"context"
	"io"
	"time"
)

// The ports in this file are candidate integration boundaries. The copied
// Store does not implement them and must not be represented as doing so.

type UserRepository interface {
	CreateUser(context.Context, RegisterInput, string, string) (AuthUser, error)
	UserByLogin(context.Context, string) (AuthUser, string, error)
	UserBySubject(context.Context, string) (AuthUser, error)
	IncrementAuthVersion(context.Context, int64) error
}

type SessionRepository interface {
	CreateSession(context.Context, User, LoginInput, time.Time, time.Time) (string, error)
	ValidateSession(context.Context, User) (User, bool, error)
	RotateRefreshToken(context.Context, RefreshInput, time.Time) (AuthResponse, error)
	RevokeSession(context.Context, int64, string, string) error
	RevokeAllSessions(context.Context, int64, string) error
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Verify(string, string) (bool, error)
	NeedsRehash(string) bool
}

type MFARepository interface {
	LoadFactors(context.Context, string) ([]MFAFactor, error)
	StoreFactor(context.Context, string, MFAFactor) error
	ConsumeTOTPWindow(context.Context, string, int64) (bool, error)
	ConsumeRecoveryCode(context.Context, string, []byte) (bool, error)
	ResetFactors(context.Context, string, string) error
}

type MFAFactor struct {
	ID        string
	Kind      string
	CreatedAt time.Time
	Verified  bool
}

type AuditSink interface {
	Record(context.Context, AuditEvent) error
}

type AuditEvent struct {
	SubjectID string
	SessionID string
	Type      string
	Outcome   string
	Metadata  map[string]string
	Occurred  time.Time
}

type MailSender interface {
	Send(context.Context, MailMessage) error
}

type MailMessage struct {
	Recipient string
	Locale    string
	Template  string
	Variables map[string]string
}

type Clock interface {
	Now() time.Time
}

type RandomSource interface {
	io.Reader
}

type OrganizationMembershipResolver interface {
	Membership(context.Context, string, string) (OrganizationMembership, error)
}

type OrganizationMembership struct {
	OrganizationID string
	SubjectID      string
	Roles          []string
	Permissions    []string
	Active         bool
}
