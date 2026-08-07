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

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistgovernance"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi/generated"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/questioncatalog"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

const canonicalCatalogPageSize = 25

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
	} else if !actor.IsCAA() {
		api.respond(writer, nil, application.ErrNotFound)
		return identity.Principal{}, false
	}
	return actor, true
}

func (api *CanonicalAPI) allowExerciseCatalog() error {
	if !api.preprodExerciseProfile {
		return application.ErrNotFound
	}
	return nil
}

func parseQuestionUsageClass(value string) (questioncatalog.UsageClass, error) {
	switch questioncatalog.UsageClass(strings.TrimSpace(value)) {
	case questioncatalog.UsageClassGovernedOperational:
		return questioncatalog.UsageClassGovernedOperational, nil
	case questioncatalog.UsageClassPreprodExercise:
		return questioncatalog.UsageClassPreprodExercise, nil
	default:
		return "", fmt.Errorf("%w: invalid usage class", application.ErrInvalid)
	}
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

type canonicalCatalogRow struct {
	CatalogVersion      string
	UsageClass          string
	QuestionID          string
	FormCode            string
	ProposalID          string
	Ordinal             int64
	Digest              string
	SourceLocator       string
	SourceGap           string
	Domain              string
	Topic               string
	RiskBand            string
	Prompt              string
	ConfiguredReference string
	ExpectedEvidence    string
}

func canonicalCatalogEntry(row canonicalCatalogRow) generated.CanonicalQuestionCatalogEntry {
	entry := generated.CanonicalQuestionCatalogEntry{
		CatalogVersion:    row.CatalogVersion,
		UsageClass:        generated.QuestionUsageClass(row.UsageClass),
		QuestionVersionId: row.QuestionID,
		FormCode:          row.FormCode, ProposalId: row.ProposalID, Ordinal: row.Ordinal,
		QuestionDigest: row.Digest, SourceGapState: row.SourceGap,
		CanSelect:  true,
		CanPublish: row.UsageClass == string(questioncatalog.UsageClassGovernedOperational),
	}
	if row.SourceLocator != "" {
		entry.SourceLocator = &row.SourceLocator
	}
	if row.Domain != "" {
		entry.ProposedDomain = &row.Domain
	}
	if row.Topic != "" {
		entry.ProposedTopic = &row.Topic
	}
	if row.RiskBand != "" {
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
	return entry
}

func (api *CanonicalAPI) listCanonicalQuestionCatalogEntries(writer http.ResponseWriter, request *http.Request) {
	_, ok := api.requireCanonicalCatalogActor(writer, request, false)
	if !ok {
		return
	}
	usage, err := parseQuestionUsageClass(request.URL.Query().Get("usageClass"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if usage == questioncatalog.UsageClassPreprodExercise {
		if err := api.allowExerciseCatalog(); err != nil {
			api.respond(writer, nil, err)
			return
		}
	}
	offset, err := parseCatalogCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	limit := canonicalCatalogPageSize
	if raw := request.URL.Query().Get("limit"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > canonicalCatalogPageSize {
			api.respond(writer, nil, fmt.Errorf("%w: catalog limit must be between 1 and 25", application.ErrInvalid))
			return
		}
		limit = parsed
	}
	if api.pool == nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	catalogVersion := chi.URLParam(request, "catalogVersion")
	search := strings.TrimSpace(request.URL.Query().Get("search"))
	formCode := strings.TrimSpace(request.URL.Query().Get("formCode"))
	domain := strings.TrimSpace(request.URL.Query().Get("domain"))
	topic := strings.TrimSpace(request.URL.Query().Get("topic"))
	riskBand := strings.TrimSpace(request.URL.Query().Get("riskBand"))
	sourceGapState := strings.TrimSpace(request.URL.Query().Get("sourceGapState"))
	selected := strings.TrimSpace(request.URL.Query().Get("selected"))
	scopeID := strings.TrimSpace(request.URL.Query().Get("scopeId"))
	if selected != "" && selected != "all" && selected != "selected" && selected != "unselected" {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	selectionPredicate := `($9='' OR $9='all' OR (($9='selected') = EXISTS (SELECT 1 FROM canonical_audit_scope_selection_questions sq JOIN canonical_audit_scope_selection_operations so ON so.id=sq.operation_id WHERE so.id=(SELECT latest.id FROM canonical_audit_scope_selection_operations latest WHERE latest.scope_draft_id=$10 ORDER BY latest.created_at DESC, latest.id DESC LIMIT 1) AND sq.question_version_id=m.question_version_id)))`
	where := `c.catalog_version=$1 AND c.usage_class=$2 AND c.status='SEALED' AND ($3='' OR m.form_code ILIKE '%' || $3 || '%' OR m.proposal_id ILIKE '%' || $3 || '%') AND ($4='' OR m.form_code=$4) AND ($5='' OR COALESCE(m.proposed_domain,'')=$5) AND ($6='' OR COALESCE(m.proposed_topic,'')=$6) AND ($7='' OR COALESCE(m.proposed_risk_band,'')=$7) AND ($8='' OR m.source_gap_state=$8) AND ` + selectionPredicate
	ctx := request.Context()
	var total int64
	if err := api.pool.QueryRow(ctx, `SELECT COUNT(*) FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id = c.id WHERE `+where, catalogVersion, string(usage), search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID).Scan(&total); err != nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	rows, err := api.pool.Query(ctx, `SELECT c.catalog_version,c.usage_class,m.question_version_id,m.form_code,m.proposal_id,m.ordinal,m.question_digest,COALESCE(m.source_locator,''),m.source_gap_state,COALESCE(m.proposed_domain,''),COALESCE(m.proposed_topic,''),COALESCE(m.proposed_risk_band,'') FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id = c.id WHERE `+where+` ORDER BY m.form_code,m.ordinal,m.question_version_id LIMIT $11 OFFSET $12`, catalogVersion, string(usage), search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID, limit+1, offset)
	if err != nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	defer rows.Close()
	items := make([]generated.CanonicalQuestionCatalogEntry, 0, limit)
	for rows.Next() {
		var row canonicalCatalogRow
		if err := rows.Scan(&row.CatalogVersion, &row.UsageClass, &row.QuestionID, &row.FormCode, &row.ProposalID, &row.Ordinal, &row.Digest, &row.SourceLocator, &row.SourceGap, &row.Domain, &row.Topic, &row.RiskBand); err != nil {
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
	api.respond(writer, generated.CanonicalQuestionCatalogPage{Items: items, NextCursor: next, CatalogVersion: catalogVersion, UsageClass: generated.QuestionUsageClass(usage), TotalCount: total}, nil)
}

func (api *CanonicalAPI) getCanonicalQuestionCatalogEntry(writer http.ResponseWriter, request *http.Request) {
	_, ok := api.requireCanonicalCatalogActor(writer, request, false)
	if !ok {
		return
	}
	usage, err := parseQuestionUsageClass(request.URL.Query().Get("usageClass"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if usage == questioncatalog.UsageClassPreprodExercise {
		if err := api.allowExerciseCatalog(); err != nil {
			api.respond(writer, nil, err)
			return
		}
	}
	if api.pool == nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	var row canonicalCatalogRow
	err = api.pool.QueryRow(request.Context(), `
		SELECT c.catalog_version,c.usage_class,m.question_version_id,m.form_code,m.proposal_id,m.ordinal,m.question_digest,
		       COALESCE(m.source_locator,''),m.source_gap_state,COALESCE(m.proposed_domain,''),COALESCE(m.proposed_topic,''),COALESCE(m.proposed_risk_band,''),
		       q.prompt,q.configured_reference,q.expected_evidence
		FROM canonical_question_catalogs c
		JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id
		JOIN question_versions q ON q.id=m.question_version_id
		WHERE c.catalog_version=$1 AND c.usage_class=$2 AND c.status='SEALED' AND m.question_version_id=$3
	`, chi.URLParam(request, "catalogVersion"), string(usage), chi.URLParam(request, "questionVersionId")).Scan(&row.CatalogVersion, &row.UsageClass, &row.QuestionID, &row.FormCode, &row.ProposalID, &row.Ordinal, &row.Digest, &row.SourceLocator, &row.SourceGap, &row.Domain, &row.Topic, &row.RiskBand, &row.Prompt, &row.ConfiguredReference, &row.ExpectedEvidence)
	if errors.Is(err, pgx.ErrNoRows) {
		err = application.ErrNotFound
	}
	api.respond(writer, canonicalCatalogEntry(row), err)
}

type canonicalScopeState struct {
	CatalogID      string
	CatalogVersion string
	UsageClass     string
	CurrentDigest  string
	Selected       []string
}

func (api *CanonicalAPI) readCanonicalScopeState(ctx context.Context, scopeID string) (canonicalScopeState, error) {
	if api.pool == nil {
		return canonicalScopeState{}, application.ErrNotFound
	}
	var state canonicalScopeState
	if err := api.pool.QueryRow(ctx, `
		SELECT c.id,c.catalog_version,c.usage_class,s.selection_digest
		FROM canonical_audit_scope_drafts s
		JOIN canonical_question_catalogs c ON c.id=s.catalog_id
		WHERE s.id=$1 AND s.status IN ('DRAFT','SUBMITTED','RELEASED')
	`, scopeID).Scan(&state.CatalogID, &state.CatalogVersion, &state.UsageClass, &state.CurrentDigest); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return canonicalScopeState{}, application.ErrNotFound
		}
		return canonicalScopeState{}, err
	}
	if state.CurrentDigest == "" {
		state.CurrentDigest = questioncatalog.SelectionDigest(nil)
	}
	var payload []byte
	var operationID string
	err := api.pool.QueryRow(ctx, `
		SELECT id
		FROM canonical_audit_scope_selection_operations
		WHERE scope_draft_id=$1
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
	if len(ids) == 0 || len(ids) > 500 {
		return fmt.Errorf("%w: selection must contain between 1 and 500 questions", application.ErrInvalid)
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
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM canonical_question_catalog_memberships WHERE catalog_id=$1 AND usage_class=$2 AND question_version_id = ANY($3::text[])`, state.CatalogID, string(usage), ids).Scan(&count); err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return fmt.Errorf("%w: selection contains a question outside the catalog", application.ErrInvalid)
	}
	return nil
}

func canonicalSelectionDigest(ids []string) string { return questioncatalog.SelectionDigest(ids) }

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
	if usage == questioncatalog.UsageClassPreprodExercise {
		if err := api.allowExerciseCatalog(); err != nil {
			api.respond(writer, nil, err)
			return
		}
	}
	state, err := api.readCanonicalScopeState(request.Context(), chi.URLParam(request, "scopeId"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if err := validateCanonicalSelection(request.Context(), api.pool, state, input.QuestionVersionIds, usage); err != nil {
		api.respond(writer, nil, err)
		return
	}
	digest := canonicalSelectionDigest(input.QuestionVersionIds)
	expectedDigest := input.ExpectedSelectionDigest
	if expectedDigest == "" {
		expectedDigest = questioncatalog.SelectionDigest(nil)
	}
	valid := expectedDigest == state.CurrentDigest
	reason := "selection is ready to commit"
	if !valid {
		reason = "selection digest is stale; reload the current scope"
	}
	api.respond(writer, generated.CanonicalAuditScopeSelectionPreview{
		Preview:       generated.AuditScopeSelectionDigest{SelectionDigest: digest, SelectedQuestionVersionIds: input.QuestionVersionIds, SelectedCount: int64(len(input.QuestionVersionIds)), CatalogVersion: state.CatalogVersion, UsageClass: generated.QuestionUsageClass(usage)},
		AffectedCount: int64(len(input.QuestionVersionIds)), Valid: valid, Reason: reason,
	}, nil)
	_ = actor
}

func (api *CanonicalAPI) commitCanonicalAuditScopeSelection(writer http.ResponseWriter, request *http.Request) {
	_, ok := api.requireCanonicalCatalogActor(writer, request, true)
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
	if usage == questioncatalog.UsageClassPreprodExercise {
		if err := api.allowExerciseCatalog(); err != nil {
			api.respond(writer, nil, err)
			return
		}
	}
	state, err := api.readCanonicalScopeState(request.Context(), chi.URLParam(request, "scopeId"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if strings.TrimSpace(input.OperationId) == "" {
		api.respond(writer, nil, fmt.Errorf("%w: operationId is required", application.ErrInvalid))
		return
	}
	expectedDigest := input.ExpectedSelectionDigest
	if expectedDigest == "" {
		expectedDigest = questioncatalog.SelectionDigest(nil)
	}
	if err := validateCanonicalSelection(request.Context(), api.pool, state, input.QuestionVersionIds, usage); err != nil {
		api.respond(writer, nil, err)
		return
	}
	digest := canonicalSelectionDigest(input.QuestionVersionIds)
	// A retry with the same operation is an idempotent receipt even when a
	// later selection has advanced the draft digest. Check the immutable
	// operation row before enforcing the current-digest CAS for new work.
	var priorDigest string
	var priorPayload []byte
	priorErr := api.pool.QueryRow(request.Context(), `SELECT result_digest, affected_question_version_ids FROM canonical_audit_scope_selection_operations WHERE scope_draft_id=$1 AND operation_id=$2`, chi.URLParam(request, "scopeId"), input.OperationId).Scan(&priorDigest, &priorPayload)
	if priorErr == nil {
		var priorIDs []string
		if json.Unmarshal(priorPayload, &priorIDs) != nil || priorDigest != digest || len(priorIDs) != len(input.QuestionVersionIds) {
			api.respond(writer, nil, fmt.Errorf("%w: operation replay payload differs", application.ErrConflict))
			return
		}
		for i := range priorIDs {
			if priorIDs[i] != input.QuestionVersionIds[i] {
				api.respond(writer, nil, fmt.Errorf("%w: operation replay payload differs", application.ErrConflict))
				return
			}
		}
		api.respond(writer, generated.CanonicalAuditScopeSelectionReceipt{OperationId: input.OperationId, Replayed: true, Selection: generated.AuditScopeSelectionDigest{SelectionDigest: digest, SelectedQuestionVersionIds: input.QuestionVersionIds, SelectedCount: int64(len(input.QuestionVersionIds)), CatalogVersion: state.CatalogVersion, UsageClass: generated.QuestionUsageClass(usage)}}, nil)
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
	payload, _ := json.Marshal(input.QuestionVersionIds)
	filterPayload, _ := json.Marshal(input.Filter)
	if input.Filter == nil || len(filterPayload) == 0 || string(filterPayload) == "null" {
		filterPayload = []byte(`{}`)
	}
	var replayed bool
	err = database.WithinTransaction(request.Context(), api.pool, func(ctx context.Context, tx pgx.Tx) error {
		var currentDigest string
		if err := tx.QueryRow(ctx, `SELECT selection_digest FROM canonical_audit_scope_drafts WHERE id=$1 AND status='DRAFT' FOR UPDATE`, chi.URLParam(request, "scopeId")).Scan(&currentDigest); err != nil {
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
		_, err := tx.Exec(ctx, `
			INSERT INTO canonical_audit_scope_selection_operations
			(id,scope_draft_id,operation_id,idempotency_key,operation_kind,expected_digest,result_digest,affected_question_version_ids,filter_payload,actor_subject_id)
			VALUES ($1,$2,$3,$4,'REPLACE',$5,$6,$7,$8,$9)
		`, "scope-selection:"+chi.URLParam(request, "scopeId")+":"+input.OperationId, chi.URLParam(request, "scopeId"), input.OperationId, operationKey, input.ExpectedSelectionDigest, digest, payload, filterPayload, requestPrincipalSubject(request))
		if err != nil {
			return err
		}
		for position, questionVersionID := range input.QuestionVersionIds {
			if _, err := tx.Exec(ctx, `INSERT INTO canonical_audit_scope_selection_questions (operation_id,catalog_id,question_version_id,position,selection_digest) VALUES ($1,$2,$3,$4,$5)`, "scope-selection:"+chi.URLParam(request, "scopeId")+":"+input.OperationId, state.CatalogID, questionVersionID, position, digest); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `UPDATE canonical_audit_scope_drafts SET selected_question_count=$2, selection_digest=$3, updated_at=now() WHERE id=$1`, chi.URLParam(request, "scopeId"), len(input.QuestionVersionIds), digest)
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
	api.respond(writer, generated.CanonicalAuditScopeSelectionReceipt{OperationId: input.OperationId, Replayed: replayed, Selection: generated.AuditScopeSelectionDigest{SelectionDigest: digest, SelectedQuestionVersionIds: input.QuestionVersionIds, SelectedCount: int64(len(input.QuestionVersionIds)), CatalogVersion: state.CatalogVersion, UsageClass: generated.QuestionUsageClass(usage)}}, nil)
}

func requestPrincipalSubject(request *http.Request) string {
	principal, _ := PrincipalFromContext(request.Context())
	return principal.SubjectID
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
	if usage == questioncatalog.UsageClassPreprodExercise {
		if err := api.allowExerciseCatalog(); err != nil {
			api.respond(writer, nil, err)
			return
		}
	}
	// Reuse the catalog queue implementation while preserving an explicit
	// review-mode capability boundary. The catalog version is a required query
	// pin so a review cannot silently move across imported versions.
	_ = actor
	items, next, total, err := api.queryCanonicalCatalog(request.Context(), request.URL.Query().Get("catalogVersion"), usage, request.URL.Query().Get("search"), request.URL.Query().Get("formCode"), request.URL.Query().Get("domain"), request.URL.Query().Get("topic"), request.URL.Query().Get("riskBand"), request.URL.Query().Get("sourceGapState"), request.URL.Query().Get("selected"), request.URL.Query().Get("scopeId"), request.URL.Query().Get("cursor"), canonicalCatalogPageSize)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	capabilities := map[string]any{"canTechnicalApprove": false, "canPublish": false, "disabledReason": "Governed technical approval and publication remain on the existing candidate authority route."}
	if usage == questioncatalog.UsageClassPreprodExercise {
		capabilities["disabledReason"] = "PREPROD_EXERCISE review records cannot invoke technical approval or publication"
	}
	api.respond(writer, generated.QuestionReviewQueue{Mode: generated.QuestionReviewMode(usage), Items: items, NextCursor: next, TotalCount: total, Capabilities: capabilities}, nil)
}

// queryCanonicalCatalog is shared by the review queue and future New Audit
// selector handlers. It intentionally returns at most 25 rows.
func (api *CanonicalAPI) queryCanonicalCatalog(ctx context.Context, catalogVersion string, usage questioncatalog.UsageClass, search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID, cursor string, limit int) ([]generated.CanonicalQuestionCatalogEntry, *string, int64, error) {
	offset, err := parseCatalogCursor(cursor)
	if err != nil {
		return nil, nil, 0, err
	}
	var total int64
	selectionPredicate := `($9='' OR $9='all' OR (($9='selected') = EXISTS (SELECT 1 FROM canonical_audit_scope_selection_questions sq JOIN canonical_audit_scope_selection_operations so ON so.id=sq.operation_id WHERE so.id=(SELECT latest.id FROM canonical_audit_scope_selection_operations latest WHERE latest.scope_draft_id=$10 ORDER BY latest.created_at DESC, latest.id DESC LIMIT 1) AND sq.question_version_id=m.question_version_id)))`
	where := `c.catalog_version=$1 AND c.usage_class=$2 AND c.status='SEALED' AND ($3='' OR m.form_code ILIKE '%'||$3||'%' OR m.proposal_id ILIKE '%'||$3||'%') AND ($4='' OR m.form_code=$4) AND ($5='' OR COALESCE(m.proposed_domain,'')=$5) AND ($6='' OR COALESCE(m.proposed_topic,'')=$6) AND ($7='' OR COALESCE(m.proposed_risk_band,'')=$7) AND ($8='' OR m.source_gap_state=$8) AND ` + selectionPredicate
	if err := api.pool.QueryRow(ctx, `SELECT COUNT(*) FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id WHERE `+where, catalogVersion, string(usage), search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID).Scan(&total); err != nil {
		return nil, nil, 0, application.ErrNotFound
	}
	rows, err := api.pool.Query(ctx, `SELECT c.catalog_version,c.usage_class,m.question_version_id,m.form_code,m.proposal_id,m.ordinal,m.question_digest,COALESCE(m.source_locator,''),m.source_gap_state,COALESCE(m.proposed_domain,''),COALESCE(m.proposed_topic,''),COALESCE(m.proposed_risk_band,'') FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id WHERE `+where+` ORDER BY m.form_code,m.ordinal,m.question_version_id LIMIT $11 OFFSET $12`, catalogVersion, string(usage), search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID, limit+1, offset)
	if err != nil {
		return nil, nil, 0, err
	}
	defer rows.Close()
	items := make([]generated.CanonicalQuestionCatalogEntry, 0, limit)
	for rows.Next() {
		var row canonicalCatalogRow
		if err := rows.Scan(&row.CatalogVersion, &row.UsageClass, &row.QuestionID, &row.FormCode, &row.ProposalID, &row.Ordinal, &row.Digest, &row.SourceLocator, &row.SourceGap, &row.Domain, &row.Topic, &row.RiskBand); err != nil {
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

func (api *CanonicalAPI) commandCanonicalQuestionReview(writer http.ResponseWriter, request *http.Request) {
	actor, ok := api.requireCanonicalCatalogActor(writer, request, true)
	if !ok {
		return
	}
	var input generated.QuestionReviewCommandInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if strings.TrimSpace(input.OperationId) == "" ||
		strings.TrimSpace(input.QuestionVersionId) == "" ||
		strings.TrimSpace(input.Reason) == "" {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	usage, err := parseQuestionReviewMode(input.Mode)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if usage == questioncatalog.UsageClassPreprodExercise {
		if err := api.allowExerciseCatalog(); err != nil {
			api.respond(writer, nil, err)
			return
		}
		action := checklistgovernance.QuestionReviewAction(input.Action)
		switch action {
		case checklistgovernance.QuestionReviewActionRetain, checklistgovernance.QuestionReviewActionInclude, checklistgovernance.QuestionReviewActionExclude, checklistgovernance.QuestionReviewActionDefer, checklistgovernance.QuestionReviewActionDomainReclassified, checklistgovernance.QuestionReviewActionTopicReclassified:
		default:
			api.respond(writer, nil, fmt.Errorf("%w: exercise review cannot invoke %s", application.ErrForbidden, input.Action))
			return
		}
		if api.pool == nil {
			api.respond(writer, nil, application.ErrNotFound)
			return
		}
		draftID := "exercise-review:" + input.QuestionVersionId
		replayed := false
		if err := database.WithinTransaction(request.Context(), api.pool, func(ctx context.Context, tx pgx.Tx) error {
			var previousAction string
			if err := tx.QueryRow(ctx, `SELECT action FROM canonical_exercise_question_review_events WHERE event_id=$1`, input.OperationId).Scan(&previousAction); err == nil {
				if previousAction != input.Action {
					return fmt.Errorf("%w: operation replay payload differs", application.ErrConflict)
				}
				replayed = true
				return nil
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			var catalogID string
			if err := tx.QueryRow(ctx, `SELECT c.id FROM canonical_question_catalogs c JOIN canonical_question_catalog_memberships m ON m.catalog_id=c.id WHERE c.usage_class='PREPROD_EXERCISE' AND m.question_version_id=$1 AND c.status='SEALED'`, input.QuestionVersionId).Scan(&catalogID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return application.ErrNotFound
				}
				return err
			}
			disposition := input.Action
			if action == checklistgovernance.QuestionReviewActionDomainReclassified || action == checklistgovernance.QuestionReviewActionTopicReclassified {
				disposition = string(checklistgovernance.QuestionReviewActionRetain)
			}
			_, err := tx.Exec(ctx, `INSERT INTO canonical_exercise_question_review_drafts (id,catalog_id,question_version_id,usage_class,revision,disposition,reviewed_domain,reviewed_topic,controlled_reason,actor_subject_id) VALUES ($1,$2,$3,'PREPROD_EXERCISE',1,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`, draftID, catalogID, input.QuestionVersionId, disposition, input.Domain, input.Topic, input.Reason, actor.SubjectID)
			if err != nil {
				return err
			}
			payload, err := json.Marshal(map[string]any{"reason": input.Reason, "domain": input.Domain, "topic": input.Topic})
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `INSERT INTO canonical_exercise_question_review_events (event_id,draft_id,action,payload,actor_subject_id) VALUES ($1,$2,$3,$4,$5)`, input.OperationId, draftID, input.Action, payload, actor.SubjectID)
			return err
		}); err != nil {
			api.respond(writer, nil, err)
			return
		}
		api.respond(writer, generated.QuestionReviewCommandOutput{OperationId: input.OperationId, Mode: input.Mode, QuestionVersionId: input.QuestionVersionId, Action: input.Action, Replayed: replayed, CanPublish: false}, nil)
		return
	}
	// Governed disposition, technical approval, and publication remain the
	// existing candidate/Draft command boundaries. Do not return a successful
	// receipt from this catalog route without writing that governed aggregate.
	api.respond(writer, nil, fmt.Errorf("%w: use the governed candidate command boundary", application.ErrInvalid))
}
