//go:build canonicaltest

package identitysetup_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/administration"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestTask4PrepareOIDCHarnessApplicationAdministrator(t *testing.T) {
	databaseURL := requiredEnvironment(t, "AVIA_TEST_DATABASE_URL")
	adminEmail := strings.ToLower(
		requiredEnvironment(t, "AVIA_OIDC_TEST_ADMIN_USERNAME"),
	)
	issuer := requiredEnvironment(t, "AVIA_TEST_OIDC_ISSUER_URL")
	ctx := context.Background()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open OIDC harness database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply OIDC harness migrations: %v", err)
	}
	now := time.Now().UTC()
	var sequence atomic.Uint64
	nextID := func(prefix string) string {
		return fmt.Sprintf(
			"%s-task4-oidc-%d",
			prefix,
			sequence.Add(1),
		)
	}
	userService := administration.NewUserService(
		pool,
		administration.UserServiceDependencies{
			Clock:       func() time.Time { return now },
			IDGenerator: nextID,
		},
	)
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_references (
			subject_id, issuer, display_name, created_at
		) VALUES (
			'task4-one-shot-identity-bootstrap',
			'urn:aviasurveil360:test-harness',
			'Task 4 one-shot identity bootstrap',
			$1
		)
	`, now); err != nil {
		t.Fatalf("retain one-shot OIDC harness bootstrap identity: %v", err)
	}
	request, err := userService.RequestLifecycle(
		ctx,
		identity.Principal{
			SubjectID:      "task4-one-shot-identity-bootstrap",
			OrganizationID: "CAA",
			Roles:          []identity.Role{identity.RoleAdmin},
		},
		administration.RequestUserLifecycleCommand{
			OperationID:                "task4-oidc-admin-provision",
			IdempotencyKey:             "task4-oidc-admin-provision",
			Action:                     administration.UserLifecycleProvision,
			Roles:                      []identity.Role{identity.RoleAdmin},
			OrganizationID:             "CAA",
			Email:                      adminEmail,
			DisplayName:                "Local Administrator",
			Reason:                     "Bind the isolated Task 4 OIDC harness administrator through the authoritative lifecycle.",
			ExpectedMembershipRevision: 0,
		},
	)
	if err != nil {
		t.Fatalf("request OIDC harness administrator lifecycle: %v", err)
	}
	keycloakClient, err := identity.NewKeycloakAdminClient(
		identity.KeycloakAdminConfig{
			BaseURL: requiredEnvironment(
				t,
				"AVIA_OIDC_TEST_KEYCLOAK_BASE_URL",
			),
			Realm:        "aviasurveil360",
			ClientID:     "aviasurveil360-lifecycle",
			ClientSecret: requiredEnvironment(t, "AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET"),
		},
	)
	if err != nil {
		t.Fatalf("configure OIDC harness lifecycle provider: %v", err)
	}
	worker := administration.NewUserLifecycleWorker(
		pool,
		keycloakClient,
		administration.UserLifecycleWorkerDependencies{
			Clock:         func() time.Time { return now },
			IDGenerator:   nextID,
			WorkerID:      "task4-oidc-identity-setup",
			LeaseDuration: time.Minute,
			RetryDelay:    time.Second,
			Issuer:        issuer,
		},
	)
	processed, err := worker.ProcessNext(ctx)
	if err != nil || !processed {
		t.Fatalf(
			"process OIDC harness administrator lifecycle = %t, %v",
			processed,
			err,
		)
	}
	var subjectID, membershipState string
	var revision int64
	if err := pool.QueryRow(ctx, `
		SELECT version.subject_id, version.membership_state, version.revision
		FROM desired_membership_versions version
		JOIN user_lifecycle_requests request
		  ON request.id = version.source_request_id
		WHERE request.id = $1
	`, request.ID).Scan(
		&subjectID,
		&membershipState,
		&revision,
	); err != nil {
		t.Fatalf("read OIDC harness administrator membership: %v", err)
	}
	if subjectID == "" || membershipState != "INVITED" || revision != 1 {
		t.Fatalf(
			"OIDC harness administrator membership = %q/%q/%d",
			subjectID,
			membershipState,
			revision,
		)
	}
	var bootstrapMemberships, bootstrapSessions int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*)
			 FROM desired_membership_versions
			 WHERE subject_id = 'task4-one-shot-identity-bootstrap'),
			(SELECT count(*)
			 FROM session_references
			 WHERE subject_id = 'task4-one-shot-identity-bootstrap')
	`).Scan(&bootstrapMemberships, &bootstrapSessions); err != nil {
		t.Fatalf("read one-shot bootstrap application authority: %v", err)
	}
	if bootstrapMemberships != 0 || bootstrapSessions != 0 {
		t.Fatalf(
			"one-shot bootstrap application authority = memberships %d sessions %d",
			bootstrapMemberships,
			bootstrapSessions,
		)
	}
}

