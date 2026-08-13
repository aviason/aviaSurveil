package regulatory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrCandidateNotFound = fmt.Errorf("%w: governed candidate or generation run was not found", application.ErrNotFound)

type ValidationIssue struct {
	FieldPath      string `json:"fieldPath"`
	Code           string `json:"code"`
	Message        string `json:"message"`
	SourceIdentity string `json:"sourceIdentity"`
	SourceHash     string `json:"sourceHash"`
	ClauseID       string `json:"clauseId"`
	Locator        string `json:"locator"`
}

type ValidationError struct {
	Issues []ValidationIssue
}

func (err *ValidationError) Error() string {
	if err == nil || len(err.Issues) == 0 {
		return "governed candidate validation failed"
	}
	return err.Issues[0].Message
}

func (err *ValidationError) Unwrap() error { return application.ErrInvalid }

type RequiredOwner struct {
	DepartmentID         string `json:"departmentId"`
	OrganizationalUnitID string `json:"organizationalUnitId"`
	ApprovalRequired     bool   `json:"approvalRequired"`
}

type CandidateSourceLineageView struct {
	SourceID        string `json:"sourceId"`
	SourceIdentity  string `json:"sourceIdentity"`
	VersionIdentity string `json:"versionIdentity"`
	SourceHash      string `json:"sourceHash"`
	ClauseID        string `json:"clauseId"`
	Locator         string `json:"locator"`
}

type CandidateView struct {
	CandidateID           string                       `json:"candidateId"`
	CandidateRootID       string                       `json:"candidateRootId"`
	SupersedesCandidateID *string                      `json:"supersedesCandidateId"`
	GenerationRunID       string                       `json:"generationRunId"`
	TemplateID            string                       `json:"templateId"`
	Version               int64                        `json:"version"`
	Revision              int64                        `json:"revision"`
	Status                string                       `json:"status"`
	ContentDigest         string                       `json:"contentDigest"`
	SchemaVersion         string                       `json:"schemaVersion"`
	ChangeReason          string                       `json:"changeReason"`
	SourceSnapshots       []CandidateSourceLineageView `json:"sourceSnapshots"`
	ScopeFactIDs          []string                     `json:"scopeFactIds"`
	CrosswalkPartitionIDs []string                     `json:"crosswalkPartitionIds"`
	Mappings              []ComplianceMapping          `json:"mappings"`
	Questions             []ChecklistQuestion          `json:"questions"`
	RequiredOwners        []RequiredOwner              `json:"requiredOwners"`
}

type GenerationRunView struct {
	GenerationRunID, Status, InputDigest, InputSchemaVersion, GenerationPolicyVersion               string
	ProviderCatalogVersion, ProviderID, ProviderAdapterVersion, InspectionType, TargetID, RequestID string
	OutputDigest                                                                                    *string
	Failure                                                                                         *GenerationFailureView
	Candidate                                                                                       *CandidateView
}

type GenerationFailureView struct {
	Code, Reason, RequestID, OperationID, IdempotencyKey string
}

type SourceSnapshotView struct {
	SourceID, SourceIdentity, VersionIdentity, Title, SourceHash, Locator, ClauseID, ClauseLocator string
	Partitions                                                                                     []SourcePartitionFactView
	ApplicabilityFacts                                                                             []SourceApplicabilityFactView
	UnresolvedGaps                                                                                 []UnresolvedSourceGap
	GenerationRunIDs, CandidateIDs                                                                 []string
}

// SourceImpactView is a read-only projection over immutable source lineage.
// It deliberately identifies affected historical artifacts without changing
// their source, candidate, publication, or Audit records.
type SourceImpactView struct {
	SourceID       string   `json:"sourceId"`
	SourceIdentity string   `json:"sourceIdentity"`
	SourceHash     string   `json:"sourceHash"`
	ClauseIDs      []string `json:"clauseIds"`
	CandidateIDs   []string `json:"candidateIds"`
	MappingIDs     []string `json:"mappingIds"`
	QuestionIDs    []string `json:"questionIds"`
	ScopeFactIDs   []string `json:"scopeFactIds"`
}

type SourcePartitionFactView struct {
	EvaluationID      string `json:"evaluationId"`
	PartitionID       string `json:"partitionId"`
	Role              string `json:"role"`
	CrosswalkRowID    string `json:"crosswalkRowId"`
	StableRowIdentity string `json:"stableRowIdentity"`
}

type SourceApplicabilityFactView struct {
	CandidateID   string     `json:"candidateId"`
	MappingID     string     `json:"mappingId"`
	Relationship  string     `json:"relationship"`
	Applicability string     `json:"applicability"`
	SourceGap     *SourceGap `json:"sourceGap"`
}

type EditCommand struct {
	OperationID, IdempotencyKey, CandidateID, ExpectedContentDigest, ChangeReason string
	ExpectedRevision                                                              int64
	Mappings                                                                      []ComplianceMapping
	Questions                                                                     []ChecklistQuestion
	RequiredOwners                                                                []RequiredOwner
}
type SubmitCommand struct {
	OperationID, IdempotencyKey, CandidateID, ExpectedContentDigest, Reason string
	ExpectedRevision                                                        int64
}

func importSemanticDigest(operationID string, bundle CandidateBundle) (string, error) {
	return CanonicalSHA256(map[string]any{
		"operationId":     operationID,
		"candidateBundle": bundle,
	})
}

func editSemanticDigest(command EditCommand) (string, error) {
	return CanonicalSHA256(map[string]any{
		"candidateId":           command.CandidateID,
		"expectedRevision":      command.ExpectedRevision,
		"expectedContentDigest": command.ExpectedContentDigest,
		"changeReason":          command.ChangeReason,
		"mappings":              command.Mappings,
		"questions":             command.Questions,
		"requiredOwners":        command.RequiredOwners,
	})
}

func submitSemanticDigest(command SubmitCommand) (string, error) {
	return CanonicalSHA256(map[string]any{
		"operationId":           command.OperationID,
		"idempotencyKey":        command.IdempotencyKey,
		"candidateId":           command.CandidateID,
		"expectedContentDigest": command.ExpectedContentDigest,
		"reason":                command.Reason,
		"expectedRevision":      command.ExpectedRevision,
	})
}

// AdminService is deliberately limited to Admin inspection, candidate revision,
// and submission. It has no technical-approval or publication operation.
type AdminService struct {
	Pool  *database.Pool
	Clock func() time.Time
}

func NewAdminService(pool *database.Pool, clock func() time.Time) *AdminService {
	if clock == nil {
		clock = time.Now
	}
	return &AdminService{Pool: pool, Clock: clock}
}

// LoadCandidateForGovernance exposes the immutable governed-candidate
// projection to the dedicated Department Manager lifecycle service without
// granting that service Admin command authority.
func LoadCandidateForGovernance(ctx context.Context, pool *database.Pool, candidateID string) (CandidateView, error) {
	return (&AdminService{Pool: pool}).getCandidate(ctx, pool, candidateID)
}

func LoadCandidateForGovernanceQuery(ctx context.Context, query interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}, candidateID string) (CandidateView, error) {
	return (&AdminService{}).getCandidate(ctx, query, candidateID)
}

func (service *AdminService) requireAdmin(actor identity.Principal) error {
	if !actor.HasRole(identity.RoleAdmin) {
		return fmt.Errorf("%w: Admin authority is required", application.ErrForbidden)
	}
	if service == nil || service.Pool == nil {
		return fmt.Errorf("%w: governed candidate database is required", application.ErrInvalid)
	}
	return nil
}

