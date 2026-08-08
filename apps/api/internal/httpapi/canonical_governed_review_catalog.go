package httpapi

import (
	"context"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi/generated"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/questioncatalog"
)

// queryGovernedReviewCandidateCatalog projects the current candidate leaf
// directly for pre-publication Question Review. New Audit never calls this
// purpose-bound path; it consumes only a sealed published catalog.
func (api *CanonicalAPI) queryGovernedReviewCandidateCatalog(ctx context.Context, actorSubjectID, candidateID, search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID, cursor string, limit int) ([]generated.CanonicalQuestionCatalogEntry, *string, int64, error) {
	offset, err := parseCatalogCursor(cursor)
	if err != nil {
		return nil, nil, 0, err
	}
	where := `candidate.id=$1 AND candidate.status IN ('DEPARTMENT_REVIEW','RETURNED','TECHNICALLY_APPROVED')
		AND NOT EXISTS (SELECT 1 FROM template_draft_versions successor WHERE successor.supersedes_candidate_id=candidate.id)
		AND ($2='' OR candidate.template_id ILIKE '%'||$2||'%' OR version.id ILIKE '%'||$2||'%' OR version.question_id ILIKE '%'||$2||'%' OR version.prompt ILIKE '%'||$2||'%')
		AND ($3='' OR candidate.template_id=$3)
		AND ($4='' OR COALESCE(snapshot.snapshot->>'reviewedDomain','')=$4)
		AND ($5='' OR COALESCE(snapshot.snapshot->>'reviewedTopic','')=$5)
		AND ($6='' OR COALESCE(snapshot.snapshot->'scopeRecommendation'->>'classification','')=$6)
		AND ($7='' OR $7='NONE')
		AND ($9='' OR $9='all' OR ($10<>'' AND (($9='selected') = EXISTS (
			SELECT 1
			FROM canonical_audit_scope_selection_questions selected_question
			JOIN canonical_audit_scope_selection_operations selected_operation ON selected_operation.id=selected_question.operation_id
			WHERE selected_operation.id=(
				SELECT latest.id
				FROM canonical_audit_scope_selection_operations latest
				WHERE latest.scope_draft_id=$10 AND latest.operation_kind <> 'PREVIEW'
				ORDER BY latest.created_at DESC,latest.id DESC
				LIMIT 1
			)
			AND selected_question.question_version_id=version.id
		))))
		AND ($10='' OR EXISTS (
			SELECT 1
			FROM regulatory_generation_runs candidate_run
			JOIN regulatory_generation_run_scope_facts candidate_fact ON candidate_fact.generation_run_id=candidate_run.id
			JOIN canonical_audit_scope_drafts requested_scope ON requested_scope.id=$10
			WHERE candidate_run.id=candidate.generation_run_id
			  AND candidate_fact.organization_service_provider_scope_id=requested_scope.provider_scope_id
			  AND candidate_fact.regulated_target_id=requested_scope.regulated_target_id
		))
		AND EXISTS (
			SELECT 1 FROM candidate_required_owner_assignments owner
			JOIN (SELECT DISTINCT ON (root_id) * FROM caa_department_memberships WHERE subject_id=$8 AND effective_from <= CURRENT_DATE ORDER BY root_id,effective_from DESC,id DESC) membership
			  ON membership.department_id=owner.department_id AND membership.organizational_unit_id=owner.organizational_unit_id
			JOIN LATERAL (SELECT status FROM caa_department_status_facts WHERE department_id=membership.department_id AND effective_from <= CURRENT_DATE ORDER BY effective_from DESC,id DESC LIMIT 1) department_status ON department_status.status='ACTIVE'
			JOIN LATERAL (SELECT status FROM caa_organizational_unit_status_facts WHERE organizational_unit_id=membership.organizational_unit_id AND effective_from <= CURRENT_DATE ORDER BY effective_from DESC,id DESC LIMIT 1) unit_status ON unit_status.status='ACTIVE'
			WHERE owner.candidate_draft_version_id=candidate.id AND owner.candidate_revision=candidate.revision AND owner.candidate_content_digest=candidate.candidate_content_digest AND owner.approval_required
			  AND membership.membership_role='DEPARTMENT_MANAGER' AND membership.status='ACTIVE' AND (membership.effective_to IS NULL OR membership.effective_to > CURRENT_DATE)
		)`
	countQuery := `SELECT COUNT(*) FROM template_draft_versions candidate JOIN question_versions version ON version.id=ANY(candidate.question_version_ids) JOIN regulatory_generated_question_snapshots snapshot ON snapshot.candidate_draft_version_id=candidate.id AND snapshot.question_id=version.question_id WHERE ` + where
	var total int64
	if err := api.pool.QueryRow(ctx, countQuery, candidateID, search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID, actorSubjectID).Scan(&total); err != nil {
		return nil, nil, 0, err
	}
	rows, err := api.pool.Query(ctx, `
			SELECT candidate.template_id,candidate.revision,candidate.candidate_content_digest,candidate.status,
			       version.id,version.question_id,ordered.ordinality,
			       governed_jsonb_sha256(jsonb_build_object('questionVersionId',version.id,'questionId',version.question_id,'prompt',version.prompt,'configuredReference',version.configured_reference,'expectedEvidence',version.expected_evidence)),
			       COALESCE(snapshot.snapshot->'regulatoryTrace'->>'locator',''),'NONE',
			       COALESCE(snapshot.snapshot->>'reviewedDomain',''),COALESCE(snapshot.snapshot->>'reviewedTopic',''),
			       COALESCE(snapshot.snapshot->'scopeRecommendation'->>'classification',''),
			       version.prompt,version.configured_reference,version.expected_evidence,
		       COALESCE(event.candidate_revision,COALESCE(NULLIF(snapshot.snapshot->>'reviewedRevision','')::bigint,0)),COALESCE(NULLIF(snapshot.snapshot->>'reviewedDisposition',''),event.action),COALESCE(NULLIF(snapshot.snapshot->>'reviewedReason',''),event.reason),
		       COALESCE(event.reviewed_domain,snapshot.snapshot->>'reviewedDomain'),
		       COALESCE(event.reviewed_topic,snapshot.snapshot->>'reviewedTopic')
		FROM template_draft_versions candidate
		CROSS JOIN unnest(candidate.question_version_ids) WITH ORDINALITY AS ordered(question_version_id,ordinality)
		JOIN question_versions version ON version.id=ordered.question_version_id
		JOIN regulatory_generated_question_snapshots snapshot ON snapshot.candidate_draft_version_id=candidate.id AND snapshot.question_id=version.question_id
		LEFT JOIN LATERAL (SELECT candidate_revision,action,reason,reviewed_domain,reviewed_topic FROM canonical_governed_question_review_events event WHERE (event.candidate_draft_version_id=candidate.id OR event.candidate_draft_version_id=candidate.supersedes_candidate_id) AND event.question_version_id=version.id ORDER BY event.created_at DESC,event.event_id DESC LIMIT 1) event ON TRUE
		WHERE `+where+`
		ORDER BY ordered.ordinality,version.id LIMIT $11 OFFSET $12`, candidateID, search, formCode, domain, topic, riskBand, sourceGapState, selected, scopeID, actorSubjectID, limit+1, offset)
	if err != nil {
		return nil, nil, 0, err
	}
	defer rows.Close()
	items := make([]generated.CanonicalQuestionCatalogEntry, 0, limit)
	for rows.Next() {
		var row canonicalCatalogRow
		var candidateStatus string
		if err := rows.Scan(&row.FormCode, &row.GovernedCandidateRevision, &row.GovernedCandidateContentDigest, &candidateStatus, &row.QuestionID, &row.ProposalID, &row.Ordinal, &row.Digest, &row.SourceLocator, &row.SourceGap, &row.Domain, &row.Topic, &row.RiskBand, &row.Prompt, &row.ConfiguredReference, &row.ExpectedEvidence, &row.ReviewRevision, &row.ReviewDisposition, &row.ReviewReason, &row.ReviewDomain, &row.ReviewTopic); err != nil {
			return nil, nil, 0, err
		}
		row.CatalogVersion = "candidate:" + candidateID
		row.ScopeID = scopeID
		row.UsageClass = string(questioncatalog.UsageClassGovernedOperational)
		row.GovernedCandidateID = &candidateID
		row.GovernedCandidateStatus = &candidateStatus
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
