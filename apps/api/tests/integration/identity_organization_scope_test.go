package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/administration"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/organizations"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/idempotency"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

func TestIdentityProfileAndSettingsAreRevisionedWithoutClientRoleAuthority(t *testing.T) {
	pool := canonicalDatabase(t, "identity_profile")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO user_profiles (
			subject_id, display_name, organization_id, revision, created_at, updated_at
		) VALUES ('auditee-xyz', 'Airline XYZ Auditee', 'airline-xyz', 1, $1, $1)
	`, canonicalNow); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO user_settings (
			subject_id, notification_preferences, locale, timezone, revision, updated_at
		) VALUES ('auditee-xyz', '{"dueDateReminders":true}', 'en', 'Africa/Windhoek', 1, $1)
	`, canonicalNow); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	idCounts := map[string]int{}
	service := identity.NewProfileService(pool, identity.ProfileServiceDependencies{
		Clock: func() time.Time { return canonicalNow },
		IDGenerator: func(prefix string) string {
			idCounts[prefix]++
			return fmt.Sprintf("%s-identity-%03d", prefix, idCounts[prefix])
		},
	})
	actor := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)

	profile, err := service.GetProfile(context.Background(), actor)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.SubjectID != actor.SubjectID || profile.OrganizationID != actor.OrganizationID ||
		profile.Role != identity.RoleAuditee || profile.DisplayName != "Airline XYZ Auditee" || profile.Revision != 1 {
		t.Fatalf("initial profile = %+v", profile)
	}

	updated, err := service.UpdateProfile(context.Background(), actor, identity.UpdateProfileCommand{
		OperationID: "op-profile-001", IdempotencyKey: "idem-profile-001",
		ExpectedRevision: 1, DisplayName: "Airline XYZ Safety Contact",
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Revision != 2 || updated.DisplayName != "Airline XYZ Safety Contact" ||
		updated.Role != identity.RoleAuditee || updated.OrganizationID != "airline-xyz" {
		t.Fatalf("updated profile = %+v", updated)
	}
	replayed, err := service.UpdateProfile(context.Background(), actor, identity.UpdateProfileCommand{
		OperationID: "op-profile-001", IdempotencyKey: "idem-profile-001",
		ExpectedRevision: 1, DisplayName: "Airline XYZ Safety Contact",
	})
	if err != nil || replayed != updated {
		t.Fatalf("profile replay = %+v, err = %v", replayed, err)
	}
	if _, err := service.UpdateProfile(context.Background(), actor, identity.UpdateProfileCommand{
		OperationID: "op-profile-reused-key", IdempotencyKey: "idem-profile-001",
		ExpectedRevision: 2, DisplayName: "Reused key overwrite",
	}); !errors.Is(err, idempotency.ErrOperationIDReuse) {
		t.Fatalf("reused profile idempotency key error = %v", err)
	}
	if _, err := service.UpdateProfile(context.Background(), actor, identity.UpdateProfileCommand{
		OperationID: "op-profile-stale", IdempotencyKey: "idem-profile-stale",
		ExpectedRevision: 1, DisplayName: "Stale overwrite",
	}); !errors.Is(err, identity.ErrConflict) {
		t.Fatalf("stale profile error = %v", err)
	}

	settings, err := service.UpdateSettings(context.Background(), actor, identity.UpdateSettingsCommand{
		OperationID:             "op-settings-001",
		IdempotencyKey:          "idem-settings-001",
		ExpectedRevision:        1,
		NotificationPreferences: json.RawMessage(`{"dueDateReminders":false,"reportReleaseUpdates":true}`),
		Locale:                  "en",
		Timezone:                "Africa/Windhoek",
	})
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if settings.SubjectID != actor.SubjectID || settings.Revision != 2 ||
		!bytes.Equal(settings.NotificationPreferences, json.RawMessage(`{"dueDateReminders": false, "reportReleaseUpdates": true}`)) {
		t.Fatalf("updated settings = %+v", settings)
	}

	var roles []string
	var organizationID string
	if err := pool.QueryRow(context.Background(), `
		SELECT roles, organization_id
		FROM session_references
		WHERE id = 'session-auditee'
	`).Scan(&roles, &organizationID); err != nil {
		t.Fatalf("read session authority: %v", err)
	}
	if fmt.Sprint(roles) != "[auditee]" || organizationID != "airline-xyz" {
		t.Fatalf("profile update changed server session authority: roles=%v organization=%q", roles, organizationID)
	}
	for _, operationID := range []string{"op-profile-001", "op-settings-001"} {
		var linked int
		if err := pool.QueryRow(context.Background(), `
			SELECT COUNT(*)
			FROM command_transaction_links
			WHERE operation_id = $1
		`, operationID).Scan(&linked); err != nil {
			t.Fatalf("count transaction links: %v", err)
		}
		if linked != 1 {
			t.Errorf("operation %s linked envelope count = %d", operationID, linked)
		}
	}
}

func TestIdentitySettingsMigrationBackfillsRetainedSubjects(t *testing.T) {
	pool := createTestDatabase(t, "identity_settings_backfill")
	applyMigrationFilesThrough(t, pool, 9)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organizations (id, legal_name, organization_type, status)
		VALUES ('operator-backfill', 'Backfill Air', 'OPERATOR', 'ACTIVE')
	`); err != nil {
		t.Fatalf("seed retained organization: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name)
		VALUES ('subject-backfill', 'urn:test', 'Backfilled User')
	`); err != nil {
		t.Fatalf("seed retained identity: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO session_references (
			id, subject_id, organization_id, expires_at, last_seen_at,
			absolute_expires_at, roles
		) VALUES (
			'session-backfill', 'subject-backfill', 'operator-backfill',
			$1, $2, $1, ARRAY['auditee']
		)
	`, canonicalNow.Add(8*time.Hour), canonicalNow); err != nil {
		t.Fatalf("seed retained session: %v", err)
	}
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("upgrade retained identity schema: %v", err)
	}
	var displayName string
	var organizationID *string
	var profileRevision, settingsRevision int64
	if err := pool.QueryRow(context.Background(), `
		SELECT profile.display_name, profile.organization_id, profile.revision, settings.revision
		FROM user_profiles profile
		JOIN user_settings settings ON settings.subject_id = profile.subject_id
		WHERE profile.subject_id = 'subject-backfill'
	`).Scan(&displayName, &organizationID, &profileRevision, &settingsRevision); err != nil {
		t.Fatalf("read backfilled identity records: %v", err)
	}
	if displayName != "Backfilled User" || organizationID == nil || *organizationID != "operator-backfill" ||
		profileRevision != 1 || settingsRevision != 1 {
		t.Fatalf("backfilled identity = name %q organization %v profile rev %d settings rev %d",
			displayName, organizationID, profileRevision, settingsRevision)
	}
}

func TestUserLifecycleRequestPersistsJobEnvelopeAndInvalidatesTargetSessions(t *testing.T) {
	pool := canonicalDatabase(t, "user_lifecycle")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name)
		VALUES ('admin-001', 'test', 'Administrator')
	`); err != nil {
		t.Fatalf("seed administrator identity: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO session_references (
			id, subject_id, organization_id, expires_at, last_seen_at,
			absolute_expires_at, roles
		) VALUES (
			'session-admin', 'admin-001', 'caa', $1, $2, $1, ARRAY['admin']
		)
	`, canonicalNow.Add(24*time.Hour), canonicalNow); err != nil {
		t.Fatalf("seed administrator session: %v", err)
	}
	service := administration.NewUserService(pool, administration.UserServiceDependencies{
		Clock: func() time.Time { return canonicalNow },
		IDGenerator: func(prefix string) string {
			return prefix + "-lifecycle-001"
		},
	})
	command := administration.RequestUserLifecycleCommand{
		OperationID: "op-user-suspend-001", IdempotencyKey: "idem-user-suspend-001",
		SubjectID: "auditee-xyz", Action: administration.UserLifecycleSuspend,
		OrganizationID: "airline-xyz", Roles: []identity.Role{identity.RoleAuditee},
	}
	if _, err := service.RequestLifecycle(context.Background(),
		principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector),
		command,
	); !errors.Is(err, administration.ErrForbidden) {
		t.Fatalf("non-admin lifecycle error = %v", err)
	}
	var beforeCount int
	if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM user_lifecycle_requests").Scan(&beforeCount); err != nil {
		t.Fatalf("count unauthorized lifecycle requests: %v", err)
	}
	if beforeCount != 0 {
		t.Fatalf("unauthorized request wrote %d lifecycle records", beforeCount)
	}

	if _, err := service.RequestLifecycle(
		context.Background(),
		principal(
			"admin-001",
			"airline-xyz",
			"session-admin",
			identity.RoleAdmin,
		),
		command,
	); !errors.Is(err, administration.ErrForbidden) {
		t.Fatalf("wrong-organization admin lifecycle error = %v", err)
	}
	if err := pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM user_lifecycle_requests",
	).Scan(&beforeCount); err != nil {
		t.Fatalf("count wrong-organization lifecycle requests: %v", err)
	}
	if beforeCount != 0 {
		t.Fatalf("wrong-organization request wrote %d lifecycle records", beforeCount)
	}

	admin := principal("admin-001", "CAA", "session-admin", identity.RoleAdmin)
	for _, invalid := range []administration.RequestUserLifecycleCommand{
		{
			OperationID:    "op-user-wrong-org-inspector",
			IdempotencyKey: "idem-user-wrong-org-inspector",
			Action:         administration.UserLifecycleProvision,
			OrganizationID: "airline-xyz",
			Email:          "wrong-org-inspector@example.test",
			DisplayName:    "Wrong Organization Inspector",
			Roles:          []identity.Role{identity.RoleInspector},
		},
		{
			OperationID:    "op-user-wrong-org-auditee",
			IdempotencyKey: "idem-user-wrong-org-auditee",
			Action:         administration.UserLifecycleProvision,
			OrganizationID: "CAA",
			Email:          "wrong-org-auditee@example.test",
			DisplayName:    "Wrong Organization Auditee",
			Roles:          []identity.Role{identity.RoleAuditee},
		},
	} {
		if _, err := service.RequestLifecycle(
			context.Background(),
			admin,
			invalid,
		); !errors.Is(err, administration.ErrInvalid) {
			t.Fatalf(
				"wrong role/organization mapping %+v error = %v",
				invalid.Roles,
				err,
			)
		}
	}
	if err := pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM user_lifecycle_requests",
	).Scan(&beforeCount); err != nil {
		t.Fatalf("count invalid role/organization lifecycle requests: %v", err)
	}
	if beforeCount != 0 {
		t.Fatalf(
			"invalid role/organization mapping wrote %d lifecycle records",
			beforeCount,
		)
	}

	requested, err := service.RequestLifecycle(context.Background(), admin, command)
	if err != nil {
		t.Fatalf("request lifecycle: %v", err)
	}
	if requested.Status != administration.UserLifecyclePending ||
		requested.SubjectID != "auditee-xyz" || requested.OutboxMessageID == "" {
		t.Fatalf("lifecycle request = %+v", requested)
	}
	replayed, err := service.RequestLifecycle(context.Background(), admin, command)
	if err != nil || replayed.ID != requested.ID || replayed.Status != requested.Status ||
		replayed.OutboxMessageID != requested.OutboxMessageID ||
		fmt.Sprint(replayed.Roles) != fmt.Sprint(requested.Roles) {
		t.Fatalf("lifecycle replay = %+v, err = %v", replayed, err)
	}
	var responseStatus int
	if err := pool.QueryRow(context.Background(), `
		SELECT response_status
		FROM idempotency_responses
		WHERE scope = $1 AND operation_id = $2
	`, admin.SubjectID+":user_lifecycle", command.OperationID).Scan(
		&responseStatus,
	); err != nil {
		t.Fatalf("read lifecycle idempotency response status: %v", err)
	}
	if responseStatus != 202 {
		t.Fatalf("lifecycle idempotency response status = %d, want 202", responseStatus)
	}
	reusedKey := command
	reusedKey.OperationID = "op-user-suspend-reused-key"
	if _, err := service.RequestLifecycle(context.Background(), admin, reusedKey); !errors.Is(err, idempotency.ErrOperationIDReuse) {
		t.Fatalf("reused lifecycle idempotency key error = %v", err)
	}

	var revokedAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT revoked_at
		FROM session_references
		WHERE id = 'session-auditee'
	`).Scan(&revokedAt); err != nil {
		t.Fatalf("read invalidated target session: %v", err)
	}
	if revokedAt == nil || !revokedAt.Equal(canonicalNow) {
		t.Fatalf("target session revoked_at = %v", revokedAt)
	}
	for _, assertion := range []struct {
		name string
		sql  string
		args []any
	}{
		{"lifecycle", "SELECT COUNT(*) FROM user_lifecycle_requests WHERE id = $1 AND outbox_message_id = $2",
			[]any{requested.ID, requested.OutboxMessageID}},
		{"outbox", "SELECT COUNT(*) FROM outbox_messages WHERE id = $1 AND operation_id = $2",
			[]any{requested.OutboxMessageID, command.OperationID}},
		{"audit", "SELECT COUNT(*) FROM audit_events WHERE operation_id = $1 AND entity_id = $2",
			[]any{command.OperationID, requested.ID}},
		{"change", "SELECT COUNT(*) FROM authorized_sync_changes WHERE operation_id = $1 AND entity_id = $2",
			[]any{command.OperationID, requested.ID}},
		{"link", "SELECT COUNT(*) FROM command_transaction_links WHERE operation_id = $1 AND outbox_message_id = $2",
			[]any{command.OperationID, requested.OutboxMessageID}},
	} {
		var count int
		if err := pool.QueryRow(context.Background(), assertion.sql, assertion.args...).Scan(&count); err != nil {
			t.Fatalf("count %s envelope record: %v", assertion.name, err)
		}
		if count != 1 {
			t.Errorf("%s envelope count = %d", assertion.name, count)
		}
	}
}

