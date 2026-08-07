package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/assignments"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/documents"
	findingstore "github.com/MarlonJD/aviaSurveil360/apps/api/internal/findings/store/postgres"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	inspectionstore "github.com/MarlonJD/aviaSurveil360/apps/api/internal/inspections/store/postgres"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/idempotency"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/potentialfindings"
	potentialstore "github.com/MarlonJD/aviaSurveil360/apps/api/internal/potentialfindings/store/postgres"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/reports"
	reportstore "github.com/MarlonJD/aviaSurveil360/apps/api/internal/reports/store/postgres"
	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrConflict  = errors.New("conflict")
	ErrInvalid   = errors.New("invalid command")
	ErrNotFound  = errors.New("not found")
	// ErrDataFeedNotConfigured prevents an authoritative mutation from silently
	// claiming a governed feed fact when its required local writer is absent.
	ErrDataFeedNotConfigured = errors.New("datafeed writer not configured")
)

type Dependencies struct {
	Clock                     func() time.Time
	IDGenerator               func(string) string
	FindingReferenceGenerator func() string
	DataFeedWriter            *datafeed.Writer
}

type Service struct {
	pool                      *database.Pool
	clock                     func() time.Time
	idGenerator               func(string) string
	findingReferenceGenerator func() string
	dataFeedWriter            *datafeed.Writer
}

func NewService(pool *database.Pool, dependencies Dependencies) *Service {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = randomID
	}
	return &Service{pool: pool, clock: clock, idGenerator: idGenerator, findingReferenceGenerator: dependencies.FindingReferenceGenerator, dataFeedWriter: dependencies.DataFeedWriter}
}

type commandEnvelope struct {
	OperationID    string
	IdempotencyKey string
	CorrelationID  string
	Kind           string
	EntityID       string
	Semantic       any
}

type transition[T any] struct {
	Response       T
	OrganizationID string
	Action         string
	EntityType     string
	EntityID       string
	EntityVersion  int64
	BeforeStatus   string
	AfterStatus    string
	Reason         string
	ClosureBasis   string
	SyncKind       string
	OutboxTopic    string
	DataFeedEvents []datafeed.EventInput
}

type MaterializeInspectionCommand struct {
	OperationID                string
	CorrelationID              string
	AssignmentID               string
	ExpectedAssignmentRevision int64
	TemplateVersionID          string
	PackageID                  string
	PackageVersion             int64
	ExpiresAt                  time.Time
}

