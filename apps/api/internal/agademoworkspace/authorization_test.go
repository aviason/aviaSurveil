package agademoworkspace

import (
	"context"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func bindingFor(subject, organization string, roles ...string) preprod.AuthorityBinding {
	return preprod.AuthorityBinding{BindingID: "binding-" + subject, SubjectSlot: subject, MembershipSlot: subject + "-membership", OrganizationID: organization, DepartmentID: "department", OrganizationalUnitID: "unit", OperationRoles: roles, BindingDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000", Active: true}
}

func principalFor(subject, organization string, role identity.Role) identity.Principal {
	return identity.Principal{SubjectID: subject, OrganizationID: organization, Roles: []identity.Role{role}}
}

func TestWorkspaceClassificationAuthorizationMatrix(t *testing.T) {
	service := NewService(ServiceConfig{Resolver: StaticBindingResolver{Bindings: map[string]preprod.AuthorityBinding{
		"manager":   bindingFor("manager", "ORG", "manager"),
		"inspector": bindingFor("inspector", "ORG", "inspector"),
	}}})
	if _, err := service.Authorize(context.Background(), principalFor("manager", "ORG", identity.RoleDepartmentManager), OperationGetSummary); err != nil {
		t.Fatalf("scoped manager classification read denied: %v", err)
	}
	if _, err := service.Authorize(context.Background(), principalFor("inspector", "ORG", identity.RoleInspector), OperationGetSummary); err == nil {
		t.Fatal("inspector received classification authority")
	}
	if _, err := service.Authorize(context.Background(), principalFor("manager", "ORG", identity.RoleDepartmentManager), OperationInclude); err != nil {
		t.Fatalf("scoped manager classification command denied: %v", err)
	}
	if _, err := service.Authorize(context.Background(), principalFor("manager", "OTHER", identity.RoleDepartmentManager), OperationInclude); err == nil {
		t.Fatal("cross-organization manager received classification authority")
	}
}

func TestWorkspaceLifecycleAuthorizationMatrix(t *testing.T) {
	service := NewService(ServiceConfig{Resolver: StaticBindingResolver{Bindings: map[string]preprod.AuthorityBinding{
		"inspector": bindingFor("inspector", "ORG", "inspector"),
		"lead":      bindingFor("lead", "ORG", "lead", "caa_reviewer"),
		"auditee":   bindingFor("auditee", "ORG", "auditee"),
	}}})
	if _, err := service.Authorize(context.Background(), principalFor("inspector", "ORG", identity.RoleInspector), OperationRecordResponse); err != nil {
		t.Fatalf("assigned inspector operation denied: %v", err)
	}
	if _, err := service.Authorize(context.Background(), principalFor("lead", "ORG", identity.RoleLeadInspector), OperationReviewCAP); err != nil {
		t.Fatalf("reviewer binding operation denied: %v", err)
	}
	if _, err := service.Authorize(context.Background(), principalFor("auditee", "ORG", identity.RoleAuditee), OperationSubmitEvidence); err != nil {
		t.Fatalf("matching auditee operation denied: %v", err)
	}
	if _, err := service.Authorize(context.Background(), principalFor("auditee", "ORG", identity.RoleAuditee), OperationGetRoleHistory); err == nil {
		t.Fatal("auditee received role history")
	}
	if _, err := service.Authorize(context.Background(), principalFor("inspector", "ORG", identity.RoleInspector), OperationAuthorizedClose); err == nil {
		t.Fatal("inspector received authorized close")
	}
}

func TestWorkspaceLifecycleAuthorizationAcceptsExactReviewerMembership(t *testing.T) {
	service := NewService(ServiceConfig{Resolver: StaticBindingResolver{Bindings: map[string]preprod.AuthorityBinding{
		"lead": {
			BindingID: "binding-lead-reviewer", MembershipSlot: "CAA_REVIEWER_MEMBERSHIP", OrganizationID: "ORG",
			DepartmentID: "department", OrganizationalUnitID: "unit", OperationRoles: []string{"lead_inspector"}, Active: true,
		},
	}}})
	if _, err := service.Authorize(context.Background(), principalFor("lead", "ORG", identity.RoleLeadInspector), OperationReviewCAP); err != nil {
		t.Fatalf("exact reviewer membership denied: %v", err)
	}
}

func TestWorkspaceAdminResetRequiresCAAOrganization(t *testing.T) {
	service := NewService(ServiceConfig{})
	if _, err := service.Authorize(context.Background(), principalFor("admin", "ORG", identity.RoleAdmin), OperationResetGeneration); err == nil {
		t.Fatal("non-CAA admin received reset authority")
	}
	if _, err := service.Authorize(context.Background(), principalFor("admin", "CAA", identity.RoleAdmin), OperationResetGeneration); err != nil {
		t.Fatalf("CAA admin reset denied: %v", err)
	}
}
