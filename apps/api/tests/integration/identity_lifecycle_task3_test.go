package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/administration"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
)

func TestTask3StaleMembershipRevisionHasNoAuthoritySideEffects(t *testing.T) {
	pool := canonicalDatabase(t, "task3_stale_membership")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (
			subject_id, issuer, display_name, email
		) VALUES (
			'task3-admin-stale', 'test', 'Task Three Administrator',
			'task3.admin.stale@example.test'
		)
	`); err != nil {
		t.Fatalf("seed stale-revision administrator: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO user_lifecycle_requests (
			id, subject_id, requested_action, requested_roles,
			requested_organization_id, requested_email,
			requested_display_name, status, idempotency_key,
			expected_membership_revision, reason, requested_by_subject_id,
			created_at, updated_at
		) VALUES (
			'task3-membership-seed', 'auditee-xyz', 'PROVISION',
			ARRAY['auditee'], 'airline-xyz', 'auditee.xyz@example.test',
			'Airline XYZ Auditee', 'SUCCEEDED',
			'task3-membership-seed', 0, 'Approved membership seed.',
			'task3-admin-stale', $1, $1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed stale-revision source request: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO desired_membership_versions (
			membership_id, subject_id, revision, membership_state,
			organization_id, roles, requested_by_subject_id, reason,
			source_request_id, requested_at, effective_at,
			observed_provider_enabled, observed_organization_id,
			observed_roles, observed_at, drift_state
		) VALUES (
			'task3-membership-auditee', 'auditee-xyz', 1, 'ACTIVE',
			'airline-xyz', ARRAY['auditee'], 'task3-admin-stale',
			'Approved membership seed.', 'task3-membership-seed',
			$1, $1, true, 'airline-xyz', ARRAY['auditee'], $1, 'IN_SYNC'
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed stale-revision membership: %v", err)
	}
	service := administration.NewUserService(
		pool,
		administration.UserServiceDependencies{
			Clock: func() time.Time { return canonicalNow },
			IDGenerator: func(prefix string) string {
				return prefix + "-task3-stale"
			},
		},
	)
	requested, err := service.RequestLifecycle(
		context.Background(),
		principal(
			"task3-admin-stale",
			"CAA",
			"session-admin",
			identity.RoleAdmin,
		),
		administration.RequestUserLifecycleCommand{
			OperationID:                "task3-stale-force-logout",
			IdempotencyKey:             "task3-stale-force-logout",
			SubjectID:                  "auditee-xyz",
			Action:                     administration.UserLifecycleForceLogout,
			Roles:                      []identity.Role{identity.RoleAuditee},
			OrganizationID:             "airline-xyz",
			Reason:                     "Approved stale-revision proof.",
			ExpectedMembershipRevision: 1,
		},
	)
	if err != nil {
		t.Fatalf("request stale-revision lifecycle action: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO desired_membership_versions (
			membership_id, subject_id, revision, membership_state,
			organization_id, roles, requested_by_subject_id, reason,
			source_request_id, requested_at, effective_at,
			observed_provider_enabled, observed_organization_id,
			observed_roles, observed_at, drift_state
		) VALUES (
			'task3-membership-auditee', 'auditee-xyz', 2, 'ACTIVE',
			'airline-xyz', ARRAY['auditee'], 'task3-admin-stale',
			'Concurrent approved membership change.', $1, $2, $2, true,
			'airline-xyz', ARRAY['auditee'], $2, 'IN_SYNC'
		)
	`, requested.ID, canonicalNow.Add(time.Second)); err != nil {
		t.Fatalf("append concurrent membership revision: %v", err)
	}
	provider := &lifecycleIdentityProvider{}
	worker := administration.NewUserLifecycleWorker(
		pool,
		provider,
		administration.UserLifecycleWorkerDependencies{
			Clock: func() time.Time { return canonicalNow.Add(2 * time.Second) },
			IDGenerator: func(prefix string) string {
				return prefix + "-task3-stale"
			},
			WorkerID: "task3-stale-worker",
			Issuer:   "https://identity.example.test/realms/aviasurveil360",
		},
	)
	processed, err := worker.ProcessNext(context.Background())
	if !processed || !errors.Is(err, administration.ErrMembershipRevisionConflict) {
		t.Fatalf("process stale lifecycle = %t, err = %v", processed, err)
	}
	if len(provider.loggedOutSubjects) != 0 ||
		len(provider.disabledSubjects) != 0 ||
		len(provider.executeActions) != 0 {
		t.Fatalf("stale lifecycle reached provider: %#v", provider)
	}
	var status, failureClass, failureReason, terminalState string
	var deliveredAt, revokedAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT request.status, request.provider_failure_class,
		       request.failure_reason, COALESCE(outbox.terminal_state, ''),
		       outbox.delivered_at
		FROM user_lifecycle_requests request
		JOIN outbox_messages outbox ON outbox.id = request.outbox_message_id
		WHERE request.id = $1
	`, requested.ID).Scan(
		&status,
		&failureClass,
		&failureReason,
		&terminalState,
		&deliveredAt,
	); err != nil {
		t.Fatalf("read stale lifecycle result: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT revoked_at
		FROM session_references
		WHERE id = 'session-auditee'
	`).Scan(&revokedAt); err != nil {
		t.Fatalf("read stale lifecycle session: %v", err)
	}
	if status != "FAILED_PERMANENT" ||
		failureClass != "PERMANENT" ||
		failureReason != "STALE_MEMBERSHIP_REVISION" ||
		terminalState != "FAILED_PERMANENT" ||
		deliveredAt != nil ||
		revokedAt != nil {
		t.Fatalf(
			"stale lifecycle result = status %q class %q reason %q terminal %q delivered %v revoked %v",
			status,
			failureClass,
			failureReason,
			terminalState,
			deliveredAt,
			revokedAt,
		)
	}
	var successCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM audit_events
		WHERE entity_id = $1
		  AND action LIKE '%SUCCEEDED'
	`, requested.ID).Scan(&successCount); err != nil {
		t.Fatalf("count stale lifecycle success audit: %v", err)
	}
	if successCount != 0 {
		t.Fatalf("stale lifecycle wrote %d success audits", successCount)
	}
}

func TestTask3ProviderFailuresAreBoundedAndClassified(t *testing.T) {
	pool := canonicalDatabase(t, "task3_provider_failures")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (
			subject_id, issuer, display_name, email
		) VALUES (
			'task3-admin-failures', 'test', 'Task Three Administrator',
			'task3.admin.failures@example.test'
		)
	`); err != nil {
		t.Fatalf("seed failure-classification administrator: %v", err)
	}
	now := canonicalNow
	counter := 0
	service := administration.NewUserService(
		pool,
		administration.UserServiceDependencies{
			Clock: func() time.Time { return now },
			IDGenerator: func(prefix string) string {
				counter++
				return fmt.Sprintf("%s-task3-failure-%03d", prefix, counter)
			},
		},
	)
	admin := principal(
		"task3-admin-failures",
		"CAA",
		"session-admin",
		identity.RoleAdmin,
	)
	requestProvision := func(label, email string) administration.UserLifecycleRequest {
		t.Helper()
		record, err := service.RequestLifecycle(
			context.Background(),
			admin,
			administration.RequestUserLifecycleCommand{
				OperationID:    label,
				IdempotencyKey: label,
				Action:         administration.UserLifecycleProvision,
				Roles:          []identity.Role{identity.RoleAuditee},
				OrganizationID: "airline-xyz",
				Email:          email,
				DisplayName:    "Task Three Auditee",
				Reason:         "Approved provider failure classification proof.",
			},
		)
		if err != nil {
			t.Fatalf("request %s: %v", label, err)
		}
		return record
	}

	permanent := requestProvision(
		"task3-permanent-provider-failure",
		"task3.permanent@example.test",
	)
	permanentProvider := &lifecycleIdentityProvider{
		provisionError: identity.ErrKeycloakPermanent,
	}
	permanentWorker := administration.NewUserLifecycleWorker(
		pool,
		permanentProvider,
		administration.UserLifecycleWorkerDependencies{
			Clock: func() time.Time { return now },
			IDGenerator: func(prefix string) string {
				counter++
				return fmt.Sprintf("%s-task3-permanent-%03d", prefix, counter)
			},
			WorkerID: "task3-permanent-worker",
			Issuer:   "https://identity.example.test/realms/aviasurveil360",
		},
	)
	if processed, err := permanentWorker.ProcessNext(context.Background()); !processed ||
		!errors.Is(err, identity.ErrKeycloakPermanent) {
		t.Fatalf("process permanent failure = %t, err = %v", processed, err)
	}
	assertTask3FailureState(
		t,
		pool,
		permanent.ID,
		"FAILED_PERMANENT",
		"PERMANENT",
		"PROVIDER_REJECTED",
		1,
	)

	retryable := requestProvision(
		"task3-retryable-provider-failure",
		"task3.retryable@example.test",
	)
	retryableProvider := &lifecycleIdentityProvider{
		provisionError: identity.ErrKeycloakUnavailable,
	}
	retryableWorker := administration.NewUserLifecycleWorker(
		pool,
		retryableProvider,
		administration.UserLifecycleWorkerDependencies{
			Clock: func() time.Time { return now },
			IDGenerator: func(prefix string) string {
				counter++
				return fmt.Sprintf("%s-task3-retryable-%03d", prefix, counter)
			},
			WorkerID:       "task3-retryable-worker",
			RetryDelay:     time.Second,
			MaxAttempts:    2,
			MaxRetryWindow: time.Hour,
			Issuer:         "https://identity.example.test/realms/aviasurveil360",
		},
	)
	if processed, err := retryableWorker.ProcessNext(context.Background()); !processed ||
		!errors.Is(err, identity.ErrKeycloakUnavailable) {
		t.Fatalf("process retryable failure = %t, err = %v", processed, err)
	}
	assertTask3FailureState(
		t,
		pool,
		retryable.ID,
		"FAILED_RETRYABLE",
		"RETRYABLE",
		"PROVIDER_UNAVAILABLE",
		0,
	)
	now = now.Add(time.Second)
	processed, exhaustedErr := retryableWorker.ProcessNext(context.Background())
	if !processed || !errors.Is(exhaustedErr, identity.ErrKeycloakUnavailable) {
		t.Fatalf("process exhausted retry = %t, err = %v", processed, exhaustedErr)
	}
	if exhaustedErr.Error() != identity.ErrKeycloakUnavailable.Error() {
		t.Fatalf("record exhausted retry: %v", exhaustedErr)
	}
	assertTask3FailureState(
		t,
		pool,
		retryable.ID,
		"MANUAL_REVIEW",
		"MANUAL_REVIEW",
		"PROVIDER_RETRY_EXHAUSTED",
		1,
	)

	retryableProvider.provisionError = nil
	retryableProvider.provisionedSubjectID = "task3-operator-reconciled-subject"
	reconciled := requestProvision(
		"task3-operator-reconciled-provider-failure",
		"task3.retryable@example.test",
	)
	if processed, err := retryableWorker.ProcessNext(context.Background()); !processed ||
		err != nil {
		t.Fatalf("process operator reconciliation = %t, err = %v", processed, err)
	}
	var reconciledStatus, reconciledSubject string
	if err := pool.QueryRow(context.Background(), `
		SELECT status, subject_id
		FROM user_lifecycle_requests
		WHERE id = $1
	`, reconciled.ID).Scan(&reconciledStatus, &reconciledSubject); err != nil {
		t.Fatalf("read operator reconciliation outcome: %v", err)
	}
	if reconciledStatus != "SUCCEEDED" ||
		reconciledSubject != "task3-operator-reconciled-subject" {
		t.Fatalf(
			"operator reconciliation = status %q subject %q",
			reconciledStatus,
			reconciledSubject,
		)
	}
}

