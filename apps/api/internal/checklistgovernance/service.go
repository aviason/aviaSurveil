package checklistgovernance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	Pool  *database.Pool
	Clock func() time.Time
}

type ReviewCommand struct {
	OperationID           string
	IdempotencyKey        string
	CandidateID           string
	ExpectedRevision      int64
	ExpectedContentDigest string
	Reason                string
}

type PublicationCommand ReviewCommand

type PublicationView struct {
	TemplateVersionID      string
	PublicationDecisionID  string
	CandidateRootID        string
	CandidateID            string
	CandidateRevision      int64
	CandidateContentDigest string
	ActorSubjectID         string
	ActorMembershipID      string
	ActorDepartmentID      string
	ActorUnitID            string
	Reason                 string
	DecidedAt              time.Time
	PublishedAt            time.Time
	OperationID            string
	IdempotencyKey         string
	SemanticPayloadDigest  string
	AuditEventID           string
}

type PublishedVersionView struct {
	Publication PublicationView
	Mappings    []regulatory.ComplianceMapping
	Questions   []regulatory.ChecklistQuestion
}

type ReviewItem struct {
	Candidate      regulatory.CandidateView
	RequiredOwners []regulatory.RequiredOwner
	Decisions      []DecisionView
	BlockingIssues []regulatory.ValidationIssue
}

type DecisionView struct {
	DecisionID                  string
	Decision                    string
	CandidateRootID             string
	CandidateID                 string
	CandidateRevision           int64
	CandidateContentDigest      string
	ActorSubjectID              string
	ActorDepartmentMembershipID string
	ActorDepartmentID           string
	ActorOrganizationalUnitID   string
	Reason                      string
	DecidedAt                   time.Time
	OperationID                 string
	IdempotencyKey              string
	SemanticPayloadDigest       string
	AuditEventID                string
}

type assignment struct {
	ID                   string
	DepartmentID         string
	OrganizationalUnitID string
}

func NewService(pool *database.Pool, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{Pool: pool, Clock: clock}
}

