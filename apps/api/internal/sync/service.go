package fieldsync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
)

const projectionVersion int64 = 1

var ErrCursorScope = errors.New("sync cursor scope mismatch")

type PushStatus string

const (
	PushAccepted       PushStatus = "accepted"
	PushAlreadyApplied PushStatus = "already_applied"
	PushConflict       PushStatus = "conflict"
	PushForbidden      PushStatus = "forbidden"
	PushInvalid        PushStatus = "invalid"
	PushRetryable      PushStatus = "retryable"
)

const (
	ConflictStaleRevision     = "STALE_REVISION"
	ConflictPackageRevoked    = "PACKAGE_REVOKED"
	ConflictAssignmentChanged = "ASSIGNMENT_CHANGED"

	ErrorGrantScope       = "GRANT_SCOPE_MISMATCH"
	ErrorGrantExpired     = "GRANT_EXPIRED"
	ErrorGrantRevoked     = "GRANT_REVOKED"
	ErrorSessionRevoked   = "SESSION_REVOKED"
	ErrorValidationFailed = "VALIDATION_FAILED"
)

type ConflictDescriptor struct {
	Code                  string  `json:"code"`
	EntityID              string  `json:"entityId"`
	AuthoritativeRevision *int64  `json:"authoritativeRevision"`
	AuthoritativeStatus   *string `json:"authoritativeStatus"`
	ChangedAt             *string `json:"changedAt"`
}

type PushResult struct {
	OperationID           string              `json:"operationId"`
	Status                PushStatus          `json:"status"`
	AuthoritativeEntityID *string             `json:"authoritativeEntityId"`
	AuthoritativeRevision *int64              `json:"authoritativeRevision"`
	ErrorCode             string              `json:"errorCode,omitempty"`
	Conflict              *ConflictDescriptor `json:"conflict"`
	AcknowledgedAt        string              `json:"acknowledgedAt"`
}

func (result PushResult) MarshalJSON() ([]byte, error) {
	type encoded PushResult
	var errorCode *string
	if result.ErrorCode != "" {
		errorCode = &result.ErrorCode
	}
	return json.Marshal(struct {
		encoded
		ErrorCode *string `json:"errorCode"`
	}{encoded: encoded(result), ErrorCode: errorCode})
}

func (result *PushResult) UnmarshalJSON(value []byte) error {
	type decoded PushResult
	var wire struct {
		decoded
		ErrorCode *string `json:"errorCode"`
	}
	if err := json.Unmarshal(value, &wire); err != nil {
		return err
	}
	*result = PushResult(wire.decoded)
	if wire.ErrorCode != nil {
		result.ErrorCode = *wire.ErrorCode
	}
	return nil
}

type OperationDependencies struct {
	Clock       func() time.Time
	IDGenerator func(string) string
}

type OperationService struct {
	pool        *database.Pool
	clock       func() time.Time
	idGenerator func(string) string
}

func NewOperationService(pool *database.Pool, dependencies OperationDependencies) *OperationService {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = grantRandomID
	}
	return &OperationService{pool: pool, clock: clock, idGenerator: idGenerator}
}

type operation struct {
	OperationID      string
	ProtocolVersion  int64
	OfflineGrantID   string
	PackageID        string
	PackageVersion   int64
	EntityID         string
	CommandType      string
	BaseRevision     *int64
	DeviceInstanceID string
	ClientOccurredAt string
	Payload          any
}

type wireOperation[T any] struct {
	OperationID      string `json:"operationId"`
	ProtocolVersion  int64  `json:"protocolVersion"`
	OfflineGrantID   string `json:"offlineGrantId"`
	PackageID        string `json:"packageId"`
	PackageVersion   int64  `json:"packageVersion"`
	EntityID         string `json:"entityId"`
	CommandType      string `json:"commandType"`
	BaseRevision     *int64 `json:"baseRevision"`
	DeviceInstanceID string `json:"deviceInstanceId"`
	ClientOccurredAt string `json:"clientOccurredAt"`
	Payload          T      `json:"payload"`
}

type upsertResponsePayload struct {
	AuditID    string `json:"auditId"`
	QuestionID string `json:"questionId"`
	Answer     string `json:"answer"`
	Comment    string `json:"comment"`
}

type createPotentialFindingPayload struct {
	AuditID                           string   `json:"auditId"`
	QuestionID                        string   `json:"questionId"`
	ChecklistResponseID               string   `json:"checklistResponseId"`
	ExpectedChecklistResponseRevision *int64   `json:"expectedChecklistResponseRevision"`
	Title                             string   `json:"title"`
	Description                       string   `json:"description"`
	RequiredComment                   string   `json:"requiredComment"`
	InspectionAttachmentIDs           []string `json:"inspectionAttachmentIds"`
}

type submitChecklistPayload struct {
	AuditID string `json:"auditId"`
}

type registerAttachmentPayload struct {
	AuditID                     string  `json:"auditId"`
	ChecklistResponseID         string  `json:"checklistResponseId"`
	PotentialFindingOperationID *string `json:"potentialFindingOperationId"`
	FileName                    string  `json:"fileName"`
	MediaType                   string  `json:"mediaType"`
	ByteSize                    int64   `json:"byteSize"`
	SHA256                      string  `json:"sha256"`
}

