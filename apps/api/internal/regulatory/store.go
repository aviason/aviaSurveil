package regulatory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

// ImportStore persists only already-validated generated drafts. It never creates
// a review, publication, checklist-template version, or Audit artifact.
type ImportStore struct{ Pool *database.Pool }
type ImportResult struct {
	GenerationRunID, InputDigest, OutputDigest string
	Replayed                                   bool
}

type persistedSourceCurrentnessBinding struct {
	EventID             string
	ImpactReviewDraftID *string
	SourceIdentity      string
	PreviousSourceID    *string
	PreviousSourceHash  *string
	CurrentSourceID     string
	CurrentSourceHash   string
}

// ValidateBlockedRealOPSAOCRequest resolves every persisted predecessor fact
// for the real request, then returns the authoritative block without creating
// a run or any downstream workflow artifact.
func (store ImportStore) ValidateBlockedRealOPSAOCRequest(ctx context.Context, request ValidatedGenerationRequest) error {
	_, err := store.ValidateBlockedRealOPSAOCInput(ctx, request.Request())
	return err
}

// ValidateBlockedRealOPSAOCInput validates the exact source-bound request and
// resolves all predecessor facts without creating lifecycle state.
func (store ImportStore) ValidateBlockedRealOPSAOCInput(ctx context.Context, request GenerationRequest) ([]UnresolvedSourceGap, error) {
	validated, err := validateRealOPSAOCRequest(request)
	if err != nil {
		return nil, err
	}
	if validated.sourceAuthorityResolved {
		return nil, ErrInvalidRequest
	}
	if store.Pool == nil {
		return nil, fmt.Errorf("local regulatory import database is required")
	}
	var resolved bool
	err = store.Pool.QueryRow(ctx, `SELECT
		EXISTS (SELECT 1 FROM organization_service_provider_scopes scope JOIN regulated_targets target ON target.id=scope.primary_target_id WHERE scope.id='SCOPE-OPS-AOC-SOURCE-BOUND' AND scope.root_id='SCOPE-OPS-AOC-SOURCE-BOUND' AND scope.organization_id='ORG-FLY-NAMIBIA' AND scope.service_provider_type_id='AIR_OPERATOR' AND scope.authorization_identifier='AOC-FLY-NAMIBIA-SOURCE-BOUND' AND scope.status='ACTIVE' AND scope.effective_from <= CURRENT_DATE AND (scope.effective_to IS NULL OR scope.effective_to > CURRENT_DATE) AND target.id='TARGET-OPS-AOC-SOURCE-BOUND' AND target.organization_id='ORG-FLY-NAMIBIA')
		AND EXISTS (SELECT 1 FROM regulatory_normalized_clauses clause JOIN regulatory_source_versions source ON source.id=clause.regulatory_source_version_id WHERE source.id='NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28' AND source.source_hash='sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2' AND clause.id='NCAA-CC-A610-4.2.2.2' AND clause.source_hash=source.source_hash AND clause.clause_locator='Annex 6 Part I 4.2.2.2')
		AND EXISTS (SELECT 1 FROM regulatory_evaluation_partition_rows row JOIN regulatory_evaluation_partitions partition ON partition.id=row.partition_id WHERE partition.id='CC-OPS-TRAIN-1' AND partition.partition_kind='GENERATION_INPUT' AND row.state_compliance_crosswalk_row_id='NCAA-CC-A610-ROW-4.2.2.2' AND row.stable_row_identity='CC:NAMB:ANNEX6:4.2.2.2')
		AND EXISTS (SELECT 1 FROM regulatory_source_gap_facts WHERE regulatory_source_version_id='NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28' AND gap_id='CONTROLLED_PROCEDURE' AND reason='The controlled NCAA Operations surveillance/ramp-inspection procedure has not been supplied.' AND ordinal=0)
		AND EXISTS (SELECT 1 FROM regulatory_source_gap_facts WHERE regulatory_source_version_id='NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28' AND gap_id='PART_140_AUTHORITY' AND reason='Current Part 140 authority and supersession require source-owner confirmation.' AND ordinal=1)
		AND EXISTS (SELECT 1 FROM regulatory_source_gap_facts WHERE regulatory_source_version_id='NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28' AND gap_id='PART_127_APPLICABILITY' AND reason='Exact Part 127 operation/configuration applicability requires Department Manager confirmation.' AND ordinal=2)
		AND EXISTS (SELECT 1 FROM regulatory_source_gap_facts WHERE regulatory_source_version_id='NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28' AND gap_id='AMBIGUOUS_OWNERSHIP' AND reason='Exact source ownership and controlled-procedure stewardship remain unresolved.' AND ordinal=3)`).Scan(&resolved)
	if err != nil {
		return nil, fmt.Errorf("resolve blocked real OPS/AOC request facts: %w", err)
	}
	if !resolved {
		return nil, ErrInvalidRequest
	}
	return append([]UnresolvedSourceGap(nil), validated.Request().UnresolvedSourceGaps...), ErrBlockedAuthority
}

