package agademoworkspace

import (
	"context"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

type operationBindingResolverForTest struct {
	binding preprod.AuthorityBinding
}

func (resolver operationBindingResolverForTest) Resolve(_ context.Context, _ identity.Principal) (preprod.AuthorityBinding, bool, error) {
	return resolver.binding, true, nil
}

func (resolver operationBindingResolverForTest) ResolveForOperation(_ context.Context, _ identity.Principal, _ string) (preprod.AuthorityBinding, bool, error) {
	return resolver.binding, true, nil
}

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

func TestWorkspaceAuthorityUsesExplicitCAAOrganizationAlias(t *testing.T) {
	service := NewService(ServiceConfig{Resolver: StaticBindingResolver{Bindings: map[string]preprod.AuthorityBinding{
		"manager": bindingFor("manager", "AGA-DEMO-CAA", "manager"),
	}}})
	if _, err := service.Authorize(context.Background(), principalFor("manager", "CAA", identity.RoleDepartmentManager), OperationInclude); err != nil {
		t.Fatalf("CAA source organization alias denied workspace manager operation: %v", err)
	}
}

func TestWorkspaceOperationAuthorizationDoesNotBorrowAnotherBindingRole(t *testing.T) {
	service := NewService(ServiceConfig{Resolver: operationBindingResolverForTest{binding: bindingFor("manager", "ORG", "inspector")}})
	if _, err := service.Authorize(context.Background(), principalFor("manager", "ORG", identity.RoleDepartmentManager), OperationInclude); err == nil {
		t.Fatal("manager operation borrowed inspector role from a distinct authority row")
	}
}

func TestAuthorizationScopeDigestPinsGenerationMembershipAndProviderScope(t *testing.T) {
	principal := principalFor("manager", "ORG", identity.RoleDepartmentManager)
	binding := bindingFor("manager", "ORG", "manager")
	binding.GenerationID = "aga-ws-generation-a"
	binding.MembershipID = "membership-a"
	binding.MembershipVersion = 1
	binding.ProviderScopeID = "aga-ws-scope-a"
	base := AuthorizationScopeDigest(principal, binding, OperationInclude)

	binding.GenerationID = "aga-ws-generation-b"
	if changed := AuthorizationScopeDigest(principal, binding, OperationInclude); changed == base {
		t.Fatal("scope digest omitted generation identity")
	}
	binding.GenerationID = "aga-ws-generation-a"
	binding.MembershipVersion = 2
	if changed := AuthorizationScopeDigest(principal, binding, OperationInclude); changed == base {
		t.Fatal("scope digest omitted membership version")
	}
	binding.MembershipVersion = 1
	binding.ProviderScopeID = "aga-ws-scope-b"
	if changed := AuthorizationScopeDigest(principal, binding, OperationInclude); changed == base {
		t.Fatal("scope digest omitted provider scope")
	}
}

func TestLifecycleScopeRequiresExactProviderScope(t *testing.T) {
	aggregate, _, _, _, _ := lifecycleFixture(t)
	binding := bindingFor("manager", aggregate.OrganizationID, "manager")
	binding.ProviderScopeID = "other-scope"
	if bindingMatchesLifecycleScope(binding, aggregate) {
		t.Fatal("cross-scope manager binding matched lifecycle aggregate")
	}
	binding.ProviderScopeID = aggregate.ProviderScopeID
	if !bindingMatchesLifecycleScope(binding, aggregate) {
		t.Fatal("matching lifecycle scope binding was denied")
	}
}

func TestLifecycleObjectBindingRequiresAssignedSubjectAndExactScope(t *testing.T) {
	aggregate, inspector, _, _, manager := lifecycleFixture(t)
	inspectorBinding := bindingFor(inspector.SubjectID, aggregate.OrganizationID, "inspector")
	inspectorBinding.SubjectID = inspector.SubjectID
	inspectorBinding.DepartmentID = aggregate.Inspector.DepartmentID
	inspectorBinding.OrganizationalUnitID = aggregate.Inspector.OrganizationalUnitID
	inspectorBinding.ProviderScopeID = aggregate.ProviderScopeID
	if !bindingMatchesLifecycleObject(inspectorBinding, aggregate, inspector) {
		t.Fatal("assigned inspector and exact scope binding were denied")
	}
	inspectorBinding.SubjectID = "different-inspector"
	if bindingMatchesLifecycleObject(inspectorBinding, aggregate, inspector) {
		t.Fatal("mismatched binding subject reached assigned inspection")
	}
	if lifecycleBindingPinMatchesPrincipal(aggregate.Inspector, identity.Principal{SubjectID: "different-inspector", OrganizationID: inspector.OrganizationID, Roles: []identity.Role{identity.RoleInspector}}) {
		t.Fatal("binding pin matched a different subject in the same organization")
	}
	managerBinding := bindingFor(manager.SubjectID, aggregate.OrganizationID, "manager")
	managerBinding.SubjectID = manager.SubjectID
	managerBinding.DepartmentID = aggregate.Inspector.DepartmentID
	managerBinding.OrganizationalUnitID = aggregate.Inspector.OrganizationalUnitID
	managerBinding.ProviderScopeID = aggregate.ProviderScopeID
	if !bindingMatchesLifecycleObject(managerBinding, aggregate, manager) {
		t.Fatal("exact manager aggregate scope was denied")
	}
	managerBinding.ProviderScopeID = "different-provider-scope"
	if bindingMatchesLifecycleObject(managerBinding, aggregate, manager) {
		t.Fatal("cross-provider manager binding reached lifecycle aggregate")
	}
}

func TestAdminCapabilityDoesNotAdvertiseManagerMutations(t *testing.T) {
	service := NewService(ServiceConfig{})
	capability, err := service.Capability(context.Background(), principalFor("admin", "CAA", identity.RoleAdmin))
	if err != nil {
		t.Fatal(err)
	}
	if capability.ClassificationEnabled || capability.RecommendationEnabled || !capability.LifecycleEnabled || !capability.ResetEnabled {
		t.Fatalf("CAA admin capability incorrectly advertises manager actions: %+v", capability)
	}
}