func (service *Service) MaterializeInspection(
	ctx context.Context,
	actor identity.Principal,
	command MaterializeInspectionCommand,
) (assignments.MaterializedInspection, error) {
	semantic := struct {
		ExpectedAssignmentRevision int64     `json:"expectedAssignmentRevision"`
		TemplateVersionID          string    `json:"templateVersionId"`
		PackageID                  string    `json:"packageId"`
		PackageVersion             int64     `json:"packageVersion"`
		ExpiresAt                  time.Time `json:"expiresAt"`
	}{
		ExpectedAssignmentRevision: command.ExpectedAssignmentRevision,
		TemplateVersionID:          command.TemplateVersionID,
		PackageID:                  command.PackageID,
		PackageVersion:             command.PackageVersion,
		ExpiresAt:                  command.ExpiresAt.UTC(),
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, CorrelationID: command.CorrelationID,
		Kind: "materialize_inspection", EntityID: command.AssignmentID,
		Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[assignments.MaterializedInspection], error) {
		if !actor.HasRole(identity.RoleDepartmentManager) {
			return transition[assignments.MaterializedInspection]{},
				fmt.Errorf("%w: Department Manager authority is required", ErrForbidden)
		}
		if command.ExpectedAssignmentRevision <= 0 || command.TemplateVersionID == "" ||
			command.PackageID == "" || command.PackageVersion <= 0 ||
			command.ExpiresAt.IsZero() {
			return transition[assignments.MaterializedInspection]{}, ErrInvalid
		}
		var inspectionID, organizationID, status string
		var assignmentRevision int64
		if err := transaction.QueryRow(ctx, `
			SELECT inspection_id, organization_id, status, revision
			FROM audit_assignments
			WHERE id = $1 AND tombstoned_at IS NULL
			FOR UPDATE
		`, command.AssignmentID).Scan(
			&inspectionID, &organizationID, &status, &assignmentRevision,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[assignments.MaterializedInspection]{}, ErrNotFound
			}
			return transition[assignments.MaterializedInspection]{}, err
		}
		if assignmentRevision != command.ExpectedAssignmentRevision ||
			assignments.Status(status) != assignments.StatusQuestionsAssigned {
			return transition[assignments.MaterializedInspection]{}, ErrConflict
		}
		var noticePolicy, configuredTemplateVersionID string
		if err := transaction.QueryRow(ctx, `
			SELECT values->>'noticePolicy', values->>'templateVersionId'
			FROM planning_intake_drafts
			WHERE values->>'preparedAuditId' = $1
			  AND tombstoned_at IS NULL
			FOR UPDATE
		`, inspectionID).Scan(&noticePolicy, &configuredTemplateVersionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[assignments.MaterializedInspection]{}, ErrNotFound
			}
			return transition[assignments.MaterializedInspection]{}, err
		}
		if configuredTemplateVersionID != command.TemplateVersionID {
			return transition[assignments.MaterializedInspection]{}, ErrConflict
		}
		var templateSnapshot []byte
		if err := transaction.QueryRow(ctx, `
			SELECT snapshot
			FROM checklist_template_versions
			WHERE id = $1
		`, command.TemplateVersionID).Scan(&templateSnapshot); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[assignments.MaterializedInspection]{}, ErrNotFound
			}
			return transition[assignments.MaterializedInspection]{}, err
		}
		var snapshot struct {
			SchemaVersion   int64                    `json:"schemaVersion"`
			ProtocolVersion int64                    `json:"protocolVersion"`
			Questions       []map[string]interface{} `json:"questions"`
		}
		if err := json.Unmarshal(templateSnapshot, &snapshot); err != nil {
			return transition[assignments.MaterializedInspection]{}, ErrInvalid
		}
		if snapshot.SchemaVersion <= 0 || snapshot.ProtocolVersion <= 0 ||
			len(snapshot.Questions) == 0 {
			return transition[assignments.MaterializedInspection]{}, ErrInvalid
		}
		rows, err := transaction.Query(ctx, `
			SELECT question_id, subject_id
			FROM audit_question_assignments
			WHERE assignment_id = $1
			ORDER BY question_id, subject_id
		`, command.AssignmentID)
		if err != nil {
			return transition[assignments.MaterializedInspection]{}, err
		}
		assignedByQuestion := map[string][]string{}
		for rows.Next() {
			var questionID, subjectID string
			if err := rows.Scan(&questionID, &subjectID); err != nil {
				rows.Close()
				return transition[assignments.MaterializedInspection]{}, err
			}
			assignedByQuestion[questionID] = append(assignedByQuestion[questionID], subjectID)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return transition[assignments.MaterializedInspection]{}, err
		}
		rows.Close()
		for _, question := range snapshot.Questions {
			questionID, _ := question["id"].(string)
			assigned := assignedByQuestion[questionID]
			if questionID == "" || len(assigned) == 0 {
				return transition[assignments.MaterializedInspection]{}, ErrConflict
			}
			question["assignedInspectorUserIds"] = assigned
			delete(assignedByQuestion, questionID)
		}
		if len(assignedByQuestion) != 0 {
			return transition[assignments.MaterializedInspection]{}, ErrConflict
		}
		snapshotJSON, err := json.Marshal(snapshot)
		if err != nil {
			return transition[assignments.MaterializedInspection]{}, err
		}
		digestBytes := sha256.Sum256(snapshotJSON)
		packageDigest := "sha256:" + hex.EncodeToString(digestBytes[:])
		now := service.clock().UTC()
		if _, err := transaction.Exec(ctx, `
			INSERT INTO inspection_packages (
				id, inspection_id, checklist_template_version_id, package_version,
				snapshot, expires_at, created_at, package_digest
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, command.PackageID, inspectionID, command.TemplateVersionID,
			command.PackageVersion, snapshotJSON, command.ExpiresAt.UTC(), now,
			packageDigest); err != nil {
			return transition[assignments.MaterializedInspection]{}, err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO inspection_checklists (inspection_id, status, revision)
			VALUES ($1, 'NOT_STARTED', 1)
		`, inspectionID); err != nil {
			return transition[assignments.MaterializedInspection]{}, err
		}
		for _, question := range snapshot.Questions {
			questionID := question["id"].(string)
			assigned := question["assignedInspectorUserIds"].([]string)
			for _, subjectID := range assigned {
				if _, err := transaction.Exec(ctx, `
					INSERT INTO inspection_question_assignments (
						inspection_id, question_id, subject_id, assignment_revision
					) VALUES ($1, $2, $3, $4)
				`, inspectionID, questionID, subjectID, assignmentRevision+1); err != nil {
					return transition[assignments.MaterializedInspection]{}, err
				}
			}
		}
		nextStatus := assignments.StatusScheduled
		noticeWithheld := noticePolicy == "WITHHELD"
		if !noticeWithheld {
			nextStatus = assignments.StatusAwaitingAuditeeConfirmation
		}
		assignmentUpdate, err := transaction.Exec(ctx, `
			UPDATE audit_assignments
			SET status = $2, revision = revision + 1, updated_at = $3
			WHERE id = $1 AND revision = $4
		`, command.AssignmentID, string(nextStatus), now, assignmentRevision)
		if err != nil {
			return transition[assignments.MaterializedInspection]{}, err
		}
		if assignmentUpdate.RowsAffected() != 1 {
			return transition[assignments.MaterializedInspection]{}, ErrConflict
		}
		inspectionUpdate, err := transaction.Exec(ctx, `
			UPDATE inspections
			SET status = 'READY_TO_EXECUTE', revision = revision + 1, updated_at = $2
			WHERE id = $1 AND status = 'PREPARATION'
		`, inspectionID, now)
		if err != nil {
			return transition[assignments.MaterializedInspection]{}, err
		}
		if inspectionUpdate.RowsAffected() != 1 {
			return transition[assignments.MaterializedInspection]{}, ErrConflict
		}
		response := assignments.MaterializedInspection{
			InspectionID: inspectionID, AssignmentID: command.AssignmentID,
			PackageID: command.PackageID, TemplateVersionID: command.TemplateVersionID,
			PackageVersion: command.PackageVersion, PackageDigest: packageDigest,
			Status: nextStatus, NoticeWithheld: noticeWithheld,
			AssignmentRevision: assignmentRevision + 1,
			ExpiresAt:          command.ExpiresAt.UTC(),
		}
		return transition[assignments.MaterializedInspection]{
			Response: response, OrganizationID: organizationID,
			Action: "inspection.materialized", EntityType: "inspection",
			EntityID: inspectionID, EntityVersion: assignmentRevision + 1,
			BeforeStatus: status, AfterStatus: string(nextStatus),
			SyncKind: "inspection", OutboxTopic: "inspection.materialized",
		}, nil
	})
}