func strictDecode(value []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func decodeOperation(raw json.RawMessage) (operation, error) {
	var discriminator wireOperation[json.RawMessage]
	if err := strictDecode(raw, &discriminator); err != nil {
		return operation{}, err
	}
	convert := func(base wireOperation[json.RawMessage], payload any) operation {
		return operation{
			OperationID: base.OperationID, ProtocolVersion: base.ProtocolVersion,
			OfflineGrantID: base.OfflineGrantID, PackageID: base.PackageID,
			PackageVersion: base.PackageVersion, EntityID: base.EntityID,
			CommandType: base.CommandType, BaseRevision: base.BaseRevision,
			DeviceInstanceID: base.DeviceInstanceID, ClientOccurredAt: base.ClientOccurredAt,
			Payload: payload,
		}
	}
	switch discriminator.CommandType {
	case "UPSERT_CHECKLIST_RESPONSE":
		var wire wireOperation[upsertResponsePayload]
		if err := strictDecode(raw, &wire); err != nil {
			return operation{}, err
		}
		return convert(discriminator, wire.Payload), nil
	case "CREATE_POTENTIAL_FINDING":
		var wire wireOperation[createPotentialFindingPayload]
		if err := strictDecode(raw, &wire); err != nil {
			return operation{}, err
		}
		return convert(discriminator, wire.Payload), nil
	case "SUBMIT_CHECKLIST":
		var wire wireOperation[submitChecklistPayload]
		if err := strictDecode(raw, &wire); err != nil {
			return operation{}, err
		}
		return convert(discriminator, wire.Payload), nil
	case "REGISTER_INSPECTION_ATTACHMENT":
		var wire wireOperation[registerAttachmentPayload]
		if err := strictDecode(raw, &wire); err != nil {
			return operation{}, err
		}
		return convert(discriminator, wire.Payload), nil
	default:
		return operation{}, fmt.Errorf("unsupported field command %q", discriminator.CommandType)
	}
}

func validateOperation(input operation) error {
	if input.OperationID == "" || input.OfflineGrantID == "" || input.PackageID == "" ||
		input.EntityID == "" || input.DeviceInstanceID == "" || input.ProtocolVersion != 1 || input.PackageVersion < 1 {
		return errors.New("complete protocol, operation, grant, package, entity, and device scope is required")
	}
	if _, err := time.Parse(time.RFC3339, input.ClientOccurredAt); err != nil {
		return errors.New("clientOccurredAt must be an RFC3339 instant")
	}
	return nil
}

func operationHash(input operation) (string, error) {
	return idempotency.SemanticHash(struct {
		OperationID      string `json:"operationId"`
		ProtocolVersion  int64  `json:"protocolVersion"`
		OfflineGrantID   string `json:"offlineGrantId"`
		PackageID        string `json:"packageId"`
		PackageVersion   int64  `json:"packageVersion"`
		EntityID         string `json:"entityId"`
		CommandType      string `json:"commandType"`
		BaseRevision     *int64 `json:"baseRevision"`
		DeviceInstanceID string `json:"deviceInstanceId"`
		Payload          any    `json:"payload"`
	}{
		input.OperationID, input.ProtocolVersion, input.OfflineGrantID, input.PackageID,
		input.PackageVersion, input.EntityID, input.CommandType, input.BaseRevision,
		input.DeviceInstanceID, input.Payload,
	})
}

type grantAuthority struct {
	GrantID               string
	SubjectID             string
	DeviceID              string
	PackageID             string
	InspectionID          string
	OrganizationID        string
	SessionID             string
	AssignmentRevision    int64
	InspectionRevision    int64
	PackageVersion        int64
	PackageDigest         string
	QuestionIDs           []string
	AllowedCommands       []string
	GrantExpiresAt        time.Time
	PackageExpiresAt      time.Time
	SessionExpiresAt      time.Time
	AbsoluteExpiresAt     *time.Time
	GrantRevokedAt        *time.Time
	PackageRevokedAt      *time.Time
	SessionRevokedAt      *time.Time
	AssignedSubjectID     string
	InspectionStatus      string
	ChecklistStatus       string
	HasQuestionAssignment bool
}

func loadGrantAuthority(ctx context.Context, transaction pgx.Tx, actor identity.Principal, grantID, packageID, deviceID, command string, expectedPackageVersion int64, now time.Time, allowPackageRevoked bool) (grantAuthority, *PushResult, error) {
	if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" || actor.SessionID == "" {
		result := forbiddenResult("", ErrorGrantScope, now)
		return grantAuthority{}, &result, nil
	}
	var authority grantAuthority
	var assignmentScope []byte
	var currentPackageVersion int64
	var currentPackageDigest string
	err := transaction.QueryRow(ctx, `
		SELECT grant_record.id, grant_record.subject_id, grant_record.device_id, grant_record.package_id,
		       grant_record.inspection_id, inspection.organization_id, grant_record.session_id,
		       grant_record.assignment_revision, inspection.revision, grant_record.package_version,
		       grant_record.package_digest, grant_record.assignment_scope, grant_record.allowed_command_types,
		       grant_record.expires_at, grant_record.revoked_at, package.expires_at, package.revoked_at,
		       session.expires_at, session.absolute_expires_at, session.revoked_at,
		       inspection.assigned_inspector_subject_id, inspection.status, checklist.status,
		       package.package_version, package.package_digest,
		       EXISTS (
			       SELECT 1 FROM inspection_question_assignments question_assignment
			       JOIN audit_assignments audit_assignment
			         ON audit_assignment.inspection_id = question_assignment.inspection_id
			        AND audit_assignment.tombstoned_at IS NULL
			       JOIN audit_team_members team_member
			         ON team_member.assignment_id = audit_assignment.id
			        AND team_member.subject_id = question_assignment.subject_id
			        AND team_member.removed_at IS NULL
			       WHERE question_assignment.inspection_id = grant_record.inspection_id
			         AND question_assignment.subject_id = $2
		       )
		FROM offline_grants grant_record
		JOIN inspections inspection ON inspection.id = grant_record.inspection_id
		JOIN inspection_checklists checklist ON checklist.inspection_id = inspection.id
		JOIN inspection_packages package ON package.id = grant_record.package_id
		JOIN session_references session ON session.id = grant_record.session_id
		WHERE grant_record.id = $1
		  AND package.canonical_scope_snapshot_id IS NOT NULL
		FOR UPDATE OF grant_record, inspection, package, session
	`, grantID, actor.SubjectID).Scan(
		&authority.GrantID, &authority.SubjectID, &authority.DeviceID, &authority.PackageID,
		&authority.InspectionID, &authority.OrganizationID, &authority.SessionID,
		&authority.AssignmentRevision, &authority.InspectionRevision, &authority.PackageVersion,
		&authority.PackageDigest, &assignmentScope, &authority.AllowedCommands,
		&authority.GrantExpiresAt, &authority.GrantRevokedAt, &authority.PackageExpiresAt,
		&authority.PackageRevokedAt, &authority.SessionExpiresAt, &authority.AbsoluteExpiresAt,
		&authority.SessionRevokedAt, &authority.AssignedSubjectID, &authority.InspectionStatus,
		&authority.ChecklistStatus,
		&currentPackageVersion, &currentPackageDigest, &authority.HasQuestionAssignment,
	)
	if err != nil {
		result := forbiddenResult("", ErrorGrantScope, now)
		return grantAuthority{}, &result, nil
	}
	var scope struct {
		QuestionIDs []string `json:"questionIds"`
	}
	if err := json.Unmarshal(assignmentScope, &scope); err != nil {
		result := forbiddenResult("", ErrorGrantScope, now)
		return grantAuthority{}, &result, nil
	}
	authority.QuestionIDs = scope.QuestionIDs
	if authority.SubjectID != actor.SubjectID || authority.SessionID != actor.SessionID ||
		authority.PackageID != packageID || (deviceID != "" && authority.DeviceID != deviceID) {
		result := forbiddenResult("", ErrorGrantScope, now)
		return grantAuthority{}, &result, nil
	}
	if authority.InspectionStatus != "IN_PROGRESS" || authority.ChecklistStatus != "IN_PROGRESS" {
		changedAt := now.Format(time.RFC3339Nano)
		result := conflictResult("", ConflictAssignmentChanged, authority.InspectionID, &authority.InspectionRevision, &authority.InspectionStatus, &changedAt, now)
		return grantAuthority{}, &result, nil
	}
	if authority.GrantRevokedAt != nil {
		result := forbiddenResult("", ErrorGrantRevoked, now)
		return grantAuthority{}, &result, nil
	}
	if now.After(authority.GrantExpiresAt.Add(clockSkewTolerance)) {
		result := forbiddenResult("", ErrorGrantExpired, now)
		return grantAuthority{}, &result, nil
	}
	if authority.SessionRevokedAt != nil || !now.Before(authority.SessionExpiresAt.Add(clockSkewTolerance)) ||
		(authority.AbsoluteExpiresAt != nil && !now.Before(authority.AbsoluteExpiresAt.Add(clockSkewTolerance))) {
		result := forbiddenResult("", ErrorSessionRevoked, now)
		return grantAuthority{}, &result, nil
	}
	if !authority.HasQuestionAssignment || authority.InspectionRevision != authority.AssignmentRevision {
		revision := authority.InspectionRevision
		changedAt := now.Format(time.RFC3339Nano)
		result := conflictResult("", ConflictAssignmentChanged, authority.InspectionID, &revision, nil, &changedAt, now)
		return grantAuthority{}, &result, nil
	}
	if authority.PackageRevokedAt != nil || now.After(authority.PackageExpiresAt.Add(clockSkewTolerance)) {
		if !allowPackageRevoked {
			changedAt := now.Format(time.RFC3339Nano)
			if authority.PackageRevokedAt != nil {
				changedAt = authority.PackageRevokedAt.UTC().Format(time.RFC3339Nano)
			}
			result := conflictResult("", ConflictPackageRevoked, authority.PackageID, nil, nil, &changedAt, now)
			return grantAuthority{}, &result, nil
		}
	}
	if expectedPackageVersion > 0 && (authority.PackageVersion != expectedPackageVersion || currentPackageVersion != expectedPackageVersion) {
		result := forbiddenResult("", ErrorGrantScope, now)
		return grantAuthority{}, &result, nil
	}
	if authority.PackageDigest == "" || authority.PackageDigest != currentPackageDigest {
		result := forbiddenResult("", ErrorGrantScope, now)
		return grantAuthority{}, &result, nil
	}
	if command != "" && !stringIn(authority.AllowedCommands, command) {
		result := forbiddenResult("", ErrorGrantScope, now)
		return grantAuthority{}, &result, nil
	}
	return authority, nil, nil
}

