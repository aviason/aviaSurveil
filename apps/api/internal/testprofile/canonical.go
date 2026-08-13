package testprofile

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/aviason/aviaSurveil/internal/documents"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/questioncatalog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	CanonicalAuditID            = "AUD-2026-001"
	CanonicalPackageID          = "PKG-CAB-2026-001"
	CanonicalFindingID          = "FND-CAB-2026-001"
	CanonicalInspectorSubjectID = "154ec5ac-6f97-4f55-916f-d2f142fc6211"
)

func CanonicalScenarioTime() time.Time {
	return time.Date(2026, time.June, 15, 9, 0, 0, 0, time.UTC)
}

type Generator struct {
	mu       sync.Mutex
	counters map[string]int
}

func NewGenerator() *Generator {
	return &Generator{counters: map[string]int{}}
}

func (generator *Generator) Reset() {
	generator.mu.Lock()
	defer generator.mu.Unlock()
	generator.counters = map[string]int{}
}

func (generator *Generator) Next(prefix string) string {
	generator.mu.Lock()
	defer generator.mu.Unlock()
	generator.counters[prefix]++
	sequence := generator.counters[prefix]
	switch prefix {
	case "potential-finding":
		return fmt.Sprintf("PF-2026-%03d", sequence)
	case "finding":
		return fmt.Sprintf("FND-CAB-2026-%03d", sequence)
	case "cap":
		return fmt.Sprintf("CAP-CAB-2026-%03d", sequence)
	case "cap-revision":
		return fmt.Sprintf("CAP-CAB-2026-001-R%d", sequence)
	case "evidence":
		return "EV-CAB-2026-001"
	case "evidence-version":
		return fmt.Sprintf("EV-CAB-2026-001-V%d", sequence)
	case "grant":
		return fmt.Sprintf("GRANT-CANDIDATE-%03d", sequence)
	default:
		return fmt.Sprintf("%s-candidate-%03d", prefix, sequence)
	}
}

func (generator *Generator) FindingReference() string {
	generator.mu.Lock()
	defer generator.mu.Unlock()
	generator.counters["finding-reference"]++
	return fmt.Sprintf("CAB-2026-%03d", generator.counters["finding-reference"])
}

type canonicalQuestion struct {
	ID                  string   `json:"id"`
	SectionID           string   `json:"sectionId"`
	Prompt              string   `json:"prompt"`
	RegulatoryReference string   `json:"regulatoryReference"`
	ExpectedEvidence    string   `json:"expectedEvidence"`
	AllowedAnswers      []string `json:"allowedAnswers"`
	CommentRequiredFor  []string `json:"commentRequiredFor"`
	Assigned            []string `json:"assignedInspectorUserIds"`
}

const historicalFullProfileChecklistBootstrapSubjectID = "TEST-HISTORICAL-CHECKLIST-BOOTSTRAP"

// BootstrapHistoricalFullProfileChecklist creates only the immutable historical
// checklist relationship required by the isolated local full-profile harness.
// It is an internal test-profile seam, never a normal application command.
func BootstrapHistoricalFullProfileChecklist(
	ctx context.Context,
	pool *database.Pool,
	now time.Time,
) error {
	if pool == nil || now.IsZero() {
		return errors.New("historical full-profile checklist bootstrap requires database and server time")
	}
	now = now.UTC()
	allowedAnswers := []string{"COMPLIANT", "NON_COMPLIANT", "OBSERVATION", "NOT_APPLICABLE"}
	commentRequiredFor := []string{"NON_COMPLIANT", "OBSERVATION"}
	questions := []canonicalQuestion{
		{ID: "CAB-EMEQ-PBE-001", SectionID: "CABIN-SAFETY", Prompt: "Verify authorized cabin control 1.", RegulatoryReference: "NAM-CAR-CABIN-1", ExpectedEvidence: "Versioned cabin evidence 1", AllowedAnswers: allowedAnswers, CommentRequiredFor: commentRequiredFor},
		{ID: "CAB-LAV-001", SectionID: "CABIN-SAFETY", Prompt: "Verify authorized cabin control 2.", RegulatoryReference: "NAM-CAR-CABIN-2", ExpectedEvidence: "Versioned cabin evidence 2", AllowedAnswers: allowedAnswers, CommentRequiredFor: commentRequiredFor},
		{ID: "CAB-PAX-SEAT-001", SectionID: "CABIN-SAFETY", Prompt: "Verify authorized cabin control 3.", RegulatoryReference: "NAM-CAR-CABIN-3", ExpectedEvidence: "Versioned cabin evidence 3", AllowedAnswers: allowedAnswers, CommentRequiredFor: commentRequiredFor},
		{ID: "CAB-VID-CREW-SEAT-001", SectionID: "CABIN-OPERATIONS", Prompt: "Verify authorized cabin control 4.", RegulatoryReference: "NAM-CAR-CABIN-4", ExpectedEvidence: "Versioned cabin evidence 4", AllowedAnswers: allowedAnswers, CommentRequiredFor: commentRequiredFor},
		{ID: "CAB-GALLEY-001", SectionID: "CABIN-OPERATIONS", Prompt: "Verify authorized cabin control 5.", RegulatoryReference: "NAM-CAR-CABIN-5", ExpectedEvidence: "Versioned cabin evidence 5", AllowedAnswers: allowedAnswers, CommentRequiredFor: commentRequiredFor},
		{ID: "CAB-DOORS-001", SectionID: "CABIN-OPERATIONS", Prompt: "Verify authorized cabin control 6.", RegulatoryReference: "NAM-CAR-CABIN-6", ExpectedEvidence: "Versioned cabin evidence 6", AllowedAnswers: allowedAnswers, CommentRequiredFor: commentRequiredFor},
	}
	snapshot, err := json.Marshal(map[string]any{
		"schemaVersion": 1, "protocolVersion": 1,
		"creatorSubjectId": historicalFullProfileChecklistBootstrapSubjectID,
		"changeReason":     "Historical immutable full-profile checklist fixture.",
		"questions":        questions,
	})
	if err != nil {
		return fmt.Errorf("marshal historical full-profile checklist snapshot: %w", err)
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, transaction pgx.Tx) error {
		identityExists, err := validateHistoricalBootstrapIdentity(ctx, transaction, now)
		if err != nil {
			return err
		}
		checklistExists, err := validateHistoricalChecklistVersion(ctx, transaction, snapshot, now)
		if err != nil {
			return err
		}
		questionsComplete, questionsPresent, err := validateHistoricalQuestionVersions(ctx, transaction, questions, now)
		if err != nil {
			return err
		}
		masterExists, err := validateHistoricalTemplateMaster(ctx, transaction, now)
		if err != nil {
			return err
		}
		if identityExists || checklistExists || questionsPresent || masterExists {
			if !(identityExists && checklistExists && questionsComplete && masterExists) {
				return errors.New("historical checklist conflict: incomplete existing fixture")
			}
			return validateHistoricalQuestionRelationships(ctx, transaction, questions)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO identity_references (subject_id, issuer, display_name, created_at)
			VALUES ($1, 'urn:avia:internal-testprofile', 'Historical checklist bootstrap', $2)
		`, historicalFullProfileChecklistBootstrapSubjectID, now); err != nil {
			return fmt.Errorf("bootstrap historical checklist identity: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO checklist_template_versions (
				id, template_id, version, title, snapshot, published_at
			) VALUES ('CTV-CABIN-1', 'TPL-CABIN-2026', 1, 'Cabin Inspection checklist', $1, $2)
		`, snapshot, now); err != nil {
			return fmt.Errorf("bootstrap historical checklist version: %w", err)
		}
		for position, question := range questions {
			questionVersionID := "QV-" + question.ID + "-V1"
			if _, err := transaction.Exec(ctx, `
				INSERT INTO question_versions (
					id, question_id, version, prompt, configured_reference,
					expected_evidence, created_by_subject_id, created_at
				) VALUES ($1, $2, 1, $3, $4, $5, $6, $7)
			`, questionVersionID, question.ID, question.Prompt, question.RegulatoryReference,
				question.ExpectedEvidence, historicalFullProfileChecklistBootstrapSubjectID, now); err != nil {
				return fmt.Errorf("bootstrap historical question %s: %w", question.ID, err)
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO template_version_questions (
					template_version_id, question_version_id, position, created_at
				) VALUES ('CTV-CABIN-1', $1, $2, $3)
			`, questionVersionID, position, now); err != nil {
				return fmt.Errorf("bootstrap historical question relationship %s: %w", question.ID, err)
			}
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO template_masters (
				id, title, owner_role, published_template_version_id,
				revision, created_at, updated_at
			) VALUES ('TPL-CABIN-2026', 'Cabin Inspection checklist', 'Department Manager', 'CTV-CABIN-1', 1, $1, $1)
		`, now); err != nil {
			return fmt.Errorf("bootstrap historical template master: %w", err)
		}
		return nil
	})
}