func (service *AdminService) ListSources(ctx context.Context, actor identity.Principal) ([]SourceSnapshotView, error) {
	if err := service.requireAdmin(actor); err != nil {
		return nil, err
	}
	rows, err := service.Pool.Query(ctx, `SELECT source.id, source.source_identity, source.version_identity, source.title, source.source_hash, source.source_locator, clause.id, clause.clause_locator FROM regulatory_source_versions source JOIN regulatory_normalized_clauses clause ON clause.regulatory_source_version_id=source.id ORDER BY source.id, clause.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SourceSnapshotView{}
	for rows.Next() {
		var view SourceSnapshotView
		if err := rows.Scan(&view.SourceID, &view.SourceIdentity, &view.VersionIdentity, &view.Title, &view.SourceHash, &view.Locator, &view.ClauseID, &view.ClauseLocator); err != nil {
			return nil, err
		}
		view.Partitions = []SourcePartitionFactView{}
		view.ApplicabilityFacts = []SourceApplicabilityFactView{}
		view.UnresolvedGaps = []UnresolvedSourceGap{}
		if err := service.Pool.QueryRow(ctx, `SELECT COALESCE(array_agg(DISTINCT run.id ORDER BY run.id), '{}'::text[]), COALESCE(array_agg(DISTINCT candidate.id ORDER BY candidate.id) FILTER (WHERE candidate.id IS NOT NULL), '{}'::text[]) FROM regulatory_generation_run_source_snapshots snapshot JOIN regulatory_generation_runs run ON run.id=snapshot.generation_run_id LEFT JOIN template_draft_versions candidate ON candidate.generation_run_id=run.id WHERE snapshot.regulatory_source_version_id=$1 AND snapshot.regulatory_normalized_clause_id=$2`, view.SourceID, view.ClauseID).Scan(&view.GenerationRunIDs, &view.CandidateIDs); err != nil {
			return nil, err
		}
		partitionRows, err := service.Pool.Query(ctx, `SELECT partition.evaluation_id, partition.id, partition.partition_kind, crosswalk.id, row.stable_row_identity FROM state_compliance_crosswalk_rows crosswalk JOIN regulatory_evaluation_partition_rows row ON row.state_compliance_crosswalk_row_id=crosswalk.id JOIN regulatory_evaluation_partitions partition ON partition.id=row.partition_id WHERE crosswalk.regulatory_source_version_id=$1 AND crosswalk.normalized_clause_id=$2 ORDER BY partition.id, row.stable_row_identity`, view.SourceID, view.ClauseID)
		if err != nil {
			return nil, err
		}
		for partitionRows.Next() {
			var fact SourcePartitionFactView
			if err := partitionRows.Scan(&fact.EvaluationID, &fact.PartitionID, &fact.Role, &fact.CrosswalkRowID, &fact.StableRowIdentity); err != nil {
				partitionRows.Close()
				return nil, err
			}
			view.Partitions = append(view.Partitions, fact)
		}
		if err := partitionRows.Err(); err != nil {
			partitionRows.Close()
			return nil, err
		}
		partitionRows.Close()
		applicabilityRows, err := service.Pool.Query(ctx, `SELECT mapping.candidate_draft_version_id, mapping.mapping_id, mapping.snapshot->>'relationship', mapping.snapshot->>'applicability', mapping.snapshot->'sourceGap' FROM regulatory_generated_mapping_snapshots mapping WHERE EXISTS (SELECT 1 FROM jsonb_array_elements(mapping.snapshot->'citations') citation WHERE citation->>'sourceSnapshotId'=$1 AND citation->>'clauseId'=$2) ORDER BY mapping.candidate_draft_version_id, mapping.mapping_ordinal`, view.SourceID, view.ClauseID)
		if err != nil {
			return nil, err
		}
		for applicabilityRows.Next() {
			var fact SourceApplicabilityFactView
			var sourceGapJSON []byte
			if err := applicabilityRows.Scan(&fact.CandidateID, &fact.MappingID, &fact.Relationship, &fact.Applicability, &sourceGapJSON); err != nil {
				applicabilityRows.Close()
				return nil, err
			}
			if string(sourceGapJSON) != "null" {
				var gap SourceGap
				if err := json.Unmarshal(sourceGapJSON, &gap); err != nil {
					applicabilityRows.Close()
					return nil, err
				}
				fact.SourceGap = &gap
			}
			view.ApplicabilityFacts = append(view.ApplicabilityFacts, fact)
		}
		if err := applicabilityRows.Err(); err != nil {
			applicabilityRows.Close()
			return nil, err
		}
		applicabilityRows.Close()
		gapRows, err := service.Pool.Query(ctx, `SELECT gap_id, reason FROM regulatory_source_gap_facts WHERE regulatory_source_version_id=$1 ORDER BY ordinal, gap_id`, view.SourceID)
		if err != nil {
			return nil, err
		}
		for gapRows.Next() {
			var gap UnresolvedSourceGap
			if err := gapRows.Scan(&gap.GapID, &gap.Reason); err != nil {
				gapRows.Close()
				return nil, err
			}
			view.UnresolvedGaps = append(view.UnresolvedGaps, gap)
		}
		if err := gapRows.Err(); err != nil {
			gapRows.Close()
			return nil, err
		}
		gapRows.Close()
		items = append(items, view)
	}
	return items, rows.Err()
}

// GetSourceImpact connects one exact source hash to every persisted clause,
// candidate, mapping, question, and provider scope reachable through immutable
// generation lineage. The projection is read-only and does not infer legal
// applicability or modify a published checklist.
func (service *AdminService) GetSourceImpact(ctx context.Context, actor identity.Principal, sourceID string) (SourceImpactView, error) {
	if err := service.requireAdmin(actor); err != nil {
		return SourceImpactView{}, err
	}
	if strings.TrimSpace(sourceID) == "" {
		return SourceImpactView{}, application.ErrInvalid
	}
	impact := SourceImpactView{ClauseIDs: []string{}, CandidateIDs: []string{}, MappingIDs: []string{}, QuestionIDs: []string{}, ScopeFactIDs: []string{}}
	if err := service.Pool.QueryRow(ctx, `SELECT id,source_identity,source_hash FROM regulatory_source_versions WHERE id=$1`, sourceID).Scan(&impact.SourceID, &impact.SourceIdentity, &impact.SourceHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SourceImpactView{}, ErrCandidateNotFound
		}
		return SourceImpactView{}, err
	}
	clauses, err := service.Pool.Query(ctx, `SELECT id FROM regulatory_normalized_clauses WHERE regulatory_source_version_id=$1 ORDER BY id`, sourceID)
	if err != nil {
		return SourceImpactView{}, err
	}
	for clauses.Next() {
		var id string
		if err := clauses.Scan(&id); err != nil {
			clauses.Close()
			return SourceImpactView{}, err
		}
		impact.ClauseIDs = append(impact.ClauseIDs, id)
	}
	if err := clauses.Err(); err != nil {
		clauses.Close()
		return SourceImpactView{}, err
	}
	clauses.Close()
	candidates, err := service.Pool.Query(ctx, `SELECT DISTINCT candidate.id FROM template_draft_versions candidate JOIN regulatory_generation_run_source_snapshots snapshot ON snapshot.generation_run_id=candidate.generation_run_id WHERE snapshot.regulatory_source_version_id=$1 ORDER BY candidate.id`, sourceID)
	if err != nil {
		return SourceImpactView{}, err
	}
	for candidates.Next() {
		var id string
		if err := candidates.Scan(&id); err != nil {
			candidates.Close()
			return SourceImpactView{}, err
		}
		impact.CandidateIDs = append(impact.CandidateIDs, id)
	}
	if err := candidates.Err(); err != nil {
		candidates.Close()
		return SourceImpactView{}, err
	}
	candidates.Close()
	mappingIDs, questionIDs := map[string]bool{}, map[string]bool{}
	for _, candidateID := range impact.CandidateIDs {
		candidate, err := service.getCandidate(ctx, service.Pool, candidateID)
		if err != nil {
			return SourceImpactView{}, err
		}
		for _, mapping := range candidate.Mappings {
			for _, citation := range mapping.Citations {
				if citation.SourceSnapshotID == sourceID && citation.SourceHash == impact.SourceHash {
					mappingIDs[mapping.MappingID] = true
				}
			}
		}
		for _, question := range candidate.Questions {
			for _, citation := range question.Citations {
				if citation.SourceSnapshotID == sourceID && citation.SourceHash == impact.SourceHash {
					questionIDs[question.QuestionID] = true
				}
			}
		}
	}
	for id := range mappingIDs {
		impact.MappingIDs = append(impact.MappingIDs, id)
	}
	for id := range questionIDs {
		impact.QuestionIDs = append(impact.QuestionIDs, id)
	}
	sort.Strings(impact.MappingIDs)
	sort.Strings(impact.QuestionIDs)
	rows, err := service.Pool.Query(ctx, `SELECT DISTINCT scope.organization_service_provider_scope_id FROM regulatory_generation_run_source_snapshots snapshot JOIN regulatory_generation_run_scope_facts scope ON scope.generation_run_id=snapshot.generation_run_id WHERE snapshot.regulatory_source_version_id=$1 ORDER BY scope.organization_service_provider_scope_id`, sourceID)
	if err != nil {
		return SourceImpactView{}, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return SourceImpactView{}, err
		}
		impact.ScopeFactIDs = append(impact.ScopeFactIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return SourceImpactView{}, err
	}
	rows.Close()
	return impact, nil
}

// EvaluateGenerationRunHoldout resolves both sides from the persisted,
// immutable evaluation partition graph. Callers cannot substitute a similarly
// named file, hash, or input row for a reserved blind-holdout identity.
func (service *AdminService) EvaluateGenerationRunHoldout(ctx context.Context, actor identity.Principal, generationRunID string, reviewed []HoldoutReview) (HoldoutEvaluationResult, error) {
	if err := service.requireAdmin(actor); err != nil {
		return HoldoutEvaluationResult{}, err
	}
	if strings.TrimSpace(generationRunID) == "" {
		return HoldoutEvaluationResult{}, application.ErrInvalid
	}
	inputRows, err := service.Pool.Query(ctx, `SELECT stable_row_identity FROM regulatory_generation_run_crosswalk_partition_rows WHERE generation_run_id=$1 ORDER BY stable_row_identity`, generationRunID)
	if err != nil {
		return HoldoutEvaluationResult{}, err
	}
	input := []string{}
	for inputRows.Next() {
		var id string
		if err := inputRows.Scan(&id); err != nil {
			inputRows.Close()
			return HoldoutEvaluationResult{}, err
		}
		input = append(input, id)
	}
	if err := inputRows.Err(); err != nil {
		inputRows.Close()
		return HoldoutEvaluationResult{}, err
	}
	inputRows.Close()
	if len(input) == 0 {
		return HoldoutEvaluationResult{}, application.ErrInvalid
	}
	holdoutRows, err := service.Pool.Query(ctx, `
		SELECT DISTINCT holdout_row.stable_row_identity
		FROM regulatory_generation_run_crosswalk_partition_rows used
		JOIN regulatory_evaluation_partitions input_partition ON input_partition.id=used.evaluation_partition_id AND input_partition.partition_kind='GENERATION_INPUT'
		JOIN regulatory_evaluation_partitions holdout_partition ON holdout_partition.evaluation_id=input_partition.evaluation_id AND holdout_partition.partition_kind='BLIND_HOLDOUT'
		JOIN regulatory_evaluation_partition_rows holdout_row ON holdout_row.partition_id=holdout_partition.id
		WHERE used.generation_run_id=$1 ORDER BY holdout_row.stable_row_identity`, generationRunID)
	if err != nil {
		return HoldoutEvaluationResult{}, err
	}
	holdout := []string{}
	for holdoutRows.Next() {
		var id string
		if err := holdoutRows.Scan(&id); err != nil {
			holdoutRows.Close()
			return HoldoutEvaluationResult{}, err
		}
		holdout = append(holdout, id)
	}
	if err := holdoutRows.Err(); err != nil {
		holdoutRows.Close()
		return HoldoutEvaluationResult{}, err
	}
	holdoutRows.Close()
	if len(holdout) == 0 {
		return HoldoutEvaluationResult{}, application.ErrInvalid
	}
	result, err := EvaluateBlindHoldout(HoldoutEvaluationInput{GenerationInputStableRowIDs: input, HoldoutStableRowIDs: holdout, Reviewed: reviewed})
	if err != nil {
		return HoldoutEvaluationResult{}, err
	}
	if err := service.recordHoldoutEvaluation(ctx, actor, generationRunID, reviewed, result); err != nil {
		return HoldoutEvaluationResult{}, err
	}
	return result, nil
}

// recordHoldoutEvaluation makes the reviewed identities, outcomes, and exact
// counts durable in the append-only Audit ledger. Its content-addressed event
// identity makes an identical retry a no-op while a changed evaluation remains
// a separately attributable review event.
func (service *AdminService) recordHoldoutEvaluation(ctx context.Context, actor identity.Principal, generationRunID string, reviewed []HoldoutReview, result HoldoutEvaluationResult) error {
	semantic, err := CanonicalSHA256(map[string]any{"generationRunId": generationRunID, "reviewed": reviewed, "result": result})
	if err != nil {
		return err
	}
	eventID := "AE-HOLDOUT-" + strings.TrimPrefix(semantic, "sha256:")[:24]
	details, err := json.Marshal(map[string]any{"semanticPayloadDigest": semantic, "generationRunId": generationRunID, "reviewed": reviewed, "counts": result})
	if err != nil {
		return err
	}
	return database.WithinTransaction(ctx, service.Pool, func(ctx context.Context, tx pgx.Tx) error {
		var storedSemantic string
		err := tx.QueryRow(ctx, `SELECT details->>'semanticPayloadDigest' FROM audit_events WHERE event_id=$1`, eventID).Scan(&storedSemantic)
		if err == nil {
			if storedSemantic == semantic {
				return nil
			}
			return application.ErrConflict
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO audit_events (event_id,occurred_at,actor_subject_id,actor_role,organization_id,action,entity_type,entity_id,entity_version,before_status,after_status,reason,operation_id,correlation_id,request_id,details) VALUES ($1,$2,$3,'admin',$4,'regulatory.blind_holdout_evaluated','REGULATORY_GENERATION_RUN',$5,1,NULL,'EVALUATED','Blind holdout evaluated against the persisted reserved partition.',$6,$6,$6,$7::jsonb)`, eventID, service.Clock().UTC(), actor.SubjectID, actor.OrganizationID, generationRunID, "HOLDOUT-"+strings.TrimPrefix(semantic, "sha256:")[:24], string(details))
		return err
	})
}

