package caps

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestAuthorizeCapRevisionReadAudience(t *testing.T) {
	lead := identity.Principal{SubjectID: "USR-LEAD-CANER", Roles: []identity.Role{identity.RoleLeadInspector}}
	manager := identity.Principal{SubjectID: "USR-MANAGER-NORA", Roles: []identity.Role{identity.RoleDepartmentManager}}
	auditee := identity.Principal{SubjectID: "USR-AUDITEE-FLY", Roles: []identity.Role{identity.RoleAuditee}, OrganizationID: "ORG-FLY-NAMIBIA"}
	gm := identity.Principal{SubjectID: "USR-GM-OMAR", Roles: []identity.Role{identity.RoleGeneralManager}}

	for _, actor := range []identity.Principal{lead, manager} {
		audience, err := AuthorizeRevisionRead(RevisionReadAuthorizationInput{
			Actor:                 actor,
			FindingOrganizationID: "ORG-FLY-NAMIBIA",
			FindingAuthorized:     true,
		})
		if err != nil {
			t.Fatalf("CAA CAP revision read failed for %s: %v", actor.SubjectID, err)
		}
		if audience != AudienceCAA {
			t.Fatalf("audience = %q, want CAA", audience)
		}
	}

	audience, err := AuthorizeRevisionRead(RevisionReadAuthorizationInput{
		Actor:                 auditee,
		FindingOrganizationID: "ORG-FLY-NAMIBIA",
		FindingAuthorized:     false,
	})
	if err != nil {
		t.Fatalf("auditee CAP revision read failed: %v", err)
	}
	if audience != AudienceAuditee {
		t.Fatalf("audience = %q, want AUDITEE", audience)
	}

	if _, err := AuthorizeRevisionRead(RevisionReadAuthorizationInput{
		Actor:                 auditee,
		FindingOrganizationID: "ORG-SKYCARGO",
		FindingAuthorized:     false,
	}); err == nil {
		t.Fatal("auditee must not read another organization's CAP revision")
	}
	if _, err := AuthorizeRevisionRead(RevisionReadAuthorizationInput{
		Actor:                 gm,
		FindingOrganizationID: "ORG-FLY-NAMIBIA",
		FindingAuthorized:     true,
	}); err == nil {
		t.Fatal("GM must not read CAP revision lifecycle detail")
	}
}