func TestUserLifecycleWorkerPersistsProviderSubjectAndDisablesProviderSessions(t *testing.T) {
	pool := canonicalDatabase(t, "user_lifecycle_worker")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name)
		VALUES ('admin-worker-001', 'test', 'Worker Administrator')
	`); err != nil {
		t.Fatalf("seed worker administrator identity: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO session_references (
			id, subject_id, organization_id, expires_at, last_seen_at,
			absolute_expires_at, roles
		) VALUES (
			'session-admin-worker', 'admin-worker-001', 'caa', $1, $2, $1,
			ARRAY['admin']
		)
	`, canonicalNow.Add(24*time.Hour), canonicalNow); err != nil {
		t.Fatalf("seed worker administrator session: %v", err)
	}
	counters := map[string]int{}
	userService := administration.NewUserService(
		pool,
		administration.UserServiceDependencies{
			Clock: func() time.Time { return canonicalNow },
			IDGenerator: func(prefix string) string {
				counters[prefix]++
				return fmt.Sprintf("%s-worker-%03d", prefix, counters[prefix])
			},
		},
	)
	admin := principal(
		"admin-worker-001",
		"CAA",
		"session-admin-worker",
		identity.RoleAdmin,
	)
	provisioned, err := userService.RequestLifecycle(
		context.Background(),
		admin,
		administration.RequestUserLifecycleCommand{
			OperationID:    "op-user-provision-worker-001",
			IdempotencyKey: "idem-user-provision-worker-001",
			Action:         administration.UserLifecycleProvision,
			Email:          "new.auditee@example.test",
			DisplayName:    "New Auditee",
			OrganizationID: "airline-xyz",
			Roles:          []identity.Role{identity.RoleAuditee},
		},
	)
	if err != nil {
		t.Fatalf("request user provisioning: %v", err)
	}
	if provisioned.SubjectID != "" ||
		provisioned.Email != "new.auditee@example.test" ||
		provisioned.DisplayName != "New Auditee" {
		t.Fatalf("pending provisioning request = %+v", provisioned)
	}

	provider := &lifecycleIdentityProvider{
		provisionedSubjectID: "keycloak-subject-001",
	}
	worker := administration.NewUserLifecycleWorker(
		pool,
		provider,
		administration.UserLifecycleWorkerDependencies{
			Clock:         func() time.Time { return canonicalNow },
			WorkerID:      "identity-worker-test",
			LeaseDuration: time.Minute,
			RetryDelay:    time.Minute,
			Issuer:        "https://localhost:8443/identity/realms/aviasurveil360",
		},
	)
	processed, err := worker.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("process provisioning = %t, err = %v", processed, err)
	}
	if provider.provisioned.Email != "new.auditee@example.test" ||
		provider.provisioned.OrganizationID != "airline-xyz" ||
		fmt.Sprint(provider.provisioned.Roles) != "[auditee]" {
		t.Fatalf("provider provisioning = %+v", provider.provisioned)
	}
	if processed, err := worker.ProcessNext(context.Background()); err != nil || processed {
		t.Fatalf("empty worker poll = %t, err = %v", processed, err)
	}

	var lifecycleStatus, lifecycleSubject, lifecycleEmail string
	if err := pool.QueryRow(context.Background(), `
		SELECT status, subject_id, requested_email
		FROM user_lifecycle_requests
		WHERE id = $1
	`, provisioned.ID).Scan(
		&lifecycleStatus,
		&lifecycleSubject,
		&lifecycleEmail,
	); err != nil {
		t.Fatalf("read completed provisioning request: %v", err)
	}
	if lifecycleStatus != "SUCCEEDED" ||
		lifecycleSubject != "keycloak-subject-001" ||
		lifecycleEmail != "new.auditee@example.test" {
		t.Fatalf(
			"completed provisioning = status %q subject %q email %q",
			lifecycleStatus,
			lifecycleSubject,
			lifecycleEmail,
		)
	}
	var issuer, displayName, organizationID string
	if err := pool.QueryRow(context.Background(), `
		SELECT identity.issuer, profile.display_name, profile.organization_id
		FROM identity_references identity
		JOIN user_profiles profile ON profile.subject_id = identity.subject_id
		JOIN user_settings settings ON settings.subject_id = identity.subject_id
		WHERE identity.subject_id = 'keycloak-subject-001'
	`).Scan(&issuer, &displayName, &organizationID); err != nil {
		t.Fatalf("read provisioned identity projection: %v", err)
	}
	if issuer != "https://localhost:8443/identity/realms/aviasurveil360" ||
		displayName != "New Auditee" ||
		organizationID != "airline-xyz" {
		t.Fatalf(
			"identity projection = issuer %q display %q organization %q",
			issuer,
			displayName,
			organizationID,
		)
	}
	var membershipRevision int64
	var membershipState, membershipOrganization, membershipDrift string
	var membershipRoles []string
	if err := pool.QueryRow(context.Background(), `
		SELECT revision, membership_state, organization_id, roles, drift_state
		FROM desired_membership_versions
		WHERE subject_id = 'keycloak-subject-001'
		ORDER BY revision DESC
		LIMIT 1
	`).Scan(
		&membershipRevision,
		&membershipState,
		&membershipOrganization,
		&membershipRoles,
		&membershipDrift,
	); err != nil {
		t.Fatalf("read desired membership: %v", err)
	}
	if membershipRevision != 1 ||
		membershipState != "ACTIVE" ||
		membershipOrganization != "airline-xyz" ||
		fmt.Sprint(membershipRoles) != "[auditee]" ||
		membershipDrift != "IN_SYNC" {
		t.Fatalf(
			"desired membership = revision %d state %q organization %q roles %v drift %q",
			membershipRevision,
			membershipState,
			membershipOrganization,
			membershipRoles,
			membershipDrift,
		)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE desired_membership_versions
		SET membership_state = 'SUSPENDED'
		WHERE subject_id = 'keycloak-subject-001'
		  AND revision = 1
	`); err == nil {
		t.Fatal("desired membership history accepted an in-place rewrite")
	}
	if _, err := pool.Exec(context.Background(), `
		DELETE FROM desired_membership_versions
		WHERE subject_id = 'keycloak-subject-001'
		  AND revision = 1
	`); err == nil {
		t.Fatal("desired membership history accepted deletion")
	}
	var delivered bool
	if err := pool.QueryRow(context.Background(), `
		SELECT delivered_at IS NOT NULL
		FROM outbox_messages
		WHERE id = $1
	`, provisioned.OutboxMessageID).Scan(&delivered); err != nil {
		t.Fatalf("read provisioning outbox: %v", err)
	}
	if !delivered {
		t.Fatal("provisioning outbox was not delivered")
	}
	var successAuditCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM audit_events
		WHERE entity_id = $1
		  AND action = 'USER_PROVISION_SUCCEEDED'
	`, provisioned.ID).Scan(&successAuditCount); err != nil {
		t.Fatalf("count provisioning success audit: %v", err)
	}
	if successAuditCount != 1 {
		t.Fatalf("provisioning success audit count = %d", successAuditCount)
	}

	if _, err := userService.RequestLifecycle(
		context.Background(),
		admin,
		administration.RequestUserLifecycleCommand{
			OperationID:    "op-user-provision-worker-duplicate",
			IdempotencyKey: "idem-user-provision-worker-duplicate",
			Action:         administration.UserLifecycleProvision,
			Email:          "new.auditee@example.test",
			DisplayName:    "Duplicate Auditee",
			OrganizationID: "airline-xyz",
			Roles:          []identity.Role{identity.RoleAuditee},
		},
	); !errors.Is(err, administration.ErrConflict) {
		t.Fatalf("duplicate provisioning email error = %v", err)
	}

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO session_references (
			id, subject_id, organization_id, expires_at, last_seen_at,
			absolute_expires_at, roles
		) VALUES (
			'session-keycloak-subject-001', 'keycloak-subject-001',
			'airline-xyz', $1, $2, $1, ARRAY['auditee']
		)
	`, canonicalNow.Add(24*time.Hour), canonicalNow); err != nil {
		t.Fatalf("seed provisioned user session: %v", err)
	}
	suspended, err := userService.RequestLifecycle(
		context.Background(),
		admin,
		administration.RequestUserLifecycleCommand{
			OperationID:    "op-user-suspend-worker-001",
			IdempotencyKey: "idem-user-suspend-worker-001",
			SubjectID:      "keycloak-subject-001",
			Action:         administration.UserLifecycleSuspend,
			OrganizationID: "airline-xyz",
			Roles:          []identity.Role{identity.RoleAuditee},
		},
	)
	if err != nil {
		t.Fatalf("request provider suspension: %v", err)
	}
	processed, err = worker.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("process suspension = %t, err = %v", processed, err)
	}
	if !slices.Equal(provider.disabledSubjects, []string{"keycloak-subject-001"}) {
		t.Fatalf("provider disabled subjects = %#v", provider.disabledSubjects)
	}
	var revokedAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT revoked_at
		FROM session_references
		WHERE id = 'session-keycloak-subject-001'
	`).Scan(&revokedAt); err != nil {
		t.Fatalf("read suspended session: %v", err)
	}
	if revokedAt == nil || !revokedAt.Equal(canonicalNow) {
		t.Fatalf("suspended session revoked_at = %v", revokedAt)
	}
	var suspendedStatus string
	if err := pool.QueryRow(context.Background(), `
		SELECT status
		FROM user_lifecycle_requests
		WHERE id = $1
	`, suspended.ID).Scan(&suspendedStatus); err != nil {
		t.Fatalf("read suspension request: %v", err)
	}
	if suspendedStatus != "SUCCEEDED" {
		t.Fatalf("suspension status = %q", suspendedStatus)
	}
	var suspendedMembershipID, suspendedMembershipState, suspendedDrift string
	var suspendedMembershipRevision, synchronizedRevision int64
	var observedProviderEnabled bool
	if err := pool.QueryRow(context.Background(), `
		SELECT version.membership_id, version.revision,
		       version.membership_state, sync.desired_revision,
		       sync.observed_provider_enabled, sync.drift_state
		FROM desired_membership_versions version
		JOIN desired_membership_sync sync
		  ON sync.membership_id = version.membership_id
		 AND sync.desired_revision = version.revision
		WHERE version.subject_id = 'keycloak-subject-001'
		ORDER BY version.revision DESC
		LIMIT 1
	`).Scan(
		&suspendedMembershipID,
		&suspendedMembershipRevision,
		&suspendedMembershipState,
		&synchronizedRevision,
		&observedProviderEnabled,
		&suspendedDrift,
	); err != nil {
		t.Fatalf("read suspended desired membership: %v", err)
	}
	if suspendedMembershipID != "membership-"+provisioned.ID ||
		suspendedMembershipRevision != 2 ||
		synchronizedRevision != 2 ||
		suspendedMembershipState != "SUSPENDED" ||
		observedProviderEnabled ||
		suspendedDrift != "IN_SYNC" {
		t.Fatalf(
			"suspended membership = id %q revision %d state %q sync %d enabled %t drift %q",
			suspendedMembershipID,
			suspendedMembershipRevision,
			suspendedMembershipState,
			synchronizedRevision,
			observedProviderEnabled,
			suspendedDrift,
		)
	}

	recovered, err := userService.RequestLifecycle(
		context.Background(),
		admin,
		administration.RequestUserLifecycleCommand{
			OperationID:    "op-user-provision-recovered-001",
			IdempotencyKey: "idem-user-provision-recovered-001",
			Action:         administration.UserLifecycleProvision,
			Email:          "recovered.auditee@example.test",
			DisplayName:    "Recovered Auditee",
			OrganizationID: "airline-xyz",
			Roles:          []identity.Role{identity.RoleAuditee},
		},
	)
	if err != nil {
		t.Fatalf("request recovered provisioning: %v", err)
	}
	provider.provisionError = identity.ErrKeycloakDuplicateEmail
	provider.reconciledSubjectID = "keycloak-subject-recovered-001"
	provider.reconcileMatched = true
	processed, err = worker.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("process reconciled provisioning = %t, err = %v", processed, err)
	}
	if provider.reconciled.Email != "recovered.auditee@example.test" {
		t.Fatalf("reconciled provider user = %+v", provider.reconciled)
	}
	var recoveredStatus, recoveredSubject string
	if err := pool.QueryRow(context.Background(), `
		SELECT status, subject_id
		FROM user_lifecycle_requests
		WHERE id = $1
	`, recovered.ID).Scan(&recoveredStatus, &recoveredSubject); err != nil {
		t.Fatalf("read reconciled provisioning request: %v", err)
	}
	if recoveredStatus != "SUCCEEDED" ||
		recoveredSubject != "keycloak-subject-recovered-001" {
		t.Fatalf(
			"reconciled provisioning = status %q subject %q",
			recoveredStatus,
			recoveredSubject,
		)
	}
}

