package identity

import "context"

// ProviderUser is the provider-neutral provisioning input owned by Avia's
// lifecycle contract. Provider adapters translate this value to their native
// administration API; callers must not depend on provider-specific field
// names, URLs, or endpoint semantics.
type ProviderUser struct {
	Email          string
	FirstName      string
	LastName       string
	OrganizationID string
	Roles          []Role
}

type ProviderDirectoryQuery struct {
	First  int
	Limit  int
	Search string
}

type ProviderDirectoryUser struct {
	SubjectID       string
	Email           string
	DisplayName     string
	OrganizationID  string
	Enabled         bool
	TOTPConfigured  bool
	RequiredActions []string
	Roles           []Role
}

type ProviderDirectoryPage struct {
	Users         []ProviderDirectoryUser
	NextFirst     int
	ProviderCalls int
}

// ProviderAdmin is the API's provider-neutral administration boundary. The
// current Keycloak adapter implements it while the first-party provider is
// qualified separately; callers must not depend on Keycloak endpoint names.
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

func AsProviderAdmin(client *KeycloakAdminClient) ProviderAdmin {
	return client
}
