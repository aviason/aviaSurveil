package notifications

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/telemetry"
	"github.com/jackc/pgx/v5"
)

type EmailDelivery struct {
	JobID                   string
	NotificationID          string
	RecipientSubjectID      string
	RecipientEmail          string
	RecipientAudience       EmailAudience
	RecipientOrganizationID string
	OrganizationID          string
	OrganizationName        string
	Title                   string
	Body                    string
	InternalContext         string
	RelatedEntityType       string
	RelatedEntityID         string
	ProviderMessageID       string
	Attempt                 int
}

type DeliveryAdapter interface {
	Deliver(context.Context, EmailDelivery) error
}

type DeliveryDependencies struct {
	Clock       func() time.Time
	IDGenerator func(string) string
	Adapter     DeliveryAdapter
	WorkerID    string
	Lease       time.Duration
	RetryDelay  func(int) time.Duration
	MaxAttempts int
}

type DeliveryService struct {
	pool        *database.Pool
	clock       func() time.Time
	idGenerator func(string) string
	adapter     DeliveryAdapter
	workerID    string
	lease       time.Duration
	retryDelay  func(int) time.Duration
	maxAttempts int
}

func NewDeliveryService(
	pool *database.Pool,
	dependencies DeliveryDependencies,
) *DeliveryService {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = deliveryRandomID
	}
	workerID := dependencies.WorkerID
	if workerID == "" {
		workerID = "notification-delivery-worker"
	}
	lease := dependencies.Lease
	if lease <= 0 {
		lease = time.Minute
	}
	retryDelay := dependencies.RetryDelay
	if retryDelay == nil {
		retryDelay = notificationRetryDelay
	}
	maxAttempts := dependencies.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return &DeliveryService{
		pool: pool, clock: clock, idGenerator: idGenerator,
		adapter: dependencies.Adapter, workerID: workerID, lease: lease,
		retryDelay: retryDelay, maxAttempts: maxAttempts,
	}
}

type claimedDelivery struct {
	EmailDelivery
	OutboxMessageID string
	TraceParent     string
	CorrelationID   string
	AvailableAt     time.Time
}

func (service *DeliveryService) ProcessNext(
	ctx context.Context,
) (processed bool, resultErr error) {
	if service.pool == nil || service.adapter == nil {
		return false, errors.New("notification delivery requires a database and adapter")
	}
	delivery, claimed, err := service.claimNext(ctx)
	if err != nil || !claimed {
		return claimed, err
	}
	jobContext, span := telemetry.StartPersistedJob(
		ctx,
		delivery.TraceParent,
		delivery.CorrelationID,
		"email",
		"mailpit",
	)
	telemetry.RecordPersistedOutboxReadyAge(
		jobContext,
		"email",
		"notification",
		delivery.AvailableAt,
		service.clock().UTC(),
	)
	defer func() {
		telemetry.FinishPersistedJob(
			jobContext,
			span,
			"email",
			"mailpit",
			resultErr,
		)
	}()
	delivery.ProviderMessageID = stableProviderMessageID(delivery.JobID)
	providerErr := validateClaimedDelivery(delivery.EmailDelivery)
	if providerErr == nil {
		providerErr = service.adapter.Deliver(jobContext, delivery.EmailDelivery)
	}
	if err := service.finalize(jobContext, delivery, providerErr); err != nil {
		return true, err
	}
	return true, providerErr
}