func executeTransition[T any](ctx context.Context, service *Service, actor identity.Principal, envelope commandEnvelope, handler func(context.Context, pgx.Tx) (transition[T], error)) (T, error) {
	var zero T
	if actor.SubjectID == "" || envelope.OperationID == "" || envelope.CorrelationID == "" || envelope.Kind == "" || envelope.EntityID == "" {
		return zero, fmt.Errorf("%w: actor, operation, correlation, kind, and entity are required", ErrInvalid)
	}
	if envelope.IdempotencyKey == "" {
		envelope.IdempotencyKey = envelope.OperationID
	}
	semanticHash, err := idempotency.SemanticHash(struct {
		Kind     string `json:"kind"`
		EntityID string `json:"entityId"`
		Payload  any    `json:"payload"`
	}{Kind: envelope.Kind, EntityID: envelope.EntityID, Payload: envelope.Semantic})
	if err != nil {
		return zero, fmt.Errorf("hash command: %w", err)
	}
	scope := actor.SubjectID + ":" + envelope.Kind
	var response T
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope+":idempotency:"+envelope.IdempotencyKey); err != nil {
			return fmt.Errorf("lock idempotent command: %w", err)
		}
		if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope+":operation:"+envelope.OperationID); err != nil {
			return fmt.Errorf("lock command operation: %w", err)
		}
		var storedHash string
		var storedBody []byte
		err := transaction.QueryRow(ctx, `
			SELECT semantic_hash, response_body
			FROM idempotency_responses
			WHERE scope = $1 AND operation_id = $2
		`, scope, envelope.OperationID).Scan(&storedHash, &storedBody)
		if err == nil {
			if storedHash != semanticHash {
				return idempotency.ErrOperationIDReuse
			}
			if err := json.Unmarshal(storedBody, &response); err != nil {
				return fmt.Errorf("decode idempotent response: %w", err)
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("read idempotent response: %w", err)
		}
		outboxIdempotencyKey := "command:" + scope + ":idempotency:" + envelope.IdempotencyKey
		var reused bool
		if err := transaction.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM outbox_messages WHERE idempotency_key = $1
			)
		`, outboxIdempotencyKey).Scan(&reused); err != nil {
			return fmt.Errorf("check idempotency key reuse: %w", err)
		}
		if reused {
			return idempotency.ErrOperationIDReuse
		}

		result, err := handler(ctx, transaction)
		if err != nil {
			return err
		}
		response = result.Response
		responseBody, err := json.Marshal(response)
		if err != nil {
			return fmt.Errorf("encode command response: %w", err)
		}
		now := service.clock().UTC()
		role := ""
		if len(actor.Roles) > 0 {
			role = string(actor.Roles[0])
		}
		if err := persistCommandTransaction(ctx, transaction, transactionEnvelopeRecord{
			OperationID: envelope.OperationID, CorrelationID: envelope.CorrelationID,
			IdempotencyKey: envelope.IdempotencyKey, IdempotencyScope: scope,
			SemanticHash: semanticHash, ResponseBody: responseBody,
			ActorSubjectID: actor.SubjectID, ActorRole: role, OrganizationID: result.OrganizationID,
			Action: result.Action, EntityType: result.EntityType, EntityID: result.EntityID,
			EntityVersion: result.EntityVersion, BeforeStatus: result.BeforeStatus,
			AfterStatus: result.AfterStatus, Reason: result.Reason, ClosureBasis: result.ClosureBasis,
			SyncKind: result.SyncKind, OutboxTopic: result.OutboxTopic,
			AuditEventID: service.idGenerator("audit"), OutboxMessageID: service.idGenerator("outbox"),
			OccurredAt: now,
		}); err != nil {
			return err
		}
		if len(result.DataFeedEvents) == 0 {
			return nil
		}
		if service.dataFeedWriter == nil {
			return fmt.Errorf("datafeed event emission is configured for %s but no datafeed writer is available", envelope.Kind)
		}
		for _, event := range result.DataFeedEvents {
			if _, err := service.dataFeedWriter.Append(ctx, transaction, envelope.OperationID, event); err != nil {
				return fmt.Errorf("append datafeed event: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return zero, err
	}
	return response, nil
}

type ConvertPotentialFindingCommand struct {
	OperationID           string
	CorrelationID         string
	PotentialFindingID    string
	ExpectedRevision      int64
	Severity              potentialfindings.Severity
	CAPRequired           bool
	EvidenceRequired      bool
	DueDate               *time.Time
	RequirementsSpecified bool
}

type ConvertPotentialFindingResult struct {
	PotentialFindingID       string `json:"potentialFindingId"`
	PotentialFindingStatus   string `json:"potentialFindingStatus"`
	PotentialFindingRevision int64  `json:"potentialFindingRevision"`
	FindingID                string `json:"findingId"`
	FindingReference         string `json:"findingReference"`
	FindingStatus            string `json:"findingStatus"`
}

func (service *Service) ConvertPotentialFinding(ctx context.Context, actor identity.Principal, command ConvertPotentialFindingCommand) (ConvertPotentialFindingResult, error) {
	capRequired := command.CAPRequired
	evidenceRequired := command.EvidenceRequired
	dueDate := command.DueDate
	if !command.RequirementsSpecified {
		if command.Severity == potentialfindings.SeverityObservation {
			capRequired = false
			evidenceRequired = false
			dueDate = nil
		} else {
			capRequired = true
			evidenceRequired = true
		}
	}
	semantic := struct {
		ExpectedRevision int64                      `json:"expectedRevision"`
		Severity         potentialfindings.Severity `json:"severity"`
		CAPRequired      bool                       `json:"capRequired"`
		EvidenceRequired bool                       `json:"evidenceRequired"`
		DueDate          *time.Time                 `json:"dueDate"`
	}{ExpectedRevision: command.ExpectedRevision, Severity: command.Severity, CAPRequired: capRequired, EvidenceRequired: evidenceRequired, DueDate: dueDate}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, CorrelationID: command.CorrelationID, Kind: "convert_potential_finding",
		EntityID: command.PotentialFindingID, Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[ConvertPotentialFindingResult], error) {
		if !actor.HasRole(identity.RoleLeadInspector) {
			return transition[ConvertPotentialFindingResult]{}, fmt.Errorf("%w: Lead Inspector role required", ErrForbidden)
		}
		record, err := potentialstore.New(transaction).GetPotentialFindingForUpdate(ctx, command.PotentialFindingID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[ConvertPotentialFindingResult]{}, ErrNotFound
			}
			return transition[ConvertPotentialFindingResult]{}, err
		}
		potentialStatus := record.Status
		revision := record.Revision
		inspectionID := record.InspectionID
		organizationID := record.OrganizationID
		decision, err := potentialfindings.Decide(potentialfindings.DecideInput{
			Actor: actor, Status: potentialfindings.Status(potentialStatus), Revision: revision, ExpectedRevision: command.ExpectedRevision,
			Decision: potentialfindings.DecisionConvert, Severity: command.Severity,
		})
		if err != nil {
			return transition[ConvertPotentialFindingResult]{}, fmt.Errorf("%w: %v", ErrConflict, err)
		}
		findingID := service.idGenerator("finding")
		findingReference := ""
		if service.findingReferenceGenerator != nil {
			findingReference = service.findingReferenceGenerator()
		} else {
			var publicSequence int64
			if err := transaction.QueryRow(ctx, "SELECT nextval('finding_public_number_sequence')").Scan(&publicSequence); err != nil {
				return transition[ConvertPotentialFindingResult]{}, err
			}
			findingReference = fmt.Sprintf("OPS-%d-%03d", service.clock().UTC().Year(), publicSequence)
		}
		findingStatus := "WAITING_FOR_CAP"
		nextAction := "Auditee to submit CAP"
		if !capRequired {
			if evidenceRequired {
				findingStatus = "EVIDENCE_REQUIRED"
				nextAction = "Auditee submits Evidence"
			} else {
				findingStatus = "PENDING_CLOSURE"
				nextAction = "CAA verifies closure path"
			}
		}
		now := service.clock().UTC()
		if _, err := transaction.Exec(ctx, `
			INSERT INTO findings (
				id, reference, potential_finding_id, inspection_id, organization_id, severity, status,
				owner_subject_id, next_action, due_date, revision, cap_required, evidence_required,
				issued_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, $8, $9, 1, $10, $11, $12, $12, $12)
		`, findingID, findingReference, command.PotentialFindingID, inspectionID, organizationID,
			string(command.Severity), findingStatus, nextAction, dueDate, capRequired,
			evidenceRequired, now); err != nil {
			return transition[ConvertPotentialFindingResult]{}, fmt.Errorf("create canonical Finding: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			UPDATE potential_findings
			SET status = $2, revision = $3, converted_finding_id = $4, updated_at = $5
			WHERE id = $1
		`, command.PotentialFindingID, string(decision.Status), decision.Revision, findingID, service.clock().UTC()); err != nil {
			return transition[ConvertPotentialFindingResult]{}, fmt.Errorf("record Potential Finding conversion: %w", err)
		}
		response := ConvertPotentialFindingResult{
			PotentialFindingID: command.PotentialFindingID, PotentialFindingStatus: string(decision.Status),
			PotentialFindingRevision: decision.Revision, FindingID: findingID, FindingReference: findingReference,
			FindingStatus: findingStatus,
		}
		return transition[ConvertPotentialFindingResult]{
			Response: response, OrganizationID: organizationID, Action: "potential_finding.converted",
			EntityType: "potential_finding", EntityID: command.PotentialFindingID, EntityVersion: decision.Revision,
			BeforeStatus: potentialStatus, AfterStatus: string(decision.Status), SyncKind: "potential_finding", OutboxTopic: "potential_finding.converted",
		}, nil
	})
}

