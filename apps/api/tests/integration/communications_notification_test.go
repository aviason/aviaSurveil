//go:build canonicaltest

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/communications"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/notifications"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

func TestCommunicationVisibilityAttachmentsNotificationsAndReadAuthority(t *testing.T) {
	pool := canonicalDatabase(t, "communications_notifications")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (
			subject_id, issuer, display_name
		) VALUES (
			'auditee-expired-session', 'test', 'Expired-session Auditee'
		)
	`); err != nil {
		t.Fatalf("seed offline Auditee identity: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO session_references (
			id, subject_id, organization_id, expires_at, last_seen_at,
			absolute_expires_at, roles
		) VALUES (
			'session-auditee-expired', 'auditee-expired-session',
			'airline-xyz', $1, $1, $1, ARRAY['auditee']
		)
	`, canonicalNow.Add(-time.Hour)); err != nil {
		t.Fatalf("seed offline Auditee session: %v", err)
	}
	workflow := application.NewCommunicationsWorkflow(
		pool,
		application.CommunicationsWorkflowDependencies{
			Clock:       func() time.Time { return canonicalNow },
			IDGenerator: scenarioIDGenerator(),
		},
	)
	inspector := principal(
		"inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector,
	)
	auditee := principal(
		"auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee,
	)
	otherAuditee := principal(
		"auditee-other", "airline-other", "session-other", identity.RoleAuditee,
	)
	if messages, err := workflow.ListCommunications(
		context.Background(),
		principal("gm-001", "caa", "session-gm", identity.RoleGeneralManager),
		"",
	); !errors.Is(err, application.ErrForbidden) || messages != nil {
		t.Fatalf("unsupported GM Communication capability = %+v, err = %v", messages, err)
	}

	external, err := workflow.SendCommunication(
		context.Background(),
		inspector,
		application.SendCommunicationCommand{
			OperationID: "op-message-auditee", CorrelationID: "corr-message-auditee",
			IdempotencyKey: "idem-message-auditee", OrganizationID: "airline-xyz",
			Subject:  "Cabin Inspection follow-up",
			Body:     "Please provide the requested public training record.",
			Audience: communications.AudienceAuditee,
		},
	)
	if err != nil {
		t.Fatalf("send Auditee-visible communication: %v", err)
	}
	if external.Visibility != communications.VisibilityAuditeeVisible ||
		external.Direction != communications.DirectionCAAToAuditee ||
		external.OrganizationID != "airline-xyz" {
		t.Fatalf("Auditee-visible communication = %+v", external)
	}
	replayed, err := workflow.SendCommunication(
		context.Background(),
		inspector,
		application.SendCommunicationCommand{
			OperationID: "op-message-auditee", CorrelationID: "corr-message-auditee",
			IdempotencyKey: "idem-message-auditee", OrganizationID: "airline-xyz",
			Subject:  "Cabin Inspection follow-up",
			Body:     "Please provide the requested public training record.",
			Audience: communications.AudienceAuditee,
		},
	)
	if err != nil || replayed.ID != external.ID || replayed.CreatedAt != external.CreatedAt {
		t.Fatalf("replay Auditee-visible communication = %+v, err = %v", replayed, err)
	}

	internal, err := workflow.SendCommunication(
		context.Background(),
		inspector,
		application.SendCommunicationCommand{
			OperationID: "op-message-internal", CorrelationID: "corr-message-internal",
			IdempotencyKey: "idem-message-internal", OrganizationID: "airline-xyz",
			Subject:  "Internal CAA Note",
			Body:     "Private enforcement deliberation must never reach the Auditee.",
			Audience: communications.AudienceCAA,
		},
	)
	if err != nil {
		t.Fatalf("send Internal CAA communication: %v", err)
	}
	if internal.Visibility != communications.VisibilityInternalCAA ||
		internal.Direction != communications.DirectionCAAInternal {
		t.Fatalf("Internal CAA communication = %+v", internal)
	}

	auditeeMessages, err := workflow.ListCommunications(
		context.Background(), auditee, "airline-xyz",
	)
	if err != nil || len(auditeeMessages) != 1 || auditeeMessages[0].ID != external.ID {
		t.Fatalf("Auditee communication list = %+v, err = %v", auditeeMessages, err)
	}
	if strings.Contains(
		auditeeMessages[0].Body,
		"enforcement",
	) {
		t.Fatalf("Auditee communication leaked Internal CAA Note: %+v", auditeeMessages[0])
	}
	if messages, err := workflow.ListCommunications(
		context.Background(), otherAuditee, "airline-xyz",
	); !errors.Is(err, application.ErrForbidden) || messages != nil {
		t.Fatalf("cross-organization communication list = %+v, err = %v", messages, err)
	}
	if _, err := workflow.GetCommunication(
		context.Background(), auditee, internal.ID,
	); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("Auditee direct Internal CAA communication error = %v", err)
	}

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO object_metadata (
			id, aggregate_type, aggregate_id, object_key, filename,
			declared_media_type, detected_media_type, sha256, size_bytes,
			scan_status, organization_id, bucket_name, object_state, created_at
		) VALUES (
			'object-message-public', 'COMMUNICATION', $1,
			'organizations/airline-xyz/communications/public-record.pdf',
			'public-record.pdf', 'application/pdf', 'application/pdf',
			'sha256:message-public', 128, 'CLEAN', 'airline-xyz',
			'avia-communications', 'CANONICAL', $2
		)
	`, external.ID, canonicalNow); err != nil {
		t.Fatalf("seed Communication attachment object metadata: %v", err)
	}
	if _, err := workflow.AttachCommunication(
		context.Background(),
		auditee,
		application.AttachCommunicationCommand{
			OperationID:    "op-message-incoming-auditee-attachment",
			CorrelationID:  "corr-message-incoming-auditee-attachment",
			IdempotencyKey: "idem-message-incoming-auditee-attachment",
			MessageID:      external.ID, ObjectMetadataID: "object-message-public",
		},
	); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("Auditee attachment to incoming CAA message error = %v", err)
	}
	attachment, err := workflow.AttachCommunication(
		context.Background(),
		inspector,
		application.AttachCommunicationCommand{
			OperationID: "op-message-attachment", CorrelationID: "corr-message-attachment",
			IdempotencyKey: "idem-message-attachment", MessageID: external.ID,
			ObjectMetadataID: "object-message-public",
		},
	)
	if err != nil {
		t.Fatalf("attach immutable Communication metadata: %v", err)
	}
	if attachment.FileName != "public-record.pdf" ||
		attachment.MediaType != "application/pdf" ||
		attachment.SizeBytes != 128 ||
		attachment.SHA256 != "sha256:message-public" {
		t.Fatalf("Communication attachment metadata = %+v", attachment)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO communication_attachments (
			id, message_id, organization_id, object_metadata_id,
			file_name, media_type, size_bytes, sha256, created_at
		)
		SELECT
			'communication-attachment-storage-duplicate',
			message_id, organization_id, object_metadata_id,
			file_name, media_type, size_bytes, sha256, created_at
		FROM communication_attachments
		WHERE id = $1
	`, attachment.ID); err == nil {
		t.Fatal("Communication attachment storage accepted duplicate object metadata")
	}
	if _, err := workflow.AttachCommunication(
		context.Background(),
		inspector,
		application.AttachCommunicationCommand{
			OperationID:    "op-message-attachment-duplicate",
			CorrelationID:  "corr-message-attachment-duplicate",
			IdempotencyKey: "idem-message-attachment-duplicate",
			MessageID:      external.ID, ObjectMetadataID: "object-message-public",
		},
	); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("duplicate immutable Communication attachment error = %v", err)
	}
	auditeeAttachments, err := workflow.ListCommunicationAttachments(
		context.Background(), auditee, external.ID,
	)
	if err != nil || len(auditeeAttachments) != 1 ||
		auditeeAttachments[0].ID != attachment.ID ||
		auditeeAttachments[0].MessageID != attachment.MessageID ||
		auditeeAttachments[0].OrganizationID != attachment.OrganizationID ||
		auditeeAttachments[0].ObjectMetadataID != attachment.ObjectMetadataID ||
		auditeeAttachments[0].FileName != attachment.FileName ||
		auditeeAttachments[0].MediaType != attachment.MediaType ||
		auditeeAttachments[0].SizeBytes != attachment.SizeBytes ||
		auditeeAttachments[0].SHA256 != attachment.SHA256 ||
		!auditeeAttachments[0].CreatedAt.Equal(attachment.CreatedAt) {
		t.Fatalf("Auditee Communication attachments = %+v, err = %v", auditeeAttachments, err)
	}
	if _, err := workflow.AttachCommunication(
		context.Background(),
		inspector,
		application.AttachCommunicationCommand{
			OperationID:    "op-message-internal-unsafe-attachment",
			CorrelationID:  "corr-message-internal-unsafe-attachment",
			IdempotencyKey: "idem-message-internal-unsafe-attachment",
			MessageID:      internal.ID, ObjectMetadataID: "object-message-public",
		},
	); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("cross-visibility attachment error = %v", err)
	}

	notificationsForAuditee, err := workflow.ListNotifications(
		context.Background(), auditee,
	)
	if err != nil || notificationsForAuditee.UnreadCount != 1 ||
		len(notificationsForAuditee.Items) != 1 {
		t.Fatalf("Auditee notifications = %+v, err = %v", notificationsForAuditee, err)
	}
	if page, err := workflow.ListNotifications(
		context.Background(),
		principal("auditee-xyz", "airline-xyz", "session-auditee"),
	); !errors.Is(err, application.ErrForbidden) || page.Items != nil {
		t.Fatalf("roleless Notification capability = %+v, err = %v", page, err)
	}
	notification := notificationsForAuditee.Items[0]
	if notification.RelatedEntityID != external.ID ||
		strings.Contains(notification.Body, internal.Body) {
		t.Fatalf("Auditee notification content = %+v", notification)
	}
	if _, err := workflow.MarkNotificationRead(
		context.Background(),
		otherAuditee,
		application.MarkNotificationReadCommand{
			OperationID:    "op-notification-read-other",
			CorrelationID:  "corr-notification-read-other",
			IdempotencyKey: "idem-notification-read-other",
			NotificationID: notification.ID, ExpectedRevision: notification.Revision,
		},
	); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("cross-recipient notification read error = %v", err)
	}
	read, err := workflow.MarkNotificationRead(
		context.Background(),
		auditee,
		application.MarkNotificationReadCommand{
			OperationID:    "op-notification-read",
			CorrelationID:  "corr-notification-read",
			IdempotencyKey: "idem-notification-read",
			NotificationID: notification.ID, ExpectedRevision: notification.Revision,
		},
	)
	if err != nil || read.ReadAt == nil || read.Revision != notification.Revision+1 {
		t.Fatalf("mark notification read = %+v, err = %v", read, err)
	}
	replayedRead, err := workflow.MarkNotificationRead(
		context.Background(),
		auditee,
		application.MarkNotificationReadCommand{
			OperationID:    "op-notification-read",
			CorrelationID:  "corr-notification-read",
			IdempotencyKey: "idem-notification-read",
			NotificationID: notification.ID, ExpectedRevision: notification.Revision,
		},
	)
	if err != nil || !reflect.DeepEqual(replayedRead, read) {
		t.Fatalf("replay notification read = %+v, err = %v", replayedRead, err)
	}

	var notificationCount, expiredRecipientCount, emailJobs, emailOutbox, commandLinks int
	if err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM notification_records
			 WHERE related_entity_id = $1),
			(SELECT count(*) FROM notification_records
			 WHERE recipient_subject_id = 'auditee-expired-session'
			   AND related_entity_id = $1),
			(SELECT count(*) FROM notification_delivery_jobs job
			 JOIN notification_records record ON record.id = job.notification_id
			 WHERE record.related_entity_id = $1 AND job.channel = 'EMAIL'),
			(SELECT count(*) FROM outbox_messages
			 WHERE topic = 'notification.email_requested'
			   AND aggregate_id = (
			       SELECT id FROM notification_records
			       WHERE related_entity_id = $1
			         AND recipient_subject_id = 'auditee-xyz'
			   )),
			(SELECT count(*) FROM command_transaction_links
			 WHERE operation_id IN ('op-message-auditee', 'op-message-attachment',
			                        'op-notification-read'))
	`, external.ID).Scan(
		&notificationCount, &expiredRecipientCount, &emailJobs, &emailOutbox, &commandLinks,
	); err != nil {
		t.Fatalf("read Communication/Notification transaction effects: %v", err)
	}
	if notificationCount != 2 || expiredRecipientCount != 1 ||
		emailJobs != 2 || emailOutbox != 1 ||
		commandLinks != 3 {
		t.Fatalf(
			"Communication/Notification effects = notifications %d expired recipients %d email jobs %d email outbox %d links %d",
			notificationCount, expiredRecipientCount, emailJobs, emailOutbox, commandLinks,
		)
	}
}

func TestReminderSchedulerCoversConfiguredStagesAndSuppressesDuplicates(t *testing.T) {
	pool := canonicalDatabase(t, "reminder_scheduler")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO reminder_rules (
			id, label, offset_days, channel, status, revision, created_at, updated_at
		) VALUES
			('REM-30', '30 days before Due Date', 30, 'IN_APP', 'ACTIVE', 1, $1, $1),
			('REM-15', '15 days before Due Date', 15, 'IN_APP', 'ACTIVE', 1, $1, $1),
			('REM-7', '7 days before Due Date', 7, 'IN_APP', 'ACTIVE', 1, $1, $1),
			('REM-DUE', 'On the Due Date', 0, 'IN_APP', 'ACTIVE', 1, $1, $1),
			('REM-OVERDUE', 'After the Due Date', -1, 'IN_APP', 'ACTIVE', 1, $1, $1)
	`, canonicalNow); err != nil {
		t.Fatalf("seed reminder rules: %v", err)
	}
	for index, offset := range []int{30, 15, 7, 0, -3} {
		due := canonicalNow.AddDate(0, 0, offset).Format("2006-01-02")
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO findings (
				id, reference, inspection_id, organization_id, severity, status,
				owner_subject_id, next_action, due_date, revision
			) VALUES (
				$1, $2, 'audit-cabin-001', 'airline-xyz', 'LEVEL_2_MAJOR',
				'WAITING_FOR_CAP', 'auditee-xyz', 'Submit CAP', $3, 1
			)
		`, "finding-reminder-"+due, "REMINDER-"+string(rune('A'+index)), due); err != nil {
			t.Fatalf("seed reminder Finding %s: %v", due, err)
		}
	}
	workflow := application.NewCommunicationsWorkflow(
		pool,
		application.CommunicationsWorkflowDependencies{
			Clock:       func() time.Time { return canonicalNow },
			IDGenerator: scenarioIDGenerator(),
		},
	)
	processed, err := workflow.ScheduleDueReminders(context.Background())
	if err != nil || processed != 5 {
		t.Fatalf("schedule configured reminder stages = %d, err = %v", processed, err)
	}
	replayed, err := workflow.ScheduleDueReminders(context.Background())
	if err != nil || replayed != 0 {
		t.Fatalf("repeat reminder schedule = %d, err = %v", replayed, err)
	}

	rows, err := pool.Query(context.Background(), `
		SELECT reminder_rule_id, due_state
		FROM reminder_dispatches
		ORDER BY reminder_rule_id
	`)
	if err != nil {
		t.Fatalf("list reminder dispatches: %v", err)
	}
	defer rows.Close()
	gotStages := []string{}
	for rows.Next() {
		var ruleID string
		var dueState notifications.DueState
		if err := rows.Scan(&ruleID, &dueState); err != nil {
			t.Fatalf("scan reminder dispatch: %v", err)
		}
		gotStages = append(gotStages, ruleID+"="+string(dueState))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate reminder dispatches: %v", err)
	}
	wantStages := []string{
		"REM-15=NOT_DUE",
		"REM-30=NOT_DUE",
		"REM-7=DUE_SOON",
		"REM-DUE=DUE_TODAY",
		"REM-OVERDUE=OVERDUE",
	}
	if !reflect.DeepEqual(gotStages, wantStages) {
		t.Fatalf("reminder stages = %v, want %v", gotStages, wantStages)
	}
	var notificationsCount, emailJobs, emailOutbox, reminderAudits, closed int
	if err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM notification_records
			 WHERE related_entity_type = 'FINDING'),
			(SELECT count(*) FROM notification_delivery_jobs
			 WHERE channel = 'EMAIL'),
			(SELECT count(*) FROM outbox_messages
			 WHERE topic = 'notification.email_requested'),
			(SELECT count(*) FROM audit_events
			 WHERE action = 'REMINDER_DISPATCHED'),
			(SELECT count(*) FROM findings
			 WHERE id LIKE 'finding-reminder-%' AND status = 'CLOSED')
	`).Scan(
		&notificationsCount, &emailJobs, &emailOutbox, &reminderAudits, &closed,
	); err != nil {
		t.Fatalf("read reminder side effects: %v", err)
	}
	if notificationsCount != 5 || emailJobs != 5 || emailOutbox != 5 ||
		reminderAudits != 5 || closed != 0 {
		t.Fatalf(
			"reminder effects = notifications %d email jobs %d email outbox %d audits %d closed %d",
			notificationsCount, emailJobs, emailOutbox, reminderAudits, closed,
		)
	}
}

