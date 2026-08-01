package checklistintake

import (
	"context"
	"fmt"
)

type postgresTransaction struct{ tx databaseTx }

func (transaction *postgresTransaction) exec(ctx context.Context, query string, args ...any) error {
	if transaction == nil || transaction.tx == nil {
		return fmt.Errorf("checklist intake transaction is not configured")
	}
	if _, err := transaction.tx.Exec(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

func (transaction *postgresTransaction) InsertImportBatch(ctx context.Context, item ImportBatch) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_batches
		(import_batch_id, operation_id, idempotency_key, expected_archive_sha256,
		 observed_archive_sha256, observed_archive_bytes, status, manifest_digest,
		 intake_safety_eligible, reason, created_by_subject_id, created_at, finalized_at)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,0),$7,NULLIF($8,''),$9,$10,$11,COALESCE($12, now()),$13)
		ON CONFLICT (operation_id) DO NOTHING`,
		item.ImportBatchID, item.OperationID, item.IdempotencyKey, item.ExpectedArchiveSHA,
		item.ObservedArchiveSHA, item.ObservedArchiveByteCount, item.Status, item.ManifestDigest,
		item.IntakeSafetyEligible, item.Reason, item.CreatedBySubjectID, item.CreatedAt, item.FinalizedAt)
}

func (transaction *postgresTransaction) InsertImportFile(ctx context.Context, item ImportFile) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_files
		(import_file_id, import_batch_id, ordinal, normalized_path, original_path,
		 sha256, byte_count, media_type, initial_identity_match_state,
		 initial_candidate_import_state, register_form_code, register_title,
		 visible_title, terminal_manifest_digest, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),NULLIF($12,''),NULLIF($13,''),$14,COALESCE($15, now()))
		ON CONFLICT (import_file_id) DO NOTHING`,
		item.ImportFileID, item.ImportBatchID, item.Ordinal, item.NormalizedPath, item.OriginalPath,
		item.SHA256, item.ByteCount, item.MediaType, item.InitialIdentityMatchState,
		item.InitialCandidateImportState, item.RegisterFormCode, item.RegisterTitle,
		item.VisibleTitle, item.TerminalManifestDigest, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertPhaseReceipt(ctx context.Context, item PhaseReceipt) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_phase_receipts
		(receipt_id, import_batch_id, import_file_id, phase, input_digest, policy_version,
		 result_digest, outcome, error_code, payload, created_at)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,NULLIF($7,''),$8,NULLIF($9,''),$10,COALESCE($11, now()))
		ON CONFLICT (receipt_id) DO NOTHING`,
		item.ReceiptID, item.ImportBatchID, item.ImportFileID, item.Phase, item.InputDigest,
		item.PolicyVersion, item.ResultDigest, item.Outcome, item.ErrorCode, item.Payload, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertRegisterEntry(ctx context.Context, item RegisterEntry) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_register_entries
		(register_entry_id, import_batch_id, register_file_id, page, row_number, ordinal,
		 form_code, title_text, version_text, status_text, matched_import_file_id,
		 match_state, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),NULLIF($10,''),NULLIF($11,''),$12,COALESCE($13, now()))
		ON CONFLICT (register_entry_id) DO NOTHING`,
		item.RegisterEntryID, item.ImportBatchID, item.RegisterFileID, item.Page, item.RowNumber,
		item.Ordinal, item.FormCode, item.TitleText, item.VersionText, item.StatusText,
		item.MatchedImportFileID, item.MatchState, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertObjectIntent(ctx context.Context, item ObjectIntent) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_object_intents
		(intent_id, import_batch_id, import_file_id, purpose, object_key, expected_sha256,
		 expected_bytes, state, object_version, observed_sha256, observed_bytes, expires_at, created_at)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,NULLIF($9,''),NULLIF($10,''),NULLIF($11,0),$12,COALESCE($13, now()))
		ON CONFLICT (intent_id) DO NOTHING`,
		item.IntentID, item.ImportBatchID, item.ImportFileID, item.Purpose, item.ObjectKey,
		item.ExpectedSHA256, item.ExpectedBytes, item.State, item.ObjectVersion,
		item.ObservedSHA256, item.ObservedBytes, item.ExpiresAt, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertAttempt(ctx context.Context, item Attempt) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_attempts
		(attempt_id, attempt_root_id, predecessor_attempt_id, ordinal, phase,
		 import_batch_id, import_file_id, input_digest, policy_version, lease_owner,
		 lease_expires_at, fencing_token, created_at)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,NULLIF($7,''),$8,$9,NULLIF($10,''),$11,$12,COALESCE($13, now()))
		ON CONFLICT (attempt_id) DO NOTHING`,
		item.AttemptID, item.AttemptRootID, item.PredecessorAttemptID, item.Ordinal, item.Phase,
		item.ImportBatchID, item.ImportFileID, item.InputDigest, item.PolicyVersion,
		item.LeaseOwner, item.LeaseExpiresAt, item.FencingToken, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertAttemptEvent(ctx context.Context, item AttemptEvent) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_attempt_events
		(event_id, attempt_id, state, result_digest, error_code, fencing_token, completed_at)
		VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6,COALESCE($7, now()))
		ON CONFLICT (attempt_id) DO NOTHING`,
		item.EventID, item.AttemptID, item.State, item.ResultDigest, item.ErrorCode,
		item.FencingToken, item.CompletedAt)
}

func (transaction *postgresTransaction) InsertIdentityResolution(ctx context.Context, item IdentityResolution) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_identity_resolutions
		(resolution_id, resolution_root_id, resolution_revision, supersedes_resolution_id,
		 import_file_id, expected_prior_leaf_id, expected_prior_digest, expected_file_sha256,
		 expected_manifest_digest, selected_identity_source, selected_identity_value,
		 transcription_reason, transcription_receipt_id, competing_values, actor_subject_id,
		 actor_membership_id, reason, operation_id, idempotency_key, semantic_payload_digest, created_at)
		VALUES ($1,$2,$3,NULLIF($4,''),$5,NULLIF($6,''),NULLIF($7,''),$8,$9,$10,$11,
		 NULLIF($12,''),NULLIF($13,''),$14,$15,NULLIF($16,''),$17,$18,$19,$20,COALESCE($21, now()))
		ON CONFLICT (resolution_id) DO NOTHING`,
		item.ResolutionID, item.ResolutionRootID, item.ResolutionRevision, item.SupersedesResolutionID,
		item.ImportFileID, item.ExpectedPriorLeafID, item.ExpectedPriorDigest, item.ExpectedFileSHA256,
		item.ExpectedManifestDigest, item.SelectedIdentitySource, item.SelectedIdentityValue,
		item.TranscriptionReason, item.TranscriptionReceiptID, item.CompetingValues, item.ActorSubjectID,
		item.ActorMembershipID, item.Reason, item.OperationID, item.IdempotencyKey,
		item.SemanticPayloadDigest, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertExtractionPacket(ctx context.Context, item ExtractionReviewPacket) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_extraction_review_packets
		(packet_id, import_batch_id, import_file_id, terminal_manifest_digest, parser_receipt_id,
		 parser_output_digest, parser_output_bytes, generator_policy_version, outcome,
		 proposal_count, packet_digest, failure_code, created_by_subject_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,''),$13,COALESCE($14, now()))
		ON CONFLICT (packet_id) DO NOTHING`,
		item.PacketID, item.ImportBatchID, item.ImportFileID, item.TerminalManifestDigest,
		item.ParserReceiptID, item.ParserOutputDigest, item.ParserOutputBytes,
		item.GeneratorPolicyVersion, item.Outcome, item.ProposalCount, item.PacketDigest,
		item.FailureCode, item.CreatedBySubjectID, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertExtractionProposal(ctx context.Context, item ExtractionProposal) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_extraction_review_proposals
		(proposal_id, packet_id, proposal_ordinal, original_text, text_digest, page,
		 section, row_locator, region_locator, text_span, parser_provenance,
		 proposed_boundary_kind, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),$10,$11,$12,COALESCE($13, now()))
		ON CONFLICT (proposal_id) DO NOTHING`,
		item.ProposalID, item.PacketID, item.ProposalOrdinal, item.OriginalText, item.TextDigest,
		item.Page, item.Section, item.RowLocator, item.RegionLocator, item.TextSpan,
		item.ParserProvenance, item.ProposedBoundaryKind, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertExtractionDecisionSet(ctx context.Context, item ExtractionDecisionSet) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_extraction_decision_sets
		(decision_set_id, decision_set_root_id, revision, supersedes_decision_set_id,
		 packet_id, import_file_id, terminal_manifest_digest, parser_output_digest,
		 expected_prior_leaf_id, expected_prior_digest, actor_subject_id, reason,
		 operation_id, idempotency_key, semantic_payload_digest, created_at)
		VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,NULLIF($9,''),NULLIF($10,''),$11,$12,$13,$14,$15,COALESCE($16, now()))
		ON CONFLICT (decision_set_id) DO NOTHING`,
		item.DecisionSetID, item.DecisionSetRootID, item.Revision, item.SupersedesDecisionSetID,
		item.PacketID, item.ImportFileID, item.TerminalManifestDigest, item.ParserOutputDigest,
		item.ExpectedPriorLeafID, item.ExpectedPriorDigest, item.ActorSubjectID, item.Reason,
		item.OperationID, item.IdempotencyKey, item.SemanticPayloadDigest, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertExtractionDecision(ctx context.Context, item ExtractionDecision) error {
	return transaction.exec(ctx, `
		INSERT INTO checklist_import_extraction_decisions
		(decision_id, decision_set_id, decision_ordinal, decision_kind, consumed_proposal_ids,
		 consumed_proposal_digests, output_seed_ids, output_payload, reason, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,COALESCE($10, now()))
		ON CONFLICT (decision_id) DO NOTHING`,
		item.DecisionID, item.DecisionSetID, item.DecisionOrdinal, item.DecisionKind,
		item.ConsumedProposalIDs, item.ConsumedProposalDigests, item.OutputSeedIDs,
		item.OutputPayload, item.Reason, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertExistingCandidate(ctx context.Context, item ExistingCandidate) error {
	return transaction.exec(ctx, `
		INSERT INTO existing_checklist_candidates
		(existing_candidate_id, candidate_root_id, revision, supersedes_existing_candidate_id,
		 content_digest, import_batch_id, import_file_id, extraction_packet_id,
		 extraction_decision_set_id, identity_basis, resolution_id, origin, schema_version,
		 source_file_sha256, form_code, title, question_count, created_by_subject_id,
		 reason, created_at)
		VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,NULLIF($11,''),$12,$13,$14,$15,NULLIF($16,''),$17,$18,$19,COALESCE($20, now()))
		ON CONFLICT (existing_candidate_id) DO NOTHING`,
		item.ExistingCandidateID, item.CandidateRootID, item.Revision, item.SupersedesCandidateID,
		item.ContentDigest, item.ImportBatchID, item.ImportFileID, item.ExtractionPacketID,
		item.ExtractionDecisionSetID, item.IdentityBasis, item.ResolutionID, OriginExistingChecklistCandidate,
		item.SchemaVersion, item.SourceFileSHA256, item.FormCode, item.Title, item.QuestionCount,
		item.CreatedBySubjectID, item.Reason, item.CreatedAt)
}

func (transaction *postgresTransaction) InsertExistingCandidateQuestion(ctx context.Context, item ExistingCandidateQuestion) error {
	return transaction.exec(ctx, `
		INSERT INTO existing_checklist_candidate_questions
		(existing_candidate_id, question_id, ordinal, wording, source_locators,
		 operational_intent, expected_evidence, result_history, applicability,
		 scope_classification, provenance_state, created_at)
		VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),NULLIF($10,''),$11,COALESCE($12, now()))
		ON CONFLICT (existing_candidate_id, question_id) DO NOTHING`,
		item.ExistingCandidateID, item.QuestionID, item.Ordinal, item.Wording, item.SourceLocators,
		item.OperationalIntent, item.ExpectedEvidence, item.ResultHistory, item.Applicability,
		item.ScopeClassification, item.ProvenanceState, item.CreatedAt)
}
