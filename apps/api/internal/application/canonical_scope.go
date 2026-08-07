package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/questioncatalog"
	"github.com/jackc/pgx/v5"
)

// CanonicalScopeFacts is the server-owned identity and selection pin for a
// planning scope. It contains references and digests only; question bodies
// remain exclusively in question_versions.
type CanonicalScopeFacts struct {
	ScopeID           string
	CatalogID         string
	CatalogVersion    string
	UsageClass        string
	ProviderScopeID   string
	RegulatedTargetID string
	SelectedCount     int
	SelectionDigest   string
}

type canonicalQueryRow interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func numberValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func selectedIDs(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		ids := make([]string, 0, len(typed))
		for _, item := range typed {
			if id, ok := item.(string); ok {
				ids = append(ids, strings.TrimSpace(id))
			}
		}
		return ids
	default:
		return nil
	}
}

// NormalizeCanonicalPlanningValues returns a copy with the canonical fields
// normalized and a server-generated scope draft identity when necessary.
func NormalizeCanonicalPlanningValues(values map[string]any, draftID string) map[string]any {
	copy := make(map[string]any, len(values)+1)
	for key, value := range values {
		copy[key] = value
	}
	for _, key := range []string{"organizationId", "applicationType", "catalogVersion", "scopeDraftId", "selectionDigest", "providerScopeId", "regulatedTargetId", "noticePolicy", "currency"} {
		if value, ok := copy[key]; ok {
			copy[key] = stringValue(value)
		}
	}
	if stringValue(copy["catalogVersion"]) != "" && stringValue(copy["scopeDraftId"]) == "" {
		copy["scopeDraftId"] = "scope-draft-" + strings.TrimSpace(draftID)
	}
	return copy
}

