package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

type AccountState string

const (
	AccountInvited         AccountState = "invited"
	AccountActive          AccountState = "active"
	AccountDisabled        AccountState = "disabled"
	AccountSuspended       AccountState = "suspended"
	AccountLocked          AccountState = "locked"
	AccountDeletionPending AccountState = "deletion-pending"
	AccountDeleted         AccountState = "deleted"
)

type IdentifierType string

const (
	IdentifierEmail    IdentifierType = "email"
	IdentifierUsername IdentifierType = "username"
)

var (
	ErrAccountNotFound              = errors.New("account not found")
	ErrDuplicateIdentifier          = errors.New("identifier already exists")
	ErrInvalidIdentifier            = errors.New("identifier is invalid")
	ErrInvalidSubject               = errors.New("subject is invalid")
	ErrInvalidState                 = errors.New("account state is invalid")
	ErrInvalidTransition            = errors.New("account state transition is invalid")
	ErrRevisionConflict             = errors.New("account revision conflict")
	ErrEmailNotVerified             = errors.New("email is not verified")
	ErrPublicRegistrationDisabled   = errors.New("public registration is disabled")
	ErrAuthenticationFailed         = errors.New("authentication failed")
	ErrAuthenticationRateLimited    = errors.New("authentication rate limited")
	ErrAuthenticationUnavailable    = errors.New("authentication temporarily unavailable")
	ErrInvitationResendLimit        = errors.New("invitation resend limit exceeded")
	ErrInvitationExpired            = errors.New("invitation expired")
	ErrInvitationNotFound           = errors.New("invitation not found")
	ErrSessionRevocationUnavailable = errors.New("provider session revocation unavailable")
	ErrProviderNotFound             = errors.New("provider account not found")
	ErrProviderRevisionConflict     = errors.New("provider membership revision conflict")
	ErrInvalidRecovery              = errors.New("recovery operation is invalid")
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)

type Identifier struct {
	Type       IdentifierType
	Normalized string
}

func NormalizeIdentifier(kind IdentifierType, value string) (Identifier, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len(trimmed) > 320 {
		return Identifier{}, ErrInvalidIdentifier
	}
	switch kind {
	case IdentifierEmail:
		address, err := mail.ParseAddress(trimmed)
		if err != nil || address.Address != trimmed || !strings.Contains(trimmed, "@") {
			return Identifier{}, ErrInvalidIdentifier
		}
		return Identifier{Type: kind, Normalized: strings.ToLower(trimmed)}, nil
	case IdentifierUsername:
		normalized := strings.ToLower(trimmed)
		if !usernamePattern.MatchString(normalized) {
			return Identifier{}, ErrInvalidIdentifier
		}
		return Identifier{Type: kind, Normalized: normalized}, nil
	default:
		return Identifier{}, ErrInvalidIdentifier
	}
}

func DetectIdentifier(value string) (Identifier, error) {
	if strings.Contains(value, "@") {
		return NormalizeIdentifier(IdentifierEmail, value)
	}
	return NormalizeIdentifier(IdentifierUsername, value)
}

func NewSubjectID() (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate opaque subject: %w", err)
	}
	return "usr_" + base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func ValidateSubjectID(subjectID string) error {
	if len(subjectID) != len("usr_")+22 || !strings.HasPrefix(subjectID, "usr_") {
		return ErrInvalidSubject
	}
	return nil
}

func DigestSecret(value string) [32]byte {
	return sha256.Sum256([]byte(value))
}

type AccountSnapshot struct {
	SubjectID      string
	Email          string
	Username       string
	State          AccountState
	EmailVerified  bool
	AuthRevision   uint64
	FailedAttempts int
	LockedUntil    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ProviderAuthority is the first-party provider's current, application-owned
// authority projection. auth_revision belongs to the account security state;
// membership_revision belongs only to the application authority contract.
type ProviderAuthority struct {
	SubjectID          string
	Email              string
	DisplayName        string
	GivenName          string
	FamilyName         string
	EmailVerified      bool
	State              string
	MembershipID       string
	OrganizationID     string
	Role               string
	MembershipRevision int64
	AuthRevision       uint64
	MFAEnrolled        bool
	Locked             bool
	UpdatedAt          time.Time
}

type ProviderAuthorityInput struct {
	MembershipID      string
	OrganizationID    string
	Role              string
	State             string
	ExpectedRevision  int64
	ResultingRevision int64
}

type ProviderProfileInput struct {
	DisplayName string
	GivenName   string
	FamilyName  string
}

type InvitationSnapshot struct {
	SubjectID     string
	Token         string // populated only at issuance; never persisted or logged
	IssuedAt      time.Time
	ExpiresAt     time.Time
	ResendCount   int
	InvalidatedAt time.Time
	ConsumedAt    time.Time
}

func (snapshot AccountSnapshot) CanIssueCredentials(now time.Time) bool {
	return snapshot.State == AccountActive && snapshot.EmailVerified &&
		(snapshot.LockedUntil.IsZero() || !snapshot.LockedUntil.After(now))
}