func forbiddenResult(operationID, code string, now time.Time) PushResult {
	return PushResult{OperationID: operationID, Status: PushForbidden, ErrorCode: code, AcknowledgedAt: now.Format(time.RFC3339Nano)}
}

func invalidResult(operationID string, now time.Time) PushResult {
	return PushResult{OperationID: operationID, Status: PushInvalid, ErrorCode: ErrorValidationFailed, AcknowledgedAt: now.Format(time.RFC3339Nano)}
}

func conflictResult(operationID, code, entityID string, revision *int64, status, changedAt *string, now time.Time) PushResult {
	return PushResult{
		OperationID: operationID, Status: PushConflict, AcknowledgedAt: now.Format(time.RFC3339Nano),
		Conflict: &ConflictDescriptor{Code: code, EntityID: entityID, AuthoritativeRevision: revision, AuthoritativeStatus: status, ChangedAt: changedAt},
	}
}

type mutation struct {
	Result         PushResult
	OrganizationID string
	AuditAction    string
	EntityType     string
	EntityID       string
	EntityRevision int64
	BeforeStatus   string
	AfterStatus    string
	ChangeKind     string
	ChangePayload  []byte
	OutboxTopic    string
}

func (service *OperationService) Push(ctx context.Context, actor identity.Principal, raw json.RawMessage) (PushResult, error) {
	now := service.clock().UTC()
	input, err := decodeOperation(raw)
	if err != nil {
		return invalidResult(extractOperationID(raw), now), nil
	}
	if err := validateOperation(input); err != nil {
		return invalidResult(input.OperationID, now), nil
	}
	semanticHash, err := operationHash(input)
	if err != nil {
		return PushResult{}, err
	}
	scope := actor.SubjectID + ":field_sync"
	var output PushResult
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope+":"+input.OperationID); err != nil {
			return err
		}
		var storedHash string
		var storedBody []byte
		err := transaction.QueryRow(ctx, `
			SELECT semantic_hash, response_body FROM idempotency_responses
			WHERE scope = $1 AND operation_id = $2
		`, scope, input.OperationID).Scan(&storedHash, &storedBody)
		if err == nil {
			if storedHash != semanticHash {
				return idempotency.ErrOperationIDReuse
			}
			return json.Unmarshal(storedBody, &output)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		authority, rejection, err := loadGrantAuthority(
			ctx, transaction, actor, input.OfflineGrantID, input.PackageID, input.DeviceInstanceID,
			input.CommandType, input.PackageVersion, now, false,
		)
		if err != nil {
			return err
		}
		if rejection != nil {
			rejection.OperationID = input.OperationID
			if rejection.Conflict != nil && rejection.Conflict.EntityID == "" {
				rejection.Conflict.EntityID = input.EntityID
			}
			output = *rejection
			return nil
		}

		result, err := service.applyMutation(ctx, transaction, actor, authority, input, now)
		if err != nil {
			return err
		}
		output = result.Result
		if output.Status != PushAccepted {
			return nil
		}
		return service.persistAppliedOperation(ctx, transaction, actor, input, semanticHash, result, now)
	})
	return output, err
}

func extractOperationID(raw json.RawMessage) string {
	var value struct {
		OperationID string `json:"operationId"`
	}
	_ = json.Unmarshal(raw, &value)
	return value.OperationID
}

