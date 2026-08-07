package agademoworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

var workspaceOperationIDs = []string{
	OperationGetSummary, OperationGetTaxonomy, OperationGetProviderConfiguration, OperationSearchItems, OperationGetDraft, OperationGetHistory,
	OperationGetSimulationSetup, OperationGetCurrentRecommendation, OperationGetCurrentInspection, OperationGetInspectionQuestionPage,
	OperationPreviewBatch, OperationExecuteBatch, OperationRetain, OperationReclassify, OperationAddTopic, OperationRemoveTopic,
	OperationResolve, OperationInclude, OperationExclude, OperationDefer, OperationAddCandidate, OperationReword, OperationMarkReady,
	OperationCreateRecommendation, OperationCreateInspection,
	OperationGetInspection, OperationGetFinding, OperationGetCAPEvidence, OperationGetRoleHistory,
	OperationStartInspection, OperationRecordResponse, OperationCreateFinding, OperationSubmitChecklist, OperationReopenChecklist,
	OperationReturnFinding, OperationDismissFinding, OperationConvertFinding, OperationSubmitCAP, OperationReviewCAP,
	OperationSubmitEvidence, OperationVerifyEvidence, OperationAuthorizedClose, OperationResetGeneration,
}

func (service *Service) ResolveBinding(ctx context.Context, principal identity.Principal) (preprod.AuthorityBinding, bool, error) {
	if service == nil || service.resolver == nil || strings.TrimSpace(principal.SubjectID) == "" {
		return preprod.AuthorityBinding{}, false, nil
	}
	binding, found, err := service.resolver.Resolve(ctx, principal)
	if err != nil || !found {
		return preprod.AuthorityBinding{}, found, err
	}
	if !validWorkspaceBinding(binding, principal) {
		return preprod.AuthorityBinding{}, false, nil
	}
	return binding, true, nil
}

// ResolveBindingForOperation selects one binding row that independently grants
// the requested operation.  It intentionally does not union operation roles,
// organization scope, or membership data from multiple rows.
func (service *Service) ResolveBindingForOperation(ctx context.Context, principal identity.Principal, operation string) (preprod.AuthorityBinding, bool, error) {
	if service == nil || service.resolver == nil || strings.TrimSpace(operation) == "" {
		return preprod.AuthorityBinding{}, false, nil
	}
	if resolver, ok := service.resolver.(OperationBindingResolver); ok {
		binding, found, err := resolver.ResolveForOperation(ctx, principal, operation)
		if err != nil || !found {
			return preprod.AuthorityBinding{}, found, err
		}
		if !validWorkspaceBinding(binding, principal) || !bindingAllowsWorkspaceOperation(binding, principal, operation) {
			return preprod.AuthorityBinding{}, false, nil
		}
		return binding, true, nil
	}
	binding, found, err := service.ResolveBinding(ctx, principal)
	if err != nil || !found || !bindingAllowsWorkspaceOperation(binding, principal, operation) {
		return preprod.AuthorityBinding{}, false, err
	}
	return binding, true, nil
}

func validWorkspaceBinding(binding preprod.AuthorityBinding, principal identity.Principal) bool {
	if !binding.Active || binding.OrganizationID == "" || principal.OrganizationID == "" || !workspaceOrganizationMatchesPrincipal(principal.OrganizationID, binding.OrganizationID) {
		return false
	}
	if binding.SubjectID != "" && binding.SubjectID != principal.SubjectID {
		return false
	}
	return binding.DepartmentID != "" && binding.OrganizationalUnitID != "" && len(binding.OperationRoles) > 0
}

func (service *Service) HasBroadAuthority(ctx context.Context, principal identity.Principal) bool {
	if principal.SubjectID == "" {
		return false
	}
	if principal.HasRole(identity.RoleAdmin) && isCAAOrganization(principal.OrganizationID) {
		return true
	}
	for _, operation := range workspaceOperationIDs {
		if _, found, err := service.ResolveBindingForOperation(ctx, principal, operation); err == nil && found {
			return true
		}
	}
	return false
}