func validateHistoricalBootstrapIdentity(ctx context.Context, transaction pgx.Tx, now time.Time) (bool, error) {
	var issuer, displayName string
	var createdAt time.Time
	err := transaction.QueryRow(ctx, `
		SELECT issuer, display_name, created_at FROM identity_references WHERE subject_id = $1
	`, historicalFullProfileChecklistBootstrapSubjectID).Scan(&issuer, &displayName, &createdAt)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read historical bootstrap identity: %w", err)
	}
	if issuer != "urn:avia:internal-testprofile" || displayName != "Historical checklist bootstrap" || !createdAt.Equal(now) {
		return false, errors.New("historical bootstrap identity conflict")
	}
	return true, nil
}

func validateHistoricalChecklistVersion(ctx context.Context, transaction pgx.Tx, expectedSnapshot []byte, now time.Time) (bool, error) {
	var templateID, title string
	var version int64
	var snapshot []byte
	var publishedAt time.Time
	err := transaction.QueryRow(ctx, `
		SELECT template_id, version, title, snapshot, published_at
		FROM checklist_template_versions WHERE id = 'CTV-CABIN-1'
	`).Scan(&templateID, &version, &title, &snapshot, &publishedAt)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read historical checklist version: %w", err)
	}
	if templateID != "TPL-CABIN-2026" || version != 1 || title != "Cabin Inspection checklist" ||
		!publishedAt.Equal(now) || !sameHistoricalJSON(snapshot, expectedSnapshot) {
		return false, errors.New("historical checklist conflict")
	}
	return true, nil
}

func validateHistoricalQuestionVersions(ctx context.Context, transaction pgx.Tx, questions []canonicalQuestion, now time.Time) (allExist bool, anyExist bool, err error) {
	allExist = true
	for _, question := range questions {
		var id, prompt, reference, evidence, creator string
		var version int64
		var createdAt time.Time
		err := transaction.QueryRow(ctx, `
			SELECT id, version, prompt, configured_reference, expected_evidence,
			       created_by_subject_id, created_at
			FROM question_versions WHERE question_id = $1 AND version = 1
		`, question.ID).Scan(&id, &version, &prompt, &reference, &evidence, &creator, &createdAt)
		if err == pgx.ErrNoRows {
			allExist = false
			continue
		}
		if err != nil {
			return false, false, fmt.Errorf("read historical question %s: %w", question.ID, err)
		}
		anyExist = true
		if id != "QV-"+question.ID+"-V1" || version != 1 || prompt != question.Prompt ||
			reference != question.RegulatoryReference || evidence != question.ExpectedEvidence ||
			creator != historicalFullProfileChecklistBootstrapSubjectID || !createdAt.Equal(now) {
			return false, true, fmt.Errorf("historical question conflict: %s", question.ID)
		}
	}
	return allExist, anyExist, nil
}

func validateHistoricalTemplateMaster(ctx context.Context, transaction pgx.Tx, now time.Time) (bool, error) {
	var title, owner, publishedVersionID string
	var revision int64
	var createdAt, updatedAt time.Time
	err := transaction.QueryRow(ctx, `
		SELECT title, owner_role, published_template_version_id, revision, created_at, updated_at
		FROM template_masters WHERE id = 'TPL-CABIN-2026'
	`).Scan(&title, &owner, &publishedVersionID, &revision, &createdAt, &updatedAt)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read historical template master: %w", err)
	}
	if title != "Cabin Inspection checklist" || owner != "Department Manager" ||
		publishedVersionID != "CTV-CABIN-1" || revision != 1 || !createdAt.Equal(now) || !updatedAt.Equal(now) {
		return false, errors.New("historical template master conflict")
	}
	return true, nil
}

func validateHistoricalQuestionRelationships(ctx context.Context, transaction pgx.Tx, questions []canonicalQuestion) error {
	rows, err := transaction.Query(ctx, `
		SELECT relation.question_version_id, relation.position
		FROM template_version_questions relation
		WHERE relation.template_version_id = 'CTV-CABIN-1'
		ORDER BY relation.position
	`)
	if err != nil {
		return fmt.Errorf("read historical checklist relationships: %w", err)
	}
	defer rows.Close()
	for position, question := range questions {
		if !rows.Next() {
			return errors.New("historical checklist relationship conflict")
		}
		var questionVersionID string
		var actualPosition int
		if err := rows.Scan(&questionVersionID, &actualPosition); err != nil {
			return fmt.Errorf("scan historical checklist relationship: %w", err)
		}
		if questionVersionID != "QV-"+question.ID+"-V1" || actualPosition != position {
			return errors.New("historical checklist relationship conflict")
		}
	}
	if rows.Next() {
		return errors.New("historical checklist relationship conflict")
	}
	return rows.Err()
}

func sameHistoricalJSON(left, right []byte) bool {
	var leftValue, rightValue any
	return json.Unmarshal(left, &leftValue) == nil && json.Unmarshal(right, &rightValue) == nil &&
		reflect.DeepEqual(leftValue, rightValue)
}