func TestTask3DuplicateProviderSubjectRequiresManualReviewBeforeDelivery(
	t *testing.T,
) {
	pool := canonicalDatabase(t, "task3_duplicate_provider_subject")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (
			subject_id, issuer, display_name, email
		) VALUES
			(
				'task3-admin-duplicate-subject', 'test',
				'Task Three Administrator',
				'task3.admin.duplicate.subject@example.test'
			),
			(
				'task3-retained-provider-subject',
				'https://identity.example.test/realms/aviasurveil360',
				'Retained Identity',
				'retained.identity@example.test'
			)
	`); err != nil {
		t.Fatalf("seed duplicate-subject identities: %v", err)
	}
	service := administration.NewUserService(
		pool,
		administration.UserServiceDependencies{
			Clock: func() time.Time { return canonicalNow },
			IDGenerator: func(prefix string) string {
				return prefix + "-task3-duplicate-subject"
			},
		},
	)
	requested, err := service.RequestLifecycle(
		context.Background(),
		principal(
			"task3-admin-duplicate-subject",
			"CAA",
			"session-admin",
			identity.RoleAdmin,
		),
		administration.RequestUserLifecycleCommand{
			OperationID:    "task3-duplicate-provider-subject",
			IdempotencyKey: "task3-duplicate-provider-subject",
			Action:         administration.UserLifecycleProvision,
			Roles:          []identity.Role{identity.RoleAuditee},
			OrganizationID: "airline-xyz",
			Email:          "new.identity@example.test",
			DisplayName:    "New Identity",
			Reason:         "Approved duplicate-subject classification proof.",
		},
	)
	if err != nil {
		t.Fatalf("request duplicate-subject provisioning: %v", err)
	}
	provider := &lifecycleIdentityProvider{
		provisionedSubjectID: "task3-retained-provider-subject",
	}
	worker := administration.NewUserLifecycleWorker(
		pool,
		provider,
		administration.UserLifecycleWorkerDependencies{
			Clock: func() time.Time { return canonicalNow },
			IDGenerator: func(prefix string) string {
				return prefix + "-task3-duplicate-subject"
			},
			WorkerID: "task3-duplicate-subject-worker",
			Issuer:   "https://identity.example.test/realms/aviasurveil360",
		},
	)
	if processed, err := worker.ProcessNext(context.Background()); !processed ||
		!errors.Is(err, identity.ErrKeycloakDuplicateSubject) {
		t.Fatalf("process duplicate provider subject = %t, err = %v", processed, err)
	}
	assertTask3FailureState(
		t,
		pool,
		requested.ID,
		"MANUAL_REVIEW",
		"MANUAL_REVIEW",
		"DUPLICATE_SUBJECT",
		1,
	)
	if len(provider.executeActions) != 0 {
		t.Fatalf(
			"duplicate provider subject reached invitation delivery: %#v",
			provider.executeActions,
		)
	}
}

func assertTask3FailureState(
	t *testing.T,
	pool *database.Pool,
	requestID,
	expectedStatus,
	expectedClass,
	expectedReason string,
	expectedAlerts int,
) {
	t.Helper()
	var status, failureClass, failureReason string
	var alertCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT status, COALESCE(provider_failure_class, ''),
		       COALESCE(failure_reason, '')
		FROM user_lifecycle_requests
		WHERE id = $1
	`, requestID).Scan(&status, &failureClass, &failureReason); err != nil {
		t.Fatalf("read lifecycle failure state: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM identity_lifecycle_alerts
		WHERE request_id = $1
	`, requestID).Scan(&alertCount); err != nil {
		t.Fatalf("count lifecycle failure alerts: %v", err)
	}
	if status != expectedStatus ||
		failureClass != expectedClass ||
		failureReason != expectedReason ||
		alertCount != expectedAlerts {
		t.Fatalf(
			"failure state = status %q class %q reason %q alerts %d",
			status,
			failureClass,
			failureReason,
			alertCount,
		)
	}
}

func TestTask3KeycloakInvitationAllEightRoles(t *testing.T) {
	baseURL := os.Getenv("AVIA_TEST_KEYCLOAK_ADMIN_URL")
	clientSecret := os.Getenv("AVIA_TEST_KEYCLOAK_SERVICE_CLIENT_SECRET")
	mailpitURL := os.Getenv("AVIA_TEST_MAILPIT_HTTP_URL")
	if baseURL == "" || clientSecret == "" || mailpitURL == "" {
		t.Skip("requires the Plan 5 Task 3 Keycloak/Mailpit harness")
	}
	client, err := identity.NewKeycloakAdminClient(identity.KeycloakAdminConfig{
		BaseURL:      baseURL,
		Realm:        os.Getenv("AVIA_TEST_KEYCLOAK_REALM"),
		ClientID:     os.Getenv("AVIA_TEST_KEYCLOAK_SERVICE_CLIENT_ID"),
		ClientSecret: clientSecret,
		HTTPClient:   &http.Client{Timeout: 10 * time.Second},
	})
	if err != nil {
		t.Fatalf("create Keycloak admin client: %v", err)
	}
	roles := []identity.Role{
		identity.RoleInspector,
		identity.RoleLeadInspector,
		identity.RoleDepartmentManager,
		identity.RoleGeneralManager,
		identity.RoleFinance,
		identity.RoleExecutiveDirector,
		identity.RoleAuditee,
		identity.RoleAdmin,
	}
	expectedEmails := make(map[string]bool, len(roles))
	for index, role := range roles {
		organizationID := "CAA"
		if role == identity.RoleAuditee {
			organizationID = "ORG-TASK3-AUDITEE"
		}
		email := fmt.Sprintf(
			"task3.%02d.%s@example.test",
			index+1,
			strings.ToLower(string(role)),
		)
		subjectID, err := client.ProvisionUser(
			context.Background(),
			identity.KeycloakUser{
				Email:          email,
				FirstName:      "Task",
				LastName:       fmt.Sprintf("Role %02d", index+1),
				OrganizationID: organizationID,
				Roles:          []identity.Role{role},
			},
		)
		if err != nil {
			t.Fatalf("provision %s: %v", role, err)
		}
		if err := client.IssueExecuteActionsEmail(
			context.Background(),
			subjectID,
			[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
			24*60*60,
		); err != nil {
			t.Fatalf("issue invitation for %s: %v", role, err)
		}
		page, err := client.ListDirectory(
			context.Background(),
			identity.KeycloakDirectoryQuery{
				Search: email,
				Limit:  5,
			},
		)
		if err != nil {
			t.Fatalf("read provider state for %s: %v", role, err)
		}
		if len(page.Users) != 1 ||
			page.Users[0].SubjectID != subjectID ||
			!slices.Equal(page.Users[0].Roles, []identity.Role{role}) ||
			!sameTask3Strings(
				page.Users[0].RequiredActions,
				[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
			) ||
			page.Users[0].TOTPConfigured {
			t.Fatalf("provider invitation state for %s = %+v", role, page.Users)
		}
		expectedEmails[email] = true
	}
	waitForTask3MailpitDeliveries(t, mailpitURL, expectedEmails)
}

func sameTask3Strings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}

func waitForTask3MailpitDeliveries(
	t *testing.T,
	mailpitURL string,
	expectedEmails map[string]bool,
) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(15 * time.Second)
	lastMessageCount := 0
	missing := make([]string, 0, len(expectedEmails))
	for time.Now().Before(deadline) {
		response, err := client.Get(mailpitURL + "/api/v1/messages?limit=50")
		if err == nil && response.StatusCode == http.StatusOK {
			var payload map[string]any
			decodeErr := json.NewDecoder(response.Body).Decode(&payload)
			response.Body.Close()
			if decodeErr == nil {
				rawMessages, _ := fieldFold(payload, "messages").([]any)
				lastMessageCount = len(rawMessages)
				allDelivered := len(rawMessages) >= len(expectedEmails)
				missing = missing[:0]
				for email := range expectedEmails {
					matched := false
					for _, rawMessage := range rawMessages {
						message, _ := rawMessage.(map[string]any)
						if containsJSONValue(fieldFold(message, "To"), email) {
							matched = true
							break
						}
					}
					if !matched {
						missing = append(missing, email)
						allDelivered = false
					}
				}
				if allDelivered {
					return
				}
			}
		} else if response != nil {
			response.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf(
		"Mailpit reported %d messages but missed %d of %d synthetic recipients: %v",
		lastMessageCount,
		len(missing),
		len(expectedEmails),
		missing,
	)
}