func (service *Service) Authorize(ctx context.Context, principal identity.Principal, operation string) (AuthorizationDecision, error) {
	decision := AuthorizationDecision{Operation: operation, Organization: principal.OrganizationID}
	if strings.TrimSpace(principal.SubjectID) == "" || strings.TrimSpace(operation) == "" {
		return decision, ErrNeutralDenied
	}
	decision.IsAdmin = principal.HasRole(identity.RoleAdmin) && isCAAOrganization(principal.OrganizationID)
	if decision.IsAdmin {
		decision.ScopeDigest = AuthorizationScopeDigest(principal, preprod.AuthorityBinding{}, operation)
		if workspaceAdminAllows(operation) {
			decision.Allowed = true
			return decision, nil
		}
		return decision, ErrNeutralDenied
	}

	binding, hasBinding, err := service.ResolveBindingForOperation(ctx, principal, operation)
	if err != nil {
		return decision, ErrNeutralDenied
	}
	decision.Binding = binding
	decision.ScopeDigest = AuthorizationScopeDigest(principal, binding, operation)
	if !hasBinding {
		return decision, ErrNeutralDenied
	}
	decision.Allowed = true
	return decision, nil
}

func workspaceAdminAllows(operation string) bool {
	switch operation {
	case OperationGetSummary, OperationGetTaxonomy, OperationGetProviderConfiguration, OperationSearchItems, OperationGetDraft, OperationGetHistory,
		OperationGetInspection, OperationGetCurrentInspection, OperationGetInspectionQuestionPage, OperationGetFinding, OperationGetCAPEvidence,
		OperationGetRoleHistory, OperationResetGeneration:
		return true
	default:
		return false
	}
}

func bindingAllowsWorkspaceOperation(binding preprod.AuthorityBinding, principal identity.Principal, operation string) bool {
	switch operation {
	case OperationGetSummary, OperationGetTaxonomy, OperationGetProviderConfiguration, OperationSearchItems, OperationGetDraft, OperationGetHistory:
		return principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
	case OperationGetSimulationSetup, OperationGetCurrentRecommendation:
		return principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
	case OperationPreviewBatch, OperationExecuteBatch, OperationRetain, OperationReclassify, OperationAddTopic, OperationRemoveTopic, OperationResolve, OperationInclude, OperationExclude, OperationDefer, OperationAddCandidate, OperationReword, OperationMarkReady:
		return principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
	case OperationCreateRecommendation, OperationCreateInspection:
		return principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
	case OperationGetInspection, OperationGetCurrentInspection, OperationGetInspectionQuestionPage, OperationGetFinding, OperationGetCAPEvidence:
		return bindingHasWorkspaceRole(binding, principal, "LIFECYCLE_READ")
	case OperationGetRoleHistory:
		return !principal.HasRole(identity.RoleAuditee) && bindingHasWorkspaceRole(binding, principal, "CAA_HISTORY")
	case OperationStartInspection, OperationRecordResponse, OperationCreateFinding, OperationSubmitChecklist:
		return principal.HasRole(identity.RoleInspector) && bindingHasWorkspaceRole(binding, principal, "INSPECTOR")
	case OperationReopenChecklist, OperationVerifyEvidence:
		return principal.HasRole(identity.RoleInspector, identity.RoleLeadInspector) && bindingHasWorkspaceRole(binding, principal, "INSPECTION_EXECUTION")
	case OperationReturnFinding, OperationDismissFinding, OperationConvertFinding:
		return principal.HasRole(identity.RoleLeadInspector) && bindingHasWorkspaceRole(binding, principal, "LEAD")
	case OperationSubmitCAP, OperationSubmitEvidence:
		return principal.HasRole(identity.RoleAuditee) && bindingHasWorkspaceRole(binding, principal, "AUDITEE")
	case OperationReviewCAP:
		return principal.HasRole(identity.RoleLeadInspector, identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "CAA_REVIEWER")
	case OperationAuthorizedClose:
		return principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
	default:
		return false
	}
}

func bindingMatchesLifecycleScope(binding preprod.AuthorityBinding, aggregate LifecycleAggregate) bool {
	return binding.Active &&
		(binding.ProviderScopeID != "" && (workspaceOrganizationMatchesPrincipal(binding.OrganizationID, aggregate.OrganizationID) || binding.ProviderScopeID == aggregate.ProviderScopeID)) &&
		binding.DepartmentID == aggregate.Inspector.DepartmentID &&
		binding.OrganizationalUnitID == aggregate.Inspector.OrganizationalUnitID &&
		binding.ProviderScopeID == aggregate.ProviderScopeID
}

