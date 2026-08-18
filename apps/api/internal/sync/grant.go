package fieldsync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
)

var (
	ErrGrantExpired      = errors.New("offline grant expired")
	ErrGrantRevoked      = errors.New("offline grant revoked")
	ErrGrantScope        = errors.New("offline grant scope mismatch")
	ErrAssignmentChanged = errors.New("inspection assignment changed")
	ErrPackageRevoked    = errors.New("inspection package revoked")
	ErrSessionRevoked    = errors.New("session revoked or expired")
)

const (
	grantDuration      = 7 * 24 * time.Hour
	clockSkewTolerance = 5 * time.Minute
)

var defaultAllowedCommands = []string{
	"UPSERT_CHECKLIST_RESPONSE",
	"CREATE_POTENTIAL_FINDING",
	"SUBMIT_CHECKLIST",
	"REGISTER_INSPECTION_ATTACHMENT",
	"RESOLVE_FIELD_CONFLICT",
}

type GrantDependencies struct {
	Clock       func() time.Time
	IDGenerator func(string) string
}

type GrantService struct {
	pool        *database.Pool
	clock       func() time.Time
	idGenerator func(string) string
}

func NewGrantService(pool *database.Pool, dependencies GrantDependencies) *GrantService {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = grantRandomID
	}
	return &GrantService{pool: pool, clock: clock, idGenerator: idGenerator}
}

type CheckoutInput struct {
	OperationID            string
	CorrelationID          string
	PackageID              string
	ExpectedPackageVersion int64
	DeviceInstanceID       string
	ProfileKeyID           string
	ProfilePublicJWK       json.RawMessage
	ClaimedSubjectID       string
}

type OfflineGrant struct {
	ID                  string    `json:"grantId"`
	SubjectID           string    `json:"subjectId"`
	OrganizationID      string    `json:"organizationId"`
	PackageID           string    `json:"packageId"`
	PackageVersion      int64     `json:"packageVersion"`
	PackageDigest       string    `json:"packageDigest"`
	AllowedCommandTypes []string  `json:"allowedCommandTypes"`
	QuestionIDs         []string  `json:"questionIds"`
	DeviceInstanceID    string    `json:"deviceInstanceId"`
	ProfileKeyID        string    `json:"profileKeyId"`
	AssignmentRevision  int64     `json:"assignmentRevision"`
	IssuedAt            time.Time `json:"issuedAt"`
	ExpiresAt           time.Time `json:"expiresAt"`
	LeaseIssuedAt       time.Time `json:"leaseIssuedAt"`
	LeaseExpiresAt      time.Time `json:"leaseExpiresAt"`
	ProtocolVersion     int       `json:"protocolVersion"`
}

