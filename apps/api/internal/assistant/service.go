package assistant

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden = errors.New("assistant advisory authority required")
	ErrInvalid   = errors.New("invalid assistant advisory request")
	ErrNotFound  = errors.New("assistant Finding context not found")
)

const (
	advisoryNonDecisionLabel = "Advisory draft — cannot create, classify, close, or enforce a Finding."
	maxPromptLength          = 2_000
)

type Guidance struct {
	AdvisoryOnly      bool     `json:"advisoryOnly"`
	ProhibitedActions []string `json:"prohibitedActions"`
	NonDecisionLabel  string   `json:"nonDecisionLabel"`
}

type CreateDraftCommand struct {
	OperationID      string
	IdempotencyKey   string
	ExpectedRevision *int64
	FindingID        string
	Prompt           string
}

type Draft struct {
	ID               string    `json:"id"`
	FindingID        string    `json:"findingId"`
	Prompt           string    `json:"prompt"`
	Draft            string    `json:"draft"`
	AdvisoryOnly     bool      `json:"advisoryOnly"`
	CanCreateFinding bool      `json:"canCreateFinding"`
	CanSetSeverity   bool      `json:"canSetSeverity"`
	CanCloseFinding  bool      `json:"canCloseFinding"`
	ProviderID       string    `json:"providerId"`
	GeneratedAt      time.Time `json:"generatedAt"`
	NonDecisionLabel string    `json:"nonDecisionLabel"`
}

type Dependencies struct {
	Clock       func() time.Time
	IDGenerator func(string) string
	Provider    Provider
}

type Service struct {
	pool        *database.Pool
	clock       func() time.Time
	idGenerator func(string) string
	provider    Provider
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
	provider := dependencies.Provider
	if provider == nil {
		provider = NewDeterministicProvider()
	}
	return &Service{
		pool: pool, clock: clock, idGenerator: idGenerator, provider: provider,
	}
}

func (service *Service) GetGuidance(actor identity.Principal) (Guidance, error) {
	if !canUseAssistant(actor) {
		return Guidance{}, ErrForbidden
	}
	return Guidance{
		AdvisoryOnly: true,
		ProhibitedActions: []string{
			"create Finding", "set severity", "close Finding", "enforcement action",
		},
		NonDecisionLabel: advisoryNonDecisionLabel,
	}, nil
}