func (store ImportStore) Import(ctx context.Context, bundle CandidateBundle) (ImportResult, error) {
	request, err := ValidateRequest(bundle.GenerationRequest, true)
	if err != nil {
		return ImportResult{}, err
	}
	if err := ValidateCandidateBundle(bundle, request); err != nil {
		return ImportResult{}, err
	}
	if store.Pool == nil {
		return ImportResult{}, fmt.Errorf("local regulatory import database is required")
	}
	result := ImportResult{}
	err = database.WithinTransaction(ctx, store.Pool, func(ctx context.Context, tx pgx.Tx) error {
		return importCandidateBundle(ctx, tx, bundle, &result)
	})
	return result, err
}

func (store ImportStore) ImportInTransaction(ctx context.Context, tx pgx.Tx, bundle CandidateBundle) (ImportResult, error) {
	request, err := ValidateRequest(bundle.GenerationRequest, true)
	if err != nil {
		return ImportResult{}, err
	}
	if err := ValidateCandidateBundle(bundle, request); err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{}
	err = importCandidateBundle(ctx, tx, bundle, &result)
	return result, err
}

func importCandidateBundle(ctx context.Context, tx pgx.Tx, bundle CandidateBundle, result *ImportResult) error {
	var existingID, existingOutput string
	err := tx.QueryRow(ctx, `SELECT id, output_digest FROM regulatory_generation_runs WHERE input_digest = $1`, bundle.InputDigest).Scan(&existingID, &existingOutput)
	if err == nil {
		if existingOutput != bundle.OutputDigest {
			return fmt.Errorf("input digest conflict: existing output differs")
		}
		if err := verifyReplayedImport(ctx, tx, bundle, existingID); err != nil {
			return err
		}
		*result = ImportResult{GenerationRunID: existingID, InputDigest: bundle.InputDigest, OutputDigest: bundle.OutputDigest, Replayed: true}
		return nil
	}
	if err != pgx.ErrNoRows {
		return fmt.Errorf("read idempotent generation run: %w", err)
	}
	requestArtifact, err := json.Marshal(bundle.GenerationRequest)
	if err != nil {
		return err
	}
	var requestObject map[string]any
	if err := json.Unmarshal(requestArtifact, &requestObject); err != nil {
		return err
	}
	delete(requestObject, "canonicalInputDigest")
	requestArtifact, err = json.Marshal(requestObject)
	if err != nil {
		return err
	}
	outputArtifact, err := json.Marshal(map[string]any{"complianceMappings": bundle.ComplianceMappings, "inspectionChecklist": bundle.InspectionChecklist})
	if err != nil {
		return err
	}
	var databaseInputDigest, databaseOutputDigest string
	if err := tx.QueryRow(ctx, `SELECT governed_jsonb_sha256($1::jsonb), governed_jsonb_sha256($2::jsonb)`, string(requestArtifact), string(outputArtifact)).Scan(&databaseInputDigest, &databaseOutputDigest); err != nil {
		return fmt.Errorf("compute persisted canonical digests: %w", err)
	}
	if databaseInputDigest != bundle.InputDigest {
		return fmt.Errorf("input digest does not match persisted canonical JSON: candidate=%s database=%s outputCandidate=%s outputDatabase=%s", bundle.InputDigest, databaseInputDigest, bundle.OutputDigest, databaseOutputDigest)
	}
	if databaseOutputDigest != bundle.OutputDigest {
		return fmt.Errorf("output digest does not match persisted canonical JSON: candidate=%s database=%s", bundle.OutputDigest, databaseOutputDigest)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO regulatory_generation_runs (id, status, input_digest, output_digest, input_schema_version, generation_policy_version, provider_catalog_version, provider_adapter_version, inspection_type, target_id, input_artifact, output_artifact) VALUES ($1, 'GENERATED', $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb)`, bundle.GenerationRunID, bundle.InputDigest, bundle.OutputDigest, bundle.GenerationRequest.SchemaVersion, bundle.GenerationRequest.GenerationPolicyVersion, bundle.GenerationRequest.ProviderCatalogVersion, bundle.GenerationRequest.ProviderVersion, bundle.GenerationRequest.InspectionType, bundle.GenerationRequest.Target.TargetID, string(requestArtifact), string(outputArtifact)); err != nil {
		return fmt.Errorf("persist generation run: %w", err)
	}
	scopeTag, err := tx.Exec(ctx, `INSERT INTO regulatory_generation_run_scope_facts (generation_run_id, organization_service_provider_scope_id, scope_root_id, organization_id, service_provider_type_id, authorization_identifier, scope_status, effective_from, effective_to, regulated_target_id) SELECT $1, id, root_id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, effective_to, primary_target_id FROM organization_service_provider_scopes WHERE id = $2`, bundle.GenerationRunID, bundle.GenerationRequest.ServiceProviderScopeFactIDs[0])
	if err != nil {
		return fmt.Errorf("persist exact generation scope fact: %w", err)
	}
	if scopeTag.RowsAffected() != 1 {
		return fmt.Errorf("persist exact generation scope fact: prerequisite synthetic scope identity is absent")
	}
	for _, source := range bundle.GenerationRequest.SourceSnapshots {
		for _, clauseID := range source.ClauseIDs {
			tag, err := tx.Exec(ctx, `INSERT INTO regulatory_generation_run_source_snapshots (generation_run_id, regulatory_source_version_id, regulatory_normalized_clause_id, source_hash, clause_locator) SELECT $1, $2, clause.id, $3, clause.clause_locator FROM regulatory_normalized_clauses clause WHERE clause.id = $4 AND clause.regulatory_source_version_id = $2`, bundle.GenerationRunID, source.SourceSnapshotID, source.SourceHash, clauseID)
			if err != nil {
				return fmt.Errorf("persist exact generation source snapshot: %w", err)
			}
			if tag.RowsAffected() != 1 {
				return fmt.Errorf("persist exact generation source snapshot: prerequisite source or clause identity is absent")
			}
		}
	}
	currentnessBinding, err := persistSourceCurrentnessBinding(ctx, tx, bundle)
	if err != nil {
		return err
	}
	for _, stableRowID := range bundle.GenerationRequest.SecondaryCrosswalkPartition.StableRowIDs {
		tag, err := tx.Exec(ctx, `INSERT INTO regulatory_generation_run_crosswalk_partition_rows (generation_run_id, evaluation_partition_id, state_compliance_crosswalk_row_id, stable_row_identity) SELECT $1, partition.id, row.state_compliance_crosswalk_row_id, row.stable_row_identity FROM regulatory_evaluation_partition_rows row JOIN regulatory_evaluation_partitions partition ON partition.id=row.partition_id WHERE partition.id=$2 AND partition.partition_kind='GENERATION_INPUT' AND row.stable_row_identity=$3`, bundle.GenerationRunID, bundle.GenerationRequest.SecondaryCrosswalkPartition.PartitionID, stableRowID)
		if err != nil {
			return fmt.Errorf("persist exact generation crosswalk partition: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("persist exact generation crosswalk partition: prerequisite input row is absent or holdout")
		}
	}
	questionVersionIDs := make([]string, 0, len(bundle.InspectionChecklist.Questions))
	for _, question := range bundle.InspectionChecklist.Questions {
		questionVersionID := "QV-" + question.QuestionID + "-V1"
		questionVersionIDs = append(questionVersionIDs, questionVersionID)
		if _, err := tx.Exec(ctx, `INSERT INTO question_versions (id, question_id, version, prompt, configured_reference, expected_evidence, created_by_subject_id) VALUES ($1, $2, 1, $3, $4, $5, 'SYNTHETIC-REGULATORY-GENERATOR')`, questionVersionID, question.QuestionID, question.Prompt, strings.Join(question.MappingIDs, ","), strings.Join(question.ExpectedEvidence, "; ")); err != nil {
			return fmt.Errorf("persist immutable question version: %w", err)
		}
	}
	templateID := "TPL-" + bundle.InspectionChecklist.ChecklistID
	if _, err := tx.Exec(ctx, `INSERT INTO template_masters (id, title, owner_role) VALUES ($1, $2, 'Admin Preview')`, templateID, bundle.InspectionChecklist.ChecklistID); err != nil {
		return fmt.Errorf("persist generated template master: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id) VALUES ($1, $2, 1, 'GENERATED_DRAFT', 'Admin Preview', 'SYNTHETIC-REGULATORY-GENERATOR', 'Imported deterministic synthetic governed candidate.', $3, 1, $4, $5, $6, $1)`, bundle.CandidateBundleID, templateID, questionVersionIDs, bundle.GenerationRunID, bundle.OutputDigest, bundle.SchemaVersion); err != nil {
		return fmt.Errorf("persist generated candidate draft: %w", err)
	}
	if currentnessBinding != nil && currentnessBinding.ImpactReviewDraftID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO regulatory_source_impact_candidate_links
				(id,impact_review_draft_id,candidate_draft_version_id,generation_run_id)
			VALUES ($1,$2,$3,$4)`,
			"SRC-IMPACT-CANDIDATE-"+bundle.CandidateBundleID,
			*currentnessBinding.ImpactReviewDraftID, bundle.CandidateBundleID, bundle.GenerationRunID,
		); err != nil {
			return fmt.Errorf("bind candidate to immutable source impact-review Draft: %w", err)
		}
	}
	for index, mapping := range bundle.ComplianceMappings {
		snapshot, err := json.Marshal(mapping)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO regulatory_generated_mapping_snapshots (candidate_draft_version_id, mapping_id, mapping_ordinal, snapshot) VALUES ($1, $2, $3, $4::jsonb)`, bundle.CandidateBundleID, mapping.MappingID, index, string(snapshot)); err != nil {
			return fmt.Errorf("persist immutable mapping snapshot: %w", err)
		}
	}
	for _, question := range bundle.InspectionChecklist.Questions {
		snapshot, err := json.Marshal(question)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO regulatory_generated_question_snapshots (candidate_draft_version_id, question_id, snapshot) VALUES ($1, $2, $3::jsonb)`, bundle.CandidateBundleID, question.QuestionID, string(snapshot)); err != nil {
			return fmt.Errorf("persist immutable question snapshot: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO candidate_required_owner_assignments (id, candidate_draft_version_id, candidate_revision, candidate_content_digest, department_id, organizational_unit_id, approval_required) VALUES ($1, $2, 1, $3, 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', true)`, "OWNER-"+bundle.CandidateBundleID, bundle.CandidateBundleID, bundle.OutputDigest); err != nil {
		return fmt.Errorf("persist required owner assignment: %w", err)
	}
	*result = ImportResult{GenerationRunID: bundle.GenerationRunID, InputDigest: bundle.InputDigest, OutputDigest: bundle.OutputDigest}
	return nil
}