func (service *OperationService) applyMutation(ctx context.Context, transaction pgx.Tx, actor identity.Principal, authority grantAuthority, input operation, now time.Time) (mutation, error) {
	switch input.CommandType {
	case "UPSERT_CHECKLIST_RESPONSE":
		return service.upsertResponse(ctx, transaction, actor, authority, input, now)
	case "CREATE_POTENTIAL_FINDING":
		return service.createPotentialFinding(ctx, transaction, actor, authority, input, now)
	case "SUBMIT_CHECKLIST":
		return service.submitChecklist(ctx, transaction, actor, authority, input, now)
	case "REGISTER_INSPECTION_ATTACHMENT":
		return service.registerAttachment(ctx, transaction, actor, authority, input, now)
	default:
		return mutation{Result: invalidResult(input.OperationID, now)}, nil
	}
}

func (service *OperationService) upsertResponse(ctx context.Context, transaction pgx.Tx, actor identity.Principal, authority grantAuthority, input operation, now time.Time) (mutation, error) {
	payload := input.Payload.(upsertResponsePayload)
	payload.Comment = strings.TrimSpace(payload.Comment)
	if payload.AuditID != authority.InspectionID || payload.QuestionID == "" || !stringIn(authority.QuestionIDs, payload.QuestionID) ||
		!validFieldAnswer(payload.Answer) || ((payload.Answer == "NON_COMPLIANT" || payload.Answer == "OBSERVATION") && payload.Comment == "") {
		return mutation{Result: invalidResult(input.OperationID, now)}, nil
	}
	var checklistStatus string
	if err := transaction.QueryRow(ctx, `
		SELECT checklist.status FROM inspection_checklists checklist
		JOIN inspection_question_assignments assignment
		  ON assignment.inspection_id = checklist.inspection_id AND assignment.question_id = $2 AND assignment.subject_id = $3
		WHERE checklist.inspection_id = $1 FOR UPDATE OF checklist
	`, authority.InspectionID, payload.QuestionID, actor.SubjectID).Scan(&checklistStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return mutation{Result: forbiddenResult(input.OperationID, ErrorGrantScope, now)}, nil
		}
		return mutation{}, err
	}
	if checklistStatus != "IN_PROGRESS" {
		status := checklistStatus
		return mutation{Result: conflictResult(input.OperationID, ConflictStaleRevision, input.EntityID, nil, &status, nil, now)}, nil
	}
	var existingID, beforeAnswer string
	var revision int64
	err := transaction.QueryRow(ctx, `
		SELECT id, response_value, revision FROM checklist_responses
		WHERE inspection_id = $1 AND question_id = $2 FOR UPDATE
	`, authority.InspectionID, payload.QuestionID).Scan(&existingID, &beforeAnswer, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		if input.BaseRevision != nil {
			return mutation{Result: conflictResult(input.OperationID, ConflictStaleRevision, input.EntityID, nil, nil, nil, now)}, nil
		}
		revision = 1
		if _, err := transaction.Exec(ctx, `
			INSERT INTO checklist_responses (
				id, inspection_id, package_id, question_id, assigned_inspector_subject_id,
				response_value, comment_to_auditee, revision, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), 1, $8)
		`, input.EntityID, authority.InspectionID, authority.PackageID, payload.QuestionID,
			actor.SubjectID, payload.Answer, payload.Comment, now); err != nil {
			return mutation{}, err
		}
	} else if err != nil {
		return mutation{}, err
	} else {
		if existingID != input.EntityID || input.BaseRevision == nil || *input.BaseRevision != revision {
			status := beforeAnswer
			changedAt := now.Format(time.RFC3339Nano)
			return mutation{Result: conflictResult(input.OperationID, ConflictStaleRevision, existingID, &revision, &status, &changedAt, now)}, nil
		}
		revision++
		if _, err := transaction.Exec(ctx, `
			UPDATE checklist_responses SET response_value = $2, comment_to_auditee = NULLIF($3, ''),
			       revision = $4, updated_at = $5 WHERE id = $1
		`, existingID, payload.Answer, payload.Comment, revision, now); err != nil {
			return mutation{}, err
		}
	}
	entityID := input.EntityID
	resultRevision := revision
	result := PushResult{
		OperationID: input.OperationID, Status: PushAccepted, AuthoritativeEntityID: &entityID,
		AuthoritativeRevision: &resultRevision, AcknowledgedAt: now.Format(time.RFC3339Nano),
	}
	change, _ := json.Marshal(map[string]any{
		"kind": "checklist_response",
		"value": map[string]any{
			"id": entityID, "questionId": payload.QuestionID, "answer": payload.Answer,
			"comment": payload.Comment, "revision": revision, "updatedAt": now.Format(time.RFC3339Nano),
		},
	})
	return mutation{
		Result: result, OrganizationID: authority.OrganizationID, AuditAction: "checklist_response.recorded",
		EntityType: "checklist_response", EntityID: entityID, EntityRevision: revision,
		BeforeStatus: beforeAnswer, AfterStatus: payload.Answer, ChangeKind: "checklist_response",
		ChangePayload: change, OutboxTopic: "checklist_response.recorded",
	}, nil
}

