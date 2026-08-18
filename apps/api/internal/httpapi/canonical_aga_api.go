package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/checklistgovernance"
	"github.com/aviason/aviaSurveil/internal/httpapi/generated"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
	"github.com/aviason/aviaSurveil/internal/questioncatalog"
	"github.com/aviason/aviaSurveil/internal/regulatory"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

const (
	canonicalCatalogPageSize             = 25
	canonicalCatalogMaximumPageSize      = 100
	canonicalCatalogMaximumSelectionSize = 2000
)

const (
	canonicalCatalogProjectionFull      = "full"
	canonicalCatalogProjectionSelection = "selection"
)

var canonicalChecklistFocusCodes = map[string]struct{}{
	"CHANGE_APPROVAL":            {},
	"DOCUMENT_AND_RECORD_REVIEW": {},
	"FOLLOW_UP":                  {},
	"INITIAL_CERTIFICATION":      {},
	"ON_SITE_INSPECTION":         {},
	"PERIODIC_SURVEILLANCE":      {},
	"RENEWAL":                    {},
	"SPECIAL_PURPOSE":            {},
}

func parseCanonicalExecutionType(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	return application.CanonicalExecutionType(value)
}

func parseCanonicalChecklistFocus(value string) ([]string, error) {
	seen := map[string]struct{}{}
	values := []string{}
	for _, raw := range strings.Split(value, ",") {
		focus := strings.ToUpper(strings.TrimSpace(raw))
		if focus == "" {
			continue
		}
		if _, ok := canonicalChecklistFocusCodes[focus]; !ok {
			return nil, fmt.Errorf("%w: unsupported checklist focus", application.ErrInvalid)
		}
		if _, ok := seen[focus]; ok {
			continue
		}
		seen[focus] = struct{}{}
		values = append(values, focus)
	}
	return values, nil
}

func validQuestionReviewReason(value string) bool {
	switch strings.TrimSpace(value) {
	case "MANAGER_SCOPE_DECISION", "SOURCE_MAPPING_REQUIRED", "CLASSIFICATION_EXPERT_REVIEW", "MANAGER_EXACT_RESOLUTION":
		return true
	default:
		return false
	}
}

func (api *CanonicalAPI) requireCanonicalCatalogActor(writer http.ResponseWriter, request *http.Request, mutation bool) (identity.Principal, bool) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return identity.Principal{}, false
	}
	if mutation {
		if !actor.HasRole(identity.RoleDepartmentManager) {
			api.respond(writer, nil, application.ErrForbidden)
			return identity.Principal{}, false
		}
	} else if !actor.HasRole(identity.RoleDepartmentManager) {
		api.respond(writer, nil, application.ErrNotFound)
		return identity.Principal{}, false
	}
	return actor, true
}

func parseQuestionUsageClass(value string) (questioncatalog.UsageClass, error) {
	switch questioncatalog.UsageClass(strings.TrimSpace(value)) {
	case questioncatalog.UsageClassGovernedOperational:
		return questioncatalog.UsageClassGovernedOperational, nil
	default:
		return "", fmt.Errorf("%w: invalid usage class", application.ErrInvalid)
	}
}

func parseCanonicalCatalogProjection(value string) (string, error) {
	projection := strings.ToLower(strings.TrimSpace(value))
	if projection == "" {
		return canonicalCatalogProjectionFull, nil
	}
	if projection != canonicalCatalogProjectionFull && projection != canonicalCatalogProjectionSelection {
		return "", fmt.Errorf("%w: unsupported catalog projection", application.ErrInvalid)
	}
	return projection, nil
}

func parseQuestionReviewMode(value generated.QuestionReviewMode) (questioncatalog.UsageClass, error) {
	return parseQuestionUsageClass(string(value))
}

func parseCatalogCursor(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid catalog cursor", application.ErrInvalid)
	}
	offset, err := strconv.Atoi(string(decoded))
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("%w: invalid catalog cursor", application.ErrInvalid)
	}
	return offset, nil
}

func encodeCatalogCursor(offset int) *string {
	if offset < 0 {
		return nil
	}
	value := base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
	return &value
}

func emptyCanonicalCatalogFacets() generated.CanonicalQuestionCatalogFacets {
	return generated.CanonicalQuestionCatalogFacets{
		Forms:                []generated.CanonicalQuestionCatalogFacetOption{},
		Domains:              []generated.CanonicalQuestionCatalogFacetOption{},
		Topics:               []generated.CanonicalQuestionCatalogFacetOption{},
		RiskTiers:            []generated.CanonicalQuestionCatalogFacetOption{},
		ChecklistFocuses:     []generated.CanonicalQuestionCatalogFacetOption{},
		RecommendationStates: []generated.CanonicalQuestionCatalogFacetOption{},
	}
}

// requireCanonicalScopeOwner is the horizontal-authorization boundary for a
// planning scope. Scope identifiers are opaque but not secret; every read or
// mutation that projects selected questions therefore binds the row to the
// creating Department Manager (and its still-live planning draft) inside the
// same database authority boundary.
func (api *CanonicalAPI) requireCanonicalScopeOwner(ctx context.Context, scopeID, subjectID string) error {
	if api.pool == nil || strings.TrimSpace(scopeID) == "" || strings.TrimSpace(subjectID) == "" {
		return application.ErrNotFound
	}
	var owner string
	if err := api.pool.QueryRow(ctx, `
		SELECT scope.created_by_subject_id
		FROM canonical_audit_scope_drafts scope
		JOIN planning_intake_drafts draft ON draft.id = scope.planning_intake_draft_id
		WHERE scope.id = $1
		  AND draft.tombstoned_at IS NULL
		  AND scope.created_by_subject_id = $2
	`, scopeID, subjectID).Scan(&owner); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		return err
	}
	if owner != subjectID {
		return application.ErrNotFound
	}
	return nil
}

