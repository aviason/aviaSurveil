package qualificationbootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

const PriorAuditHistoryLoaderActor = "avia-bootstrap"

type PriorAuditHistoryManifest struct {
	SchemaVersion       int                      `json:"schemaVersion"`
	ManifestVersion     string                   `json:"manifestVersion"`
	Target              string                   `json:"target"`
	Enabled             bool                     `json:"enabled"`
	QualificationOnly   bool                     `json:"qualificationOnly"`
	CatalogVersion      string                   `json:"catalogVersion"`
	OrganizationID      string                   `json:"organizationId"`
	ProviderScopeRootID string                   `json:"providerScopeRootId"`
	ProviderScopeID     string                   `json:"providerScopeId"`
	RegulatedTargetID   string                   `json:"regulatedTargetId"`
	Location            string                   `json:"location"`
	AuditType           string                   `json:"auditType"`
	HistoryWindowMonths int                      `json:"historyWindowMonths"`
	QuestionVersionIDs  []string                 `json:"questionVersionIds"`
	Audits              []PriorAuditHistoryAudit `json:"audits"`
}

type PriorAuditHistoryAudit struct {
	AuditID             string                         `json:"auditId"`
	PlanningDraftID     string                         `json:"planningDraftId"`
	ScopeDraftID        string                         `json:"scopeDraftId"`
	ScopeSnapshotID     string                         `json:"scopeSnapshotId"`
	InspectionPackageID string                         `json:"inspectionPackageId"`
	ReportID            string                         `json:"reportId"`
	ReportVersionID     string                         `json:"reportVersionId"`
	IssuedAt            time.Time                      `json:"issuedAt"`
	ScheduledDate       string                         `json:"scheduledDate"`
	Observations        []PriorAuditHistoryObservation `json:"observations"`
}

type PriorAuditHistoryObservation struct {
	QuestionVersionID string `json:"questionVersionId"`
	Result            string `json:"result"`
	Signal            string `json:"signal"`
}

type priorAuditQuestion struct {
	VersionID  string
	QuestionID string
}

func ValidatePriorAuditHistoryManifest(manifest PriorAuditHistoryManifest, target string) error {
	if manifest.SchemaVersion != 1 || strings.TrimSpace(manifest.ManifestVersion) == "" {
		return fmt.Errorf("prior-audit history manifest identity is invalid")
	}
	if manifest.Target != target || !isPriorAuditHistoryTarget(target) {
		return fmt.Errorf("prior-audit history manifest target is not an approved Namibia local or demo target")
	}
	if !manifest.Enabled || !manifest.QualificationOnly {
		return fmt.Errorf("prior-audit history manifest must be enabled and qualification-only")
	}
	if manifest.CatalogVersion != "aga-approved-source@2.0.0" || manifest.AuditType != "RAMP_INSPECTION" || manifest.HistoryWindowMonths != 36 {
		return fmt.Errorf("prior-audit history manifest policy identity is invalid")
	}
	for _, value := range []string{manifest.OrganizationID, manifest.ProviderScopeRootID, manifest.ProviderScopeID, manifest.RegulatedTargetID, manifest.Location} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("prior-audit history manifest contains an empty scope identity")
		}
	}
	if len(manifest.QuestionVersionIDs) != 8 || len(manifest.Audits) != 3 {
		return fmt.Errorf("prior-audit history manifest must contain exactly eight questions and three Audits")
	}
	questionSet := make(map[string]struct{}, len(manifest.QuestionVersionIDs))
	for _, questionID := range manifest.QuestionVersionIDs {
		if strings.TrimSpace(questionID) == "" {
			return fmt.Errorf("prior-audit history question identity is empty")
		}
		if _, exists := questionSet[questionID]; exists {
			return fmt.Errorf("prior-audit history question identities are not distinct")
		}
		questionSet[questionID] = struct{}{}
	}
	auditSet := make(map[string]struct{}, len(manifest.Audits))
	for _, audit := range manifest.Audits {
		for _, value := range []string{audit.AuditID, audit.PlanningDraftID, audit.ScopeDraftID, audit.ScopeSnapshotID, audit.InspectionPackageID, audit.ReportID, audit.ReportVersionID, audit.ScheduledDate} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("prior-audit history record contains an empty identity")
			}
		}
		if audit.IssuedAt.IsZero() || len(audit.Observations) == 0 {
			return fmt.Errorf("prior-audit history record %s is incomplete", audit.AuditID)
		}
		if _, exists := auditSet[audit.AuditID]; exists {
			return fmt.Errorf("prior-audit history Audit identities are not distinct")
		}
		auditSet[audit.AuditID] = struct{}{}
		seenObservations := make(map[string]struct{}, len(audit.Observations))
		for _, observation := range audit.Observations {
			if _, exists := questionSet[observation.QuestionVersionID]; !exists {
				return fmt.Errorf("prior-audit history observation references an undeclared question")
			}
			if observation.Result != "COMPLIANT" && observation.Result != "NON_COMPLIANT" {
				return fmt.Errorf("prior-audit history observation result is invalid")
			}
			if observation.Signal != "CLEAN" && observation.Signal != "OPEN" && observation.Signal != "REPEAT" && observation.Signal != "OVERDUE" {
				return fmt.Errorf("prior-audit history observation signal is invalid")
			}
			if _, exists := seenObservations[observation.QuestionVersionID]; exists {
				return fmt.Errorf("prior-audit history observations are not distinct")
			}
			seenObservations[observation.QuestionVersionID] = struct{}{}
		}
	}
	return nil
}