func (service *OperationService) createPotentialFinding(ctx context.Context, transaction pgx.Tx, actor identity.Principal, authority grantAuthority, input operation, now time.Time) (mutation, error) {
	payload := input.Payload.(createPotentialFindingPayload)
	payload.Title = strings.TrimSpace(payload.Title)
	payload.Description = strings.TrimSpace(payload.Description)
	payload.RequiredComment = strings.TrimSpace(payload.RequiredComment)
	if payload.AuditID != authority.InspectionID || !stringIn(authority.QuestionIDs, payload.QuestionID) ||
		payload.ChecklistResponseID == "" || payload.ExpectedChecklistResponseRevision == nil ||
		payload.Title == "" || payload.Description == "" || payload.RequiredComment == "" {
		return mutation{Result: invalidResult(input.OperationID, now)}, nil
	}
	var responseQuestionID, answer string
	var responseRevision int64
	if err := transaction.QueryRow(ctx, `
		SELECT response.question_id, response.response_value, response.revision
		FROM checklist_responses response
		JOIN inspection_question_assignments assignment
		  ON assignment.inspection_id = response.inspection_id AND assignment.question_id = response.question_id AND assignment.subject_id = $4
		JOIN audit_assignments audit_assignment
		  ON audit_assignment.inspection_id = assignment.inspection_id AND audit_assignment.tombstoned_at IS NULL
		JOIN audit_team_members team_member
		  ON team_member.assignment_id = audit_assignment.id AND team_member.subject_id = assignment.subject_id AND team_member.removed_at IS NULL
		WHERE response.id = $1 AND response.inspection_id = $2 AND response.question_id = $3
		FOR UPDATE OF response
	`, payload.ChecklistResponseID, authority.InspectionID, payload.QuestionID, actor.SubjectID).Scan(
		&responseQuestionID, &answer, &responseRevision,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return mutation{Result: forbiddenResult(input.OperationID, ErrorGrantScope, now)}, nil
		}
		return mutation{}, err
	}
	if responseRevision != *payload.ExpectedChecklistResponseRevision {
		status := answer
		return mutation{Result: conflictResult(input.OperationID, ConflictStaleRevision, payload.ChecklistResponseID, &responseRevision, &status, nil, now)}, nil
	}
	if answer != "NON_COMPLIANT" && answer != "OBSERVATION" {
		return mutation{Result: invalidResult(input.OperationID, now)}, nil
	}
	// Offline commands must observe the same immutable object boundary as the
	// connected Finding path. A declared filename or attachment id is not
	// Evidence: every referenced attachment must already be uploaded, CLEAN,
	// and bound to a canonical object before a Potential Finding can use it.
	attachmentIDs := uniqueNonBlank(payload.InspectionAttachmentIDs)
	if len(attachmentIDs) != len(payload.InspectionAttachmentIDs) {
		return mutation{Result: invalidResult(input.OperationID, now)}, nil
	}
	if len(attachmentIDs) > 0 {
		var cleanCount int
		if err := transaction.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM inspection_attachments
			WHERE id = ANY($1::text[])
			  AND inspection_id = $2
			  AND organization_id = $3
			  AND checklist_response_id = $4
			  AND upload_state = 'UPLOADED'
			  AND scan_state = 'CLEAN'
			  AND canonical_object_metadata_id IS NOT NULL
		`, attachmentIDs, authority.InspectionID, authority.OrganizationID, payload.ChecklistResponseID).Scan(&cleanCount); err != nil {
			return mutation{}, err
		}
		if cleanCount != len(attachmentIDs) {
			return mutation{Result: invalidResult(input.OperationID, now)}, nil
		}
	}
	var existingID, existingStatus string
	var existingRevision int64
	var supersedesPotentialFindingID *string
	err := transaction.QueryRow(ctx, `SELECT id, status, revision FROM potential_findings WHERE checklist_response_id = $1 ORDER BY revision DESC, id DESC LIMIT 1 FOR UPDATE`, payload.ChecklistResponseID).Scan(&existingID, &existingStatus, &existingRevision)
	if err == nil {
		if existingStatus != "RETURNED" {
			return mutation{Result: conflictResult(input.OperationID, ConflictStaleRevision, existingID, &existingRevision, &existingStatus, nil, now)}, nil
		}
		supersedesPotentialFindingID = &existingID
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return mutation{}, err
	}
	potentialFindingID := service.idGenerator("potential-finding")
	if _, err := transaction.Exec(ctx, `
		INSERT INTO potential_findings (
			id, inspection_id, checklist_response_id, organization_id, status, finding_basis,
			expected_evidence, comment_to_auditee, revision, created_at, updated_at,
			question_id, title, description, created_by_subject_id, supersedes_potential_finding_id
		) VALUES ($1, $2, $3, $4, 'PENDING_LEAD_REVIEW', $5, '', $6, 1, $7, $7, $8, $9, $5, $10, $11)
	`, potentialFindingID, authority.InspectionID, payload.ChecklistResponseID, authority.OrganizationID,
		payload.Description, payload.RequiredComment, now, payload.QuestionID, payload.Title, actor.SubjectID, supersedesPotentialFindingID); err != nil {
		return mutation{}, err
	}
	if len(attachmentIDs) > 0 {
		if _, err := transaction.Exec(ctx, `
			UPDATE inspection_attachments
			SET potential_finding_id = $1, revision = revision + 1, updated_at = $2
			WHERE id = ANY($3::text[])
			  AND inspection_id = $4
			  AND checklist_response_id = $5
			  AND potential_finding_id IS NULL
		`, potentialFindingID, now, attachmentIDs, authority.InspectionID, payload.ChecklistResponseID); err != nil {
			return mutation{}, err
		}
	}
	revision := int64(1)
	result := PushResult{
		OperationID: input.OperationID, Status: PushAccepted, AuthoritativeEntityID: &potentialFindingID,
		AuthoritativeRevision: &revision, AcknowledgedAt: now.Format(time.RFC3339Nano),
	}
	change, _ := json.Marshal(map[string]any{
		"kind": "potential_finding",
		"value": map[string]any{
			"id": potentialFindingID, "auditId": authority.InspectionID, "questionId": payload.QuestionID,
			"organizationId": authority.OrganizationID, "title": payload.Title,
			"description": payload.Description, "status": "PENDING_LEAD_REVIEW", "revision": 1,
			"convertedFindingId": nil,
		},
	})
	return mutation{
		Result: result, OrganizationID: authority.OrganizationID, AuditAction: "potential_finding.created",
		EntityType: "potential_finding", EntityID: potentialFindingID, EntityRevision: 1,
		BeforeStatus: "", AfterStatus: "PENDING_LEAD_REVIEW", ChangeKind: "potential_finding",
		ChangePayload: change, OutboxTopic: "potential_finding.created",
	}, nil
}

func (service *OperationService) submitChecklist(ctx context.Context, transaction pgx.Tx, actor identity.Principal, authority grantAuthority, input operation, now time.Time) (mutation, error) {
	payload := input.Payload.(submitChecklistPayload)
	if payload.AuditID != authority.InspectionID || input.EntityID != authority.InspectionID || input.BaseRevision == nil {
		return mutation{Result: invalidResult(input.OperationID, now)}, nil
	}
	var status string
	var revision int64
	if err := transaction.QueryRow(ctx, `SELECT status, revision FROM inspection_checklists WHERE inspection_id = $1 FOR UPDATE`, authority.InspectionID).Scan(&status, &revision); err != nil {
		return mutation{}, err
	}
	if status != "IN_PROGRESS" || revision != *input.BaseRevision {
		return mutation{Result: conflictResult(input.OperationID, ConflictStaleRevision, input.EntityID, &revision, &status, nil, now)}, nil
	}
	var canonicalSnapshotID string
	if err := transaction.QueryRow(ctx, `
		SELECT canonical_scope_snapshot_id
		FROM inspection_packages
		WHERE id = $1 AND inspection_id = $2
		  AND canonical_scope_snapshot_id IS NOT NULL
		  AND checklist_template_version_id IS NULL
		FOR SHARE
	`, authority.PackageID, authority.InspectionID).Scan(&canonicalSnapshotID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return mutation{Result: invalidResult(input.OperationID, now)}, nil
		}
		return mutation{}, err
	}
	if canonicalSnapshotID == "" {
		return mutation{Result: invalidResult(input.OperationID, now)}, nil
	}
	var unclearedAttachments bool
	if err := transaction.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM inspection_attachments
			WHERE inspection_id = $1
			  AND (upload_state <> 'UPLOADED'
			       OR scan_state <> 'CLEAN'
			       OR canonical_object_metadata_id IS NULL)
		)
	`, authority.InspectionID).Scan(&unclearedAttachments); err != nil {
		return mutation{}, err
	}
	if unclearedAttachments {
		return mutation{Result: invalidResult(input.OperationID, now)}, nil
	}
	var scopeCount, assignedCount, uncoveredCount, responseCount, missingRequiredCommentCount int
	if err := transaction.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)
			 FROM canonical_audit_scope_snapshot_questions
			 WHERE snapshot_id = $1),
			(SELECT COUNT(DISTINCT question_id)
			 FROM inspection_question_assignments
			 WHERE inspection_id = $2),
			(SELECT COUNT(*)
			 FROM canonical_audit_scope_snapshot_questions scoped
			 WHERE scoped.snapshot_id = $1
			   AND NOT EXISTS (
				   SELECT 1 FROM inspection_question_assignments assignment
				   WHERE assignment.inspection_id = $2
				     AND assignment.question_id = scoped.question_version_id
			   )),
			(SELECT COUNT(*)
			 FROM canonical_audit_scope_snapshot_questions scoped
			 WHERE scoped.snapshot_id = $1
			   AND EXISTS (
				   SELECT 1 FROM checklist_responses response
				   WHERE response.inspection_id = $2
				     AND response.question_id = scoped.question_version_id
				     AND response.response_value <> ''
			   )),
			(SELECT COUNT(*)
			 FROM canonical_audit_scope_snapshot_questions scoped
			 WHERE scoped.snapshot_id = $1
			   AND EXISTS (
				   SELECT 1 FROM checklist_responses response
				   WHERE response.inspection_id = $2
				     AND response.question_id = scoped.question_version_id
				     AND response.response_value IN ('NON_COMPLIANT', 'OBSERVATION')
				     AND NULLIF(BTRIM(COALESCE(response.comment_to_auditee, '') || COALESCE(response.internal_caa_note, '')), '') IS NULL
			   ))
	`, canonicalSnapshotID, authority.InspectionID).Scan(&scopeCount, &assignedCount, &uncoveredCount, &responseCount, &missingRequiredCommentCount); err != nil {
		return mutation{}, err
	}
	if scopeCount == 0 || assignedCount != scopeCount || uncoveredCount != 0 || responseCount != scopeCount || missingRequiredCommentCount != 0 {
		return mutation{Result: invalidResult(input.OperationID, now)}, nil
	}
	revision++
	if _, err := transaction.Exec(ctx, `UPDATE inspection_checklists SET status = 'SUBMITTED', revision = $2, submitted_at = $3 WHERE inspection_id = $1`, authority.InspectionID, revision, now); err != nil {
		return mutation{}, err
	}
	entityID := authority.InspectionID
	resultRevision := revision
	result := PushResult{
		OperationID: input.OperationID, Status: PushAccepted, AuthoritativeEntityID: &entityID,
		AuthoritativeRevision: &resultRevision, AcknowledgedAt: now.Format(time.RFC3339Nano),
	}
	change, _ := json.Marshal(map[string]any{
		"kind": "inspection_checklist", "auditId": authority.InspectionID,
		"status": "SUBMITTED", "revision": revision,
	})
	return mutation{
		Result: result, OrganizationID: authority.OrganizationID, AuditAction: "checklist.submitted",
		EntityType: "inspection_checklist", EntityID: entityID, EntityRevision: revision,
		BeforeStatus: status, AfterStatus: "SUBMITTED", ChangeKind: "inspection_checklist",
		ChangePayload: change, OutboxTopic: "checklist.submitted",
	}, nil
}

var sha256Pattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func (service *OperationService) registerAttachment(ctx context.Context, transaction pgx.Tx, actor identity.Principal, authority grantAuthority, input operation, now time.Time) (mutation, error) {
	payload := input.Payload.(registerAttachmentPayload)
	payload.FileName = strings.TrimSpace(payload.FileName)
	if payload.AuditID != authority.InspectionID || payload.ChecklistResponseID == "" || payload.FileName == "" ||
		strings.ContainsAny(payload.FileName, `/\\`) || payload.ByteSize <= 0 || payload.ByteSize > 25*1024*1024 ||
		!stringIn([]string{"application/pdf", "image/jpeg", "image/png"}, payload.MediaType) || !sha256Pattern.MatchString(payload.SHA256) {
		return mutation{Result: invalidResult(input.OperationID, now)}, nil
	}
	var questionID string
	if err := transaction.QueryRow(ctx, `
		SELECT response.question_id FROM checklist_responses response
		JOIN inspection_question_assignments assignment
		  ON assignment.inspection_id = response.inspection_id AND assignment.question_id = response.question_id AND assignment.subject_id = $3
		JOIN audit_assignments audit_assignment
		  ON audit_assignment.inspection_id = assignment.inspection_id AND audit_assignment.tombstoned_at IS NULL
		JOIN audit_team_members team_member
		  ON team_member.assignment_id = audit_assignment.id AND team_member.subject_id = assignment.subject_id AND team_member.removed_at IS NULL
		WHERE response.id = $1 AND response.inspection_id = $2
	`, payload.ChecklistResponseID, authority.InspectionID, actor.SubjectID).Scan(&questionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return mutation{Result: forbiddenResult(input.OperationID, ErrorGrantScope, now)}, nil
		}
		return mutation{}, err
	}
	if !stringIn(authority.QuestionIDs, questionID) {
		return mutation{Result: forbiddenResult(input.OperationID, ErrorGrantScope, now)}, nil
	}
	var potentialFindingID *string
	if payload.PotentialFindingOperationID != nil {
		var body []byte
		if err := transaction.QueryRow(ctx, `
			SELECT response_body FROM idempotency_responses
			WHERE scope = $1 AND operation_id = $2
		`, actor.SubjectID+":field_sync", *payload.PotentialFindingOperationID).Scan(&body); err != nil {
			return mutation{Result: invalidResult(input.OperationID, now)}, nil
		}
		var prerequisite PushResult
		if err := json.Unmarshal(body, &prerequisite); err != nil || prerequisite.Status != PushAccepted || prerequisite.AuthoritativeEntityID == nil {
			return mutation{Result: invalidResult(input.OperationID, now)}, nil
		}
		var linkedResponse string
		if err := transaction.QueryRow(ctx, `SELECT checklist_response_id FROM potential_findings WHERE id = $1`, *prerequisite.AuthoritativeEntityID).Scan(&linkedResponse); err != nil || linkedResponse != payload.ChecklistResponseID {
			return mutation{Result: forbiddenResult(input.OperationID, ErrorGrantScope, now)}, nil
		}
		resolved := *prerequisite.AuthoritativeEntityID
		potentialFindingID = &resolved
	}
	attachmentID := service.idGenerator("inspection-attachment")
	if _, err := transaction.Exec(ctx, `
		INSERT INTO inspection_attachments (
			id, inspection_id, package_id, question_id, checklist_response_id, potential_finding_id,
			organization_id, created_by_subject_id, offline_grant_id, device_instance_id,
			file_name, declared_media_type, declared_size_bytes, declared_sha256,
			upload_state, scan_state, revision, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 'PENDING', 'PENDING', 1, $15, $15)
	`, attachmentID, authority.InspectionID, authority.PackageID, questionID,
		payload.ChecklistResponseID, potentialFindingID, authority.OrganizationID, actor.SubjectID,
		authority.GrantID, authority.DeviceID, payload.FileName, payload.MediaType,
		payload.ByteSize, payload.SHA256, now); err != nil {
		return mutation{}, err
	}
	revision := int64(1)
	result := PushResult{
		OperationID: input.OperationID, Status: PushAccepted, AuthoritativeEntityID: &attachmentID,
		AuthoritativeRevision: &revision, AcknowledgedAt: now.Format(time.RFC3339Nano),
	}
	change, _ := json.Marshal(map[string]any{
		"kind": "inspection_attachment", "id": attachmentID, "uploadState": "PENDING", "revision": 1,
	})
	return mutation{
		Result: result, OrganizationID: authority.OrganizationID, AuditAction: "inspection_attachment.registered",
		EntityType: "inspection_attachment", EntityID: attachmentID, EntityRevision: 1,
		BeforeStatus: "", AfterStatus: "PENDING", ChangeKind: "inspection_attachment",
		ChangePayload: change, OutboxTopic: "inspection_attachment.registered",
	}, nil
}

func (service *OperationService) persistAppliedOperation(ctx context.Context, transaction pgx.Tx, actor identity.Principal, input operation, semanticHash string, result mutation, now time.Time) error {
	responseBody, err := json.Marshal(result.Result)
	if err != nil {
		return err
	}
	role := ""
	if len(actor.Roles) > 0 {
		role = string(actor.Roles[0])
	}
	if result.BeforeStatus != result.AfterStatus {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_events (
				sequence_id, event_id, occurred_at, actor_subject_id, actor_role, organization_id,
				action, entity_type, entity_id, entity_version, before_status, after_status,
				operation_id, correlation_id, request_id, details
			) VALUES (
				nextval(pg_get_serial_sequence('audit_events', 'sequence_id')), $1, $2, $3, $4, $5,
				$6, $7, $8, $9, NULLIF($10, ''), NULLIF($11, ''), $12, $12, $12, '{}'::jsonb
			)
		`, service.idGenerator("audit"), now, actor.SubjectID, role, result.OrganizationID,
			result.AuditAction, result.EntityType, result.EntityID, result.EntityRevision,
			result.BeforeStatus, result.AfterStatus, input.OperationID); err != nil {
			return fmt.Errorf("append sync audit event: %w", err)
		}
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO authorized_sync_changes (
			subject_id, organization_id, package_id, kind, entity_id, entity_revision, payload, changed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, actor.SubjectID, result.OrganizationID, input.PackageID, result.ChangeKind,
		result.EntityID, result.EntityRevision, result.ChangePayload, now); err != nil {
		return fmt.Errorf("append authorized sync change: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO outbox_messages (
			id, topic, aggregate_type, aggregate_id, payload, available_at, event_version, idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6, 1, $7)
	`, service.idGenerator("outbox"), result.OutboxTopic, result.EntityType,
		result.EntityID, result.ChangePayload, now, "field_sync:"+input.OperationID); err != nil {
		return fmt.Errorf("enqueue sync outbox: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO idempotency_responses (
			scope, operation_id, semantic_hash, response_status, response_headers, response_body, created_at
		) VALUES ($1, $2, $3, 200, '{}'::jsonb, $4, $5)
	`, actor.SubjectID+":field_sync", input.OperationID, semanticHash, responseBody, now); err != nil {
		return fmt.Errorf("store sync acknowledgement: %w", err)
	}
	return nil
}

