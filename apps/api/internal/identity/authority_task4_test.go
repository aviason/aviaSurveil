package identity_test

import (
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

func TestTask4ApplicationAuthorityRequiresOneExactRoleAndOrganization(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name           string
		organizationID string
		roles          []identity.Role
		wantError      bool
	}{
		{
			name:           "CAA inspector",
			organizationID: "CAA",
			roles:          []identity.Role{identity.RoleInspector},
		},
		{
			name:           "Auditee operator",
			organizationID: "ORG-OPERATOR",
			roles:          []identity.Role{identity.RoleAuditee},
		},
		{
			name:           "lowercase CAA",
			organizationID: "caa",
			roles:          []identity.Role{identity.RoleInspector},
			wantError:      true,
		},
		{
			name:           "padded CAA",
			organizationID: " CAA ",
			roles:          []identity.Role{identity.RoleInspector},
			wantError:      true,
		},
		{
			name:           "CAA role outside CAA",
			organizationID: "ORG-OPERATOR",
			roles:          []identity.Role{identity.RoleAdmin},
			wantError:      true,
		},
		{
			name:           "Auditee inside CAA",
			organizationID: "CAA",
			roles:          []identity.Role{identity.RoleAuditee},
			wantError:      true,
		},
		{
			name:           "multiple roles",
			organizationID: "CAA",
			roles: []identity.Role{
				identity.RoleAdmin,
				identity.RoleInspector,
			},
			wantError: true,
		},
		{
			name:           "duplicate role",
			organizationID: "CAA",
			roles: []identity.Role{
				identity.RoleAdmin,
				identity.RoleAdmin,
			},
			wantError: true,
		},
		{
			name:           "unknown role",
			organizationID: "CAA",
			roles:          []identity.Role{"realm-admin"},
			wantError:      true,
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := identity.ValidateApplicationAuthority(
				testCase.organizationID,
				testCase.roles,
			)
			if testCase.wantError && err == nil {
				t.Fatal("invalid authority was accepted")
			}
			if !testCase.wantError && err != nil {
				t.Fatalf("valid authority rejected: %v", err)
			}
		})
	}
}