func (service *DeliveryService) claimNext(
	ctx context.Context,
) (claimedDelivery, bool, error) {
	var delivery claimedDelivery
	claimed := false
	now := service.clock().UTC()
	err := database.WithinTransaction(
		ctx,
		service.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			err := transaction.QueryRow(ctx, `
				SELECT job.id, job.notification_id,
				       job.recipient_subject_id,
				       COALESCE(recipient.email, ''),
				       CASE
				           WHEN recipient_organization.organization_type = 'AUTHORITY'
				           THEN 'CAA'
				           ELSE 'AUDITEE'
				       END,
				       COALESCE(
				           recipient_profile.organization_id,
				           recipient_session.organization_id,
				           ''
				       ),
				       COALESCE(record.organization_id, ''),
				       COALESCE(record_organization.legal_name, ''),
				       record.title, record.body,
				       COALESCE(record.related_entity_type, ''),
				       COALESCE(record.related_entity_id, ''),
				       job.attempt_count, job.outbox_message_id,
				       COALESCE(outbox.traceparent, ''),
				       COALESCE(outbox.correlation_id, ''),
				       outbox.available_at
				FROM notification_delivery_jobs job
				JOIN notification_records record
				  ON record.id = job.notification_id
				JOIN identity_references recipient
				  ON recipient.subject_id = job.recipient_subject_id
				 AND recipient.tombstoned_at IS NULL
				JOIN outbox_messages outbox
				  ON outbox.id = job.outbox_message_id
				LEFT JOIN user_profiles recipient_profile
				  ON recipient_profile.subject_id = recipient.subject_id
				 AND recipient_profile.tombstoned_at IS NULL
				LEFT JOIN LATERAL (
				    SELECT session.organization_id
				    FROM session_references session
				    WHERE session.subject_id = recipient.subject_id
				      AND session.revoked_at IS NULL
				      AND session.expires_at > $1
				      AND session.absolute_expires_at > $1
				    ORDER BY session.created_at DESC, session.id DESC
				    LIMIT 1
				) recipient_session ON true
				LEFT JOIN organizations recipient_organization
				  ON recipient_organization.id = COALESCE(
				      recipient_profile.organization_id,
				      recipient_session.organization_id
				  )
				LEFT JOIN organizations record_organization
				  ON record_organization.id = record.organization_id
				WHERE job.channel = 'EMAIL'
				  AND job.status IN ('PENDING', 'FAILED')
				  AND (
				      job.next_attempt_at IS NULL
				      OR job.next_attempt_at <= $1
				  )
				  AND outbox.delivered_at IS NULL
				  AND outbox.terminal_state IS NULL
				  AND outbox.available_at <= $1
				  AND (
				      outbox.lease_expires_at IS NULL
				      OR outbox.lease_expires_at <= $1
				  )
				ORDER BY job.created_at, job.id
				FOR UPDATE OF job, outbox SKIP LOCKED
				LIMIT 1
			`, now).Scan(
				&delivery.JobID,
				&delivery.NotificationID,
				&delivery.RecipientSubjectID,
				&delivery.RecipientEmail,
				&delivery.RecipientAudience,
				&delivery.RecipientOrganizationID,
				&delivery.OrganizationID,
				&delivery.OrganizationName,
				&delivery.Title,
				&delivery.Body,
				&delivery.RelatedEntityType,
				&delivery.RelatedEntityID,
				&delivery.Attempt,
				&delivery.OutboxMessageID,
				&delivery.TraceParent,
				&delivery.CorrelationID,
				&delivery.AvailableAt,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			claimed = true
			delivery.Attempt++
			if _, err := transaction.Exec(ctx, `
				UPDATE notification_delivery_jobs
				SET attempt_count = $2,
				    updated_at = $3
				WHERE id = $1
			`, delivery.JobID, delivery.Attempt, now); err != nil {
				return err
			}
			result, err := transaction.Exec(ctx, `
				UPDATE outbox_messages
				SET claimed_at = $2,
				    lease_owner = $3,
				    lease_expires_at = $4,
				    attempt_count = attempt_count + 1
				WHERE id = $1
				  AND delivered_at IS NULL
				  AND terminal_state IS NULL
			`, delivery.OutboxMessageID, now, service.workerID,
				now.Add(service.lease))
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return errors.New("notification delivery claim changed")
			}
			return nil
		},
	)
	return delivery, claimed, err
}

