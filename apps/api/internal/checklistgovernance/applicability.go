package checklistgovernance

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/jackc/pgx/v5"
)

var (
	ErrPublishedChecklistSelectionInvalid = errors.New("invalid published checklist selection")
	ErrPublishedChecklistNotApplicable    = errors.New("published checklist is not applicable")
	ErrPublishedChecklistQuestionConflict = errors.New("published checklist question identity conflicts")
)

// PublishedChecklistSelectionRequest is the complete CAA-side applicability
// decision. It is deliberately typed and becomes part of the immutable Audit
// package snapshot; it is never inferred from an organization type or UI label.
type PublishedChecklistSelectionRequest struct {
	OrganizationID      string
	InspectionType      string
	TargetID            string
	TargetKind          string
	DepartmentID        string
	At                  time.Time
	OperationQualifiers map[string]string
	ActivityQualifiers  map[string]string
}

// PublishedChecklistApplicability is the exact active scope/version candidate
// supplied by the persistence adapter. It intentionally carries the complete
// published question snapshots so composition cannot re-read mutable sources.
type PublishedChecklistApplicability struct {
	TemplateVersionID        string
	TemplateID               string
	Version                  int
	CandidateID              string
	CandidateGenerationRunID string
	CandidateRevision        int64
	CandidateContentDigest   string
	OrganizationID           string
	ProviderScopeID          string
	ProviderTypeID           string
	InspectionType           string
	TargetID                 string
	TargetKind               string
	DepartmentID             string
	EffectiveFrom            time.Time
	EffectiveTo              *time.Time
	OperationQualifiers      map[string]string
	ActivityQualifiers       map[string]string
	Questions                []regulatory.ChecklistQuestion
}

type PublishedChecklistVersionPin struct {
	TemplateVersionID      string `json:"templateVersionId"`
	TemplateID             string `json:"templateId"`
	Version                int    `json:"version"`
	CandidateContentDigest string `json:"candidateContentDigest"`
	ProviderScopeID        string `json:"providerScopeId"`
	ProviderTypeID         string `json:"providerTypeId"`
}

type PublishedChecklistApplicabilityPin struct {
	OrganizationID      string            `json:"organizationId"`
	InspectionType      string            `json:"inspectionType"`
	TargetID            string            `json:"targetId"`
	TargetKind          string            `json:"targetKind"`
	DepartmentID        string            `json:"departmentId"`
	EffectiveAt         time.Time         `json:"effectiveAt"`
	OperationQualifiers map[string]string `json:"operationQualifiers"`
	ActivityQualifiers  map[string]string `json:"activityQualifiers"`
}

// ComposedPublishedChecklistPackage is the data to persist verbatim in an
// inspection package. The storage layer owns serialization/digesting; this
// package owns deterministic selection and duplicate-question denial.
type ComposedPublishedChecklistPackage struct {
	PublishedVersions []PublishedChecklistVersionPin     `json:"publishedVersions"`
	Applicability     PublishedChecklistApplicabilityPin `json:"applicability"`
	Questions         []regulatory.ChecklistQuestion     `json:"questions"`
}

// MaterializeApplicablePublishedPackageCommand creates a runner-compatible,
// immutable Audit package only from a full governed applicability decision.
// The caller must assign every composed question explicitly; no Inspector can
// choose, alter, or silently omit a governed question.
type MaterializeApplicablePublishedPackageCommand struct {
	OperationID                 string
	IdempotencyKey              string
	CorrelationID               string
	InspectionID                string
	PackageID                   string
	PackageVersion              int
	ExpiresAt                   time.Time
	Selection                   PublishedChecklistSelectionRequest
	AssignedInspectorSubjectIDs map[string][]string
}

type MaterializedApplicablePublishedPackage struct {
	InspectionID      string `json:"inspectionId"`
	PackageID         string `json:"packageId"`
	TemplateVersionID string `json:"templateVersionId"`
	PackageDigest     string `json:"packageDigest"`
}