// bindingMatchesLifecycleObject adds a subject/role relationship to the
// aggregate scope check. Manager bindings are intentionally aggregate-scoped:
// the lifecycle aggregate has no manager assignment pin, so the exact active
// department, organizational-unit, and provider-scope binding is the durable
// authorization boundary for that role.
func bindingMatchesLifecycleObject(binding preprod.AuthorityBinding, aggregate LifecycleAggregate, principal identity.Principal) bool {
	if !bindingMatchesLifecycleScope(binding, aggregate) || binding.SubjectID == "" || binding.SubjectID != principal.SubjectID {
		return false
	}
	if principal.HasRole(identity.RoleDepartmentManager) {
		return true
	}
	if principal.HasRole(identity.RoleInspector) && principal.SubjectID == aggregate.Inspector.SubjectID {
		return true
	}
	if principal.HasRole(identity.RoleLeadInspector) && principal.SubjectID == aggregate.Lead.SubjectID {
		return true
	}
	return principal.HasRole(identity.RoleAuditee) && principal.SubjectID == aggregate.Auditee.SubjectID
}

func AuthorizationScopeDigest(principal identity.Principal, binding preprod.AuthorityBinding, operation string) string {
	roles := append([]string(nil), binding.OperationRoles...)
	for index := range roles {
		roles[index] = strings.TrimSpace(roles[index])
	}
	sort.Strings(roles)
	payload := struct {
		SubjectID            string   `json:"subjectId"`
		OrganizationID       string   `json:"organizationId"`
		GenerationID         string   `json:"generationId"`
		BindingID            string   `json:"bindingId"`
		MembershipID         string   `json:"membershipId"`
		MembershipVersion    int      `json:"membershipVersion"`
		MembershipSlot       string   `json:"membershipSlot"`
		DepartmentID         string   `json:"departmentId"`
		OrganizationalUnitID string   `json:"organizationalUnitId"`
		ProviderScopeID      string   `json:"providerScopeId"`
		Operation            string   `json:"operation"`
		OperationRoles       []string `json:"operationRoles"`
		BindingDigest        string   `json:"bindingDigest"`
	}{
		principal.SubjectID, principal.OrganizationID, binding.GenerationID, binding.BindingID, binding.MembershipID,
		binding.MembershipVersion, binding.MembershipSlot, binding.DepartmentID, binding.OrganizationalUnitID,
		binding.ProviderScopeID, operation, roles, binding.BindingDigest,
	}
	data, _ := json.Marshal(payload)
	digest := sha256.Sum256(append([]byte("AGA-DEMO-WORKSPACE-AUTHORITY-SCOPE-V1\n"), data...))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func isCAAOrganization(value string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	return value == "CAA" || value == "AGA-DEMO-CAA"
}

// workspaceOrganizationMatchesPrincipal keeps the synthetic workspace
// organization identity separate from the source OIDC organization identity.
// The connected fixture intentionally binds source subjects into the sealed
// workspace graph; only the two explicit CAA labels are aliases here.
func workspaceOrganizationMatchesPrincipal(principalOrganization, workspaceOrganization string) bool {
	if principalOrganization == workspaceOrganization {
		return true
	}
	return isCAAOrganization(principalOrganization) && isCAAOrganization(workspaceOrganization)
}

func bindingHasWorkspaceRole(binding preprod.AuthorityBinding, principal identity.Principal, required string) bool {
	if !binding.Active || !workspaceOrganizationMatchesPrincipal(principal.OrganizationID, binding.OrganizationID) {
		return false
	}
	if required == "CAA_REVIEWER" && strings.EqualFold(strings.TrimSpace(binding.MembershipSlot), "CAA_REVIEWER_MEMBERSHIP") {
		return true
	}
	for _, role := range binding.OperationRoles {
		normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(role), "-", "_"))
		if required == "" {
			if normalized != "" {
				return true
			}
			continue
		}
		if normalized == required || normalized == "MANAGER" && (required == "MANAGER" || required == "LIFECYCLE_READ" || required == "CAA_HISTORY") || normalized == "DEPARTMENT_MANAGER" && (required == "MANAGER" || required == "LIFECYCLE_READ" || required == "CAA_HISTORY") || normalized == "LEAD_INSPECTOR" && (required == "LEAD" || required == "CAA_HISTORY" || required == "LIFECYCLE_READ" || required == "INSPECTION_EXECUTION") || normalized == "INSPECTOR" && (required == "INSPECTOR" || required == "INSPECTION_EXECUTION" || required == "LIFECYCLE_READ" || required == "CAA_HISTORY") || normalized == "AUDITEE" && (required == "AUDITEE" || required == "LIFECYCLE_READ") || normalized == "CAA_REVIEWER" && (required == "CAA_REVIEWER" || required == "CAA_HISTORY" || required == "LIFECYCLE_READ") {
			return true
		}
	}
	return false
}
