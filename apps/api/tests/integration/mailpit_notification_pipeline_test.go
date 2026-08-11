package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/communications"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/notifications"
)

func TestNotificationDeliveryUsesAuthoritativeRecipientAndRecordsRetryProvenance(
	t *testing.T,
) {
	pool := canonicalDatabase(t, "notification_delivery_provenance")
	if _, err := pool.Exec(context.Background(), `
		UPDATE identity_references
		SET email = CASE subject_id
			WHEN 'auditee-xyz' THEN 'auditee.xyz@example.test'
			WHEN 'auditee-other' THEN 'other.auditee@example.test'
			ELSE lower(subject_id) || '@caa.example.test'
		END
	`); err != nil {
		t.Fatalf("seed authoritative recipient email: %v", err)
	}
	ids := scenarioIDGenerator()
	workflow := application.NewCommunicationsWorkflow(
		pool,
		application.CommunicationsWorkflowDependencies{
			Clock: func() time.Time { return canonicalNow }, IDGenerator: ids,
		},
	)
	command := application.SendCommunicationCommand{
		OperationID: "op-mailpit-provenance", CorrelationID: "corr-mailpit-provenance",
		IdempotencyKey: "idem-mailpit-provenance", OrganizationID: "airline-xyz",
		Subject: "CAP due soon", Body: "Open the authorized CAP record.",
		Audience: communications.AudienceAuditee,
	}
	first, err := workflow.SendCommunication(
		context.Background(),
		principal(
			"inspector-cabin-001",
			"caa",
			"session-inspector",
			identity.RoleInspector,
		),
		command,
	)
	if err != nil {
		t.Fatalf("create email notification: %v", err)
	}
	replayed, err := workflow.SendCommunication(
		context.Background(),
		principal(
			"inspector-cabin-001",
			"caa",
			"session-inspector",
			identity.RoleInspector,
		),
		command,
	)
	if err != nil || replayed.ID != first.ID {
		t.Fatalf("replay email notification = %+v, err = %v", replayed, err)
	}

	var attempted notifications.EmailDelivery
	retryable := errors.New("private provider response and credential secret")
	failing := notifications.NewDeliveryService(
		pool,
		notifications.DeliveryDependencies{
			Clock: func() time.Time { return canonicalNow }, IDGenerator: ids,
			Adapter: notificationDeliveryAdapterFunc(
				func(
					_ context.Context,
					delivery notifications.EmailDelivery,
				) error {
					attempted = delivery
					return retryable
				},
			),
		},
	)
	processed, err := failing.ProcessNext(context.Background())
	if !processed || !errors.Is(err, retryable) {
		t.Fatalf("first delivery = processed %t err %v", processed, err)
	}
	if attempted.RecipientSubjectID != "auditee-xyz" ||
		attempted.RecipientEmail != "auditee.xyz@example.test" ||
		attempted.RecipientAudience != notifications.EmailAudienceAuditee ||
		attempted.OrganizationID != "airline-xyz" ||
		attempted.OrganizationName != "Airline XYZ" ||
		attempted.ProviderMessageID == "" {
		t.Fatalf("authoritative delivery envelope = %+v", attempted)
	}
	if strings.Contains(attempted.RecipientEmail, "other.auditee") {
		t.Fatalf("cross-organization recipient leaked: %+v", attempted)
	}

	var status, lastError string
	var attemptCount int
	var nextAttemptAt *time.Time
	var providerMessageID *string
	if err := pool.QueryRow(context.Background(), `
		SELECT status, attempt_count, COALESCE(last_error, ''),
		       next_attempt_at, provider_message_id
		FROM notification_delivery_jobs
		WHERE notification_id = $1
	`, attempted.NotificationID).Scan(
		&status,
		&attemptCount,
		&lastError,
		&nextAttemptAt,
		&providerMessageID,
	); err != nil {
		t.Fatalf("read retry provenance: %v", err)
	}
	if status != "FAILED" || attemptCount != 1 ||
		lastError != "SMTP_DELIVERY_FAILED" ||
		nextAttemptAt == nil ||
		!nextAttemptAt.Equal(canonicalNow.Add(30*time.Second)) ||
		providerMessageID != nil {
		t.Fatalf(
			"retry provenance = status %q attempts %d error %q next %v message %v",
			status,
			attemptCount,
			lastError,
			nextAttemptAt,
			providerMessageID,
		)
	}
	retryingPage, err := workflow.ListNotifications(
		context.Background(),
		principal(
			"auditee-xyz",
			"airline-xyz",
			"session-auditee",
			identity.RoleAuditee,
		),
	)
	if err != nil || len(retryingPage.Items) != 1 ||
		retryingPage.Items[0].EmailDeliveryStatus !=
			notifications.EmailDeliveryRetrying ||
		retryingPage.Items[0].EmailDeliveryAttempts != 1 ||
		retryingPage.Items[0].EmailNextAttemptAt == nil {
		t.Fatalf("retrying in-app delivery state = %+v, err = %v", retryingPage, err)
	}
	tooSoon := notifications.NewDeliveryService(
		pool,
		notifications.DeliveryDependencies{
			Clock: func() time.Time { return canonicalNow.Add(29 * time.Second) },
			Adapter: notificationDeliveryAdapterFunc(
				func(_ context.Context, _ notifications.EmailDelivery) error {
					t.Fatal("retry was attempted before next_attempt_at")
					return nil
				},
			),
		},
	)
	if processed, err := tooSoon.ProcessNext(context.Background()); err != nil ||
		processed {
		t.Fatalf("early retry = processed %t err %v", processed, err)
	}

	var accepted notifications.EmailDelivery
	successful := notifications.NewDeliveryService(
		pool,
		notifications.DeliveryDependencies{
			Clock:       func() time.Time { return canonicalNow.Add(30 * time.Second) },
			IDGenerator: ids,
			Adapter: notificationDeliveryAdapterFunc(
				func(
					_ context.Context,
					delivery notifications.EmailDelivery,
				) error {
					accepted = delivery
					return nil
				},
			),
		},
	)
	processed, err = successful.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("successful retry = processed %t err %v", processed, err)
	}
	if accepted.ProviderMessageID != attempted.ProviderMessageID ||
		accepted.Attempt != 2 {
		t.Fatalf("stable retry envelope = first %+v retry %+v", attempted, accepted)
	}
	var acceptedAt, deliveredAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT job.status, job.attempt_count, COALESCE(job.last_error, ''),
		       job.next_attempt_at, job.provider_message_id, job.accepted_at,
		       outbox.delivered_at
		FROM notification_delivery_jobs job
		JOIN outbox_messages outbox ON outbox.id = job.outbox_message_id
		WHERE job.notification_id = $1
	`, accepted.NotificationID).Scan(
		&status,
		&attemptCount,
		&lastError,
		&nextAttemptAt,
		&providerMessageID,
		&acceptedAt,
		&deliveredAt,
	); err != nil {
		t.Fatalf("read accepted delivery: %v", err)
	}
	if status != "DELIVERED" || attemptCount != 2 || lastError != "" ||
		nextAttemptAt != nil || providerMessageID == nil ||
		*providerMessageID != accepted.ProviderMessageID ||
		acceptedAt == nil ||
		!acceptedAt.Equal(canonicalNow.Add(30*time.Second)) ||
		deliveredAt == nil ||
		!deliveredAt.Equal(canonicalNow.Add(30*time.Second)) {
		t.Fatalf(
			"accepted provenance = status %q attempts %d error %q next %v message %v accepted %v delivered %v",
			status,
			attemptCount,
			lastError,
			nextAttemptAt,
			providerMessageID,
			acceptedAt,
			deliveredAt,
		)
	}
	deliveredPage, err := workflow.ListNotifications(
		context.Background(),
		principal(
			"auditee-xyz",
			"airline-xyz",
			"session-auditee",
			identity.RoleAuditee,
		),
	)
	if err != nil || len(deliveredPage.Items) != 1 ||
		deliveredPage.Items[0].EmailDeliveryStatus !=
			notifications.EmailDeliveryDelivered ||
		deliveredPage.Items[0].EmailDeliveryAttempts != 2 ||
		deliveredPage.Items[0].EmailAcceptedAt == nil ||
		deliveredPage.Items[0].EmailNextAttemptAt != nil {
		t.Fatalf("delivered in-app delivery state = %+v, err = %v", deliveredPage, err)
	}

	var auditText string
	if err := pool.QueryRow(context.Background(), `
		SELECT string_agg(details::text, ' ')
		FROM audit_events
		WHERE entity_type = 'NOTIFICATION_DELIVERY'
	`).Scan(&auditText); err != nil {
		t.Fatalf("read delivery audit: %v", err)
	}
	for _, forbidden := range []string{
		"Open the authorized CAP record.",
		"private provider response",
		"credential secret",
		"auditee.xyz@example.test",
	} {
		if strings.Contains(auditText, forbidden) {
			t.Fatalf("delivery audit leaked %q: %s", forbidden, auditText)
		}
	}
}

func TestPermanentNotificationFailureDeadLettersWithoutRetry(t *testing.T) {
	pool := canonicalDatabase(t, "notification_delivery_dead_letter")
	if _, err := pool.Exec(context.Background(), `
		UPDATE identity_references
		SET email = 'auditee.xyz@example.test'
		WHERE subject_id = 'auditee-xyz'
	`); err != nil {
		t.Fatalf("seed authoritative recipient email: %v", err)
	}
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
			"inspector-cabin-001",
			"caa",
			"session-inspector",
			identity.RoleInspector,
		),
		application.SendCommunicationCommand{
			OperationID: "op-dead-letter", CorrelationID: "corr-dead-letter",
			IdempotencyKey: "idem-dead-letter", OrganizationID: "airline-xyz",
			Subject: "Permanent refusal", Body: "Open the authorized record.",
			Audience: communications.AudienceAuditee,
		},
	)
	if err != nil {
		t.Fatalf("create permanent-failure notification: %v", err)
	}
	permanent := notifications.NewPermanentDeliveryFailure(
		"SMTP_RECIPIENT_REJECTED",
	)
	service := notifications.NewDeliveryService(
		pool,
		notifications.DeliveryDependencies{
			Clock:       func() time.Time { return canonicalNow },
			IDGenerator: ids,
			Adapter: notificationDeliveryAdapterFunc(
				func(_ context.Context, _ notifications.EmailDelivery) error {
					return permanent
				},
			),
		},
	)
	processed, err := service.ProcessNext(context.Background())
	if !processed || !errors.Is(err, permanent) {
		t.Fatalf("permanent failure = processed %t err %v", processed, err)
	}
	var status, terminalState, lastError string
	var nextAttemptAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT job.status, COALESCE(job.last_error, ''),
		       job.next_attempt_at, COALESCE(outbox.terminal_state, '')
		FROM notification_delivery_jobs job
		JOIN outbox_messages outbox ON outbox.id = job.outbox_message_id
		JOIN notification_records record ON record.id = job.notification_id
		WHERE record.related_entity_id = $1
	`, message.ID).Scan(
		&status,
		&lastError,
		&nextAttemptAt,
		&terminalState,
	); err != nil {
		t.Fatalf("read dead letter: %v", err)
	}
	if status != "DEAD_LETTER" ||
		lastError != "SMTP_RECIPIENT_REJECTED" ||
		nextAttemptAt != nil ||
		terminalState != "PERMANENT_FAILURE" {
		t.Fatalf(
			"dead letter = status %q error %q next %v terminal %q",
			status,
			lastError,
			nextAttemptAt,
			terminalState,
		)
	}
	if processed, err := service.ProcessNext(context.Background()); err != nil ||
		processed {
		t.Fatalf("dead-letter replay = processed %t err %v", processed, err)
	}
}