type lifecycleIdentityProvider struct {
	provisionedSubjectID string
	provisioned          identity.KeycloakUser
	provisionError       error
	reconciledSubjectID  string
	reconcileMatched     bool
	reconciled           identity.KeycloakUser
	disabledSubjects     []string
}

func (provider *lifecycleIdentityProvider) ProvisionUser(
	_ context.Context,
	user identity.KeycloakUser,
) (string, error) {
	provider.provisioned = user
	if provider.provisionError != nil {
		return "", provider.provisionError
	}
	return provider.provisionedSubjectID, nil
}

func (provider *lifecycleIdentityProvider) ReconcileProvisionedUser(
	_ context.Context,
	user identity.KeycloakUser,
) (string, bool, error) {
	provider.reconciled = user
	return provider.reconciledSubjectID, provider.reconcileMatched, nil
}

func (provider *lifecycleIdentityProvider) UpdateUserAuthority(
	_ context.Context,
	_ string,
	_ string,
	_ []identity.Role,
) error {
	return nil
}

func (provider *lifecycleIdentityProvider) DisableUser(
	_ context.Context,
	subjectID string,
) error {
	provider.disabledSubjects = append(provider.disabledSubjects, subjectID)
	return nil
}

func (provider *lifecycleIdentityProvider) EnableUser(
	_ context.Context,
	_ string,
) error {
	return nil
}

