package identity

import (
	"context"
	"errors"
)

var (
	ErrProviderDuplicateEmail   = errors.New("identity provider email already exists")
	ErrProviderDuplicateSubject = errors.New("identity provider subject is already bound")
	ErrProviderUnavailable      = errors.New("identity provider unavailable")
	ErrProviderPermanent        = errors.New("identity provider rejected the operation")
	ErrProviderManualReview     = errors.New("identity provider operation requires manual review")
	ErrProviderRevisionConflict = errors.New("identity provider membership revision conflict")
)

type ProviderFailureClass string

const (
	ProviderFailureRetryable    ProviderFailureClass = "RETRYABLE"
	ProviderFailurePermanent    ProviderFailureClass = "PERMANENT"
	ProviderFailureManualReview ProviderFailureClass = "MANUAL_REVIEW"
)

// ClassifyProviderError keeps worker failure semantics stable across the
// first-party administration boundary.
func ClassifyProviderError(err error) ProviderFailureClass {
	switch {
	case errors.Is(err, ErrProviderUnavailable):
		return ProviderFailureRetryable
	case errors.Is(err, ErrProviderDuplicateEmail), errors.Is(err, ErrProviderPermanent):
		return ProviderFailurePermanent
	case errors.Is(err, ErrProviderDuplicateSubject), errors.Is(err, ErrProviderManualReview):
		return ProviderFailureManualReview
	default:
		return ProviderFailureManualReview
	}
}

func ProviderFailureReasonCode(err error) string {
	switch {
	case errors.Is(err, ErrProviderDuplicateEmail):
		return "DUPLICATE_EMAIL"
	case errors.Is(err, ErrProviderDuplicateSubject):
		return "DUPLICATE_SUBJECT"
	case errors.Is(err, ErrProviderUnavailable):
		return "PROVIDER_UNAVAILABLE"
	case errors.Is(err, ErrProviderPermanent):
		return "PROVIDER_REJECTED"
	default:
		return "PROVIDER_MANUAL_REVIEW"
	}
}

// ProviderUser is the provider-neutral provisioning input owned by Avia's
// lifecycle contract. Provider adapters translate this value to their native
// administration API; callers must not depend on provider-specific field
// names, URLs, or endpoint semantics.
type ProviderUser struct {
	Email          string
	FirstName      string
	LastName       string
	DisplayName    string
	MembershipID   string
	OrganizationID string
	Roles          []Role
}

type ProviderDirectoryQuery struct {
	First  int
	Limit  int
	Search string
}

type ProviderDirectoryUser struct {
	SubjectID          string
	Email              string
	DisplayName        string
	OrganizationID     string
	Enabled            bool
	TOTPConfigured     bool
	RequiredActions    []string
	Roles              []Role
	MembershipID       string
	MembershipRevision int64
	AuthRevision       uint64
	State              string
}

type ProviderDirectoryPage struct {
	Users         []ProviderDirectoryUser
	NextFirst     int
	ProviderCalls int
}

// ProviderAdmin is the API's first-party administration boundary. Callers
// depend only on neutral lifecycle semantics, never provider endpoint names.
type ProviderAdmin interface {
	ListDirectory(context.Context, ProviderDirectoryQuery) (ProviderDirectoryPage, error)
	ObserveUserAuthority(context.Context, string) (AuthorityObservation, error)
	ProvisionUser(context.Context, ProviderUser) (string, error)
	ReconcileProvisionedUser(context.Context, ProviderUser) (string, bool, error)
	DisableUser(context.Context, string) error
	UpdateUserAuthority(context.Context, string, string, []Role) error
	EnableUser(context.Context, string) error
	IssueExecuteActionsEmail(context.Context, string, []string, int) error
	ResetUserMFA(context.Context, string) error
	ForceUserLogout(context.Context, string) error
}

// RevisionedProviderAdmin participates in the application membership revision
// contract owned by the first-party provider.
type RevisionedProviderAdmin interface {
	ProvisionUserAtRevision(context.Context, ProviderUser, int64, int64) (string, error)
	UpdateUserAuthorityAtRevision(context.Context, string, string, []Role, string, int64, int64) error
	SetUserStateAtRevision(context.Context, string, string, int64, int64) error
}