func (service *AdminService) Import(ctx context.Context, actor identity.Principal, operationID, idempotencyKey string, bundle CandidateBundle) (GenerationRunView, error) {
	if err := service.requireAdmin(actor); err != nil {
		return GenerationRunView{}, err
	}
	if strings.TrimSpace(operationID) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return GenerationRunView{}, application.ErrInvalid
	}
	semantic, err := importSemanticDigest(operationID, bundle)
	if err != nil {
		return GenerationRunView{}, err
	}
	var generationRunID string
	var failed bool
	var failureReason string
	err = database.WithinTransaction(ctx, service.Pool, func(ctx context.Context, tx pgx.Tx) error {
		var candidateID *string
		var storedOperationID, storedIdempotencyKey, storedSemantic, storedKind string
		replayErr := tx.QueryRow(ctx, `SELECT generation_run_id,candidate_draft_version_id,operation_id,idempotency_key,semantic_payload_digest,command_kind FROM governed_candidate_commands WHERE operation_id=$1 OR idempotency_key=$2`, operationID, idempotencyKey).Scan(&generationRunID, &candidateID, &storedOperationID, &storedIdempotencyKey, &storedSemantic, &storedKind)
		if replayErr == nil {
			if storedOperationID != operationID || storedIdempotencyKey != idempotencyKey || storedSemantic != semantic {
				return application.ErrConflict
			}
			if storedKind == "FAILED_IMPORT" {
				failed = true
				return nil
			}
			if candidateID == nil {
				return application.ErrConflict
			}
			return nil
		}
		if !errors.Is(replayErr, pgx.ErrNoRows) {
			return replayErr
		}

		// Import persistence writes a deliberately dense immutable graph. Keep the
		// whole attempt behind a savepoint so a semantic denial (for example an
		// unactivated source-currentness binding) cannot leave an orphaned run,
		// scope fact, source snapshot, or partition row behind before the durable
		// FAILED_IMPORT receipt is written in the outer transaction.
		attempt, beginErr := tx.Begin(ctx)
		if beginErr != nil {
			return fmt.Errorf("begin governed import savepoint: %w", beginErr)
		}
		result, importErr := (ImportStore{Pool: service.Pool}).ImportInTransaction(ctx, attempt, bundle)
		if importErr != nil {
			if rollbackErr := attempt.Rollback(ctx); rollbackErr != nil {
				return fmt.Errorf("rollback failed governed import savepoint after %v: %w", importErr, rollbackErr)
			}
			// Preserve a real storage error exactly. The savepoint has restored the
			// outer transaction, but a FAILED_IMPORT receipt would still obscure the
			// primary persistence failure and is therefore intentionally omitted.
			var databaseError *pgconn.PgError
			if errors.As(importErr, &databaseError) {
				return importErr
			}
			failed = true
			failureReason = importErr.Error()
			var persistErr error
			generationRunID, persistErr = service.recordFailedImport(ctx, tx, operationID, idempotencyKey, semantic, bundle, "CANDIDATE_VALIDATION_FAILED", failureReason, actor)
			if persistErr != nil {
				return fmt.Errorf("record failed governed import after primary import failure %v: %w", importErr, persistErr)
			}
			return persistErr
		}
		if err := attempt.Commit(ctx); err != nil {
			return fmt.Errorf("commit governed import savepoint: %w", err)
		}
		generationRunID = result.GenerationRunID
		candidate, candidateErr := service.getCandidate(ctx, tx, bundle.CandidateBundleID)
		if candidateErr != nil {
			return candidateErr
		}
		if err := service.recordSourceImpactCandidate(ctx, tx, bundle, candidate, actor); err != nil {
			return err
		}
		return persistGovernedCommand(ctx, tx, "IMPORTED_GENERATION_RUN", operationID, idempotencyKey, semantic, generationRunID, candidate.CandidateID, candidate.Revision, candidate.ContentDigest, actor, "Imported deterministic governed generation run.", "", GeneratedDraft, service.Clock())
	})
	if err != nil {
		return GenerationRunView{}, err
	}
	if failed {
		if failureReason == "" {
			failureReason = "identical failed import replay"
		}
		return GenerationRunView{}, fmt.Errorf("%w: import governed candidate: %s", application.ErrInvalid, failureReason)
	}
	return service.GetRun(ctx, actor, generationRunID)
}