type PullInput struct {
	PackageID        string
	OfflineGrantID   string
	DeviceInstanceID string
	Cursor           *string
	Limit            int
}

type PullResult struct {
	Changes            []json.RawMessage `json:"changes"`
	NextCursor         *string           `json:"nextCursor"`
	HasMore            bool              `json:"hasMore"`
	ResnapshotRequired bool              `json:"resnapshotRequired"`
	ProjectionVersion  int64             `json:"projectionVersion"`
}

type changeRow struct {
	SequenceID int64
	Kind       string
	Payload    []byte
}

func (service *OperationService) Pull(ctx context.Context, actor identity.Principal, input PullInput) (PullResult, error) {
	if input.PackageID == "" || input.OfflineGrantID == "" {
		return PullResult{}, ErrCursorScope
	}
	if input.Limit <= 0 {
		input.Limit = 100
	}
	if input.Limit > 250 {
		input.Limit = 250
	}
	now := service.clock().UTC()
	output := PullResult{Changes: []json.RawMessage{}, ProjectionVersion: projectionVersion}
	err := database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		authority, rejection, err := loadGrantAuthority(
			ctx, transaction, actor, input.OfflineGrantID, input.PackageID, input.DeviceInstanceID,
			"", 0, now, true,
		)
		if err != nil {
			return err
		}
		if rejection != nil {
			return rejectionError(*rejection)
		}
		highWater := int64(0)
		if input.Cursor != nil {
			if err := transaction.QueryRow(ctx, `
				SELECT high_water_mark FROM sync_cursor_tokens
				WHERE token = $1 AND subject_id = $2 AND organization_id = $3 AND package_id = $4
				  AND grant_id = $5 AND device_id = $6 AND projection_version = $7
			`, *input.Cursor, actor.SubjectID, authority.OrganizationID, authority.PackageID,
				authority.GrantID, authority.DeviceID, projectionVersion).Scan(&highWater); err != nil {
				return ErrCursorScope
			}
			if highWater > 0 {
				var retained bool
				if err := transaction.QueryRow(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM authorized_sync_changes
						WHERE sequence_id = $1 AND subject_id = $2 AND organization_id = $3 AND package_id = $4
						  AND kind IN ('checklist_response', 'potential_finding', 'package_revoked', 'tombstone')
					)
				`, highWater, actor.SubjectID, authority.OrganizationID, authority.PackageID).Scan(&retained); err != nil {
					return err
				}
				if !retained {
					output.NextCursor = input.Cursor
					output.ResnapshotRequired = true
					return nil
				}
			}
		}
		rows, err := transaction.Query(ctx, `
			SELECT sequence_id, kind, payload
			FROM authorized_sync_changes
			WHERE subject_id = $1 AND organization_id = $2 AND package_id = $3 AND sequence_id > $4
			  AND kind IN ('checklist_response', 'potential_finding', 'package_revoked', 'tombstone')
			ORDER BY sequence_id
			LIMIT $5
		`, actor.SubjectID, authority.OrganizationID, authority.PackageID, highWater, input.Limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		changes := []changeRow{}
		for rows.Next() {
			var row changeRow
			if err := rows.Scan(&row.SequenceID, &row.Kind, &row.Payload); err != nil {
				return err
			}
			changes = append(changes, row)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		output.HasMore = len(changes) > input.Limit
		if output.HasMore {
			changes = changes[:input.Limit]
		}
		for _, row := range changes {
			safe, err := safeAuthorizedChange(row.Kind, row.Payload)
			if err != nil {
				return fmt.Errorf("reject unsafe authorized change %d: %w", row.SequenceID, err)
			}
			output.Changes = append(output.Changes, safe)
			highWater = row.SequenceID
		}
		if len(changes) == 0 {
			output.NextCursor = input.Cursor
			return nil
		}
		token := cursorToken(actor.SubjectID, authority.OrganizationID, authority.PackageID,
			authority.GrantID, authority.DeviceID, projectionVersion, highWater)
		if _, err := transaction.Exec(ctx, `
			INSERT INTO sync_cursor_tokens (
				token, subject_id, organization_id, package_id, grant_id, device_id,
				projection_version, high_water_mark, issued_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT DO NOTHING
		`, token, actor.SubjectID, authority.OrganizationID, authority.PackageID, authority.GrantID,
			authority.DeviceID, projectionVersion, highWater, now); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO sync_cursors (subject_id, device_id, cursor_sequence, projection_version, updated_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (subject_id, device_id) DO UPDATE
			SET cursor_sequence = GREATEST(sync_cursors.cursor_sequence, EXCLUDED.cursor_sequence),
			    projection_version = EXCLUDED.projection_version, updated_at = EXCLUDED.updated_at
		`, actor.SubjectID, authority.DeviceID, highWater, projectionVersion, now); err != nil {
			return err
		}
		output.NextCursor = &token
		return nil
	})
	return output, err
}

