package regulatory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// LoadCandidateBlockingIssues recomputes review blockers from persisted
// generation lineage. It is shared by Admin submission and Department Manager
// approval/publication so no transport can bypass the same fail-closed facts.
func LoadCandidateBlockingIssues(
	ctx context.Context,
	query interface {
		Query(context.Context, string, ...any) (pgx.Rows, error)
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	candidate CandidateView,
) ([]ValidationIssue, error) {
	issues := []ValidationIssue{}
	rows, err := query.Query(ctx, `
		SELECT gap.reason,source.source_identity,source.source_hash,
		       snapshot.regulatory_normalized_clause_id,snapshot.clause_locator
		FROM regulatory_source_gap_facts gap
		JOIN regulatory_source_versions source ON source.id=gap.regulatory_source_version_id
		JOIN regulatory_generation_run_source_snapshots snapshot
		  ON snapshot.generation_run_id=$1
		 AND snapshot.regulatory_source_version_id=gap.regulatory_source_version_id
		ORDER BY gap.ordinal,gap.gap_id`, candidate.GenerationRunID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var issue ValidationIssue
		if err := rows.Scan(&issue.Message, &issue.SourceIdentity, &issue.SourceHash, &issue.ClauseID, &issue.Locator); err != nil {
			rows.Close()
			return nil, err
		}
		issue.FieldPath = "sourceSnapshots"
		issue.Code = "UNRESOLVED_SOURCE_GAP"
		issues = append(issues, issue)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	var sourceIdentity, sourceHash, clauseID, locator string
	_ = query.QueryRow(ctx, `
		SELECT source.source_identity,source.source_hash,
		       snapshot.regulatory_normalized_clause_id,snapshot.clause_locator
		FROM regulatory_generation_run_source_snapshots snapshot
		JOIN regulatory_source_versions source ON source.id=snapshot.regulatory_source_version_id
		WHERE snapshot.generation_run_id=$1
		ORDER BY snapshot.regulatory_source_version_id,snapshot.regulatory_normalized_clause_id
		LIMIT 1`, candidate.GenerationRunID).
		Scan(&sourceIdentity, &sourceHash, &clauseID, &locator)
	appendCountIssue := func(count int, fieldPath, code, message string) {
		if count > 0 {
			issues = append(issues, ValidationIssue{
				FieldPath: fieldPath, Code: code, Message: message,
				SourceIdentity: sourceIdentity, SourceHash: sourceHash,
				ClauseID: clauseID, Locator: locator,
			})
		}
	}
	questionIssue := func(question ChecklistQuestion, fieldPath, code, message string) {
		issueSourceIdentity, issueSourceHash, issueClauseID, issueLocator := sourceIdentity, sourceHash, clauseID, locator
		if question.RegulatoryTrace.State == "RESOLVED" {
			if question.RegulatoryTrace.SourceIdentity != "" {
				issueSourceIdentity = question.RegulatoryTrace.SourceIdentity
			}
			if question.RegulatoryTrace.SHA256 != "" {
				issueSourceHash = question.RegulatoryTrace.SHA256
			}
			if question.RegulatoryTrace.Clause != "" {
				issueClauseID = question.RegulatoryTrace.Clause
			}
			if question.RegulatoryTrace.Locator != "" {
				issueLocator = question.RegulatoryTrace.Locator
			}
		}
		issues = append(issues, ValidationIssue{
			FieldPath:      fieldPath,
			Code:           code,
			Message:        message,
			SourceIdentity: issueSourceIdentity,
			SourceHash:     issueSourceHash,
			ClauseID:       issueClauseID,
			Locator:        issueLocator,
		})
	}
	questionRows, err := query.Query(ctx, `
		SELECT snapshot
		FROM regulatory_generated_question_snapshots
		WHERE candidate_draft_version_id=$1
		ORDER BY question_id`, candidate.CandidateID)
	if err != nil {
		return nil, err
	}
	questions := []ChecklistQuestion{}
	for questionRows.Next() {
		var raw []byte
		if err := questionRows.Scan(&raw); err != nil {
			questionRows.Close()
			return nil, err
		}
		var question ChecklistQuestion
		if err := json.Unmarshal(raw, &question); err != nil {
			questionRows.Close()
			return nil, fmt.Errorf("decode persisted governed question snapshot: %w", err)
		}
		questions = append(questions, question)
	}
	if err := questionRows.Err(); err != nil {
		questionRows.Close()
		return nil, err
	}
	questionRows.Close()
	for _, question := range questions {
		appendQuestionGovernanceIssues(ctx, query, question, questionIssue)
		if question.RegulatoryTrace.State == "RESOLVED" {
			bound, err := traceHasActivatedCurrentnessBinding(ctx, query, candidate.GenerationRunID, question)
			if err != nil {
				questionIssue(question, "questions["+question.QuestionID+"].regulatoryTrace.currentnessState", "SOURCE_CURRENTNESS_REQUIRED", "source-currentness activation binding could not be verified")
			} else if !bound {
				questionIssue(question, "questions["+question.QuestionID+"].regulatoryTrace.currentnessState", "SOURCE_CURRENTNESS_REQUIRED", "a traced candidate requires an immutable source-currentness activation binding before review or publication")
			}
		}
	}

	var invalidSources int
	if err := query.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM regulatory_generation_run_source_snapshots snapshot
		LEFT JOIN regulatory_source_versions source ON source.id=snapshot.regulatory_source_version_id
		LEFT JOIN regulatory_normalized_clauses clause ON clause.id=snapshot.regulatory_normalized_clause_id
		WHERE snapshot.generation_run_id=$1
		  AND (source.id IS NULL OR clause.id IS NULL
		       OR source.source_hash<>snapshot.source_hash
		       OR clause.regulatory_source_version_id<>snapshot.regulatory_source_version_id
		       OR clause.source_hash<>snapshot.source_hash
		       OR clause.clause_locator<>snapshot.clause_locator)`,
		candidate.GenerationRunID).Scan(&invalidSources); err != nil {
		return nil, err
	}
	appendCountIssue(invalidSources, "sourceSnapshots", "FROZEN_SOURCE_LINEAGE_MISMATCH", "persisted source identity, hash, clause, or locator no longer matches the frozen candidate lineage")

	var invalidPartitions int
	if err := query.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM regulatory_generation_run_crosswalk_partition_rows used
		LEFT JOIN regulatory_evaluation_partitions partition ON partition.id=used.evaluation_partition_id
		LEFT JOIN regulatory_evaluation_partition_rows source_row
		  ON source_row.partition_id=used.evaluation_partition_id
		 AND source_row.state_compliance_crosswalk_row_id=used.state_compliance_crosswalk_row_id
		 AND source_row.stable_row_identity=used.stable_row_identity
		WHERE used.generation_run_id=$1
		  AND (partition.id IS NULL OR partition.partition_kind<>'GENERATION_INPUT'
		       OR source_row.state_compliance_crosswalk_row_id IS NULL)`,
		candidate.GenerationRunID).Scan(&invalidPartitions); err != nil {
		return nil, err
	}
	appendCountIssue(invalidPartitions, "crosswalkPartitionIds", "HOLDOUT_OR_CROSSWALK_LINEAGE_MISMATCH", "candidate lineage contains a holdout or changed crosswalk partition row")

	var invalidScopes int
	if err := query.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM regulatory_generation_run_scope_facts frozen
		JOIN regulatory_generation_runs run ON run.id=frozen.generation_run_id
		LEFT JOIN organization_service_provider_scopes scope
		  ON scope.id=frozen.organization_service_provider_scope_id
		WHERE frozen.generation_run_id=$1
		  AND (scope.id IS NULL OR scope.root_id<>frozen.scope_root_id
		       OR scope.organization_id<>frozen.organization_id
		       OR scope.service_provider_type_id<>frozen.service_provider_type_id
		       OR scope.authorization_identifier<>frozen.authorization_identifier
		       OR scope.status<>frozen.scope_status
		       OR scope.effective_from<>frozen.effective_from
		       OR scope.effective_to IS DISTINCT FROM frozen.effective_to
		       OR run.target_id<>frozen.regulated_target_id
		       OR EXISTS (SELECT 1 FROM organization_service_provider_scopes successor
		                  WHERE successor.supersedes_id=scope.id))`,
		candidate.GenerationRunID).Scan(&invalidScopes); err != nil {
		return nil, err
	}
	appendCountIssue(invalidScopes, "scopeFactIds", "FROZEN_SCOPE_MISMATCH", "candidate target, provider, or organization scope no longer matches its frozen lineage")

	var requiredOwners int
	if err := query.QueryRow(ctx, `
		SELECT COUNT(*) FROM candidate_required_owner_assignments
		WHERE candidate_draft_version_id=$1 AND candidate_revision=$2
		  AND candidate_content_digest=$3 AND approval_required`,
		candidate.CandidateID, candidate.Revision, candidate.ContentDigest).Scan(&requiredOwners); err != nil {
		return nil, err
	}
	if requiredOwners == 0 {
		appendCountIssue(1, "requiredOwners", "REVIEW_REQUIRED_OWNER", "at least one exact mandatory department and organizational-unit owner must be resolved")
	}
	return issues, nil
}

func appendQuestionGovernanceIssues(
	ctx context.Context,
	query interface {
		Query(context.Context, string, ...any) (pgx.Rows, error)
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	question ChecklistQuestion,
	appendIssue func(ChecklistQuestion, string, string, string),
) {
	prefix := "questions[" + question.QuestionID + "]"
	if !oneOf(string(question.Origin), string(RegulatoryTraceOrigin), string(ExistingChecklistCandidateOrigin), string(HybridReconciledOrigin)) {
		appendIssue(question, prefix+".origin", "QUESTION_ORIGIN_REQUIRED", "every generated or published question must name one exact origin")
	}
	scope := question.ScopeRecommendation
	if !oneOf(scope.Classification, "MANDATORY_CORE", "FOCUSED_FULL", "ROTATIONAL_SAMPLE", "DEFER_ELIGIBLE") {
		appendIssue(question, prefix+".scopeRecommendation.classification", "SCOPE_CLASSIFICATION_REQUIRED", "scope recommendation requires a visible classification")
	}
	if len(scope.InputSignals) == 0 || hasBlank(scope.InputSignals) || strings.TrimSpace(scope.OperationalHistoryBasis) == "" {
		appendIssue(question, prefix+".scopeRecommendation", "SCOPE_RECOMMENDATION_REQUIRED", "scope recommendation requires input signals and an operational-history basis")
	}
	if strings.TrimSpace(scope.Rationale) == "" {
		appendIssue(question, prefix+".scopeRecommendation.rationale", "SCOPE_RATIONALE_REQUIRED", "scope recommendation requires a visible inclusion or deferral rationale")
	}
	if scope.ApprovalReviewState != "TECHNICAL_REVIEW_REQUIRED" {
		appendIssue(question, prefix+".scopeRecommendation.approvalReviewState", "SCOPE_REVIEW_STATE_REQUIRED", "immutable Draft scope state must require technical review; approval is projected only from an attributed decision")
	}
	guardrails := scope.Guardrails
	if question.MandatoryCore != guardrails.MandatoryControl || question.SafetyCritical != guardrails.SafetyCritical {
		appendIssue(question, prefix+".scopeRecommendation.guardrails", "SCOPE_GUARDRAIL_MISMATCH", "scope guardrails must preserve the question's mandatory and safety-critical controls")
	}
	if scope.AutomaticDeferral && (guardrails.MandatoryControl || guardrails.SafetyCritical || guardrails.UnknownHistory || guardrails.SourceChanged || guardrails.OverdueControl || !guardrails.AutomaticDeferralPermitted) {
		appendIssue(question, prefix+".scopeRecommendation.automaticDeferral", "AUTOMATIC_DEFERRAL_DENIED", "mandatory, safety-critical, changed, overdue, or unknown-history controls cannot be automatically deferred")
	}
	if question.Origin == HybridReconciledOrigin && !validQuestionReconciliation(question) {
		appendIssue(question, prefix+".reconciliation", "HYBRID_RECONCILIATION_REQUIRED", "hybrid reconciliation requires a complete candidate-only legacy/current comparison")
	} else if question.Origin != HybridReconciledOrigin && question.Reconciliation != nil {
		appendIssue(question, prefix+".reconciliation", "HYBRID_RECONCILIATION_REQUIRED", "only HYBRID_RECONCILED questions may carry a legacy/current comparison")
	}

	trace := question.RegulatoryTrace
	if question.Origin == ExistingChecklistCandidateOrigin && trace.State != SourceMappingRequired {
		appendIssue(question, prefix+".origin", "QUESTION_ORIGIN_TRACE_MISMATCH", "EXISTING_CHECKLIST_CANDIDATE remains a non-authoritative source-gap Draft until a HYBRID_RECONCILED current-source trace is created")
	}
	if question.Origin != ExistingChecklistCandidateOrigin && trace.State == SourceMappingRequired {
		appendIssue(question, prefix+".origin", "QUESTION_ORIGIN_TRACE_MISMATCH", "SOURCE_MAPPING_REQUIRED is reserved for an explicit EXISTING_CHECKLIST_CANDIDATE repair Draft")
	}
	if trace.State == SourceMappingRequired {
		if !isLiteralSourceMappingRequired(trace) || len(question.Citations) != 0 {
			appendIssue(question, prefix+".regulatoryTrace", "REGULATORY_TRACE_REQUIRED", "SOURCE_MAPPING_REQUIRED must remain a literal repair state without a partial citation or trace")
		}
		appendIssue(question, prefix+".regulatoryTrace", "SOURCE_MAPPING_REQUIRED", "SOURCE_MAPPING_REQUIRED must be repaired before validation, publication, deferral, or executable Audit use")
		return
	}
	if trace.State != "RESOLVED" {
		appendIssue(question, prefix+".regulatoryTrace", "REGULATORY_TRACE_REQUIRED", "every question requires a resolved regulatory trace or the literal SOURCE_MAPPING_REQUIRED state")
		return
	}
	if strings.TrimSpace(trace.SourceIdentity) == "" || strings.TrimSpace(trace.SourceTitle) == "" ||
		strings.TrimSpace(trace.ImmutableVersion) == "" || !strings.HasPrefix(trace.SHA256, "sha256:") ||
		strings.TrimSpace(trace.Locator) == "" || strings.TrimSpace(trace.Page) == "" ||
		strings.TrimSpace(trace.Section) == "" || strings.TrimSpace(trace.Clause) == "" ||
		strings.TrimSpace(trace.SourceType) == "" || strings.TrimSpace(trace.Applicability) == "" ||
		strings.TrimSpace(trace.NationalReference) == "" || strings.TrimSpace(trace.ControlledCAAProcedureMapping) == "" ||
		strings.TrimSpace(trace.VerificationObjective) == "" || len(trace.ExpectedEvidence) == 0 || hasBlank(trace.ExpectedEvidence) {
		appendIssue(question, prefix+".regulatoryTrace", "REGULATORY_TRACE_REQUIRED", "regulatory trace requires source identity, immutable version/hash, locator, applicability, procedure mapping, objective, and expected Evidence")
		return
	}
	if !sameStrings(trace.ExpectedEvidence, question.ExpectedEvidence) || len(question.Citations) != 1 ||
		question.Citations[0].SourceHash != trace.SHA256 || question.Citations[0].ClauseID != trace.Clause ||
		question.Citations[0].Locator != trace.Locator {
		appendIssue(question, prefix+".regulatoryTrace", "REGULATORY_TRACE_MISMATCH", "regulatory trace must exactly match the persisted citation and expected Evidence")
	}
	if trace.CurrentnessState == "STALE" {
		appendIssue(question, prefix+".regulatoryTrace.currentnessState", "STALE_SOURCE_TRACE", "a stale source version or hash blocks publication until a new impact-review Draft is approved")
	} else if trace.CurrentnessState != "CURRENT" {
		appendIssue(question, prefix+".regulatoryTrace.currentnessState", "SOURCE_CURRENTNESS_REQUIRED", "regulatory trace requires a currentness result")
	}
	if trace.TechnicalReviewState != "TECHNICAL_REVIEW_REQUIRED" {
		appendIssue(question, prefix+".regulatoryTrace.technicalReviewState", "TRACE_TECHNICAL_REVIEW_REQUIRED", "immutable Draft trace state must require technical review; approval is projected only from an attributed decision")
	}
	lineageMatches, err := traceMatchesPersistedLineage(ctx, query, question)
	if err != nil {
		appendIssue(question, prefix+".regulatoryTrace", "REGULATORY_TRACE_LINEAGE_UNAVAILABLE", "persisted regulatory trace lineage could not be verified")
	} else if !lineageMatches {
		appendIssue(question, prefix+".regulatoryTrace", "STALE_SOURCE_TRACE", "regulatory trace source version, hash, title, locator, or clause no longer matches the persisted chain")
	}
	changed, err := traceHasRecordedSourceChange(ctx, query, question)
	if err != nil {
		appendIssue(question, prefix+".regulatoryTrace.currentnessState", "SOURCE_CURRENTNESS_REQUIRED", "source-change impact state could not be verified")
	} else if changed {
		appendIssue(question, prefix+".regulatoryTrace.currentnessState", "STALE_SOURCE_TRACE", "a later source version created an immutable impact-review Draft for this trace")
	}
}

func traceMatchesPersistedLineage(
	ctx context.Context,
	query interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	question ChecklistQuestion,
) (bool, error) {
	if len(question.Citations) != 1 {
		return false, nil
	}
	citation := question.Citations[0]
	trace := question.RegulatoryTrace
	var count int
	err := query.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM regulatory_source_versions source
		JOIN regulatory_normalized_clauses clause
		  ON clause.regulatory_source_version_id=source.id
		WHERE source.id=$1 AND source.source_identity=$2 AND source.version_identity=$3
		  AND source.source_hash=$4 AND source.title=$5 AND clause.id=$6
		  AND clause.source_hash=$4 AND clause.clause_locator=$7`,
		citation.SourceSnapshotID, trace.SourceIdentity, trace.ImmutableVersion,
		trace.SHA256, trace.SourceTitle, trace.Clause, trace.Locator,
	).Scan(&count)
	return count == 1, err
}