// recordSourceImpactCandidate records only a candidate's binding to an
// already-activated source impact. Activation itself is deliberately separate
// and precedes import, so this method cannot infer a source change from dates
// or silently turn a supplied source row into a current authority state.
func (service *AdminService) recordSourceImpactCandidate(ctx context.Context, tx pgx.Tx, bundle CandidateBundle, candidate CandidateView, actor identity.Principal) error {
	if bundle.SourceCurrentness == nil || bundle.SourceCurrentness.PreviousSourceSnapshotID == "" {
		return nil
	}
	binding := bundle.SourceCurrentness
	var eventID, impactReviewDraftID, sourceIdentity string
	var previousSourceID, previousSourceHash, currentSourceID, currentSourceHash string
	err := tx.QueryRow(ctx, `
		SELECT binding.currentness_event_id,binding.impact_review_draft_id,binding.source_identity,
			binding.previous_source_version_id,binding.previous_source_hash,
			binding.current_source_version_id,binding.current_source_hash
		FROM regulatory_generation_run_source_currentness_bindings binding
		WHERE binding.generation_run_id=$1`, candidate.GenerationRunID,
	).Scan(&eventID, &impactReviewDraftID, &sourceIdentity, &previousSourceID, &previousSourceHash, &currentSourceID, &currentSourceHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: source-change candidate is missing its immutable activation binding", application.ErrInvalid)
	}
	if err != nil {
		return err
	}
	if previousSourceID != binding.PreviousSourceSnapshotID || previousSourceHash != binding.PreviousSourceHash ||
		currentSourceID != binding.CurrentSourceSnapshotID || currentSourceHash != binding.CurrentSourceHash ||
		impactReviewDraftID == "" {
		return fmt.Errorf("%w: source-change candidate binding does not match the activated predecessor/current pair", application.ErrInvalid)
	}
	details, err := json.Marshal(map[string]any{
		"candidateId":         candidate.CandidateID,
		"currentnessEventId":  eventID,
		"impactReviewDraftId": impactReviewDraftID,
		"sourceIdentity":      sourceIdentity,
		"previous":            map[string]string{"sourceId": previousSourceID, "sourceHash": previousSourceHash},
		"current":             map[string]string{"sourceId": currentSourceID, "sourceHash": currentSourceHash},
	})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events
			(event_id,occurred_at,actor_subject_id,actor_role,organization_id,action,entity_type,entity_id,entity_version,before_status,after_status,reason,operation_id,correlation_id,request_id,details)
		VALUES
			($1,$2,$3,'admin',$4,'regulatory.source_impact_candidate_bound','REGULATORY_SOURCE_IMPACT',$5,1,$6,$7,'Candidate was bound to an already activated immutable source impact-review Draft.',$8,$8,$8,$9::jsonb)
		ON CONFLICT (event_id) DO NOTHING`,
		"AE-SOURCE-IMPACT-"+candidate.CandidateID, service.Clock().UTC(), actor.SubjectID, actor.OrganizationID,
		candidate.CandidateID, previousSourceHash, currentSourceHash, "SOURCE-IMPACT-"+candidate.CandidateID, string(details),
	)
	return err
}

func (service *AdminService) GetRun(ctx context.Context, actor identity.Principal, id string) (GenerationRunView, error) {
	if err := service.requireAdmin(actor); err != nil {
		return GenerationRunView{}, err
	}
	view := GenerationRunView{GenerationRunID: id}
	var output, failureCode, failureReason *string
	var inputArtifact []byte
	if err := service.Pool.QueryRow(ctx, `SELECT status,input_digest,output_digest,input_schema_version,generation_policy_version,provider_catalog_version,provider_adapter_version,inspection_type,target_id,input_artifact::text,failure_code,failure_reason FROM regulatory_generation_runs WHERE id=$1`, id).Scan(&view.Status, &view.InputDigest, &output, &view.InputSchemaVersion, &view.GenerationPolicyVersion, &view.ProviderCatalogVersion, &view.ProviderAdapterVersion, &view.InspectionType, &view.TargetID, &inputArtifact, &failureCode, &failureReason); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return view, ErrCandidateNotFound
		}
		return view, err
	}
	view.OutputDigest = output
	var artifact map[string]json.RawMessage
	if err := json.Unmarshal(inputArtifact, &artifact); err != nil {
		return view, fmt.Errorf("decode persisted generation request identity: %w", err)
	}
	requestArtifact := inputArtifact
	if nested, ok := artifact["generationRequest"]; ok {
		requestArtifact = nested
	}
	var requestIdentity struct {
		RequestID  string `json:"requestId"`
		ProviderID string `json:"providerId"`
	}
	if err := json.Unmarshal(requestArtifact, &requestIdentity); err != nil {
		return view, fmt.Errorf("decode persisted request identity: %w", err)
	}
	view.RequestID = requestIdentity.RequestID
	view.ProviderID = requestIdentity.ProviderID
	if failureReason != nil || failureCode != nil {
		failure := &GenerationFailureView{RequestID: view.RequestID}
		if failureCode != nil {
			failure.Code = *failureCode
		}
		if failureReason != nil {
			failure.Reason = *failureReason
		}
		if raw, ok := artifact["importOperationId"]; ok {
			_ = json.Unmarshal(raw, &failure.OperationID)
		}
		if raw, ok := artifact["idempotencyKey"]; ok {
			_ = json.Unmarshal(raw, &failure.IdempotencyKey)
		}
		view.Failure = failure
	}
	var candidateID string
	if err := service.Pool.QueryRow(ctx, `SELECT id FROM template_draft_versions WHERE generation_run_id=$1 ORDER BY revision DESC LIMIT 1`, id).Scan(&candidateID); err == nil {
		candidate, err := service.GetCandidate(ctx, actor, candidateID)
		if err != nil {
			return view, err
		}
		view.Candidate = &candidate
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return view, err
	}
	return view, nil
}

func (service *AdminService) recordFailedImport(ctx context.Context, tx pgx.Tx, operationID, idempotencyKey, semantic string, bundle CandidateBundle, code, reason string, actor identity.Principal) (string, error) {
	input := map[string]any{
		"generationRequest": bundle.GenerationRequest,
		"candidateBundle":   bundle,
		"importOperationId": operationID,
		"idempotencyKey":    idempotencyKey,
	}
	failedInputDigest, err := CanonicalSHA256(input)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	id := "GENRUN-FAILED-" + strings.TrimPrefix(operationID, "TASK5-")
	nonblank := func(value, fallback string) string {
		if strings.TrimSpace(value) == "" {
			return fallback
		}
		return value
	}
	if _, err := tx.Exec(ctx, `INSERT INTO regulatory_generation_runs (id,status,input_digest,output_digest,input_schema_version,generation_policy_version,provider_catalog_version,provider_adapter_version,inspection_type,target_id,input_artifact,output_artifact,failure_code,failure_reason) VALUES ($1,'FAILED',$2,NULL,$3,$4,$5,$6,$7,'TARGET-SYNTHETIC-AOC',$8::jsonb,NULL,$9,$10)`, id, failedInputDigest, nonblank(bundle.GenerationRequest.SchemaVersion, "unknown"), nonblank(bundle.GenerationRequest.GenerationPolicyVersion, "unknown"), nonblank(bundle.GenerationRequest.ProviderCatalogVersion, "unknown"), nonblank(bundle.GenerationRequest.ProviderVersion, "unknown"), nonblank(bundle.GenerationRequest.InspectionType, "unknown"), string(encoded), code, reason); err != nil {
		return "", err
	}
	at := service.Clock()
	auditID := "AE-" + operationID
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events (event_id,occurred_at,actor_subject_id,actor_role,organization_id,action,entity_type,entity_id,entity_version,before_status,after_status,reason,operation_id,correlation_id,request_id,details) VALUES ($1,$2,$3,'admin',$4,'FAILED_IMPORT','GOVERNED_GENERATION_RUN',$5,0,NULL,'FAILED',$6,$7,$7,$8,'{}'::jsonb)`, auditID, at, actor.SubjectID, actor.OrganizationID, id, reason, operationID, bundle.GenerationRequest.RequestID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO governed_candidate_commands (id,command_kind,operation_id,idempotency_key,semantic_payload_digest,generation_run_id,candidate_draft_version_id,candidate_revision,candidate_content_digest,actor_subject_id,reason,audit_event_id,created_at) VALUES ($1,'FAILED_IMPORT',$2,$3,$4,$5,NULL,NULL,NULL,$6,$7,$8,$9)`, "CMD-"+operationID, operationID, idempotencyKey, semantic, id, actor.SubjectID, reason, auditID, at); err != nil {
		return "", err
	}
	return id, nil
}

func (service *AdminService) GetCandidate(ctx context.Context, actor identity.Principal, id string) (CandidateView, error) {
	if err := service.requireAdmin(actor); err != nil {
		return CandidateView{}, err
	}
	return service.getCandidate(ctx, service.Pool, id)
}

type dbQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (service *AdminService) getCandidate(ctx context.Context, query dbQuerier, id string) (CandidateView, error) {
	view := CandidateView{CandidateID: id, SourceSnapshots: []CandidateSourceLineageView{}, Mappings: []ComplianceMapping{}, Questions: []ChecklistQuestion{}, RequiredOwners: []RequiredOwner{}, ScopeFactIDs: []string{}, CrosswalkPartitionIDs: []string{}}
	if err := query.QueryRow(ctx, `SELECT candidate_root_id, supersedes_candidate_id, generation_run_id, template_id, version, revision, status, candidate_content_digest, candidate_schema_version, change_reason FROM template_draft_versions WHERE id=$1 AND generation_run_id IS NOT NULL`, id).Scan(&view.CandidateRootID, &view.SupersedesCandidateID, &view.GenerationRunID, &view.TemplateID, &view.Version, &view.Revision, &view.Status, &view.ContentDigest, &view.SchemaVersion, &view.ChangeReason); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return view, ErrCandidateNotFound
		}
		return view, err
	}
	if err := typedJSONRows(ctx, query, `SELECT jsonb_build_object('sourceId', source.id, 'sourceIdentity', source.source_identity, 'versionIdentity', source.version_identity, 'sourceHash', source.source_hash, 'clauseId', clause.id, 'locator', clause.clause_locator) FROM regulatory_generation_run_source_snapshots snapshot JOIN regulatory_source_versions source ON source.id=snapshot.regulatory_source_version_id JOIN regulatory_normalized_clauses clause ON clause.id=snapshot.regulatory_normalized_clause_id WHERE snapshot.generation_run_id=$1 ORDER BY source.id, clause.id`, []any{view.GenerationRunID}, &view.SourceSnapshots); err != nil {
		return view, err
	}
	if err := typedJSONRows(ctx, query, `SELECT snapshot FROM regulatory_generated_mapping_snapshots WHERE candidate_draft_version_id=$1 ORDER BY mapping_ordinal`, []any{id}, &view.Mappings); err != nil {
		return view, err
	}
	if err := typedJSONRows(ctx, query, `SELECT snapshot FROM regulatory_generated_question_snapshots WHERE candidate_draft_version_id=$1 ORDER BY question_id`, []any{id}, &view.Questions); err != nil {
		return view, err
	}
	if err := ProjectQuestionGovernanceCurrentness(ctx, query, view.Status, view.GenerationRunID, view.Questions); err != nil {
		return view, err
	}
	if err := typedJSONRows(ctx, query, `SELECT jsonb_build_object('departmentId', department_id, 'organizationalUnitId', organizational_unit_id, 'approvalRequired', approval_required) FROM candidate_required_owner_assignments WHERE candidate_draft_version_id=$1 AND candidate_revision=$2 AND candidate_content_digest=$3 ORDER BY department_id, organizational_unit_id`, []any{id, view.Revision, view.ContentDigest}, &view.RequiredOwners); err != nil {
		return view, err
	}
	if err := query.QueryRow(ctx, `SELECT COALESCE(array_agg(organization_service_provider_scope_id ORDER BY organization_service_provider_scope_id), '{}'::text[]) FROM regulatory_generation_run_scope_facts WHERE generation_run_id=$1`, view.GenerationRunID).Scan(&view.ScopeFactIDs); err != nil {
		return view, err
	}
	if err := query.QueryRow(ctx, `SELECT COALESCE(array_agg(DISTINCT evaluation_partition_id ORDER BY evaluation_partition_id), '{}'::text[]) FROM regulatory_generation_run_crosswalk_partition_rows WHERE generation_run_id=$1`, view.GenerationRunID).Scan(&view.CrosswalkPartitionIDs); err != nil {
		return view, err
	}
	return view, nil
}
func typedJSONRows[T any](ctx context.Context, query dbQuerier, statement string, arguments []any, target *[]T) error {
	rows, err := query.Query(ctx, statement, arguments...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		var item T
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		*target = append(*target, item)
	}
	return rows.Err()
}

func validateEditCommand(parent CandidateView, command EditCommand) []ValidationIssue {
	issue := func(fieldPath, code, message string) []ValidationIssue {
		value := ValidationIssue{FieldPath: fieldPath, Code: code, Message: message}
		if len(parent.SourceSnapshots) > 0 {
			source := parent.SourceSnapshots[0]
			value.SourceIdentity = source.SourceIdentity
			value.SourceHash = source.SourceHash
			value.ClauseID = source.ClauseID
			value.Locator = source.Locator
		}
		return []ValidationIssue{value}
	}
	if command.ExpectedContentDigest != parent.ContentDigest {
		return issue("expectedContentDigest", "DIGEST_CONFLICT", "expected candidate content digest does not match the immutable parent")
	}
	if len(command.Mappings) != len(parent.Mappings) {
		index := len(command.Mappings) - 1
		if index < 0 {
			index = 0
		}
		return issue(fmt.Sprintf("mappings[%d].mappingId", index), "MAPPING_IDENTITY_MISMATCH", "the edit must preserve the complete mapping identity set")
	}
	seenMappings := map[string]bool{}
	for index, mapping := range command.Mappings {
		prefix := fmt.Sprintf("mappings[%d]", index)
		if strings.TrimSpace(mapping.MappingID) == "" || seenMappings[mapping.MappingID] {
			return issue(prefix+".mappingId", "DUPLICATE_OR_BLANK_MAPPING_ID", "mapping IDs must be nonblank and unique")
		}
		seenMappings[mapping.MappingID] = true
		if index >= len(parent.Mappings) || mapping.MappingID != parent.Mappings[index].MappingID {
			return issue(prefix+".mappingId", "MAPPING_IDENTITY_MISMATCH", "mapping identity is immutable")
		}
		if mapping.Requirement != syntheticSupportedRequirement {
			return issue(prefix+".requirement", "UNSUPPORTED_CLAIM", "requirement is outside the controlled synthetic registry")
		}
		if mapping.Relationship != "ADDRESSES" {
			return issue(prefix+".relationship", "RELATIONSHIP_MISMATCH", "relationship must preserve the exact supported mapping")
		}
		if mapping.Applicability != "DIRECT" {
			return issue(prefix+".applicability", "APPLICABILITY_MISMATCH", "applicability must preserve the exact supported mapping")
		}
		if mapping.SourceGap != nil {
			return issue(prefix+".sourceGap", "SOURCE_GAP_MISMATCH", "source gaps may not be inferred or fabricated")
		}
		if mapping.Rationale != syntheticSupportedRationale && mapping.Rationale != syntheticEditedRationale {
			return issue(prefix+".rationale", "UNSUPPORTED_CLAIM", "rationale is outside the controlled synthetic registry")
		}
		if len(mapping.Citations) != 1 {
			return issue(prefix+".citations", "CITATION_MISMATCH", "one exact persisted citation is required")
		}
		citation := mapping.Citations[0]
		if len(parent.SourceSnapshots) == 0 {
			return issue(prefix+".citations[0]", "LINEAGE_MISSING", "persisted source lineage is missing")
		}
		source := parent.SourceSnapshots[0]
		switch {
		case citation.SourceSnapshotID != source.SourceID:
			return issue(prefix+".citations[0].sourceSnapshotId", "SOURCE_IDENTITY_MISMATCH", "citation source identity is immutable")
		case citation.SourceHash != source.SourceHash:
			return issue(prefix+".citations[0].sourceHash", "SOURCE_HASH_MISMATCH", "citation source hash is immutable")
		case citation.ClauseID != source.ClauseID:
			return issue(prefix+".citations[0].clauseId", "CLAUSE_IDENTITY_MISMATCH", "citation clause identity is immutable")
		case citation.Locator != source.Locator:
			return issue(prefix+".citations[0].locator", "LOCATOR_MISMATCH", "citation locator is immutable")
		}
	}
	if len(command.Questions) != len(parent.Questions) {
		index := len(command.Questions) - 1
		if index < 0 {
			index = 0
		}
		return issue(fmt.Sprintf("questions[%d].questionId", index), "QUESTION_IDENTITY_MISMATCH", "the edit must preserve the complete question identity set")
	}
	seenQuestions := map[string]bool{}
	for index, question := range command.Questions {
		prefix := fmt.Sprintf("questions[%d]", index)
		if strings.TrimSpace(question.QuestionID) == "" || seenQuestions[question.QuestionID] {
			return issue(prefix+".questionId", "DUPLICATE_OR_BLANK_QUESTION_ID", "question IDs must be nonblank and unique")
		}
		seenQuestions[question.QuestionID] = true
		if index >= len(parent.Questions) || question.QuestionID != parent.Questions[index].QuestionID {
			return issue(prefix+".questionId", "QUESTION_IDENTITY_MISMATCH", "question identity is immutable")
		}
		if !sameStrings(question.MappingIDs, []string{parent.Mappings[0].MappingID}) {
			return issue(prefix+".mappingIds[0]", "MAPPING_REFERENCE_MISMATCH", "question mapping references must resolve to the exact preserved mapping")
		}
		if question.Prompt != syntheticSupportedQuestion {
			return issue(prefix+".prompt", "UNSUPPORTED_CLAIM", "question text is outside the controlled synthetic registry")
		}
		if question.VerificationMethod != syntheticSupportedVerificationMethod {
			return issue(prefix+".verificationMethod", "UNSUPPORTED_CLAIM", "verification method is outside the controlled synthetic registry")
		}
		if len(question.ExpectedEvidence) == 0 || hasBlank(question.ExpectedEvidence) {
			return issue(prefix+".expectedEvidence[0]", "BLANK_EVIDENCE", "expected Evidence entries must be nonblank")
		}
		if !sameStrings(question.ExpectedEvidence, syntheticSupportedExpectedEvidence) {
			return issue(prefix+".expectedEvidence", "UNSUPPORTED_CLAIM", "expected Evidence is outside the controlled synthetic registry")
		}
		if !sameStrings(question.AllowedAnswers, allowedAnswers) {
			return issue(prefix+".allowedAnswers", "INVALID_ALLOWED_ANSWERS", "allowed answers must preserve the exact governed set")
		}
		if !question.MandatoryCore {
			return issue(prefix+".mandatoryCore", "MANDATORY_FLAG_MISMATCH", "mandatory-core classification is immutable")
		}
		if !question.SafetyCritical {
			return issue(prefix+".safetyCritical", "SAFETY_FLAG_MISMATCH", "safety-critical classification is immutable")
		}
		if len(question.Citations) != 1 {
			return issue(prefix+".citations", "CITATION_MISMATCH", "one exact persisted citation is required")
		}
		source := parent.SourceSnapshots[0]
		citation := question.Citations[0]
		switch {
		case citation.SourceSnapshotID != source.SourceID:
			return issue(prefix+".citations[0].sourceSnapshotId", "SOURCE_IDENTITY_MISMATCH", "citation source identity is immutable")
		case citation.SourceHash != source.SourceHash:
			return issue(prefix+".citations[0].sourceHash", "SOURCE_HASH_MISMATCH", "citation source hash is immutable")
		case citation.ClauseID != source.ClauseID:
			return issue(prefix+".citations[0].clauseId", "CLAUSE_IDENTITY_MISMATCH", "citation clause identity is immutable")
		case citation.Locator != source.Locator:
			return issue(prefix+".citations[0].locator", "LOCATOR_MISMATCH", "citation locator is immutable")
		}
	}
	if len(command.RequiredOwners) != len(parent.RequiredOwners) {
		return issue("requiredOwners", "OWNER_SET_MISMATCH", "the complete required-owner set is immutable")
	}
	for index, owner := range command.RequiredOwners {
		expected := parent.RequiredOwners[index]
		switch {
		case owner.DepartmentID != expected.DepartmentID:
			return issue(fmt.Sprintf("requiredOwners[%d].departmentId", index), "UNKNOWN_OWNER", "required owner department is unknown or changed")
		case owner.OrganizationalUnitID != expected.OrganizationalUnitID:
			return issue(fmt.Sprintf("requiredOwners[%d].organizationalUnitId", index), "UNKNOWN_OWNER", "required owner organizational unit is unknown or changed")
		case owner.ApprovalRequired != expected.ApprovalRequired:
			return issue(fmt.Sprintf("requiredOwners[%d].approvalRequired", index), "OWNER_APPROVAL_MISMATCH", "required owner approval classification is immutable")
		}
	}
	return nil
}

func (service *AdminService) CreateRevision(ctx context.Context, actor identity.Principal, command EditCommand) (CandidateView, error) {
	if err := service.requireAdmin(actor); err != nil {
		return CandidateView{}, err
	}
	if strings.TrimSpace(command.OperationID) == "" || strings.TrimSpace(command.IdempotencyKey) == "" || strings.TrimSpace(command.CandidateID) == "" || strings.TrimSpace(command.ExpectedContentDigest) == "" || strings.TrimSpace(command.ChangeReason) == "" || command.ExpectedRevision < 1 || len(command.Mappings) == 0 || len(command.Questions) == 0 || len(command.RequiredOwners) == 0 {
		return CandidateView{}, application.ErrInvalid
	}
	semantic, err := editSemanticDigest(command)
	if err != nil {
		return CandidateView{}, err
	}
	var output CandidateView
	err = database.WithinTransaction(ctx, service.Pool, func(ctx context.Context, tx pgx.Tx) error {
		if replay, ok, err := replayCommand(ctx, tx, command.OperationID, command.IdempotencyKey, semantic); err != nil {
			return err
		} else if ok {
			result, err := service.getCandidate(ctx, tx, replay)
			output = result
			return err
		}
		parent, err := service.getCandidate(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		if err := lockCandidateRoot(ctx, tx, parent.CandidateRootID); err != nil {
			return err
		}
		if replay, ok, err := replayCommand(ctx, tx, command.OperationID, command.IdempotencyKey, semantic); err != nil {
			return err
		} else if ok {
			result, err := service.getCandidate(ctx, tx, replay)
			output = result
			return err
		}
		parent, err = service.getCandidate(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		if (parent.Status != GeneratedDraft && parent.Status != "RETURNED") || parent.Revision != command.ExpectedRevision {
			return application.ErrConflict
		}
		if issues := validateEditCommand(parent, command); len(issues) > 0 {
			return &ValidationError{Issues: issues}
		}
		leafID, leafRevision, leafDigest, err := exactCurrentLeaf(ctx, tx, parent.CandidateRootID)
		if err != nil {
			return err
		}
		if leafID != command.CandidateID || leafRevision != command.ExpectedRevision || leafDigest != command.ExpectedContentDigest {
			return application.ErrConflict
		}
		digest, err := CanonicalSHA256(map[string]any{"complianceMappings": command.Mappings, "inspectionChecklist": map[string]any{"checklistId": parent.TemplateID, "questions": command.Questions}})
		if err != nil {
			return err
		}
		var requestArtifact []byte
		var inputDigest string
		if err := tx.QueryRow(ctx, `SELECT input_artifact::text, input_digest FROM regulatory_generation_runs WHERE id=$1`, parent.GenerationRunID).Scan(&requestArtifact, &inputDigest); err != nil {
			return err
		}
		var lineageRequest GenerationRequest
		if err := json.Unmarshal(requestArtifact, &lineageRequest); err != nil {
			return fmt.Errorf("%w: persisted generation request is invalid", application.ErrInvalid)
		}
		lineageRequest.CanonicalInputDigest = inputDigest
		validatedRequest, err := ValidateRequest(lineageRequest, true)
		if err != nil {
			return fmt.Errorf("%w: immutable source/scope/provider lineage mismatch", application.ErrInvalid)
		}
		currentnessBinding, err := loadVerifiedGenerationRunSourceCurrentnessBinding(ctx, tx, parent.GenerationRunID, parent.CandidateRootID)
		if err != nil {
			return fmt.Errorf("%w: immutable generation source-currentness binding is invalid: %v", application.ErrInvalid, err)
		}
		candidateForValidation := CandidateBundle{SchemaVersion: parent.SchemaVersion, CandidateBundleID: parent.CandidateID, GenerationRunID: parent.GenerationRunID, Status: GeneratedDraft, GenerationRequest: lineageRequest, InputDigest: inputDigest, OutputDigest: digest, ComplianceMappings: command.Mappings, InspectionChecklist: InspectionChecklist{ChecklistID: parent.TemplateID, Questions: command.Questions}, SourceCurrentness: currentnessBinding}
		importRegistryCandidate := candidateForValidation
		importRegistryCandidate.ComplianceMappings = append([]ComplianceMapping(nil), candidateForValidation.ComplianceMappings...)
		for index := range importRegistryCandidate.ComplianceMappings {
			if importRegistryCandidate.ComplianceMappings[index].Rationale == syntheticEditedRationale {
				importRegistryCandidate.ComplianceMappings[index].Rationale = syntheticSupportedRationale
			}
		}
		importRegistryCandidate.OutputDigest, _ = candidateOutputDigest(importRegistryCandidate)
		if err := ValidateCandidateBundle(importRegistryCandidate, validatedRequest); err != nil {
			return fmt.Errorf("%w: complete candidate validation failed", application.ErrInvalid)
		}
		root, err := service.getCandidate(ctx, tx, parent.CandidateRootID)
		if err != nil {
			return err
		}
		rootLineage := candidateForValidation
		rootLineage.CandidateBundleID = root.CandidateID
		rootLineage.OutputDigest = root.ContentDigest
		if err := verifyReplayLineageValues(ctx, tx, rootLineage, parent.GenerationRunID, false); err != nil {
			return fmt.Errorf("%w: persisted source/scope/target/partition lineage mismatch: %v", application.ErrInvalid, err)
		}
		candidateID := "CAND-EDIT-" + strings.TrimPrefix(semantic, "sha256:")[:20]
		questionIDs := make([]string, 0, len(command.Questions))
		for index, question := range command.Questions {
			questionID := fmt.Sprintf("QV-%s-%02d", candidateID, index+1)
			questionIDs = append(questionIDs, questionID)
			if _, err := tx.Exec(ctx, `INSERT INTO question_versions (id, question_id, version, prompt, configured_reference, expected_evidence, created_by_subject_id) VALUES ($1,$2,$3,$4,$5,$6,$7)`, questionID, question.QuestionID, parent.Version+1, question.Prompt, strings.Join(question.MappingIDs, ","), strings.Join(question.ExpectedEvidence, "; "), actor.SubjectID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id, supersedes_candidate_id) VALUES ($1,$2,$3,'GENERATED_DRAFT','Admin Preview',$4,$5,$6,$7,$8,$9,$10,$11,$12)`, candidateID, parent.TemplateID, parent.Version+1, actor.SubjectID, command.ChangeReason, questionIDs, parent.Revision+1, parent.GenerationRunID, digest, parent.SchemaVersion, parent.CandidateRootID, parent.CandidateID); err != nil {
			return fmt.Errorf("persist immutable candidate revision: %w", err)
		}
		for index, mapping := range command.Mappings {
			raw, _ := json.Marshal(mapping)
			if _, err := tx.Exec(ctx, `INSERT INTO regulatory_generated_mapping_snapshots (candidate_draft_version_id,mapping_id,mapping_ordinal,snapshot) VALUES ($1,$2,$3,$4::jsonb)`, candidateID, mapping.MappingID, index, string(raw)); err != nil {
				return err
			}
		}
		for _, question := range command.Questions {
			raw, _ := json.Marshal(question)
			if _, err := tx.Exec(ctx, `INSERT INTO regulatory_generated_question_snapshots (candidate_draft_version_id,question_id,snapshot) VALUES ($1,$2,$3::jsonb)`, candidateID, question.QuestionID, string(raw)); err != nil {
				return err
			}
		}
		for index, owner := range command.RequiredOwners {
			if _, err := tx.Exec(ctx, `INSERT INTO candidate_required_owner_assignments (id,candidate_draft_version_id,candidate_revision,candidate_content_digest,department_id,organizational_unit_id,approval_required) VALUES ($1,$2,$3,$4,$5,$6,$7)`, fmt.Sprintf("OWNER-%s-%02d", candidateID, index+1), candidateID, parent.Revision+1, digest, owner.DepartmentID, owner.OrganizationalUnitID, owner.ApprovalRequired); err != nil {
				return err
			}
		}
		if err := persistGovernedCommand(ctx, tx, "REVISION_CREATED", command.OperationID, command.IdempotencyKey, semantic, parent.GenerationRunID, candidateID, parent.Revision+1, digest, actor, command.ChangeReason, parent.Status, GeneratedDraft, service.Clock()); err != nil {
			return err
		}
		result, err := service.getCandidate(ctx, tx, candidateID)
		output = result
		return err
	})
	return output, err
}