func TestTask4DiagnoseOIDCHarnessApplicationAdministrator(t *testing.T) {
	ctx := context.Background()
	pool, err := database.Open(
		ctx,
		requiredEnvironment(t, "AVIA_TEST_DATABASE_URL"),
	)
	if err != nil {
		t.Fatalf("open OIDC harness database: %v", err)
	}
	t.Cleanup(pool.Close)
	subjectID := requiredEnvironment(t, "AVIA_OIDC_TEST_ADMIN_SUBJECT_ID")
	var membershipState, organizationID, driftState, observedOrganizationID string
	var revision, syncRevision, sessionCount int64
	var roles, observedRoles []string
	var providerEnabled bool
	var observedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT version.membership_state, version.organization_id,
		       version.roles, version.revision, sync.desired_revision,
		       sync.observed_provider_enabled,
		       sync.observed_organization_id, sync.observed_roles,
		       sync.observed_at, sync.drift_state,
		       (SELECT count(*)
		        FROM session_references
		        WHERE subject_id = version.subject_id)
		FROM desired_membership_versions version
		JOIN desired_membership_sync sync
		  ON sync.membership_id = version.membership_id
		WHERE version.subject_id = $1
		ORDER BY version.revision DESC
		LIMIT 1
	`, subjectID).Scan(
		&membershipState,
		&organizationID,
		&roles,
		&revision,
		&syncRevision,
		&providerEnabled,
		&observedOrganizationID,
		&observedRoles,
		&observedAt,
		&driftState,
		&sessionCount,
	); err != nil {
		t.Fatalf("read OIDC harness application authority: %v", err)
	}
	var retainedIssuer, profileOrganizationID string
	if err := pool.QueryRow(ctx, `
		SELECT identity.issuer, profile.organization_id
		FROM identity_references identity
		JOIN user_profiles profile
		  ON profile.subject_id = identity.subject_id
		WHERE identity.subject_id = $1
	`, subjectID).Scan(
		&retainedIssuer,
		&profileOrganizationID,
	); err != nil {
		t.Fatalf("read OIDC harness retained identity: %v", err)
	}
	var sessionState, sessionRevokedAt, sessionObservedAt string
	var sessionOrganizationID, sessionMembershipID string
	var sessionMembershipRevision int64
	var sessionRoles []string
	var denialReason string
	if sessionCount > 0 {
		if err := pool.QueryRow(ctx, `
			SELECT COALESCE(session.authority_state, ''),
			       COALESCE(session.revoked_at::text, ''),
			       COALESCE(session.authority_observed_at::text, ''),
			       COALESCE(session.organization_id, ''),
			       COALESCE(session.membership_id, ''),
			       COALESCE(session.membership_revision, 0),
			       session.roles,
			       COALESCE((
			         SELECT event.details ->> 'reasonCode'
			         FROM audit_events event
			         WHERE event.entity_type = 'SESSION'
			           AND event.entity_id = session.id
			           AND event.action = 'SESSION_AUTHORITY_DENIED'
			         ORDER BY event.sequence_id DESC
			         LIMIT 1
			       ), '')
			FROM session_references session
			WHERE session.subject_id = $1
			ORDER BY session.created_at DESC
			LIMIT 1
		`, subjectID).Scan(
			&sessionState,
			&sessionRevokedAt,
			&sessionObservedAt,
			&sessionOrganizationID,
			&sessionMembershipID,
			&sessionMembershipRevision,
			&sessionRoles,
			&denialReason,
		); err != nil {
			t.Fatalf("read OIDC harness session authority: %v", err)
		}
	}
	keycloakClient, err := identity.NewKeycloakAdminClient(
		identity.KeycloakAdminConfig{
			BaseURL: requiredEnvironment(
				t,
				"AVIA_OIDC_TEST_KEYCLOAK_BASE_URL",
			),
			Realm:        "aviasurveil360",
			ClientID:     "aviasurveil360-lifecycle",
			ClientSecret: requiredEnvironment(t, "AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET"),
		},
	)
	if err != nil {
		t.Fatalf("configure OIDC harness authority observer: %v", err)
	}
	observation, err := keycloakClient.ObserveUserAuthority(ctx, subjectID)
	if err != nil {
		t.Fatalf("observe OIDC harness provider authority: %v", err)
	}
	t.Logf(
		"application authority: state=%s revision=%d syncRevision=%d organization=%s roles=%v providerEnabled=%t observedOrganization=%s observedRoles=%v observedAt=%s drift=%s sessions=%d retainedIssuer=%s profileOrganization=%s",
		membershipState,
		revision,
		syncRevision,
		organizationID,
		roles,
		providerEnabled,
		observedOrganizationID,
		observedRoles,
		observedAt.UTC().Format(time.RFC3339Nano),
		driftState,
		sessionCount,
		retainedIssuer,
		profileOrganizationID,
	)
	t.Logf(
		"session authority: state=%s revokedAt=%s observedAt=%s organization=%s roles=%v membership=%s revision=%d denialReason=%s",
		sessionState,
		sessionRevokedAt,
		sessionObservedAt,
		sessionOrganizationID,
		sessionRoles,
		sessionMembershipID,
		sessionMembershipRevision,
		denialReason,
	)
	t.Logf(
		"provider authority: enabled=%t locked=%t organization=%s roles=%v requiredActions=%v mfaEnrolled=%t",
		observation.Enabled,
		observation.Locked,
		observation.OrganizationID,
		observation.Roles,
		observation.RequiredActions,
		observation.MFAEnrolled,
	)
}

func requiredEnvironment(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}