func isPriorAuditHistoryTarget(target string) bool {
	return target == "namibia/dev" || target == "namibia/demo"
}

func ReadPriorAuditHistoryManifest(path, target string) (PriorAuditHistoryManifest, string, error) {
	var manifest PriorAuditHistoryManifest
	if !strings.HasPrefix(path, "/") || strings.ContainsRune(path, '\x00') {
		return manifest, "", fmt.Errorf("prior-audit history manifest path must be absolute")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, "", fmt.Errorf("read prior-audit history manifest: %w", err)
	}
	sum := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, "", fmt.Errorf("decode prior-audit history manifest: %w", err)
	}
	if decoder.More() {
		return manifest, "", fmt.Errorf("prior-audit history manifest contains trailing JSON")
	}
	if err := ValidatePriorAuditHistoryManifest(manifest, target); err != nil {
		return manifest, "", err
	}
	return manifest, digest, nil
}

func LoadPriorAuditHistory(ctx context.Context, pool *database.Pool, manifest PriorAuditHistoryManifest, actorSubjectID string, now time.Time) error {
	if pool == nil || strings.TrimSpace(actorSubjectID) == "" || now.IsZero() {
		return fmt.Errorf("prior-audit history loader requires database, actor, and timestamp")
	}
	if err := ValidatePriorAuditHistoryManifest(manifest, manifest.Target); err != nil {
		return err
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(41010202)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ($1, 'avia:bootstrap', 'Avia deployment bootstrap') ON CONFLICT (subject_id) DO NOTHING`, actorSubjectID); err != nil {
			return fmt.Errorf("register prior-audit loader actor: %w", err)
		}
		var catalogID string
		if err := tx.QueryRow(ctx, `SELECT id FROM canonical_question_catalogs WHERE catalog_version=$1 AND status='SEALED' AND source_origin='IMPORTED_APPROVED_SOURCE'`, manifest.CatalogVersion).Scan(&catalogID); err != nil {
			return fmt.Errorf("resolve approved catalog for prior-audit history: %w", err)
		}
		questions, err := resolvePriorAuditQuestions(ctx, tx, catalogID, manifest.QuestionVersionIDs)
		if err != nil {
			return err
		}
		selectionDigest := orderedSelectionDigest(manifest.QuestionVersionIDs)
		for _, audit := range manifest.Audits {
			if err := loadPriorAudit(ctx, tx, manifest, catalogID, questions, selectionDigest, audit, actorSubjectID, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func resolvePriorAuditQuestions(ctx context.Context, tx pgx.Tx, catalogID string, versionIDs []string) (map[string]priorAuditQuestion, error) {
	rows, err := tx.Query(ctx, `
		SELECT membership.question_version_id, version.question_id
		FROM canonical_question_catalog_memberships membership
		JOIN question_versions version ON version.id=membership.question_version_id
		WHERE membership.catalog_id=$1 AND membership.question_version_id=ANY($2::text[])
	`, catalogID, versionIDs)
	if err != nil {
		return nil, fmt.Errorf("resolve prior-audit question identities: %w", err)
	}
	defer rows.Close()
	result := make(map[string]priorAuditQuestion, len(versionIDs))
	for rows.Next() {
		var question priorAuditQuestion
		if err := rows.Scan(&question.VersionID, &question.QuestionID); err != nil {
			return nil, err
		}
		result[question.VersionID] = question
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) != len(versionIDs) {
		return nil, fmt.Errorf("approved catalog does not contain every public prior-audit question")
	}
	return result, nil
}

func orderedSelectionDigest(ids []string) string {
	hash := sha256.New()
	for position, id := range ids {
		_, _ = fmt.Fprintf(hash, "%d\x00%s\n", position, id)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func loadPriorAudit(ctx context.Context, tx pgx.Tx, manifest PriorAuditHistoryManifest, catalogID string, questions map[string]priorAuditQuestion, selectionDigest string, audit PriorAuditHistoryAudit, actor string, now time.Time) error {
	observations := make(map[string]PriorAuditHistoryObservation, len(audit.Observations))
	for _, observation := range audit.Observations {
		observations[observation.QuestionVersionID] = observation
	}
	values, err := json.Marshal(map[string]any{
		"organizationId": manifest.OrganizationID, "organizationName": priorAuditOrganizationName(manifest.Target),
		"applicationType": manifest.AuditType, "domain": "Cabin Safety", "inspectionCategory": "Routine / Announced",
		"noticePolicy": "ADVANCE", "purpose": "Immutable prior-Audit recommendation fixture.",
		"triggerType": "Department Manager initiated", "riskCategory": "Operational Safety",
		"plannedDate": audit.ScheduledDate, "mode": "On-site", "location": manifest.Location,
		"catalogVersion": manifest.CatalogVersion, "scopeDraftId": audit.ScopeDraftID,
		"providerScopeId": manifest.ProviderScopeID, "regulatedTargetId": manifest.RegulatedTargetID,
		"selectedQuestionVersionIds": manifest.QuestionVersionIDs, "selectionDigest": selectionDigest,
		"estimatedResourceRequirement": len(manifest.QuestionVersionIDs), "requestedBudget": 0, "currency": "NAD",
	})
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO planning_intake_drafts (id, organization_id, values, revision, created_by_subject_id, created_at, updated_at)
		VALUES ($1,$2,$3::jsonb,1,$4,$5,$5) ON CONFLICT (id) DO NOTHING
	`, audit.PlanningDraftID, manifest.OrganizationID, string(values), actor, audit.IssuedAt); err != nil {
		return fmt.Errorf("insert prior-audit planning draft %s: %w", audit.AuditID, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO canonical_audit_scope_drafts (
			id, planning_intake_draft_id, organization_id, provider_scope_id, regulated_target_id,
			audit_type, catalog_id, usage_class, revision, status, selected_question_count,
			selection_digest, requested_budget, notice_policy, created_by_subject_id, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,'GOVERNED_OPERATIONAL',1,'RELEASED',$8,$9,0,'ADVANCE',$10,$11,$11)
		ON CONFLICT (id) DO NOTHING
	`, audit.ScopeDraftID, audit.PlanningDraftID, manifest.OrganizationID, manifest.ProviderScopeID, manifest.RegulatedTargetID, manifest.AuditType, catalogID, len(manifest.QuestionVersionIDs), selectionDigest, actor, audit.IssuedAt); err != nil {
		return fmt.Errorf("insert prior-audit scope %s: %w", audit.AuditID, err)
	}
	for position, versionID := range manifest.QuestionVersionIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO canonical_audit_scope_draft_questions (scope_draft_id, revision, catalog_id, question_version_id, position, selection_digest)
			VALUES ($1,1,$2,$3,$4,$5) ON CONFLICT (scope_draft_id, revision, question_version_id) DO NOTHING
		`, audit.ScopeDraftID, catalogID, versionID, position, selectionDigest); err != nil {
			return fmt.Errorf("insert prior-audit scope question %s: %w", audit.AuditID, err)
		}
	}
	snapshot, err := json.Marshal(map[string]any{
		"fixture": manifest.ManifestVersion, "auditId": audit.AuditID, "planningDraftId": audit.PlanningDraftID,
		"organizationId": manifest.OrganizationID, "providerScopeId": manifest.ProviderScopeID,
		"regulatedTargetId": manifest.RegulatedTargetID, "location": manifest.Location,
		"auditType": manifest.AuditType, "catalogVersion": manifest.CatalogVersion,
		"selectedQuestionVersionIds": manifest.QuestionVersionIDs, "selectionDigest": selectionDigest,
	})
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO canonical_audit_scope_snapshots (
			id, scope_draft_id, revision, stage, catalog_id, usage_class, selection_digest,
			selected_question_count, snapshot, planning_snapshot_digest, created_by_subject_id, created_at
		) VALUES ($1,$2,1,'RELEASED',$3,'GOVERNED_OPERATIONAL',$4,$5,$6::jsonb,governed_jsonb_sha256($6::jsonb),$7,$8)
		ON CONFLICT (id) DO NOTHING
	`, audit.ScopeSnapshotID, audit.ScopeDraftID, catalogID, selectionDigest, len(manifest.QuestionVersionIDs), string(snapshot), actor, audit.IssuedAt); err != nil {
		return fmt.Errorf("insert prior-audit scope snapshot %s: %w", audit.AuditID, err)
	}
	for position, versionID := range manifest.QuestionVersionIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO canonical_audit_scope_snapshot_questions (snapshot_id, catalog_id, question_version_id, position)
			VALUES ($1,$2,$3,$4) ON CONFLICT (snapshot_id, question_version_id) DO NOTHING
		`, audit.ScopeSnapshotID, catalogID, versionID, position); err != nil {
			return fmt.Errorf("insert prior-audit snapshot question %s: %w", audit.AuditID, err)
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO inspections (id, organization_id, assigned_inspector_subject_id, title, inspection_type, status, due_date, revision, created_at, updated_at)
		VALUES ($1,$2,$3,'Public prior-Audit recommendation fixture',$4,'COMPLETED',$5,1,$6,$6)
		ON CONFLICT (id) DO NOTHING
	`, audit.AuditID, manifest.OrganizationID, actor, manifest.AuditType, audit.ScheduledDate, audit.IssuedAt); err != nil {
		return fmt.Errorf("insert prior-Audit %s: %w", audit.AuditID, err)
	}
	packageSnapshot, err := json.Marshal(map[string]any{"fixture": manifest.ManifestVersion, "auditId": audit.AuditID, "questionVersionIds": manifest.QuestionVersionIDs, "scopeSnapshotId": audit.ScopeSnapshotID})
	if err != nil {
		return err
	}
	packageSum := sha256.Sum256(packageSnapshot)
	packageDigest := "sha256:" + hex.EncodeToString(packageSum[:])
	if _, err := tx.Exec(ctx, `
		INSERT INTO inspection_packages (id, inspection_id, checklist_template_version_id, canonical_scope_snapshot_id, package_version, snapshot, expires_at, package_digest)
		VALUES ($1,$2,NULL,$3,1,$4::jsonb,$5,$6) ON CONFLICT (id) DO NOTHING
	`, audit.InspectionPackageID, audit.AuditID, audit.ScopeSnapshotID, string(packageSnapshot), audit.IssuedAt.AddDate(10, 0, 0), packageDigest); err != nil {
		return fmt.Errorf("insert prior-Audit package %s: %w", audit.AuditID, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO inspection_checklists (inspection_id, status, revision, submitted_at) VALUES ($1,'COMPLETED',1,$2) ON CONFLICT (inspection_id) DO NOTHING`, audit.AuditID, audit.IssuedAt); err != nil {
		return fmt.Errorf("insert prior-Audit checklist %s: %w", audit.AuditID, err)
	}
	for position, versionID := range manifest.QuestionVersionIDs {
		question := questions[versionID]
		if _, err := tx.Exec(ctx, `INSERT INTO inspection_question_assignments (inspection_id, question_id, subject_id, assignment_revision) VALUES ($1,$2,$3,1) ON CONFLICT DO NOTHING`, audit.AuditID, question.QuestionID, actor); err != nil {
			return fmt.Errorf("insert prior-Audit assignment %s: %w", audit.AuditID, err)
		}
		observation, observed := observations[versionID]
		if !observed {
			continue
		}
		responseID := fmt.Sprintf("response-%s-%02d", audit.AuditID, position+1)
		if _, err := tx.Exec(ctx, `
			INSERT INTO checklist_responses (id, inspection_id, package_id, question_id, assigned_inspector_subject_id, response_value, comment_to_auditee, internal_caa_note, revision, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,'Historical fixture response','Internal prior-Audit recommendation fixture',1,$7)
			ON CONFLICT (id) DO NOTHING
		`, responseID, audit.AuditID, audit.InspectionPackageID, question.QuestionID, actor, observation.Result, audit.IssuedAt); err != nil {
			return fmt.Errorf("insert prior-Audit response %s: %w", audit.AuditID, err)
		}
		if err := loadPriorAuditSignal(ctx, tx, audit, question.QuestionID, responseID, observation.Signal, actor); err != nil {
			return err
		}
	}
	reportSnapshot := fmt.Sprintf(`{"kind":"FINAL","fixture":"%s","auditId":"%s","scopeSnapshotId":"%s"}`, manifest.ManifestVersion, audit.AuditID, audit.ScopeSnapshotID)
	if _, err := tx.Exec(ctx, `
		INSERT INTO report_versions (id, report_id, inspection_id, version, status, snapshot, created_at)
		VALUES ($1,$2,$3,1,'ISSUED',$4::jsonb,$5) ON CONFLICT (id) DO NOTHING
	`, audit.ReportVersionID, audit.ReportID, audit.AuditID, reportSnapshot, audit.IssuedAt); err != nil {
		return fmt.Errorf("insert prior-Audit report %s: %w", audit.AuditID, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO report_approval_states (report_version_id, status, revision, issued_at, updated_at)
		VALUES ($1,'LOCKED',1,$2,$3) ON CONFLICT (report_version_id) DO NOTHING
	`, audit.ReportVersionID, audit.IssuedAt, now); err != nil {
		return fmt.Errorf("lock prior-Audit report %s: %w", audit.AuditID, err)
	}
	return nil
}

func priorAuditOrganizationName(target string) string {
	if target == "namibia/dev" {
		return "Namibia Dev AGA Qualification Operator"
	}
	return "Namibia AGA Qualification Operator"
}

func loadPriorAuditSignal(ctx context.Context, tx pgx.Tx, audit PriorAuditHistoryAudit, questionID, responseID, signal, actor string) error {
	if signal == "CLEAN" {
		return nil
	}
	potentialID := responseID + "-potential"
	if signal == "OPEN" {
		_, err := tx.Exec(ctx, `
			INSERT INTO potential_findings (id, inspection_id, checklist_response_id, organization_id, status, finding_basis, expected_evidence, comment_to_auditee, internal_caa_note, revision, question_id, title, description, created_by_subject_id)
			SELECT $1,$2,$3,organization_id,'PENDING_LEAD_REVIEW','Open prior-Audit work','Updated evidence','Provide updated evidence','Internal prior-Audit signal',1,$4,'Open prior-Audit work','Open prior-Audit work',$5 FROM inspections WHERE id=$2
			ON CONFLICT (id) DO NOTHING
		`, potentialID, audit.AuditID, responseID, questionID, actor)
		if err != nil {
			return fmt.Errorf("insert open prior-Audit signal %s: %w", audit.AuditID, err)
		}
		return nil
	}
	findingID := potentialID + "-finding"
	potentialStatus := "CONVERTED"
	if _, err := tx.Exec(ctx, `
		INSERT INTO potential_findings (id, inspection_id, checklist_response_id, organization_id, status, finding_basis, expected_evidence, comment_to_auditee, internal_caa_note, converted_finding_id, revision, question_id, title, description, created_by_subject_id)
		SELECT $1,$2,$3,organization_id,$4,'Historical prior-Audit signal','Evidence for prior-Audit signal','Provide evidence','Internal prior-Audit signal',$5,1,$6,'Historical prior-Audit signal','Historical prior-Audit signal',$7 FROM inspections WHERE id=$2
		ON CONFLICT (id) DO NOTHING
	`, potentialID, audit.AuditID, responseID, potentialStatus, findingID, questionID, actor); err != nil {
		return fmt.Errorf("insert prior-Audit finding precursor %s: %w", audit.AuditID, err)
	}
	findingStatus, nextAction, dueDate := "CLOSED", "Repeat control in next Audit", any(nil)
	if signal == "OVERDUE" {
		findingStatus, nextAction, dueDate = "WAITING_FOR_CAP", "Submit CAP", "2026-07-01"
	}
	if signal != "REPEAT" && signal != "OVERDUE" {
		return fmt.Errorf("unsupported prior-Audit signal %s", signal)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO findings (id, reference, potential_finding_id, inspection_id, organization_id, severity, status, owner_subject_id, next_action, due_date, closure_basis, closure_reason, revision, created_at, updated_at)
		SELECT $1,$2,$3,$4,organization_id,'LEVEL_2_MAJOR',$5,$6,$7,$8,$9,$10,1,now(),now() FROM inspections WHERE id=$4
		ON CONFLICT (id) DO NOTHING
	`, findingID, "PRIOR-"+audit.AuditID, potentialID, audit.AuditID, findingStatus, actor, nextAction, dueDate, nullableClosureBasis(findingStatus), nullableClosureReason(findingStatus)); err != nil {
		return fmt.Errorf("insert prior-Audit finding %s: %w", audit.AuditID, err)
	}
	if signal == "OVERDUE" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO cap_revisions (id, cap_id, finding_id, organization_id, revision, status, root_cause, corrective_action, preventive_action, target_completion_date, submitted_by_subject_id, submitted_at)
			SELECT $1,$2,$3,organization_id,1,'SUBMITTED','Historical overdue CAP','Corrective action pending','Preventive action pending','2026-07-01',$4,now() FROM findings WHERE id=$3
			ON CONFLICT (id) DO NOTHING
		`, findingID+"-cap", findingID+"-cap", findingID, actor); err != nil {
			return fmt.Errorf("insert overdue prior-Audit CAP %s: %w", audit.AuditID, err)
		}
	}
	return nil
}

func nullableClosureBasis(status string) any {
	if status == "CLOSED" {
		return "EVIDENCE_VERIFIED"
	}
	return nil
}

func nullableClosureReason(status string) any {
	if status == "CLOSED" {
		return "Historical repeat-control evidence retained for recommendation policy."
	}
	return nil
}