func rejectionError(result PushResult) error {
	if result.Conflict != nil {
		switch result.Conflict.Code {
		case ConflictAssignmentChanged:
			return ErrAssignmentChanged
		case ConflictPackageRevoked:
			return ErrPackageRevoked
		}
	}
	switch result.ErrorCode {
	case ErrorGrantExpired:
		return ErrGrantExpired
	case ErrorGrantRevoked:
		return ErrGrantRevoked
	case ErrorSessionRevoked:
		return ErrSessionRevoked
	default:
		return ErrGrantScope
	}
}

func cursorToken(subjectID, organizationID, packageID, grantID, deviceID string, version, highWater int64) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%d\x00%d",
		subjectID, organizationID, packageID, grantID, deviceID, version, highWater)))
	return "sync_" + hex.EncodeToString(digest[:])
}

func safeAuthorizedChange(kind string, payload []byte) (json.RawMessage, error) {
	switch kind {
	case "checklist_response":
		var value struct {
			Kind  string `json:"kind"`
			Value struct {
				ID         string `json:"id"`
				QuestionID string `json:"questionId"`
				Answer     string `json:"answer"`
				Comment    string `json:"comment"`
				Revision   int64  `json:"revision"`
				UpdatedAt  string `json:"updatedAt"`
			} `json:"value"`
		}
		if err := strictDecode(payload, &value); err != nil || value.Kind != kind || !validFieldAnswer(value.Value.Answer) {
			return nil, errors.New("invalid checklist response projection")
		}
		return json.Marshal(value)
	case "potential_finding":
		var value struct {
			Kind  string `json:"kind"`
			Value struct {
				ID                 string  `json:"id"`
				AuditID            string  `json:"auditId"`
				QuestionID         string  `json:"questionId"`
				OrganizationID     string  `json:"organizationId"`
				Title              string  `json:"title"`
				Description        string  `json:"description"`
				Status             string  `json:"status"`
				Revision           int64   `json:"revision"`
				ConvertedFindingID *string `json:"convertedFindingId"`
			} `json:"value"`
		}
		if err := strictDecode(payload, &value); err != nil || value.Kind != kind || value.Value.ID == "" {
			return nil, errors.New("invalid Potential Finding projection")
		}
		return json.Marshal(value)
	case "package_revoked":
		var value struct {
			Kind       string `json:"kind"`
			PackageID  string `json:"packageId"`
			ReasonCode string `json:"reasonCode"`
			RevokedAt  string `json:"revokedAt"`
		}
		if err := strictDecode(payload, &value); err != nil || value.Kind != kind || value.PackageID == "" {
			return nil, errors.New("invalid package revocation projection")
		}
		return json.Marshal(value)
	case "tombstone":
		var value struct {
			Kind       string `json:"kind"`
			EntityType string `json:"entityType"`
			EntityID   string `json:"entityId"`
			Revision   int64  `json:"revision"`
		}
		if err := strictDecode(payload, &value); err != nil || value.Kind != kind ||
			!stringIn([]string{"checklist_response", "potential_finding"}, value.EntityType) || value.EntityID == "" {
			return nil, errors.New("invalid tombstone projection")
		}
		return json.Marshal(value)
	default:
		return nil, errors.New("unrecognized authorized change kind")
	}
}

func validFieldAnswer(value string) bool {
	return stringIn([]string{"COMPLIANT", "NON_COMPLIANT", "OBSERVATION", "NOT_APPLICABLE", "NOT_CHECKED"}, value)
}

func stringIn(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func uniqueNonBlank(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