func (service *GrantService) Issue(ctx context.Context, actor identity.Principal, input CheckoutInput) (OfflineGrant, error) {
	if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" || actor.SessionID == "" {
		return OfflineGrant{}, ErrGrantScope
	}
	if input.OperationID == "" || input.CorrelationID == "" || input.PackageID == "" || input.DeviceInstanceID == "" || input.ProfileKeyID == "" || len(input.ProfilePublicJWK) == 0 {
		return OfflineGrant{}, fmt.Errorf("checkout operation, correlation, package, and device are required")
	}
	if err := validateProfilePublicJWK(input.ProfileKeyID, input.ProfilePublicJWK); err != nil {
		return OfflineGrant{}, fmt.Errorf("profile authority registration failed: %w", err)
	}
	semanticHash, err := idempotency.SemanticHash(struct {
		PackageID              string `json:"packageId"`
		ExpectedPackageVersion int64  `json:"expectedPackageVersion"`
		DeviceInstanceID       string `json:"deviceInstanceId"`
		ProfileKeyID           string `json:"profileKeyId"`
		ProfilePublicJWK       []byte `json:"profilePublicJwk"`
	}{input.PackageID, input.ExpectedPackageVersion, input.DeviceInstanceID, input.ProfileKeyID, input.ProfilePublicJWK})
	if err != nil {
		return OfflineGrant{}, fmt.Errorf("hash checkout command: %w", err)
	}
	now := service.clock().UTC()
	var grant OfflineGrant
	scope := actor.SubjectID + ":checkout_inspection_package"
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		lockKey := scope + ":" + input.OperationID
		if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", lockKey); err != nil {
			return fmt.Errorf("lock idempotent checkout: %w", err)
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
			if err := json.Unmarshal(storedBody, &grant); err != nil {
				return fmt.Errorf("decode idempotent checkout response: %w", err)
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("read idempotent checkout response: %w", err)
		}

		var inspectionID, organizationID, assignedSubject, digest, inspectionStatus, checklistStatus string
		var inspectionRevision, packageVersion int64
		var packageExpiry time.Time
		var packageRevoked *time.Time
		var assignedQuestion bool
		if err := transaction.QueryRow(ctx, `
			SELECT package.inspection_id, inspection.organization_id, inspection.assigned_inspector_subject_id,
			       inspection.revision, package.package_version, package.package_digest, package.expires_at, package.revoked_at,
			       inspection.status, checklist.status,
				       EXISTS (
				       SELECT 1 FROM inspection_question_assignments question_assignment
				       JOIN audit_assignments audit_assignment
				         ON audit_assignment.inspection_id = question_assignment.inspection_id
				        AND audit_assignment.tombstoned_at IS NULL
				       JOIN audit_team_members team_member
				         ON team_member.assignment_id = audit_assignment.id
				        AND team_member.subject_id = question_assignment.subject_id
				        AND team_member.removed_at IS NULL
				       WHERE question_assignment.inspection_id = package.inspection_id
				         AND question_assignment.subject_id = $2
			       )
			FROM inspection_packages package
			JOIN inspections inspection ON inspection.id = package.inspection_id
			JOIN inspection_checklists checklist ON checklist.inspection_id = inspection.id
			WHERE package.id = $1
			  AND package.canonical_scope_snapshot_id IS NOT NULL
			FOR UPDATE OF package, inspection
		`, input.PackageID, actor.SubjectID).Scan(&inspectionID, &organizationID, &assignedSubject, &inspectionRevision, &packageVersion, &digest, &packageExpiry, &packageRevoked, &inspectionStatus, &checklistStatus, &assignedQuestion); err != nil {
			return err
		}
		if !assignedQuestion || packageVersion != input.ExpectedPackageVersion || inspectionStatus != "IN_PROGRESS" || checklistStatus != "IN_PROGRESS" {
			return ErrGrantScope
		}
		if packageRevoked != nil || now.After(packageExpiry) {
			return ErrPackageRevoked
		}
		var sessionSubject string
		var sessionExpires time.Time
		var absoluteExpires *time.Time
		var sessionRevoked *time.Time
		if err := transaction.QueryRow(ctx, `
			SELECT subject_id, expires_at, absolute_expires_at, revoked_at
			FROM session_references WHERE id = $1 FOR UPDATE
		`, actor.SessionID).Scan(&sessionSubject, &sessionExpires, &absoluteExpires, &sessionRevoked); err != nil {
			return ErrSessionRevoked
		}
		if sessionSubject != actor.SubjectID || sessionRevoked != nil || !now.Before(sessionExpires) || (absoluteExpires != nil && !now.Before(*absoluteExpires)) {
			return ErrSessionRevoked
		}
		rows, err := transaction.Query(ctx, `
			SELECT question_assignment.question_id
			FROM inspection_question_assignments question_assignment
			JOIN audit_assignments audit_assignment
			  ON audit_assignment.inspection_id = question_assignment.inspection_id
			 AND audit_assignment.tombstoned_at IS NULL
			JOIN audit_team_members team_member
			  ON team_member.assignment_id = audit_assignment.id
			 AND team_member.subject_id = question_assignment.subject_id
			 AND team_member.removed_at IS NULL
			WHERE question_assignment.inspection_id = $1
			  AND question_assignment.subject_id = $2
			ORDER BY question_assignment.question_id
		`, inspectionID, actor.SubjectID)
		if err != nil {
			return err
		}
		defer rows.Close()
		questionIDs := []string{}
		for rows.Next() {
			var questionID string
			if err := rows.Scan(&questionID); err != nil {
				return err
			}
			questionIDs = append(questionIDs, questionID)
		}
		if len(questionIDs) == 0 {
			return ErrGrantScope
		}
		expiresAt := now.Add(grantDuration)
		if packageExpiry.Before(expiresAt) {
			expiresAt = packageExpiry
		}
		grant = OfflineGrant{
			ID: service.idGenerator("grant"), SubjectID: actor.SubjectID, OrganizationID: organizationID,
			PackageID: input.PackageID, PackageVersion: packageVersion, PackageDigest: digest,
			AllowedCommandTypes: append([]string(nil), defaultAllowedCommands...), QuestionIDs: questionIDs,
			DeviceInstanceID: input.DeviceInstanceID, ProfileKeyID: input.ProfileKeyID,
			AssignmentRevision: inspectionRevision, IssuedAt: now, ExpiresAt: expiresAt,
			LeaseIssuedAt: now, LeaseExpiresAt: expiresAt, ProtocolVersion: 1,
		}
		assignmentScope, _ := json.Marshal(map[string]any{"questionIds": questionIDs})
		if _, err := transaction.Exec(ctx, `
			INSERT INTO offline_grants (
				id, subject_id, device_id, package_id, inspection_id, assignment_revision, granted_at, expires_at,
				session_id, package_version, package_digest, allowed_command_types, assignment_scope, protocol_version,
				profile_key_id, profile_public_jwk, lease_issued_at, lease_expires_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 1, $14, $15, $16, $17)
		`, grant.ID, grant.SubjectID, grant.DeviceInstanceID, grant.PackageID, inspectionID, inspectionRevision, now, expiresAt, actor.SessionID, packageVersion, digest, defaultAllowedCommands, assignmentScope, input.ProfileKeyID, input.ProfilePublicJWK, now, expiresAt); err != nil {
			return err
		}
		responseBody, err := json.Marshal(grant)
		if err != nil {
			return fmt.Errorf("encode checkout response: %w", err)
		}
		role := ""
		if len(actor.Roles) > 0 {
			role = string(actor.Roles[0])
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_events (
				sequence_id, event_id, occurred_at, actor_subject_id, actor_role, organization_id,
				action, entity_type, entity_id, entity_version, before_status, after_status,
				operation_id, correlation_id, request_id, details
			) VALUES (
				nextval(pg_get_serial_sequence('audit_events', 'sequence_id')), $1, $2, $3, $4, $5,
				'offline_grant.issued', 'offline_grant', $6, 1, NULL, 'ACTIVE', $7, $8, $8, '{}'::jsonb
			)
		`, service.idGenerator("audit"), now, actor.SubjectID, role, organizationID, grant.ID, input.OperationID, input.CorrelationID); err != nil {
			return fmt.Errorf("append checkout audit event: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO authorized_sync_changes (
				subject_id, organization_id, package_id, kind, entity_id, entity_revision, payload, changed_at
			) VALUES ($1, $2, $3, 'offline_grant', $4, 1, $5, $6)
		`, actor.SubjectID, organizationID, input.PackageID, grant.ID, responseBody, now); err != nil {
			return fmt.Errorf("append checkout sync change: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO outbox_messages (id, topic, aggregate_type, aggregate_id, payload, available_at)
			VALUES ($1, 'offline_grant.issued', 'offline_grant', $2, $3, $4)
		`, service.idGenerator("outbox"), grant.ID, responseBody, now); err != nil {
			return fmt.Errorf("enqueue checkout outbox: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO idempotency_responses (
				scope, operation_id, semantic_hash, response_status, response_headers, response_body, created_at
			) VALUES ($1, $2, $3, 200, '{}'::jsonb, $4, $5)
		`, scope, input.OperationID, semanticHash, responseBody, now); err != nil {
			return fmt.Errorf("store idempotent checkout response: %w", err)
		}
		return nil
	})
	return grant, err
}