// ValidateCanonicalScopeMap verifies catalog usage, active provider scope /
// target compatibility, department authority, and exact selected identities.
// For an incomplete draft an empty selection is allowed; a complete submit
// requires a non-empty selection and digest.
func ValidateCanonicalScopeMap(
	ctx context.Context,
	tx canonicalQueryRow,
	actor identity.Principal,
	values map[string]any,
	complete bool,
) (CanonicalScopeFacts, error) {
	catalogVersion := stringValue(values["catalogVersion"])
	if catalogVersion == "" && stringValue(values["scopeDraftId"]) == "" && stringValue(values["providerScopeId"]) == "" && stringValue(values["regulatedTargetId"]) == "" {
		return CanonicalScopeFacts{}, nil
	}
	providerScopeID := stringValue(values["providerScopeId"])
	regulatedTargetID := stringValue(values["regulatedTargetId"])
	if catalogVersion == "" || providerScopeID == "" || regulatedTargetID == "" {
		return CanonicalScopeFacts{}, fmt.Errorf("%w: catalog, provider scope, and regulated target are required", ErrInvalid)
	}
	var facts CanonicalScopeFacts
	var profileName, status string
	if err := tx.QueryRow(ctx, `
		SELECT id, catalog_version, usage_class, status, profile_name
		FROM canonical_question_catalogs WHERE catalog_version = $1
	`, catalogVersion).Scan(&facts.CatalogID, &facts.CatalogVersion, &facts.UsageClass, &status, &profileName); err != nil {
		if err == pgx.ErrNoRows {
			return CanonicalScopeFacts{}, ErrNotFound
		}
		return CanonicalScopeFacts{}, err
	}
	if status != "SEALED" {
		return CanonicalScopeFacts{}, fmt.Errorf("%w: catalog is not sealed", ErrConflict)
	}
	if facts.UsageClass != string(questioncatalog.UsageClassGovernedOperational) && facts.UsageClass != string(questioncatalog.UsageClassPreprodExercise) {
		return CanonicalScopeFacts{}, fmt.Errorf("%w: unsupported catalog usage class", ErrInvalid)
	}
	if facts.UsageClass == string(questioncatalog.UsageClassPreprodExercise) && profileName != "aga-preprod" {
		return CanonicalScopeFacts{}, fmt.Errorf("%w: exercise catalog requires the dedicated disposable aga-preprod profile", ErrForbidden)
	}
	var organizationID, targetOrganizationID, scopeStatus string
	if err := tx.QueryRow(ctx, `
		SELECT scope.organization_id, COALESCE(target.organization_id, target.owner_organization_id), scope.status
		FROM organization_service_provider_scopes scope
		JOIN regulated_targets target ON target.id = $2
		WHERE scope.id = $1
		  AND (scope.primary_target_id = target.id OR EXISTS (
			SELECT 1 FROM organization_service_provider_scope_targets linked
			WHERE linked.organization_service_provider_scope_id = scope.id
			  AND linked.regulated_target_id = target.id
		  ))
	`, providerScopeID, regulatedTargetID).Scan(&organizationID, &targetOrganizationID, &scopeStatus); err != nil {
		if err == pgx.ErrNoRows {
			return CanonicalScopeFacts{}, fmt.Errorf("%w: provider scope and target are not a compatible server-authorized pair", ErrForbidden)
		}
		return CanonicalScopeFacts{}, err
	}
	if scopeStatus != "ACTIVE" || (strings.TrimSpace(targetOrganizationID) != "" && targetOrganizationID != organizationID) {
		return CanonicalScopeFacts{}, fmt.Errorf("%w: provider scope is not active for the regulated target", ErrForbidden)
	}
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return CanonicalScopeFacts{}, fmt.Errorf("%w: Department Manager authority is required", ErrForbidden)
	}
	var departmentMatch bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM caa_department_memberships membership
			JOIN service_provider_unit_responsibilities responsibility
			  ON responsibility.organizational_unit_id = membership.organizational_unit_id
			JOIN organization_service_provider_scopes scope
			  ON scope.service_provider_type_id = responsibility.service_provider_type_id
			WHERE membership.subject_id = $1 AND scope.id = $2
			  AND membership.status = 'ACTIVE'
			  AND membership.effective_from <= CURRENT_DATE
			  AND (membership.effective_to IS NULL OR membership.effective_to > CURRENT_DATE)
		)
	`, actor.SubjectID, providerScopeID).Scan(&departmentMatch); err != nil {
		return CanonicalScopeFacts{}, err
	}
	if !departmentMatch {
		return CanonicalScopeFacts{}, fmt.Errorf("%w: provider scope is outside the manager department authority", ErrForbidden)
	}
	if organizationID != stringValue(values["organizationId"]) {
		return CanonicalScopeFacts{}, fmt.Errorf("%w: organization does not match the provider scope", ErrConflict)
	}
	facts.ScopeID = stringValue(values["scopeDraftId"])
	facts.ProviderScopeID = providerScopeID
	facts.RegulatedTargetID = regulatedTargetID
	ids := selectedIDs(values["selectedQuestionVersionIds"])
	if complete && (facts.ScopeID == "" || stringValue(values["selectionDigest"]) == "" || len(ids) == 0) {
		return CanonicalScopeFacts{}, fmt.Errorf("%w: exact question selection is required before Finance submission", ErrInvalid)
	}
	if len(ids) > 500 {
		return CanonicalScopeFacts{}, fmt.Errorf("%w: question selection exceeds the bounded batch", ErrInvalid)
	}
	if len(ids) > 0 {
		seen := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			if id == "" {
				return CanonicalScopeFacts{}, fmt.Errorf("%w: question version identity is empty", ErrInvalid)
			}
			if _, exists := seen[id]; exists {
				return CanonicalScopeFacts{}, fmt.Errorf("%w: question selection contains a duplicate identity", ErrInvalid)
			}
			seen[id] = struct{}{}
		}
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM canonical_question_catalog_memberships
			WHERE catalog_id = $1 AND usage_class = $2 AND question_version_id = ANY($3::text[])
		`, facts.CatalogID, facts.UsageClass, ids).Scan(&facts.SelectedCount); err != nil {
			return CanonicalScopeFacts{}, err
		}
		if facts.SelectedCount != len(ids) {
			return CanonicalScopeFacts{}, fmt.Errorf("%w: selected question is outside the sealed catalog", ErrInvalid)
		}
		facts.SelectionDigest = questioncatalog.SelectionDigest(ids)
		if supplied := stringValue(values["selectionDigest"]); supplied != "" && supplied != facts.SelectionDigest {
			return CanonicalScopeFacts{}, fmt.Errorf("%w: selection digest does not match the exact question identities", ErrConflict)
		}
	}
	return facts, nil
}

// ValidateCanonicalScopeDraft binds a saved selection to the immutable scope
// identity created with the planning draft and rejects uncommitted changes.
func ValidateCanonicalScopeDraft(ctx context.Context, tx canonicalQueryRow, draftID string, facts CanonicalScopeFacts) error {
	if facts.ScopeID == "" {
		return nil
	}
	var catalogID, version, usage, providerScopeID, targetID, digest string
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT scope.catalog_id, catalog.catalog_version, scope.usage_class,
		       scope.provider_scope_id, scope.regulated_target_id,
		       scope.selected_question_count, scope.selection_digest
		FROM canonical_audit_scope_drafts scope
		JOIN canonical_question_catalogs catalog ON catalog.id = scope.catalog_id
		WHERE scope.id = $1 AND scope.planning_intake_draft_id = $2 FOR UPDATE
	`, facts.ScopeID, draftID).Scan(&catalogID, &version, &usage, &providerScopeID, &targetID, &count, &digest); err != nil {
		if err == pgx.ErrNoRows {
			return ErrConflict
		}
		return err
	}
	if catalogID != facts.CatalogID || version != facts.CatalogVersion || usage != facts.UsageClass || providerScopeID != facts.ProviderScopeID || targetID != facts.RegulatedTargetID {
		return fmt.Errorf("%w: canonical scope identity changed", ErrConflict)
	}
	if facts.SelectedCount > 0 && (count != facts.SelectedCount || digest != facts.SelectionDigest) {
		return fmt.Errorf("%w: selection must be committed through the canonical scope CAS operation", ErrConflict)
	}
	return nil
}
