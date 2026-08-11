package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/communications"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/notifications"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type CommunicationsWorkflowDependencies struct {
	Clock       func() time.Time
	IDGenerator func(string) string
}

type CommunicationsWorkflow struct {
	pool    *database.Pool
	service *Service
}

func NewCommunicationsWorkflow(
	pool *database.Pool,
	dependencies CommunicationsWorkflowDependencies,
) *CommunicationsWorkflow {
	return &CommunicationsWorkflow{
		pool: pool,
		service: NewService(pool, Dependencies{
			Clock:       dependencies.Clock,
			IDGenerator: dependencies.IDGenerator,
		}),
	}
}

type SendCommunicationCommand struct {
	OperationID    string
	CorrelationID  string
	IdempotencyKey string
	OrganizationID string
	Subject        string
	Body           string
	Audience       communications.Audience
}

func (workflow *CommunicationsWorkflow) SendCommunication(
	ctx context.Context,
	actor identity.Principal,
	command SendCommunicationCommand,
) (communications.Message, error) {
	command.OrganizationID = strings.TrimSpace(command.OrganizationID)
	command.Subject = strings.TrimSpace(command.Subject)
	command.Body = strings.TrimSpace(command.Body)
	policy, err := communications.ResolvePolicy(
		actor,
		command.OrganizationID,
		command.Audience,
	)
	if err != nil {
		return communications.Message{}, mapCommunicationError(err)
	}
	if command.Subject == "" || command.Body == "" ||
		command.IdempotencyKey == "" {
		return communications.Message{}, ErrInvalid
	}
	semantic := struct {
		OrganizationID string                  `json:"organizationId"`
		Subject        string                  `json:"subject"`
		Body           string                  `json:"body"`
		Audience       communications.Audience `json:"audience"`
	}{
		OrganizationID: command.OrganizationID,
		Subject:        command.Subject,
		Body:           command.Body,
		Audience:       command.Audience,
	}
	return executeTransition(
		ctx,
		workflow.service,
		actor,
		commandEnvelope{
			OperationID:    command.OperationID,
			IdempotencyKey: command.IdempotencyKey,
			CorrelationID:  command.CorrelationID,
			Kind:           "send_communication",
			EntityID:       "communication:" + command.IdempotencyKey,
			Semantic:       semantic,
		},
		func(
			ctx context.Context,
			transaction pgx.Tx,
		) (transition[communications.Message], error) {
			if command.OrganizationID != "" {
				var active bool
				if err := transaction.QueryRow(ctx, `
					SELECT status = 'ACTIVE'
					FROM organizations
					WHERE id = $1
				`, command.OrganizationID).Scan(&active); err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return transition[communications.Message]{}, ErrNotFound
					}
					return transition[communications.Message]{}, err
				}
				if !active {
					return transition[communications.Message]{}, ErrForbidden
				}
			}
			now := workflow.service.clock().UTC()
			threadID := workflow.service.idGenerator("communication-thread")
			messageID := workflow.service.idGenerator("communication-message")
			if _, err := transaction.Exec(ctx, `
				INSERT INTO communication_threads (
					id, organization_id, visibility, subject, revision,
					created_at, updated_at
				) VALUES (
					$1, NULLIF($2, ''), $3, $4, 1, $5, $5
				)
			`, threadID, command.OrganizationID, policy.Visibility,
				command.Subject, now); err != nil {
				return transition[communications.Message]{}, err
			}
			message := communications.Message{
				ID:              messageID,
				ThreadID:        threadID,
				OrganizationID:  command.OrganizationID,
				Visibility:      policy.Visibility,
				SenderSubjectID: actor.SubjectID,
				Audience:        command.Audience,
				Direction:       policy.Direction,
				Subject:         command.Subject,
				Body:            command.Body,
				Revision:        1,
				CreatedAt:       now,
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO communication_messages (
					id, thread_id, organization_id, visibility,
					sender_subject_id, audience, direction, subject, body,
					idempotency_key, revision, created_at
				) VALUES (
					$1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8, $9, $10, 1, $11
				)
			`, message.ID, message.ThreadID, message.OrganizationID,
				message.Visibility, message.SenderSubjectID, message.Audience,
				message.Direction, message.Subject, message.Body,
				command.IdempotencyKey, now); err != nil {
				return transition[communications.Message]{}, err
			}
			recipients, err := workflow.communicationRecipients(
				ctx,
				transaction,
				actor,
				message,
			)
			if err != nil {
				return transition[communications.Message]{}, err
			}
			for _, recipient := range recipients {
				title := "New CAA communication"
				if message.Direction == communications.DirectionAuditeeToCAA {
					title = "New Auditee communication"
				} else if message.Direction == communications.DirectionCAAInternal {
					title = "New Internal CAA Note"
				}
				body := fmt.Sprintf(
					"%s — open the authorized message record for details.",
					message.Subject,
				)
				if _, err := workflow.createNotification(
					ctx,
					transaction,
					recipient,
					message.OrganizationID,
					title,
					body,
					"COMMUNICATION",
					message.ID,
					"communication:"+message.ID+":recipient:"+recipient,
					now,
				); err != nil {
					return transition[communications.Message]{}, err
				}
			}
			organizationID := message.OrganizationID
			if organizationID == "" {
				organizationID = actor.OrganizationID
			}
			return transition[communications.Message]{
				Response:       message,
				OrganizationID: organizationID,
				Action:         "COMMUNICATION_SENT",
				EntityType:     "COMMUNICATION",
				EntityID:       message.ID,
				EntityVersion:  message.Revision,
				BeforeStatus:   "",
				AfterStatus:    string(message.Visibility),
				SyncKind:       "communication",
				OutboxTopic:    "communications.message_sent",
			}, nil
		},
	)
}

func (workflow *CommunicationsWorkflow) communicationRecipients(
	ctx context.Context,
	transaction pgx.Tx,
	actor identity.Principal,
	message communications.Message,
) ([]string, error) {
	query := `
		SELECT DISTINCT subject_id
		FROM session_references
		WHERE revoked_at IS NULL
		  AND subject_id <> $1
		  AND (
		      ($2 = 'AUDITEE' AND organization_id = $3
		       AND roles @> ARRAY['auditee']::text[])
		      OR
		      ($2 = 'CAA' AND roles && ARRAY[
		          'inspector', 'leadInspector', 'manager'
		      ]::text[])
		  )
		ORDER BY subject_id
	`
	rows, err := transaction.Query(
		ctx,
		query,
		actor.SubjectID,
		message.Audience,
		message.OrganizationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recipients := []string{}
	for rows.Next() {
		var subjectID string
		if err := rows.Scan(&subjectID); err != nil {
			return nil, err
		}
		recipients = append(recipients, subjectID)
	}
	return recipients, rows.Err()
}

func (workflow *CommunicationsWorkflow) createNotification(
	ctx context.Context,
	transaction pgx.Tx,
	recipientSubjectID string,
	organizationID string,
	title string,
	body string,
	relatedEntityType string,
	relatedEntityID string,
	deduplicationKey string,
	now time.Time,
) (notifications.Notification, error) {
	notification := notifications.Notification{
		ID:                  workflow.service.idGenerator("notification"),
		RecipientSubjectID:  recipientSubjectID,
		OrganizationID:      organizationID,
		Title:               title,
		Body:                body,
		RelatedEntityType:   relatedEntityType,
		RelatedEntityID:     relatedEntityID,
		DeduplicationKey:    deduplicationKey,
		EmailDeliveryStatus: notifications.EmailDeliveryPending,
		Revision:            1,
		CreatedAt:           now,
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO notification_records (
			id, recipient_subject_id, organization_id, title, body,
			related_entity_type, related_entity_id, deduplication_key,
			revision, created_at
		) VALUES (
			$1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8, 1, $9
		)
	`, notification.ID, notification.RecipientSubjectID,
		notification.OrganizationID, notification.Title, notification.Body,
		notification.RelatedEntityType, notification.RelatedEntityID,
		notification.DeduplicationKey, notification.CreatedAt); err != nil {
		return notifications.Notification{}, err
	}
	payload, err := json.Marshal(notification)
	if err != nil {
		return notifications.Notification{}, err
	}
	outboxID := workflow.service.idGenerator("outbox")
	emailIdempotencyKey := "notification-email:" + notification.DeduplicationKey
	if _, err := transaction.Exec(ctx, `
		INSERT INTO outbox_messages (
			id, topic, aggregate_type, aggregate_id, payload,
			available_at, idempotency_key
		) VALUES (
			$1, 'notification.email_requested', 'NOTIFICATION', $2,
			$3, $4, $5
		)
	`, outboxID, notification.ID, payload, now, emailIdempotencyKey); err != nil {
		return notifications.Notification{}, err
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO notification_delivery_jobs (
			id, notification_id, recipient_subject_id, channel, status,
			idempotency_key, outbox_message_id, attempt_count,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, 'EMAIL', 'PENDING', $4, $5, 0, $6, $6
		)
	`, workflow.service.idGenerator("notification-delivery"),
		notification.ID, recipientSubjectID, emailIdempotencyKey,
		outboxID, now); err != nil {
		return notifications.Notification{}, err
	}
	return notification, nil
}

func (workflow *CommunicationsWorkflow) ListCommunications(
	ctx context.Context,
	actor identity.Principal,
	organizationID string,
) ([]communications.Message, error) {
	organizationID = strings.TrimSpace(organizationID)
	if actor.HasRole(identity.RoleAuditee) {
		if organizationID != "" && organizationID != actor.OrganizationID {
			return nil, ErrForbidden
		}
		organizationID = actor.OrganizationID
	} else if !communications.CanUseCommunications(actor) {
		return nil, ErrForbidden
	}
	rows, err := workflow.pool.Query(ctx, `
		SELECT id, thread_id, COALESCE(organization_id, ''), visibility,
		       sender_subject_id, audience, direction, subject, body,
		       revision, created_at
		FROM communication_messages
		WHERE (
			$1 = false
			OR (
				organization_id = $2
				AND (
					$3 = false
					OR (
						visibility = 'AUDITEE_VISIBLE'
						AND (
							(direction = 'CAA_TO_AUDITEE' AND audience = 'AUDITEE')
							OR (
								direction = 'AUDITEE_TO_CAA'
								AND audience = 'CAA'
								AND sender_subject_id = $4
							)
						)
					)
				)
			)
		)
		ORDER BY created_at DESC, id DESC
	`, organizationID != "", organizationID,
		actor.HasRole(identity.RoleAuditee), actor.SubjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []communications.Message{}
	for rows.Next() {
		message, err := scanCommunication(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, message)
	}
	return items, rows.Err()
}

func (workflow *CommunicationsWorkflow) GetCommunication(
	ctx context.Context,
	actor identity.Principal,
	messageID string,
) (communications.Message, error) {
	message, err := workflow.getCommunication(ctx, workflow.pool, messageID)
	if err != nil {
		return communications.Message{}, err
	}
	if !communications.CanRead(actor, message) {
		return communications.Message{}, ErrNotFound
	}
	return message, nil
}

type communicationRow interface {
	Scan(dest ...any) error
}

func scanCommunication(row communicationRow) (communications.Message, error) {
	var message communications.Message
	err := row.Scan(
		&message.ID,
		&message.ThreadID,
		&message.OrganizationID,
		&message.Visibility,
		&message.SenderSubjectID,
		&message.Audience,
		&message.Direction,
		&message.Subject,
		&message.Body,
		&message.Revision,
		&message.CreatedAt,
	)
	return message, err
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (workflow *CommunicationsWorkflow) getCommunication(
	ctx context.Context,
	queryer queryRower,
	messageID string,
) (communications.Message, error) {
	message, err := scanCommunication(queryer.QueryRow(ctx, `
		SELECT id, thread_id, COALESCE(organization_id, ''), visibility,
		       sender_subject_id, audience, direction, subject, body,
		       revision, created_at
		FROM communication_messages
		WHERE id = $1
	`, messageID))
	if errors.Is(err, pgx.ErrNoRows) {
		return communications.Message{}, ErrNotFound
	}
	return message, err
}

type AttachCommunicationCommand struct {
	OperationID      string
	CorrelationID    string
	IdempotencyKey   string
	MessageID        string
	ObjectMetadataID string
}

func (workflow *CommunicationsWorkflow) AttachCommunication(
	ctx context.Context,
	actor identity.Principal,
	command AttachCommunicationCommand,
) (communications.Attachment, error) {
	semantic := struct {
		ObjectMetadataID string `json:"objectMetadataId"`
	}{ObjectMetadataID: command.ObjectMetadataID}
	return executeTransition(
		ctx,
		workflow.service,
		actor,
		commandEnvelope{
			OperationID:    command.OperationID,
			IdempotencyKey: command.IdempotencyKey,
			CorrelationID:  command.CorrelationID,
			Kind:           "attach_communication",
			EntityID:       command.MessageID,
			Semantic:       semantic,
		},
		func(
			ctx context.Context,
			transaction pgx.Tx,
		) (transition[communications.Attachment], error) {
			message, err := workflow.getCommunication(
				ctx,
				transaction,
				command.MessageID,
			)
			if err != nil {
				return transition[communications.Attachment]{}, err
			}
			if !communications.CanRead(actor, message) {
				return transition[communications.Attachment]{}, ErrForbidden
			}
			if actor.HasRole(identity.RoleAuditee) &&
				message.SenderSubjectID != actor.SubjectID {
				return transition[communications.Attachment]{}, ErrForbidden
			}
			var attachment communications.Attachment
			var aggregateType, aggregateID, scanStatus, objectState string
			if err := transaction.QueryRow(ctx, `
				SELECT id, COALESCE(organization_id, ''), filename,
				       COALESCE(detected_media_type, declared_media_type),
				       size_bytes, sha256, aggregate_type, aggregate_id,
				       scan_status, object_state
				FROM object_metadata
				WHERE id = $1
			`, command.ObjectMetadataID).Scan(
				&attachment.ObjectMetadataID,
				&attachment.OrganizationID,
				&attachment.FileName,
				&attachment.MediaType,
				&attachment.SizeBytes,
				&attachment.SHA256,
				&aggregateType,
				&aggregateID,
				&scanStatus,
				&objectState,
			); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return transition[communications.Attachment]{}, ErrNotFound
				}
				return transition[communications.Attachment]{}, err
			}
			if aggregateType != "COMMUNICATION" ||
				aggregateID != message.ID ||
				attachment.OrganizationID != message.OrganizationID ||
				scanStatus != "CLEAN" ||
				objectState != "CANONICAL" {
				return transition[communications.Attachment]{}, ErrForbidden
			}
			var duplicate bool
			if err := transaction.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM communication_attachments
					WHERE object_metadata_id = $1
				)
			`, command.ObjectMetadataID).Scan(&duplicate); err != nil {
				return transition[communications.Attachment]{}, err
			}
			if duplicate {
				return transition[communications.Attachment]{}, ErrConflict
			}
			attachment.ID = workflow.service.idGenerator(
				"communication-attachment",
			)
			attachment.MessageID = message.ID
			attachment.CreatedAt = workflow.service.clock().UTC()
			if _, err := transaction.Exec(ctx, `
				INSERT INTO communication_attachments (
					id, message_id, organization_id, object_metadata_id,
					file_name, media_type, size_bytes, sha256, created_at
				) VALUES (
					$1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8, $9
				)
			`, attachment.ID, attachment.MessageID,
				attachment.OrganizationID, attachment.ObjectMetadataID,
				attachment.FileName, attachment.MediaType,
				attachment.SizeBytes, attachment.SHA256,
				attachment.CreatedAt); err != nil {
				return transition[communications.Attachment]{}, err
			}
			return transition[communications.Attachment]{
				Response:       attachment,
				OrganizationID: message.OrganizationID,
				Action:         "COMMUNICATION_ATTACHMENT_ADDED",
				EntityType:     "COMMUNICATION_ATTACHMENT",
				EntityID:       attachment.ID,
				EntityVersion:  1,
				AfterStatus:    "IMMUTABLE",
				SyncKind:       "communication_attachment",
				OutboxTopic:    "communications.attachment_added",
			}, nil
		},
	)
}