func TestNotificationEmailDeliveryFailureIsRetriedAndAudited(t *testing.T) {
	pool := canonicalDatabase(t, "notification_delivery_retry")
	ids := scenarioIDGenerator()
	workflow := application.NewCommunicationsWorkflow(
		pool,
		application.CommunicationsWorkflowDependencies{
			Clock:       func() time.Time { return canonicalNow },
			IDGenerator: ids,
		},
	)
	message, err := workflow.SendCommunication(
		context.Background(),
		principal(
			"inspector-cabin-001", "caa", "session-inspector",
			identity.RoleInspector,
		),
		application.SendCommunicationCommand{
			OperationID:    "op-delivery-retry-message",
			CorrelationID:  "corr-delivery-retry-message",
			IdempotencyKey: "idem-delivery-retry-message",
			OrganizationID: "airline-xyz",
			Subject:        "Delivery retry test",
			Body:           "Open the authorized message record.",
			Audience:       communications.AudienceAuditee,
		},
	)
	if err != nil {
		t.Fatalf("seed Notification email job: %v", err)
	}
	providerFailure := errors.New("deterministic provider unavailable")
	failing := notifications.NewDeliveryService(
		pool,
		notifications.DeliveryDependencies{
			Clock:       func() time.Time { return canonicalNow },
			IDGenerator: ids,
			Adapter: notificationDeliveryAdapterFunc(
				func(_ context.Context, delivery notifications.EmailDelivery) error {
					if delivery.RelatedEntityID != message.ID ||
						delivery.RecipientSubjectID != "auditee-xyz" {
						t.Fatalf("failed delivery envelope = %+v", delivery)
					}
					lockContext, cancel := context.WithTimeout(
						context.Background(),
						250*time.Millisecond,
					)
					defer cancel()
					var jobStatus string
					if err := pool.QueryRow(lockContext, `
						SELECT status
						FROM notification_delivery_jobs
						WHERE id = $1
						FOR UPDATE
					`, delivery.JobID).Scan(&jobStatus); err != nil {
						return fmt.Errorf(
							"delivery adapter was called while the job row remained locked: %w",
							err,
						)
					}
					return providerFailure
				},
			),
		},
	)
	processed, err := failing.ProcessNext(context.Background())
	if !processed || !errors.Is(err, providerFailure) {
		t.Fatalf("failed Notification delivery = processed %t err %v", processed, err)
	}
	var status, lastError string
	var attempts int
	var deliveredAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT job.status, job.attempt_count, COALESCE(job.last_error, ''),
		       outbox.delivered_at
		FROM notification_delivery_jobs job
		JOIN outbox_messages outbox ON outbox.id = job.outbox_message_id
		WHERE job.channel = 'EMAIL'
	`).Scan(&status, &attempts, &lastError, &deliveredAt); err != nil {
		t.Fatalf("read failed Notification delivery: %v", err)
	}
	if status != "FAILED" || attempts != 1 ||
		lastError != "SMTP_DELIVERY_FAILED" ||
		deliveredAt != nil {
		t.Fatalf(
			"failed Notification delivery state = %s attempts %d error %q deliveredAt %v",
			status, attempts, lastError, deliveredAt,
		)
	}

	successful := notifications.NewDeliveryService(
		pool,
		notifications.DeliveryDependencies{
			Clock:       func() time.Time { return canonicalNow.Add(time.Minute) },
			IDGenerator: ids,
			Adapter: notificationDeliveryAdapterFunc(
				func(_ context.Context, delivery notifications.EmailDelivery) error {
					if strings.Contains(delivery.Body, "enforcement") {
						t.Fatalf("Notification delivery leaked internal content: %+v", delivery)
					}
					return nil
				},
			),
		},
	)
	processed, err = successful.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("retry Notification delivery = processed %t err %v", processed, err)
	}
	processed, err = successful.ProcessNext(context.Background())
	if err != nil || processed {
		t.Fatalf("repeat delivered Notification job = processed %t err %v", processed, err)
	}
	var auditCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT job.status, job.attempt_count, COALESCE(job.last_error, ''),
		       outbox.delivered_at,
		       (
		           SELECT count(*)
		           FROM audit_events
		           WHERE entity_type = 'NOTIFICATION_DELIVERY'
		             AND action IN (
		                 'NOTIFICATION_EMAIL_DELIVERY_FAILED',
		                 'NOTIFICATION_EMAIL_DELIVERED'
		             )
		       )
		FROM notification_delivery_jobs job
		JOIN outbox_messages outbox ON outbox.id = job.outbox_message_id
		WHERE job.channel = 'EMAIL'
	`).Scan(&status, &attempts, &lastError, &deliveredAt, &auditCount); err != nil {
		t.Fatalf("read delivered Notification retry: %v", err)
	}
	if status != "DELIVERED" || attempts != 2 || lastError != "" ||
		deliveredAt == nil || !deliveredAt.Equal(canonicalNow.Add(time.Minute)) ||
		auditCount != 2 {
		t.Fatalf(
			"delivered Notification retry state = %s attempts %d error %q deliveredAt %v audits %d",
			status, attempts, lastError, deliveredAt, auditCount,
		)
	}
}

