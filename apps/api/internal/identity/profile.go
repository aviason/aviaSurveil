package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	identitystore "github.com/aviason/aviaSurveil/internal/identity/store/postgres"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrConflict        = errors.New("identity revision conflict")
	ErrInvalidProfile  = errors.New("invalid identity command")
	ErrPrecondition    = errors.New("identity precondition failed")
)

type Profile struct {
	SubjectID      string `json:"subjectId"`
	Role           Role   `json:"role"`
	OrganizationID string `json:"organizationId"`
	DisplayName    string `json:"displayName"`
	Revision       int64  `json:"revision"`
}

type Settings struct {
	SubjectID               string          `json:"subjectId"`
	NotificationPreferences json.RawMessage `json:"notificationPreferences"`
	Locale                  string          `json:"locale"`
	Timezone                string          `json:"timezone"`
	Revision                int64           `json:"revision"`
}

type UpdateProfileCommand struct {
	OperationID      string
	IdempotencyKey   string
	ExpectedRevision int64
	DisplayName      string
}

type UpdateSettingsCommand struct {
	OperationID             string
	IdempotencyKey          string
	ExpectedRevision        int64
	NotificationPreferences json.RawMessage
	Locale                  string
	Timezone                string
}

type ProfileServiceDependencies struct {
	Clock       func() time.Time
	IDGenerator func(string) string
}

type ProfileService struct {
	pool        *database.Pool
	clock       func() time.Time
	idGenerator func(string) string
}

func NewProfileService(pool *database.Pool, dependencies ProfileServiceDependencies) *ProfileService {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = randomProfileID
	}
	return &ProfileService{pool: pool, clock: clock, idGenerator: idGenerator}
}

func (service *ProfileService) GetProfile(ctx context.Context, actor Principal) (Profile, error) {
	role, err := effectiveRole(actor)
	if err != nil {
		return Profile{}, err
	}
	record, err := identitystore.New(service.pool).GetProfile(ctx, actor.SubjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	return Profile{
		SubjectID: actor.SubjectID, Role: role, OrganizationID: actor.OrganizationID,
		DisplayName: record.DisplayName, Revision: record.Revision,
	}, nil
}

func (service *ProfileService) GetSettings(ctx context.Context, actor Principal) (Settings, error) {
	if _, err := effectiveRole(actor); err != nil {
		return Settings{}, err
	}
	record, err := identitystore.New(service.pool).GetSettings(ctx, actor.SubjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Settings{}, ErrProfileNotFound
	}
	if err != nil {
		return Settings{}, err
	}
	return settingsFromRecord(record), nil
}

func (service *ProfileService) UpdateProfile(ctx context.Context, actor Principal, command UpdateProfileCommand) (Profile, error) {
	role, err := effectiveRole(actor)
	if err != nil {
		return Profile{}, err
	}
	command.DisplayName = strings.TrimSpace(command.DisplayName)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	if strings.TrimSpace(command.OperationID) == "" || command.IdempotencyKey == "" ||
		command.ExpectedRevision <= 0 || command.DisplayName == "" {
		return Profile{}, ErrInvalidProfile
	}
	semanticHash, err := idempotency.SemanticHash(struct {
		IdempotencyKey   string `json:"idempotencyKey"`
		ExpectedRevision int64  `json:"expectedRevision"`
		DisplayName      string `json:"displayName"`
	}{command.IdempotencyKey, command.ExpectedRevision, command.DisplayName})
	if err != nil {
		return Profile{}, err
	}
	scope := actor.SubjectID + ":profile_update"
	var output Profile
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		replayed, err := lockAndReplay(
			ctx, transaction, scope, command.OperationID, command.IdempotencyKey, semanticHash, &output,
		)
		if err != nil || replayed {
			return err
		}
		now := service.clock().UTC()
		record, err := identitystore.New(transaction).UpdateProfile(ctx, identitystore.UpdateProfileParams{
			SubjectID: actor.SubjectID, DisplayName: command.DisplayName,
			UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true}, Revision: command.ExpectedRevision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return err
		}
		output = Profile{
			SubjectID: actor.SubjectID, Role: role, OrganizationID: actor.OrganizationID,
			DisplayName: record.DisplayName, Revision: record.Revision,
		}
		return service.persistEnvelope(ctx, transaction, actor, envelope{
			Scope: scope, OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
			SemanticHash: semanticHash,
			Action:       "PROFILE_UPDATED", EntityType: "USER_PROFILE", EntityID: actor.SubjectID,
			EntityVersion: record.Revision, OutboxTopic: "identity.profile.updated",
			Response: output, OccurredAt: now,
		})
	})
	return output, err
}