func (service *AdminService) Submit(ctx context.Context, actor identity.Principal, command SubmitCommand) (CandidateView, error) {
	if err := service.requireAdmin(actor); err != nil {
		return CandidateView{}, err
	}
	if strings.TrimSpace(command.OperationID) == "" || strings.TrimSpace(command.IdempotencyKey) == "" || strings.TrimSpace(command.CandidateID) == "" || strings.TrimSpace(command.ExpectedContentDigest) == "" || strings.TrimSpace(command.Reason) == "" || command.ExpectedRevision < 1 {
		return CandidateView{}, application.ErrInvalid
	}
	semantic, err := submitSemanticDigest(command)
	if err != nil {
		return CandidateView{}, err
	}
	var output CandidateView
	err = database.WithinTransaction(ctx, service.Pool, func(ctx context.Context, tx pgx.Tx) error {
		if replay, ok, err := replayCommand(ctx, tx, command.OperationID, command.IdempotencyKey, semantic); err != nil {
			return err
		} else if ok {
			result, err := service.getCandidate(ctx, tx, replay)
			output = result
			return err
		}
		candidate, err := service.getCandidate(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		if err := lockCandidateRoot(ctx, tx, candidate.CandidateRootID); err != nil {
			return err
		}
		if replay, ok, err := replayCommand(ctx, tx, command.OperationID, command.IdempotencyKey, semantic); err != nil {
			return err
		} else if ok {
			result, err := service.getCandidate(ctx, tx, replay)
			output = result
			return err
		}
		candidate, err = service.getCandidate(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		if candidate.Status != GeneratedDraft || candidate.Revision != command.ExpectedRevision || candidate.ContentDigest != command.ExpectedContentDigest {
			return application.ErrConflict
		}
		leafID, leafRevision, leafDigest, err := exactCurrentLeaf(ctx, tx, candidate.CandidateRootID)
		if err != nil {
			return err
		}
		if leafID != command.CandidateID || leafRevision != command.ExpectedRevision || leafDigest != command.ExpectedContentDigest {
			return application.ErrConflict
		}
		if err := LockGenerationRunSourceCurrentness(ctx, tx, []string{candidate.GenerationRunID}); err != nil {
			return err
		}
		candidate, err = service.getCandidate(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		if candidate.Status != GeneratedDraft || candidate.Revision != command.ExpectedRevision || candidate.ContentDigest != command.ExpectedContentDigest {
			return application.ErrConflict
		}
		blockers, err := LoadCandidateBlockingIssues(ctx, tx, candidate)
		if err != nil {
			return err
		}
		if len(blockers) > 0 {
			return &ValidationError{Issues: blockers}
		}
		if tag, err := tx.Exec(ctx, `UPDATE template_draft_versions SET status='DEPARTMENT_REVIEW' WHERE id=$1 AND revision=$2 AND status='GENERATED_DRAFT'`, candidate.CandidateID, command.ExpectedRevision); err != nil {
			return err
		} else if tag.RowsAffected() != 1 {
			return application.ErrConflict
		}
		if err := persistGovernedCommand(ctx, tx, "DEPARTMENT_REVIEW_SUBMITTED", command.OperationID, command.IdempotencyKey, semantic, candidate.GenerationRunID, candidate.CandidateID, candidate.Revision, candidate.ContentDigest, actor, command.Reason, GeneratedDraft, "DEPARTMENT_REVIEW", service.Clock()); err != nil {
			return err
		}
		result, err := service.getCandidate(ctx, tx, candidate.CandidateID)
		output = result
		return err
	})
	return output, err
}

func exactCurrentLeaf(ctx context.Context, query dbQuerier, candidateRootID string) (string, int64, string, error) {
	rows, err := query.Query(ctx, `SELECT candidate.id,candidate.revision,candidate.candidate_content_digest FROM template_draft_versions candidate WHERE candidate.candidate_root_id=$1 AND NOT EXISTS (SELECT 1 FROM template_draft_versions successor WHERE successor.supersedes_candidate_id=candidate.id) ORDER BY candidate.revision DESC,candidate.id`, candidateRootID)
	if err != nil {
		return "", 0, "", err
	}
	defer rows.Close()
	var id, digest string
	var revision int64
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", 0, "", err
		}
		return "", 0, "", application.ErrConflict
	}
	if err := rows.Scan(&id, &revision, &digest); err != nil {
		return "", 0, "", err
	}
	if rows.Next() {
		return "", 0, "", application.ErrConflict
	}
	if err := rows.Err(); err != nil {
		return "", 0, "", err
	}
	return id, revision, digest, nil
}

func lockCandidateRoot(ctx context.Context, tx pgx.Tx, candidateRootID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, candidateRootID)
	return err
}