type FindingProjection struct {
	ID             string `json:"id"`
	Reference      string `json:"reference"`
	OrganizationID string `json:"organizationId"`
	RelatedAuditID string `json:"relatedAuditId"`
	Severity       string `json:"severity"`
	Status         string `json:"status"`
	Owner          string `json:"owner"`
	NextAction     string `json:"nextAction"`
	DueDate        string `json:"dueDate"`
	Revision       int64  `json:"revision"`
}

func (service *Service) ListFindings(ctx context.Context, actor identity.Principal) ([]FindingProjection, error) {
	store := findingstore.New(service.pool)
	var records []findingstore.Finding
	var err error
	if actor.HasRole(identity.RoleAuditee) {
		records, err = store.ListFindingsByOrganization(ctx, actor.OrganizationID)
	} else if !actor.IsCAA() {
		return nil, ErrForbidden
	} else {
		records, err = store.ListFindings(ctx)
	}
	if err != nil {
		return nil, err
	}
	items := make([]FindingProjection, 0, len(records))
	for _, record := range records {
		items = append(items, findingProjection(record))
	}
	return items, nil
}

func (service *Service) GetFinding(ctx context.Context, actor identity.Principal, findingID string) (FindingProjection, error) {
	record, err := findingstore.New(service.pool).GetFinding(ctx, findingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return FindingProjection{}, ErrNotFound
	}
	if err != nil {
		return FindingProjection{}, err
	}
	if actor.HasRole(identity.RoleAuditee) {
		if actor.OrganizationID != record.OrganizationID {
			return FindingProjection{}, ErrNotFound
		}
	} else if !actor.IsCAA() {
		return FindingProjection{}, ErrForbidden
	}
	return findingProjection(record), nil
}