func TestRealMailpitDeliveryFailureRestartAndExactMetadata(t *testing.T) {
	smtpAddress := strings.TrimSpace(os.Getenv("AVIA_TEST_MAILPIT_SMTP_ADDRESS"))
	apiBaseURL := strings.TrimRight(
		strings.TrimSpace(os.Getenv("AVIA_TEST_MAILPIT_API_URL")),
		"/",
	)
	passwordFile := strings.TrimSpace(
		os.Getenv("AVIA_TEST_SMTP_PASSWORD_FILE"),
	)
	composeFilesValue := strings.TrimSpace(os.Getenv("AVIA_TEST_COMPOSE_FILES"))
	if composeFilesValue == "" {
		composeFilesValue = strings.TrimSpace(os.Getenv("AVIA_TEST_COMPOSE_FILE"))
	}
	composeFiles := strings.Fields(composeFilesValue)
	composeProject := strings.TrimSpace(os.Getenv("AVIA_TEST_COMPOSE_PROJECT"))
	composeService := strings.TrimSpace(os.Getenv("AVIA_TEST_MAILPIT_SERVICE"))
	if composeService == "" {
		composeService = "mailpit-tools"
	}
	if smtpAddress == "" || apiBaseURL == "" || passwordFile == "" ||
		len(composeFiles) == 0 || composeProject == "" {
		t.Skip("real Mailpit integration environment is not configured")
	}
	passwordBytes, err := os.ReadFile(passwordFile)
	if err != nil {
		t.Fatalf("read task-owned SMTP password: %v", err)
	}
	sender, err := notifications.NewSMTPSender(notifications.SMTPConfig{
		Address:        smtpAddress,
		From:           "no-reply@aviasurveil360.local",
		Username:       "aviasurveil360",
		Password:       strings.TrimSpace(string(passwordBytes)),
		Timeout:        2 * time.Second,
		PrivateNetwork: true,
	})
	if err != nil {
		t.Fatalf("configure real Mailpit sender: %v", err)
	}
	t.Cleanup(func() {
		_ = runMailpitCompose(
			context.Background(),
			composeFiles,
			composeProject,
			"down",
			"--volumes",
			"--remove-orphans",
		)
	})
	waitForMailpitAPI(t, apiBaseURL, true)

	pool := canonicalDatabase(t, "real_mailpit_pipeline")
	ids := scenarioIDGenerator()
	workflow := application.NewCommunicationsWorkflow(
		pool,
		application.CommunicationsWorkflowDependencies{
			Clock: func() time.Time { return canonicalNow }, IDGenerator: ids,
		},
	)
	create := func(
		suffix string,
		subject string,
	) application.SendCommunicationCommand {
		return application.SendCommunicationCommand{
			OperationID:    "op-real-mailpit-" + suffix,
			CorrelationID:  "corr-real-mailpit-" + suffix,
			IdempotencyKey: "idem-real-mailpit-" + suffix,
			OrganizationID: "airline-xyz",
			Subject:        subject,
			Body:           "Open the authorized Auditee record.",
			Audience:       communications.AudienceAuditee,
		}
	}
	firstMessage, err := workflow.SendCommunication(
		context.Background(),
		principal(
			"inspector-cabin-001",
			"caa",
			"session-inspector",
			identity.RoleInspector,
		),
		create("accepted", "Due Soon Evidence reminder"),
	)
	if err != nil {
		t.Fatalf("create real Mailpit notification: %v", err)
	}
	deliveryService := notifications.NewDeliveryService(
		pool,
		notifications.DeliveryDependencies{
			Clock:       func() time.Time { return canonicalNow },
			IDGenerator: ids,
			Adapter:     sender,
		},
	)
	processed, err := deliveryService.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("real Mailpit delivery = processed %t err %v", processed, err)
	}
	if processed, err := deliveryService.ProcessNext(context.Background()); err != nil ||
		processed {
		t.Fatalf("duplicate delivery = processed %t err %v", processed, err)
	}
	messages := waitForMailpitMessages(t, apiBaseURL, 1)
	first := messages[0]
	messageID := stringField(first, "MessageID")
	if stringField(first, "Subject") != "New CAA communication" ||
		strings.Trim(messageID, "<>") == "" ||
		!containsJSONValue(first["To"], "auditee.xyz@example.test") {
		t.Fatalf("real Mailpit metadata = %+v", first)
	}
	mailpitID := stringField(first, "ID")
	if mailpitID == "" {
		t.Fatalf("Mailpit message omitted ID: %+v", first)
	}
	detail := getMailpitJSON(t, apiBaseURL+"/api/v1/message/"+url.PathEscape(mailpitID))
	detailText := strings.ToLower(fmt.Sprint(detail))
	for _, expected := range []string{
		"auditee.xyz@example.test",
		"due soon evidence reminder",
		"open the authorized message record",
	} {
		if !strings.Contains(detailText, expected) {
			t.Fatalf("Mailpit detail omitted %q: %+v", expected, detail)
		}
	}
	for _, forbidden := range []string{
		"internal caa note",
		"enforcement deliberation",
		"auditee.other@example.test",
	} {
		if strings.Contains(detailText, forbidden) {
			t.Fatalf("Mailpit detail leaked %q: %+v", forbidden, detail)
		}
	}
	var storedMessageID string
	if err := pool.QueryRow(context.Background(), `
		SELECT provider_message_id
		FROM notification_delivery_jobs job
		JOIN notification_records record ON record.id = job.notification_id
		WHERE record.related_entity_id = $1
	`, firstMessage.ID).Scan(&storedMessageID); err != nil {
		t.Fatalf("read accepted provider Message-ID: %v", err)
	}
	if strings.Trim(storedMessageID, "<>") != strings.Trim(messageID, "<>") {
		t.Fatalf(
			"Message-ID mismatch: database %q Mailpit %q",
			storedMessageID,
			messageID,
		)
	}

	secondMessage, err := workflow.SendCommunication(
		context.Background(),
		principal(
			"inspector-cabin-001",
			"caa",
			"session-inspector",
			identity.RoleInspector,
		),
		create("retry", "Overdue CAP reminder"),
	)
	if err != nil {
		t.Fatalf("create retry notification: %v", err)
	}
	if err := runMailpitCompose(
		context.Background(),
		composeFiles,
		composeProject,
		"stop",
		composeService,
	); err != nil {
		t.Fatalf("stop task-owned Mailpit: %v", err)
	}
	waitForMailpitAPI(t, apiBaseURL, false)
	failingService := notifications.NewDeliveryService(
		pool,
		notifications.DeliveryDependencies{
			Clock:       func() time.Time { return canonicalNow.Add(time.Minute) },
			IDGenerator: ids,
			Adapter:     sender,
		},
	)
	processed, err = failingService.ProcessNext(context.Background())
	if !processed || err == nil ||
		notifications.IsPermanentDeliveryFailure(err) {
		t.Fatalf("Mailpit outage delivery = processed %t err %v", processed, err)
	}
	var retryStatus string
	var nextAttemptAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT job.status, job.next_attempt_at
		FROM notification_delivery_jobs job
		JOIN notification_records record ON record.id = job.notification_id
		WHERE record.related_entity_id = $1
	`, secondMessage.ID).Scan(&retryStatus, &nextAttemptAt); err != nil {
		t.Fatalf("read Mailpit outage retry: %v", err)
	}
	if retryStatus != "FAILED" || nextAttemptAt == nil ||
		!nextAttemptAt.Equal(canonicalNow.Add(90*time.Second)) {
		t.Fatalf("Mailpit outage retry = %q next %v", retryStatus, nextAttemptAt)
	}
	if err := runMailpitCompose(
		context.Background(),
		composeFiles,
		composeProject,
		"--profile",
		"tools",
		"up",
		"--detach",
		"--wait",
		composeService,
	); err != nil {
		t.Fatalf("restart task-owned Mailpit: %v", err)
	}
	waitForMailpitAPI(t, apiBaseURL, true)
	retryService := notifications.NewDeliveryService(
		pool,
		notifications.DeliveryDependencies{
			Clock:       func() time.Time { return canonicalNow.Add(90 * time.Second) },
			IDGenerator: ids,
			Adapter:     sender,
		},
	)
	processed, err = retryService.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("Mailpit restart delivery = processed %t err %v", processed, err)
	}
	messages = waitForMailpitMessages(t, apiBaseURL, 2)
	messageMetadata := strings.ToLower(fmt.Sprint(messages))
	for _, message := range messages {
		if stringField(message, "Subject") != "New CAA communication" {
			t.Fatalf("Mailpit restart subject metadata = %+v", message)
		}
	}
	if !strings.Contains(messageMetadata, "due soon evidence reminder") ||
		!strings.Contains(messageMetadata, "overdue cap reminder") {
		t.Fatalf("Mailpit restart message metadata = %+v", messages)
	}
}

func runMailpitCompose(
	ctx context.Context,
	composeFiles []string,
	project string,
	arguments ...string,
) error {
	base := []string{"compose"}
	for _, composeFile := range composeFiles {
		base = append(base, "--file", composeFile)
	}
	base = append(base, "--project-name", project)
	command := exec.CommandContext(
		ctx,
		"docker",
		append(base, arguments...)...,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose failed: %w: %s", err, output)
	}
	return nil
}

func waitForMailpitAPI(t *testing.T, baseURL string, available bool) {
	t.Helper()
	client := &http.Client{Timeout: 250 * time.Millisecond}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(baseURL + "/api/v1/messages")
		if err == nil {
			_ = response.Body.Close()
		}
		reachable := err == nil && response.StatusCode == http.StatusOK
		if reachable == available {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Mailpit API availability did not become %t", available)
}

func waitForMailpitMessages(
	t *testing.T,
	baseURL string,
	expected int,
) []map[string]any {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response := getMailpitJSON(t, baseURL+"/api/v1/messages")
		rawMessages, ok := fieldFold(response, "messages").([]any)
		if ok && len(rawMessages) == expected {
			messages := make([]map[string]any, 0, len(rawMessages))
			for _, raw := range rawMessages {
				message, ok := raw.(map[string]any)
				if !ok {
					t.Fatalf("Mailpit message metadata type = %T", raw)
				}
				messages = append(messages, message)
			}
			return messages
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Mailpit did not expose exactly %d messages", expected)
	return nil
}

func getMailpitJSON(t *testing.T, endpoint string) map[string]any {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(endpoint)
	if err != nil {
		t.Fatalf("GET Mailpit API: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		t.Fatalf("read Mailpit API: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Mailpit API status = %d", response.StatusCode)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode Mailpit API: %v", err)
	}
	return decoded
}

func fieldFold(value map[string]any, expected string) any {
	for key, candidate := range value {
		if strings.EqualFold(key, expected) {
			return candidate
		}
	}
	return nil
}

func stringField(value map[string]any, expected string) string {
	candidate, _ := fieldFold(value, expected).(string)
	return candidate
}

func containsJSONValue(value any, expected string) bool {
	switch candidate := value.(type) {
	case string:
		return candidate == expected
	case []any:
		for _, item := range candidate {
			if containsJSONValue(item, expected) {
				return true
			}
		}
	case map[string]any:
		for _, item := range candidate {
			if containsJSONValue(item, expected) {
				return true
			}
		}
	}
	return false
}