func (service *Service) ListQueue(ctx context.Context, actor identity.Principal) ([]ReviewItem, error) {
	assignments, err := service.currentAssignments(ctx, service.Pool, actor)
	if err != nil {
		return nil, err
	}
	items := []ReviewItem{}
	seen := map[string]bool{}
	for _, current := range assignments {
		rows, err := service.Pool.Query(ctx, `
			SELECT DISTINCT candidate.id
			FROM template_draft_versions candidate
			JOIN candidate_required_owner_assignments owner
			  ON owner.candidate_draft_version_id=candidate.id
			 AND owner.candidate_revision=candidate.revision
			 AND owner.candidate_content_digest=candidate.candidate_content_digest
			WHERE owner.department_id=$1 AND owner.organizational_unit_id=$2
			  AND candidate.status IN ('DEPARTMENT_REVIEW','RETURNED','TECHNICALLY_APPROVED')
			  AND NOT EXISTS (SELECT 1 FROM template_draft_versions successor WHERE successor.supersedes_candidate_id=candidate.id)
			ORDER BY candidate.id`, current.DepartmentID, current.OrganizationalUnitID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var candidateID string
			if err := rows.Scan(&candidateID); err != nil {
				rows.Close()
				return nil, err
			}
			if seen[candidateID] {
				continue
			}
			seen[candidateID] = true
			candidate, err := regulatory.LoadCandidateForGovernance(ctx, service.Pool, candidateID)
			if err != nil {
				rows.Close()
				return nil, err
			}
			decisions, blockers, err := service.reviewMetadata(ctx, service.Pool, candidate)
			if err != nil {
				rows.Close()
				return nil, err
			}
			items = append(items, ReviewItem{Candidate: candidate, RequiredOwners: candidate.RequiredOwners, Decisions: decisions, BlockingIssues: blockers})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return items, nil
}

func (service *Service) GetReviewItem(ctx context.Context, actor identity.Principal, candidateID string) (ReviewItem, error) {
	candidate, err := regulatory.LoadCandidateForGovernance(ctx, service.Pool, candidateID)
	if err != nil {
		return ReviewItem{}, err
	}
	if _, err := service.assignmentForCandidate(
		ctx, service.Pool, actor, candidate.CandidateID,
		candidate.Revision, candidate.ContentDigest,
	); err != nil {
		return ReviewItem{}, err
	}
	decisions, blockers, err := service.reviewMetadata(ctx, service.Pool, candidate)
	if err != nil {
		return ReviewItem{}, err
	}
	return ReviewItem{
		Candidate: candidate, RequiredOwners: candidate.RequiredOwners,
		Decisions: decisions, BlockingIssues: blockers,
	}, nil
}

func (service *Service) GetPublishedVersion(
	ctx context.Context,
	actor identity.Principal,
	templateVersionID string,
) (PublishedVersionView, error) {
	if strings.TrimSpace(templateVersionID) == "" {
		return PublishedVersionView{}, application.ErrInvalid
	}
	var output PublishedVersionView
	var snapshot []byte
	err := service.Pool.QueryRow(ctx, `
		SELECT version.id,decision.id,decision.candidate_root_id,
		       decision.candidate_draft_version_id,decision.candidate_revision,
		       decision.candidate_content_digest,decision.actor_subject_id,
		       decision.actor_department_membership_id,decision.actor_department_id,
		       decision.actor_organizational_unit_id,decision.reason,
		       decision.decided_at,version.published_at,decision.operation_id,
		       decision.idempotency_key,decision.semantic_payload_digest,
		       audit.event_id,version.snapshot
		FROM checklist_template_versions version
		JOIN checklist_publication_decisions decision
		  ON decision.id=version.publication_decision_id
		JOIN audit_events audit
		  ON audit.event_id='AE-' || decision.operation_id
		WHERE version.id=$1`,
		templateVersionID,
	).Scan(
		&output.Publication.TemplateVersionID,
		&output.Publication.PublicationDecisionID,
		&output.Publication.CandidateRootID,
		&output.Publication.CandidateID,
		&output.Publication.CandidateRevision,
		&output.Publication.CandidateContentDigest,
		&output.Publication.ActorSubjectID,
		&output.Publication.ActorMembershipID,
		&output.Publication.ActorDepartmentID,
		&output.Publication.ActorUnitID,
		&output.Publication.Reason,
		&output.Publication.DecidedAt,
		&output.Publication.PublishedAt,
		&output.Publication.OperationID,
		&output.Publication.IdempotencyKey,
		&output.Publication.SemanticPayloadDigest,
		&output.Publication.AuditEventID,
		&snapshot,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublishedVersionView{}, application.ErrNotFound
	}
	if err != nil {
		return PublishedVersionView{}, err
	}
	if _, err := service.assignmentForCandidate(
		ctx, service.Pool, actor, output.Publication.CandidateID,
		output.Publication.CandidateRevision,
		output.Publication.CandidateContentDigest,
	); err != nil {
		return PublishedVersionView{}, err
	}
	var immutable struct {
		Mappings  []regulatory.ComplianceMapping `json:"complianceMappings"`
		Questions []regulatory.ChecklistQuestion `json:"questions"`
	}
	if err := json.Unmarshal(snapshot, &immutable); err != nil {
		return PublishedVersionView{}, application.ErrConflict
	}
	if len(immutable.Mappings) == 0 || len(immutable.Questions) == 0 {
		return PublishedVersionView{}, application.ErrConflict
	}
	output.Mappings = immutable.Mappings
	output.Questions = immutable.Questions
	return output, nil
}

func (service *Service) Return(ctx context.Context, actor identity.Principal, command ReviewCommand) (regulatory.CandidateView, error) {
	return service.decide(ctx, actor, command, "RETURNED")
}

func (service *Service) Reject(ctx context.Context, actor identity.Principal, command ReviewCommand) (regulatory.CandidateView, error) {
	return service.decide(ctx, actor, command, "REJECTED")
}

func (service *Service) Approve(ctx context.Context, actor identity.Principal, command ReviewCommand) (regulatory.CandidateView, error) {
	if err := validateReviewCommand(command); err != nil {
		return regulatory.CandidateView{}, err
	}
	semantic, err := reviewSemantic(command, "TECHNICALLY_APPROVED")
	if err != nil {
		return regulatory.CandidateView{}, err
	}
	var output regulatory.CandidateView
	err = database.WithinTransaction(ctx, service.Pool, func(ctx context.Context, tx pgx.Tx) error {
		if replay, ok, err := replayReview(ctx, tx, command, semantic, "TECHNICALLY_APPROVED"); err != nil {
			return err
		} else if ok {
			if replay.CandidateID != command.CandidateID {
				return application.ErrConflict
			}
			output = replay
			return nil
		}
		rootID, err := lockCandidateRoot(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		if replay, ok, err := replayReview(ctx, tx, command, semantic, "TECHNICALLY_APPROVED"); err != nil {
			return err
		} else if ok {
			if replay.CandidateID != command.CandidateID {
				return application.ErrConflict
			}
			output = replay
			return nil
		}
		candidate, err := regulatory.LoadCandidateForGovernanceQuery(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		revision, digest := candidate.Revision, candidate.ContentDigest
		if candidate.CandidateRootID != rootID || candidate.Status != "DEPARTMENT_REVIEW" ||
			revision != command.ExpectedRevision || digest != command.ExpectedContentDigest {
			return application.ErrConflict
		}
		if err := requireCurrentLeaf(ctx, tx, rootID, command); err != nil {
			return err
		}
		if err := service.lockGenerationRunSourceCurrentness(ctx, tx, []string{candidate.GenerationRunID}); err != nil {
			return err
		}
		candidate, err = regulatory.LoadCandidateForGovernanceQuery(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		if candidate.CandidateRootID != rootID || candidate.Status != "DEPARTMENT_REVIEW" ||
			candidate.Revision != revision || candidate.ContentDigest != digest {
			return application.ErrConflict
		}
		_, blockers, err := service.reviewMetadata(ctx, tx, candidate)
		if err != nil {
			return err
		}
		if len(blockers) > 0 {
			return &regulatory.ValidationError{Issues: blockers}
		}
		current, err := service.assignmentForCandidate(ctx, tx, actor, command.CandidateID, revision, digest)
		if err != nil {
			return err
		}
		var alreadyApproved bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM department_review_decisions
				 WHERE candidate_draft_version_id=$1 AND candidate_revision=$2
				   AND candidate_content_digest=$3 AND decision='TECHNICALLY_APPROVED'
				   AND actor_department_id=$4 AND actor_organizational_unit_id=$5
			)`, command.CandidateID, revision, digest, current.DepartmentID, current.OrganizationalUnitID).Scan(&alreadyApproved); err != nil {
			return err
		}
		if alreadyApproved {
			return application.ErrConflict
		}
		now := service.Clock().UTC()
		if err := insertReviewDecision(ctx, tx, actor, current, command, revision, digest, "TECHNICALLY_APPROVED", semantic, now); err != nil {
			return err
		}
		var remaining int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM candidate_required_owner_assignments owner
			WHERE owner.candidate_draft_version_id=$1 AND owner.candidate_revision=$2
			  AND owner.candidate_content_digest=$3 AND owner.approval_required
			  AND NOT EXISTS (
				SELECT 1 FROM department_review_decisions decision
				 WHERE decision.candidate_draft_version_id=owner.candidate_draft_version_id
				   AND decision.candidate_revision=owner.candidate_revision
				   AND decision.candidate_content_digest=owner.candidate_content_digest
				   AND decision.decision='TECHNICALLY_APPROVED'
				   AND decision.actor_department_id=owner.department_id
				   AND decision.actor_organizational_unit_id=owner.organizational_unit_id
			  )`, command.CandidateID, revision, digest).Scan(&remaining); err != nil {
			return err
		}
		after := "DEPARTMENT_REVIEW"
		if remaining == 0 {
			after = "TECHNICALLY_APPROVED"
			if tag, err := tx.Exec(ctx, `UPDATE template_draft_versions SET status='TECHNICALLY_APPROVED' WHERE id=$1 AND status='DEPARTMENT_REVIEW'`, command.CandidateID); err != nil {
				return err
			} else if tag.RowsAffected() != 1 {
				return application.ErrConflict
			}
		}
		if err := persistAudit(ctx, tx, actor, command, "TECHNICAL_APPROVAL_RECORDED", revision, "DEPARTMENT_REVIEW", after, now); err != nil {
			return err
		}
		output, err = regulatory.LoadCandidateForGovernanceQuery(ctx, tx, command.CandidateID)
		return err
	})
	if err != nil {
		return regulatory.CandidateView{}, err
	}
	return output, nil
}

func (service *Service) Publish(ctx context.Context, actor identity.Principal, command PublicationCommand) (PublicationView, error) {
	reviewCommand := ReviewCommand(command)
	if err := validateReviewCommand(reviewCommand); err != nil {
		return PublicationView{}, err
	}
	semantic, err := reviewSemantic(reviewCommand, "PUBLISHED")
	if err != nil {
		return PublicationView{}, err
	}
	var output PublicationView
	err = database.WithinTransaction(ctx, service.Pool, func(ctx context.Context, tx pgx.Tx) error {
		if replay, ok, err := replayPublication(ctx, tx, reviewCommand, semantic); err != nil {
			return err
		} else if ok {
			output = replay
			return nil
		}
		rootID, err := lockCandidateRoot(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		if replay, ok, err := replayPublication(ctx, tx, reviewCommand, semantic); err != nil {
			return err
		} else if ok {
			output = replay
			return nil
		}
		candidate, err := regulatory.LoadCandidateForGovernanceQuery(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		revision, digest := candidate.Revision, candidate.ContentDigest
		templateID, candidateGenerationRunID := candidate.TemplateID, candidate.GenerationRunID
		if candidate.CandidateRootID != rootID || candidate.Status != "TECHNICALLY_APPROVED" ||
			revision != command.ExpectedRevision || digest != command.ExpectedContentDigest {
			return application.ErrConflict
		}
		if err := requireCurrentLeaf(ctx, tx, rootID, reviewCommand); err != nil {
			return err
		}
		if err := service.lockGenerationRunSourceCurrentness(ctx, tx, []string{candidateGenerationRunID}); err != nil {
			return err
		}
		blockers, err := regulatory.LoadCandidateBlockingIssues(ctx, tx, regulatory.CandidateView{
			CandidateID: command.CandidateID, GenerationRunID: candidateGenerationRunID,
			Revision: revision, ContentDigest: digest,
		})
		if err != nil {
			return err
		}
		if len(blockers) > 0 {
			return &regulatory.ValidationError{Issues: blockers}
		}
		current, err := service.assignmentForCandidate(ctx, tx, actor, command.CandidateID, revision, digest)
		if err != nil {
			return err
		}
		if err := validateExactApprovals(ctx, tx, command.CandidateID, revision, digest); err != nil {
			return fmt.Errorf("validate exact technical approvals: %w", err)
		}
		mappings, questions, recomputed, err := persistedCandidateContent(ctx, tx, command.CandidateID, templateID, revision)
		if err != nil {
			return err
		}
		if recomputed != digest || recomputed != command.ExpectedContentDigest {
			return fmt.Errorf("persisted candidate digest %s does not match approved digest %s: %w", recomputed, digest, application.ErrConflict)
		}
		now := service.Clock().UTC()
		decisionID := "PUBDEC-" + command.OperationID
		if _, err := tx.Exec(ctx, `
			INSERT INTO checklist_publication_decisions
				(id,candidate_root_id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
				 actor_subject_id,actor_department_membership_id,actor_department_id,
				 actor_organizational_unit_id,reason,decided_at,operation_id,idempotency_key,
				 semantic_payload_digest,created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			decisionID, rootID, command.CandidateID, revision, digest, actor.SubjectID,
			current.ID, current.DepartmentID, current.OrganizationalUnitID,
			command.Reason, now, command.OperationID, command.IdempotencyKey,
			semantic, now); err != nil {
			return err
		}
		var version int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0)+1 FROM checklist_template_versions WHERE template_id=$1`, templateID).Scan(&version); err != nil {
			return err
		}
		templateVersionID := "CTV-GOV-" + strings.TrimPrefix(semantic, "sha256:")[:20]
		snapshot, err := json.Marshal(map[string]any{
			"candidateId": command.CandidateID, "candidateRevision": revision,
			"candidateContentDigest": digest, "complianceMappings": mappings,
			"questions": questions,
		})
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO checklist_template_versions
				(id,template_id,version,title,snapshot,published_at,
				 candidate_draft_version_id,candidate_revision,candidate_content_digest,
				 publication_decision_id)
			VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7,$8,$9,$10)`,
			templateVersionID, templateID, version, "Governed "+templateID, string(snapshot), now,
			command.CandidateID, revision, digest, decisionID); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT question_version_id,ordinality-1 FROM template_draft_versions,unnest(question_version_ids) WITH ORDINALITY AS ordered(question_version_id,ordinality) WHERE id=$1 ORDER BY ordinality`, command.CandidateID)
		if err != nil {
			return err
		}
		type orderedQuestion struct {
			id       string
			position int
		}
		orderedQuestions := []orderedQuestion{}
		for rows.Next() {
			var question orderedQuestion
			if err := rows.Scan(&question.id, &question.position); err != nil {
				rows.Close()
				return err
			}
			orderedQuestions = append(orderedQuestions, question)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		for _, question := range orderedQuestions {
			if _, err := tx.Exec(ctx, `INSERT INTO template_version_questions (template_version_id,question_version_id,position) VALUES ($1,$2,$3)`, templateVersionID, question.id, question.position); err != nil {
				return err
			}
		}
		if tag, err := tx.Exec(ctx, `UPDATE template_draft_versions SET status='PUBLISHED' WHERE id=$1 AND status='TECHNICALLY_APPROVED'`, command.CandidateID); err != nil {
			return err
		} else if tag.RowsAffected() != 1 {
			return application.ErrConflict
		}
		if _, err := tx.Exec(ctx, `UPDATE template_masters SET published_template_version_id=$2,revision=revision+1,updated_at=$3 WHERE id=$1`, templateID, templateVersionID, now); err != nil {
			return err
		}
		if err := persistAudit(ctx, tx, actor, reviewCommand, "CHECKLIST_PUBLISHED", revision, "TECHNICALLY_APPROVED", "PUBLISHED", now); err != nil {
			return err
		}
		output = PublicationView{
			TemplateVersionID: templateVersionID, PublicationDecisionID: decisionID,
			CandidateRootID: rootID, CandidateID: command.CandidateID,
			CandidateRevision: revision, CandidateContentDigest: digest,
			ActorSubjectID: actor.SubjectID, ActorMembershipID: current.ID,
			ActorDepartmentID: current.DepartmentID, ActorUnitID: current.OrganizationalUnitID,
			Reason: command.Reason, DecidedAt: now, PublishedAt: now,
			OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
			SemanticPayloadDigest: semantic, AuditEventID: "AE-" + command.OperationID,
		}
		return nil
	})
	return output, err
}

func (service *Service) decide(ctx context.Context, actor identity.Principal, command ReviewCommand, decision string) (regulatory.CandidateView, error) {
	if err := validateReviewCommand(command); err != nil {
		return regulatory.CandidateView{}, err
	}
	semantic, err := reviewSemantic(command, decision)
	if err != nil {
		return regulatory.CandidateView{}, err
	}
	var output regulatory.CandidateView
	err = database.WithinTransaction(ctx, service.Pool, func(ctx context.Context, tx pgx.Tx) error {
		if replay, ok, err := replayReview(ctx, tx, command, semantic, decision); err != nil {
			return err
		} else if ok {
			if replay.CandidateID != command.CandidateID {
				return application.ErrConflict
			}
			output = replay
			return nil
		}
		rootID, err := lockCandidateRoot(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		if replay, ok, err := replayReview(ctx, tx, command, semantic, decision); err != nil {
			return err
		} else if ok {
			if replay.CandidateID != command.CandidateID {
				return application.ErrConflict
			}
			output = replay
			return nil
		}
		candidate, err := regulatory.LoadCandidateForGovernanceQuery(ctx, tx, command.CandidateID)
		if err != nil {
			return err
		}
		revision, digest := candidate.Revision, candidate.ContentDigest
		if candidate.CandidateRootID != rootID || candidate.Status != "DEPARTMENT_REVIEW" ||
			revision != command.ExpectedRevision || digest != command.ExpectedContentDigest {
			return application.ErrConflict
		}
		if err := requireCurrentLeaf(ctx, tx, rootID, command); err != nil {
			return err
		}
		current, err := service.assignmentForCandidate(ctx, tx, actor, command.CandidateID, revision, digest)
		if err != nil {
			return err
		}
		now := service.Clock().UTC()
		if err := insertReviewDecision(ctx, tx, actor, current, command, revision, digest, decision, semantic, now); err != nil {
			return err
		}
		if tag, err := tx.Exec(ctx, `UPDATE template_draft_versions SET status=$2 WHERE id=$1 AND status='DEPARTMENT_REVIEW'`, command.CandidateID, decision); err != nil {
			return err
		} else if tag.RowsAffected() != 1 {
			return application.ErrConflict
		}
		if err := persistAudit(ctx, tx, actor, command, decision, revision, "DEPARTMENT_REVIEW", decision, now); err != nil {
			return err
		}
		output, err = regulatory.LoadCandidateForGovernanceQuery(ctx, tx, command.CandidateID)
		return err
	})
	if err != nil {
		return regulatory.CandidateView{}, err
	}
	return output, nil
}

func lockCandidateRoot(ctx context.Context, tx pgx.Tx, candidateID string) (string, error) {
	var rootID string
	if err := tx.QueryRow(ctx,
		`SELECT candidate_root_id FROM template_draft_versions WHERE id=$1`,
		candidateID,
	).Scan(&rootID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", application.ErrNotFound
		}
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
		rootID,
	); err != nil {
		return "", err
	}
	return rootID, nil
}

func validateReviewCommand(command ReviewCommand) error {
	if strings.TrimSpace(command.OperationID) == "" || strings.TrimSpace(command.IdempotencyKey) == "" ||
		strings.TrimSpace(command.CandidateID) == "" || command.ExpectedRevision < 1 ||
		strings.TrimSpace(command.ExpectedContentDigest) == "" || strings.TrimSpace(command.Reason) == "" {
		return application.ErrInvalid
	}
	return nil
}

func reviewSemantic(command ReviewCommand, decision string) (string, error) {
	return regulatory.CanonicalSHA256(map[string]any{
		"command": decision, "operationId": command.OperationID,
		"candidateId": command.CandidateID, "expectedRevision": command.ExpectedRevision,
		"expectedContentDigest": command.ExpectedContentDigest, "reason": command.Reason,
	})
}

func insertReviewDecision(ctx context.Context, tx pgx.Tx, actor identity.Principal, current assignment, command ReviewCommand, revision int64, digest, decision, semantic string, now time.Time) error {
	var rootID string
	if err := tx.QueryRow(ctx, `SELECT candidate_root_id FROM template_draft_versions WHERE id=$1`, command.CandidateID).Scan(&rootID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO department_review_decisions
			(id,candidate_root_id,candidate_draft_version_id,candidate_revision,candidate_content_digest,decision,
			 actor_subject_id,actor_department_membership_id,actor_department_id,
			 actor_organizational_unit_id,reason,decided_at,operation_id,idempotency_key,
			 semantic_payload_digest)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		"DRD-"+command.OperationID, rootID, command.CandidateID, revision, digest, decision,
		actor.SubjectID, current.ID, current.DepartmentID, current.OrganizationalUnitID,
		command.Reason, now, command.OperationID, command.IdempotencyKey, semantic)
	return err
}

func replayPublication(ctx context.Context, tx pgx.Tx, command ReviewCommand, semantic string) (PublicationView, bool, error) {
	var output PublicationView
	err := tx.QueryRow(ctx, `
		SELECT version.id,decision.id,decision.candidate_root_id,
		       decision.candidate_draft_version_id,
		       decision.candidate_revision,decision.candidate_content_digest,
		       decision.actor_subject_id,decision.actor_department_membership_id,
		       decision.actor_department_id,decision.actor_organizational_unit_id,
		       decision.reason,decision.decided_at,version.published_at,
		       decision.operation_id,decision.idempotency_key,
		       decision.semantic_payload_digest,audit.event_id
		FROM checklist_publication_decisions decision
		JOIN checklist_template_versions version ON version.publication_decision_id=decision.id
		JOIN audit_events audit ON audit.event_id='AE-' || decision.operation_id
		WHERE decision.operation_id=$1 OR decision.idempotency_key=$2`,
		command.OperationID, command.IdempotencyKey).
		Scan(
			&output.TemplateVersionID, &output.PublicationDecisionID,
			&output.CandidateRootID, &output.CandidateID,
			&output.CandidateRevision, &output.CandidateContentDigest,
			&output.ActorSubjectID, &output.ActorMembershipID,
			&output.ActorDepartmentID, &output.ActorUnitID,
			&output.Reason, &output.DecidedAt, &output.PublishedAt,
			&output.OperationID, &output.IdempotencyKey,
			&output.SemanticPayloadDigest, &output.AuditEventID,
		)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicationView{}, false, nil
	}
	if err != nil {
		return PublicationView{}, false, err
	}
	if output.OperationID != command.OperationID ||
		output.IdempotencyKey != command.IdempotencyKey ||
		output.SemanticPayloadDigest != semantic {
		return PublicationView{}, false, application.ErrConflict
	}
	return output, true, nil
}

func validateExactApprovals(ctx context.Context, tx pgx.Tx, candidateID string, revision int64, digest string) error {
	var required, remaining int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*),COUNT(*) FILTER (WHERE NOT EXISTS (
			SELECT 1 FROM department_review_decisions decision
			 WHERE decision.candidate_draft_version_id=owner.candidate_draft_version_id
			   AND decision.candidate_revision=owner.candidate_revision
			   AND decision.candidate_content_digest=owner.candidate_content_digest
			   AND decision.decision='TECHNICALLY_APPROVED'
			   AND decision.actor_department_id=owner.department_id
			   AND decision.actor_organizational_unit_id=owner.organizational_unit_id
		))
		FROM candidate_required_owner_assignments owner
		WHERE owner.candidate_draft_version_id=$1 AND owner.candidate_revision=$2
		  AND owner.candidate_content_digest=$3 AND owner.approval_required`,
		candidateID, revision, digest).Scan(&required, &remaining); err != nil {
		return err
	}
	if required == 0 || remaining != 0 {
		return application.ErrConflict
	}
	return nil
}

func persistedCandidateContent(ctx context.Context, tx pgx.Tx, candidateID, templateID string, revision int64) ([]regulatory.ComplianceMapping, []regulatory.ChecklistQuestion, string, error) {
	mappings := []regulatory.ComplianceMapping{}
	rows, err := tx.Query(ctx, `SELECT snapshot FROM regulatory_generated_mapping_snapshots WHERE candidate_draft_version_id=$1 ORDER BY mapping_ordinal`, candidateID)
	if err != nil {
		return nil, nil, "", err
	}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, nil, "", err
		}
		var mapping regulatory.ComplianceMapping
		if err := json.Unmarshal(raw, &mapping); err != nil {
			rows.Close()
			return nil, nil, "", application.ErrConflict
		}
		mappings = append(mappings, mapping)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, "", err
	}
	rows.Close()
	questions := []regulatory.ChecklistQuestion{}
	rows, err = tx.Query(ctx, `
		SELECT snapshot.snapshot
		FROM template_draft_versions candidate
		CROSS JOIN unnest(candidate.question_version_ids) WITH ORDINALITY AS ordered(question_version_id,ordinality)
		JOIN question_versions version ON version.id=ordered.question_version_id
		JOIN regulatory_generated_question_snapshots snapshot
		  ON snapshot.candidate_draft_version_id=candidate.id AND snapshot.question_id=version.question_id
		WHERE candidate.id=$1 ORDER BY ordered.ordinality`, candidateID)
	if err != nil {
		return nil, nil, "", err
	}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, nil, "", err
		}
		var question regulatory.ChecklistQuestion
		if err := json.Unmarshal(raw, &question); err != nil {
			rows.Close()
			return nil, nil, "", application.ErrConflict
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, "", err
	}
	rows.Close()
	if len(mappings) == 0 || len(questions) == 0 {
		return nil, nil, "", application.ErrConflict
	}
	checklistID := templateID
	if revision == 1 {
		if err := tx.QueryRow(ctx, `SELECT title FROM template_masters WHERE id=$1`, templateID).Scan(&checklistID); err != nil {
			return nil, nil, "", err
		}
	}
	digest, err := regulatory.CanonicalSHA256(map[string]any{
		"complianceMappings":  mappings,
		"inspectionChecklist": map[string]any{"checklistId": checklistID, "questions": questions},
	})
	return mappings, questions, digest, err
}

func (service *Service) currentAssignments(ctx context.Context, query interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, actor identity.Principal) ([]assignment, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) || strings.TrimSpace(actor.SubjectID) == "" {
		return nil, application.ErrForbidden
	}
	rows, err := query.Query(ctx, `
		SELECT membership.id,membership.department_id,membership.organizational_unit_id
		FROM (
			SELECT DISTINCT ON (root_id) * FROM caa_department_memberships
			WHERE effective_from <= $2::date
			ORDER BY root_id,effective_from DESC,id DESC
		) membership
		JOIN LATERAL (
			SELECT status FROM caa_department_status_facts
			WHERE department_id=membership.department_id AND effective_from <= $2::date
			ORDER BY effective_from DESC,id DESC LIMIT 1
		) department_status ON department_status.status='ACTIVE'
		JOIN LATERAL (
			SELECT status FROM caa_organizational_unit_status_facts
			WHERE organizational_unit_id=membership.organizational_unit_id AND effective_from <= $2::date
			ORDER BY effective_from DESC,id DESC LIMIT 1
		) unit_status ON unit_status.status='ACTIVE'
		WHERE membership.subject_id=$1
		  AND membership.membership_role='DEPARTMENT_MANAGER' AND membership.status='ACTIVE'
		  AND (membership.effective_to IS NULL OR membership.effective_to > $2::date)
		ORDER BY membership.department_id,membership.organizational_unit_id`,
		actor.SubjectID, service.Clock().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []assignment{}
	for rows.Next() {
		var item assignment
		if err := rows.Scan(&item.ID, &item.DepartmentID, &item.OrganizationalUnitID); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, application.ErrForbidden
	}
	return result, nil
}

func (service *Service) assignmentForCandidate(ctx context.Context, query rowQuerier, actor identity.Principal, candidateID string, revision int64, digest string) (assignment, error) {
	assignments, err := service.currentAssignments(ctx, query, actor)
	if err != nil {
		return assignment{}, err
	}
	for _, current := range assignments {
		var exists bool
		if err := query.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM candidate_required_owner_assignments
			 WHERE candidate_draft_version_id=$1 AND candidate_revision=$2
			   AND candidate_content_digest=$3 AND department_id=$4
			   AND organizational_unit_id=$5 AND approval_required
			)`, candidateID, revision, digest, current.DepartmentID, current.OrganizationalUnitID).Scan(&exists); err != nil {
			return assignment{}, err
		}
		if exists {
			return current, nil
		}
	}
	return assignment{}, application.ErrForbidden
}

type rowQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (service *Service) reviewMetadata(ctx context.Context, query rowQuerier, candidate regulatory.CandidateView) ([]DecisionView, []regulatory.ValidationIssue, error) {
	decisions := []DecisionView{}
	rows, err := query.Query(ctx, `
		SELECT decision.id,decision.decision,decision.candidate_root_id,
		       decision.candidate_draft_version_id,decision.candidate_revision,
		       decision.candidate_content_digest,decision.actor_subject_id,
		       decision.actor_department_membership_id,
		       decision.actor_department_id,decision.actor_organizational_unit_id,
		       decision.reason,decision.decided_at,decision.operation_id,
		       decision.idempotency_key,decision.semantic_payload_digest,
		       audit.event_id
		FROM department_review_decisions decision
		JOIN audit_events audit ON audit.event_id='AE-' || decision.operation_id
		WHERE candidate_draft_version_id=$1 AND candidate_revision=$2
		  AND candidate_content_digest=$3
		ORDER BY decision.decided_at,decision.id`,
		candidate.CandidateID, candidate.Revision, candidate.ContentDigest)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var decision DecisionView
		if err := rows.Scan(
			&decision.DecisionID, &decision.Decision,
			&decision.CandidateRootID, &decision.CandidateID,
			&decision.CandidateRevision, &decision.CandidateContentDigest,
			&decision.ActorSubjectID, &decision.ActorDepartmentMembershipID,
			&decision.ActorDepartmentID, &decision.ActorOrganizationalUnitID,
			&decision.Reason, &decision.DecidedAt, &decision.OperationID,
			&decision.IdempotencyKey, &decision.SemanticPayloadDigest,
			&decision.AuditEventID,
		); err != nil {
			rows.Close()
			return nil, nil, err
		}
		decisions = append(decisions, decision)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()
	blockers, err := regulatory.LoadCandidateBlockingIssues(ctx, query, candidate)
	if err != nil {
		return nil, nil, err
	}
	return decisions, blockers, nil
}