type organizationReaderSpy struct {
	listCalls int
	getCalls  int
	listScope string
	records   []organizations.Record
}

func (spy *organizationReaderSpy) ListRegistry(
	_ context.Context,
	organizationScope string,
	_ int32,
) ([]organizations.Record, error) {
	spy.listCalls++
	spy.listScope = organizationScope
	records := append([]organizations.Record(nil), spy.records...)
	if organizationScope == "" {
		return records, nil
	}
	filtered := make([]organizations.Record, 0, 1)
	for _, record := range records {
		if record.ID == organizationScope {
			filtered = append(filtered, record)
		}
	}
	return filtered, nil
}

func (spy *organizationReaderSpy) Get(context.Context, string) (organizations.Record, error) {
	spy.getCalls++
	return spy.records[0], nil
}

func TestOrganizationAuthorizationGuardsBeforeFetchAndDirectIDsDoNotLeakExistence(t *testing.T) {
	spy := &organizationReaderSpy{records: []organizations.Record{{
		ID: "airline-xyz", LegalName: "Airline XYZ", OrganizationType: "OPERATOR",
		Status: "ACTIVE", Revision: 1,
	}}}
	service := organizations.NewService(spy)

	auditeeRecords, err := service.ListRegistry(context.Background(),
		principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee), 20,
	)
	if err != nil || len(auditeeRecords) != 1 || auditeeRecords[0].ID != "airline-xyz" {
		t.Fatalf("Auditee scoped registry = %+v, err = %v", auditeeRecords, err)
	}
	if spy.listCalls != 1 || spy.listScope != "airline-xyz" {
		t.Fatalf("Auditee registry fetch calls=%d scope=%q", spy.listCalls, spy.listScope)
	}
	if _, err := service.Get(context.Background(),
		principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee),
		"airline-other",
	); !errors.Is(err, organizations.ErrNotFound) {
		t.Fatalf("cross-organization direct-ID error = %v", err)
	}
	if spy.getCalls != 0 {
		t.Fatalf("cross-organization direct ID fetched %d times", spy.getCalls)
	}

	record, err := service.Get(context.Background(),
		principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee),
		"airline-xyz",
	)
	if err != nil {
		t.Fatalf("get own organization: %v", err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal organization: %v", err)
	}
	if !strings.Contains(string(raw), "airline-xyz") {
		t.Fatalf("own-organization raw JSON = %s", raw)
	}
	for _, forbidden := range []string{
		"airline-other", "internalCaaNote", "internalRisk", "enforcementDeliberation",
		"inspectorWorkload", "session", "roles",
	} {
		if strings.Contains(string(raw), forbidden) {
			t.Errorf("own-organization raw JSON contains %q: %s", forbidden, raw)
		}
	}
	if spy.getCalls != 1 {
		t.Fatalf("own-organization direct ID fetched %d times", spy.getCalls)
	}

	for _, role := range []identity.Role{
		identity.RoleInspector, identity.RoleLeadInspector, identity.RoleDepartmentManager,
		identity.RoleFinance, identity.RoleGeneralManager, identity.RoleExecutiveDirector,
		identity.RoleAdmin,
	} {
		if _, err := service.ListRegistry(context.Background(),
			principal("caa-user", "caa", "session-caa", role), 20,
		); err != nil {
			t.Errorf("role %s registry list error = %v", role, err)
		}
	}
}