// listCanonicalAuditScopeOptions is the server-owned selector for New Audit.
// The browser never invents an organization, provider scope, target, or
// catalog identity; all options are filtered through the current manager's
// effective department authority and the sealed canonical catalog.
func (api *CanonicalAPI) listCanonicalAuditScopeOptions(writer http.ResponseWriter, request *http.Request) {
	actor, ok := api.requireCanonicalCatalogActor(writer, request, true)
	if !ok {
		return
	}
	usageValue := strings.TrimSpace(request.URL.Query().Get("usageClass"))
	if usageValue == "" {
		usageValue = string(questioncatalog.UsageClassGovernedOperational)
	}
	usage, err := parseQuestionUsageClass(usageValue)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	catalogVersion := strings.TrimSpace(request.URL.Query().Get("catalogVersion"))
	offset, err := parseCatalogCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	limit := canonicalCatalogPageSize
	if raw := request.URL.Query().Get("limit"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > canonicalCatalogPageSize {
			api.respond(writer, nil, fmt.Errorf("%w: scope option limit must be between 1 and 25", application.ErrInvalid))
			return
		}
		limit = parsed
	}
	if api.pool == nil {
		api.respond(writer, generated.CanonicalAuditScopeOptionPage{Items: []generated.CanonicalAuditScopeOption{}, NextCursor: nil}, nil)
		return
	}
	if strings.EqualFold(strings.TrimSpace(request.URL.Query().Get("review")), "true") {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	if catalogVersion == "" {
		// Governed discovery is server-owned: resolve the newest sealed
		// operational catalog instead of requiring the browser to guess an
		// immutable version before it can render the selector.
		if err := api.pool.QueryRow(request.Context(), `
			SELECT COALESCE((
				SELECT catalog_version
				FROM canonical_question_catalogs
				WHERE usage_class = $1 AND status = 'SEALED'
				  AND source_origin = 'IMPORTED_APPROVED_SOURCE'
				ORDER BY created_at DESC, catalog_version DESC
				LIMIT 1
			), '')
		`, string(usage)).Scan(&catalogVersion); err != nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
		if catalogVersion == "" {
			api.respond(writer, generated.CanonicalAuditScopeOptionPage{Items: []generated.CanonicalAuditScopeOption{}, NextCursor: nil}, nil)
			return
		}
	}
	rows, err := api.pool.Query(request.Context(), `
		SELECT organization.id, organization.legal_name,
		       scope.id, target.id, provider.id, provider.label, scope.status,
		       COALESCE(target.external_identifier, organization.legal_name || ' regulated target'),
		       catalog.catalog_version, catalog.usage_class,
		   CASE
		     WHEN provider.id = 'AIR_OPERATOR'
		       THEN ARRAY['RAMP_INSPECTION','CABIN_INSPECTION','CHANGE_APPROVAL','DOCUMENT_AND_RECORD_REVIEW','FOLLOW_UP','INITIAL_CERTIFICATION','ON_SITE_INSPECTION','PERIODIC_SURVEILLANCE','RENEWAL','SPECIAL_PURPOSE']::text[]
		     WHEN provider.id = 'AERODROME_OPERATOR'
		       THEN ARRAY['RAMP_INSPECTION','CHANGE_APPROVAL','DOCUMENT_AND_RECORD_REVIEW','FOLLOW_UP','INITIAL_CERTIFICATION','ON_SITE_INSPECTION','PERIODIC_SURVEILLANCE','RENEWAL','SPECIAL_PURPOSE']::text[]
		     WHEN provider.id = 'FUEL_PROVIDER'
		       THEN ARRAY['CHANGE_APPROVAL','DOCUMENT_AND_RECORD_REVIEW','FOLLOW_UP','INITIAL_CERTIFICATION','ON_SITE_INSPECTION','PERIODIC_SURVEILLANCE','RENEWAL','SPECIAL_PURPOSE']::text[]
		     ELSE ARRAY[]::text[]
	           END
		FROM (
			SELECT DISTINCT ON (root_id) *
			FROM organization_service_provider_scopes
			WHERE effective_from <= CURRENT_DATE
			ORDER BY root_id, effective_from DESC, id DESC
		) scope
		JOIN organizations organization ON organization.id = scope.organization_id
		JOIN service_provider_types provider ON provider.id = scope.service_provider_type_id
		JOIN regulated_targets target
		  ON target.id = scope.primary_target_id
		  OR EXISTS (
			  SELECT 1
			  FROM organization_service_provider_scope_targets linked_target
			  WHERE linked_target.organization_service_provider_scope_id = scope.id
			    AND linked_target.regulated_target_id = target.id
		  )
		JOIN (
			SELECT DISTINCT ON (root_id) *
			FROM caa_department_memberships
			WHERE subject_id = $1
			  AND effective_from <= CURRENT_DATE
			ORDER BY root_id, effective_from DESC, id DESC
		) membership
		  ON membership.subject_id = $1
		 AND membership.membership_role = 'DEPARTMENT_MANAGER'
		 AND membership.status = 'ACTIVE'
		 AND (membership.effective_to IS NULL OR membership.effective_to > CURRENT_DATE)
		JOIN LATERAL (
			SELECT status FROM caa_department_status_facts
			WHERE department_id = membership.department_id AND effective_from <= CURRENT_DATE
			ORDER BY effective_from DESC, id DESC LIMIT 1
		) department_status ON department_status.status = 'ACTIVE'
		JOIN LATERAL (
			SELECT status FROM caa_organizational_unit_status_facts
			WHERE organizational_unit_id = membership.organizational_unit_id AND effective_from <= CURRENT_DATE
			ORDER BY effective_from DESC, id DESC LIMIT 1
		) unit_status ON unit_status.status = 'ACTIVE'
		JOIN service_provider_unit_responsibilities responsibility
		  ON responsibility.organizational_unit_id = membership.organizational_unit_id
		 AND responsibility.service_provider_type_id = scope.service_provider_type_id
		JOIN canonical_question_catalogs catalog
		  ON ($2 = '' OR catalog.catalog_version = $2)
		 AND catalog.usage_class = $3
		 AND catalog.status = 'SEALED'
			AND catalog.source_origin = 'IMPORTED_APPROVED_SOURCE'
			 AND NOT EXISTS (
			SELECT 1
			FROM canonical_question_catalog_memberships catalog_membership
			WHERE catalog_membership.catalog_id = catalog.id
			  AND NOT EXISTS (
				SELECT 1
				FROM canonical_question_catalog_applicabilities applicability
				WHERE applicability.catalog_id = catalog_membership.catalog_id
				  AND applicability.question_version_id = catalog_membership.question_version_id
				  AND applicability.provider_scope_id = scope.id
				  AND applicability.regulated_target_id = target.id
				  AND applicability.status = 'ELIGIBLE'
			  )
			 )
		WHERE scope.status = 'ACTIVE'
		  AND provider.id IN ('AIR_OPERATOR', 'AERODROME_OPERATOR', 'FUEL_PROVIDER')
		  AND (scope.effective_to IS NULL OR scope.effective_to > CURRENT_DATE)
		  AND (target.organization_id IS NULL OR target.organization_id = scope.organization_id)
		  AND (target.owner_organization_id IS NULL OR target.owner_organization_id = scope.organization_id)
		  AND organization.tombstoned_at IS NULL
		ORDER BY organization.legal_name, provider.label, scope.id, target.id
		LIMIT $4 OFFSET $5
	`, actor.SubjectID, catalogVersion, string(usage), limit+1, offset)
	if err != nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	defer rows.Close()
	items := make([]generated.CanonicalAuditScopeOption, 0, limit)
	for rows.Next() {
		var item generated.CanonicalAuditScopeOption
		if err := rows.Scan(
			&item.OrganizationId, &item.OrganizationName, &item.ProviderScopeId,
			&item.RegulatedTargetId, &item.ProviderTypeId, &item.ProviderTypeLabel,
			&item.ScopeStatus, &item.TargetLabel, &item.CatalogVersion, &item.UsageClass,
			&item.InspectionTypes,
		); err != nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = encodeCatalogCursor(offset + limit)
	}
	api.respond(writer, generated.CanonicalAuditScopeOptionPage{Items: items, NextCursor: next}, nil)
}

// listGovernedReviewScopeOptions exposes the candidate aggregate directly
// before publication.  Operational New Audit discovery never calls this
// purpose-bound path; it continues to require a sealed published catalog.
func (api *CanonicalAPI) listGovernedReviewScopeOptions(writer http.ResponseWriter, request *http.Request, subjectID, catalogVersion string, limit, offset int) {
	rows, err := api.pool.Query(request.Context(), `
		SELECT organization.id,organization.legal_name,scope.id,target.id,
		       provider.id,provider.label,scope.status,
		       COALESCE(target.external_identifier,organization.legal_name || ' regulated target'),
		       'candidate:' || candidate.id,'GOVERNED_OPERATIONAL',
		       ARRAY[CASE WHEN run.inspection_type IN ('RAMP','CABIN','RAMP_INSPECTION','CABIN_INSPECTION') THEN run.inspection_type ELSE 'RAMP_INSPECTION' END]::text[]
		FROM template_draft_versions candidate
		JOIN regulatory_generation_runs run ON run.id=candidate.generation_run_id
		JOIN regulatory_generation_run_scope_facts fact ON fact.generation_run_id=run.id
		JOIN organization_service_provider_scopes scope ON scope.id=fact.organization_service_provider_scope_id
		JOIN organizations organization ON organization.id=scope.organization_id
		JOIN service_provider_types provider ON provider.id=scope.service_provider_type_id
		JOIN regulated_targets target ON target.id=fact.regulated_target_id
		JOIN candidate_required_owner_assignments owner
		  ON owner.candidate_draft_version_id=candidate.id
		 AND owner.candidate_revision=candidate.revision
		 AND owner.candidate_content_digest=candidate.candidate_content_digest
		 AND owner.approval_required
		JOIN (
			SELECT DISTINCT ON (root_id) *
			FROM caa_department_memberships
			WHERE subject_id=$1 AND effective_from <= CURRENT_DATE
			ORDER BY root_id,effective_from DESC,id DESC
		) membership
		  ON membership.department_id=owner.department_id
		 AND membership.organizational_unit_id=owner.organizational_unit_id
		 AND membership.membership_role='DEPARTMENT_MANAGER'
		 AND membership.status='ACTIVE'
		 AND (membership.effective_to IS NULL OR membership.effective_to > CURRENT_DATE)
		JOIN LATERAL (
			SELECT status FROM caa_department_status_facts
			WHERE department_id=membership.department_id AND effective_from <= CURRENT_DATE
			ORDER BY effective_from DESC,id DESC LIMIT 1
		) department_status ON department_status.status='ACTIVE'
		JOIN LATERAL (
			SELECT status FROM caa_organizational_unit_status_facts
			WHERE organizational_unit_id=membership.organizational_unit_id AND effective_from <= CURRENT_DATE
			ORDER BY effective_from DESC,id DESC LIMIT 1
		) unit_status ON unit_status.status='ACTIVE'
		WHERE candidate.status IN ('DEPARTMENT_REVIEW','RETURNED','TECHNICALLY_APPROVED')
		  AND NOT EXISTS (SELECT 1 FROM template_draft_versions successor WHERE successor.supersedes_candidate_id=candidate.id)
		  AND ($2='' OR ('candidate:' || candidate.id)=$2)
		  AND scope.status='ACTIVE'
		  AND (scope.effective_to IS NULL OR scope.effective_to > CURRENT_DATE)
		  AND organization.tombstoned_at IS NULL
		ORDER BY organization.legal_name,provider.label,scope.id,target.id,candidate.id
		LIMIT $3 OFFSET $4`, subjectID, catalogVersion, limit+1, offset)
	if err != nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	defer rows.Close()
	items := make([]generated.CanonicalAuditScopeOption, 0, limit)
	for rows.Next() {
		var item generated.CanonicalAuditScopeOption
		if err := rows.Scan(&item.OrganizationId, &item.OrganizationName, &item.ProviderScopeId, &item.RegulatedTargetId, &item.ProviderTypeId, &item.ProviderTypeLabel, &item.ScopeStatus, &item.TargetLabel, &item.CatalogVersion, &item.UsageClass, &item.InspectionTypes); err != nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = encodeCatalogCursor(offset + limit)
	}
	api.respond(writer, generated.CanonicalAuditScopeOptionPage{Items: items, NextCursor: next}, nil)
}

type canonicalCatalogRow struct {
	CatalogVersion                 string
	ScopeID                        string
	UsageClass                     string
	QuestionID                     string
	FormCode                       string
	ProposalID                     string
	Ordinal                        int64
	Digest                         string
	SourceLocator                  string
	SourceGap                      string
	Domain                         string
	Topic                          string
	RiskBand                       string
	Prompt                         string
	ConfiguredReference            string
	ExpectedEvidence               string
	GovernedCandidateID            *string
	GovernedCandidateRevision      *int64
	GovernedCandidateContentDigest *string
	GovernedCandidateStatus        *string
	ReviewRevision                 int64
	ReviewDisposition              *string
	ReviewReason                   *string
	ReviewDomain                   *string
	ReviewTopic                    *string
	ReviewHistory                  []generated.QuestionReviewHistoryItem
	AIAdvisory                     generated.CanonicalQuestionAIAdvisory
	Recommendation                 generated.CanonicalQuestionRecommendation
	MandatoryControl               bool
}

func fallbackCanonicalQuestionRecommendation(advisory generated.CanonicalQuestionAIAdvisory) generated.CanonicalQuestionRecommendation {
	state := advisory.AdvisoryState
	if state == "" {
		state = "UNCERTAIN_SIGNAL"
	}
	classification := "ROTATIONAL_SAMPLE"
	included := state != "RECENTLY_VERIFIED" && state != "OUTSIDE_FOCUS"
	canDefer := state == "RECENTLY_VERIFIED" && advisory.RiskTier != "HIGH" && advisory.RiskTier != "UNKNOWN" && !advisory.SafetyCritical
	if advisory.SafetyCritical || advisory.RiskTier == "HIGH" {
		classification = "MANDATORY_CORE"
		included = true
		canDefer = false
	}
	if canDefer {
		classification = "DEFER_ELIGIBLE"
	}
	guardrails := []string{"MANDATORY_FLOOR_ENFORCED", "FULL_CATALOG_OVERRIDE_ALLOWED"}
	if canDefer {
		guardrails = append(guardrails, "EXPLICIT_MANAGER_DEVIATION_REQUIRED")
	}
	rationale := "Server recommendation evidence is unavailable; keep the question in the suggested scope."
	if canDefer {
		rationale = "The server marked this optional question recently verified; a manager may restore it from the full catalog."
	}
	return generated.CanonicalQuestionRecommendation{
		RecommendationState: state, Classification: classification, IncludedByDefault: included, CanDefer: canDefer,
		LastComparableResult: nil, LastComparableAuditId: nil, LastVerifiedAt: advisory.PreviouslyVerifiedAt,
		RecurrenceDueAt: advisory.RecurrenceDueAt, SignalCodes: append([]string(nil), advisory.RecommendationReasonCodes...),
		Rationale: rationale, Guardrails: guardrails,
	}
}

func canonicalCatalogEntry(row canonicalCatalogRow) generated.CanonicalQuestionCatalogEntry {
	aiAdvisory := row.AIAdvisory
	if aiAdvisory.RiskTier == "" {
		topicCodes := []string{}
		if row.Topic != "" {
			topicCodes = []string{row.Topic}
		}
		aiAdvisory = generated.CanonicalQuestionAIAdvisory{
			DomainCode:                row.Domain,
			TopicCodes:                topicCodes,
			RiskTier:                  "UNKNOWN",
			AdvisoryState:             "UNCERTAIN_SIGNAL",
			RecommendationReasonCodes: []string{"ADVISORY_DATA_UNAVAILABLE"},
			RecurrenceMonths:          12,
		}
	}
	domain := row.Domain
	topic := row.Topic
	if row.ReviewDomain != nil {
		domain = *row.ReviewDomain
	}
	if row.ReviewTopic != nil {
		topic = *row.ReviewTopic
	}
	entry := generated.CanonicalQuestionCatalogEntry{
		CatalogVersion:    row.CatalogVersion,
		UsageClass:        generated.QuestionUsageClass(row.UsageClass),
		QuestionVersionId: row.QuestionID,
		FormCode:          row.FormCode, ProposalId: row.ProposalID, Ordinal: row.Ordinal,
		QuestionDigest: row.Digest, SourceGapState: row.SourceGap,
		CanSelect:                      true,
		CanPublish:                     row.UsageClass == string(questioncatalog.UsageClassGovernedOperational) && row.GovernedCandidateID != nil && row.GovernedCandidateStatus != nil && *row.GovernedCandidateStatus == "TECHNICALLY_APPROVED",
		GovernedCandidateId:            row.GovernedCandidateID,
		GovernedCandidateRevision:      row.GovernedCandidateRevision,
		GovernedCandidateContentDigest: row.GovernedCandidateContentDigest,
		GovernedCandidateStatus:        row.GovernedCandidateStatus,
		AiAdvisory:                     aiAdvisory,
		Recommendation:                 row.Recommendation,
	}
	if entry.Recommendation.RecommendationState == "" {
		entry.Recommendation = fallbackCanonicalQuestionRecommendation(aiAdvisory)
	}
	reviewRevision := row.ReviewRevision
	entry.ReviewRevision = &reviewRevision
	if row.ReviewDisposition != nil {
		entry.ReviewDisposition = row.ReviewDisposition
	}
	if reviewDigest, err := idempotency.SemanticHash(map[string]any{
		"revision": row.ReviewRevision, "disposition": row.ReviewDisposition,
		"reason": row.ReviewReason, "domain": row.ReviewDomain, "topic": row.ReviewTopic,
	}); err == nil {
		entry.ReviewDigest = &reviewDigest
	}
	if row.SourceLocator != "" {
		entry.SourceLocator = &row.SourceLocator
	}
	if domain != "" {
		if aiAdvisory.DomainCode == "" {
			entry.ProposedDomain = &domain
		}
	}
	if topic != "" {
		if aiAdvisory.DomainCode == "" {
			entry.ProposedTopic = &topic
		}
	}
	if row.RiskBand != "" && aiAdvisory.DomainCode == "" {
		entry.ProposedRiskBand = &row.RiskBand
	}
	if row.Prompt != "" {
		entry.Prompt = &row.Prompt
	}
	if row.ConfiguredReference != "" {
		entry.ConfiguredReference = &row.ConfiguredReference
	}
	if row.ExpectedEvidence != "" {
		entry.ExpectedEvidence = &row.ExpectedEvidence
	}
	if len(row.ReviewHistory) > 0 {
		entry.ReviewHistory = row.ReviewHistory
	}
	return entry
}

// loadQuestionReviewHistory projects the bounded append-only decision register
// for the exact question/scope currently being displayed. The latest review
// fields are useful for queue density, but they are not a substitute for the
// attributed history required by the Decision file. Exercise history is
// scope-bound; governed history follows the immutable candidate root and also
// includes pre-publication events whose catalog_id is intentionally NULL.
func (api *CanonicalAPI) loadQuestionReviewHistory(ctx context.Context, row canonicalCatalogRow) ([]generated.QuestionReviewHistoryItem, error) {
	if api.pool == nil || strings.TrimSpace(row.QuestionID) == "" {
		return nil, nil
	}
	// Imported Aviation source rows have no internal review draft. Optional
	// enrichment is explicitly non-authoritative and therefore has no review
	// history to project into the operational catalog.
	return nil, nil
}

const canonicalCatalogAIProjectionJoins = `
JOIN canonical_question_catalog_ai_enrichments ai
  ON ai.catalog_id = m.catalog_id AND ai.question_version_id = m.question_version_id
	LEFT JOIN canonical_audit_scope_drafts active_scope ON active_scope.id = $10
	LEFT JOIN planning_intake_drafts active_planning ON active_planning.id = active_scope.planning_intake_draft_id
	LEFT JOIN LATERAL (
	    SELECT
	      COUNT(DISTINCT prior_report.inspection_id) FILTER (WHERE prior_response.id IS NOT NULL) AS question_history_count,
	      (
	        SELECT COUNT(DISTINCT comparison_report.inspection_id)
	        FROM inspection_packages comparison_package
	        JOIN canonical_audit_scope_snapshots comparison_snapshot ON comparison_snapshot.id = comparison_package.canonical_scope_snapshot_id
	        JOIN canonical_audit_scope_drafts comparison_scope ON comparison_scope.id = comparison_snapshot.scope_draft_id
	        JOIN planning_intake_drafts comparison_planning ON comparison_planning.id = comparison_scope.planning_intake_draft_id
	        JOIN report_versions comparison_report ON comparison_report.inspection_id = comparison_package.inspection_id
	        JOIN report_approval_states comparison_state ON comparison_state.report_version_id = comparison_report.id
	        WHERE comparison_snapshot.stage = 'RELEASED'
	          AND comparison_scope.organization_id = active_scope.organization_id
	          AND comparison_scope.provider_scope_id = active_scope.provider_scope_id
	          AND comparison_scope.regulated_target_id = active_scope.regulated_target_id
	          AND comparison_scope.audit_type = active_scope.audit_type
	          AND COALESCE(comparison_planning.values->>'location','') = COALESCE(active_planning.values->>'location','')
	          AND comparison_report.snapshot->>'kind' = 'FINAL'
	          AND comparison_state.status = 'LOCKED'
	          AND comparison_state.issued_at IS NOT NULL
	          AND comparison_state.issued_at <= $14::timestamptz
	          AND comparison_state.issued_at >= ($14::timestamptz - make_interval(months => 36))
	      ) AS comparable_audit_count,
	      COALESCE(bool_or(
        EXISTS (
          SELECT 1
          FROM potential_findings potential
          WHERE potential.inspection_id = prior_report.inspection_id
            AND potential.question_id = q.question_id
            AND (
              COALESCE(potential.status, '') IN ('PENDING_LEAD_REVIEW', 'RETURNED')
              OR (COALESCE(potential.status, '') = 'CONVERTED' AND potential.converted_finding_id IS NULL)
            )
        )
        OR EXISTS (
          SELECT 1
          FROM findings finding
          JOIN potential_findings potential ON potential.id = finding.potential_finding_id
          WHERE finding.inspection_id = prior_report.inspection_id
            AND potential.question_id = q.question_id
            AND COALESCE(finding.status, '') <> 'CLOSED'
        )
        OR EXISTS (
          SELECT 1
          FROM evidence_versions evidence
          JOIN findings finding ON finding.id = evidence.finding_id
          JOIN potential_findings potential ON potential.id = finding.potential_finding_id
          LEFT JOIN evidence_version_states evidence_state ON evidence_state.evidence_version_id = evidence.id
          WHERE finding.inspection_id = prior_report.inspection_id
            AND potential.question_id = q.question_id
            AND evidence.version = (
              SELECT max(latest.version)
              FROM evidence_versions latest
              WHERE latest.evidence_id = evidence.evidence_id
            )
            AND (COALESCE(evidence_state.scan_state, '') <> 'CLEAN'
              OR COALESCE(evidence_state.review_state, '') NOT IN ('PENDING_CAA_REVIEW', 'ACCEPTED'))
        )
      ), false) AS has_open_work,
      max(prior_state.issued_at) FILTER (
        WHERE prior_report.snapshot->>'kind' = 'FINAL'
          AND prior_state.status = 'LOCKED'
          AND prior_state.issued_at IS NOT NULL
          AND NOT EXISTS (
            SELECT 1
            FROM potential_findings potential
            WHERE potential.inspection_id = prior_report.inspection_id
              AND potential.question_id = q.question_id
              AND (
                COALESCE(potential.status, '') IN ('PENDING_LEAD_REVIEW', 'RETURNED')
                OR (COALESCE(potential.status, '') = 'CONVERTED' AND potential.converted_finding_id IS NULL)
              )
          )
          AND NOT EXISTS (
            SELECT 1
            FROM findings finding
            JOIN potential_findings potential ON potential.id = finding.potential_finding_id
            WHERE finding.inspection_id = prior_report.inspection_id
              AND potential.question_id = q.question_id
              AND COALESCE(finding.status, '') <> 'CLOSED'
          )
          AND NOT EXISTS (
            SELECT 1
            FROM evidence_versions evidence
            JOIN findings finding ON finding.id = evidence.finding_id
            JOIN potential_findings potential ON potential.id = finding.potential_finding_id
            LEFT JOIN evidence_version_states evidence_state ON evidence_state.evidence_version_id = evidence.id
            WHERE finding.inspection_id = prior_report.inspection_id
              AND potential.question_id = q.question_id
              AND evidence.version = (
                SELECT max(latest.version)
                FROM evidence_versions latest
                WHERE latest.evidence_id = evidence.evidence_id
              )
              AND (COALESCE(evidence_state.scan_state, '') <> 'CLEAN'
                OR COALESCE(evidence_state.review_state, '') NOT IN ('PENDING_CAA_REVIEW', 'ACCEPTED'))
          )
          AND EXISTS (
            SELECT 1
            FROM checklist_responses response
            WHERE response.inspection_id = prior_report.inspection_id
              AND response.question_id = q.question_id
	              AND response.response_value = 'COMPLIANT'
              AND NOT EXISTS (
                SELECT 1
                FROM potential_findings response_potential
                WHERE response_potential.checklist_response_id = response.id
                  AND COALESCE(response_potential.status, '') <> 'DISMISSED'
              )
          )
	      ) AS last_verified_at,
	      (array_agg(prior_response.response_value ORDER BY prior_state.issued_at DESC, prior_report.id DESC) FILTER (WHERE prior_response.id IS NOT NULL))[1] AS last_comparable_result,
	      (array_agg(prior_report.inspection_id ORDER BY prior_state.issued_at DESC, prior_report.id DESC))[1] AS last_comparable_audit_id,
	      COALESCE(bool_or(prior_response.id IS NULL OR prior_response.response_value <> 'COMPLIANT'), false) AS has_non_clean_history,
	      COALESCE(bool_or(EXISTS (
	        SELECT 1 FROM findings repeat_finding
	        JOIN potential_findings repeat_potential ON repeat_potential.id = repeat_finding.potential_finding_id
	        WHERE repeat_finding.inspection_id = prior_report.inspection_id
	          AND repeat_potential.question_id = q.question_id
	          AND repeat_finding.next_action ILIKE '%repeat%'
	      )), false) AS has_repeat_finding,
	      COALESCE(bool_or(EXISTS (
	        SELECT 1
	        FROM cap_revisions overdue_cap
	        JOIN findings overdue_finding ON overdue_finding.id = overdue_cap.finding_id
	        JOIN potential_findings overdue_potential ON overdue_potential.id = overdue_finding.potential_finding_id
	        WHERE overdue_finding.inspection_id = prior_report.inspection_id
	          AND overdue_potential.question_id = q.question_id
	          AND overdue_cap.target_completion_date < ($14::timestamptz AT TIME ZONE 'UTC')::date
	          AND overdue_cap.status NOT IN ('COMPLETED', 'ACCEPTED', 'CLOSED')
	      )), false) AS has_overdue_cap
	    FROM inspection_packages prior_package
    JOIN canonical_audit_scope_snapshots prior_snapshot
      ON prior_snapshot.id = prior_package.canonical_scope_snapshot_id
    JOIN canonical_audit_scope_drafts prior_scope
      ON prior_scope.id = prior_snapshot.scope_draft_id
    JOIN canonical_audit_scope_snapshot_questions prior_question
      ON prior_question.snapshot_id = prior_snapshot.id
     AND prior_question.question_version_id = m.question_version_id
	    JOIN report_versions prior_report ON prior_report.inspection_id = prior_package.inspection_id
	    JOIN report_approval_states prior_state ON prior_state.report_version_id = prior_report.id
	    LEFT JOIN checklist_responses prior_response
	      ON prior_response.inspection_id = prior_package.inspection_id
	     AND prior_response.question_id = q.question_id
	    JOIN planning_intake_drafts prior_planning ON prior_planning.id = prior_scope.planning_intake_draft_id
	    WHERE active_scope.id IS NOT NULL
	      AND prior_snapshot.stage = 'RELEASED'
			AND prior_scope.organization_id = active_scope.organization_id
		AND prior_scope.provider_scope_id = active_scope.provider_scope_id
		AND prior_scope.regulated_target_id = active_scope.regulated_target_id
			AND prior_scope.audit_type = active_scope.audit_type
			AND COALESCE(prior_planning.values->>'location','') = COALESCE(active_planning.values->>'location','')
			AND prior_report.snapshot->>'kind' = 'FINAL'
			AND prior_state.status = 'LOCKED'
			AND prior_state.issued_at IS NOT NULL
			AND prior_state.issued_at <= $14::timestamptz
			AND prior_state.issued_at >= ($14::timestamptz - make_interval(months => 36))
) history ON TRUE
LEFT JOIN LATERAL (
    SELECT
      CASE
        WHEN active_scope.id IS NULL THEN ai.default_recommendation_bucket
        WHEN history.has_open_work THEN 'SUGGESTED_NOW'
		WHEN history.has_repeat_finding THEN 'SUGGESTED_NOW'
		WHEN history.has_overdue_cap THEN 'SUGGESTED_NOW'
        WHEN $11::text[] <> '{}'::text[] AND NOT (ai.inspection_type_codes && $11::text[]) THEN 'OUTSIDE_FOCUS'
		WHEN history.has_non_clean_history THEN 'UNCERTAIN_SIGNAL'
		WHEN history.question_history_count < history.comparable_audit_count THEN 'UNCERTAIN_SIGNAL'
		WHEN history.comparable_audit_count < 2 THEN 'UNCERTAIN_SIGNAL'
		WHEN ai.risk_tier IN ('HIGH', 'UNKNOWN') THEN 'SUGGESTED_NOW'
        WHEN active_scope.id IS NOT NULL
         AND NOT canonical_audit_type_matches_question_focus(active_scope.audit_type, ai.inspection_type_codes)
          THEN 'OUTSIDE_FOCUS'
        WHEN history.last_verified_at IS NOT NULL
		 AND history.last_verified_at >= $14::timestamptz - make_interval(months => ai.recurrence_months)
          THEN 'RECENTLY_VERIFIED'
        WHEN history.last_verified_at IS NOT NULL
		 AND history.last_verified_at < $14::timestamptz - make_interval(months => ai.recurrence_months)
          THEN 'SUGGESTED_NOW'
        ELSE ai.default_recommendation_bucket
      END AS recommendation_state,
	      array_remove(ARRAY[
	        CASE WHEN history.has_open_work THEN 'OPEN_WORK' END,
		CASE WHEN history.has_repeat_finding THEN 'REPEAT_FINDING' END,
		CASE WHEN history.has_overdue_cap THEN 'OVERDUE_CAP' END,
		CASE WHEN history.has_non_clean_history THEN 'NON_CLEAN_OR_MISSING_ANSWER' END,
		CASE WHEN history.question_history_count < history.comparable_audit_count THEN 'UNKNOWN_HISTORY' END,
		CASE WHEN history.comparable_audit_count < 2 THEN 'INSUFFICIENT_LONGITUDINAL_HISTORY' END,
        CASE WHEN ai.risk_tier IN ('HIGH', 'UNKNOWN') THEN 'HIGH_OR_UNKNOWN_RISK' END,
        CASE WHEN history.last_verified_at IS NOT NULL
				   AND history.last_verified_at >= $14::timestamptz - make_interval(months => ai.recurrence_months)
             THEN 'RECENT_FINAL_VERIFICATION' END,
        CASE WHEN history.last_verified_at IS NOT NULL
				   AND history.last_verified_at < $14::timestamptz - make_interval(months => ai.recurrence_months)
             THEN 'RECURRENCE_DUE' END,
        CASE WHEN $11::text[] <> '{}'::text[] AND NOT (ai.inspection_type_codes && $11::text[])
             THEN 'OUTSIDE_SELECTED_FOCUS' END,
        CASE WHEN active_scope.id IS NOT NULL
                   AND canonical_audit_type_matches_question_focus(active_scope.audit_type, ai.inspection_type_codes)
             THEN 'AUDIT_TYPE_FOCUS_MATCH' END,
        CASE WHEN active_scope.id IS NOT NULL
                   AND NOT canonical_audit_type_matches_question_focus(active_scope.audit_type, ai.inspection_type_codes)
             THEN 'OUTSIDE_AUDIT_TYPE_FOCUS' END,
        CASE WHEN ai.external_applicability_unresolved THEN 'SOURCE_CONTEXT_INCOMPLETE' END
      ], NULL)::text[] AS reason_codes,
	      history.last_verified_at,
	      CASE WHEN history.last_verified_at IS NULL THEN NULL ELSE history.last_verified_at + make_interval(months => ai.recurrence_months) END AS recurrence_due_at,
	      history.last_comparable_result,
	      history.last_comparable_audit_id,
	      history.question_history_count,
	      history.comparable_audit_count,
	      history.has_open_work,
	      history.has_repeat_finding,
	      history.has_overdue_cap,
	      history.has_non_clean_history,
	      CASE
	        WHEN ai.mandatory_control OR ai.safety_critical OR ai.risk_tier = 'HIGH' THEN 'MANDATORY_CORE'
	        WHEN history.has_open_work OR history.has_repeat_finding OR history.has_overdue_cap OR history.has_non_clean_history THEN 'FOCUSED_FULL'
	        WHEN history.question_history_count < history.comparable_audit_count OR history.comparable_audit_count < 2 THEN 'ROTATIONAL_SAMPLE'
	        WHEN history.last_verified_at IS NOT NULL
	         AND history.last_verified_at >= $14::timestamptz - make_interval(months => ai.recurrence_months)
	         AND ai.risk_tier NOT IN ('HIGH','UNKNOWN')
	         AND ($11::text[] = '{}'::text[] OR ai.inspection_type_codes && $11::text[])
	         AND (active_scope.id IS NULL OR canonical_audit_type_matches_question_focus(active_scope.audit_type, ai.inspection_type_codes)) THEN 'DEFER_ELIGIBLE'
	        ELSE 'ROTATIONAL_SAMPLE'
	      END AS recommendation_classification,
	      CASE
	        WHEN ai.mandatory_control OR ai.safety_critical OR ai.risk_tier = 'HIGH' THEN true
	        WHEN history.last_verified_at IS NOT NULL
	         AND history.last_verified_at >= $14::timestamptz - make_interval(months => ai.recurrence_months)
	         AND NOT history.has_open_work AND NOT history.has_repeat_finding AND NOT history.has_overdue_cap AND NOT history.has_non_clean_history
	         AND history.question_history_count >= history.comparable_audit_count
	         AND history.comparable_audit_count >= 2
	         AND ai.risk_tier NOT IN ('HIGH','UNKNOWN')
	         AND ($11::text[] = '{}'::text[] OR ai.inspection_type_codes && $11::text[])
	         AND (active_scope.id IS NULL OR canonical_audit_type_matches_question_focus(active_scope.audit_type, ai.inspection_type_codes))
	         AND NOT ai.mandatory_control AND NOT ai.safety_critical AND ai.risk_tier <> 'HIGH' THEN false
	        ELSE true
	      END AS included_by_default,
	      CASE
	        WHEN history.last_verified_at IS NOT NULL
	         AND history.last_verified_at >= $14::timestamptz - make_interval(months => ai.recurrence_months)
	         AND NOT history.has_open_work AND NOT history.has_repeat_finding AND NOT history.has_overdue_cap AND NOT history.has_non_clean_history
	         AND history.question_history_count >= history.comparable_audit_count
	         AND history.comparable_audit_count >= 2
	         AND ai.risk_tier NOT IN ('HIGH','UNKNOWN')
	         AND ($11::text[] = '{}'::text[] OR ai.inspection_type_codes && $11::text[])
	         AND (active_scope.id IS NULL OR canonical_audit_type_matches_question_focus(active_scope.audit_type, ai.inspection_type_codes))
	         AND NOT ai.mandatory_control AND NOT ai.safety_critical AND ai.risk_tier <> 'HIGH' THEN true
	        ELSE false
	      END AS can_defer,
	      CASE WHEN history.last_verified_at IS NOT NULL
	                 AND history.last_verified_at >= $14::timestamptz - make_interval(months => ai.recurrence_months)
	                 AND NOT history.has_open_work AND NOT history.has_repeat_finding AND NOT history.has_overdue_cap AND NOT history.has_non_clean_history
	                 AND history.question_history_count >= history.comparable_audit_count
	                 AND history.comparable_audit_count >= 2
	                 AND ai.risk_tier NOT IN ('HIGH','UNKNOWN')
	                 AND ($11::text[] = '{}'::text[] OR ai.inspection_type_codes && $11::text[])
	                 AND (active_scope.id IS NULL OR canonical_audit_type_matches_question_focus(active_scope.audit_type, ai.inspection_type_codes))
	                 AND NOT ai.mandatory_control AND NOT ai.safety_critical AND ai.risk_tier <> 'HIGH' THEN 'Repeated validated-clean history is within its recurrence interval; the question is safe to defer by default.'
	           WHEN history.has_open_work OR history.has_repeat_finding OR history.has_overdue_cap THEN 'Open, repeat, or overdue work keeps this question in the suggested scope.'
	           WHEN history.has_non_clean_history OR history.question_history_count < history.comparable_audit_count THEN 'History is incomplete or non-clean; the question remains suggested.'
	           WHEN history.comparable_audit_count < 2 THEN 'One clean Audit is not sufficient longitudinal evidence for omission.'
	           ELSE 'The server recommendation keeps this question in the current rotational scope.' END AS recommendation_rationale,
	      ARRAY['MANDATORY_FLOOR_ENFORCED','FULL_CATALOG_OVERRIDE_ALLOWED']::text[] AS recommendation_guardrails
) recommendation ON TRUE
`

func (api *CanonicalAPI) listCanonicalQuestionCatalogEntries(writer http.ResponseWriter, request *http.Request) {
	actor, ok := api.requireCanonicalCatalogActor(writer, request, false)
	if !ok {
		return
	}
	usage, err := parseQuestionUsageClass(request.URL.Query().Get("usageClass"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	offset, err := parseCatalogCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	projection, err := parseCanonicalCatalogProjection(request.URL.Query().Get("projection"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	maximumPageSize := canonicalCatalogMaximumPageSize
	if projection == canonicalCatalogProjectionSelection {
		maximumPageSize = canonicalCatalogMaximumSelectionSize
	}
	limit := canonicalCatalogPageSize
	if raw := request.URL.Query().Get("limit"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > maximumPageSize {
			api.respond(writer, nil, fmt.Errorf("%w: catalog limit must be between 1 and %d", application.ErrInvalid, maximumPageSize))
			return
		}
		limit = parsed
	}
	if api.pool == nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	catalogVersion := decodedCanonicalPathParam(request, "catalogVersion")
	search := strings.TrimSpace(request.URL.Query().Get("search"))
	formCode := strings.TrimSpace(request.URL.Query().Get("formCode"))
	domain := strings.TrimSpace(request.URL.Query().Get("domain"))
	topic := strings.TrimSpace(request.URL.Query().Get("topic"))
	riskBand := strings.TrimSpace(request.URL.Query().Get("riskBand"))
	checklistFocus, err := parseCanonicalChecklistFocus(request.URL.Query().Get("checklistFocus"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	recommendationState := strings.ToUpper(strings.TrimSpace(request.URL.Query().Get("recommendationState")))
	if recommendationState != "" {
		validStates := map[string]struct{}{"SUGGESTED_NOW": {}, "MATCHING_OPTIONAL": {}, "RECENTLY_VERIFIED": {}, "OUTSIDE_FOCUS": {}, "UNCERTAIN_SIGNAL": {}}
		if _, ok := validStates[recommendationState]; !ok {
			api.respond(writer, nil, fmt.Errorf("%w: unsupported recommendation state", application.ErrInvalid))
			return
		}
	}
	var includedByDefault *bool
	if raw := strings.TrimSpace(request.URL.Query().Get("includedByDefault")); raw != "" {
		parsed, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			api.respond(writer, nil, fmt.Errorf("%w: includedByDefault must be boolean", application.ErrInvalid))
			return
		}
		includedByDefault = &parsed
	}
	sourceGapState := strings.TrimSpace(request.URL.Query().Get("sourceGapState"))
	selected := strings.TrimSpace(request.URL.Query().Get("selected"))
	scopeID := strings.TrimSpace(request.URL.Query().Get("scopeId"))
	applicationType, err := parseCanonicalExecutionType(request.URL.Query().Get("applicationType"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if scopeID != "" {
		if err := api.requireCanonicalScopeOwner(request.Context(), scopeID, actor.SubjectID); err != nil {
			api.respond(writer, nil, err)
			return
		}
	}
	if applicationType != "" {
		if scopeID == "" {
			api.respond(writer, nil, fmt.Errorf("%w: application type requires an owned scope draft", application.ErrInvalid))
			return
		}
		var scopeAuditType string
		if err := api.pool.QueryRow(request.Context(), `
			SELECT audit_type
			FROM canonical_audit_scope_drafts
			WHERE id = $1 AND status = 'DRAFT'
		`, scopeID).Scan(&scopeAuditType); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				api.respond(writer, nil, application.ErrNotFound)
				return
			}
			api.respond(writer, nil, err)
			return
		}
		if strings.TrimSpace(scopeAuditType) != applicationType {
			api.respond(writer, nil, fmt.Errorf("%w: application type does not match the selected scope draft", application.ErrConflict))
			return
		}
	}
	if selected != "" && selected != "all" && selected != "selected" && selected != "unselected" {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	selectionPredicate := `($9='' OR $9='all' OR (($9='selected') = EXISTS (SELECT 1 FROM canonical_audit_scope_selection_questions sq JOIN canonical_audit_scope_selection_operations so ON so.id=sq.operation_id WHERE so.id=(SELECT latest.id FROM canonical_audit_scope_selection_operations latest WHERE latest.scope_draft_id=$10 AND latest.operation_kind <> 'PREVIEW' ORDER BY latest.created_at DESC, latest.id DESC LIMIT 1) AND sq.question_version_id=m.question_version_id)))`
	where := `c.catalog_version=$1 AND c.usage_class=$2 AND c.status='SEALED' AND c.source_origin='IMPORTED_APPROVED_SOURCE' AND ($3='' OR m.form_code ILIKE '%' || $3 || '%' OR m.proposal_id ILIKE '%' || $3 || '%' OR q.prompt ILIKE '%' || $3 || '%') AND ($4='' OR m.form_code = ANY(string_to_array($4, ','))) AND ($5='' OR ai.domain_code = ANY(string_to_array($5, ','))) AND ($6='' OR ai.topic_codes && string_to_array($6, ',')) AND ($7='' OR ai.risk_tier = ANY(string_to_array($7, ','))) AND ($8='' OR m.source_gap_state=$8) AND (($11::text[] = '{}'::text[] OR ai.inspection_type_codes && $11::text[]) OR ($12='OUTSIDE_FOCUS' AND $11::text[] <> '{}'::text[] AND NOT (ai.inspection_type_codes && $11::text[]))) AND ($12='' OR recommendation.recommendation_state=$12) AND ($15::boolean IS NULL OR recommendation.included_by_default=$15::boolean) AND COALESCE((SELECT status FROM canonical_question_catalog_membership_events event WHERE event.catalog_id=m.catalog_id AND event.question_version_id=m.question_version_id ORDER BY occurred_at DESC,event_id DESC LIMIT 1),'AVAILABLE')='AVAILABLE' AND ($10='' OR EXISTS (SELECT 1 FROM canonical_question_catalog_applicabilities applicability JOIN canonical_audit_scope_drafts scope ON scope.id=$10 WHERE applicability.catalog_id=m.catalog_id AND applicability.question_version_id=m.question_version_id AND applicability.provider_scope_id=scope.provider_scope_id AND applicability.regulated_target_id=scope.regulated_target_id AND applicability.status='ELIGIBLE')) AND ($13='' OR active_scope.audit_type=$13) AND ` + selectionPredicate
	ctx := request.Context()
	var total int64
	queryArgs := []any{catalogVersion, string(usage), search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID, checklistFocus, recommendationState, applicationType, api.clock().UTC(), includedByDefault}
	if projection == canonicalCatalogProjectionSelection {
		rows, err := api.pool.Query(ctx, `
			SELECT c.catalog_version,c.usage_class,m.question_version_id,m.form_code,m.proposal_id,m.ordinal,m.question_digest,
			       COALESCE(m.source_locator,''),m.source_gap_state,
			       ai.domain_code,ai.topic_codes,ai.inspection_type_codes,ai.inspection_profile_codes,ai.applicability_disposition,
			       ai.risk_tier,ai.safety_critical,ai.agreement_confidence,recommendation.recommendation_state,
			       recommendation.reason_codes,ai.recurrence_months,
			       to_char(recommendation.last_verified_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
			       to_char(recommendation.recurrence_due_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
			       recommendation.last_comparable_result,recommendation.last_comparable_audit_id,
			       recommendation.question_history_count,recommendation.comparable_audit_count,
			       recommendation.recommendation_classification,recommendation.included_by_default,
			       recommendation.can_defer,recommendation.recommendation_rationale,recommendation.recommendation_guardrails,
			       ai.external_applicability_unresolved
			FROM canonical_question_catalogs c
			JOIN canonical_question_catalog_memberships m ON m.catalog_id = c.id
			JOIN question_versions q ON q.id=m.question_version_id
			`+canonicalCatalogAIProjectionJoins+`
			WHERE `+where+` ORDER BY m.form_code,m.ordinal,m.question_version_id LIMIT $16 OFFSET $17`, append(queryArgs, limit+1, offset)...)
		if err != nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
		defer rows.Close()
		items := make([]generated.CanonicalQuestionCatalogEntry, 0, limit)
		for rows.Next() {
			var row canonicalCatalogRow
			var advisoryDomain, advisoryApplicability, advisoryRiskTier, advisoryConfidence, advisoryState string
			var advisoryTopics, advisoryInspectionTypes, advisoryInspectionProfiles, advisoryReasons []string
			var advisorySafetyCritical, advisoryUnresolved bool
			var advisoryRecurrence int64
			var previouslyVerifiedAt, recurrenceDueAt *string
			var lastComparableResult, lastComparableAuditID *string
			var recommendationHistoryCount, comparableAuditCount int64
			var recommendationClassification, recommendationRationale string
			var recommendationIncludedByDefault, recommendationCanDefer bool
			var recommendationGuardrails []string
			if err := rows.Scan(&row.CatalogVersion, &row.UsageClass, &row.QuestionID, &row.FormCode, &row.ProposalID, &row.Ordinal, &row.Digest, &row.SourceLocator, &row.SourceGap, &advisoryDomain, &advisoryTopics, &advisoryInspectionTypes, &advisoryInspectionProfiles, &advisoryApplicability, &advisoryRiskTier, &advisorySafetyCritical, &advisoryConfidence, &advisoryState, &advisoryReasons, &advisoryRecurrence, &previouslyVerifiedAt, &recurrenceDueAt, &lastComparableResult, &lastComparableAuditID, &recommendationHistoryCount, &comparableAuditCount, &recommendationClassification, &recommendationIncludedByDefault, &recommendationCanDefer, &recommendationRationale, &recommendationGuardrails, &advisoryUnresolved); err != nil {
				api.respond(writer, nil, application.ErrNotFound)
				return
			}
			if advisoryTopics == nil {
				advisoryTopics = []string{}
			}
			if advisoryInspectionTypes == nil {
				advisoryInspectionTypes = []string{}
			}
			if advisoryInspectionProfiles == nil {
				advisoryInspectionProfiles = []string{}
			}
			if advisoryReasons == nil {
				advisoryReasons = []string{}
			}
			if advisoryRiskTier == "" {
				advisoryRiskTier = "UNKNOWN"
			}
			if advisoryConfidence == "" {
				advisoryConfidence = "LOW"
			}
			if advisoryState == "" {
				advisoryState = "UNCERTAIN_SIGNAL"
			}
			if advisoryRecurrence < 1 {
				advisoryRecurrence = 12
			}
			row.ScopeID = scopeID
			row.AIAdvisory = generated.CanonicalQuestionAIAdvisory{
				DomainCode: advisoryDomain, TopicCodes: advisoryTopics, InspectionTypeCodes: advisoryInspectionTypes,
				InspectionProfileCodes: advisoryInspectionProfiles, ApplicabilityDisposition: advisoryApplicability,
				RiskTier: advisoryRiskTier, SafetyCritical: advisorySafetyCritical, AgreementConfidence: advisoryConfidence,
				AdvisoryState: advisoryState, RecommendationReasonCodes: advisoryReasons, RecurrenceMonths: advisoryRecurrence,
				PreviouslyVerifiedAt: previouslyVerifiedAt, RecurrenceDueAt: recurrenceDueAt,
				ExternalApplicabilityUnresolved: advisoryUnresolved,
			}
			row.Recommendation = generated.CanonicalQuestionRecommendation{RecommendationState: advisoryState, Classification: recommendationClassification, IncludedByDefault: recommendationIncludedByDefault, CanDefer: recommendationCanDefer, HistoryCount: recommendationHistoryCount, ComparableAuditCount: comparableAuditCount, LastComparableResult: lastComparableResult, LastComparableAuditId: lastComparableAuditID, LastVerifiedAt: previouslyVerifiedAt, RecurrenceDueAt: recurrenceDueAt, SignalCodes: advisoryReasons, Rationale: recommendationRationale, Guardrails: recommendationGuardrails}
			items = append(items, canonicalCatalogEntry(row))
		}
		if err := rows.Err(); err != nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
		var next *string
		if len(items) > limit {
			items = items[:limit]
			next = encodeCatalogCursor(offset + limit)
		}
		api.respond(writer, generated.CanonicalQuestionCatalogPage{
			Items: items, NextCursor: next, CatalogVersion: catalogVersion,
			UsageClass: generated.QuestionUsageClass(usage), TotalCount: 0,
			Facets: emptyCanonicalCatalogFacets(),
		}, nil)
		return
	}
	if projection == canonicalCatalogProjectionFull {
		if err := api.pool.QueryRow(ctx, `SELECT COUNT(*) FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id = c.id JOIN question_versions q ON q.id=m.question_version_id `+canonicalCatalogAIProjectionJoins+` WHERE `+where, queryArgs...).Scan(&total); err != nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
	}
	rows, err := api.pool.Query(ctx, `
		SELECT c.catalog_version,c.usage_class,m.question_version_id,m.form_code,m.proposal_id,m.ordinal,m.question_digest,
		       COALESCE(m.source_locator,''),m.source_gap_state,ai.domain_code,COALESCE(array_to_string(ai.topic_codes, ','),''),
		       '',q.prompt,q.configured_reference,q.expected_evidence,
		       NULL::text,NULL::bigint,NULL::text,NULL::text,
		       0::bigint,NULL::text,NULL::text,NULL::text,NULL::text,
		       ai.domain_code,ai.topic_codes,ai.inspection_type_codes,ai.inspection_profile_codes,ai.applicability_disposition,
		       ai.risk_tier,ai.safety_critical,ai.agreement_confidence,recommendation.recommendation_state,
		       recommendation.reason_codes,ai.recurrence_months,
		       to_char(recommendation.last_verified_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
		       to_char(recommendation.recurrence_due_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
		       recommendation.last_comparable_result,recommendation.last_comparable_audit_id,
		       recommendation.question_history_count,recommendation.comparable_audit_count,
		       recommendation.recommendation_classification,recommendation.included_by_default,
		       recommendation.can_defer,recommendation.recommendation_rationale,recommendation.recommendation_guardrails,
		       ai.external_applicability_unresolved
		FROM canonical_question_catalogs c
		JOIN canonical_question_catalog_memberships m ON m.catalog_id = c.id
		JOIN question_versions q ON q.id=m.question_version_id
		`+canonicalCatalogAIProjectionJoins+`
		WHERE `+where+` ORDER BY m.form_code,m.ordinal,m.question_version_id LIMIT $16 OFFSET $17`, append(queryArgs, limit+1, offset)...)
	if err != nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	defer rows.Close()
	items := make([]generated.CanonicalQuestionCatalogEntry, 0, limit)
	for rows.Next() {
		var row canonicalCatalogRow
		var advisoryDomain, advisoryApplicability, advisoryRiskTier, advisoryConfidence, advisoryState string
		var advisoryTopics, advisoryInspectionTypes, advisoryInspectionProfiles, advisoryReasons []string
		var advisorySafetyCritical, advisoryUnresolved bool
		var advisoryRecurrence int64
		var previouslyVerifiedAt, recurrenceDueAt *string
		var lastComparableResult, lastComparableAuditID *string
		var recommendationHistoryCount, comparableAuditCount int64
		var recommendationClassification, recommendationRationale string
		var recommendationIncludedByDefault, recommendationCanDefer bool
		var recommendationGuardrails []string
		if err := rows.Scan(&row.CatalogVersion, &row.UsageClass, &row.QuestionID, &row.FormCode, &row.ProposalID, &row.Ordinal, &row.Digest, &row.SourceLocator, &row.SourceGap, &row.Domain, &row.Topic, &row.RiskBand, &row.Prompt, &row.ConfiguredReference, &row.ExpectedEvidence, &row.GovernedCandidateID, &row.GovernedCandidateRevision, &row.GovernedCandidateContentDigest, &row.GovernedCandidateStatus, &row.ReviewRevision, &row.ReviewDisposition, &row.ReviewReason, &row.ReviewDomain, &row.ReviewTopic, &advisoryDomain, &advisoryTopics, &advisoryInspectionTypes, &advisoryInspectionProfiles, &advisoryApplicability, &advisoryRiskTier, &advisorySafetyCritical, &advisoryConfidence, &advisoryState, &advisoryReasons, &advisoryRecurrence, &previouslyVerifiedAt, &recurrenceDueAt, &lastComparableResult, &lastComparableAuditID, &recommendationHistoryCount, &comparableAuditCount, &recommendationClassification, &recommendationIncludedByDefault, &recommendationCanDefer, &recommendationRationale, &recommendationGuardrails, &advisoryUnresolved); err != nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
		row.AIAdvisory = generated.CanonicalQuestionAIAdvisory{DomainCode: advisoryDomain, TopicCodes: advisoryTopics, InspectionTypeCodes: advisoryInspectionTypes, InspectionProfileCodes: advisoryInspectionProfiles, ApplicabilityDisposition: advisoryApplicability, RiskTier: advisoryRiskTier, SafetyCritical: advisorySafetyCritical, AgreementConfidence: advisoryConfidence, AdvisoryState: advisoryState, RecommendationReasonCodes: advisoryReasons, RecurrenceMonths: advisoryRecurrence, PreviouslyVerifiedAt: previouslyVerifiedAt, RecurrenceDueAt: recurrenceDueAt, ExternalApplicabilityUnresolved: advisoryUnresolved}
		row.Recommendation = generated.CanonicalQuestionRecommendation{RecommendationState: advisoryState, Classification: recommendationClassification, IncludedByDefault: recommendationIncludedByDefault, CanDefer: recommendationCanDefer, HistoryCount: recommendationHistoryCount, ComparableAuditCount: comparableAuditCount, LastComparableResult: lastComparableResult, LastComparableAuditId: lastComparableAuditID, LastVerifiedAt: previouslyVerifiedAt, RecurrenceDueAt: recurrenceDueAt, SignalCodes: advisoryReasons, Rationale: recommendationRationale, Guardrails: recommendationGuardrails}
		row.ScopeID = scopeID
		row.ReviewHistory, err = api.loadQuestionReviewHistory(ctx, row)
		if err != nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
		items = append(items, canonicalCatalogEntry(row))
	}
	if err := rows.Err(); err != nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = encodeCatalogCursor(offset + limit)
	}
	var facets generated.CanonicalQuestionCatalogFacets
	if projection == canonicalCatalogProjectionFull {
		var facetErr error
		facets, facetErr = api.loadCanonicalCatalogFacets(ctx, catalogVersion, usage, search, formCode, domain, topic, riskBand, sourceGapState, scopeID, checklistFocus, recommendationState, applicationType)
		if facetErr != nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
	}
	api.respond(writer, generated.CanonicalQuestionCatalogPage{Items: items, NextCursor: next, CatalogVersion: catalogVersion, UsageClass: generated.QuestionUsageClass(usage), TotalCount: total, Facets: facets}, nil)
}

func canonicalCatalogFacetWhere(exclude string) string {
	parts := []string{
		"c.catalog_version=$1::text",
		"c.usage_class=$2::text",
		"c.status='SEALED'",
		"c.source_origin='IMPORTED_APPROVED_SOURCE'",
		"($3::text='' OR m.form_code ILIKE '%' || $3::text || '%' OR m.proposal_id ILIKE '%' || $3::text || '%' OR q.prompt ILIKE '%' || $3::text || '%')",
		"($8::text='' OR m.source_gap_state=$8::text)",
		"$9::text=$9::text",
		"COALESCE((SELECT status FROM canonical_question_catalog_membership_events event WHERE event.catalog_id=m.catalog_id AND event.question_version_id=m.question_version_id ORDER BY occurred_at DESC,event_id DESC LIMIT 1),'AVAILABLE')='AVAILABLE'",
		"($10::text='' OR EXISTS (SELECT 1 FROM canonical_question_catalog_applicabilities applicability JOIN canonical_audit_scope_drafts scope ON scope.id=$10::text WHERE applicability.catalog_id=m.catalog_id AND applicability.question_version_id=m.question_version_id AND applicability.provider_scope_id=scope.provider_scope_id AND applicability.regulated_target_id=scope.regulated_target_id AND applicability.status='ELIGIBLE'))",
	}
	if exclude != "focus" {
		parts = append(parts, "($11::text[] = '{}'::text[] OR ai.inspection_type_codes && $11::text[])")
	} else {
		parts = append(parts, "$11::text[]=$11::text[]")
	}
	if exclude != "recommendation" {
		parts = append(parts, "($12::text='' OR recommendation.recommendation_state=$12::text)")
	} else {
		parts = append(parts, "$12::text=$12::text")
	}
	parts = append(parts, "($15::boolean IS NULL OR recommendation.included_by_default=$15::boolean)")
	parts = append(parts, "($13::text='' OR active_scope.audit_type=$13::text)")
	if exclude != "form" {
		parts = append(parts, "($4::text='' OR m.form_code = ANY(string_to_array($4::text, ',')))")
	} else {
		parts = append(parts, "$4::text=$4::text")
	}
	if exclude != "domain" {
		parts = append(parts, "($5::text='' OR ai.domain_code = ANY(string_to_array($5::text, ',')))")
	} else {
		parts = append(parts, "$5::text=$5::text")
	}
	if exclude != "topic" {
		parts = append(parts, "($6::text='' OR ai.topic_codes && string_to_array($6::text, ','))")
	} else {
		parts = append(parts, "$6::text=$6::text")
	}
	if exclude != "risk" {
		parts = append(parts, "($7::text='' OR ai.risk_tier = ANY(string_to_array($7::text, ',')))")
	} else {
		parts = append(parts, "$7::text=$7::text")
	}
	return strings.Join(parts, " AND ")
}

func (api *CanonicalAPI) loadCanonicalCatalogFacetOptions(ctx context.Context, catalogVersion string, usage questioncatalog.UsageClass, search, formCode, domain, topic, riskBand, sourceGapState, scopeID string, checklistFocus []string, recommendationState, applicationType, exclude, valueSQL string) ([]generated.CanonicalQuestionCatalogFacetOption, error) {
	query := `SELECT value, count(*) FROM (` + valueSQL + `) valueset GROUP BY value ORDER BY value LIMIT 200`
	args := []any{catalogVersion, string(usage), search, formCode, domain, topic, riskBand, sourceGapState, "", scopeID, checklistFocus, recommendationState, applicationType, api.clock().UTC(), nil}
	rows, err := api.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	options := []generated.CanonicalQuestionCatalogFacetOption{}
	for rows.Next() {
		var value string
		var count int64
		if err := rows.Scan(&value, &count); err != nil {
			return nil, err
		}
		if strings.TrimSpace(value) != "" {
			options = append(options, generated.CanonicalQuestionCatalogFacetOption{Value: value, Count: count})
		}
	}
	return options, rows.Err()
}

func (api *CanonicalAPI) loadCanonicalCatalogFacets(ctx context.Context, catalogVersion string, usage questioncatalog.UsageClass, search, formCode, domain, topic, riskBand, sourceGapState, scopeID string, checklistFocus []string, recommendationState, applicationType string) (generated.CanonicalQuestionCatalogFacets, error) {
	base := `SELECT %s AS value FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id JOIN question_versions q ON q.id=m.question_version_id ` + canonicalCatalogAIProjectionJoins + ` WHERE `
	forms, err := api.loadCanonicalCatalogFacetOptions(ctx, catalogVersion, usage, search, formCode, domain, topic, riskBand, sourceGapState, scopeID, checklistFocus, recommendationState, applicationType, "form", strings.Replace(base+canonicalCatalogFacetWhere("form"), "%s", "m.form_code", 1))
	if err != nil {
		return generated.CanonicalQuestionCatalogFacets{}, err
	}
	domains, err := api.loadCanonicalCatalogFacetOptions(ctx, catalogVersion, usage, search, formCode, domain, topic, riskBand, sourceGapState, scopeID, checklistFocus, recommendationState, applicationType, "domain", strings.Replace(base+canonicalCatalogFacetWhere("domain"), "%s", "ai.domain_code", 1))
	if err != nil {
		return generated.CanonicalQuestionCatalogFacets{}, err
	}
	topics, err := api.loadCanonicalCatalogFacetOptions(ctx, catalogVersion, usage, search, formCode, domain, topic, riskBand, sourceGapState, scopeID, checklistFocus, recommendationState, applicationType, "topic", "SELECT unnest(ai.topic_codes) AS value FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id JOIN question_versions q ON q.id=m.question_version_id "+canonicalCatalogAIProjectionJoins+" WHERE "+canonicalCatalogFacetWhere("topic"))
	if err != nil {
		return generated.CanonicalQuestionCatalogFacets{}, err
	}
	riskTiers, err := api.loadCanonicalCatalogFacetOptions(ctx, catalogVersion, usage, search, formCode, domain, topic, riskBand, sourceGapState, scopeID, checklistFocus, recommendationState, applicationType, "risk", strings.Replace(base+canonicalCatalogFacetWhere("risk"), "%s", "ai.risk_tier", 1))
	if err != nil {
		return generated.CanonicalQuestionCatalogFacets{}, err
	}
	focuses, err := api.loadCanonicalCatalogFacetOptions(ctx, catalogVersion, usage, search, formCode, domain, topic, riskBand, sourceGapState, scopeID, checklistFocus, recommendationState, applicationType, "focus", "SELECT unnest(ai.inspection_type_codes) AS value FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id JOIN question_versions q ON q.id=m.question_version_id "+canonicalCatalogAIProjectionJoins+" WHERE "+canonicalCatalogFacetWhere("focus"))
	if err != nil {
		return generated.CanonicalQuestionCatalogFacets{}, err
	}
	recommendations, err := api.loadCanonicalCatalogFacetOptions(ctx, catalogVersion, usage, search, formCode, domain, topic, riskBand, sourceGapState, scopeID, checklistFocus, recommendationState, applicationType, "recommendation", "SELECT recommendation.recommendation_state AS value FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id JOIN question_versions q ON q.id=m.question_version_id "+canonicalCatalogAIProjectionJoins+" WHERE "+canonicalCatalogFacetWhere("recommendation"))
	if err != nil {
		return generated.CanonicalQuestionCatalogFacets{}, err
	}
	return generated.CanonicalQuestionCatalogFacets{Forms: forms, Domains: domains, Topics: topics, RiskTiers: riskTiers, ChecklistFocuses: focuses, RecommendationStates: recommendations}, nil
}

func (api *CanonicalAPI) getCanonicalQuestionCatalogEntry(writer http.ResponseWriter, request *http.Request) {
	actor, ok := api.requireCanonicalCatalogActor(writer, request, false)
	if !ok {
		return
	}
	usage, err := parseQuestionUsageClass(request.URL.Query().Get("usageClass"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	scopeID := strings.TrimSpace(request.URL.Query().Get("scopeId"))
	if scopeID != "" {
		if err := api.requireCanonicalScopeOwner(request.Context(), scopeID, actor.SubjectID); err != nil {
			api.respond(writer, nil, err)
			return
		}
	}
	if api.pool == nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	catalogVersion := decodedCanonicalPathParam(request, "catalogVersion")
	questionVersionID := decodedCanonicalPathParam(request, "questionVersionId")
	var row canonicalCatalogRow
	var advisoryDomain, advisoryApplicability, advisoryRiskTier, advisoryConfidence, advisoryState string
	var advisoryTopics, advisoryInspectionTypes, advisoryInspectionProfiles, advisoryReasons []string
	var advisorySafetyCritical, advisoryUnresolved bool
	var advisoryRecurrence int64
	var previouslyVerifiedAt, recurrenceDueAt *string
	applicationType, err := parseCanonicalExecutionType(request.URL.Query().Get("applicationType"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if applicationType != "" && scopeID != "" {
		var scopeAuditType string
		if err := api.pool.QueryRow(request.Context(), `SELECT audit_type FROM canonical_audit_scope_drafts WHERE id = $1 AND status = 'DRAFT'`, scopeID).Scan(&scopeAuditType); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				api.respond(writer, nil, application.ErrNotFound)
				return
			}
			api.respond(writer, nil, err)
			return
		}
		if scopeAuditType != applicationType {
			api.respond(writer, nil, fmt.Errorf("%w: application type does not match the selected scope draft", application.ErrConflict))
			return
		}
	}
	err = api.pool.QueryRow(request.Context(), `
		SELECT c.catalog_version,c.usage_class,m.question_version_id,m.form_code,m.proposal_id,m.ordinal,m.question_digest,
		       COALESCE(m.source_locator,''),m.source_gap_state,ai.domain_code,COALESCE(array_to_string(ai.topic_codes, ','),''),'',
		       q.prompt,q.configured_reference,q.expected_evidence,
		       NULL::text,NULL::bigint,NULL::text,NULL::text,
		       0::bigint,NULL::text,NULL::text,NULL::text,NULL::text,
		       ai.domain_code,ai.topic_codes,ai.inspection_type_codes,ai.inspection_profile_codes,ai.applicability_disposition,
		       ai.risk_tier,ai.safety_critical,ai.agreement_confidence,
		       CASE
		         WHEN active_scope.id IS NULL THEN ai.default_recommendation_bucket
		         WHEN history.has_open_work THEN 'SUGGESTED_NOW'
		         WHEN ai.risk_tier IN ('HIGH', 'UNKNOWN') THEN 'SUGGESTED_NOW'
		         WHEN NOT canonical_audit_type_matches_question_focus(active_scope.audit_type, ai.inspection_type_codes) THEN 'OUTSIDE_FOCUS'
		         WHEN history.last_verified_at IS NOT NULL
		           AND history.last_verified_at >= $5::timestamptz - make_interval(months => ai.recurrence_months) THEN 'RECENTLY_VERIFIED'
		         WHEN history.last_verified_at IS NOT NULL
		           AND history.last_verified_at < $5::timestamptz - make_interval(months => ai.recurrence_months) THEN 'SUGGESTED_NOW'
		         ELSE ai.default_recommendation_bucket
		       END,
		       array_remove(ARRAY[
		         CASE WHEN history.has_open_work THEN 'OPEN_WORK' END,
		         CASE WHEN ai.risk_tier IN ('HIGH', 'UNKNOWN') THEN 'HIGH_OR_UNKNOWN_RISK' END,
		         CASE WHEN history.last_verified_at IS NOT NULL
		                    AND history.last_verified_at >= $5::timestamptz - make_interval(months => ai.recurrence_months)
		              THEN 'RECENT_FINAL_VERIFICATION' END,
		         CASE WHEN history.last_verified_at IS NOT NULL
		                    AND history.last_verified_at < $5::timestamptz - make_interval(months => ai.recurrence_months)
		              THEN 'RECURRENCE_DUE' END,
		         CASE WHEN canonical_audit_type_matches_question_focus(active_scope.audit_type, ai.inspection_type_codes)
		              THEN 'AUDIT_TYPE_FOCUS_MATCH' END,
		         CASE WHEN NOT canonical_audit_type_matches_question_focus(active_scope.audit_type, ai.inspection_type_codes)
		              THEN 'OUTSIDE_AUDIT_TYPE_FOCUS' END,
		         CASE WHEN ai.external_applicability_unresolved THEN 'SOURCE_CONTEXT_INCOMPLETE' END
	       ], NULL)::text[],ai.recurrence_months,
		       to_char(history.last_verified_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
		       CASE WHEN history.last_verified_at IS NULL THEN NULL ELSE to_char((history.last_verified_at + make_interval(months => ai.recurrence_months)) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') END,
		       ai.external_applicability_unresolved
		FROM canonical_question_catalogs c
		JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id
		JOIN question_versions q ON q.id=m.question_version_id
		JOIN canonical_question_catalog_ai_enrichments ai ON ai.catalog_id=m.catalog_id AND ai.question_version_id=m.question_version_id
		LEFT JOIN canonical_audit_scope_drafts active_scope ON active_scope.id = $4
		LEFT JOIN LATERAL (
		  SELECT
		    COALESCE(bool_or(
		      EXISTS (
		        SELECT 1 FROM potential_findings potential
		        WHERE potential.inspection_id = prior_report.inspection_id
		          AND potential.question_id = q.question_id
		          AND (COALESCE(potential.status, '') IN ('PENDING_LEAD_REVIEW', 'RETURNED')
		            OR (COALESCE(potential.status, '') = 'CONVERTED' AND potential.converted_finding_id IS NULL))
		      )
		      OR EXISTS (
		        SELECT 1
		        FROM findings finding
		        JOIN potential_findings potential ON potential.id = finding.potential_finding_id
		        WHERE finding.inspection_id = prior_report.inspection_id
		          AND potential.question_id = q.question_id
		          AND COALESCE(finding.status, '') <> 'CLOSED'
		      )
		      OR EXISTS (
		        SELECT 1
		        FROM evidence_versions evidence
		        JOIN findings finding ON finding.id = evidence.finding_id
		        JOIN potential_findings potential ON potential.id = finding.potential_finding_id
		        LEFT JOIN evidence_version_states evidence_state ON evidence_state.evidence_version_id = evidence.id
		        WHERE finding.inspection_id = prior_report.inspection_id
		          AND potential.question_id = q.question_id
		          AND evidence.version = (SELECT max(latest.version) FROM evidence_versions latest WHERE latest.evidence_id = evidence.evidence_id)
		          AND (COALESCE(evidence_state.scan_state, '') <> 'CLEAN'
		            OR COALESCE(evidence_state.review_state, '') NOT IN ('PENDING_CAA_REVIEW', 'ACCEPTED'))
		      )
		    ), false) AS has_open_work,
		    max(prior_state.issued_at) FILTER (
		      WHERE prior_report.snapshot->>'kind' = 'FINAL'
		        AND prior_state.status = 'LOCKED'
		        AND prior_state.issued_at IS NOT NULL
		        AND NOT EXISTS (
		          SELECT 1 FROM potential_findings potential
		          WHERE potential.inspection_id = prior_report.inspection_id
		            AND potential.question_id = q.question_id
		            AND (COALESCE(potential.status, '') IN ('PENDING_LEAD_REVIEW', 'RETURNED')
		              OR (COALESCE(potential.status, '') = 'CONVERTED' AND potential.converted_finding_id IS NULL))
		        )
		        AND NOT EXISTS (
		          SELECT 1
		          FROM findings finding
		          JOIN potential_findings potential ON potential.id = finding.potential_finding_id
		          WHERE finding.inspection_id = prior_report.inspection_id
		            AND potential.question_id = q.question_id
		            AND COALESCE(finding.status, '') <> 'CLOSED'
		        )
		        AND NOT EXISTS (
		          SELECT 1
		          FROM evidence_versions evidence
		          JOIN findings finding ON finding.id = evidence.finding_id
		          JOIN potential_findings potential ON potential.id = finding.potential_finding_id
		          LEFT JOIN evidence_version_states evidence_state ON evidence_state.evidence_version_id = evidence.id
		          WHERE finding.inspection_id = prior_report.inspection_id
		            AND potential.question_id = q.question_id
		            AND evidence.version = (SELECT max(latest.version) FROM evidence_versions latest WHERE latest.evidence_id = evidence.evidence_id)
		            AND (COALESCE(evidence_state.scan_state, '') <> 'CLEAN'
		              OR COALESCE(evidence_state.review_state, '') NOT IN ('PENDING_CAA_REVIEW', 'ACCEPTED'))
		        )
		        AND EXISTS (
		          SELECT 1 FROM checklist_responses response
		          WHERE response.inspection_id = prior_report.inspection_id
		            AND response.question_id = q.question_id
		            AND response.response_value = 'COMPLIANT'
		            AND NOT EXISTS (
		              SELECT 1 FROM potential_findings response_potential
		              WHERE response_potential.checklist_response_id = response.id
		                AND COALESCE(response_potential.status, '') <> 'DISMISSED'
		            )
		        )
		    ) AS last_verified_at
		  FROM inspection_packages prior_package
		  JOIN canonical_audit_scope_snapshots prior_snapshot ON prior_snapshot.id = prior_package.canonical_scope_snapshot_id
		  JOIN canonical_audit_scope_drafts prior_scope ON prior_scope.id = prior_snapshot.scope_draft_id
		  JOIN canonical_audit_scope_snapshot_questions prior_question ON prior_question.snapshot_id = prior_snapshot.id AND prior_question.question_version_id = m.question_version_id
		  JOIN report_versions prior_report ON prior_report.inspection_id = prior_package.inspection_id
		  JOIN report_approval_states prior_state ON prior_state.report_version_id = prior_report.id
		  WHERE active_scope.id IS NOT NULL
		    AND prior_scope.organization_id = active_scope.organization_id
		    AND prior_scope.provider_scope_id = active_scope.provider_scope_id
		    AND prior_scope.regulated_target_id = active_scope.regulated_target_id
		    AND prior_scope.audit_type = active_scope.audit_type
		) history ON TRUE
		WHERE c.catalog_version=$1 AND c.usage_class=$2 AND c.status='SEALED'
		  AND c.source_origin='IMPORTED_APPROVED_SOURCE'
		  AND m.question_version_id=$3
		  AND COALESCE((SELECT status FROM canonical_question_catalog_membership_events event
		                WHERE event.catalog_id=m.catalog_id AND event.question_version_id=m.question_version_id
		                ORDER BY occurred_at DESC,event_id DESC LIMIT 1),'AVAILABLE')='AVAILABLE'
		  AND ($4='' OR EXISTS (
			SELECT 1 FROM canonical_question_catalog_applicabilities applicability
			JOIN canonical_audit_scope_drafts scope ON scope.id=$4
			WHERE applicability.catalog_id=c.id AND applicability.question_version_id=m.question_version_id
			  AND applicability.provider_scope_id=scope.provider_scope_id AND applicability.regulated_target_id=scope.regulated_target_id
			  AND applicability.status='ELIGIBLE'
		  ))
		`, catalogVersion, string(usage), questionVersionID, scopeID, api.clock().UTC()).Scan(&row.CatalogVersion, &row.UsageClass, &row.QuestionID, &row.FormCode, &row.ProposalID, &row.Ordinal, &row.Digest, &row.SourceLocator, &row.SourceGap, &row.Domain, &row.Topic, &row.RiskBand, &row.Prompt, &row.ConfiguredReference, &row.ExpectedEvidence, &row.GovernedCandidateID, &row.GovernedCandidateRevision, &row.GovernedCandidateContentDigest, &row.GovernedCandidateStatus, &row.ReviewRevision, &row.ReviewDisposition, &row.ReviewReason, &row.ReviewDomain, &row.ReviewTopic, &advisoryDomain, &advisoryTopics, &advisoryInspectionTypes, &advisoryInspectionProfiles, &advisoryApplicability, &advisoryRiskTier, &advisorySafetyCritical, &advisoryConfidence, &advisoryState, &advisoryReasons, &advisoryRecurrence, &previouslyVerifiedAt, &recurrenceDueAt, &advisoryUnresolved)
	row.AIAdvisory = generated.CanonicalQuestionAIAdvisory{DomainCode: advisoryDomain, TopicCodes: advisoryTopics, InspectionTypeCodes: advisoryInspectionTypes, InspectionProfileCodes: advisoryInspectionProfiles, ApplicabilityDisposition: advisoryApplicability, RiskTier: advisoryRiskTier, SafetyCritical: advisorySafetyCritical, AgreementConfidence: advisoryConfidence, AdvisoryState: advisoryState, RecommendationReasonCodes: advisoryReasons, RecurrenceMonths: advisoryRecurrence, PreviouslyVerifiedAt: previouslyVerifiedAt, RecurrenceDueAt: recurrenceDueAt, ExternalApplicabilityUnresolved: advisoryUnresolved}
	if errors.Is(err, pgx.ErrNoRows) {
		err = application.ErrNotFound
	}
	if err == nil {
		row.ScopeID = scopeID
		row.ReviewHistory, err = api.loadQuestionReviewHistory(request.Context(), row)
	}
	api.respond(writer, canonicalCatalogEntry(row), err)
}

type canonicalScopeState struct {
	CatalogID                     string
	CatalogVersion                string
	UsageClass                    string
	AuditType                     string
	OrganizationID                string
	ProviderScopeID               string
	RegulatedTargetID             string
	GovernedPublicationDecisionID *string
	GovernedCandidateID           *string
	GovernedCandidateRevision     *int64
	GovernedCandidateDigest       *string
	CurrentDigest                 string
	Selected                      []string
}

type canonicalSelectionOperationKind string

const (
	canonicalSelectionAdd     canonicalSelectionOperationKind = "ADD"
	canonicalSelectionRemove  canonicalSelectionOperationKind = "REMOVE"
	canonicalSelectionReplace canonicalSelectionOperationKind = "REPLACE"
)

func parseCanonicalSelectionOperationKind(raw string) (canonicalSelectionOperationKind, error) {
	switch canonicalSelectionOperationKind(strings.ToUpper(strings.TrimSpace(raw))) {
	case "", canonicalSelectionReplace:
		return canonicalSelectionReplace, nil
	case canonicalSelectionAdd:
		return canonicalSelectionAdd, nil
	case canonicalSelectionRemove:
		return canonicalSelectionRemove, nil
	default:
		return "", fmt.Errorf("%w: unsupported canonical selection operation", application.ErrInvalid)
	}
}

func (api *CanonicalAPI) loadCanonicalScopeSelection(ctx context.Context, scopeID string) ([]string, error) {
	var operationID string
	err := api.pool.QueryRow(ctx, `
		SELECT id
		FROM canonical_audit_scope_selection_operations
		WHERE scope_draft_id=$1 AND operation_kind <> 'PREVIEW'
		ORDER BY created_at DESC,id DESC LIMIT 1
	`, scopeID).Scan(&operationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := api.pool.Query(ctx, `
		SELECT question_version_id
		FROM canonical_audit_scope_selection_questions
		WHERE operation_id=$1 ORDER BY position
	`, operationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	selection := []string{}
	for rows.Next() {
		var questionID string
		if err := rows.Scan(&questionID); err != nil {
			return nil, err
		}
		selection = append(selection, questionID)
	}
	return selection, rows.Err()
}

func applyCanonicalSelectionOperation(current, affected []string, kind canonicalSelectionOperationKind) []string {
	if kind == canonicalSelectionReplace {
		return append([]string(nil), affected...)
	}
	seen := make(map[string]struct{}, len(current)+len(affected))
	result := make([]string, 0, len(current)+len(affected))
	for _, id := range current {
		if kind == canonicalSelectionRemove {
			remove := false
			for _, affectedID := range affected {
				if affectedID == id {
					remove = true
					break
				}
			}
			if remove {
				continue
			}
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	if kind == canonicalSelectionAdd {
		for _, id := range affected {
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

func (api *CanonicalAPI) readCanonicalScopeState(ctx context.Context, scopeID string, actor identity.Principal) (canonicalScopeState, error) {
	if api.pool == nil {
		return canonicalScopeState{}, application.ErrNotFound
	}
	if err := api.requireCanonicalScopeOwner(ctx, scopeID, actor.SubjectID); err != nil {
		return canonicalScopeState{}, err
	}
	var state canonicalScopeState
	if err := api.pool.QueryRow(ctx, `
		SELECT c.id,c.catalog_version,c.usage_class,s.audit_type,s.organization_id,s.provider_scope_id,s.regulated_target_id,
		       c.governed_publication_decision_id,c.governed_candidate_draft_version_id,
		       c.governed_candidate_revision,c.governed_candidate_content_digest,
		       s.selection_digest
		FROM canonical_audit_scope_drafts s
		JOIN canonical_question_catalogs c ON c.id=s.catalog_id
		WHERE s.id=$1 AND s.status IN ('DRAFT','SUBMITTED','RELEASED')
	`, scopeID).Scan(&state.CatalogID, &state.CatalogVersion, &state.UsageClass, &state.AuditType, &state.OrganizationID, &state.ProviderScopeID, &state.RegulatedTargetID,
		&state.GovernedPublicationDecisionID, &state.GovernedCandidateID, &state.GovernedCandidateRevision,
		&state.GovernedCandidateDigest, &state.CurrentDigest); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return canonicalScopeState{}, application.ErrNotFound
		}
		return canonicalScopeState{}, err
	}
	if state.CurrentDigest == "" {
		state.CurrentDigest = questioncatalog.SelectionDigest(nil)
	}
	// Re-resolve current provider/department authority in a transaction for
	// every scope read used by preview/commit. Ownership alone is insufficient:
	// a creator who has since lost the effective responsibility must receive no
	// selected-set or catalog signal and cannot mutate the draft.
	if err := database.WithinTransaction(ctx, api.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := application.ValidateCanonicalScopeMap(ctx, tx, actor, map[string]any{
			"organizationId":    state.OrganizationID,
			"applicationType":   state.AuditType,
			"catalogVersion":    state.CatalogVersion,
			"scopeDraftId":      scopeID,
			"selectionDigest":   state.CurrentDigest,
			"providerScopeId":   state.ProviderScopeID,
			"regulatedTargetId": state.RegulatedTargetID,
		}, false)
		return err
	}); err != nil {
		return canonicalScopeState{}, err
	}
	var payload []byte
	var operationID string
	err := api.pool.QueryRow(ctx, `
		SELECT id
		FROM canonical_audit_scope_selection_operations
		WHERE scope_draft_id=$1 AND operation_kind <> 'PREVIEW'
		ORDER BY created_at DESC,id DESC LIMIT 1
	`, scopeID).Scan(&operationID)
	if err == nil && operationID != "" {
		rows, queryErr := api.pool.Query(ctx, `
			SELECT question_version_id
			FROM canonical_audit_scope_selection_questions
			WHERE operation_id=$1 ORDER BY position
		`, operationID)
		if queryErr == nil {
			defer rows.Close()
			for rows.Next() {
				var id string
				if scanErr := rows.Scan(&id); scanErr != nil {
					return canonicalScopeState{}, scanErr
				}
				state.Selected = append(state.Selected, id)
			}
			if scanErr := rows.Err(); scanErr != nil {
				return canonicalScopeState{}, scanErr
			}
		} else {
			// Older local rows may only contain the receipt JSON. The relation is
			// authoritative for new writes, but this read fallback is limited to
			// the same immutable operation and never accepts question bodies.
			if err := api.pool.QueryRow(ctx, `SELECT affected_question_version_ids FROM canonical_audit_scope_selection_operations WHERE id=$1`, operationID).Scan(&payload); err == nil {
				_ = json.Unmarshal(payload, &state.Selected)
			}
		}
	}
	return state, nil
}

func validateCanonicalSelection(ctx context.Context, pool *database.Pool, state canonicalScopeState, ids []string, usage questioncatalog.UsageClass) error {
	if usage != questioncatalog.UsageClass(state.UsageClass) {
		return fmt.Errorf("%w: selection usage class does not match scope", application.ErrInvalid)
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("%w: question version id is empty", application.ErrInvalid)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%w: duplicate question version id", application.ErrInvalid)
		}
		seen[id] = struct{}{}
	}
	var count int64
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM canonical_question_catalog_memberships membership
		WHERE membership.catalog_id=$1 AND membership.usage_class=$2
		  AND membership.question_version_id = ANY($3::text[])
		  AND COALESCE((SELECT status FROM canonical_question_catalog_membership_events event
		                WHERE event.catalog_id=membership.catalog_id
		                  AND event.question_version_id=membership.question_version_id
		                ORDER BY occurred_at DESC,event_id DESC LIMIT 1),'AVAILABLE')='AVAILABLE'
	`, state.CatalogID, string(usage), ids).Scan(&count); err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return fmt.Errorf("%w: selection contains a question outside the catalog", application.ErrInvalid)
	}
	{
		var applicableCount int64
		if err := pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM canonical_question_catalog_applicabilities applicability
			JOIN canonical_question_catalogs catalog ON catalog.id=applicability.catalog_id
			WHERE applicability.catalog_id=$1
			  AND applicability.provider_scope_id=$2
			  AND applicability.regulated_target_id=$3
			  AND applicability.status='ELIGIBLE'
			  AND applicability.question_version_id=ANY($4::text[])
			  AND catalog.source_origin='IMPORTED_APPROVED_SOURCE'
		`, state.CatalogID, state.ProviderScopeID, state.RegulatedTargetID, ids).Scan(&applicableCount); err != nil {
			return err
		}
		if applicableCount != int64(len(ids)) {
			return fmt.Errorf("%w: selection contains a question without provider-scope/target eligibility", application.ErrForbidden)
		}
	}
	var protectedOmissions int64
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM canonical_question_catalog_ai_enrichments enrichment
		WHERE enrichment.catalog_id=$1
		  AND enrichment.mandatory_control=true
		  AND NOT (enrichment.question_version_id = ANY($2::text[]))
	`, state.CatalogID, ids).Scan(&protectedOmissions); err != nil {
		return err
	}
	if protectedOmissions > 0 {
		return fmt.Errorf("%w: selection omits a mandatory server-protected question", application.ErrConflict)
	}
	return nil
}

func validateCanonicalSelectionBatch(ctx context.Context, pool *database.Pool, state canonicalScopeState, ids []string, usage questioncatalog.UsageClass) error {
	if len(ids) == 0 || len(ids) > 500 {
		return fmt.Errorf("%w: selection batch must contain between 1 and 500 questions", application.ErrInvalid)
	}
	return validateCanonicalSelection(ctx, pool, state, ids, usage)
}

func canonicalSelectionDigest(ids []string) string { return questioncatalog.SelectionDigest(ids) }

// canonicalSelectionSummary is calculated from the immutable catalog
// membership rows on the server.  The client may display it, but it cannot
// invent a resource estimate or distribution for a selected question set.
func (api *CanonicalAPI) canonicalSelectionSummary(ctx context.Context, catalogID string, ids []string) (map[string]any, map[string]any, int64, error) {
	forms := make(map[string]any)
	domains := make(map[string]any)
	if len(ids) == 0 {
		return forms, domains, 0, nil
	}
	rows, err := api.pool.Query(ctx, `
		SELECT membership.form_code, COALESCE(NULLIF(membership.proposed_domain, ''), 'Unclassified')
		FROM canonical_question_catalog_memberships membership
		WHERE membership.catalog_id=$1 AND membership.question_version_id = ANY($2::text[])
		ORDER BY membership.ordinal, membership.question_version_id
	`, catalogID, ids)
	if err != nil {
		return nil, nil, 0, err
	}
	defer rows.Close()
	var count int64
	for rows.Next() {
		var formCode, domain string
		if err := rows.Scan(&formCode, &domain); err != nil {
			return nil, nil, 0, err
		}
		forms[formCode] = int64Value(forms[formCode]) + 1
		domains[domain] = int64Value(domains[domain]) + 1
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, nil, 0, err
	}
	if count != int64(len(ids)) {
		return nil, nil, 0, fmt.Errorf("%w: selection summary did not resolve every immutable question", application.ErrConflict)
	}
	// Until a catalog publishes a separately governed effort model, one
	// selected immutable question is one server-derived question-hour. This is
	// deliberately explicit and reproducible rather than a client estimate.
	return forms, domains, count, nil
}

func int64Value(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func (api *CanonicalAPI) previewCanonicalAuditScopeSelection(writer http.ResponseWriter, request *http.Request) {
	actor, ok := api.requireCanonicalCatalogActor(writer, request, true)
	if !ok {
		return
	}
	var input generated.CanonicalAuditScopeSelectionInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	usage, err := parseQuestionUsageClass(string(input.UsageClass))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	rawOperationKind := ""
	if input.OperationKind != nil {
		rawOperationKind = *input.OperationKind
	}
	operationKind, err := parseCanonicalSelectionOperationKind(rawOperationKind)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if len(input.QuestionVersionIds) == 0 || len(input.QuestionVersionIds) > 500 {
		api.respond(writer, nil, fmt.Errorf("%w: selection preview is bounded to 500 question identities per batch", application.ErrInvalid))
		return
	}
	state, err := api.readCanonicalScopeState(request.Context(), chi.URLParam(request, "scopeId"), actor)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if err := validateCanonicalSelectionBatch(request.Context(), api.pool, state, input.QuestionVersionIds, usage); err != nil {
		api.respond(writer, nil, err)
		return
	}
	if strings.TrimSpace(input.OperationId) == "" {
		api.respond(writer, nil, fmt.Errorf("%w: preview operationId is required", application.ErrInvalid))
		return
	}
	if len(input.QuestionVersionIds) > 500 {
		api.respond(writer, nil, fmt.Errorf("%w: preview is bounded to 500 questions per batch", application.ErrInvalid))
		return
	}
	currentSelection, err := api.loadCanonicalScopeSelection(request.Context(), chi.URLParam(request, "scopeId"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	resultSelection := applyCanonicalSelectionOperation(currentSelection, input.QuestionVersionIds, operationKind)
	if err := validateCanonicalSelection(request.Context(), api.pool, state, resultSelection, usage); err != nil {
		api.respond(writer, nil, err)
		return
	}
	digest := canonicalSelectionDigest(resultSelection)
	expectedDigest := input.ExpectedSelectionDigest
	if expectedDigest == "" {
		expectedDigest = questioncatalog.SelectionDigest(nil)
	}
	valid := expectedDigest == state.CurrentDigest
	reason := "selection is ready to commit"
	if !valid {
		reason = "selection digest is stale; reload the current scope"
	}
	if valid {
		payload, _ := json.Marshal(input.QuestionVersionIds)
		filterPayload, _ := json.Marshal(map[string]any{"operationKind": string(operationKind), "affectedQuestionVersionIds": input.QuestionVersionIds, "filter": input.Filter})
		previewExpiry := api.clock().UTC().Add(5 * time.Minute)
		err = database.WithinTransaction(request.Context(), api.pool, func(ctx context.Context, tx pgx.Tx) error {
			var currentDigest, currentOrganizationID, currentCatalogVersion, currentProviderScopeID, currentTargetID string
			if err := tx.QueryRow(ctx, `
				SELECT scope.selection_digest, scope.organization_id, catalog.catalog_version,
				       scope.provider_scope_id, scope.regulated_target_id
				FROM canonical_audit_scope_drafts scope
				JOIN canonical_question_catalogs catalog ON catalog.id = scope.catalog_id
				WHERE scope.id=$1 AND scope.created_by_subject_id=$2 AND scope.status='DRAFT'
				FOR UPDATE OF scope
			`, chi.URLParam(request, "scopeId"), actor.SubjectID).Scan(&currentDigest, &currentOrganizationID, &currentCatalogVersion, &currentProviderScopeID, &currentTargetID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return application.ErrNotFound
				}
				return err
			}
			if currentDigest == "" {
				currentDigest = questioncatalog.SelectionDigest(nil)
			}
			if currentDigest != expectedDigest {
				return application.ErrConflict
			}
			if _, err := application.ValidateCanonicalScopeMap(ctx, tx, actor, map[string]any{
				"organizationId":    currentOrganizationID,
				"applicationType":   state.AuditType,
				"catalogVersion":    currentCatalogVersion,
				"scopeDraftId":      chi.URLParam(request, "scopeId"),
				"selectionDigest":   currentDigest,
				"providerScopeId":   currentProviderScopeID,
				"regulatedTargetId": currentTargetID,
			}, false); err != nil {
				return err
			}
			var existingDigest string
			var existingPayload []byte
			var existingExpiry *time.Time
			err := tx.QueryRow(ctx, `
				SELECT result_digest, affected_question_version_ids, expires_at
				FROM canonical_audit_scope_selection_operations
				WHERE scope_draft_id=$1 AND operation_id=$2 AND operation_kind='PREVIEW'
			`, chi.URLParam(request, "scopeId"), input.OperationId).Scan(&existingDigest, &existingPayload, &existingExpiry)
			if err == nil {
				if existingDigest != digest || string(existingPayload) != string(payload) || (existingExpiry != nil && !api.clock().UTC().Before(*existingExpiry)) {
					return fmt.Errorf("%w: preview operation replay differs or has expired", application.ErrConflict)
				}
				return nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			operationID := "scope-selection-preview:" + chi.URLParam(request, "scopeId") + ":" + input.OperationId
			idempotencyKey := input.OperationId
			if input.IdempotencyKey != nil && strings.TrimSpace(*input.IdempotencyKey) != "" {
				idempotencyKey = strings.TrimSpace(*input.IdempotencyKey)
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO canonical_audit_scope_selection_operations
				(id,scope_draft_id,operation_id,idempotency_key,operation_kind,expected_digest,result_digest,affected_question_version_ids,filter_payload,actor_subject_id,expires_at)
				VALUES ($1,$2,$3,$4,'PREVIEW',$5,$6,$7,$8,$9,$10)
			`, operationID, chi.URLParam(request, "scopeId"), input.OperationId, idempotencyKey, expectedDigest, digest, payload, filterPayload, actor.SubjectID, previewExpiry)
			return err
		})
		if err != nil {
			api.respond(writer, nil, err)
			return
		}
	}
	formDistribution, domainDistribution, resourceRequirement, err := api.canonicalSelectionSummary(request.Context(), state.CatalogID, resultSelection)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respond(writer, generated.CanonicalAuditScopeSelectionPreview{
		Preview:       generated.AuditScopeSelectionDigest{SelectionDigest: digest, SelectedQuestionVersionIds: resultSelection, SelectedCount: int64(len(resultSelection)), CatalogVersion: state.CatalogVersion, UsageClass: generated.QuestionUsageClass(usage), FormDistribution: formDistribution, DomainDistribution: domainDistribution, EstimatedResourceRequirement: resourceRequirement},
		AffectedCount: int64(len(input.QuestionVersionIds)), Valid: valid, Reason: reason,
	}, nil)
}

func (api *CanonicalAPI) commitCanonicalAuditScopeSelection(writer http.ResponseWriter, request *http.Request) {
	actor, ok := api.requireCanonicalCatalogActor(writer, request, true)
	if !ok {
		return
	}
	var input generated.CanonicalAuditScopeSelectionInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	previewOperationID := ""
	if input.PreviewOperationId != nil {
		previewOperationID = strings.TrimSpace(*input.PreviewOperationId)
	}
	usage, err := parseQuestionUsageClass(string(input.UsageClass))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	rawOperationKind := ""
	if input.OperationKind != nil {
		rawOperationKind = *input.OperationKind
	}
	operationKind, err := parseCanonicalSelectionOperationKind(rawOperationKind)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if len(input.QuestionVersionIds) == 0 || len(input.QuestionVersionIds) > 500 {
		api.respond(writer, nil, fmt.Errorf("%w: selection commit is bounded to 500 question identities per batch", application.ErrInvalid))
		return
	}
	state, err := api.readCanonicalScopeState(request.Context(), chi.URLParam(request, "scopeId"), actor)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if strings.TrimSpace(input.OperationId) == "" {
		api.respond(writer, nil, fmt.Errorf("%w: operationId is required", application.ErrInvalid))
		return
	}
	if previewOperationID == "" {
		api.respond(writer, nil, fmt.Errorf("%w: an unexpired single-use selection preview is required", application.ErrConflict))
		return
	}
	expectedDigest := input.ExpectedSelectionDigest
	if expectedDigest == "" {
		expectedDigest = questioncatalog.SelectionDigest(nil)
	}
	if err := validateCanonicalSelectionBatch(request.Context(), api.pool, state, input.QuestionVersionIds, usage); err != nil {
		api.respond(writer, nil, err)
		return
	}
	currentSelection, err := api.loadCanonicalScopeSelection(request.Context(), chi.URLParam(request, "scopeId"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	resultSelection := applyCanonicalSelectionOperation(currentSelection, input.QuestionVersionIds, operationKind)
	if err := validateCanonicalSelection(request.Context(), api.pool, state, resultSelection, usage); err != nil {
		api.respond(writer, nil, err)
		return
	}
	digest := canonicalSelectionDigest(resultSelection)
	var previewDigest string
	var previewPayload []byte
	var previewExpiry *time.Time
	var previewActor string
	var previewFilter []byte
	if err := api.pool.QueryRow(request.Context(), `
		SELECT result_digest, affected_question_version_ids, expires_at, actor_subject_id, filter_payload
		FROM canonical_audit_scope_selection_operations
		WHERE scope_draft_id=$1 AND operation_id=$2 AND operation_kind='PREVIEW'
	`, chi.URLParam(request, "scopeId"), previewOperationID).Scan(&previewDigest, &previewPayload, &previewExpiry, &previewActor, &previewFilter); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = fmt.Errorf("%w: selection preview is missing", application.ErrConflict)
		}
		api.respond(writer, nil, err)
		return
	}
	var previewIDs []string
	var previewMetadata struct {
		OperationKind       string   `json:"operationKind"`
		AffectedQuestionIDs []string `json:"affectedQuestionVersionIds"`
	}
	if json.Unmarshal(previewPayload, &previewIDs) != nil || json.Unmarshal(previewFilter, &previewMetadata) != nil ||
		previewActor != actor.SubjectID || previewMetadata.OperationKind != string(operationKind) ||
		previewDigest != digest || len(previewIDs) != len(input.QuestionVersionIds) ||
		len(previewMetadata.AffectedQuestionIDs) != len(input.QuestionVersionIds) ||
		(previewExpiry != nil && !api.clock().UTC().Before(*previewExpiry)) {
		api.respond(writer, nil, fmt.Errorf("%w: selection preview is stale, expired, or owned by another actor", application.ErrConflict))
		return
	}
	for index := range previewIDs {
		if previewIDs[index] != input.QuestionVersionIds[index] || previewMetadata.AffectedQuestionIDs[index] != input.QuestionVersionIds[index] {
			api.respond(writer, nil, fmt.Errorf("%w: selection preview does not match commit order", application.ErrConflict))
			return
		}
	}
	// A retry with the same operation is an idempotent receipt even when a
	// later selection has advanced the draft digest. Check the immutable
	// operation row before enforcing the current-digest CAS for new work.
	var priorDigest string
	var priorPayload []byte
	var priorFilter []byte
	priorErr := api.pool.QueryRow(request.Context(), `SELECT result_digest, affected_question_version_ids, filter_payload FROM canonical_audit_scope_selection_operations WHERE scope_draft_id=$1 AND operation_id=$2 AND operation_kind <> 'PREVIEW'`, chi.URLParam(request, "scopeId"), input.OperationId).Scan(&priorDigest, &priorPayload, &priorFilter)
	if priorErr == nil {
		var priorIDs []string
		var priorMetadata struct {
			OperationKind       string   `json:"operationKind"`
			AffectedQuestionIDs []string `json:"affectedQuestionVersionIds"`
		}
		if json.Unmarshal(priorPayload, &priorIDs) != nil || json.Unmarshal(priorFilter, &priorMetadata) != nil || priorDigest != digest || priorMetadata.OperationKind != string(operationKind) || len(priorMetadata.AffectedQuestionIDs) != len(input.QuestionVersionIds) {
			api.respond(writer, nil, fmt.Errorf("%w: operation replay payload differs", application.ErrConflict))
			return
		}
		for i := range input.QuestionVersionIds {
			if priorMetadata.AffectedQuestionIDs[i] != input.QuestionVersionIds[i] {
				api.respond(writer, nil, fmt.Errorf("%w: operation replay payload differs", application.ErrConflict))
				return
			}
		}
		formDistribution, domainDistribution, resourceRequirement, summaryErr := api.canonicalSelectionSummary(request.Context(), state.CatalogID, priorIDs)
		if summaryErr != nil {
			api.respond(writer, nil, summaryErr)
			return
		}
		api.respond(writer, generated.CanonicalAuditScopeSelectionReceipt{OperationId: input.OperationId, Replayed: true, Selection: generated.AuditScopeSelectionDigest{SelectionDigest: priorDigest, SelectedQuestionVersionIds: priorIDs, SelectedCount: int64(len(priorIDs)), CatalogVersion: state.CatalogVersion, UsageClass: generated.QuestionUsageClass(usage), FormDistribution: formDistribution, DomainDistribution: domainDistribution, EstimatedResourceRequirement: resourceRequirement}}, nil)
		return
	}
	if !errors.Is(priorErr, pgx.ErrNoRows) {
		api.respond(writer, nil, priorErr)
		return
	}
	if expectedDigest != state.CurrentDigest {
		api.respond(writer, nil, application.ErrConflict)
		return
	}
	operationKey := input.OperationId
	if input.IdempotencyKey != nil && strings.TrimSpace(*input.IdempotencyKey) != "" {
		operationKey = *input.IdempotencyKey
	}
	payload, _ := json.Marshal(resultSelection)
	filterPayload, _ := json.Marshal(map[string]any{"operationKind": string(operationKind), "affectedQuestionVersionIds": input.QuestionVersionIds, "filter": input.Filter})
	var replayed bool
	err = database.WithinTransaction(request.Context(), api.pool, func(ctx context.Context, tx pgx.Tx) error {
		operationID := "scope-selection:" + chi.URLParam(request, "scopeId") + ":" + input.OperationId
		var storedDigest string
		var storedPayload []byte
		var storedFilter []byte
		storedErr := tx.QueryRow(ctx, `
				SELECT result_digest, affected_question_version_ids, filter_payload
				FROM canonical_audit_scope_selection_operations
				WHERE scope_draft_id=$1 AND operation_id=$2 AND operation_kind <> 'PREVIEW'
			`, chi.URLParam(request, "scopeId"), input.OperationId).Scan(&storedDigest, &storedPayload, &storedFilter)
		if storedErr == nil {
			var storedIDs []string
			var storedMetadata struct {
				OperationKind       string   `json:"operationKind"`
				AffectedQuestionIDs []string `json:"affectedQuestionVersionIds"`
			}
			if json.Unmarshal(storedPayload, &storedIDs) != nil || json.Unmarshal(storedFilter, &storedMetadata) != nil || storedDigest != digest || storedMetadata.OperationKind != string(operationKind) || len(storedMetadata.AffectedQuestionIDs) != len(input.QuestionVersionIds) {
				return fmt.Errorf("%w: operation replay payload differs", application.ErrConflict)
			}
			for index := range input.QuestionVersionIds {
				if storedMetadata.AffectedQuestionIDs[index] != input.QuestionVersionIds[index] {
					return fmt.Errorf("%w: operation replay payload differs", application.ErrConflict)
				}
			}
			resultSelection = storedIDs
			replayed = true
			return nil
		}
		if !errors.Is(storedErr, pgx.ErrNoRows) {
			return storedErr
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO canonical_audit_scope_selection_preview_consumptions
			(preview_operation_id, commit_operation_id, actor_subject_id, consumed_at)
			VALUES ($1,$2,$3,$4)
			`, previewOperationID, input.OperationId, actor.SubjectID, api.clock().UTC()); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return fmt.Errorf("%w: selection preview has already been consumed", application.ErrConflict)
			}
			return err
		}
		var currentDigest, currentOrganizationID, currentCatalogVersion, currentProviderScopeID, currentTargetID string
		if err := tx.QueryRow(ctx, `
			SELECT scope.selection_digest, scope.organization_id, catalog.catalog_version,
			       scope.provider_scope_id, scope.regulated_target_id
			FROM canonical_audit_scope_drafts scope
			JOIN canonical_question_catalogs catalog ON catalog.id = scope.catalog_id
			WHERE scope.id=$1 AND scope.created_by_subject_id=$2 AND scope.status='DRAFT'
			FOR UPDATE OF scope
		`, chi.URLParam(request, "scopeId"), actor.SubjectID).Scan(&currentDigest, &currentOrganizationID, &currentCatalogVersion, &currentProviderScopeID, &currentTargetID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return application.ErrNotFound
			}
			return err
		}
		if currentDigest == "" {
			currentDigest = questioncatalog.SelectionDigest(nil)
		}
		if _, err := application.ValidateCanonicalScopeMap(ctx, tx, actor, map[string]any{
			"organizationId":    currentOrganizationID,
			"applicationType":   state.AuditType,
			"catalogVersion":    currentCatalogVersion,
			"scopeDraftId":      chi.URLParam(request, "scopeId"),
			"selectionDigest":   currentDigest,
			"providerScopeId":   currentProviderScopeID,
			"regulatedTargetId": currentTargetID,
		}, false); err != nil {
			return err
		}
		if currentDigest != expectedDigest {
			return application.ErrConflict
		}
		_, err := tx.Exec(ctx, `
				INSERT INTO canonical_audit_scope_selection_operations
				(id,scope_draft_id,operation_id,idempotency_key,operation_kind,expected_digest,result_digest,affected_question_version_ids,filter_payload,actor_subject_id)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			`, operationID, chi.URLParam(request, "scopeId"), input.OperationId, operationKey, string(operationKind), expectedDigest, digest, payload, filterPayload, requestPrincipalSubject(request))
		if err != nil {
			return err
		}
		for position, questionVersionID := range resultSelection {
			if _, err := tx.Exec(ctx, `INSERT INTO canonical_audit_scope_selection_questions (operation_id,catalog_id,question_version_id,position,selection_digest) VALUES ($1,$2,$3,$4,$5)`, operationID, state.CatalogID, questionVersionID, position, digest); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `UPDATE canonical_audit_scope_drafts SET selected_question_count=$2, selection_digest=$3, updated_at=now() WHERE id=$1`, chi.URLParam(request, "scopeId"), len(resultSelection), digest)
		return err
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			api.respond(writer, nil, application.ErrConflict)
			return
		}
		api.respond(writer, nil, err)
		return
	}
	formDistribution, domainDistribution, resourceRequirement, summaryErr := api.canonicalSelectionSummary(request.Context(), state.CatalogID, resultSelection)
	if summaryErr != nil {
		api.respond(writer, nil, summaryErr)
		return
	}
	api.respond(writer, generated.CanonicalAuditScopeSelectionReceipt{OperationId: input.OperationId, Replayed: replayed, Selection: generated.AuditScopeSelectionDigest{SelectionDigest: digest, SelectedQuestionVersionIds: resultSelection, SelectedCount: int64(len(resultSelection)), CatalogVersion: state.CatalogVersion, UsageClass: generated.QuestionUsageClass(usage), FormDistribution: formDistribution, DomainDistribution: domainDistribution, EstimatedResourceRequirement: resourceRequirement}}, nil)
}

func requestPrincipalSubject(request *http.Request) string {
	principal, _ := PrincipalFromContext(request.Context())
	return principal.SubjectID
}

// governedLifecycleReplay reports whether the lifecycle command already has a
// durable decision.  The lifecycle service still performs the authoritative
// transaction-level replay and current-owner check; this read only lets the
// HTTP response truthfully distinguish a replay from a newly committed
// approval/publication.
func (api *CanonicalAPI) governedLifecycleReplay(ctx context.Context, actor identity.Principal, command checklistgovernance.ReviewCommand, action checklistgovernance.QuestionReviewAction) (bool, error) {
	if api.pool == nil {
		return false, application.ErrNotFound
	}
	var candidateID, operationID, idempotencyKey, actorSubject, reason, decision string
	var revision int64
	var digest string
	var err error
	switch action {
	case checklistgovernance.QuestionReviewActionTechnicalApprove:
		err = api.pool.QueryRow(ctx, `
			SELECT candidate_draft_version_id, operation_id, idempotency_key,
			       candidate_revision, candidate_content_digest, actor_subject_id,
			       reason, decision
			FROM department_review_decisions
			WHERE (operation_id=$1 OR idempotency_key=$2) AND decision='TECHNICALLY_APPROVED'
			ORDER BY decided_at DESC, id DESC LIMIT 1
		`, command.OperationID, command.IdempotencyKey).Scan(&candidateID, &operationID, &idempotencyKey,
			&revision, &digest, &actorSubject, &reason, &decision)
	case checklistgovernance.QuestionReviewActionPublish:
		err = api.pool.QueryRow(ctx, `
			SELECT candidate_draft_version_id, operation_id, idempotency_key,
			       candidate_revision, candidate_content_digest, actor_subject_id,
			       reason, 'PUBLISHED'
			FROM checklist_publication_decisions
			WHERE (operation_id=$1 OR idempotency_key=$2)
			ORDER BY decided_at DESC, id DESC LIMIT 1
		`, command.OperationID, command.IdempotencyKey).Scan(&candidateID, &operationID, &idempotencyKey,
			&revision, &digest, &actorSubject, &reason, &decision)
	default:
		return false, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if candidateID != command.CandidateID || operationID != command.OperationID ||
		idempotencyKey != command.IdempotencyKey || revision != command.ExpectedRevision ||
		digest != command.ExpectedContentDigest || actorSubject != actor.SubjectID ||
		reason != command.Reason || (action == checklistgovernance.QuestionReviewActionTechnicalApprove && decision != "TECHNICALLY_APPROVED") {
		return false, application.ErrConflict
	}
	return true, nil
}

func (api *CanonicalAPI) getCanonicalQuestionReviewQueue(writer http.ResponseWriter, request *http.Request) {
	actor, ok := api.requireCanonicalCatalogActor(writer, request, false)
	if !ok {
		return
	}
	usage, err := parseQuestionReviewMode(generated.QuestionReviewMode(request.URL.Query().Get("mode")))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if usage != questioncatalog.UsageClassGovernedOperational {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	// Reuse the catalog queue implementation while preserving an explicit
	// review-mode capability boundary. The catalog version is a required query
	// pin so a review cannot silently move across imported versions.
	if scopeID := strings.TrimSpace(request.URL.Query().Get("scopeId")); scopeID != "" {
		if err := api.requireCanonicalScopeOwner(request.Context(), scopeID, actor.SubjectID); err != nil {
			api.respond(writer, nil, err)
			return
		}
	}
	catalogVersion := strings.TrimSpace(request.URL.Query().Get("catalogVersion"))
	search := request.URL.Query().Get("search")
	formCode := request.URL.Query().Get("formCode")
	domain := request.URL.Query().Get("domain")
	topic := request.URL.Query().Get("topic")
	riskBand := request.URL.Query().Get("riskBand")
	sourceGapState := request.URL.Query().Get("sourceGapState")
	selected := request.URL.Query().Get("selected")
	scopeID := request.URL.Query().Get("scopeId")
	var items []generated.CanonicalQuestionCatalogEntry
	var next *string
	var total int64
	items, next, total, err = api.queryCanonicalCatalog(request.Context(), actor.SubjectID, catalogVersion, usage, search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID, request.URL.Query().Get("cursor"), canonicalCatalogPageSize)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	capabilities := map[string]any{"canTechnicalApprove": false, "canPublish": false, "disabledReason": "No current governed candidate authority is bound to the returned question revisions."}
	_ = items
	api.respond(writer, generated.QuestionReviewQueue{Mode: generated.QuestionReviewMode(usage), Items: items, NextCursor: next, TotalCount: total, Capabilities: capabilities}, nil)
}

// queryCanonicalCatalog is shared by the review queue and future New Audit
// selector handlers. It intentionally returns at most 25 rows.
func (api *CanonicalAPI) queryCanonicalCatalog(ctx context.Context, _ string, catalogVersion string, usage questioncatalog.UsageClass, search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID, cursor string, limit int) ([]generated.CanonicalQuestionCatalogEntry, *string, int64, error) {
	offset, err := parseCatalogCursor(cursor)
	if err != nil {
		return nil, nil, 0, err
	}
	var total int64
	selectionPredicate := `($9='' OR $9='all' OR (($9='selected') = EXISTS (SELECT 1 FROM canonical_audit_scope_selection_questions sq JOIN canonical_audit_scope_selection_operations so ON so.id=sq.operation_id WHERE so.id=(SELECT latest.id FROM canonical_audit_scope_selection_operations latest WHERE latest.scope_draft_id=$10 AND latest.operation_kind <> 'PREVIEW' ORDER BY latest.created_at DESC, latest.id DESC LIMIT 1) AND sq.question_version_id=m.question_version_id)))`
	where := `c.catalog_version=$1 AND c.usage_class=$2 AND c.status='SEALED' AND c.source_origin='IMPORTED_APPROVED_SOURCE' AND ($3='' OR m.form_code ILIKE '%'||$3||'%' OR m.proposal_id ILIKE '%'||$3||'%' OR q.prompt ILIKE '%'||$3||'%') AND ($4='' OR m.form_code=$4) AND ($5='' OR COALESCE(m.proposed_domain,'')=$5) AND ($6='' OR COALESCE(m.proposed_topic,'')=$6) AND ($7='' OR COALESCE(m.proposed_risk_band,'')=$7) AND ($8='' OR m.source_gap_state=$8) AND COALESCE((SELECT status FROM canonical_question_catalog_membership_events event WHERE event.catalog_id=m.catalog_id AND event.question_version_id=m.question_version_id ORDER BY occurred_at DESC,event_id DESC LIMIT 1),'AVAILABLE')='AVAILABLE' AND ($10='' OR EXISTS (SELECT 1 FROM canonical_question_catalog_applicabilities applicability JOIN canonical_audit_scope_drafts scope ON scope.id=$10 WHERE applicability.catalog_id=m.catalog_id AND applicability.question_version_id=m.question_version_id AND applicability.provider_scope_id=scope.provider_scope_id AND applicability.regulated_target_id=scope.regulated_target_id AND applicability.status='ELIGIBLE')) AND ` + selectionPredicate
	args := []any{catalogVersion, string(usage), search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID}
	if err := api.pool.QueryRow(ctx, `SELECT COUNT(*) FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id JOIN question_versions q ON q.id=m.question_version_id WHERE `+where, args...).Scan(&total); err != nil {
		return nil, nil, 0, application.ErrNotFound
	}
	rows, err := api.pool.Query(ctx, `
		SELECT c.catalog_version,c.usage_class,m.question_version_id,m.form_code,m.proposal_id,m.ordinal,m.question_digest,
		       COALESCE(m.source_locator,''),m.source_gap_state,COALESCE(m.proposed_domain,''),COALESCE(m.proposed_topic,''),COALESCE(m.proposed_risk_band,''),
		       NULL::text,NULL::bigint,NULL::text,NULL::text,
		       0::bigint,NULL::text,NULL::text,NULL::text,NULL::text
		FROM canonical_question_catalogs c
		JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id
		JOIN question_versions q ON q.id=m.question_version_id
		WHERE `+where+` ORDER BY m.form_code,m.ordinal,m.question_version_id LIMIT $11 OFFSET $12`, catalogVersion, string(usage), search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID, limit+1, offset)
	if err != nil {
		return nil, nil, 0, err
	}
	defer rows.Close()
	items := make([]generated.CanonicalQuestionCatalogEntry, 0, limit)
	for rows.Next() {
		var row canonicalCatalogRow
		if err := rows.Scan(&row.CatalogVersion, &row.UsageClass, &row.QuestionID, &row.FormCode, &row.ProposalID, &row.Ordinal, &row.Digest, &row.SourceLocator, &row.SourceGap, &row.Domain, &row.Topic, &row.RiskBand, &row.GovernedCandidateID, &row.GovernedCandidateRevision, &row.GovernedCandidateContentDigest, &row.GovernedCandidateStatus, &row.ReviewRevision, &row.ReviewDisposition, &row.ReviewReason, &row.ReviewDomain, &row.ReviewTopic); err != nil {
			return nil, nil, 0, err
		}
		row.ScopeID = scopeID
		row.ReviewHistory, err = api.loadQuestionReviewHistory(ctx, row)
		if err != nil {
			return nil, nil, 0, err
		}
		items = append(items, canonicalCatalogEntry(row))
	}
	if err := rows.Err(); err != nil {
		return nil, nil, 0, err
	}
	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = encodeCatalogCursor(offset + limit)
	}
	return items, next, total, nil
}

func stringValueOrDefault(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return *value
}

// createGovernedReviewSuccessor materializes a review decision as the next
// immutable candidate leaf.  The parent revision/digest was already checked
// by the handler and locked by the lifecycle authority; the new leaf therefore
// becomes the only candidate that technical approval/publication may consume.
func createGovernedReviewSuccessor(
	ctx context.Context,
	tx pgx.Tx,
	parent regulatory.CandidateView,
	questionVersionID string,
	action checklistgovernance.QuestionReviewAction,
	domain, topic *string,
	reason string,
	actor identity.Principal,
	operationID string,
	idempotencyKey string,
) (string, int64, string, error) {
	rows, err := tx.Query(ctx, `
		SELECT ordered.question_version_id, version.question_id
		FROM template_draft_versions candidate
		CROSS JOIN unnest(candidate.question_version_ids) WITH ORDINALITY AS ordered(question_version_id,ordinality)
		JOIN question_versions version ON version.id=ordered.question_version_id
		WHERE candidate.id=$1
		ORDER BY ordered.ordinality`, parent.CandidateID)
	if err != nil {
		return "", 0, "", err
	}
	type questionIdentity struct {
		versionID  string
		questionID string
	}
	identities := make([]questionIdentity, 0, len(parent.Questions))
	for rows.Next() {
		var item questionIdentity
		if err := rows.Scan(&item.versionID, &item.questionID); err != nil {
			rows.Close()
			return "", 0, "", err
		}
		identities = append(identities, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", 0, "", err
	}
	rows.Close()
	if len(identities) != len(parent.Questions) {
		return "", 0, "", fmt.Errorf("%w: candidate question identity/snapshot count differs", application.ErrConflict)
	}
	// Candidate snapshots are read in their own stable identity order by the
	// regulatory projection. Never pair them by array position: the candidate's
	// question_version_ids array is the authority for version/question identity,
	// while the JSON snapshot query is allowed to order by question id.
	questionByID := make(map[string]regulatory.ChecklistQuestion, len(parent.Questions))
	for _, question := range parent.Questions {
		if strings.TrimSpace(question.QuestionID) == "" {
			return "", 0, "", fmt.Errorf("%w: candidate question snapshot has no question identity", application.ErrConflict)
		}
		if _, exists := questionByID[question.QuestionID]; exists {
			return "", 0, "", fmt.Errorf("%w: candidate question snapshot identity is duplicated", application.ErrConflict)
		}
		questionByID[question.QuestionID] = question
	}
	questions := make([]regulatory.ChecklistQuestion, 0, len(identities))
	questionIndex := -1
	for index, item := range identities {
		question, ok := questionByID[item.questionID]
		if !ok {
			return "", 0, "", fmt.Errorf("%w: question version %s does not match immutable question snapshot %s", application.ErrConflict, item.versionID, item.questionID)
		}
		questions = append(questions, question)
		if item.versionID == questionVersionID {
			questionIndex = index
		}
	}
	if questionIndex < 0 {
		return "", 0, "", application.ErrConflict
	}
	if action == checklistgovernance.QuestionReviewActionDomainReclassified {
		questions[questionIndex].ReviewedDomain = domain
	} else if action == checklistgovernance.QuestionReviewActionTopicReclassified {
		questions[questionIndex].ReviewedTopic = topic
	}
	reviewedAt := time.Now().UTC()
	questions[questionIndex].ReviewedReason = reason
	questions[questionIndex].ReviewedBySubjectID = actor.SubjectID
	questions[questionIndex].ReviewedAt = reviewedAt.Format(time.RFC3339Nano)
	questions[questionIndex].ReviewedRevision = parent.Revision + 1
	switch action {
	case checklistgovernance.QuestionReviewActionRetain,
		checklistgovernance.QuestionReviewActionInclude,
		checklistgovernance.QuestionReviewActionExclude,
		checklistgovernance.QuestionReviewActionDefer:
		questions[questionIndex].ReviewedDisposition = string(action)
	}
	questionVersionIDs := make([]string, 0, len(identities))
	questionSnapshots := make([]struct {
		questionID string
		raw        string
	}, 0, len(identities))
	for index, item := range identities {
		questionVersionIDs = append(questionVersionIDs, item.versionID)
		raw, err := json.Marshal(questions[index])
		if err != nil {
			return "", 0, "", err
		}
		questionSnapshots = append(questionSnapshots, struct {
			questionID string
			raw        string
		}{questionID: item.questionID, raw: string(raw)})
	}
	if len(questionVersionIDs) == 0 {
		return "", 0, "", fmt.Errorf("%w: a governed candidate must retain at least one question", application.ErrInvalid)
	}
	digest, err := regulatory.CanonicalSHA256(map[string]any{
		"complianceMappings":  parent.Mappings,
		"inspectionChecklist": map[string]any{"checklistId": parent.TemplateID, "questions": questions},
	})
	if err != nil {
		return "", 0, "", err
	}
	semantic, err := idempotency.SemanticHash(map[string]any{
		"operationId": operationID, "candidateId": parent.CandidateID,
		"action": string(action), "questionVersionId": questionVersionID,
		"domain": domain, "topic": topic, "reason": reason,
	})
	if err != nil {
		return "", 0, "", err
	}
	suffix := strings.TrimPrefix(semantic, "sha256:")
	if len(suffix) > 24 {
		suffix = suffix[:24]
	}
	successorID := "CAND-REVIEW-" + suffix
	revision := parent.Revision + 1
	version := parent.Version + 1
	if _, err := tx.Exec(ctx, `
		INSERT INTO template_draft_versions
			(id,template_id,version,status,owner_role,creator_subject_id,change_reason,
			 question_version_ids,revision,generation_run_id,candidate_content_digest,
			 candidate_schema_version,candidate_root_id,supersedes_candidate_id)
		VALUES ($1,$2,$3,'DEPARTMENT_REVIEW','Admin Preview',$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		successorID, parent.TemplateID, version, actor.SubjectID,
		"Question Review: "+reason, questionVersionIDs, revision, parent.GenerationRunID,
		digest, parent.SchemaVersion, parent.CandidateRootID, parent.CandidateID); err != nil {
		return "", 0, "", fmt.Errorf("persist governed review candidate successor: %w", err)
	}
	for index, mapping := range parent.Mappings {
		raw, err := json.Marshal(mapping)
		if err != nil {
			return "", 0, "", err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO regulatory_generated_mapping_snapshots (candidate_draft_version_id,mapping_id,mapping_ordinal,snapshot) VALUES ($1,$2,$3,$4::jsonb)`, successorID, mapping.MappingID, index, string(raw)); err != nil {
			return "", 0, "", err
		}
	}
	for _, snapshot := range questionSnapshots {
		if _, err := tx.Exec(ctx, `INSERT INTO regulatory_generated_question_snapshots (candidate_draft_version_id,question_id,snapshot) VALUES ($1,$2,$3::jsonb)`, successorID, snapshot.questionID, snapshot.raw); err != nil {
			return "", 0, "", err
		}
	}
	for index, owner := range parent.RequiredOwners {
		if _, err := tx.Exec(ctx, `INSERT INTO candidate_required_owner_assignments (id,candidate_draft_version_id,candidate_revision,candidate_content_digest,department_id,organizational_unit_id,approval_required) VALUES ($1,$2,$3,$4,$5,$6,$7)`, fmt.Sprintf("OWNER-%s-%02d", successorID, index+1), successorID, revision, digest, owner.DepartmentID, owner.OrganizationalUnitID, owner.ApprovalRequired); err != nil {
			return "", 0, "", err
		}
	}
	now := reviewedAt
	auditID := "AE-" + operationID
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (event_id,occurred_at,actor_subject_id,actor_role,organization_id,action,entity_type,entity_id,entity_version,before_status,after_status,reason,operation_id,correlation_id,request_id,details)
		VALUES ($1,$2,$3,'manager',$4,'QUESTION_REVIEW_SUCCESSOR','GOVERNED_CANDIDATE',$5,$6,$7,'DEPARTMENT_REVIEW',$8,$9,$9,$9,'{}'::jsonb)`, auditID, now, actor.SubjectID, actor.OrganizationID, successorID, revision, parent.Status, reason, operationID); err != nil {
		return "", 0, "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO governed_candidate_commands
			(id,command_kind,operation_id,idempotency_key,semantic_payload_digest,
			 generation_run_id,candidate_draft_version_id,candidate_revision,
			 candidate_content_digest,actor_subject_id,reason,audit_event_id,created_at)
		VALUES ($1,'QUESTION_REVIEW_SUCCESSOR',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		"CMD-"+operationID, operationID, idempotencyKey, semantic, parent.GenerationRunID,
		successorID, revision, digest, actor.SubjectID, reason, auditID, now); err != nil {
		return "", 0, "", err
	}
	return successorID, revision, digest, nil
}

func questionsForCandidateDigest(questions []regulatory.ChecklistQuestion, action checklistgovernance.QuestionReviewAction, excludedIndex int) []regulatory.ChecklistQuestion {
	if action != checklistgovernance.QuestionReviewActionExclude {
		return questions
	}
	filtered := make([]regulatory.ChecklistQuestion, 0, len(questions)-1)
	for index, question := range questions {
		if index != excludedIndex {
			filtered = append(filtered, question)
		}
	}
	return filtered
}

func (api *CanonicalAPI) commandCanonicalGovernedQuestionReview(writer http.ResponseWriter, request *http.Request) {
	actor, ok := api.requireCanonicalCatalogActor(writer, request, true)
	if !ok {
		return
	}
	var input generated.GovernedQuestionReviewCommandInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if strings.TrimSpace(input.OperationId) == "" || strings.TrimSpace(input.QuestionVersionId) == "" || strings.TrimSpace(input.CatalogVersion) == "" || strings.TrimSpace(input.CandidateId) == "" || input.ExpectedCandidateRevision < 1 || strings.TrimSpace(input.ExpectedCandidateContentDigest) == "" || strings.TrimSpace(input.Reason) == "" || !validQuestionReviewReason(input.Reason) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	if api.pool == nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	action := checklistgovernance.QuestionReviewAction(input.Action)
	idempotencyKey := input.OperationId
	if input.IdempotencyKey != nil && strings.TrimSpace(*input.IdempotencyKey) != "" {
		idempotencyKey = strings.TrimSpace(*input.IdempotencyKey)
	}
	effectiveReplayTopic := input.Topic
	if action == checklistgovernance.QuestionReviewActionTopicReclassified && effectiveReplayTopic == nil {
		cleared := ""
		effectiveReplayTopic = &cleared
	}
	// Replay is resolved before leaf/status preflight. A successful command
	// necessarily turns its parent into a non-leaf successor, so retrying the
	// original request must return the original successor instead of being
	// misreported as a stale candidate conflict.
	switch action {
	case checklistgovernance.QuestionReviewActionRetain, checklistgovernance.QuestionReviewActionInclude,
		checklistgovernance.QuestionReviewActionExclude, checklistgovernance.QuestionReviewActionDefer,
		checklistgovernance.QuestionReviewActionDomainReclassified, checklistgovernance.QuestionReviewActionTopicReclassified:
		var replayed bool
		var replaySuccessorID string
		var replaySuccessorRevision int64
		var replaySuccessorDigest string
		replayErr := database.WithinTransaction(request.Context(), api.pool, func(ctx context.Context, tx pgx.Tx) error {
			var replayCandidate, replayQuestion, replayIdempotency, replayAction, replayReason, replayActor string
			var replayRevision int64
			var replayDigest string
			var replayDomain, replayTopic *string
			replayErr := tx.QueryRow(ctx, `
			SELECT candidate_draft_version_id, question_version_id, idempotency_key,
			       action, reason, actor_subject_id, candidate_revision, candidate_content_digest,
			       reviewed_domain, reviewed_topic
			FROM canonical_governed_question_review_events
			WHERE operation_id=$1 OR idempotency_key=$2
			ORDER BY CASE WHEN operation_id=$1 THEN 0 ELSE 1 END
			LIMIT 1`, input.OperationId, idempotencyKey).Scan(
				&replayCandidate, &replayQuestion, &replayIdempotency, &replayAction,
				&replayReason, &replayActor, &replayRevision, &replayDigest,
				&replayDomain, &replayTopic)
			if replayErr == nil {
				if replayCandidate != input.CandidateId || replayQuestion != input.QuestionVersionId ||
					replayIdempotency != idempotencyKey || replayAction != input.Action ||
					replayReason != input.Reason || replayActor != actor.SubjectID ||
					replayRevision != input.ExpectedCandidateRevision ||
					replayDigest != input.ExpectedCandidateContentDigest ||
					!sameOptionalString(replayDomain, input.Domain) ||
					!sameOptionalString(replayTopic, effectiveReplayTopic) {
					return application.ErrConflict
				}
				if api.governedLifecycle == nil {
					return application.ErrNotFound
				}
				if err := api.governedLifecycle.RequireCurrentDepartmentReviewAuthority(ctx, tx, actor, replayCandidate, replayRevision, replayDigest); err != nil {
					return err
				}
				if err := tx.QueryRow(ctx, `
				SELECT id, revision, candidate_content_digest
				FROM template_draft_versions
				WHERE supersedes_candidate_id=$1
				ORDER BY revision DESC, id DESC
				LIMIT 1`, replayCandidate).Scan(&replaySuccessorID, &replaySuccessorRevision, &replaySuccessorDigest); err != nil {
					return err
				}
				replayed = true
				return nil
			} else if !errors.Is(replayErr, pgx.ErrNoRows) {
				return replayErr
			}
			return nil
		})
		if replayErr != nil {
			api.respond(writer, nil, replayErr)
			return
		}
		if replayed {
			api.respond(writer, generated.QuestionReviewCommandOutput{
				OperationId: input.OperationId, Mode: generated.QuestionReviewModeGOVERNEDOPERATIONAL,
				QuestionVersionId: input.QuestionVersionId, Action: input.Action, Replayed: true,
				CanPublish: false, CurrentCandidateId: &replaySuccessorID,
				CurrentCandidateRevision: &replaySuccessorRevision, CurrentCandidateContentDigest: &replaySuccessorDigest,
			}, nil)
			return
		}
	default:
		// Technical approval and publication are owned by checklistgovernance;
		// their own transaction-level replay boundary runs below.
	}
	var catalogID, candidateStatus string
	if strings.HasPrefix(input.CatalogVersion, "candidate:") {
		virtualCandidateID := strings.TrimPrefix(input.CatalogVersion, "candidate:")
		if virtualCandidateID == "" || virtualCandidateID != input.CandidateId {
			api.respond(writer, nil, application.ErrConflict)
			return
		}
		if err := api.pool.QueryRow(request.Context(), `
			SELECT candidate.status
			FROM template_draft_versions candidate
			WHERE candidate.id=$1 AND candidate.revision=$2 AND candidate.candidate_content_digest=$3
			  AND candidate.question_version_ids @> ARRAY[$4]::text[]
			  AND candidate.status IN ('DEPARTMENT_REVIEW','RETURNED','TECHNICALLY_APPROVED')
			  AND NOT EXISTS (SELECT 1 FROM template_draft_versions successor WHERE successor.supersedes_candidate_id=candidate.id)
		`, input.CandidateId, input.ExpectedCandidateRevision, input.ExpectedCandidateContentDigest, input.QuestionVersionId).Scan(&candidateStatus); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				api.respond(writer, nil, application.ErrConflict)
			} else {
				api.respond(writer, nil, err)
			}
			return
		}
	} else if err := api.pool.QueryRow(request.Context(), `
		SELECT catalog.id, candidate.status
		FROM canonical_question_catalogs catalog
		JOIN canonical_question_catalog_memberships membership ON membership.catalog_id=catalog.id AND membership.question_version_id=$3
		JOIN template_draft_versions candidate ON candidate.id=$4
		WHERE catalog.catalog_version=$1 AND catalog.usage_class='GOVERNED_OPERATIONAL' AND catalog.status='SEALED'
		  AND catalog.governed_candidate_draft_version_id=candidate.id
		  AND catalog.governed_candidate_revision=candidate.revision
		  AND catalog.governed_candidate_content_digest=candidate.candidate_content_digest
		  AND candidate.revision=$5 AND candidate.candidate_content_digest=$6
		  AND candidate.question_version_ids @> ARRAY[$3]::text[]
		  AND candidate.status IN ('DEPARTMENT_REVIEW','RETURNED','TECHNICALLY_APPROVED')
		  AND NOT EXISTS (SELECT 1 FROM template_draft_versions successor WHERE successor.supersedes_candidate_id=candidate.id)
	`, input.CatalogVersion, actor.SubjectID, input.QuestionVersionId, input.CandidateId, input.ExpectedCandidateRevision, input.ExpectedCandidateContentDigest).Scan(&catalogID, &candidateStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.respond(writer, nil, application.ErrConflict)
		} else {
			api.respond(writer, nil, err)
		}
		return
	}
	command := checklistgovernance.ReviewCommand{OperationID: input.OperationId, IdempotencyKey: idempotencyKey, CandidateID: input.CandidateId, ExpectedRevision: input.ExpectedCandidateRevision, ExpectedContentDigest: input.ExpectedCandidateContentDigest, Reason: input.Reason}
	switch action {
	case checklistgovernance.QuestionReviewActionTechnicalApprove:
		if api.governedLifecycle == nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
		replayed, replayErr := api.governedLifecycleReplay(request.Context(), actor, command, action)
		if replayErr != nil {
			api.respond(writer, nil, replayErr)
			return
		}
		candidate, err := api.governedLifecycle.Approve(request.Context(), actor, command)
		candidateID, candidateRevision, candidateDigest := candidate.CandidateID, candidate.Revision, candidate.ContentDigest
		api.respond(writer, generated.QuestionReviewCommandOutput{OperationId: input.OperationId, Mode: generated.QuestionReviewModeGOVERNEDOPERATIONAL, QuestionVersionId: input.QuestionVersionId, Action: input.Action, Replayed: replayed, CanPublish: candidate.Status == "TECHNICALLY_APPROVED", CurrentCandidateId: &candidateID, CurrentCandidateRevision: &candidateRevision, CurrentCandidateContentDigest: &candidateDigest}, err)
		return
	case checklistgovernance.QuestionReviewActionPublish:
		if api.governedLifecycle == nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
		replayed, replayErr := api.governedLifecycleReplay(request.Context(), actor, command, action)
		if replayErr != nil {
			api.respond(writer, nil, replayErr)
			return
		}
		publication, err := api.governedLifecycle.Publish(request.Context(), actor, checklistgovernance.PublicationCommand(command))
		candidateID, candidateRevision, candidateDigest := command.CandidateID, command.ExpectedRevision, command.ExpectedContentDigest
		api.respond(writer, generated.QuestionReviewCommandOutput{OperationId: input.OperationId, Mode: generated.QuestionReviewModeGOVERNEDOPERATIONAL, QuestionVersionId: input.QuestionVersionId, Action: input.Action, Replayed: replayed, CanPublish: true, CurrentCandidateId: &candidateID, CurrentCandidateRevision: &candidateRevision, CurrentCandidateContentDigest: &candidateDigest}, err)
		_ = publication
		return
	case checklistgovernance.QuestionReviewActionRetain, checklistgovernance.QuestionReviewActionInclude, checklistgovernance.QuestionReviewActionExclude, checklistgovernance.QuestionReviewActionDefer, checklistgovernance.QuestionReviewActionDomainReclassified, checklistgovernance.QuestionReviewActionTopicReclassified:
		if action != checklistgovernance.QuestionReviewActionDomainReclassified && action != checklistgovernance.QuestionReviewActionTopicReclassified && (input.Domain != nil || input.Topic != nil) {
			api.respond(writer, nil, fmt.Errorf("%w: disposition commands cannot carry classification fields", application.ErrInvalid))
			return
		}
		if action == checklistgovernance.QuestionReviewActionDomainReclassified && (input.Domain == nil || strings.TrimSpace(*input.Domain) == "") {
			api.respond(writer, nil, fmt.Errorf("%w: domain reclassification requires a nonblank domain", application.ErrInvalid))
			return
		}
		if candidateStatus == "TECHNICALLY_APPROVED" {
			api.respond(writer, nil, fmt.Errorf("%w: governed candidate is already technically approved", application.ErrConflict))
			return
		}
		payload, err := json.Marshal(map[string]any{"catalogVersion": input.CatalogVersion, "catalogId": catalogID, "questionVersionId": input.QuestionVersionId, "candidateId": input.CandidateId, "candidateRevision": input.ExpectedCandidateRevision, "candidateContentDigest": input.ExpectedCandidateContentDigest, "action": input.Action, "reason": input.Reason, "domain": input.Domain, "topic": input.Topic, "actorSubjectId": actor.SubjectID})
		if err != nil {
			api.respond(writer, nil, err)
			return
		}
		_ = payload
		replayed := false
		var successorRevision int64
		var successorDigest string
		var successorID string
		reviewedTopic := input.Topic
		if action == checklistgovernance.QuestionReviewActionTopicReclassified && reviewedTopic == nil {
			cleared := ""
			reviewedTopic = &cleared
		}
		err = database.WithinTransaction(request.Context(), api.pool, func(ctx context.Context, tx pgx.Tx) error {
			var existingQuestion, existingCandidate, existingAction, existingReason, existingActor, existingIdempotency string
			var existingCatalogValue *string
			var existingRevision int64
			var existingDigest string
			var existingDomain, existingTopic *string
			if err := tx.QueryRow(ctx, `SELECT catalog_id, question_version_id, candidate_draft_version_id, idempotency_key, action, reason, reviewed_domain, reviewed_topic, actor_subject_id, candidate_revision, candidate_content_digest FROM canonical_governed_question_review_events WHERE operation_id=$1`, input.OperationId).Scan(&existingCatalogValue, &existingQuestion, &existingCandidate, &existingIdempotency, &existingAction, &existingReason, &existingDomain, &existingTopic, &existingActor, &existingRevision, &existingDigest); err == nil {
				expectedCatalog := (*string)(nil)
				if catalogID != "" {
					expectedCatalog = &catalogID
				}
				if !sameOptionalString(existingCatalogValue, expectedCatalog) || existingQuestion != input.QuestionVersionId || existingCandidate != input.CandidateId || existingIdempotency != idempotencyKey || existingAction != input.Action || existingReason != input.Reason || existingActor != actor.SubjectID || existingRevision != input.ExpectedCandidateRevision || existingDigest != input.ExpectedCandidateContentDigest || !sameOptionalString(existingDomain, input.Domain) || !sameOptionalString(existingTopic, reviewedTopic) {
					return fmt.Errorf("%w: operation replay payload differs", application.ErrConflict)
				}
				if api.governedLifecycle == nil {
					return application.ErrNotFound
				}
				if err := api.governedLifecycle.RequireCurrentDepartmentReviewAuthority(ctx, tx, actor, input.CandidateId, input.ExpectedCandidateRevision, input.ExpectedCandidateContentDigest); err != nil {
					return err
				}
				replayed = true
				if err := tx.QueryRow(ctx, `SELECT id, revision, candidate_content_digest FROM template_draft_versions WHERE supersedes_candidate_id=$1 ORDER BY revision DESC,id DESC LIMIT 1`, existingCandidate).Scan(&successorID, &successorRevision, &successorDigest); err != nil {
					return fmt.Errorf("load governed review successor replay: %w", err)
				}
				return nil
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			if api.governedLifecycle == nil {
				return application.ErrNotFound
			}
			var currentRootID, currentStatus, currentDigest string
			var currentRevision int64
			if err := tx.QueryRow(ctx, `
				SELECT candidate_root_id,status,revision,candidate_content_digest
				FROM template_draft_versions
				WHERE id=$1
				FOR UPDATE`, input.CandidateId).Scan(&currentRootID, &currentStatus, &currentRevision, &currentDigest); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return application.ErrConflict
				}
				return err
			}
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "governed-candidate-root:"+currentRootID); err != nil {
				return err
			}
			// A concurrent identical request may have missed the event lookup
			// before waiting on the candidate row/root lock. Recheck after the
			// lock so the loser returns the original successor instead of a
			// misleading CAS conflict.
			var postReplayCandidate, postReplayQuestion, postReplayIdempotency, postReplayAction, postReplayReason, postReplayActor string
			var postReplayRevision int64
			var postReplayDigest string
			var postReplayDomain, postReplayTopic *string
			postReplayErr := tx.QueryRow(ctx, `
				SELECT candidate_draft_version_id, question_version_id, idempotency_key,
				       action, reason, actor_subject_id, candidate_revision,
				       candidate_content_digest, reviewed_domain, reviewed_topic
				FROM canonical_governed_question_review_events
				WHERE operation_id=$1 OR idempotency_key=$2
				ORDER BY CASE WHEN operation_id=$1 THEN 0 ELSE 1 END
				LIMIT 1`, input.OperationId, idempotencyKey).Scan(
				&postReplayCandidate, &postReplayQuestion, &postReplayIdempotency,
				&postReplayAction, &postReplayReason, &postReplayActor,
				&postReplayRevision, &postReplayDigest, &postReplayDomain, &postReplayTopic)
			if postReplayErr == nil {
				if postReplayCandidate != input.CandidateId || postReplayQuestion != input.QuestionVersionId ||
					postReplayIdempotency != idempotencyKey || postReplayAction != input.Action ||
					postReplayReason != input.Reason || postReplayActor != actor.SubjectID ||
					postReplayRevision != input.ExpectedCandidateRevision || postReplayDigest != input.ExpectedCandidateContentDigest ||
					!sameOptionalString(postReplayDomain, input.Domain) || !sameOptionalString(postReplayTopic, reviewedTopic) {
					return fmt.Errorf("%w: operation replay payload differs", application.ErrConflict)
				}
				replayed = true
				if err := tx.QueryRow(ctx, `SELECT id, revision, candidate_content_digest FROM template_draft_versions WHERE supersedes_candidate_id=$1 ORDER BY revision DESC,id DESC LIMIT 1`, postReplayCandidate).Scan(&successorID, &successorRevision, &successorDigest); err != nil {
					return fmt.Errorf("load governed review successor replay: %w", err)
				}
				return nil
			} else if !errors.Is(postReplayErr, pgx.ErrNoRows) {
				return postReplayErr
			}
			if currentRevision != input.ExpectedCandidateRevision || currentDigest != input.ExpectedCandidateContentDigest || (currentStatus != "DEPARTMENT_REVIEW" && currentStatus != "RETURNED") {
				return application.ErrConflict
			}
			var hasSuccessor bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM template_draft_versions WHERE supersedes_candidate_id=$1)`, input.CandidateId).Scan(&hasSuccessor); err != nil {
				return err
			}
			if hasSuccessor {
				return application.ErrConflict
			}
			if err := api.governedLifecycle.RequireCurrentDepartmentReviewAuthority(ctx, tx, actor, input.CandidateId, input.ExpectedCandidateRevision, input.ExpectedCandidateContentDigest); err != nil {
				return err
			}
			parent, err := regulatory.LoadCandidateForGovernanceQuery(ctx, tx, input.CandidateId)
			if err != nil {
				return err
			}
			var revision int64
			var digest string
			successorID, revision, digest, err = createGovernedReviewSuccessor(ctx, tx, parent, input.QuestionVersionId, action, input.Domain, reviewedTopic, input.Reason, actor, input.OperationId, idempotencyKey)
			if err != nil {
				return err
			}
			successorRevision, successorDigest = revision, digest
			var eventCatalog any = catalogID
			if strings.HasPrefix(input.CatalogVersion, "candidate:") {
				eventCatalog = nil
			}
			_, err = tx.Exec(ctx, `INSERT INTO canonical_governed_question_review_events (event_id,operation_id,idempotency_key,catalog_id,question_version_id,candidate_draft_version_id,candidate_revision,candidate_content_digest,action,reason,reviewed_domain,reviewed_topic,actor_subject_id) VALUES ($1,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, input.OperationId, idempotencyKey, eventCatalog, input.QuestionVersionId, input.CandidateId, input.ExpectedCandidateRevision, input.ExpectedCandidateContentDigest, input.Action, input.Reason, input.Domain, reviewedTopic, actor.SubjectID)
			_ = successorID
			return err
		})
		var outputRevision *int64
		if successorRevision > 0 {
			outputRevision = &successorRevision
		}
		var outputDigest *string
		if successorDigest != "" {
			outputDigest = &successorDigest
		}
		currentID := successorID
		var currentRevision *int64
		if successorRevision > 0 {
			currentRevision = &successorRevision
		}
		var currentDigest *string
		if successorDigest != "" {
			currentDigest = &successorDigest
		}
		api.respond(writer, generated.QuestionReviewCommandOutput{OperationId: input.OperationId, Mode: generated.QuestionReviewModeGOVERNEDOPERATIONAL, QuestionVersionId: input.QuestionVersionId, Action: input.Action, Replayed: replayed, CanPublish: false, ReviewRevision: outputRevision, ReviewDigest: outputDigest, CurrentCandidateId: &currentID, CurrentCandidateRevision: currentRevision, CurrentCandidateContentDigest: currentDigest}, err)
	default:
		api.respond(writer, nil, fmt.Errorf("%w: unsupported governed Question Review action", application.ErrInvalid))
	}
}

func sameOptionalString(left, right *string) bool {
	if left == nil || strings.TrimSpace(*left) == "" {
		return right == nil || strings.TrimSpace(*right) == ""
	}
	return right != nil && *left == *right
}