// MaterializeApplicablePublishedPackage is the server-side bridge from
// governed publication to the existing checklist runner. It persists the full
// ordered source questions, version/digest pins, and scope decision in the
// package snapshot before any Inspector can answer it.
func (service *Service) MaterializeApplicablePublishedPackage(
	ctx context.Context,
	actor identity.Principal,
	command MaterializeApplicablePublishedPackageCommand,
) (MaterializedApplicablePublishedPackage, error) {
	if strings.TrimSpace(command.OperationID) == "" || strings.TrimSpace(command.CorrelationID) == "" ||
		strings.TrimSpace(command.InspectionID) == "" || strings.TrimSpace(command.PackageID) == "" ||
		command.PackageVersion <= 0 || command.ExpiresAt.IsZero() || !command.ExpiresAt.After(service.Clock()) {
		return MaterializedApplicablePublishedPackage{}, application.ErrInvalid
	}
	assignments, err := service.currentAssignments(ctx, service.Pool, actor)
	if err != nil {
		return MaterializedApplicablePublishedPackage{}, err
	}
	authorizedDepartment := false
	for _, current := range assignments {
		if current.DepartmentID == command.Selection.DepartmentID {
			authorizedDepartment = true
			break
		}
	}
	if !authorizedDepartment {
		return MaterializedApplicablePublishedPackage{}, application.ErrForbidden
	}
	versions, err := service.ListApplicablePublishedVersions(ctx, command.Selection)
	if err != nil {
		return MaterializedApplicablePublishedPackage{}, err
	}
	if err := service.validatePublishedVersionsForExecution(ctx, service.Pool, versions); err != nil {
		return MaterializedApplicablePublishedPackage{}, err
	}
	composed, err := ComposeApplicablePublishedPackage(command.Selection, versions)
	if err != nil {
		return MaterializedApplicablePublishedPackage{}, err
	}
	snapshot, err := runnerPackageSnapshot(composed, command.AssignedInspectorSubjectIDs)
	if err != nil {
		return MaterializedApplicablePublishedPackage{}, err
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return MaterializedApplicablePublishedPackage{}, err
	}
	digest, err := regulatory.CanonicalSHA256(snapshot)
	if err != nil {
		return MaterializedApplicablePublishedPackage{}, err
	}
	result := MaterializedApplicablePublishedPackage{
		InspectionID: command.InspectionID, PackageID: command.PackageID,
		TemplateVersionID: composed.PublishedVersions[0].TemplateVersionID, PackageDigest: digest,
	}
	idempotencyKey := strings.TrimSpace(command.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = command.OperationID
	}
	semanticHash, err := regulatory.CanonicalSHA256(map[string]any{
		"kind": "materialize_applicable_published_package", "inspectionId": command.InspectionID,
		"packageId": command.PackageID, "packageVersion": command.PackageVersion,
		"expiresAt": command.ExpiresAt.UTC(), "snapshot": snapshot,
	})
	if err != nil {
		return MaterializedApplicablePublishedPackage{}, err
	}
	responseBody, err := json.Marshal(result)
	if err != nil {
		return MaterializedApplicablePublishedPackage{}, err
	}
	idempotencyScope := actor.SubjectID + ":materialize_applicable_published_package"
	if err := database.WithinTransaction(ctx, service.Pool, func(ctx context.Context, transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", idempotencyScope+":"+idempotencyKey); err != nil {
			return err
		}
		var storedHash string
		var storedBody []byte
		err := transaction.QueryRow(ctx, `
			SELECT semantic_hash,response_body FROM idempotency_responses WHERE scope=$1 AND operation_id=$2
		`, idempotencyScope, command.OperationID).Scan(&storedHash, &storedBody)
		if err == nil {
			if storedHash != semanticHash {
				return application.ErrConflict
			}
			if err := json.Unmarshal(storedBody, &result); err != nil {
				return err
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var organizationID, inspectionType, inspectionStatus string
		if err := transaction.QueryRow(ctx, `
			SELECT organization_id,inspection_type,status FROM inspections WHERE id=$1 FOR UPDATE
		`, command.InspectionID).Scan(&organizationID, &inspectionType, &inspectionStatus); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return application.ErrNotFound
			}
			return err
		}
		if organizationID != command.Selection.OrganizationID || inspectionType != command.Selection.InspectionType || inspectionStatus != "PREPARATION" {
			return application.ErrConflict
		}
		if err := service.lockPublishedVersionSourceCurrentness(ctx, transaction, versions); err != nil {
			return err
		}
		// A published version is immutable, but its source-currentness projection
		// is live. Recheck inside the write transaction so a newly recorded source
		// impact cannot turn an old version into a fresh executable Audit package.
		if err := service.validatePublishedVersionsForExecution(ctx, transaction, versions); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO inspection_packages (id,inspection_id,checklist_template_version_id,package_version,snapshot,expires_at,created_at,package_digest)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		`, command.PackageID, command.InspectionID, result.TemplateVersionID, command.PackageVersion,
			snapshotJSON, command.ExpiresAt.UTC(), service.Clock().UTC(), digest); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `INSERT INTO inspection_checklists (inspection_id,status,revision) VALUES ($1,'IN_PROGRESS',1)`, command.InspectionID); err != nil {
			return err
		}
		for _, question := range snapshot.Questions {
			for _, subjectID := range question.AssignedInspectorUserIDs {
				if _, err := transaction.Exec(ctx, `
					INSERT INTO inspection_question_assignments (inspection_id,question_id,subject_id,assignment_revision)
					VALUES ($1,$2,$3,1)
				`, command.InspectionID, question.ID, subjectID); err != nil {
					return err
				}
			}
		}
		if changed, err := transaction.Exec(ctx, `
			UPDATE inspections SET status='READY_TO_EXECUTE',revision=revision+1,updated_at=$2
			WHERE id=$1 AND status='PREPARATION'
		`, command.InspectionID, service.Clock().UTC()); err != nil {
			return err
		} else if changed.RowsAffected() != 1 {
			return application.ErrConflict
		}
		role := ""
		if len(actor.Roles) > 0 {
			role = string(actor.Roles[0])
		}
		details, err := json.Marshal(map[string]any{
			"publishedVersions": composed.PublishedVersions, "applicability": composed.Applicability,
			"packageDigest": digest,
		})
		if err != nil {
			return err
		}
		_, err = transaction.Exec(ctx, `
			INSERT INTO audit_events (event_id,occurred_at,actor_subject_id,actor_role,organization_id,action,entity_type,entity_id,entity_version,before_status,after_status,operation_id,correlation_id,request_id,details)
			VALUES ($1,$2,$3,$4,$5,'inspection.applicable_package_materialized','inspection_package',$6,$7,'PREPARATION','READY_TO_EXECUTE',$8,$9,$9,$10)
		`, "AE-"+command.OperationID, service.Clock().UTC(), actor.SubjectID, role, organizationID,
			command.PackageID, command.PackageVersion, command.OperationID, command.CorrelationID, details)
		if err != nil {
			return err
		}
		_, err = transaction.Exec(ctx, `
			INSERT INTO idempotency_responses (scope,operation_id,semantic_hash,response_status,response_headers,response_body,created_at)
			VALUES ($1,$2,$3,200,'{}'::jsonb,$4,$5)
		`, idempotencyScope, command.OperationID, semanticHash, responseBody, service.Clock().UTC())
		return err
	}); err != nil {
		return MaterializedApplicablePublishedPackage{}, err
	}
	return result, nil
}

// lockPublishedVersionSourceCurrentness shares an advisory lock with the
// source-impact import path. It locks every source identity in deterministic
// order before revalidating the selected published versions, so either the
// materialized package sees no committed change or it sees the stale impact
// and fails closed.
func (service *Service) lockPublishedVersionSourceCurrentness(
	ctx context.Context,
	tx pgx.Tx,
	versions []PublishedChecklistApplicability,
) error {
	runIDs := make([]string, 0, len(versions))
	for _, version := range versions {
		runIDs = append(runIDs, version.CandidateGenerationRunID)
	}
	return service.lockGenerationRunSourceCurrentness(ctx, tx, runIDs)
}

// lockGenerationRunSourceCurrentness acquires the shared source-chain locks
// for immutable candidate lineage. Publication and package materialization use
// this same helper before their final currentness validation, while source
// impact import takes the matching source identity lock before its event is
// committed.
func (service *Service) lockGenerationRunSourceCurrentness(
	ctx context.Context,
	tx pgx.Tx,
	generationRunIDs []string,
) error {
	identities := map[string]bool{}
	seenRuns := map[string]bool{}
	for _, generationRunID := range generationRunIDs {
		if generationRunID == "" || seenRuns[generationRunID] {
			continue
		}
		seenRuns[generationRunID] = true
		rows, err := tx.Query(ctx, `
			SELECT DISTINCT source.source_identity
			FROM regulatory_generation_run_source_snapshots snapshot
			JOIN regulatory_source_versions source
			  ON source.id=snapshot.regulatory_source_version_id
			WHERE snapshot.generation_run_id=$1
			ORDER BY source.source_identity`, generationRunID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var identity string
			if err := rows.Scan(&identity); err != nil {
				rows.Close()
				return err
			}
			identities[identity] = true
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}
	ordered := make([]string, 0, len(identities))
	for identity := range identities {
		ordered = append(ordered, identity)
	}
	slices.Sort(ordered)
	for _, identity := range ordered {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", regulatory.SourceCurrentnessLockKey(identity)); err != nil {
			return err
		}
	}
	return nil
}

type runnerQuestionSnapshot struct {
	ID                       string   `json:"id"`
	SectionID                string   `json:"sectionId"`
	Prompt                   string   `json:"prompt"`
	RegulatoryReference      string   `json:"regulatoryReference"`
	ExpectedEvidence         string   `json:"expectedEvidence"`
	AssignedInspectorUserIDs []string `json:"assignedInspectorUserIds"`
}

type governedRunnerPackageSnapshot struct {
	SchemaVersion      int                                `json:"schemaVersion"`
	ProtocolVersion    int                                `json:"protocolVersion"`
	PublishedVersions  []PublishedChecklistVersionPin     `json:"publishedVersions"`
	Applicability      PublishedChecklistApplicabilityPin `json:"applicability"`
	PublishedQuestions []regulatory.ChecklistQuestion     `json:"publishedQuestions"`
	Questions          []runnerQuestionSnapshot           `json:"questions"`
}

func runnerPackageSnapshot(composed ComposedPublishedChecklistPackage, assigned map[string][]string) (governedRunnerPackageSnapshot, error) {
	output := governedRunnerPackageSnapshot{
		SchemaVersion: 1, ProtocolVersion: 1, PublishedVersions: slices.Clone(composed.PublishedVersions),
		Applicability: composed.Applicability, PublishedQuestions: slices.Clone(composed.Questions),
		Questions: make([]runnerQuestionSnapshot, 0, len(composed.Questions)),
	}
	seen := map[string]bool{}
	for _, question := range composed.Questions {
		ids := slices.Clone(assigned[question.QuestionID])
		slices.Sort(ids)
		if question.QuestionID == "" || len(ids) == 0 || seen[question.QuestionID] {
			return governedRunnerPackageSnapshot{}, application.ErrInvalid
		}
		for index, id := range ids {
			if strings.TrimSpace(id) == "" || (index > 0 && ids[index-1] == id) {
				return governedRunnerPackageSnapshot{}, application.ErrInvalid
			}
		}
		seen[question.QuestionID] = true
		citations, err := json.Marshal(question.Citations)
		if err != nil {
			return governedRunnerPackageSnapshot{}, err
		}
		output.Questions = append(output.Questions, runnerQuestionSnapshot{
			ID: question.QuestionID, SectionID: "GOVERNED", Prompt: question.Prompt,
			RegulatoryReference: string(citations), ExpectedEvidence: strings.Join(question.ExpectedEvidence, "\n"),
			AssignedInspectorUserIDs: ids,
		})
	}
	if len(assigned) != len(seen) {
		return governedRunnerPackageSnapshot{}, application.ErrInvalid
	}
	return output, nil
}

// ListApplicablePublishedVersions resolves only immutable governed versions
// whose frozen generation lineage still matches a current effective provider
// scope and typed target. It does not return historical direct-published
// templates, and it does not infer applicability from organization type.
func (service *Service) ListApplicablePublishedVersions(
	ctx context.Context,
	request PublishedChecklistSelectionRequest,
) ([]PublishedChecklistApplicability, error) {
	if !validSelectionRequest(request) {
		return nil, ErrPublishedChecklistSelectionInvalid
	}
	rows, err := service.Pool.Query(ctx, `
		SELECT version.id,version.template_id,version.version,
		       candidate.id,candidate.generation_run_id,candidate.revision,
		       version.candidate_content_digest,current_scope.id,
		       current_scope.service_provider_type_id,run.inspection_type,
		       current_scope.organization_id,target.id,target.target_kind,
		       owner.department_id,current_scope.effective_from,current_scope.effective_to,
		       current_scope.operation_qualifiers,current_scope.activity_qualifiers,
		       version.snapshot
		FROM checklist_template_versions version
		JOIN template_draft_versions candidate
		  ON candidate.id=version.candidate_draft_version_id
		 AND candidate.revision=version.candidate_revision
		 AND candidate.candidate_content_digest=version.candidate_content_digest
		 AND candidate.status='PUBLISHED'
		JOIN regulatory_generation_runs run ON run.id=candidate.generation_run_id
		JOIN regulatory_generation_run_scope_facts frozen
		  ON frozen.generation_run_id=run.id
		JOIN LATERAL (
		  SELECT DISTINCT ON (scope.root_id) scope.*
		  FROM organization_service_provider_scopes scope
		  WHERE scope.root_id=frozen.scope_root_id
		    AND scope.organization_id=$1
		    AND scope.effective_from <= $6::date
		  ORDER BY scope.root_id,scope.effective_from DESC,scope.id DESC
		) current_scope ON current_scope.status='ACTIVE'
		  AND (current_scope.effective_to IS NULL OR current_scope.effective_to > $6::date)
		  AND current_scope.service_provider_type_id=frozen.service_provider_type_id
		  AND (current_scope.primary_target_id=$3 OR EXISTS (
			SELECT 1 FROM organization_service_provider_scope_targets current_target
			WHERE current_target.organization_service_provider_scope_id=current_scope.id
			  AND current_target.regulated_target_id=$3
		  ))
		JOIN regulated_targets target ON target.id=run.target_id
		JOIN candidate_required_owner_assignments owner
		  ON owner.candidate_draft_version_id=candidate.id
		 AND owner.candidate_revision=candidate.revision
		 AND owner.candidate_content_digest=candidate.candidate_content_digest
		 AND owner.approval_required
		WHERE run.inspection_type=$2
		  AND run.target_id=$3 AND target.target_kind=$4
		  AND owner.department_id=$5
		ORDER BY version.id,owner.department_id`,
		request.OrganizationID, request.InspectionType, request.TargetID,
		request.TargetKind, request.DepartmentID, request.At.UTC(),
	)
	if err != nil {
		return nil, err
	}
	versions := []PublishedChecklistApplicability{}
	seen := map[string]bool{}
	for rows.Next() {
		var version PublishedChecklistApplicability
		var operationQualifiers, activityQualifiers, snapshot []byte
		if err := rows.Scan(
			&version.TemplateVersionID, &version.TemplateID, &version.Version,
			&version.CandidateID, &version.CandidateGenerationRunID, &version.CandidateRevision,
			&version.CandidateContentDigest, &version.ProviderScopeID,
			&version.ProviderTypeID, &version.InspectionType, &version.OrganizationID,
			&version.TargetID, &version.TargetKind, &version.DepartmentID,
			&version.EffectiveFrom, &version.EffectiveTo, &operationQualifiers,
			&activityQualifiers, &snapshot,
		); err != nil {
			return nil, err
		}
		if seen[version.TemplateVersionID] {
			continue
		}
		seen[version.TemplateVersionID] = true
		if err := json.Unmarshal(operationQualifiers, &version.OperationQualifiers); err != nil {
			return nil, fmt.Errorf("decode published scope operation qualifiers: %w", err)
		}
		if err := json.Unmarshal(activityQualifiers, &version.ActivityQualifiers); err != nil {
			return nil, fmt.Errorf("decode published scope activity qualifiers: %w", err)
		}
		var immutable struct {
			Questions []regulatory.ChecklistQuestion `json:"questions"`
		}
		if err := json.Unmarshal(snapshot, &immutable); err != nil {
			return nil, fmt.Errorf("decode immutable published checklist snapshot: %w", err)
		}
		version.Questions = immutable.Questions
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	applicable, err := ResolveApplicablePublishedVersions(request, versions)
	if err != nil {
		return nil, err
	}
	if err := service.validatePublishedVersionsForExecution(ctx, service.Pool, applicable); err != nil {
		return nil, err
	}
	return applicable, nil
}

// validatePublishedVersionsForExecution revalidates immutable published
// question snapshots against live source-currentness and review facts before a
// new executable Audit package may use them. It never mutates the published
// version or any package already pinned to it.
func (service *Service) validatePublishedVersionsForExecution(
	ctx context.Context,
	query interface {
		Query(context.Context, string, ...any) (pgx.Rows, error)
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	versions []PublishedChecklistApplicability,
) error {
	for _, version := range versions {
		blockers, err := regulatory.LoadCandidateBlockingIssues(ctx, query, regulatory.CandidateView{
			CandidateID: version.CandidateID, GenerationRunID: version.CandidateGenerationRunID,
			Revision: version.CandidateRevision, ContentDigest: version.CandidateContentDigest,
		})
		if err != nil {
			return err
		}
		if len(blockers) > 0 {
			return &regulatory.ValidationError{Issues: blockers}
		}
	}
	return nil
}

func ResolveApplicablePublishedVersions(
	request PublishedChecklistSelectionRequest,
	versions []PublishedChecklistApplicability,
) ([]PublishedChecklistApplicability, error) {
	if !validSelectionRequest(request) {
		return nil, ErrPublishedChecklistSelectionInvalid
	}
	applicable := make([]PublishedChecklistApplicability, 0, len(versions))
	for _, version := range versions {
		if publishedChecklistMatches(request, version) {
			version.OperationQualifiers = maps.Clone(version.OperationQualifiers)
			version.ActivityQualifiers = maps.Clone(version.ActivityQualifiers)
			version.Questions = slices.Clone(version.Questions)
			applicable = append(applicable, version)
		}
	}
	slices.SortFunc(applicable, func(left, right PublishedChecklistApplicability) int {
		return cmp.Or(
			cmp.Compare(left.TemplateVersionID, right.TemplateVersionID),
			cmp.Compare(left.TemplateID, right.TemplateID),
			cmp.Compare(left.Version, right.Version),
		)
	})
	return applicable, nil
}

func ComposeApplicablePublishedPackage(
	request PublishedChecklistSelectionRequest,
	versions []PublishedChecklistApplicability,
) (ComposedPublishedChecklistPackage, error) {
	applicable, err := ResolveApplicablePublishedVersions(request, versions)
	if err != nil {
		return ComposedPublishedChecklistPackage{}, err
	}
	if len(applicable) == 0 {
		return ComposedPublishedChecklistPackage{}, ErrPublishedChecklistNotApplicable
	}
	output := ComposedPublishedChecklistPackage{
		PublishedVersions: make([]PublishedChecklistVersionPin, 0, len(applicable)),
		Applicability: PublishedChecklistApplicabilityPin{
			OrganizationID: request.OrganizationID, InspectionType: request.InspectionType,
			TargetID: request.TargetID, TargetKind: request.TargetKind,
			DepartmentID: request.DepartmentID, EffectiveAt: request.At.UTC(),
			OperationQualifiers: maps.Clone(request.OperationQualifiers),
			ActivityQualifiers:  maps.Clone(request.ActivityQualifiers),
		},
		Questions: []regulatory.ChecklistQuestion{},
	}
	questions := map[string]regulatory.ChecklistQuestion{}
	for _, version := range applicable {
		output.PublishedVersions = append(output.PublishedVersions, PublishedChecklistVersionPin{
			TemplateVersionID: version.TemplateVersionID, TemplateID: version.TemplateID,
			Version: version.Version, CandidateContentDigest: version.CandidateContentDigest,
			ProviderScopeID: version.ProviderScopeID, ProviderTypeID: version.ProviderTypeID,
		})
		for _, question := range version.Questions {
			if strings.TrimSpace(question.QuestionID) == "" {
				return ComposedPublishedChecklistPackage{}, fmt.Errorf("%w: blank question identity", ErrPublishedChecklistQuestionConflict)
			}
			if previous, exists := questions[question.QuestionID]; exists {
				if !sameQuestionSnapshot(previous, question) {
					return ComposedPublishedChecklistPackage{}, fmt.Errorf(
						"%w: %s", ErrPublishedChecklistQuestionConflict, question.QuestionID,
					)
				}
				continue
			}
			questions[question.QuestionID] = question
			output.Questions = append(output.Questions, question)
		}
	}
	return output, nil
}

func validSelectionRequest(request PublishedChecklistSelectionRequest) bool {
	return strings.TrimSpace(request.OrganizationID) != "" &&
		strings.TrimSpace(request.InspectionType) != "" &&
		strings.TrimSpace(request.TargetID) != "" &&
		strings.TrimSpace(request.TargetKind) != "" &&
		strings.TrimSpace(request.DepartmentID) != "" && !request.At.IsZero()
}

func publishedChecklistMatches(request PublishedChecklistSelectionRequest, version PublishedChecklistApplicability) bool {
	if version.TemplateVersionID == "" || version.TemplateID == "" || version.Version <= 0 ||
		version.CandidateContentDigest == "" || version.ProviderScopeID == "" ||
		version.ProviderTypeID == "" || len(version.Questions) == 0 ||
		version.OrganizationID != request.OrganizationID ||
		version.InspectionType != request.InspectionType ||
		version.TargetID != request.TargetID || version.TargetKind != request.TargetKind ||
		version.DepartmentID != request.DepartmentID || version.EffectiveFrom.After(request.At) ||
		(version.EffectiveTo != nil && !request.At.Before(*version.EffectiveTo)) {
		return false
	}
	return qualifiersMatch(request.OperationQualifiers, version.OperationQualifiers) &&
		qualifiersMatch(request.ActivityQualifiers, version.ActivityQualifiers)
}

func qualifiersMatch(required, actual map[string]string) bool {
	if len(required) != len(actual) {
		return false
	}
	for key, value := range actual {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" || required[key] != value {
			return false
		}
	}
	return true
}

func sameQuestionSnapshot(left, right regulatory.ChecklistQuestion) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}