func traceHasRecordedSourceChange(
	ctx context.Context,
	query interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	question ChecklistQuestion,
) (bool, error) {
	if len(question.Citations) != 1 {
		return false, nil
	}
	citation := question.Citations[0]
	var changed bool
	err := query.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM regulatory_source_currentness_events event
			WHERE event.previous_source_version_id=$1
			  AND event.previous_source_hash=$2
		)`, citation.SourceSnapshotID, question.RegulatoryTrace.SHA256).Scan(&changed)
	return changed, err
}

func traceHasActivatedCurrentnessBinding(
	ctx context.Context,
	query interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	generationRunID string,
	question ChecklistQuestion,
) (bool, error) {
	if len(question.Citations) != 1 {
		return false, nil
	}
	citation := question.Citations[0]
	var bound bool
	err := query.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM regulatory_generation_run_source_currentness_bindings binding
			JOIN regulatory_source_currentness_events event ON event.event_id=binding.currentness_event_id
			WHERE binding.generation_run_id=$1
			  AND binding.current_source_version_id=$2
			  AND binding.current_source_hash=$3
			  AND event.current_source_version_id=$2
			  AND event.current_source_hash=$3
		)`, generationRunID, citation.SourceSnapshotID, question.RegulatoryTrace.SHA256).Scan(&bound)
	return bound, err
}

