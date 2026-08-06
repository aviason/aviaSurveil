package agademoworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func (service *Service) ResolveBinding(ctx context.Context, principal identity.Principal) (preprod.AuthorityBinding, bool, error) {
	if service == nil || service.resolver == nil || strings.TrimSpace(principal.SubjectID) == "" {
		return preprod.AuthorityBinding{}, false, nil
	}
	binding, found, err := service.resolver.Resolve(ctx, principal)
	if err != nil || !found {
		return preprod.AuthorityBinding{}, found, err
	}
	if !binding.Active || binding.OrganizationID == "" || principal.OrganizationID == "" || binding.OrganizationID != principal.OrganizationID {
		return preprod.AuthorityBinding{}, false, nil
	}
	if binding.DepartmentID == "" || binding.OrganizationalUnitID == "" || len(binding.OperationRoles) == 0 {
		return preprod.AuthorityBinding{}, false, nil
	}
	return binding, true, nil
}

func (service *Service) HasBroadAuthority(ctx context.Context, principal identity.Principal) bool {
	if principal.SubjectID == "" {
		return false
	}
	if principal.HasRole(identity.RoleAdmin) && isCAAOrganization(principal.OrganizationID) {
		return true
	}
	binding, found, err := service.ResolveBinding(ctx, principal)
	return err == nil && found && bindingHasWorkspaceRole(binding, principal, "")
}

func (service *Service) Authorize(ctx context.Context, principal identity.Principal, operation string) (AuthorizationDecision, error) {
	decision := AuthorizationDecision{Operation: operation, Organization: principal.OrganizationID}
	if strings.TrimSpace(principal.SubjectID) == "" || strings.TrimSpace(operation) == "" {
		return decision, ErrNeutralDenied
	}
	decision.IsAdmin = principal.HasRole(identity.RoleAdmin) && isCAAOrganization(principal.OrganizationID)
	binding, hasBinding, err := service.ResolveBinding(ctx, principal)
	if err != nil {
		return decision, ErrNeutralDenied
	}
	decision.Binding = binding
	decision.ScopeDigest = AuthorizationScopeDigest(principal, binding, operation)

	allowed := false
	switch operation {
	case OperationGetSummary, OperationGetTaxonomy, OperationGetProviderConfiguration, OperationSearchItems, OperationGetDraft, OperationGetHistory:
		allowed = decision.IsAdmin || (hasBinding && principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER"))
	case OperationGetSimulationSetup, OperationGetCurrentRecommendation:
		allowed = hasBinding && principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
	case OperationPreviewBatch, OperationExecuteBatch, OperationRetain, OperationReclassify, OperationAddTopic, OperationRemoveTopic, OperationResolve, OperationInclude, OperationExclude, OperationDefer, OperationAddCandidate, OperationReword, OperationMarkReady:
		allowed = hasBinding && principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
	case OperationCreateRecommendation, OperationCreateInspection:
		allowed = hasBinding && principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
	case OperationGetInspection, OperationGetCurrentInspection, OperationGetInspectionQuestionPage, OperationGetFinding, OperationGetCAPEvidence:
		allowed = decision.IsAdmin || (hasBinding && bindingHasWorkspaceRole(binding, principal, "LIFECYCLE_READ"))
	case OperationGetRoleHistory:
		allowed = !principal.HasRole(identity.RoleAuditee) && (decision.IsAdmin || (hasBinding && bindingHasWorkspaceRole(binding, principal, "CAA_HISTORY")))
	case OperationStartInspection, OperationRecordResponse, OperationCreateFinding, OperationSubmitChecklist:
		allowed = hasBinding && principal.HasRole(identity.RoleInspector) && bindingHasWorkspaceRole(binding, principal, "INSPECTOR")
	case OperationReopenChecklist, OperationVerifyEvidence:
		allowed = hasBinding && (principal.HasRole(identity.RoleInspector) || principal.HasRole(identity.RoleLeadInspector)) && bindingHasWorkspaceRole(binding, principal, "INSPECTION_EXECUTION")
	case OperationReturnFinding, OperationDismissFinding, OperationConvertFinding:
		allowed = hasBinding && principal.HasRole(identity.RoleLeadInspector) && bindingHasWorkspaceRole(binding, principal, "LEAD")
	case OperationSubmitCAP, OperationSubmitEvidence:
		allowed = hasBinding && principal.HasRole(identity.RoleAuditee) && bindingHasWorkspaceRole(binding, principal, "AUDITEE")
	case OperationReviewCAP:
		allowed = hasBinding && (principal.HasRole(identity.RoleLeadInspector) || principal.HasRole(identity.RoleDepartmentManager)) && bindingHasWorkspaceRole(binding, principal, "CAA_REVIEWER")
	case OperationAuthorizedClose:
		allowed = hasBinding && principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
	case OperationResetGeneration:
		allowed = decision.IsAdmin
	default:
		return decision, ErrNeutralDenied
	}
	if !allowed {
		return decision, ErrNeutralDenied
	}
	decision.Allowed = true
	return decision, nil
}

func AuthorizationScopeDigest(principal identity.Principal, binding preprod.AuthorityBinding, operation string) string {
	roles := append([]string(nil), binding.OperationRoles...)
	for index := range roles {
		roles[index] = strings.TrimSpace(roles[index])
	}
	payload := struct {
		SubjectID            string   `json:"subjectId"`
		OrganizationID       string   `json:"organizationId"`
		BindingID            string   `json:"bindingId"`
		MembershipSlot       string   `json:"membershipSlot"`
		DepartmentID         string   `json:"departmentId"`
		OrganizationalUnitID string   `json:"organizationalUnitId"`
		Operation            string   `json:"operation"`
		OperationRoles       []string `json:"operationRoles"`
		BindingDigest        string   `json:"bindingDigest"`
	}{principal.SubjectID, principal.OrganizationID, binding.BindingID, binding.MembershipSlot, binding.DepartmentID, binding.OrganizationalUnitID, operation, roles, binding.BindingDigest}
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
	if !binding.Active || binding.OrganizationID != principal.OrganizationID {
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