func TestCalendarIsProjectionOfAuthorizedAuditWork(t *testing.T) {
	pool := canonicalDatabase(t, "authorized_calendar")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO planning_intake_drafts (
			id, organization_id, values, revision, created_by_subject_id
		) VALUES (
			'draft-cabin-advance', 'airline-xyz',
			'{"preparedAuditId":"audit-cabin-001","noticePolicy":"ADVANCE"}',
			1, 'manager-001'
		);
		UPDATE audit_assignments
		SET status = 'AWAITING_AUDITEE_CONFIRMATION',
		    scheduled_start_date = '2026-07-20',
		    scheduled_end_date = '2026-08-01'
		WHERE id = 'assignment-cabin-001';
		INSERT INTO inspections (
			id, organization_id, assigned_inspector_subject_id, title,
			inspection_type, status, due_date, revision
		) VALUES (
			'audit-preparation-001', 'airline-xyz', 'inspector-other',
			'Unreleased Preparation Inspection', 'CABIN', 'PREPARATION',
			'2026-08-10', 1
		);
		INSERT INTO planning_intake_drafts (
			id, organization_id, values, revision, created_by_subject_id
		) VALUES (
			'draft-preparation-advance', 'airline-xyz',
			'{"preparedAuditId":"audit-preparation-001","noticePolicy":"ADVANCE"}',
			1, 'manager-001'
		);
		INSERT INTO inspections (
			id, organization_id, assigned_inspector_subject_id, title,
			inspection_type, status, due_date, revision
		) VALUES (
			'audit-other-001', 'airline-other', 'inspector-other',
			'Other Airline Cargo Inspection', 'CARGO', 'IN_PROGRESS',
			'2026-07-22', 1
		);
		INSERT INTO inspection_question_assignments (
			inspection_id, question_id, subject_id, assignment_revision
		) VALUES ('audit-other-001', 'q-cargo-001', 'inspector-other', 1);
		INSERT INTO planning_intake_drafts (
			id, organization_id, values, revision, created_by_subject_id
		) VALUES (
			'draft-other-withheld', 'airline-other',
			'{"preparedAuditId":"audit-other-001","noticePolicy":"WITHHELD"}',
			1, 'manager-001'
		);
		INSERT INTO audit_assignments (
			id, inspection_id, organization_id, lead_subject_id, status,
			scheduled_start_date, scheduled_end_date, revision
		) VALUES (
			'assignment-other-001', 'audit-other-001', 'airline-other',
			'lead-001', 'AWAITING_AUDITEE_CONFIRMATION',
			'2026-07-21', '2026-07-22', 1
		)
	`); err != nil {
		t.Fatalf("seed released, preparation, cross-scope, and withheld calendar work: %v", err)
	}
	workflow := application.NewCommunicationsWorkflow(
		pool,
		application.CommunicationsWorkflowDependencies{
			Clock:       func() time.Time { return canonicalNow },
			IDGenerator: scenarioIDGenerator(),
		},
	)
	inspector := principal(
		"inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector,
	)
	inspectorItems, err := workflow.ListCalendarItems(
		context.Background(), inspector, "",
	)
	if err != nil || len(inspectorItems) != 1 ||
		inspectorItems[0].AuditID != "audit-cabin-001" {
		t.Fatalf("Inspector calendar = %+v, err = %v", inspectorItems, err)
	}
	if _, err := workflow.GetCalendarItem(
		context.Background(), inspector, "CAL-audit-other-001",
	); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("Inspector cross-assignment calendar detail error = %v", err)
	}
	auditeeItems, err := workflow.ListCalendarItems(
		context.Background(),
		principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee),
		"",
	)
	if err != nil || len(auditeeItems) != 1 ||
		auditeeItems[0].OrganizationID != "airline-xyz" {
		t.Fatalf("Auditee calendar = %+v, err = %v", auditeeItems, err)
	}
	otherAuditeeItems, err := workflow.ListCalendarItems(
		context.Background(),
		principal("auditee-other", "airline-other", "session-other", identity.RoleAuditee),
		"",
	)
	if err != nil || len(otherAuditeeItems) != 0 {
		t.Fatalf("withheld Auditee calendar = %+v, err = %v", otherAuditeeItems, err)
	}

	managerItems, err := workflow.ListCalendarItems(
		context.Background(),
		principal("manager-001", "caa", "session-manager", identity.RoleDepartmentManager),
		"",
	)
	if err != nil || len(managerItems) != 3 {
		t.Fatalf("Manager Calendar capability = %+v, err = %v", managerItems, err)
	}
}

func TestCommunicationsNotificationCalendarExactHTTP(t *testing.T) {
	pool := createTestDatabase(t, "communications_http")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
		t.Fatalf("reset canonical profile: %v", err)
	}
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Application: testService(pool), Clock: func() time.Time { return canonicalNow },
	})
	handler := httpapi.NewCanonicalTestBoundary("task-8-token").Protect(api.Handler())
	request := func(method, path, body, subjectID string) *httptest.ResponseRecorder {
		httpRequest := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		httpRequest.Header.Set(httpapi.CanonicalTestTokenHeader, "task-8-token")
		httpRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, subjectID)
		if body != "" {
			httpRequest.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httpRequest)
		return response
	}
	mutation := func(
		method, path, body, subjectID, idempotencyKey string,
		expectedRevision *int64,
	) *httptest.ResponseRecorder {
		httpRequest := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		httpRequest.Header.Set(httpapi.CanonicalTestTokenHeader, "task-8-token")
		httpRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, subjectID)
		httpRequest.Header.Set("Content-Type", "application/json")
		httpRequest.Header.Set("Idempotency-Key", idempotencyKey)
		if expectedRevision != nil {
			httpRequest.Header.Set("If-Match", fmt.Sprintf(`"rev-%d"`, *expectedRevision))
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httpRequest)
		return response
	}

	sendBody := `{
		"operationId":"OP-TASK8-MESSAGE",
		"expectedRevision":null,
		"idempotencyKey":"IDEM-TASK8-MESSAGE",
		"organizationId":"ORG-FLY-NAMIBIA",
		"subject":"Cabin Inspection follow-up",
		"body":"Please provide the requested public training record.",
		"audience":"AUDITEE"
	}`
	missingHeaders := request(
		http.MethodPost, "/v1/communications", sendBody,
		testprofile.CanonicalInspectorSubjectID,
	)
	if missingHeaders.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing Communication command headers status=%d body=%s",
			missingHeaders.Code, missingHeaders.Body.String())
	}
	sent := mutation(
		http.MethodPost, "/v1/communications", sendBody,
		testprofile.CanonicalInspectorSubjectID,
		"IDEM-TASK8-MESSAGE", nil,
	)
	if sent.Code != http.StatusCreated ||
		!strings.Contains(sent.Body.String(), `"direction":"CAA_TO_AUDITEE"`) {
		t.Fatalf("POST Communication status=%d body=%s", sent.Code, sent.Body.String())
	}
	internal := mutation(http.MethodPost, "/v1/communications", `{
		"operationId":"OP-TASK8-INTERNAL",
		"expectedRevision":null,
		"idempotencyKey":"IDEM-TASK8-INTERNAL",
		"organizationId":"ORG-FLY-NAMIBIA",
		"subject":"Internal CAA Note",
		"body":"Private enforcement deliberation.",
		"audience":"CAA"
	}`, testprofile.CanonicalInspectorSubjectID, "IDEM-TASK8-INTERNAL", nil)
	if internal.Code != http.StatusCreated ||
		!strings.Contains(internal.Body.String(), `"direction":"CAA_INTERNAL"`) {
		t.Fatalf("POST Internal CAA Communication status=%d body=%s",
			internal.Code, internal.Body.String())
	}
	auditeeMessages := request(
		http.MethodGet,
		"/v1/communications?organizationId=ORG-FLY-NAMIBIA",
		"",
		"USR-AUDITEE-FLY",
	)
	if auditeeMessages.Code != http.StatusOK ||
		!strings.Contains(auditeeMessages.Body.String(), `"direction":"CAA_TO_AUDITEE"`) ||
		strings.Contains(auditeeMessages.Body.String(), "enforcement") ||
		strings.Contains(auditeeMessages.Body.String(), "Internal CAA Note") {
		t.Fatalf("GET Auditee Communications status=%d body=%s",
			auditeeMessages.Code, auditeeMessages.Body.String())
	}

	var notificationID string
	var notificationRevision int64
	if err := pool.QueryRow(context.Background(), `
		SELECT id, revision
		FROM notification_records
		WHERE recipient_subject_id = 'USR-AUDITEE-FLY'
		  AND related_entity_type = 'COMMUNICATION'
	`).Scan(&notificationID, &notificationRevision); err != nil {
		t.Fatalf("read HTTP notification: %v", err)
	}
	notificationsResponse := request(
		http.MethodGet, "/v1/notifications", "", "USR-AUDITEE-FLY",
	)
	if notificationsResponse.Code != http.StatusOK ||
		!strings.Contains(notificationsResponse.Body.String(), `"id":"`+notificationID+`"`) ||
		strings.Contains(notificationsResponse.Body.String(), "enforcement") {
		t.Fatalf("GET Notifications status=%d body=%s",
			notificationsResponse.Code, notificationsResponse.Body.String())
	}
	revisionOne := int64(1)
	drift := mutation(
		http.MethodPost,
		"/v1/notifications/WRONG-NOTIFICATION/read",
		`{
			"operationId":"OP-TASK8-READ-DRIFT",
			"expectedRevision":1,
			"idempotencyKey":"IDEM-TASK8-READ-DRIFT",
			"notificationId":"`+notificationID+`"
		}`,
		"USR-AUDITEE-FLY",
		"IDEM-TASK8-READ-DRIFT",
		&revisionOne,
	)
	if drift.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Notification path/body drift status=%d body=%s",
			drift.Code, drift.Body.String())
	}
	read := mutation(
		http.MethodPost,
		"/v1/notifications/"+notificationID+"/read",
		`{
			"operationId":"OP-TASK8-READ",
			"expectedRevision":`+fmt.Sprint(notificationRevision)+`,
			"idempotencyKey":"IDEM-TASK8-READ",
			"notificationId":"`+notificationID+`"
		}`,
		"USR-AUDITEE-FLY",
		"IDEM-TASK8-READ",
		&notificationRevision,
	)
	if read.Code != http.StatusOK ||
		!strings.Contains(read.Body.String(), `"revision":2`) ||
		!strings.Contains(read.Body.String(), `"readAt":"`) {
		t.Fatalf("POST Notification read status=%d body=%s",
			read.Code, read.Body.String())
	}

	inspectorCalendar := request(
		http.MethodGet, "/v1/calendar-items", "",
		testprofile.CanonicalInspectorSubjectID,
	)
	if inspectorCalendar.Code != http.StatusOK ||
		!strings.Contains(inspectorCalendar.Body.String(), `"auditId":"AUD-2026-001"`) ||
		strings.Contains(inspectorCalendar.Body.String(), `"auditId":"AUD-2026-099"`) {
		t.Fatalf("GET Inspector Calendar status=%d body=%s",
			inspectorCalendar.Code, inspectorCalendar.Body.String())
	}
	calendarDetail := request(
		http.MethodGet, "/v1/calendar-items/CAL-AUD-2026-001", "",
		testprofile.CanonicalInspectorSubjectID,
	)
	if calendarDetail.Code != http.StatusOK ||
		!strings.Contains(calendarDetail.Body.String(), `"id":"CAL-AUD-2026-001"`) {
		t.Fatalf("GET Calendar detail status=%d body=%s",
			calendarDetail.Code, calendarDetail.Body.String())
	}
}