func Reset(ctx context.Context, pool *database.Pool, now time.Time) error {
	if pool == nil || now.IsZero() {
		return errors.New("canonical test reset requires database and server time")
	}
	now = now.UTC()
	allowedAnswers := []string{"COMPLIANT", "NON_COMPLIANT", "OBSERVATION", "NOT_APPLICABLE", "NOT_CHECKED"}
	commentRequiredFor := []string{"NON_COMPLIANT", "OBSERVATION"}
	question := func(id, sectionID, prompt, expectedEvidence string, assigned []string) canonicalQuestion {
		return canonicalQuestion{
			ID: id, SectionID: sectionID, Prompt: prompt,
			RegulatoryReference: fmt.Sprintf("Configured Cabin Inspection reference — %s", sectionID),
			ExpectedEvidence:    expectedEvidence,
			AllowedAnswers:      append([]string(nil), allowedAnswers...),
			CommentRequiredFor:  append([]string(nil), commentRequiredFor...),
			Assigned:            append([]string(nil), assigned...),
		}
	}
	questions := []canonicalQuestion{
		question("CAB-GALLEY-001", "GALLEY", "Are galley restraints and stowage areas serviceable and secure?", "Inspector observation and required exception comment", []string{"USR-INSPECTOR-DAVID"}),
		question("CAB-LAV-001", "LAV", "Are lavatory safety equipment and placards present and serviceable?", "Inspector observation and required exception comment", []string{CanonicalInspectorSubjectID}),
		question("CAB-PAX-SEAT-001", "PAX SEAT", "Are passenger seats, belts, and adjacent fittings serviceable?", "Inspector observation and required exception comment", []string{CanonicalInspectorSubjectID}),
		question("CAB-EMEQ-PBE-001", "EM EQ / PBE", "Is the PBE installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?", "PBE serviceability record and cabin position confirmation", []string{CanonicalInspectorSubjectID}),
		question("CAB-VID-CREW-SEAT-001", "VID+CREW SEAT", "Are cabin information displays and crew seats serviceable?", "Inspector observation and required exception comment", []string{CanonicalInspectorSubjectID}),
		question("CAB-COCKPIT-GEN-001", "COCKPIT+CAB GEN COND+EXITS", "Are cabin general condition and emergency exits satisfactory?", "Inspector observation and required exception comment", []string{CanonicalInspectorSubjectID}),
	}
	canonicalCatalogID := "catalog-canonical-test"
	canonicalCatalogVersion := "aga-canonical-test@1.0.0"
	canonicalProviderScopeID := "SCOPE-OPS-AOC-SOURCE-BOUND"
	canonicalRegulatedTargetID := "TARGET-OPS-AOC-SOURCE-BOUND"
	canonicalQuestionVersionIDs := make([]string, 0, len(questions))
	for _, currentQuestion := range questions {
		canonicalQuestionVersionIDs = append(canonicalQuestionVersionIDs, "QV-"+currentQuestion.ID+"-V2")
	}
	canonicalSelectionDigest := questioncatalog.SelectionDigest(canonicalQuestionVersionIDs)
	snapshot, err := json.Marshal(map[string]any{
		"schemaVersion": 1, "protocolVersion": 1,
		"creatorSubjectId": "USR-MANAGER-NORA",
		"changeReason":     "Initial immutable published Cabin Inspection version.",
		"riskFocus": []string{
			"Emergency equipment serviceability",
			"PBE serviceability",
			"Cabin inspection CAP follow-up",
		},
		"questions": questions,
	})
	if err != nil {
		return err
	}
	planningDraftValues := map[string]any{
		"organizationId": "ORG-FLY-NAMIBIA", "organizationName": "Fly Namibia",
		"applicationType": "CABIN", "domain": "Cabin Safety",
		"inspectionCategory": "Routine / Announced", "noticePolicy": "ADVANCE",
		"purpose": "", "triggerType": "Department Manager initiated",
		"riskCategory": "", "plannedDate": "2026-12-10", "mode": "On-site",
		"location": "", "catalogVersion": canonicalCatalogVersion,
		"scopeDraftId":               "scope-draft-PLAN-DRAFT-2026-001",
		"selectionDigest":            canonicalSelectionDigest,
		"selectedQuestionVersionIds": canonicalQuestionVersionIDs,
		"providerScopeId":            canonicalProviderScopeID, "regulatedTargetId": canonicalRegulatedTargetID,
		"requestedBudget": 0, "currency": "USD",
	}
	planningDraftSnapshot, err := json.Marshal(planningDraftValues)
	if err != nil {
		return err
	}
	coordinationValues := map[string]any{
		"organizationId": "ORG-FLY-NAMIBIA", "organizationName": "Fly Namibia",
		"applicationType": "CABIN", "domain": "Cabin Safety",
		"inspectionCategory": "Routine / Announced", "noticePolicy": "ADVANCE",
		"purpose": "Annual routine oversight", "triggerType": "Annual Plan",
		"riskCategory": "Cabin Safety", "plannedDate": "2026-06-15", "mode": "On-site",
		"location": "Windhoek", "catalogVersion": canonicalCatalogVersion,
		"scopeDraftId":               "scope-draft-PLAN-DRAFT-COORDINATION",
		"selectionDigest":            canonicalSelectionDigest,
		"selectedQuestionVersionIds": canonicalQuestionVersionIDs,
		"providerScopeId":            canonicalProviderScopeID, "regulatedTargetId": canonicalRegulatedTargetID,
		"requestedBudget": 48000, "currency": "USD",
		"preparedAuditId": CanonicalAuditID,
	}
	coordinationSnapshot, err := json.Marshal(coordinationValues)
	if err != nil {
		return err
	}
	emptyTemplateSnapshot, err := json.Marshal(map[string]any{
		"schemaVersion": 1, "protocolVersion": 1,
		"creatorSubjectId": "USR-MANAGER-NORA",
		"changeReason":     "Initial immutable published Flight Operations version.",
		"questions":        []canonicalQuestion{},
	})
	if err != nil {
		return err
	}
	canonicalScopeSnapshotValues := map[string]any{
		"draftId": "PLAN-DRAFT-COORDINATION", "organizationId": "ORG-FLY-NAMIBIA", "organizationName": "Fly Namibia",
		"applicationType": "CABIN", "domain": "Cabin Safety", "inspectionCategory": "Routine / Announced",
		"purpose": "Annual routine oversight", "triggerType": "Annual Plan", "riskCategory": "Cabin Safety",
		"plannedDate": "2026-06-15", "mode": "On-site", "location": "Windhoek",
		"providerScopeId": canonicalProviderScopeID, "regulatedTargetId": canonicalRegulatedTargetID,
		"catalogVersion": canonicalCatalogVersion, "usageClass": "PREPROD_EXERCISE",
		"selectionDigest": canonicalSelectionDigest, "selectedQuestionVersionIds": canonicalQuestionVersionIDs,
		"estimatedResourceRequirement": float64(len(questions)),
		"formDistribution":             map[string]any{"CABIN": len(questions)},
		"domainDistribution":           map[string]any{"Cabin Safety": len(questions)},
		"requestedBudget":              48000, "currency": "USD", "noticePolicy": "ADVANCE",
	}
	canonicalScopeSnapshot, err := json.Marshal(canonicalScopeSnapshotValues)
	if err != nil {
		return err
	}
	canonicalScopePlanningDigestBytes := sha256.Sum256(canonicalScopeSnapshot)
	canonicalScopePlanningDigest := "sha256:" + fmt.Sprintf("%x", canonicalScopePlanningDigestBytes[:])
	packageScopeSnapshotValues := map[string]any{
		"draftId": "PLAN-DRAFT-PACKAGE-001", "organizationId": "ORG-FLY-NAMIBIA", "organizationName": "Fly Namibia",
		"applicationType": "CABIN", "domain": "Cabin Safety", "inspectionCategory": "Routine / Announced",
		"purpose": "Canonical execution package fixture", "triggerType": "Canonical test profile",
		"riskCategory": "Cabin Safety", "plannedDate": "2026-06-15", "mode": "On-site", "location": "Windhoek",
		"preparedAuditId": CanonicalAuditID,
		"providerScopeId": canonicalProviderScopeID, "regulatedTargetId": canonicalRegulatedTargetID,
		"catalogVersion": canonicalCatalogVersion, "usageClass": "PREPROD_EXERCISE",
		"selectionDigest": canonicalSelectionDigest, "selectedQuestionVersionIds": canonicalQuestionVersionIDs,
		"estimatedResourceRequirement": float64(len(questions)),
		"formDistribution":             map[string]any{"CABIN": len(questions)},
		"domainDistribution":           map[string]any{"Cabin Safety": len(questions)},
		"requestedBudget":              48000, "currency": "USD", "noticePolicy": "ADVANCE",
	}
	packageScopeSnapshot, err := json.Marshal(packageScopeSnapshotValues)
	if err != nil {
		return err
	}
	packageScopePlanningDigestBytes := sha256.Sum256(packageScopeSnapshot)
	packageScopePlanningDigest := "sha256:" + fmt.Sprintf("%x", packageScopePlanningDigestBytes[:])
	selectionIDsJSON, err := json.Marshal(canonicalQuestionVersionIDs)
	if err != nil {
		return err
	}
	if err := retryCanonicalReset(ctx, func() error {
		return database.WithinTransaction(ctx, pool, func(ctx context.Context, transaction pgx.Tx) error {
			if _, err := transaction.Exec(ctx, `
			TRUNCATE TABLE
				command_transaction_links, audit_question_assignments, audit_team_members,
				audit_assignments, planning_intake_drafts,
				template_draft_versions, template_version_questions, template_masters,
				question_versions, regulatory_reference_versions,
				reminder_rules, surveillance_plan_items,
				oidc_login_states, idempotency_responses, authorized_sync_changes, sync_cursors, sync_cursor_tokens,
				audit_events, outbox_messages, review_decisions, report_decisions,
				report_approval_states, report_versions, evidence_version_states, evidence_versions,
				inspection_attachments, upload_sessions, object_metadata, cap_revisions, findings,
				potential_findings, checklist_responses, offline_grants, inspection_checklists,
				inspection_question_assignments, inspection_packages, checklist_template_versions,
				inspections, session_references, identity_references, organizations
			RESTART IDENTITY CASCADE
		`); err != nil {
				return fmt.Errorf("truncate canonical test state: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO organizations (id, legal_name, organization_type, status, created_at, updated_at) VALUES
				('CAA', 'Civil Aviation Authority', 'AUTHORITY', 'ACTIVE', $1, $1),
				('ORG-FLY-NAMIBIA', 'Fly Namibia', 'OPERATOR', 'ACTIVE', $1, $1),
				('ORG-SKYCARGO', 'SkyCargo Air', 'OPERATOR', 'ACTIVE', $1, $1)
		`, now); err != nil {
				return fmt.Errorf("seed canonical organizations: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO identity_references (
				subject_id, issuer, display_name, email, created_at
			) VALUES
				($2, 'urn:avia:test', 'Local Inspector', 'local.inspector@example.test', $1),
				('USR-INSPECTOR-DAVID', 'urn:avia:test', 'David Inspector', 'david.inspector@example.test', $1),
				('USR-LEAD-CANER', 'urn:avia:test', 'Caner Lead Inspector', 'caner.lead@example.test', $1),
				('USR-MANAGER-NORA', 'urn:avia:test', 'Nora Department Manager', 'nora.manager@example.test', $1),
				('USR-FINANCE-LINA', 'urn:avia:test', 'Lina Finance Reviewer', 'lina.finance@example.test', $1),
				('USR-GM-OMAR', 'urn:avia:test', 'Omar General Manager', 'omar.gm@example.test', $1),
				('USR-ED-ZARA', 'urn:avia:test', 'Zara Executive Director', 'zara.executive@example.test', $1),
				('USR-ADMIN-ADA', 'urn:avia:test', 'Ada Administrator', 'ada.admin@example.test', $1),
				('USR-AUDITEE-FLY', 'urn:avia:test', 'Fly Namibia Auditee', 'auditee.fly@example.test', $1)
		`, now, CanonicalInspectorSubjectID); err != nil {
				return fmt.Errorf("seed canonical identities: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO user_profiles (
				subject_id, display_name, organization_id, revision, created_at, updated_at
			) VALUES
				($2, 'Local Inspector', 'CAA', 1, $1, $1),
				('USR-INSPECTOR-DAVID', 'David Inspector', 'CAA', 1, $1, $1),
				('USR-LEAD-CANER', 'Caner Lead Inspector', 'CAA', 1, $1, $1),
				('USR-MANAGER-NORA', 'Nora Department Manager', 'CAA', 1, $1, $1),
				('USR-FINANCE-LINA', 'Lina Finance Reviewer', 'CAA', 1, $1, $1),
				('USR-GM-OMAR', 'Omar General Manager', 'CAA', 1, $1, $1),
				('USR-ED-ZARA', 'Zara Executive Director', 'CAA', 1, $1, $1),
				('USR-ADMIN-ADA', 'Ada Administrator', 'CAA', 1, $1, $1),
				('USR-AUDITEE-FLY', 'Fly Namibia Auditee', 'ORG-FLY-NAMIBIA', 1, $1, $1)
		`, now, CanonicalInspectorSubjectID); err != nil {
				return fmt.Errorf("seed canonical profiles: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO caa_department_memberships
				(id, root_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from)
			VALUES
				('TEST-USR-MANAGER-NORA-FOI', 'TEST-USR-MANAGER-NORA-FOI',
				 'USR-MANAGER-NORA', 'FLIGHT_OPERATIONS_INSPECTORATE',
				 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01')
		`); err != nil {
				return fmt.Errorf("seed canonical department-manager authority: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO regulated_targets (id, target_kind, organization_id, created_at)
			VALUES ($1, 'ORGANIZATION', 'ORG-FLY-NAMIBIA', $2)
		`, canonicalRegulatedTargetID, now); err != nil {
				return fmt.Errorf("seed canonical regulated target: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO organization_service_provider_scopes (
				id, organization_id, service_provider_type_id, authorization_identifier,
				status, effective_from, primary_target_id, created_at
			) VALUES ($1, 'ORG-FLY-NAMIBIA', 'AIR_OPERATOR', 'AOC-FLY-NAMIBIA-SOURCE-BOUND',
				'ACTIVE', '2025-01-01', $2, $3)
		`, canonicalProviderScopeID, canonicalRegulatedTargetID, now); err != nil {
				return fmt.Errorf("seed canonical provider scope: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO organization_service_provider_scope_targets (
				organization_service_provider_scope_id, regulated_target_id, created_at
			) VALUES ($1, $2, $3)
		`, canonicalProviderScopeID, canonicalRegulatedTargetID, now); err != nil {
				return fmt.Errorf("seed canonical provider target applicability: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO user_settings (
				subject_id, notification_preferences, locale, timezone, revision, updated_at
			)
			SELECT subject_id, '{}'::jsonb, 'en', 'UTC', 1, $1
			FROM identity_references
		`, now); err != nil {
				return fmt.Errorf("seed canonical settings: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO session_references (
				id, subject_id, organization_id, expires_at, last_seen_at, absolute_expires_at, roles, created_at
			) VALUES
				('TEST-CANONICAL-INSPECTOR', $3, 'CAA', $2, $1, $2, ARRAY['inspector'], $1),
				('TEST-USR-INSPECTOR-DAVID', 'USR-INSPECTOR-DAVID', 'CAA', $2, $1, $2, ARRAY['inspector'], $1),
				('TEST-USR-LEAD-CANER', 'USR-LEAD-CANER', 'CAA', $2, $1, $2, ARRAY['leadInspector'], $1),
				('TEST-USR-MANAGER-NORA', 'USR-MANAGER-NORA', 'CAA', $2, $1, $2, ARRAY['manager'], $1),
				('TEST-USR-FINANCE-LINA', 'USR-FINANCE-LINA', 'CAA', $2, $1, $2, ARRAY['finance'], $1),
				('TEST-USR-GM-OMAR', 'USR-GM-OMAR', 'CAA', $2, $1, $2, ARRAY['gm'], $1),
				('TEST-USR-ED-ZARA', 'USR-ED-ZARA', 'CAA', $2, $1, $2, ARRAY['executiveDirector'], $1),
				('TEST-USR-ADMIN-ADA', 'USR-ADMIN-ADA', 'CAA', $2, $1, $2, ARRAY['admin'], $1),
				('TEST-USR-AUDITEE-FLY', 'USR-AUDITEE-FLY', 'ORG-FLY-NAMIBIA', $2, $1, $2, ARRAY['auditee'], $1)
		`, now, now.Add(8*time.Hour), CanonicalInspectorSubjectID); err != nil {
				return fmt.Errorf("seed canonical sessions: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO inspections (
				id, organization_id, assigned_inspector_subject_id, title, inspection_type, status,
				due_date, revision, created_at, updated_at
			) VALUES
				('AUD-2026-001', 'ORG-FLY-NAMIBIA', $2,
					 '2026 Cabin Inspection - Fly Namibia', 'CABIN', 'IN_PROGRESS', '2026-06-18', 1, $1, $1),
				('AUD-2026-099', 'ORG-SKYCARGO', 'USR-INSPECTOR-DAVID',
				 '2026 Cargo Inspection - SkyCargo Air', 'CARGO', 'IN_PROGRESS', '2026-07-30', 1, $1, $1)
		`, now, CanonicalInspectorSubjectID); err != nil {
				return fmt.Errorf("seed canonical Audits: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO surveillance_plan_items (
				id, title, plan_year, organization_id, inspection_type, scheduled_date,
				estimated_budget, status, current_owner_role, next_action, revision, created_at, updated_at
			) VALUES (
				'PLAN-2026-CAB-001', '2026 Cabin Surveillance — Fly Namibia', 2026,
				'ORG-FLY-NAMIBIA', 'CABIN', '2026-07-15', 48000,
				'FINANCE_REVIEW', 'finance', 'Finance to review budget', 1, $1, $1
			)
		`, now); err != nil {
				return fmt.Errorf("seed canonical surveillance plan: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO reminder_rules (id, label, offset_days, channel, status, revision, created_at, updated_at) VALUES
				('REM-30', '30 days before Due Date', 30, 'IN_APP', 'ACTIVE', 1, $1, $1),
				('REM-15', '15 days before Due Date', 15, 'IN_APP', 'ACTIVE', 1, $1, $1),
				('REM-7', '7 days before Due Date', 7, 'IN_APP', 'ACTIVE', 1, $1, $1),
				('REM-DUE', 'On the Due Date', 0, 'IN_APP', 'ACTIVE', 1, $1, $1),
				('REM-OVERDUE', 'After the Due Date', -1, 'IN_APP', 'ACTIVE', 1, $1, $1)
		`, now); err != nil {
				return fmt.Errorf("seed canonical reminder rules: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO regulatory_reference_versions (
				id, reference_id, version, title, status, effective_date, snapshot, created_at
			) VALUES
				('NAMCARS-CAB-001-V1', 'NAMCARS-CAB-001', 1,
					'Configured Cabin Safety reference', 'ACTIVE', '2026-01-01',
					$2,
					$1),
				('NAMCARS-FOPS-004-V1', 'NAMCARS-FOPS-004', 1,
					'Configured Flight Operations reference', 'SUPERSEDED', '2025-10-01',
					'{"versionLabel":"2025.4","configuredRules":["Reference-only Flight Operations sampling metadata"],"changeHistory":["2026-01-01 — Superseded in demo data"],"mappings":[]}',
					$1)
		`, now, []byte(opsAOCCabinRampRegulatorySnapshot)); err != nil {
				return fmt.Errorf("seed canonical regulatory references: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO report_definition_versions (
				id, definition_id, version, title, description, definition,
				created_by_subject_id, created_at
			) VALUES (
				'ADMIN-RPT-PACKAGE-001-V1', 'ADMIN-RPT-PACKAGE-001', 1,
				'Inspection package configuration preview',
				'Typed mock report definition; this is not a real report or PDF engine.',
				'{
					"packageFields":[
						"packageId","auditId","organizationId","questionIds",
						"configuredReferences","expectedEvidence","riskFocus"
					],
					"actionReason":"ADMIN-RPT-PACKAGE-001 generation is unavailable because Task 10 provides a typed browser-local preview only."
				}',
				'USR-ADMIN-ADA', $1
			)
		`, now); err != nil {
				return fmt.Errorf("seed canonical report definition: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO checklist_template_versions (id, template_id, version, title, snapshot, published_at)
			VALUES
				('CTV-CABIN-1', 'CABIN', 1, 'Cabin Inspection checklist', $1, $3),
				('CTV-FOPS-1', 'FOPS', 1, 'Flight Operations checklist', $2, $3)
		`, snapshot, emptyTemplateSnapshot, now); err != nil {
				return fmt.Errorf("seed canonical checklist template: %w", err)
			}
			for position, question := range questions {
				questionVersionID := "QV-" + question.ID + "-V1"
				if _, err := transaction.Exec(ctx, `
				INSERT INTO question_versions (
					id, question_id, version, prompt, configured_reference,
					expected_evidence, created_by_subject_id, created_at
				) VALUES ($1, $2, 1, $3, $4, $5, 'USR-MANAGER-NORA', $6)
			`, questionVersionID, question.ID, question.Prompt,
					question.RegulatoryReference, question.ExpectedEvidence, now); err != nil {
					return fmt.Errorf("seed Question version %s: %w", question.ID, err)
				}
				if _, err := transaction.Exec(ctx, `
				INSERT INTO template_version_questions (
					template_version_id, question_version_id, position, created_at
				) VALUES ('CTV-CABIN-1', $1, $2, $3)
			`, questionVersionID, position, now); err != nil {
					return fmt.Errorf("seed Template Question %s: %w", question.ID, err)
				}
			}
			// V1 remains bound to the historical published template. The
			// canonical PREPROD_EXERCISE catalog uses separate immutable V2
			// question-version identities so exercise provenance cannot be
			// reused by an administrative template draft.
			for _, question := range questions {
				questionVersionID := "QV-" + question.ID + "-V2"
				if _, err := transaction.Exec(ctx, `
				INSERT INTO question_versions (
					id, question_id, version, prompt, configured_reference,
					expected_evidence, created_by_subject_id, created_at
				) VALUES ($1, $2, 2, $3, $4, $5, 'USR-MANAGER-NORA', $6)
			`, questionVersionID, question.ID, question.Prompt,
					question.RegulatoryReference, question.ExpectedEvidence, now); err != nil {
					return fmt.Errorf("seed Exercise Question version %s: %w", question.ID, err)
				}
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO canonical_question_catalogs (
				id, catalog_version, usage_class, profile_name, profile_version, status,
				source_package_version, source_package_json_sha256, source_package_zip_sha256,
				root_digest, question_count, form_count, created_by_subject_id, created_at
			) VALUES ($1, $2, 'PREPROD_EXERCISE', 'aga-preprod', '1.0.0', 'SEALED',
				'canonical-test-fixture@1.0.0', 'sha256:canonical-test-package-json',
				'sha256:canonical-test-package-zip', 'sha256:canonical-test-catalog-root',
				$3, 1, 'USR-MANAGER-NORA', $4)
		`, canonicalCatalogID, canonicalCatalogVersion, len(questions), now); err != nil {
				return fmt.Errorf("seed canonical question catalog: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO canonical_question_catalog_forms (
				catalog_id, form_code, form_digest, archive_digest, question_count,
				source_gap_state, created_at
			) VALUES ($1, 'CABIN', 'sha256:canonical-test-form',
				'sha256:canonical-test-archive', $2, 'SOURCE_MAPPING_REQUIRED', $3)
		`, canonicalCatalogID, len(questions), now); err != nil {
				return fmt.Errorf("seed canonical question form: %w", err)
			}
			for position, currentQuestion := range questions {
				questionVersionID := "QV-" + currentQuestion.ID + "-V2"
				if _, err := transaction.Exec(ctx, `
				INSERT INTO canonical_question_version_provenance (
					question_version_id, usage_class, catalog_id, recorded_at
				) VALUES ($1, 'PREPROD_EXERCISE', $2, $3)
			`, questionVersionID, canonicalCatalogID, now); err != nil {
					return fmt.Errorf("seed canonical question provenance %s: %w", currentQuestion.ID, err)
				}
				if _, err := transaction.Exec(ctx, `
				INSERT INTO canonical_question_catalog_memberships (
					catalog_id, question_version_id, usage_class, form_code, proposal_id,
					ordinal, question_digest, source_locator, source_gap_state,
					proposed_domain, proposed_topic, proposed_risk_band, created_at
				) VALUES ($1, $2, 'PREPROD_EXERCISE', 'CABIN', $3, $4, $5, $6,
					'SOURCE_MAPPING_REQUIRED', 'Cabin Safety', $7, 'MEDIUM', $8)
				`, canonicalCatalogID, questionVersionID, currentQuestion.ID, position+1,
					"sha256:canonical-question-"+fmt.Sprintf("%02d", position+1),
					"fixture://canonical-cabin/"+currentQuestion.ID, currentQuestion.SectionID,
					now); err != nil {
					return fmt.Errorf("seed canonical question membership %s: %w", currentQuestion.ID, err)
				}
				if _, err := transaction.Exec(ctx, `
				INSERT INTO canonical_question_catalog_membership_events (
					event_id, catalog_id, question_version_id, status, reason,
					actor_subject_id, occurred_at
				) VALUES ($1, $2, $3, 'AVAILABLE', 'canonical local exercise fixture',
					'USR-MANAGER-NORA', $4)
			`, "canonical-membership:"+currentQuestion.ID+":available", canonicalCatalogID, questionVersionID, now); err != nil {
					return fmt.Errorf("seed canonical question availability %s: %w", currentQuestion.ID, err)
				}
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO canonical_question_catalog_applicabilities (
				catalog_id, question_version_id, provider_scope_id, regulated_target_id,
				status, reason, actor_subject_id, created_at
			)
			SELECT $1, membership.question_version_id, $2, $3, 'ELIGIBLE',
				'canonical local exercise fixture applicability', 'USR-MANAGER-NORA', $4
			FROM canonical_question_catalog_memberships membership
			WHERE membership.catalog_id = $1
		`, canonicalCatalogID, canonicalProviderScopeID, canonicalRegulatedTargetID, now); err != nil {
				return fmt.Errorf("seed canonical question applicability: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO template_masters (
				id, title, owner_role, published_template_version_id,
				revision, created_at, updated_at
			) VALUES
				('TPL-CABIN-2026', 'Cabin Inspection checklist', 'Department Manager',
					'CTV-CABIN-1', 1, $1, $1),
				('TPL-FOPS-2026', 'Flight Operations checklist', 'Department Manager',
					'CTV-FOPS-1', 1, $1, $1)
		`, now); err != nil {
				return fmt.Errorf("seed Template masters: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO planning_intake_drafts (
				id, organization_id, values, submitted_planning_item_id, revision,
				created_by_subject_id, created_at, updated_at
			) VALUES
				('PLAN-DRAFT-2026-001', 'ORG-FLY-NAMIBIA', $1, NULL, 1,
					'USR-MANAGER-NORA', $4, $4),
				('PLAN-DRAFT-COORDINATION', 'ORG-FLY-NAMIBIA', $2,
					'PLAN-2026-CAB-001', 2, 'USR-MANAGER-NORA', $4, $4),
				('PLAN-DRAFT-PACKAGE-001', 'ORG-FLY-NAMIBIA', $3, NULL, 1,
					'USR-MANAGER-NORA', $4, $4)
		`, planningDraftSnapshot, coordinationSnapshot, packageScopeSnapshot, now); err != nil {
				return fmt.Errorf("seed canonical Planning intake drafts: %w", err)
			}
			for _, scopeFixture := range []struct {
				id                 string
				draftID            string
				status             string
				revision           int
				requestedBudget    float64
				snapshotID         string
				snapshotStage      string
				snapshot           []byte
				planningDigest     string
				selectionOperation string
			}{
				{
					id: "scope-draft-PLAN-DRAFT-2026-001", draftID: "PLAN-DRAFT-2026-001",
					status: "DRAFT", revision: 1, requestedBudget: 0,
					selectionOperation: "canonical-select:PLAN-DRAFT-2026-001",
				},
				{
					id: "scope-draft-PLAN-DRAFT-COORDINATION", draftID: "PLAN-DRAFT-COORDINATION",
					status: "SUBMITTED", revision: 2, requestedBudget: 48000,
					snapshotID:    "scope-snapshot:PLAN-DRAFT-COORDINATION:submitted:2",
					snapshotStage: "SUBMITTED", snapshot: canonicalScopeSnapshot,
					planningDigest:     canonicalScopePlanningDigest,
					selectionOperation: "canonical-select:PLAN-DRAFT-COORDINATION",
				},
				{
					id: "scope-draft-package-001", draftID: "PLAN-DRAFT-PACKAGE-001",
					status: "RELEASED", revision: 1, requestedBudget: 48000,
					snapshotID: "scope-snapshot-package-001", snapshotStage: "RELEASED",
					snapshot: packageScopeSnapshot, planningDigest: packageScopePlanningDigest,
				},
			} {
				if _, err := transaction.Exec(ctx, `
				INSERT INTO canonical_audit_scope_drafts (
					id, planning_intake_draft_id, organization_id, provider_scope_id,
					regulated_target_id, audit_type, catalog_id, usage_class, revision,
					status, selected_question_count, selection_digest, requested_budget,
					notice_policy, created_by_subject_id, created_at, updated_at
				) VALUES ($1, $2, 'ORG-FLY-NAMIBIA', $3, $4, 'CABIN', $5,
					'PREPROD_EXERCISE', $6, $7, $8, $9, $10, 'ADVANCE',
					'USR-MANAGER-NORA', $11, $11)
				`, scopeFixture.id, scopeFixture.draftID, canonicalProviderScopeID,
					canonicalRegulatedTargetID, canonicalCatalogID, scopeFixture.revision,
					scopeFixture.status, len(canonicalQuestionVersionIDs), canonicalSelectionDigest,
					scopeFixture.requestedBudget, now); err != nil {
					return fmt.Errorf("seed canonical scope draft %s: %w", scopeFixture.id, err)
				}
				for position, questionVersionID := range canonicalQuestionVersionIDs {
					if _, err := transaction.Exec(ctx, `
					INSERT INTO canonical_audit_scope_draft_questions (
						scope_draft_id, revision, catalog_id, question_version_id,
						position, selection_digest, created_at
					) VALUES ($1, $2, $3, $4, $5, $6, $7)
					`, scopeFixture.id, scopeFixture.revision, canonicalCatalogID,
						questionVersionID, position, canonicalSelectionDigest, now); err != nil {
						return fmt.Errorf("seed canonical scope question %s: %w", scopeFixture.id, err)
					}
				}
				if scopeFixture.selectionOperation != "" {
					if _, err := transaction.Exec(ctx, `
					INSERT INTO canonical_audit_scope_selection_operations (
						id, scope_draft_id, operation_id, idempotency_key, operation_kind,
						expected_digest, result_digest, affected_question_version_ids,
						filter_payload, actor_subject_id, created_at
					) VALUES ($1, $2, $1, $1, 'REPLACE', $3, $3, $4::jsonb,
						'{}'::jsonb, 'USR-MANAGER-NORA', $5)
					`, scopeFixture.selectionOperation, scopeFixture.id, canonicalSelectionDigest,
						selectionIDsJSON, now); err != nil {
						return fmt.Errorf("seed canonical selection operation %s: %w", scopeFixture.id, err)
					}
					for position, questionVersionID := range canonicalQuestionVersionIDs {
						if _, err := transaction.Exec(ctx, `
						INSERT INTO canonical_audit_scope_selection_questions (
							operation_id, catalog_id, question_version_id, position,
							selection_digest, created_at
						) VALUES ($1, $2, $3, $4, $5, $6)
						`, scopeFixture.selectionOperation, canonicalCatalogID,
							questionVersionID, position, canonicalSelectionDigest, now); err != nil {
							return fmt.Errorf("seed canonical selection question %s: %w", scopeFixture.id, err)
						}
					}
				}
				if scopeFixture.snapshotID != "" {
					if _, err := transaction.Exec(ctx, `
					INSERT INTO canonical_audit_scope_snapshots (
						id, scope_draft_id, revision, stage, catalog_id, usage_class,
						selection_digest, planning_snapshot_digest, selected_question_count,
						snapshot, created_by_subject_id, created_at
					) VALUES ($1, $2, $3, $4, $5, 'PREPROD_EXERCISE', $6, $7, $8, $9,
						'USR-MANAGER-NORA', $10)
					`, scopeFixture.snapshotID, scopeFixture.id, scopeFixture.revision,
						scopeFixture.snapshotStage, canonicalCatalogID, canonicalSelectionDigest,
						scopeFixture.planningDigest, len(canonicalQuestionVersionIDs),
						scopeFixture.snapshot, now); err != nil {
						return fmt.Errorf("seed canonical scope snapshot %s: %w", scopeFixture.snapshotID, err)
					}
					for position, questionVersionID := range canonicalQuestionVersionIDs {
						if _, err := transaction.Exec(ctx, `
						INSERT INTO canonical_audit_scope_snapshot_questions (
							snapshot_id, catalog_id, question_version_id, position
						) VALUES ($1, $2, $3, $4)
						`, scopeFixture.snapshotID, canonicalCatalogID, questionVersionID, position); err != nil {
							return fmt.Errorf("seed canonical snapshot question %s: %w", scopeFixture.snapshotID, err)
						}
					}
				}
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO inspection_packages (
				id, inspection_id, checklist_template_version_id, canonical_scope_snapshot_id,
				package_version, snapshot, expires_at, created_at, package_digest
			) VALUES ('PKG-CAB-2026-001', 'AUD-2026-001', NULL, 'scope-snapshot-package-001', 1, $1, $2, $3,
				'sha256:candidate-cabin-package-v1')
		`, snapshot, now.Add(72*time.Hour), now); err != nil {
				return fmt.Errorf("seed canonical inspection package: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_assignments (
				id, inspection_id, planning_item_id, released_scope_snapshot_id,
				organization_id, lead_subject_id, status, scheduled_start_date,
				scheduled_end_date, revision, created_at, updated_at
			) VALUES (
				'ASSIGN-AUD-2026-001', 'AUD-2026-001', 'PLAN-2026-CAB-001',
				'scope-snapshot-package-001', 'ORG-FLY-NAMIBIA',
				'USR-LEAD-CANER', 'AWAITING_AUDITEE_CONFIRMATION',
				'2026-06-15', '2026-06-18', 1, $1, $1
			)
		`, now); err != nil {
				return fmt.Errorf("seed canonical Audit assignment: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_team_members (
				assignment_id, subject_id, member_role, revision, created_at
			) VALUES
				('ASSIGN-AUD-2026-001', 'USR-LEAD-CANER', 'LEAD_INSPECTOR', 1, $1),
				('ASSIGN-AUD-2026-001', $2, 'INSPECTOR', 1, $1),
				('ASSIGN-AUD-2026-001', 'USR-INSPECTOR-DAVID', 'INSPECTOR', 1, $1)
		`, now, CanonicalInspectorSubjectID); err != nil {
				return fmt.Errorf("seed canonical Audit team: %w", err)
			}
			for _, question := range questions {
				if _, err := transaction.Exec(ctx, `
				INSERT INTO inspection_question_assignments (inspection_id, question_id, subject_id, assignment_revision)
				VALUES ('AUD-2026-001', $1, $2, 1)
			`, question.ID, question.Assigned[0]); err != nil {
					return fmt.Errorf("seed assignment %s: %w", question.ID, err)
				}
				if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_question_assignments (
					assignment_id, question_id, subject_id, revision, created_at
				) VALUES ('ASSIGN-AUD-2026-001', $1, $2, 1, $3)
			`, question.ID, question.Assigned[0], now); err != nil {
					return fmt.Errorf("seed Audit question assignment %s: %w", question.ID, err)
				}
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO inspection_checklists (inspection_id, status, revision)
			VALUES ('AUD-2026-001', 'IN_PROGRESS', 1)
		`); err != nil {
				return fmt.Errorf("seed canonical checklist: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO findings (
				id, reference, inspection_id, organization_id, severity, status, owner_subject_id,
				next_action, due_date, revision, cap_required, evidence_required, issued_at, created_at, updated_at
			) VALUES ('FND-SKYCARGO-2026-099', 'CAR-2026-099', 'AUD-2026-099', 'ORG-SKYCARGO',
				'LEVEL_2_MAJOR', 'OPEN', NULL, 'SkyCargo Air to submit CAP', '2026-06-10', 1,
				true, true, $1, $1, $1)
		`, now); err != nil {
				return fmt.Errorf("seed isolation Finding: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO cap_revisions (
				id, cap_id, finding_id, organization_id, revision, status,
				root_cause, corrective_action, preventive_action,
				target_completion_date, submitted_by_subject_id, submitted_at,
				responsible_person, comment_to_caa
			) VALUES
				(
					'CAP-CAR-2026-099-R1', 'CAP-CAR-2026-099',
					'FND-SKYCARGO-2026-099', 'ORG-SKYCARGO', 1, 'SUPERSEDED',
					'Initial configured root cause.',
					'Initial configured corrective action.',
					'Initial configured preventive action.',
					'2026-06-01', 'USR-AUDITEE-SKYCARGO', $1,
					'SkyCargo Safety Manager', 'Initial submission.'
				),
				(
					'CAP-CAR-2026-099-R2', 'CAP-CAR-2026-099',
					'FND-SKYCARGO-2026-099', 'ORG-SKYCARGO', 2,
					'MORE_INFORMATION_REQUESTED',
					'Updated configured root cause.',
					'Updated configured corrective action.',
					'Updated configured preventive action.',
					'2026-06-10', 'USR-AUDITEE-SKYCARGO', $1,
					'SkyCargo Safety Manager', 'Updated submission.'
				)
		`, now); err != nil {
				return fmt.Errorf("seed isolation CAP history: %w", err)
			}
			finalContent, finalContentHash := canonicalReportContent(
				"Final cabin safety surveillance report",
			)
			preliminaryV0Content, preliminaryV0ContentHash := canonicalReportContent(
				"Preliminary cabin safety surveillance report draft",
			)
			preliminaryV1Content, preliminaryV1ContentHash := canonicalReportContent(
				"Preliminary cabin safety surveillance report",
			)
			reportSnapshot, _ := json.Marshal(map[string]any{
				"kind": "FINAL", "ready": true,
				"findingIds": []string{}, "contentHash": finalContentHash,
				"content": finalContent, "createdBySubject": "USR-LEAD-CANER",
				"responseDueDate": nil, "caaVisibleComment": nil,
			})
			preliminaryV0Snapshot, _ := json.Marshal(map[string]any{
				"kind": "PRELIMINARY", "ready": false,
				"findingIds": []string{}, "contentHash": preliminaryV0ContentHash,
				"content": preliminaryV0Content, "createdBySubject": "USR-LEAD-CANER",
				"responseDueDate": nil, "caaVisibleComment": nil,
			})
			preliminaryV1Snapshot, _ := json.Marshal(map[string]any{
				"kind": "PRELIMINARY", "ready": true,
				"findingIds": []string{}, "contentHash": preliminaryV1ContentHash,
				"content": preliminaryV1Content, "createdBySubject": "USR-LEAD-CANER",
				"responseDueDate": nil, "caaVisibleComment": nil,
			})
			if _, err := transaction.Exec(ctx, `
			INSERT INTO report_versions (
				id, report_id, inspection_id, version, status, snapshot, created_at
			) VALUES
				('PR-2026-018-V0', 'PR-2026-018', 'AUD-2026-001', 0, 'RETURNED', $1, $3),
				('PR-2026-018-V1', 'PR-2026-018', 'AUD-2026-001', 1, 'DEPARTMENT_REVIEW', $2, $3)
		`, preliminaryV0Snapshot, preliminaryV1Snapshot, now); err != nil {
				return fmt.Errorf("seed preliminary report versions: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO report_approval_states (
				report_version_id, status, revision, updated_at
			) VALUES
				('PR-2026-018-V0', 'RETURNED', 1, $1),
				('PR-2026-018-V1', 'DEPARTMENT_REVIEW', 1, $1)
		`, now); err != nil {
				return fmt.Errorf("seed preliminary report approvals: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO report_versions (id, report_id, inspection_id, version, status, snapshot, created_at)
			VALUES ('RPT-CAB-2026-001-V1', 'RPT-CAB-2026-001', 'AUD-2026-001', 1,
				'DEPARTMENT_REVIEW', $1, $2)
		`, reportSnapshot, now); err != nil {
				return fmt.Errorf("seed canonical report version: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
			INSERT INTO report_approval_states (report_version_id, status, revision, updated_at)
			VALUES ('RPT-CAB-2026-001-V1', 'DEPARTMENT_REVIEW', 1, $1)
		`, now); err != nil {
				return fmt.Errorf("seed canonical report approval: %w", err)
			}
			return nil
		})
	}); err != nil {
		return err
	}
	if err := BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		return fmt.Errorf("bootstrap synthetic governed-checklist test inputs: %w", err)
	}
	if err := BootstrapBlockedRealOPSAOCGenerationInputs(ctx, pool); err != nil {
		return fmt.Errorf("bootstrap blocked real OPS/AOC test inputs: %w", err)
	}
	return nil
}

// ResetCoordination prepares the same canonical records in the pre-execution
// state required to exercise a successful announced-date confirmation. The
// ordinary canonical reset remains execution-ready because the offline grant
// contract requires both the inspection and checklist to be IN_PROGRESS.
func ResetCoordination(ctx context.Context, pool *database.Pool, now time.Time) error {
	if err := Reset(ctx, pool, now); err != nil {
		return err
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, transaction pgx.Tx) error {
		updates := []struct {
			name  string
			query string
		}{
			{
				name:  "inspection",
				query: `UPDATE inspections SET status = 'AWAITING_AUDITEE_CONFIRMATION', revision = 1, updated_at = $2 WHERE id = $1`,
			},
			{
				name:  "checklist",
				query: `UPDATE inspection_checklists SET status = 'NOT_STARTED', revision = 1 WHERE inspection_id = $1`,
			},
			{
				name:  "assignment",
				query: `UPDATE audit_assignments SET status = 'AWAITING_AUDITEE_CONFIRMATION', revision = 1, updated_at = $2 WHERE inspection_id = $1`,
			},
		}
		for _, update := range updates {
			var result pgconn.CommandTag
			var err error
			if update.name == "checklist" {
				result, err = transaction.Exec(ctx, update.query, CanonicalAuditID)
			} else {
				result, err = transaction.Exec(ctx, update.query, CanonicalAuditID, now)
			}
			if err != nil {
				return fmt.Errorf("reset coordination %s: %w", update.name, err)
			}
			if result.RowsAffected() != 1 {
				return fmt.Errorf("reset coordination %s affected %d rows", update.name, result.RowsAffected())
			}
		}
		return nil
	})
}

func canonicalReportContent(title string) (documents.ReportContent, string) {
	content := documents.ReportContent{
		Schema: documents.ReportContentSchema, LanguageTag: "en", Title: title,
		ExecutiveSummary: "Synthetic local surveillance narrative for deterministic verification.",
		Scope:            "Cabin safety surveillance for the authorized synthetic organization.",
		Methodology:      "Review of the immutable Audit, Finding, CAP, and Evidence records.",
		Sections: []documents.ReportSection{{
			ID: "overview", Heading: "Surveillance overview",
			Paragraphs: []string{"The report preserves the exact authorized synthetic record boundary."},
		}},
		Findings:        []documents.ReportFinding{},
		Conclusion:      "The synthetic report is ready for its configured approval state.",
		Recommendations: []string{"Continue configured local oversight verification."},
	}
	canonical, err := json.Marshal(content)
	if err != nil {
		panic("encode canonical test report content: " + err.Error())
	}
	digest := sha256.Sum256(canonical)
	return content, fmt.Sprintf("sha256:%x", digest)
}

func retryCanonicalReset(ctx context.Context, operation func() error) error {
	const maximumAttempts = 3

	for attempt := 1; attempt <= maximumAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}
		var postgresError *pgconn.PgError
		if !errors.As(err, &postgresError) ||
			postgresError.Code != "40P01" ||
			attempt == maximumAttempts {
			return err
		}
		timer := time.NewTimer(time.Duration(attempt) * 25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

func Principal(subjectID string) (identity.Principal, bool) {
	principals := map[string]identity.Principal{
		CanonicalInspectorSubjectID: {SubjectID: CanonicalInspectorSubjectID, DisplayName: "Local Inspector", OrganizationID: "CAA", SessionID: "TEST-CANONICAL-INSPECTOR", Roles: []identity.Role{identity.RoleInspector}},
		"USR-INSPECTOR-DAVID":       {SubjectID: "USR-INSPECTOR-DAVID", DisplayName: "David Inspector", OrganizationID: "CAA", SessionID: "TEST-USR-INSPECTOR-DAVID", Roles: []identity.Role{identity.RoleInspector}},
		"USR-LEAD-CANER":            {SubjectID: "USR-LEAD-CANER", DisplayName: "Caner Lead Inspector", OrganizationID: "CAA", SessionID: "TEST-USR-LEAD-CANER", Roles: []identity.Role{identity.RoleLeadInspector}},
		"USR-MANAGER-NORA":          {SubjectID: "USR-MANAGER-NORA", DisplayName: "Nora Department Manager", OrganizationID: "CAA", SessionID: "TEST-USR-MANAGER-NORA", Roles: []identity.Role{identity.RoleDepartmentManager}},
		"USR-TASK6-AIR-MANAGER":     {SubjectID: "USR-TASK6-AIR-MANAGER", DisplayName: "Task 6 AIR Manager", OrganizationID: "CAA", SessionID: "TEST-USR-TASK6-AIR-MANAGER", Roles: []identity.Role{identity.RoleDepartmentManager}},
		"USR-FINANCE-LINA":          {SubjectID: "USR-FINANCE-LINA", DisplayName: "Lina Finance Reviewer", OrganizationID: "CAA", SessionID: "TEST-USR-FINANCE-LINA", Roles: []identity.Role{identity.RoleFinance}},
		"USR-GM-OMAR":               {SubjectID: "USR-GM-OMAR", DisplayName: "Omar General Manager", OrganizationID: "CAA", SessionID: "TEST-USR-GM-OMAR", Roles: []identity.Role{identity.RoleGeneralManager}},
		"USR-ED-ZARA":               {SubjectID: "USR-ED-ZARA", DisplayName: "Zara Executive Director", OrganizationID: "CAA", SessionID: "TEST-USR-ED-ZARA", Roles: []identity.Role{identity.RoleExecutiveDirector}},
		"USR-ADMIN-ADA":             {SubjectID: "USR-ADMIN-ADA", DisplayName: "Ada Administrator", OrganizationID: "CAA", SessionID: "TEST-USR-ADMIN-ADA", Roles: []identity.Role{identity.RoleAdmin}},
		"USR-AUDITEE-FLY":           {SubjectID: "USR-AUDITEE-FLY", DisplayName: "Fly Namibia Auditee", OrganizationID: "ORG-FLY-NAMIBIA", SessionID: "TEST-USR-AUDITEE-FLY", Roles: []identity.Role{identity.RoleAuditee}},
	}
	principal, ok := principals[subjectID]
	return principal, ok
}