func replayReview(ctx context.Context, tx pgx.Tx, command ReviewCommand, semantic, decision string) (regulatory.CandidateView, bool, error) {
	var candidateID, operationID, idempotencyKey, storedSemantic, storedDecision, committedStatus string
	err := tx.QueryRow(ctx, `
		SELECT decision.candidate_draft_version_id,decision.operation_id,
		       decision.idempotency_key,decision.semantic_payload_digest,
		       decision.decision,audit.after_status
		FROM department_review_decisions decision
		JOIN audit_events audit ON audit.event_id='AE-' || decision.operation_id
		WHERE decision.operation_id=$1 OR decision.idempotency_key=$2`,
		command.OperationID, command.IdempotencyKey).
		Scan(&candidateID, &operationID, &idempotencyKey, &storedSemantic, &storedDecision, &committedStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return regulatory.CandidateView{}, false, nil
	}
	if err != nil {
		return regulatory.CandidateView{}, false, err
	}
	if operationID != command.OperationID || idempotencyKey != command.IdempotencyKey ||
		storedSemantic != semantic || storedDecision != decision {
		return regulatory.CandidateView{}, false, application.ErrConflict
	}
	output, err := regulatory.LoadCandidateForGovernanceQuery(ctx, tx, candidateID)
	if err != nil {
		return regulatory.CandidateView{}, false, err
	}
	output.Status = committedStatus
	return output, true, nil
}