func (service *ProfileService) UpdateSettings(ctx context.Context, actor Principal, command UpdateSettingsCommand) (Settings, error) {
	if _, err := effectiveRole(actor); err != nil {
		return Settings{}, err
	}
	command.Locale = strings.TrimSpace(command.Locale)
	command.Timezone = strings.TrimSpace(command.Timezone)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	if strings.TrimSpace(command.OperationID) == "" || command.IdempotencyKey == "" || command.ExpectedRevision <= 0 ||
		command.Locale == "" || command.Timezone == "" || len(command.NotificationPreferences) == 0 ||
		!json.Valid(command.NotificationPreferences) {
		return Settings{}, ErrInvalidProfile
	}
	var preferences map[string]bool
	if err := json.Unmarshal(command.NotificationPreferences, &preferences); err != nil {
		return Settings{}, ErrInvalidProfile
	}
	canonicalPreferences, err := json.Marshal(preferences)
	if err != nil {
		return Settings{}, err
	}
	semanticHash, err := idempotency.SemanticHash(struct {
		IdempotencyKey          string          `json:"idempotencyKey"`
		ExpectedRevision        int64           `json:"expectedRevision"`
		NotificationPreferences json.RawMessage `json:"notificationPreferences"`
		Locale                  string          `json:"locale"`
		Timezone                string          `json:"timezone"`
	}{command.IdempotencyKey, command.ExpectedRevision, canonicalPreferences, command.Locale, command.Timezone})
	if err != nil {
		return Settings{}, err
	}
	scope := actor.SubjectID + ":settings_update"
	var output Settings
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		replayed, err := lockAndReplay(
			ctx, transaction, scope, command.OperationID, command.IdempotencyKey, semanticHash, &output,
		)
		if err != nil || replayed {
			return err
		}
		now := service.clock().UTC()
		record, err := identitystore.New(transaction).UpdateSettings(ctx, identitystore.UpdateSettingsParams{
			SubjectID: actor.SubjectID, NotificationPreferences: canonicalPreferences,
			Locale: command.Locale, Timezone: command.Timezone,
			UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true}, Revision: command.ExpectedRevision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return err
		}
		output = settingsFromRecord(record)
		return service.persistEnvelope(ctx, transaction, actor, envelope{
			Scope: scope, OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
			SemanticHash: semanticHash,
			Action:       "SETTINGS_UPDATED", EntityType: "USER_SETTINGS", EntityID: actor.SubjectID,
			EntityVersion: record.Revision, OutboxTopic: "identity.settings.updated",
			Response: output, OccurredAt: now,
		})
	})
	return output, err
}

func settingsFromRecord(record identitystore.UserSetting) Settings {
	return Settings{
		SubjectID: record.SubjectID, NotificationPreferences: append(json.RawMessage(nil), record.NotificationPreferences...),
		Locale: record.Locale, Timezone: record.Timezone, Revision: record.Revision,
	}
}

func effectiveRole(actor Principal) (Role, error) {
	if strings.TrimSpace(actor.SubjectID) == "" || len(actor.Roles) == 0 {
		return "", ErrInvalidProfile
	}
	for _, role := range actor.Roles {
		switch role {
		case RoleInspector, RoleLeadInspector, RoleDepartmentManager, RoleGeneralManager,
			RoleFinance, RoleExecutiveDirector, RoleAuditee, RoleAdmin:
			return role, nil
		}
	}
	return "", ErrInvalidProfile
}