func TestOrganizationLiveListAndDirectIDIsolation(t *testing.T) {
	pool := canonicalDatabase(t, "organization_scope")
	service := organizations.NewPostgresService(pool)
	caa := principal("manager-001", "caa", "session-manager", identity.RoleDepartmentManager)
	records, err := service.ListRegistry(context.Background(), caa, 20)
	if err != nil {
		t.Fatalf("list live organization registry: %v", err)
	}
	if len(records) != 2 || records[0].ID != "airline-xyz" || records[1].ID != "airline-other" {
		t.Fatalf("live organization registry = %+v", records)
	}
	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	auditeeRecords, err := service.ListRegistry(context.Background(), auditee, 20)
	if err != nil || len(auditeeRecords) != 1 || auditeeRecords[0].ID != "airline-xyz" {
		t.Fatalf("Auditee live registry = %+v, err = %v", auditeeRecords, err)
	}
	own, err := service.Get(context.Background(), auditee, "airline-xyz")
	if err != nil || own.ID != "airline-xyz" {
		t.Fatalf("Auditee own live organization = %+v, err = %v", own, err)
	}
	if _, err := service.Get(context.Background(), auditee, "airline-other"); !errors.Is(err, organizations.ErrNotFound) {
		t.Fatalf("Auditee cross-organization live direct-ID error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE organizations
		SET tombstoned_at = $2
		WHERE id = $1
	`, "airline-other", canonicalNow); err != nil {
		t.Fatalf("tombstone organization: %v", err)
	}
	if _, err := service.Get(context.Background(), caa, "airline-other"); !errors.Is(err, organizations.ErrNotFound) {
		t.Fatalf("tombstoned direct-ID error = %v", err)
	}
}

func TestProfileHTTPRouteUsesSessionPrincipalAndClosedAuditeeWireProjection(t *testing.T) {
	pool := createTestDatabase(t, "profile_http")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
		t.Fatalf("reset canonical profile: %v", err)
	}
	profiles := identity.NewProfileService(pool, identity.ProfileServiceDependencies{
		Clock: func() time.Time { return canonicalNow },
		IDGenerator: func(prefix string) string {
			return prefix + "-http-001"
		},
	})
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Application: testService(pool), Profiles: profiles, Clock: func() time.Time { return canonicalNow },
	})
	handler := httpapi.NewCanonicalTestBoundary("task-3-token").Protect(api.Handler())

	get := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
	get.Header.Set(httpapi.CanonicalTestTokenHeader, "task-3-token")
	get.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-AUDITEE-FLY")
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET profile status = %d, body = %s", getResponse.Code, getResponse.Body.String())
	}
	if getResponse.Header().Get("ETag") != `"rev-1"` {
		t.Errorf("GET profile ETag = %q", getResponse.Header().Get("ETag"))
	}
	raw := getResponse.Body.String()
	for _, required := range []string{`"subjectId":"USR-AUDITEE-FLY"`, `"role":"auditee"`, `"organizationId":"ORG-FLY-NAMIBIA"`} {
		if !strings.Contains(raw, required) {
			t.Errorf("GET profile missing %s: %s", required, raw)
		}
	}
	for _, forbidden := range []string{
		"ORG-SKYCARGO", "USR-ADMIN-ADA", "internalCaaNote", "internalRisk",
		"enforcementDeliberation", "roles", "sessionId",
	} {
		if strings.Contains(raw, forbidden) {
			t.Errorf("GET profile raw JSON contains %q: %s", forbidden, raw)
		}
	}

	tamper := httptest.NewRequest(http.MethodPut, "/v1/profile", strings.NewReader(`{
		"operationId":"op-profile-tamper",
		"expectedRevision":1,
		"idempotencyKey":"idem-profile-tamper",
		"displayName":"Tampered",
		"roles":["admin"],
		"organizationId":"ORG-SKYCARGO"
	}`))
	tamper.Header.Set(httpapi.CanonicalTestTokenHeader, "task-3-token")
	tamper.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-AUDITEE-FLY")
	tamperResponse := httptest.NewRecorder()
	handler.ServeHTTP(tamperResponse, tamper)
	if tamperResponse.Code != http.StatusBadRequest {
		t.Fatalf("tampered profile status = %d, body = %s", tamperResponse.Code, tamperResponse.Body.String())
	}

	update := httptest.NewRequest(http.MethodPut, "/v1/profile", strings.NewReader(`{
		"operationId":"op-profile-http-001",
		"expectedRevision":1,
		"idempotencyKey":"idem-profile-http-001",
		"displayName":"Fly Namibia Safety Contact"
	}`))
	update.Header.Set(httpapi.CanonicalTestTokenHeader, "task-3-token")
	update.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-AUDITEE-FLY")
	update.Header.Set("Idempotency-Key", "idem-profile-http-001")
	update.Header.Set("If-Match", `"rev-1"`)
	updateResponse := httptest.NewRecorder()
	handler.ServeHTTP(updateResponse, update)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("PUT profile status = %d, body = %s", updateResponse.Code, updateResponse.Body.String())
	}
	if body := updateResponse.Body.String(); !strings.Contains(body, `"revision":2`) ||
		!strings.Contains(body, `"role":"auditee"`) ||
		!strings.Contains(body, `"organizationId":"ORG-FLY-NAMIBIA"`) {
		t.Fatalf("PUT profile body = %s", body)
	}
	if updateResponse.Header().Get("ETag") != `"rev-2"` {
		t.Errorf("PUT profile ETag = %q", updateResponse.Header().Get("ETag"))
	}

	for _, subjectID := range []string{
		testprofile.CanonicalInspectorSubjectID, "USR-LEAD-CANER", "USR-MANAGER-NORA",
		"USR-FINANCE-LINA", "USR-GM-OMAR", "USR-ED-ZARA", "USR-ADMIN-ADA",
	} {
		request := httptest.NewRequest(http.MethodGet, "/v1/organizations?limit=20", nil)
		request.Header.Set(httpapi.CanonicalTestTokenHeader, "task-3-token")
		request.Header.Set(httpapi.CanonicalTestSubjectHeader, subjectID)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("%s organization registry status = %d, body = %s", subjectID, response.Code, response.Body.String())
		}
		if !regexp.MustCompile(`^"rev-[0-9]+"$`).MatchString(response.Header().Get("ETag")) {
			t.Errorf("%s organization registry ETag = %q", subjectID, response.Header().Get("ETag"))
		}
	}
	auditeeRegistry := httptest.NewRequest(http.MethodGet, "/v1/organizations?limit=20", nil)
	auditeeRegistry.Header.Set(httpapi.CanonicalTestTokenHeader, "task-3-token")
	auditeeRegistry.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-AUDITEE-FLY")
	auditeeRegistryResponse := httptest.NewRecorder()
	handler.ServeHTTP(auditeeRegistryResponse, auditeeRegistry)
	if auditeeRegistryResponse.Code != http.StatusOK ||
		!strings.Contains(auditeeRegistryResponse.Body.String(), `"id":"ORG-FLY-NAMIBIA"`) ||
		strings.Contains(auditeeRegistryResponse.Body.String(), "ORG-SKYCARGO") {
		t.Fatalf("Auditee organization registry status = %d, body = %s",
			auditeeRegistryResponse.Code, auditeeRegistryResponse.Body.String())
	}
}

func TestAdminHTTPCanRequestUserProvisioningWithoutWrongRoleWrites(t *testing.T) {
	pool := createTestDatabase(t, "user_lifecycle_http")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
		t.Fatalf("reset canonical profile: %v", err)
	}
	userService := administration.NewUserService(
		pool,
		administration.UserServiceDependencies{
			Clock: func() time.Time { return canonicalNow },
			IDGenerator: func(prefix string) string {
				return prefix + "-http-001"
			},
		},
	)
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Application: testService(pool), Users: userService,
		Clock: func() time.Time { return canonicalNow },
	})
	handler := httpapi.NewCanonicalTestBoundary(
		"task-3-lifecycle-token",
	).Protect(api.Handler())
	body := `{
		"operationId":"op-user-provision-http-001",
		"idempotencyKey":"idem-user-provision-http-001",
		"action":"PROVISION",
		"email":"new.http.auditee@example.test",
		"displayName":"HTTP Auditee",
		"organizationId":"ORG-FLY-NAMIBIA",
		"roles":["auditee"]
	}`
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/user-lifecycle-requests",
		strings.NewReader(body),
	)
	request.Header.Set(
		httpapi.CanonicalTestTokenHeader,
		"task-3-lifecycle-token",
	)
	request.Header.Set(
		httpapi.CanonicalTestSubjectHeader,
		"USR-ADMIN-ADA",
	)
	request.Header.Set(
		"Idempotency-Key",
		"idem-user-provision-http-001",
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted ||
		!strings.Contains(response.Body.String(), `"status":"PENDING"`) ||
		!strings.Contains(
			response.Body.String(),
			`"email":"new.http.auditee@example.test"`,
		) {
		t.Fatalf(
			"POST user provisioning status=%d body=%s",
			response.Code,
			response.Body.String(),
		)
	}

	denied := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/user-lifecycle-requests",
		strings.NewReader(strings.ReplaceAll(
			body,
			"op-user-provision-http-001",
			"op-user-provision-http-denied",
		)),
	)
	denied.Header.Set(
		httpapi.CanonicalTestTokenHeader,
		"task-3-lifecycle-token",
	)
	denied.Header.Set(
		httpapi.CanonicalTestSubjectHeader,
		"USR-MANAGER-NORA",
	)
	denied.Header.Set(
		"Idempotency-Key",
		"idem-user-provision-http-001",
	)
	deniedResponse := httptest.NewRecorder()
	handler.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusForbidden {
		t.Fatalf(
			"wrong-role provisioning status=%d body=%s",
			deniedResponse.Code,
			deniedResponse.Body.String(),
		)
	}
	var requestCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM user_lifecycle_requests
	`).Scan(&requestCount); err != nil {
		t.Fatalf("count HTTP lifecycle requests: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("HTTP lifecycle request count = %d", requestCount)
	}
}