func (service *DeliveryService) finalize(
	ctx context.Context,
	delivery claimedDelivery,
	providerErr error,
) error {
	return database.WithinTransaction(
		ctx,
		service.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			now := service.clock().UTC()
			var leaseOwner string
			if err := transaction.QueryRow(ctx, `
				SELECT COALESCE(lease_owner, '')
				FROM outbox_messages
				WHERE id = $1
				  AND delivered_at IS NULL
				FOR UPDATE
			`, delivery.OutboxMessageID).Scan(&leaseOwner); err != nil {
				return err
			}
			if leaseOwner != service.workerID {
				return errors.New("notification delivery lease changed")
			}
			status := "DELIVERED"
			failureCode := ""
			action := "NOTIFICATION_EMAIL_DELIVERED"
			var nextAttemptAt *time.Time
			var terminalAt *time.Time
			providerMessageID := delivery.ProviderMessageID
			var acceptedAt *time.Time
			if providerErr == nil {
				acceptedAt = &now
			} else {
				providerMessageID = ""
				failureCode = DeliveryFailureCode(providerErr)
				permanent := IsPermanentDeliveryFailure(providerErr) ||
					delivery.Attempt >= service.maxAttempts
				if permanent {
					status = "DEAD_LETTER"
					action = "NOTIFICATION_EMAIL_DEAD_LETTERED"
					terminalAt = &now
				} else {
					status = "FAILED"
					action = "NOTIFICATION_EMAIL_DELIVERY_FAILED"
					next := now.Add(service.retryDelay(delivery.Attempt))
					nextAttemptAt = &next
				}
			}
			result, err := transaction.Exec(ctx, `
				UPDATE notification_delivery_jobs
				SET status = $2,
				    last_error = NULLIF($3, ''),
				    provider_message_id = NULLIF($4, ''),
				    accepted_at = $5,
				    next_attempt_at = $6,
				    terminal_at = $7,
				    updated_at = $8
				WHERE id = $1
				  AND attempt_count = $9
			`, delivery.JobID, status, failureCode, providerMessageID,
				acceptedAt, nextAttemptAt, terminalAt, now, delivery.Attempt)
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return errors.New("notification delivery job state changed")
			}
			if status == "DELIVERED" {
				result, err = transaction.Exec(ctx, `
					UPDATE outbox_messages
					SET delivered_at = $2,
					    claimed_at = NULL,
					    lease_owner = NULL,
					    lease_expires_at = NULL,
					    last_error = NULL
					WHERE id = $1
					  AND lease_owner = $3
					  AND delivered_at IS NULL
				`, delivery.OutboxMessageID, now, service.workerID)
			} else if status == "FAILED" {
				result, err = transaction.Exec(ctx, `
					UPDATE outbox_messages
					SET claimed_at = NULL,
					    lease_owner = NULL,
					    lease_expires_at = NULL,
					    last_error = $2,
					    available_at = $3
					WHERE id = $1
					  AND lease_owner = $4
					  AND delivered_at IS NULL
				`, delivery.OutboxMessageID, failureCode,
					*nextAttemptAt, service.workerID)
			} else {
				result, err = transaction.Exec(ctx, `
					UPDATE outbox_messages
					SET claimed_at = NULL,
					    lease_owner = NULL,
					    lease_expires_at = NULL,
					    last_error = $2,
					    terminal_state = 'PERMANENT_FAILURE'
					WHERE id = $1
					  AND lease_owner = $3
					  AND delivered_at IS NULL
					  AND terminal_state IS NULL
				`, delivery.OutboxMessageID, failureCode, service.workerID)
			}
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return errors.New("notification delivery outbox state changed")
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_events (
					event_id, occurred_at, actor_role, organization_id,
					action, entity_type, entity_id, entity_version,
					before_status, after_status, request_id, details
				) VALUES (
					$1, $2, 'SYSTEM', NULLIF($3, ''), $4,
					'NOTIFICATION_DELIVERY', $5, $6,
					'', $7, $1,
					jsonb_build_object(
						'notificationId', $8::text,
						'recipientSubjectId', $9::text,
						'attempt', $6::bigint,
						'failureCode', NULLIF($10::text, ''),
						'providerMessageId', NULLIF($11::text, ''),
						'nextAttemptAt', $12::timestamptz
					)
				)
			`, service.idGenerator("audit"), now,
				delivery.OrganizationID, action, delivery.JobID,
				delivery.Attempt, status, delivery.NotificationID,
				delivery.RecipientSubjectID, failureCode,
				providerMessageID, nextAttemptAt); err != nil {
				return err
			}
			return nil
		},
	)
}

func validateClaimedDelivery(delivery EmailDelivery) error {
	if strings.TrimSpace(delivery.RecipientEmail) == "" {
		return NewPermanentDeliveryFailure("SMTP_RECIPIENT_NOT_CONFIGURED")
	}
	if delivery.RecipientAudience != EmailAudienceCAA &&
		delivery.RecipientAudience != EmailAudienceAuditee {
		return NewPermanentDeliveryFailure("SMTP_RECIPIENT_SCOPE_INVALID")
	}
	if delivery.RecipientAudience == EmailAudienceAuditee &&
		(strings.TrimSpace(delivery.OrganizationID) == "" ||
			delivery.RecipientOrganizationID != delivery.OrganizationID) {
		return NewPermanentDeliveryFailure("SMTP_RECIPIENT_SCOPE_INVALID")
	}
	return nil
}

func notificationRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := 30 * time.Second
	for current := 1; current < attempt && delay < 15*time.Minute; current++ {
		delay *= 2
	}
	if delay > 15*time.Minute {
		return 15 * time.Minute
	}
	return delay
}

func deliveryRandomID(prefix string) string {
	var buffer [16]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		panic(fmt.Sprintf("generate %s id: %v", prefix, err))
	}
	return prefix + "-" + hex.EncodeToString(buffer[:])
}