func requireCurrentLeaf(ctx context.Context, tx pgx.Tx, rootID string, command ReviewCommand) error {
	var id, digest string
	var revision int64
	err := tx.QueryRow(ctx, `
		SELECT candidate.id,candidate.revision,candidate.candidate_content_digest
		FROM template_draft_versions candidate
		WHERE candidate.candidate_root_id=$1
		  AND NOT EXISTS (SELECT 1 FROM template_draft_versions successor WHERE successor.supersedes_candidate_id=candidate.id)`,
		rootID).Scan(&id, &revision, &digest)
	if err != nil {
		return err
	}
	if id != command.CandidateID || revision != command.ExpectedRevision || digest != command.ExpectedContentDigest {
		return application.ErrConflict
	}
	return nil
}

func persistAudit(ctx context.Context, tx pgx.Tx, actor identity.Principal, command ReviewCommand, action string, revision int64, before, after string, at time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_events
			(event_id,occurred_at,actor_subject_id,actor_role,organization_id,action,
			 entity_type,entity_id,entity_version,before_status,after_status,reason,
			 operation_id,correlation_id,request_id,details)
		VALUES ($1,$2,$3,'manager',$4,$5,'GOVERNED_CANDIDATE',$6,$7,$8,$9,$10,$11,$11,$11,'{}'::jsonb)`,
		"AE-"+command.OperationID, at, actor.SubjectID, actor.OrganizationID, action,
		command.CandidateID, revision, before, after, command.Reason, command.OperationID)
	if err != nil {
		return fmt.Errorf("persist governed lifecycle Audit event: %w", err)
	}
	return nil
}