func lockAndReplay(
	ctx context.Context,
	transaction pgx.Tx,
	scope string,
	operationID string,
	idempotencyKey string,
	semanticHash string,
	output any,
) (bool, error) {
	if _, err := transaction.Exec(
		ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
		scope+":idempotency:"+idempotencyKey,
	); err != nil {
		return false, err
	}
	if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope+":"+operationID); err != nil {
		return false, err
	}
	var storedHash string
	var responseBody []byte
	err := transaction.QueryRow(ctx, `
		SELECT semantic_hash, response_body
		FROM idempotency_responses
		WHERE scope = $1 AND operation_id = $2
	`, scope, operationID).Scan(&storedHash, &responseBody)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		err := transaction.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM outbox_messages
				WHERE idempotency_key = $1
			)
		`, commandIdempotencyKey(scope, idempotencyKey)).Scan(&exists)
		if err != nil {
			return false, err
		}
		if exists {
			return false, idempotency.ErrOperationIDReuse
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if storedHash != semanticHash {
		return false, idempotency.ErrOperationIDReuse
	}
	return true, json.Unmarshal(responseBody, output)
}

type envelope struct {
	Scope          string
	OperationID    string
	IdempotencyKey string
	SemanticHash   string
	Action         string
	EntityType     string
	EntityID       string
	EntityVersion  int64
	OutboxTopic    string
	Response       any
	OccurredAt     time.Time
}

func (service *ProfileService) persistEnvelope(
	ctx context.Context,
	transaction pgx.Tx,
	actor Principal,
	record envelope,
) error {
	responseBody, err := json.Marshal(record.Response)
	if err != nil {
		return err
	}
	auditEventID := service.idGenerator("audit-identity")
	outboxMessageID := service.idGenerator("outbox-identity")
	actorRole, _ := effectiveRole(actor)
	if _, err := transaction.Exec(ctx, `
		INSERT INTO audit_events (
			event_id, occurred_at, actor_subject_id, actor_role, organization_id,
			action, entity_type, entity_id, entity_version, operation_id,
			correlation_id, request_id, details
		) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, $10, $10, $10, '{}'::jsonb)
	`, auditEventID, record.OccurredAt, actor.SubjectID, string(actorRole), actor.OrganizationID,
		record.Action, record.EntityType, record.EntityID, record.EntityVersion, record.OperationID); err != nil {
		return fmt.Errorf("append identity audit event: %w", err)
	}
	var changeSequenceID int64
	if err := transaction.QueryRow(ctx, `
		INSERT INTO authorized_sync_changes (
			subject_id, organization_id, kind, entity_id, entity_revision,
			payload, changed_at, operation_id, correlation_id
		) VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7, $8, $8)
		RETURNING sequence_id
	`, actor.SubjectID, actor.OrganizationID, record.EntityType, record.EntityID,
		record.EntityVersion, responseBody, record.OccurredAt, record.OperationID).Scan(&changeSequenceID); err != nil {
		return fmt.Errorf("append identity change: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO outbox_messages (
			id, topic, aggregate_type, aggregate_id, payload, available_at,
			idempotency_key, operation_id, correlation_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
	`, outboxMessageID, record.OutboxTopic, record.EntityType, record.EntityID,
		responseBody, record.OccurredAt, commandIdempotencyKey(record.Scope, record.IdempotencyKey),
		record.OperationID); err != nil {
		return fmt.Errorf("enqueue identity outbox: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO idempotency_responses (
			scope, operation_id, semantic_hash, response_status,
			response_headers, response_body, created_at
		) VALUES ($1, $2, $3, 200, '{}'::jsonb, $4, $5)
	`, record.Scope, record.OperationID, record.SemanticHash, responseBody, record.OccurredAt); err != nil {
		return fmt.Errorf("store identity idempotency response: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO command_transaction_links (
			operation_id, idempotency_scope, audit_event_id,
			change_sequence_id, outbox_message_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, record.OperationID, record.Scope, auditEventID, changeSequenceID,
		outboxMessageID, record.OccurredAt); err != nil {
		return fmt.Errorf("link identity command transaction: %w", err)
	}
	return nil
}

func commandIdempotencyKey(scope, idempotencyKey string) string {
	return "command:" + scope + ":idempotency:" + idempotencyKey
}

func randomProfileID(prefix string) string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(fmt.Sprintf("secure identity identifier generation failed: %v", err))
	}
	return prefix + "-" + hex.EncodeToString(bytes[:])
}