// SourceCurrentnessLockKey serializes a source-change impact record with the
// executable-package currentness check for one source identity. The advisory
// lock is deliberately transaction scoped: it never mutates source or
// published-history rows, but it prevents a stale source event from racing a
// new Audit package snapshot.
func SourceCurrentnessLockKey(sourceIdentity string) string {
	return "regulatory-source-currentness:" + sourceIdentity
}

// LockGenerationRunSourceCurrentness takes the same deterministic source
// locks used by activation, publication, and executable-package materializing.
// Submission and technical approval call it before their final blocker read so
// no new review claim can race a committed source invalidation.
func LockGenerationRunSourceCurrentness(ctx context.Context, tx pgx.Tx, generationRunIDs []string) error {
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
			JOIN regulatory_source_versions source ON source.id=snapshot.regulatory_source_version_id
			WHERE snapshot.generation_run_id=$1
			ORDER BY source.source_identity`, generationRunID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var sourceIdentity string
			if err := rows.Scan(&sourceIdentity); err != nil {
				rows.Close()
				return err
			}
			identities[sourceIdentity] = true
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}
	ordered := make([]string, 0, len(identities))
	for sourceIdentity := range identities {
		ordered = append(ordered, sourceIdentity)
	}
	sort.Strings(ordered)
	for _, sourceIdentity := range ordered {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", SourceCurrentnessLockKey(sourceIdentity)); err != nil {
			return err
		}
	}
	return nil
}

// ProjectQuestionGovernanceCurrentness overlays mutable review facts onto the
// read model only. The immutable question snapshot is never updated: a newer
// source instead appears as STALE and its source-changed guardrail becomes
// visible until an impact-review Draft is separately approved and published.
func ProjectQuestionGovernanceCurrentness(
	ctx context.Context,
	query interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	candidateStatus string,
	generationRunID string,
	questions []ChecklistQuestion,
) error {
	for index := range questions {
		question := &questions[index]
		projectQuestionTechnicalReviewState(candidateStatus, question)
		if question.RegulatoryTrace.State == SourceMappingRequired {
			continue
		}
		bound, err := traceHasActivatedCurrentnessBinding(ctx, query, generationRunID, *question)
		if err != nil {
			return err
		}
		if !bound {
			question.RegulatoryTrace.CurrentnessState = "STALE"
			continue
		}
		changed, err := traceHasRecordedSourceChange(ctx, query, *question)
		if err != nil {
			return err
		}
		if changed {
			question.RegulatoryTrace.CurrentnessState = "STALE"
			question.ScopeRecommendation.Guardrails.SourceChanged = true
		}
	}
	return nil
}

// projectQuestionTechnicalReviewState decorates a read-only candidate view
// with the lifecycle fact already recorded for its exact immutable revision.
// It never writes the question snapshot: that snapshot must continue to state
// the review requirement that existed when the Draft was created.
func projectQuestionTechnicalReviewState(candidateStatus string, question *ChecklistQuestion) {
	if question.RegulatoryTrace.State == SourceMappingRequired {
		question.RegulatoryTrace.CurrentnessState = SourceMappingRequired
		question.RegulatoryTrace.TechnicalReviewState = "NOT_AVAILABLE"
		return
	}
	if candidateStatus == "TECHNICALLY_APPROVED" || candidateStatus == "PUBLISHED" {
		question.ScopeRecommendation.ApprovalReviewState = "TECHNICALLY_APPROVED"
		question.RegulatoryTrace.TechnicalReviewState = "TECHNICALLY_APPROVED"
	}
}