func findingProjection(record findingstore.Finding) FindingProjection {
	owner := ""
	if record.OwnerSubjectID != nil {
		owner = *record.OwnerSubjectID
	}
	dueDate := ""
	if record.DueDate.Valid {
		dueDate = record.DueDate.Time.UTC().Format("2006-01-02")
	}
	return FindingProjection{
		ID: record.ID, Reference: record.Reference, OrganizationID: record.OrganizationID,
		RelatedAuditID: record.InspectionID, Severity: record.Severity, Status: record.Status,
		Owner: owner, NextAction: record.NextAction, DueDate: dueDate, Revision: record.Revision,
	}
}

type DecideReportCommand struct {
	OperationID      string
	CorrelationID    string
	ReportVersionID  string
	ExpectedRevision int64
	Decision         reports.Decision
	Reason           string
}

type DecideReportResult struct {
	ReportVersionID string         `json:"reportVersionId"`
	Status          reports.Status `json:"status"`
	Revision        int64          `json:"revision"`
	IssuedAt        *time.Time     `json:"issuedAt"`
}

func (service *Service) DecideReport(ctx context.Context, actor identity.Principal, command DecideReportCommand) (DecideReportResult, error) {
	semantic := struct {
		ExpectedRevision int64            `json:"expectedRevision"`
		Decision         reports.Decision `json:"decision"`
		Reason           string           `json:"reason"`
	}{command.ExpectedRevision, command.Decision, command.Reason}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, CorrelationID: command.CorrelationID, Kind: "decide_report",
		EntityID: command.ReportVersionID, Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[DecideReportResult], error) {
		store := reportstore.New(transaction)
		state, err := store.GetReportApprovalState(ctx, command.ReportVersionID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[DecideReportResult]{}, ErrNotFound
			}
			return transition[DecideReportResult]{}, err
		}
		version, err := store.GetReportVersion(ctx, command.ReportVersionID)
		if err != nil {
			return transition[DecideReportResult]{}, err
		}
		var latestVersion int32
		if err := transaction.QueryRow(ctx, `
			SELECT max(version) FROM report_versions WHERE report_id = $1
		`, version.ReportID).Scan(&latestVersion); err != nil {
			return transition[DecideReportResult]{}, err
		}
		if version.Version != latestVersion {
			return transition[DecideReportResult]{}, fmt.Errorf(
				"%w: report decision must bind the latest immutable family version",
				ErrConflict,
			)
		}
		var reportSnapshot struct {
			Kind              reports.Kind `json:"kind"`
			Ready             bool         `json:"ready"`
			FindingIDs        []string     `json:"findingIds"`
			ContentHash       string       `json:"contentHash"`
			ResponseDueDate   *string      `json:"responseDueDate"`
			CAAVisibleComment *string      `json:"caaVisibleComment"`
		}
		if err := json.Unmarshal(version.Snapshot, &reportSnapshot); err != nil {
			return transition[DecideReportResult]{}, fmt.Errorf("%w: decode report snapshot", ErrInvalid)
		}
		var mismatchedFamilyKinds int
		if err := transaction.QueryRow(ctx, `
			SELECT count(*)
			FROM report_versions
			WHERE report_id = $1
			  AND COALESCE(snapshot ->> 'kind', '') <> $2
		`, version.ReportID, string(reportSnapshot.Kind)).Scan(&mismatchedFamilyKinds); err != nil {
			return transition[DecideReportResult]{}, err
		}
		if mismatchedFamilyKinds != 0 {
			return transition[DecideReportResult]{}, fmt.Errorf(
				"%w: Preliminary and Final Report families cannot share one report identity",
				ErrConflict,
			)
		}
		if _, err := reports.Prepare(reports.PrepareInput{
			ReportID: version.ReportID, Kind: reportSnapshot.Kind,
			Version: int64(version.Version), FindingIDs: reportSnapshot.FindingIDs,
			ContentHash: reportSnapshot.ContentHash, Ready: reportSnapshot.Ready,
		}); err != nil {
			return transition[DecideReportResult]{}, fmt.Errorf("%w: %v", ErrConflict, err)
		}
		inspection, err := inspectionstore.New(transaction).GetInspection(ctx, version.InspectionID)
		if err != nil {
			return transition[DecideReportResult]{}, err
		}
		var mismatchedFamilyOrganizations int
		if err := transaction.QueryRow(ctx, `
			SELECT count(*)
			FROM report_versions candidate
			JOIN inspections candidate_inspection
			  ON candidate_inspection.id = candidate.inspection_id
			WHERE candidate.report_id = $1
			  AND candidate_inspection.organization_id <> $2
		`, version.ReportID, inspection.OrganizationID).Scan(&mismatchedFamilyOrganizations); err != nil {
			return transition[DecideReportResult]{}, err
		}
		if mismatchedFamilyOrganizations != 0 {
			return transition[DecideReportResult]{}, fmt.Errorf(
				"%w: report family organization cannot change between immutable versions",
				ErrConflict,
			)
		}
		if len(reportSnapshot.FindingIDs) > 0 {
			var authorizedFindingCount int
			if err := transaction.QueryRow(ctx, `
				SELECT count(*)
				FROM findings
				WHERE id = ANY($1::text[])
				  AND organization_id = $2
				  AND inspection_id = $3
			`, reportSnapshot.FindingIDs, inspection.OrganizationID, version.InspectionID).Scan(&authorizedFindingCount); err != nil {
				return transition[DecideReportResult]{}, err
			}
			if authorizedFindingCount != len(reportSnapshot.FindingIDs) {
				return transition[DecideReportResult]{}, fmt.Errorf(
					"%w: report Finding identities must belong to the exact Audit organization",
					ErrConflict,
				)
			}
		}
		status := state.Status
		revision := state.Revision
		organizationID := inspection.OrganizationID
		decision, err := reports.Decide(reports.DecideInput{
			Actor: actor, Status: reports.Status(status), Version: revision, ExpectedVersion: command.ExpectedRevision,
			Decision: command.Decision, Reason: command.Reason,
		})
		if err != nil {
			if !actor.HasRole(identity.RoleDepartmentManager, identity.RoleGeneralManager, identity.RoleExecutiveDirector) {
				return transition[DecideReportResult]{}, fmt.Errorf("%w: %v", ErrForbidden, err)
			}
			return transition[DecideReportResult]{}, fmt.Errorf("%w: %v", ErrConflict, err)
		}
		nextRevision := revision + 1
		var issuedAt *time.Time
		if decision.Status == reports.StatusLocked {
			value := service.clock().UTC()
			issuedAt = &value
		}
		if _, err := transaction.Exec(ctx, `
			UPDATE report_approval_states SET status = $2, revision = $3, issued_at = $4, updated_at = $5 WHERE report_version_id = $1
		`, command.ReportVersionID, string(decision.Status), nextRevision, issuedAt, service.clock().UTC()); err != nil {
			return transition[DecideReportResult]{}, err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO report_decisions (id, report_version_id, expected_version, decision, reason, decided_by_subject_id, decided_at)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7)
		`, service.idGenerator("report-decision"), command.ReportVersionID, command.ExpectedRevision, string(command.Decision), command.Reason, actor.SubjectID, service.clock().UTC()); err != nil {
			return transition[DecideReportResult]{}, err
		}
		if decision.Status == reports.StatusLocked {
			documentID := "report-document:" + version.ReportID
			if _, err := transaction.Exec(ctx, `
				INSERT INTO document_records (id, organization_id, kind, title, revision)
				VALUES ($1, $2, 'REPORT', $3, 1)
				ON CONFLICT (id) DO NOTHING
			`, documentID, organizationID, "Report "+version.ReportID); err != nil {
				return transition[DecideReportResult]{}, fmt.Errorf("create Report document record: %w", err)
			}
			var documentOrganizationID, documentKind string
			if err := transaction.QueryRow(ctx, `
				SELECT organization_id, kind FROM document_records WHERE id = $1
			`, documentID).Scan(&documentOrganizationID, &documentKind); err != nil {
				return transition[DecideReportResult]{}, err
			}
			if documentOrganizationID != organizationID || documentKind != "REPORT" {
				return transition[DecideReportResult]{}, fmt.Errorf(
					"%w: report document identity belongs to another immutable family",
					ErrConflict,
				)
			}
			renderSnapshot := documents.RenderSnapshot{
				ReportVersionID: command.ReportVersionID, ReportID: version.ReportID,
				Kind: string(reportSnapshot.Kind), OrganizationID: organizationID,
				AuditID: version.InspectionID, FindingIDs: append([]string(nil), reportSnapshot.FindingIDs...),
				ContentHash: reportSnapshot.ContentHash, Version: int64(version.Version),
				CreatedBySubject: actor.SubjectID,
			}
			encodedSnapshot, err := json.Marshal(renderSnapshot)
			if err != nil {
				return transition[DecideReportResult]{}, err
			}
			renderJobID := service.idGenerator("document-render-job")
			if _, err := transaction.Exec(ctx, `
				INSERT INTO document_render_jobs (
					id, document_id, organization_id, requested_version, status,
					idempotency_key, input_snapshot
				) VALUES ($1, $2, $3, $4, 'PENDING', $5, $6)
				ON CONFLICT (idempotency_key) DO NOTHING
			`, renderJobID, documentID, organizationID, version.Version,
				"report-render:"+command.ReportVersionID, encodedSnapshot); err != nil {
				return transition[DecideReportResult]{}, fmt.Errorf("queue Report document render job: %w", err)
			}
			renderPayload, err := json.Marshal(map[string]any{
				"reportVersionId":  command.ReportVersionID,
				"documentId":       documentID,
				"renderJobId":      renderJobID,
				"requestedVersion": version.Version,
			})
			if err != nil {
				return transition[DecideReportResult]{}, err
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO outbox_messages (
					id, topic, aggregate_type, aggregate_id, payload, available_at,
					event_version, idempotency_key, operation_id, correlation_id
				) VALUES (
					$1, 'document.render_requested', 'report_version', $2, $3, $4,
					1, $5, $6, $7
				)
				ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
			`, service.idGenerator("document-render-outbox"), command.ReportVersionID,
				renderPayload, service.clock().UTC(),
				"document.render_requested:"+command.ReportVersionID,
				command.OperationID, command.CorrelationID); err != nil {
				return transition[DecideReportResult]{}, fmt.Errorf("enqueue Report document render: %w", err)
			}
		}
		response := DecideReportResult{ReportVersionID: command.ReportVersionID, Status: decision.Status, Revision: nextRevision, IssuedAt: issuedAt}
		return transition[DecideReportResult]{
			Response: response, OrganizationID: organizationID, Action: "report.decision_recorded", EntityType: "report_version",
			EntityID: command.ReportVersionID, EntityVersion: nextRevision, BeforeStatus: status, AfterStatus: string(decision.Status),
			Reason: command.Reason, SyncKind: "report_version", OutboxTopic: "report.decision_recorded",
		}, nil
	})
}

func randomID(prefix string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("generate random ID: %v", err))
	}
	return prefix + "-" + hex.EncodeToString(bytes)
}
