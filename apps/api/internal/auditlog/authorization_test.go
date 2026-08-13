package auditlog_test

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/auditlog"
	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestInternalAuditLogAuthorityExcludesAuditeeAndBudgetOnlyFinance(t *testing.T) {
	t.Parallel()
	for _, role := range []identity.Role{
		identity.RoleInspector, identity.RoleLeadInspector, identity.RoleDepartmentManager,
		identity.RoleGeneralManager, identity.RoleExecutiveDirector, identity.RoleAdmin,
	} {
		if !auditlog.CanReadInternal(identity.Principal{Roles: []identity.Role{role}}) {
			t.Errorf("role %s denied internal audit log", role)
		}
	}
	for _, role := range []identity.Role{identity.RoleAuditee, identity.RoleFinance} {
		if auditlog.CanReadInternal(identity.Principal{Roles: []identity.Role{role}}) {
			t.Errorf("role %s allowed internal audit log", role)
		}
	}
}

func TestAuditEventsAreAppendOnlyDomainFacts(t *testing.T) {
	t.Parallel()
	event := auditlog.Event{
		ActorSubjectID: "lead-001", ActorRole: identity.RoleLeadInspector, OrganizationID: "airline-xyz",
		Action: "potential_finding.converted", EntityType: "potential_finding", EntityID: "potential-001",
		EntityVersion: 2, BeforeStatus: "PENDING_LEAD_REVIEW", AfterStatus: "CONVERTED",
		OperationID: "op-001", CorrelationID: "corr-001",
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("valid audit event: %v", err)
	}
	event.OperationID = ""
	if err := event.Validate(); err == nil {
		t.Fatal("audit event without operation identity accepted")
	}
}