// persistSourceCurrentnessBinding consumes an already committed explicit
// activation. Import deliberately cannot activate a supplied source row or
// create an impact-review Draft: observed source rows remain inert until the
// controlled currentness command commits the exact predecessor/current pair.
func persistSourceCurrentnessBinding(ctx context.Context, tx pgx.Tx, bundle CandidateBundle) (*persistedSourceCurrentnessBinding, error) {
	if bundle.SourceCurrentness == nil {
		return nil, nil
	}
	binding := bundle.SourceCurrentness
	result := &persistedSourceCurrentnessBinding{}
	err := tx.QueryRow(ctx, `
		SELECT event.event_id,draft.id,event.source_identity,
			event.previous_source_version_id,event.previous_source_hash,
			event.current_source_version_id,event.current_source_hash
		FROM regulatory_source_currentness_events event
		LEFT JOIN regulatory_source_impact_review_drafts draft ON draft.currentness_event_id=event.event_id
		WHERE event.current_source_version_id=$1
		  AND event.current_source_hash=$2
		  AND event.previous_source_version_id IS NOT DISTINCT FROM NULLIF($3, '')
		  AND event.previous_source_hash IS NOT DISTINCT FROM NULLIF($4, '')`,
		binding.CurrentSourceSnapshotID, binding.CurrentSourceHash,
		binding.PreviousSourceSnapshotID, binding.PreviousSourceHash,
	).Scan(
		&result.EventID, &result.ImpactReviewDraftID, &result.SourceIdentity,
		&result.PreviousSourceID, &result.PreviousSourceHash,
		&result.CurrentSourceID, &result.CurrentSourceHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: source-currentness activation is required before a traced candidate may be imported", application.ErrInvalid)
	}
	if err != nil {
		return nil, err
	}
	if result.CurrentSourceID != binding.CurrentSourceSnapshotID || result.CurrentSourceHash != binding.CurrentSourceHash ||
		(result.PreviousSourceID == nil) != (strings.TrimSpace(binding.PreviousSourceSnapshotID) == "") ||
		(result.PreviousSourceHash == nil) != (strings.TrimSpace(binding.PreviousSourceHash) == "") {
		return nil, fmt.Errorf("%w: source-currentness activation binding is inconsistent", application.ErrInvalid)
	}
	if binding.PreviousSourceSnapshotID != "" && result.ImpactReviewDraftID == nil {
		return nil, fmt.Errorf("%w: source-change activation is missing its impact-review Draft", application.ErrInvalid)
	}
	if binding.PreviousSourceSnapshotID == "" && result.ImpactReviewDraftID != nil {
		return nil, fmt.Errorf("%w: baseline source activation cannot bind an impact-review Draft", application.ErrInvalid)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO regulatory_generation_run_source_currentness_bindings
			(generation_run_id,currentness_event_id,impact_review_draft_id,source_identity,previous_source_version_id,previous_source_hash,current_source_version_id,current_source_hash)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		bundle.GenerationRunID, result.EventID, result.ImpactReviewDraftID, result.SourceIdentity,
		result.PreviousSourceID, result.PreviousSourceHash, result.CurrentSourceID, result.CurrentSourceHash,
	); err != nil {
		return nil, fmt.Errorf("persist immutable generation source-currentness binding: %w", err)
	}
	return result, nil
}