func (workflow *CommunicationsWorkflow) ListCommunicationAttachments(
	ctx context.Context,
	actor identity.Principal,
	messageID string,
) ([]communications.Attachment, error) {
	if _, err := workflow.GetCommunication(ctx, actor, messageID); err != nil {
		return nil, err
	}
	rows, err := workflow.pool.Query(ctx, `
		SELECT id, message_id, COALESCE(organization_id, ''),
		       object_metadata_id, file_name, media_type, size_bytes,
		       sha256, created_at
		FROM communication_attachments
		WHERE message_id = $1
		ORDER BY id
	`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []communications.Attachment{}
	for rows.Next() {
		var item communications.Attachment
		if err := rows.Scan(
			&item.ID,
			&item.MessageID,
			&item.OrganizationID,
			&item.ObjectMetadataID,
			&item.FileName,
			&item.MediaType,
			&item.SizeBytes,
			&item.SHA256,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (workflow *CommunicationsWorkflow) ListNotifications(
	ctx context.Context,
	actor identity.Principal,
) (notifications.Page, error) {
	if actor.SubjectID == "" || !notifications.CanUse(actor) {
		return notifications.Page{}, ErrForbidden
	}
	rows, err := workflow.pool.Query(ctx, `
		SELECT record.id, record.recipient_subject_id,
		       COALESCE(record.organization_id, ''),
		       record.title, record.body,
		       COALESCE(record.related_entity_type, ''),
		       COALESCE(record.related_entity_id, ''),
		       record.deduplication_key, record.read_at,
		       CASE COALESCE(delivery.status, '')
		           WHEN 'PENDING' THEN 'PENDING'
		           WHEN 'FAILED' THEN 'RETRYING'
		           WHEN 'DELIVERED' THEN 'DELIVERED'
		           WHEN 'DEAD_LETTER' THEN 'FAILED'
		           ELSE 'NOT_CONFIGURED'
		       END,
		       COALESCE(delivery.attempt_count, 0),
		       delivery.accepted_at, delivery.next_attempt_at,
		       record.revision, record.created_at
		FROM notification_records record
		LEFT JOIN LATERAL (
		    SELECT job.status, job.attempt_count, job.accepted_at,
		           job.next_attempt_at
		    FROM notification_delivery_jobs job
		    WHERE job.notification_id = record.id
		      AND job.channel = 'EMAIL'
		    ORDER BY job.created_at DESC, job.id DESC
		    LIMIT 1
		) delivery ON true
		WHERE record.recipient_subject_id = $1
		  AND record.tombstoned_at IS NULL
		ORDER BY record.created_at DESC, record.id DESC
	`, actor.SubjectID)
	if err != nil {
		return notifications.Page{}, err
	}
	defer rows.Close()
	page := notifications.Page{Items: []notifications.Notification{}}
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return notifications.Page{}, err
		}
		if item.ReadAt == nil {
			page.UnreadCount++
		}
		page.Items = append(page.Items, item)
	}
	return page, rows.Err()
}

type notificationRow interface {
	Scan(dest ...any) error
}

func scanNotification(row notificationRow) (notifications.Notification, error) {
	var item notifications.Notification
	err := row.Scan(
		&item.ID,
		&item.RecipientSubjectID,
		&item.OrganizationID,
		&item.Title,
		&item.Body,
		&item.RelatedEntityType,
		&item.RelatedEntityID,
		&item.DeduplicationKey,
		&item.ReadAt,
		&item.EmailDeliveryStatus,
		&item.EmailDeliveryAttempts,
		&item.EmailAcceptedAt,
		&item.EmailNextAttemptAt,
		&item.Revision,
		&item.CreatedAt,
	)
	return item, err
}

type MarkNotificationReadCommand struct {
	OperationID      string
	CorrelationID    string
	IdempotencyKey   string
	NotificationID   string
	ExpectedRevision int64
}

func (workflow *CommunicationsWorkflow) MarkNotificationRead(
	ctx context.Context,
	actor identity.Principal,
	command MarkNotificationReadCommand,
) (notifications.Notification, error) {
	if actor.SubjectID == "" || !notifications.CanUse(actor) {
		return notifications.Notification{}, ErrForbidden
	}
	semantic := struct {
		ExpectedRevision int64 `json:"expectedRevision"`
	}{ExpectedRevision: command.ExpectedRevision}
	return executeTransition(
		ctx,
		workflow.service,
		actor,
		commandEnvelope{
			OperationID:    command.OperationID,
			IdempotencyKey: command.IdempotencyKey,
			CorrelationID:  command.CorrelationID,
			Kind:           "mark_notification_read",
			EntityID:       command.NotificationID,
			Semantic:       semantic,
		},
		func(
			ctx context.Context,
			transaction pgx.Tx,
		) (transition[notifications.Notification], error) {
			item, err := scanNotification(transaction.QueryRow(ctx, `
				SELECT record.id, record.recipient_subject_id,
				       COALESCE(record.organization_id, ''),
				       record.title, record.body,
				       COALESCE(record.related_entity_type, ''),
				       COALESCE(record.related_entity_id, ''),
				       record.deduplication_key, record.read_at,
				       CASE COALESCE(delivery.status, '')
				           WHEN 'PENDING' THEN 'PENDING'
				           WHEN 'FAILED' THEN 'RETRYING'
				           WHEN 'DELIVERED' THEN 'DELIVERED'
				           WHEN 'DEAD_LETTER' THEN 'FAILED'
				           ELSE 'NOT_CONFIGURED'
				       END,
				       COALESCE(delivery.attempt_count, 0),
				       delivery.accepted_at, delivery.next_attempt_at,
				       record.revision, record.created_at
				FROM notification_records record
				LEFT JOIN LATERAL (
				    SELECT job.status, job.attempt_count, job.accepted_at,
				           job.next_attempt_at
				    FROM notification_delivery_jobs job
				    WHERE job.notification_id = record.id
				      AND job.channel = 'EMAIL'
				    ORDER BY job.created_at DESC, job.id DESC
				    LIMIT 1
				) delivery ON true
				WHERE record.id = $1
				  AND record.recipient_subject_id = $2
				  AND record.tombstoned_at IS NULL
				FOR UPDATE OF record
			`, command.NotificationID, actor.SubjectID))
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[notifications.Notification]{}, ErrNotFound
			}
			if err != nil {
				return transition[notifications.Notification]{}, err
			}
			if item.ReadAt != nil ||
				command.ExpectedRevision != item.Revision {
				return transition[notifications.Notification]{}, ErrConflict
			}
			now := workflow.service.clock().UTC()
			if _, err := transaction.Exec(ctx, `
				UPDATE notification_records
				SET read_at = $3, revision = revision + 1
				WHERE id = $1
				  AND recipient_subject_id = $2
				  AND revision = $4
				  AND read_at IS NULL
				  AND tombstoned_at IS NULL
			`, item.ID, actor.SubjectID, now, item.Revision); err != nil {
				return transition[notifications.Notification]{}, err
			}
			item.ReadAt = &now
			item.Revision++
			return transition[notifications.Notification]{
				Response:       item,
				OrganizationID: actor.OrganizationID,
				Action:         "NOTIFICATION_READ",
				EntityType:     "NOTIFICATION",
				EntityID:       item.ID,
				EntityVersion:  item.Revision,
				BeforeStatus:   "UNREAD",
				AfterStatus:    "READ",
				SyncKind:       "notification",
				OutboxTopic:    "notifications.read",
			}, nil
		},
	)
}

type reminderCandidate struct {
	RuleID             string
	OffsetDays         int
	FindingID          string
	FindingReference   string
	OrganizationID     string
	RecipientSubjectID string
	DueDate            time.Time
}

const (
	reminderBatchSize = 64
	reminderMaxPages  = 16
)

func (workflow *CommunicationsWorkflow) ScheduleDueReminders(
	ctx context.Context,
) (int, error) {
	now := workflow.service.clock().UTC()
	processed := 0
	var failures []error
	lastRuleID, lastFindingID := "", ""
	for page := 0; page < reminderMaxPages; page++ {
		rows, err := workflow.pool.Query(ctx, `
			SELECT rule.id, rule.offset_days, finding.id, finding.reference,
			       finding.organization_id, finding.owner_subject_id,
			       finding.due_date
			FROM reminder_rules rule
			CROSS JOIN findings finding
			JOIN identity_references recipient
			  ON recipient.subject_id = finding.owner_subject_id
			WHERE rule.status = 'ACTIVE'
			  AND finding.due_date IS NOT NULL
			  AND finding.owner_subject_id IS NOT NULL
			  AND finding.status <> 'CLOSED'
			  AND (rule.id, finding.id) > ($1, $2)
			ORDER BY rule.id, finding.id
			LIMIT $3
		`, lastRuleID, lastFindingID, reminderBatchSize)
		if err != nil {
			return processed, joinReminderErrors(failures, err)
		}
		pageRows := 0
		for rows.Next() {
			pageRows++
			var candidate reminderCandidate
			if err := rows.Scan(
				&candidate.RuleID,
				&candidate.OffsetDays,
				&candidate.FindingID,
				&candidate.FindingReference,
				&candidate.OrganizationID,
				&candidate.RecipientSubjectID,
				&candidate.DueDate,
			); err != nil {
				rows.Close()
				return processed, joinReminderErrors(failures, err)
			}
			lastRuleID, lastFindingID = candidate.RuleID, candidate.FindingID
			if !notifications.RuleMatches(candidate.OffsetDays, candidate.DueDate, now) {
				continue
			}
			created, err := workflow.scheduleReminder(ctx, candidate, now)
			if err != nil {
				// A poison candidate must not prevent independent candidates or a
				// later bounded tick from retrying it.
				failures = append(failures, fmt.Errorf("reminder candidate failed: %w", err))
				continue
			}
			if created {
				processed++
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return processed, joinReminderErrors(failures, err)
		}
		rows.Close()
		if pageRows < reminderBatchSize {
			break
		}
	}
	return processed, errors.Join(failures...)
}

func joinReminderErrors(failures []error, current error) error {
	return errors.Join(append(failures, current)...)
}

func (workflow *CommunicationsWorkflow) scheduleReminder(
	ctx context.Context,
	candidate reminderCandidate,
	now time.Time,
) (bool, error) {
	created := false
	err := database.WithinTransaction(
		ctx,
		workflow.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			lockKey := fmt.Sprintf(
				"reminder:%s:%s:%s:%s",
				candidate.RuleID,
				candidate.FindingID,
				candidate.RecipientSubjectID,
				candidate.DueDate.UTC().Format("2006-01-02"),
			)
			if _, err := transaction.Exec(
				ctx,
				"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
				lockKey,
			); err != nil {
				return err
			}
			var exists bool
			if err := transaction.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM reminder_dispatches
					WHERE reminder_rule_id = $1
					  AND entity_type = 'FINDING'
					  AND entity_id = $2
					  AND recipient_subject_id = $3
					  AND due_date = $4
				)
			`, candidate.RuleID, candidate.FindingID,
				candidate.RecipientSubjectID,
				candidate.DueDate).Scan(&exists); err != nil {
				return err
			}
			if exists {
				return nil
			}
			dueState := notifications.DueStateFor(
				candidate.DueDate,
				now,
			)
			notification, err := workflow.createNotification(
				ctx,
				transaction,
				candidate.RecipientSubjectID,
				candidate.OrganizationID,
				"Finding due-date reminder",
				fmt.Sprintf(
					"Finding %s has a %s due-date reminder. Open the authorized record for details.",
					candidate.FindingReference,
					dueState,
				),
				"FINDING",
				candidate.FindingID,
				lockKey,
				now,
			)
			if err != nil {
				return err
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO reminder_dispatches (
					id, reminder_rule_id, entity_type, entity_id,
					recipient_subject_id, due_date, due_state,
					notification_id, dispatched_at
				) VALUES (
					$1, $2, 'FINDING', $3, $4, $5, $6, $7, $8
				)
			`, workflow.service.idGenerator("reminder-dispatch"),
				candidate.RuleID, candidate.FindingID,
				candidate.RecipientSubjectID, candidate.DueDate,
				dueState, notification.ID, now); err != nil {
				return err
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_events (
					event_id, occurred_at, actor_role, organization_id,
					action, entity_type, entity_id, entity_version,
					before_status, after_status, request_id, details
				) VALUES (
					$1, $2, 'SYSTEM', $3, 'REMINDER_DISPATCHED',
					'FINDING', $4, 1, '', $5, $1,
					jsonb_build_object(
						'reminderRuleId', $6::text,
						'recipientSubjectId', $7::text,
						'notificationId', $8::text
					)
				)
			`, workflow.service.idGenerator("audit"), now,
				candidate.OrganizationID, candidate.FindingID,
				dueState, candidate.RuleID, candidate.RecipientSubjectID,
				notification.ID); err != nil {
				return err
			}
			created = true
			return nil
		},
	)
	return created, err
}

func (workflow *CommunicationsWorkflow) ListCalendarItems(
	ctx context.Context,
	actor identity.Principal,
	organizationID string,
) ([]communications.CalendarItem, error) {
	organizationID = strings.TrimSpace(organizationID)
	if !communications.CanUseCalendar(actor) {
		return nil, ErrForbidden
	}
	if actor.HasRole(identity.RoleAuditee) {
		if organizationID != "" && organizationID != actor.OrganizationID {
			return nil, ErrForbidden
		}
		organizationID = actor.OrganizationID
	}
	isInspector := actor.HasRole(identity.RoleInspector) &&
		!actor.HasRole(identity.RoleLeadInspector)
	isLead := actor.HasRole(identity.RoleLeadInspector)
	isAuditee := actor.HasRole(identity.RoleAuditee)
	today := workflow.service.clock().UTC().Format("2006-01-02")
	rows, err := workflow.pool.Query(ctx, `
		SELECT inspection.id, inspection.organization_id,
		       organization.legal_name, inspection.title, inspection.status,
		       COALESCE(
		           assignment.scheduled_end_date,
		           inspection.due_date,
		           $8::date
		       ) AS scheduled_date,
		       COALESCE(
		           assignment.scheduled_end_date,
		           inspection.due_date,
		           $8::date
		       ) AS due_date
		FROM inspections inspection
		JOIN organizations organization
		  ON organization.id = inspection.organization_id
		LEFT JOIN audit_assignments assignment
		  ON assignment.inspection_id = inspection.id
		 AND assignment.tombstoned_at IS NULL
		WHERE ($1 = false OR EXISTS (
		          SELECT 1
		          FROM inspection_question_assignments question_assignment
		          WHERE question_assignment.inspection_id = inspection.id
		            AND question_assignment.subject_id = $2
		      ))
		  AND ($3 = false OR (
		          assignment.lead_subject_id = $2
		          OR EXISTS (
		              SELECT 1
		              FROM audit_team_members member
		              WHERE member.assignment_id = assignment.id
		                AND member.subject_id = $2
		                AND member.removed_at IS NULL
		          )
		      ))
		  AND ($4 = false OR (
		          inspection.organization_id = $5
		          AND assignment.status IN (
		              'AWAITING_AUDITEE_CONFIRMATION',
		              'CONFIRMED',
		              'ALTERNATIVE_PROPOSED'
		          )
		          AND EXISTS (
		              SELECT 1
		              FROM planning_intake_drafts draft
		              WHERE draft.values->>'preparedAuditId' = inspection.id
		                AND draft.values->>'noticePolicy' = 'ADVANCE'
		                AND draft.tombstoned_at IS NULL
		          )
		      ))
		  AND ($6 = false OR inspection.organization_id = $7)
		ORDER BY scheduled_date, inspection.id
	`, isInspector, actor.SubjectID, isLead, isAuditee,
		actor.OrganizationID, organizationID != "", organizationID, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []communications.CalendarItem{}
	for rows.Next() {
		var item communications.CalendarItem
		var status string
		var scheduledDate time.Time
		var dueDate time.Time
		if err := rows.Scan(
			&item.AuditID,
			&item.OrganizationID,
			&item.OrganizationName,
			&item.Title,
			&status,
			&scheduledDate,
			&dueDate,
		); err != nil {
			return nil, err
		}
		item.ID = "CAL-" + item.AuditID
		item.ScheduledDate = scheduledDate.UTC().Format("2006-01-02")
		item.DueState = notifications.DueStateFor(
			dueDate,
			workflow.service.clock(),
		)
		item.NextAction = "Continue Cabin Inspection checklist"
		if status == "COMPLETED" || status == "CLOSED" {
			item.NextAction = "Review completed Audit"
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (workflow *CommunicationsWorkflow) GetCalendarItem(
	ctx context.Context,
	actor identity.Principal,
	calendarItemID string,
) (communications.CalendarItem, error) {
	items, err := workflow.ListCalendarItems(ctx, actor, "")
	if err != nil {
		return communications.CalendarItem{}, err
	}
	for _, item := range items {
		if item.ID == calendarItemID {
			return item, nil
		}
	}
	return communications.CalendarItem{}, ErrNotFound
}

func mapCommunicationError(err error) error {
	switch {
	case errors.Is(err, communications.ErrForbidden):
		return fmt.Errorf("%w: %v", ErrForbidden, err)
	case errors.Is(err, communications.ErrInvalid):
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	default:
		return err
	}
}