func (service *Service) CreateDraft(
	ctx context.Context,
	actor identity.Principal,
	command CreateDraftCommand,
) (Draft, error) {
	if !canUseAssistant(actor) {
		return Draft{}, ErrForbidden
	}
	command.OperationID = strings.TrimSpace(command.OperationID)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.FindingID = strings.TrimSpace(command.FindingID)
	command.Prompt = strings.TrimSpace(command.Prompt)
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		command.FindingID == "" || command.Prompt == "" ||
		len(command.Prompt) > maxPromptLength || command.ExpectedRevision != nil {
		return Draft{}, ErrInvalid
	}
	semanticHash, err := idempotency.SemanticHash(struct {
		IdempotencyKey string `json:"idempotencyKey"`
		FindingID      string `json:"findingId"`
		Prompt         string `json:"prompt"`
	}{
		IdempotencyKey: command.IdempotencyKey,
		FindingID:      command.FindingID,
		Prompt:         command.Prompt,
	})
	if err != nil {
		return Draft{}, err
	}
	scope := actor.SubjectID + ":assistant_advisory_draft"
	var output Draft
	err = database.WithinTransaction(ctx, service.pool, func(
		ctx context.Context,
		transaction pgx.Tx,
	) error {
		if _, err := transaction.Exec(
			ctx,
			"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
			scope+":idempotency:"+command.IdempotencyKey,
		); err != nil {
			return err
		}
		if _, err := transaction.Exec(
			ctx,
			"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
			scope+":operation:"+command.OperationID,
		); err != nil {
			return err
		}
		var storedHash string
		var responseBody []byte
		err := transaction.QueryRow(ctx, `
			SELECT semantic_hash, response_body
			FROM idempotency_responses
			WHERE scope = $1 AND operation_id = $2
		`, scope, command.OperationID).Scan(&storedHash, &responseBody)
		if err == nil {
			if storedHash != semanticHash {
				return idempotency.ErrOperationIDReuse
			}
			return json.Unmarshal(responseBody, &output)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var reusedOperationID string
		err = transaction.QueryRow(ctx, `
			SELECT link.operation_id
			FROM command_transaction_links link
			JOIN idempotency_responses response
			  ON response.scope = link.idempotency_scope
			 AND response.operation_id = link.operation_id
			WHERE link.idempotency_scope = $1
			  AND response.response_headers ->> 'idempotencyKey' = $2
			LIMIT 1
		`, scope, command.IdempotencyKey).Scan(&reusedOperationID)
		if err == nil {
			return idempotency.ErrOperationIDReuse
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		var findingReference, assignedInspectorSubjectID string
		err = transaction.QueryRow(ctx, `
			SELECT finding.reference, inspection.assigned_inspector_subject_id
			FROM findings finding
			JOIN inspections inspection ON inspection.id = finding.inspection_id
			WHERE finding.id = $1
			  AND finding.tombstoned_at IS NULL
		`, command.FindingID).Scan(&findingReference, &assignedInspectorSubjectID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if actor.HasRole(identity.RoleInspector) &&
			assignedInspectorSubjectID != actor.SubjectID {
			return ErrNotFound
		}
		providerResponse, err := service.provider.Generate(ctx, ProviderRequest{
			FindingID: command.FindingID, FindingReference: findingReference,
			Prompt: command.Prompt,
		})
		if err != nil {
			return fmt.Errorf("generate advisory draft: %w", err)
		}
		providerResponse.Text = strings.TrimSpace(providerResponse.Text)
		providerResponse.ProviderID = strings.TrimSpace(providerResponse.ProviderID)
		if providerResponse.Text == "" || providerResponse.ProviderID == "" {
			return fmt.Errorf("%w: provider returned incomplete provenance", ErrInvalid)
		}
		now := service.clock().UTC()
		output = Draft{
			ID: service.idGenerator("assistant-draft"), FindingID: command.FindingID,
			Prompt: command.Prompt, Draft: providerResponse.Text,
			AdvisoryOnly: true, CanCreateFinding: false, CanSetSeverity: false,
			CanCloseFinding: false, ProviderID: providerResponse.ProviderID,
			GeneratedAt: now, NonDecisionLabel: advisoryNonDecisionLabel,
		}
		responseBody, err = json.Marshal(output)
		if err != nil {
			return err
		}
		details, err := json.Marshal(map[string]any{
			"advisoryOnly":     true,
			"nonDecisionLabel": advisoryNonDecisionLabel,
			"providerId":       providerResponse.ProviderID,
			"providerInputFields": []string{
				"findingId", "findingReference", "prompt",
			},
			"promptStoredInAudit": false,
		})
		if err != nil {
			return err
		}
		actorRole := identity.RoleInspector
		if actor.HasRole(identity.RoleLeadInspector) {
			actorRole = identity.RoleLeadInspector
		}
		auditEventID := service.idGenerator("audit-assistant-draft")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_events (
				event_id, occurred_at, actor_subject_id, actor_role, organization_id,
				action, entity_type, entity_id, entity_version, after_status,
				operation_id, correlation_id, request_id, details
			) VALUES (
				$1, $2, $3, $4, NULLIF($5, ''),
				'assistant.advisory_draft_generated', 'ASSISTANT_DRAFT', $6, 1,
				'ADVISORY_ONLY', $7, $7, $7, $8
			)
		`, auditEventID, now, actor.SubjectID, string(actorRole),
			actor.OrganizationID, output.ID, command.OperationID, details); err != nil {
			return fmt.Errorf("append assistant audit event: %w", err)
		}
		var changeSequenceID int64
		if err := transaction.QueryRow(ctx, `
			INSERT INTO authorized_sync_changes (
				subject_id, organization_id, kind, entity_id, entity_revision,
				payload, changed_at, operation_id, correlation_id
			) VALUES (
				$1, NULLIF($2, ''), 'ASSISTANT_ADVISORY_DRAFT', $3, 1,
				$4, $5, $6, $6
			)
			RETURNING sequence_id
		`, actor.SubjectID, actor.OrganizationID, output.ID, responseBody, now,
			command.OperationID).Scan(&changeSequenceID); err != nil {
			return fmt.Errorf("append assistant authorized change: %w", err)
		}
		outboxMessageID := service.idGenerator("outbox-assistant-draft")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO outbox_messages (
				id, topic, aggregate_type, aggregate_id, payload, available_at,
				idempotency_key, operation_id, correlation_id
			) VALUES (
				$1, 'assistant.advisory-draft.generated', 'ASSISTANT_DRAFT', $2,
				$3, $4, $5, $6, $6
			)
		`, outboxMessageID, output.ID, responseBody, now,
			"command:"+scope+":idempotency:"+command.IdempotencyKey,
			command.OperationID); err != nil {
			return fmt.Errorf("enqueue assistant advisory event: %w", err)
		}
		responseHeaders, err := json.Marshal(map[string]string{
			"idempotencyKey": command.IdempotencyKey,
		})
		if err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO idempotency_responses (
				scope, operation_id, semantic_hash, response_status,
				response_headers, response_body, created_at
			) VALUES ($1, $2, $3, 200, $4, $5, $6)
		`, scope, command.OperationID, semanticHash, responseHeaders,
			responseBody, now); err != nil {
			return fmt.Errorf("store assistant idempotent response: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO command_transaction_links (
				operation_id, idempotency_scope, audit_event_id,
				change_sequence_id, outbox_message_id, created_at
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, command.OperationID, scope, auditEventID, changeSequenceID,
			outboxMessageID, now); err != nil {
			return fmt.Errorf("link assistant command transaction: %w", err)
		}
		return nil
	})
	return output, err
}

func canUseAssistant(actor identity.Principal) bool {
	return actor.HasRole(identity.RoleInspector, identity.RoleLeadInspector)
}

func randomID(prefix string) string {
	var value [12]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(fmt.Sprintf("generate %s ID: %v", prefix, err))
	}
	return prefix + "-" + hex.EncodeToString(value[:])
}