func replayCommand(ctx context.Context, tx pgx.Tx, operationID, idempotencyKey, semantic string) (string, bool, error) {
	var candidateID *string
	var storedOperationID, storedIdempotencyKey, stored string
	err := tx.QueryRow(ctx, `SELECT candidate_draft_version_id,operation_id,idempotency_key,semantic_payload_digest FROM governed_candidate_commands WHERE operation_id=$1 OR idempotency_key=$2`, operationID, idempotencyKey).Scan(&candidateID, &storedOperationID, &storedIdempotencyKey, &stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if storedOperationID != operationID || storedIdempotencyKey != idempotencyKey || stored != semantic {
		return "", false, application.ErrConflict
	}
	if candidateID == nil {
		return "", false, application.ErrConflict
	}
	return *candidateID, true, nil
}
func persistGovernedCommand(ctx context.Context, tx pgx.Tx, kind, operationID, idempotencyKey, semantic, generationRunID, candidateID string, revision int64, digest string, actor identity.Principal, reason, beforeStatus, afterStatus string, at time.Time) error {
	auditID := "AE-" + operationID
	var before, after any
	if beforeStatus != "" {
		before = beforeStatus
	}
	if afterStatus != "" {
		after = afterStatus
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events (event_id,occurred_at,actor_subject_id,actor_role,organization_id,action,entity_type,entity_id,entity_version,before_status,after_status,reason,operation_id,correlation_id,request_id,details) VALUES ($1,$2,$3,'admin',$4,$5,'GOVERNED_CANDIDATE',$6,$7,$8,$9,$10,$11,$11,$11,'{}'::jsonb)`, auditID, at, actor.SubjectID, actor.OrganizationID, kind, candidateID, revision, before, after, reason, operationID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO governed_candidate_commands (id,command_kind,operation_id,idempotency_key,semantic_payload_digest,generation_run_id,candidate_draft_version_id,candidate_revision,candidate_content_digest,actor_subject_id,reason,audit_event_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, "CMD-"+operationID, kind, operationID, idempotencyKey, semantic, generationRunID, candidateID, revision, digest, actor.SubjectID, reason, auditID, at)
	return err
}