// loadVerifiedGenerationRunSourceCurrentnessBinding reconstructs the immutable
// activation proof for an Admin-created candidate revision. A revision keeps
// its parent's frozen generation run; it must neither invent a new activation
// nor lose the original proof while validating its edited question snapshots.
// A source-change binding additionally has to retain the original root's
// immutable impact-Draft link.
func loadVerifiedGenerationRunSourceCurrentnessBinding(
	ctx context.Context,
	tx pgx.Tx,
	generationRunID string,
	candidateRootID string,
) (*SourceCurrentnessBinding, error) {
	var eventID, sourceIdentity, currentSourceID, currentSourceHash string
	var previousSourceID, previousSourceHash, impactReviewDraftID *string
	err := tx.QueryRow(ctx, `
		SELECT currentness_event_id,source_identity,
			previous_source_version_id,previous_source_hash,
			current_source_version_id,current_source_hash,impact_review_draft_id
		FROM regulatory_generation_run_source_currentness_bindings
		WHERE generation_run_id=$1`, generationRunID,
	).Scan(
		&eventID, &sourceIdentity,
		&previousSourceID, &previousSourceHash,
		&currentSourceID, &currentSourceHash, &impactReviewDraftID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var eventSourceIdentity, eventCurrentSourceID, eventCurrentSourceHash string
	var eventPreviousSourceID, eventPreviousSourceHash *string
	if err := tx.QueryRow(ctx, `
		SELECT source_identity,previous_source_version_id,previous_source_hash,
			current_source_version_id,current_source_hash
		FROM regulatory_source_currentness_events
		WHERE event_id=$1`, eventID,
	).Scan(
		&eventSourceIdentity, &eventPreviousSourceID, &eventPreviousSourceHash,
		&eventCurrentSourceID, &eventCurrentSourceHash,
	); err != nil {
		return nil, fmt.Errorf("load immutable source-currentness event: %w", err)
	}
	if sourceIdentity != eventSourceIdentity ||
		currentSourceID != eventCurrentSourceID || currentSourceHash != eventCurrentSourceHash ||
		!sameOptionalString(previousSourceID, eventPreviousSourceID) ||
		!sameOptionalString(previousSourceHash, eventPreviousSourceHash) ||
		(previousSourceID == nil) != (previousSourceHash == nil) {
		return nil, fmt.Errorf("%w: generation source-currentness binding does not match its immutable activation event", application.ErrInvalid)
	}
	if previousSourceID == nil {
		if impactReviewDraftID != nil {
			return nil, fmt.Errorf("%w: baseline generation source-currentness binding unexpectedly references an impact-review Draft", application.ErrInvalid)
		}
	} else {
		if impactReviewDraftID == nil {
			return nil, fmt.Errorf("%w: source-change generation currentness binding is missing its impact-review Draft", application.ErrInvalid)
		}
		var linked int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM regulatory_source_impact_review_drafts draft
			JOIN regulatory_source_impact_candidate_links link
				ON link.impact_review_draft_id=draft.id
			WHERE draft.id=$1
			  AND draft.currentness_event_id=$2
			  AND link.generation_run_id=$3
			  AND link.candidate_draft_version_id=$4`,
			*impactReviewDraftID, eventID, generationRunID, candidateRootID,
		).Scan(&linked); err != nil || linked != 1 {
			return nil, fmt.Errorf("%w: generation source-change binding is missing its immutable impact-Draft candidate link: count=%d err=%v", application.ErrInvalid, linked, err)
		}
	}
	binding := &SourceCurrentnessBinding{
		CurrentSourceSnapshotID: currentSourceID,
		CurrentSourceHash:       currentSourceHash,
	}
	if previousSourceID != nil {
		binding.PreviousSourceSnapshotID = *previousSourceID
		binding.PreviousSourceHash = *previousSourceHash
	}
	return binding, nil
}

func sameOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

// verifyReplayedImport treats replay as a read of the complete immutable graph,
// not merely a matching generation-run digest. A partial or conflicting graph is
// always an import error; it must never be silently presented as idempotency.
func verifyReplayedImport(ctx context.Context, tx pgx.Tx, bundle CandidateBundle, generationRunID string) error {
	expectedSources := 0
	for _, source := range bundle.GenerationRequest.SourceSnapshots {
		expectedSources += len(source.ClauseIDs)
	}
	checks := []struct {
		name            string
		query           string
		want            int
		candidateScoped bool
	}{
		{name: "scope facts", query: `SELECT COUNT(*) FROM regulatory_generation_run_scope_facts WHERE generation_run_id=$1`, want: len(bundle.GenerationRequest.ServiceProviderScopeFactIDs)},
		{name: "source snapshots", query: `SELECT COUNT(*) FROM regulatory_generation_run_source_snapshots WHERE generation_run_id=$1`, want: expectedSources},
		{name: "input partition rows", query: `SELECT COUNT(*) FROM regulatory_generation_run_crosswalk_partition_rows WHERE generation_run_id=$1`, want: len(bundle.GenerationRequest.SecondaryCrosswalkPartition.StableRowIDs)},
		{name: "question snapshots", query: `SELECT COUNT(*) FROM regulatory_generated_question_snapshots WHERE candidate_draft_version_id=$1`, want: len(bundle.InspectionChecklist.Questions), candidateScoped: true},
		{name: "mapping snapshots", query: `SELECT COUNT(*) FROM regulatory_generated_mapping_snapshots WHERE candidate_draft_version_id=$1`, want: len(bundle.ComplianceMappings), candidateScoped: true},
		{name: "owner assignments", query: `SELECT COUNT(*) FROM candidate_required_owner_assignments WHERE candidate_draft_version_id=$1 AND candidate_content_digest=$2`, want: 1},
	}
	for _, check := range checks {
		var got int
		var err error
		if check.name == "owner assignments" {
			err = tx.QueryRow(ctx, check.query, bundle.CandidateBundleID, bundle.OutputDigest).Scan(&got)
		} else if check.candidateScoped {
			err = tx.QueryRow(ctx, check.query, bundle.CandidateBundleID).Scan(&got)
		} else {
			err = tx.QueryRow(ctx, check.query, generationRunID).Scan(&got)
		}
		if err != nil || got != check.want {
			return fmt.Errorf("replay graph is incomplete or conflicting for %s: got=%d want=%d err=%v", check.name, got, check.want, err)
		}
	}
	var draftCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM template_draft_versions WHERE id=$1 AND generation_run_id=$2 AND candidate_content_digest=$3 AND candidate_schema_version=$4 AND cardinality(question_version_ids)=$5`, bundle.CandidateBundleID, generationRunID, bundle.OutputDigest, bundle.SchemaVersion, len(bundle.InspectionChecklist.Questions)).Scan(&draftCount); err != nil || draftCount != 1 {
		return fmt.Errorf("replay graph is incomplete or conflicting for candidate draft: got=%d err=%v", draftCount, err)
	}
	if err := verifyReplayLineageValues(ctx, tx, bundle, generationRunID, true); err != nil {
		return err
	}
	if err := verifyReplaySourceCurrentnessBinding(ctx, tx, bundle, generationRunID); err != nil {
		return err
	}
	for index, mapping := range bundle.ComplianceMappings {
		if err := verifyMappingSnapshot(ctx, tx, bundle.CandidateBundleID, mapping.MappingID, index, mapping); err != nil {
			return err
		}
	}
	for _, question := range bundle.InspectionChecklist.Questions {
		if err := verifySnapshot(ctx, tx, "regulatory_generated_question_snapshots", "question_id", bundle.CandidateBundleID, question.QuestionID, question); err != nil {
			return err
		}
	}
	return nil
}

func verifyReplaySourceCurrentnessBinding(ctx context.Context, tx pgx.Tx, bundle CandidateBundle, generationRunID string) error {
	if bundle.SourceCurrentness == nil {
		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generation_run_source_currentness_bindings WHERE generation_run_id=$1`, generationRunID).Scan(&count); err != nil || count != 0 {
			return fmt.Errorf("replay graph has unexpected source-currentness binding: got=%d err=%v", count, err)
		}
		return nil
	}
	binding := bundle.SourceCurrentness
	var eventID string
	var impactReviewDraftID *string
	var count int
	err := tx.QueryRow(ctx, `
		SELECT binding.currentness_event_id,binding.impact_review_draft_id,COUNT(*) OVER ()
		FROM regulatory_generation_run_source_currentness_bindings binding
		JOIN regulatory_source_currentness_events event ON event.event_id=binding.currentness_event_id
		WHERE binding.generation_run_id=$1
		  AND binding.current_source_version_id=$2 AND binding.current_source_hash=$3
		  AND binding.previous_source_version_id IS NOT DISTINCT FROM NULLIF($4, '')
		  AND binding.previous_source_hash IS NOT DISTINCT FROM NULLIF($5, '')
		  AND event.current_source_version_id=$2 AND event.current_source_hash=$3
		  AND event.previous_source_version_id IS NOT DISTINCT FROM NULLIF($4, '')
		  AND event.previous_source_hash IS NOT DISTINCT FROM NULLIF($5, '')`,
		generationRunID, binding.CurrentSourceSnapshotID, binding.CurrentSourceHash,
		binding.PreviousSourceSnapshotID, binding.PreviousSourceHash,
	).Scan(&eventID, &impactReviewDraftID, &count)
	if err != nil || count != 1 || eventID == "" {
		return fmt.Errorf("replay graph is incomplete or conflicting for source-currentness binding: count=%d err=%v", count, err)
	}
	if binding.PreviousSourceSnapshotID == "" {
		if impactReviewDraftID != nil {
			return fmt.Errorf("replay graph baseline source-currentness binding unexpectedly has impact-review Draft")
		}
		return nil
	}
	if impactReviewDraftID == nil {
		return fmt.Errorf("replay graph source-change currentness binding is missing impact-review Draft")
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM regulatory_source_impact_candidate_links
		WHERE impact_review_draft_id=$1 AND candidate_draft_version_id=$2 AND generation_run_id=$3`,
		*impactReviewDraftID, bundle.CandidateBundleID, generationRunID,
	).Scan(&count); err != nil || count != 1 {
		return fmt.Errorf("replay graph is incomplete or conflicting for source-impact candidate binding: count=%d err=%v", count, err)
	}
	return nil
}

func verifyMappingSnapshot(ctx context.Context, tx pgx.Tx, candidateID, mappingID string, ordinal int, expected ComplianceMapping) error {
	expectedDigest, err := CanonicalSHA256(expected)
	if err != nil {
		return fmt.Errorf("canonicalize replay snapshot %s: %w", mappingID, err)
	}
	var actualDigest string
	var actualOrdinal int
	if err := tx.QueryRow(ctx, `
		SELECT governed_jsonb_sha256(snapshot),mapping_ordinal
		FROM regulatory_generated_mapping_snapshots
		WHERE candidate_draft_version_id=$1 AND mapping_id=$2`,
		candidateID, mappingID,
	).Scan(&actualDigest, &actualOrdinal); err != nil ||
		actualDigest != expectedDigest || actualOrdinal != ordinal {
		return fmt.Errorf(
			"replay graph mapping snapshot is missing, conflicting, or reordered for %s: digest=%s/%s ordinal=%d/%d err=%v",
			mappingID, actualDigest, expectedDigest, actualOrdinal, ordinal, err,
		)
	}
	return nil
}

func verifyReplayLineageValues(ctx context.Context, tx pgx.Tx, bundle CandidateBundle, generationRunID string, requireNoDownstreamEffects bool) error {
	request := bundle.GenerationRequest
	var exact int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generation_run_scope_facts fact JOIN organization_service_provider_scopes scope ON scope.id=fact.organization_service_provider_scope_id JOIN regulated_targets target ON target.id=fact.regulated_target_id WHERE fact.generation_run_id=$1 AND fact.organization_service_provider_scope_id=$2 AND fact.scope_root_id=scope.root_id AND fact.organization_id=scope.organization_id AND fact.service_provider_type_id=scope.service_provider_type_id AND fact.authorization_identifier=scope.authorization_identifier AND fact.scope_status=scope.status AND fact.effective_from=scope.effective_from AND fact.effective_to IS NOT DISTINCT FROM scope.effective_to AND fact.regulated_target_id=scope.primary_target_id AND target.organization_id=scope.organization_id`, generationRunID, request.ServiceProviderScopeFactIDs[0]).Scan(&exact); err != nil || exact != 1 {
		return fmt.Errorf("replay graph is incomplete or conflicting for exact scope fact: got=%d err=%v", exact, err)
	}
	for _, source := range request.SourceSnapshots {
		for index, clauseID := range source.ClauseIDs {
			if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generation_run_source_snapshots snapshot JOIN regulatory_normalized_clauses clause ON clause.id=snapshot.regulatory_normalized_clause_id JOIN regulatory_source_versions source ON source.id=snapshot.regulatory_source_version_id WHERE snapshot.generation_run_id=$1 AND snapshot.regulatory_source_version_id=$2 AND snapshot.regulatory_normalized_clause_id=$3 AND snapshot.source_hash=$4 AND snapshot.clause_locator=$5 AND source.source_hash=$4 AND clause.regulatory_source_version_id=$2 AND clause.source_hash=$4 AND clause.clause_locator=$5`, generationRunID, source.SourceSnapshotID, clauseID, source.SourceHash, source.ClauseLocators[index]).Scan(&exact); err != nil || exact != 1 {
				return fmt.Errorf("replay graph is incomplete or conflicting for exact source snapshot %s: got=%d err=%v", clauseID, exact, err)
			}
		}
	}
	for _, stableRowID := range request.SecondaryCrosswalkPartition.StableRowIDs {
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generation_run_crosswalk_partition_rows snapshot JOIN regulatory_evaluation_partitions partition ON partition.id=snapshot.evaluation_partition_id JOIN regulatory_evaluation_partition_rows row ON row.partition_id=snapshot.evaluation_partition_id AND row.state_compliance_crosswalk_row_id=snapshot.state_compliance_crosswalk_row_id JOIN state_compliance_crosswalk_rows crosswalk ON crosswalk.id=snapshot.state_compliance_crosswalk_row_id WHERE snapshot.generation_run_id=$1 AND snapshot.evaluation_partition_id=$2 AND snapshot.stable_row_identity=$3 AND partition.partition_kind='GENERATION_INPUT' AND row.stable_row_identity=$3 AND crosswalk.stable_row_identity=$3`, generationRunID, request.SecondaryCrosswalkPartition.PartitionID, stableRowID).Scan(&exact); err != nil || exact != 1 {
			return fmt.Errorf("replay graph is incomplete or conflicting for exact input partition row %s: got=%d err=%v", stableRowID, exact, err)
		}
	}
	questionVersionIDs := make([]string, 0, len(bundle.InspectionChecklist.Questions))
	for _, question := range bundle.InspectionChecklist.Questions {
		questionVersionIDs = append(questionVersionIDs, "QV-"+question.QuestionID+"-V1")
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM template_draft_versions WHERE id=$1 AND generation_run_id=$2 AND revision=1 AND candidate_content_digest=$3 AND candidate_schema_version=$4 AND question_version_ids=$5`, bundle.CandidateBundleID, generationRunID, bundle.OutputDigest, bundle.SchemaVersion, questionVersionIDs).Scan(&exact); err != nil || exact != 1 {
		return fmt.Errorf("replay graph is incomplete or conflicting for exact candidate identity/revision/questions: got=%d err=%v", exact, err)
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM candidate_required_owner_assignments WHERE candidate_draft_version_id=$1 AND candidate_revision=1 AND candidate_content_digest=$2 AND department_id='FLIGHT_OPERATIONS_INSPECTORATE' AND organizational_unit_id='FLIGHT_OPERATIONS_INSPECTORATE' AND approval_required=true`, bundle.CandidateBundleID, bundle.OutputDigest).Scan(&exact); err != nil || exact != 1 {
		return fmt.Errorf("replay graph is incomplete or conflicting for exact required owner: got=%d err=%v", exact, err)
	}
	if requireNoDownstreamEffects {
		if err := tx.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM department_review_decisions WHERE candidate_draft_version_id=$1) + (SELECT COUNT(*) FROM checklist_publication_decisions WHERE candidate_draft_version_id=$1) + (SELECT COUNT(*) FROM checklist_template_versions WHERE candidate_draft_version_id=$1) + (SELECT COUNT(*) FROM inspection_packages WHERE checklist_template_version_id IN (SELECT id FROM checklist_template_versions WHERE candidate_draft_version_id=$1))`, bundle.CandidateBundleID).Scan(&exact); err != nil || exact != 0 {
			return fmt.Errorf("replay graph has forbidden review, publication, template-version, or Audit effects: got=%d err=%v", exact, err)
		}
	}
	return nil
}

func verifySnapshot(ctx context.Context, tx pgx.Tx, table, identityColumn, candidateID, identity string, expected any) error {
	expectedDigest, err := CanonicalSHA256(expected)
	if err != nil {
		return fmt.Errorf("canonicalize replay snapshot %s: %w", identity, err)
	}
	query := fmt.Sprintf(`SELECT governed_jsonb_sha256(snapshot) FROM %s WHERE candidate_draft_version_id=$1 AND %s=$2`, table, identityColumn)
	var actualDigest string
	if err := tx.QueryRow(ctx, query, candidateID, identity).Scan(&actualDigest); err != nil || actualDigest != expectedDigest {
		return fmt.Errorf("replay graph snapshot is missing or conflicting for %s: actual=%s expected=%s err=%v", identity, actualDigest, expectedDigest, err)
	}
	return nil
}