func grantRandomID(prefix string) string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		panic(fmt.Sprintf("generate grant identifier: %v", err))
	}
	return prefix + "-" + hex.EncodeToString(value)
}

type AuthorizationInput struct {
	GrantID          string
	PackageID        string
	DeviceInstanceID string
	ServerNow        time.Time
	CommandType      string
}

func (service *GrantService) Authorize(ctx context.Context, actor identity.Principal, input AuthorizationInput) error {
	now := input.ServerNow.UTC()
	if now.IsZero() {
		now = service.clock().UTC()
	}
	var subjectID, deviceID, packageID, inspectionID, sessionID, inspectionStatus, checklistStatus string
	var activeTeamMember bool
	var assignmentRevision, inspectionRevision int64
	var expiresAt, packageExpires, sessionExpires time.Time
	var grantRevoked, packageRevoked, sessionRevoked, absoluteExpires *time.Time
	var allowedCommands []string
	err := service.pool.QueryRow(ctx, `
		SELECT offline_grant.subject_id, offline_grant.device_id, offline_grant.package_id, offline_grant.inspection_id, offline_grant.session_id,
		       offline_grant.assignment_revision, offline_grant.expires_at, offline_grant.revoked_at, offline_grant.allowed_command_types,
		       inspection.status, checklist.status, inspection.revision,
		       EXISTS (
		         SELECT 1 FROM audit_assignments assignment
		         JOIN audit_team_members member ON member.assignment_id=assignment.id
		         WHERE assignment.inspection_id=inspection.id AND assignment.tombstoned_at IS NULL
		           AND member.subject_id=$2 AND member.removed_at IS NULL
		       ),
		       package.expires_at, package.revoked_at,
		       session.expires_at, session.absolute_expires_at, session.revoked_at
		FROM offline_grants offline_grant
		JOIN inspections inspection ON inspection.id = offline_grant.inspection_id
		JOIN inspection_checklists checklist ON checklist.inspection_id = inspection.id
		JOIN inspection_packages package ON package.id = offline_grant.package_id
		JOIN session_references session ON session.id = offline_grant.session_id
		WHERE offline_grant.id = $1
	`, input.GrantID, actor.SubjectID).Scan(
		&subjectID, &deviceID, &packageID, &inspectionID, &sessionID, &assignmentRevision, &expiresAt, &grantRevoked, &allowedCommands,
		&inspectionStatus, &checklistStatus, &inspectionRevision, &activeTeamMember, &packageExpires, &packageRevoked, &sessionExpires, &absoluteExpires, &sessionRevoked,
	)
	if err != nil {
		return fmt.Errorf("%w: load grant authority: %v", ErrGrantScope, err)
	}
	if subjectID != actor.SubjectID || sessionID != actor.SessionID || deviceID != input.DeviceInstanceID || packageID != input.PackageID {
		return ErrGrantScope
	}
	if grantRevoked != nil {
		return ErrGrantRevoked
	}
	if now.After(expiresAt.Add(clockSkewTolerance)) {
		return ErrGrantExpired
	}
	if packageRevoked != nil || now.After(packageExpires.Add(clockSkewTolerance)) {
		return ErrPackageRevoked
	}
	if inspectionStatus != "IN_PROGRESS" || checklistStatus != "IN_PROGRESS" {
		return ErrAssignmentChanged
	}
	if !activeTeamMember || inspectionRevision != assignmentRevision {
		return ErrAssignmentChanged
	}
	if sessionRevoked != nil || !now.Before(sessionExpires.Add(clockSkewTolerance)) || (absoluteExpires != nil && !now.Before(absoluteExpires.Add(clockSkewTolerance))) {
		return ErrSessionRevoked
	}
	if input.CommandType != "" && !contains(allowedCommands, input.CommandType) {
		return ErrGrantScope
	}
	return nil
}

func (service *GrantService) Revoke(ctx context.Context, actor identity.Principal, grantID, reason string) error {
	command, err := service.pool.Exec(ctx, `
		UPDATE offline_grants SET revoked_at = $1, revoke_reason = $2
		WHERE id = $3 AND subject_id = $4 AND revoked_at IS NULL
	`, service.clock().UTC(), reason, grantID, actor.SubjectID)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrGrantScope
	}
	return nil
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
