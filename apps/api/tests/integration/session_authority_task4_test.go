package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/administration"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/session"
)

type task4AuthorityObserver struct {
	observation identity.AuthorityObservation
	err         error
	calls       int
}

func (observer *task4AuthorityObserver) ObserveUserAuthority(
	_ context.Context,
	subjectID string,
) (identity.AuthorityObservation, error) {
	observer.calls++
	observation := observer.observation
	if observation.SubjectID == "" {
		observation.SubjectID = subjectID
	}
	return observation, observer.err
}

func TestTask4SessionBindsExactDesiredProviderAndTokenAuthority(t *testing.T) {
	pool := canonicalDatabase(t, "task4_session_authority")
	now := canonicalNow
	subjectID := "task4-session-inspector"
	seedTask4Membership(
		t,
		pool,
		subjectID,
		"membership-task4-inspector",
		1,
		"ACTIVE",
		"CAA",
		[]string{"inspector"},
		now,
	)
	observer := &task4AuthorityObserver{
		observation: identity.AuthorityObservation{
			SubjectID:      subjectID,
			Enabled:        true,
			OrganizationID: "CAA",
			Roles:          []identity.Role{identity.RoleInspector},
			ObservedAt:     now,
		},
	}
	manager := newTask4SessionManager(t, pool, &now, observer)
	created, err := manager.Create(context.Background(), session.CreateInput{
		SubjectID:      subjectID,
		Issuer:         "https://identity.example/realms/avia",
		DisplayName:    "Task 4 Inspector",
		Email:          "task4.inspector@example.test",
		OrganizationID: "CAA",
		Roles:          []identity.Role{identity.RoleInspector},
		ProviderTokens: identity.ProviderTokens{
			AccessToken: "task4-access-token",
			IDToken:     "task4-id-token",
			Expiry:      now.Add(time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("create exact-authority session: %v", err)
	}
	var membershipID string
	var membershipRevision int64
	if err := pool.QueryRow(context.Background(), `
		SELECT membership_id, membership_revision
		FROM session_references
		WHERE id = $1
	`, created.ID).Scan(&membershipID, &membershipRevision); err != nil {
		t.Fatalf("read persisted session authority: %v", err)
	}
	if membershipID != "membership-task4-inspector" ||
		membershipRevision != 1 {
		t.Fatalf(
			"session membership authority = %q/%d",
			membershipID,
			membershipRevision,
		)
	}
	if _, err := manager.Authenticate(
		context.Background(),
		created.Token,
	); err != nil {
		t.Fatalf("authenticate exact-authority session: %v", err)
	}

	seedTask4Membership(
		t,
		pool,
		subjectID,
		"membership-task4-inspector",
		2,
		"ACTIVE",
		"CAA",
		[]string{"leadInspector"},
		now.Add(time.Second),
	)
	if _, err := manager.Authenticate(
		context.Background(),
		created.Token,
	); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("old membership revision session error = %v", err)
	}
	var revokedAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT revoked_at
		FROM session_references
		WHERE id = $1
	`, created.ID).Scan(&revokedAt); err != nil {
		t.Fatalf("read stale session revocation: %v", err)
	}
	if revokedAt == nil {
		t.Fatal("old membership revision did not revoke the session")
	}
}

func TestTask4SessionRejectsProviderDriftUnavailableAndNoMembershipAuthorities(
	t *testing.T,
) {
	pool := canonicalDatabase(t, "task4_session_rejections")
	now := canonicalNow
	subjectID := "task4-session-auditee"
	seedTask4Membership(
		t,
		pool,
		subjectID,
		"membership-task4-auditee",
		1,
		"ACTIVE",
		"airline-xyz",
		[]string{"auditee"},
		now,
	)
	exact := identity.AuthorityObservation{
		SubjectID:      subjectID,
		Enabled:        true,
		OrganizationID: "airline-xyz",
		Roles:          []identity.Role{identity.RoleAuditee},
		ObservedAt:     now,
	}
	for _, testCase := range []struct {
		name          string
		subjectID     string
		observation   identity.AuthorityObservation
		observerError error
	}{
		{
			name:      "provider role drift",
			subjectID: subjectID,
			observation: identity.AuthorityObservation{
				SubjectID:      subjectID,
				Enabled:        true,
				OrganizationID: "airline-xyz",
				Roles:          []identity.Role{identity.RoleInspector},
				ObservedAt:     now,
			},
		},
		{
			name:          "provider unavailable",
			subjectID:     subjectID,
			observation:   exact,
			observerError: identity.ErrKeycloakUnavailable,
		},
		{
			name:      "provider disabled",
			subjectID: subjectID,
			observation: identity.AuthorityObservation{
				SubjectID:      subjectID,
				Enabled:        false,
				OrganizationID: "airline-xyz",
				Roles:          []identity.Role{identity.RoleAuditee},
				ObservedAt:     now,
			},
		},
		{
			name:      "provider locked",
			subjectID: subjectID,
			observation: identity.AuthorityObservation{
				SubjectID:      subjectID,
				Enabled:        true,
				Locked:         true,
				OrganizationID: "airline-xyz",
				Roles:          []identity.Role{identity.RoleAuditee},
				ObservedAt:     now,
			},
		},
		{
			name:      "provider required action incomplete",
			subjectID: subjectID,
			observation: identity.AuthorityObservation{
				SubjectID:      subjectID,
				Enabled:        true,
				OrganizationID: "airline-xyz",
				Roles:          []identity.Role{identity.RoleAuditee},
				RequiredActions: []string{
					"VERIFY_EMAIL",
				},
				ObservedAt: now,
			},
		},
		{
			name:      "bootstrap identity without membership",
			subjectID: "local-bootstrap-admin",
			observation: identity.AuthorityObservation{
				SubjectID:      "local-bootstrap-admin",
				Enabled:        true,
				OrganizationID: "CAA",
				Roles:          []identity.Role{identity.RoleAdmin},
				ObservedAt:     now,
			},
		},
		{
			name:      "break glass without membership",
			subjectID: "local-break-glass",
			observation: identity.AuthorityObservation{
				SubjectID:      "local-break-glass",
				Enabled:        true,
				OrganizationID: "CAA",
				Roles:          []identity.Role{identity.RoleAdmin},
				ObservedAt:     now,
			},
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			observer := &task4AuthorityObserver{
				observation: testCase.observation,
				err:         testCase.observerError,
			}
			manager := newTask4SessionManager(t, pool, &now, observer)
			organizationID := testCase.observation.OrganizationID
			roles := testCase.observation.Roles
			if _, err := manager.Create(
				context.Background(),
				session.CreateInput{
					SubjectID:      testCase.subjectID,
					Issuer:         "https://identity.example/realms/avia",
					DisplayName:    "Rejected Task 4 Identity",
					Email:          "rejected.task4@example.test",
					OrganizationID: organizationID,
					Roles:          roles,
				},
			); !errors.Is(err, session.ErrUnauthenticated) {
				t.Fatalf("invalid session authority error = %v", err)
			}
		})
	}
}

func TestTask4SessionHeartbeatAndProviderLossMeetFrozenDenialDeadline(
	t *testing.T,
) {
	pool := canonicalDatabase(t, "task4_session_observation_deadline")
	now := canonicalNow
	subjectID := "task4-session-heartbeat"
	seedTask4Membership(
		t,
		pool,
		subjectID,
		"membership-task4-heartbeat",
		1,
		"ACTIVE",
		"CAA",
		[]string{"inspector"},
		now,
	)
	observer := &task4AuthorityObserver{
		observation: identity.AuthorityObservation{
			SubjectID:      subjectID,
			Enabled:        true,
			OrganizationID: "CAA",
			Roles:          []identity.Role{identity.RoleInspector},
		},
	}
	manager := newTask4SessionManager(t, pool, &now, observer)
	created, err := manager.Create(context.Background(), session.CreateInput{
		SubjectID:      subjectID,
		Issuer:         "https://identity.example/realms/avia",
		DisplayName:    "Task 4 Heartbeat Inspector",
		Email:          "task4.heartbeat@example.test",
		OrganizationID: "CAA",
		Roles:          []identity.Role{identity.RoleInspector},
	})
	if err != nil {
		t.Fatalf("create heartbeat-bound session: %v", err)
	}
	if observer.calls != 1 {
		t.Fatalf("new-login provider observations = %d", observer.calls)
	}

	now = canonicalNow.Add(29 * time.Second)
	if _, err := manager.Authenticate(
		context.Background(),
		created.Token,
	); err != nil {
		t.Fatalf("healthy authority before heartbeat: %v", err)
	}
	if observer.calls != 1 {
		t.Fatalf("premature provider heartbeat calls = %d", observer.calls)
	}

	now = canonicalNow.Add(31 * time.Second)
	if _, err := manager.Authenticate(
		context.Background(),
		created.Token,
	); err != nil {
		t.Fatalf("healthy provider heartbeat: %v", err)
	}
	if observer.calls != 2 {
		t.Fatalf("provider heartbeat calls = %d", observer.calls)
	}
	var observedAt time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT authority_observed_at
		FROM session_references
		WHERE id = $1
	`, created.ID).Scan(&observedAt); err != nil {
		t.Fatalf("read refreshed session observation: %v", err)
	}
	if !observedAt.Equal(now) {
		t.Fatalf("session authority observation = %s, want %s", observedAt, now)
	}

	observer.err = identity.ErrKeycloakUnavailable
	now = canonicalNow.Add(62 * time.Second)
	if _, err := manager.Authenticate(
		context.Background(),
		created.Token,
	); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("provider-loss session error = %v", err)
	}
	var authorityState string
	var revokedAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT authority_state, revoked_at
		FROM session_references
		WHERE id = $1
	`, created.ID).Scan(&authorityState, &revokedAt); err != nil {
		t.Fatalf("read provider-loss session state: %v", err)
	}
	if authorityState != "REVOCATION_PENDING" || revokedAt != nil {
		t.Fatalf(
			"provider-loss session = state %q revoked %v",
			authorityState,
			revokedAt,
		)
	}

	now = canonicalNow.Add(151 * time.Second)
	if _, err := manager.Authenticate(
		context.Background(),
		created.Token,
	); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("provider-loss deadline session error = %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT authority_state, revoked_at
		FROM session_references
		WHERE id = $1
	`, created.ID).Scan(&authorityState, &revokedAt); err != nil {
		t.Fatalf("read provider-loss deadline state: %v", err)
	}
	if authorityState != "DENIED_STALE_AUTHORITY" ||
		revokedAt == nil ||
		!revokedAt.Equal(now) {
		t.Fatalf(
			"provider-loss deadline session = state %q revoked %v",
			authorityState,
			revokedAt,
		)
	}
}

func TestTask4AuthorityMutationForcesFreshProviderObservation(t *testing.T) {
	pool := canonicalDatabase(t, "task4_session_mutation_freshness")
	now := canonicalNow
	subjectID := "task4-mutation-inspector"
	seedTask4Membership(
		t,
		pool,
		subjectID,
		"membership-task4-mutation-inspector",
		1,
		"ACTIVE",
		"CAA",
		[]string{"inspector"},
		now,
	)
	observer := &task4AuthorityObserver{
		observation: identity.AuthorityObservation{
			SubjectID:      subjectID,
			Enabled:        true,
			OrganizationID: "CAA",
			Roles:          []identity.Role{identity.RoleInspector},
		},
	}
	manager := newTask4SessionManager(t, pool, &now, observer)
	created, err := manager.Create(context.Background(), session.CreateInput{
		SubjectID:      subjectID,
		Issuer:         "https://identity.example/realms/avia",
		DisplayName:    "Task 4 Mutation Inspector",
		OrganizationID: "CAA",
		Roles:          []identity.Role{identity.RoleInspector},
	})
	if err != nil {
		t.Fatalf("create mutation-authority session: %v", err)
	}
	observer.observation.Roles = []identity.Role{identity.RoleLeadInspector}
	if _, err := manager.Authenticate(
		session.RequireFreshAuthorityObservation(context.Background()),
		created.Token,
	); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("stale-authority mutation session error = %v", err)
	}
	if observer.calls != 2 {
		t.Fatalf("mutation provider observation calls = %d", observer.calls)
	}
}

func TestTask4ConcurrentMembershipRevisionCannotCreateStaleSession(
	t *testing.T,
) {
	pool := canonicalDatabase(t, "task4_session_creation_revision_race")
	now := canonicalNow.Add(2 * time.Second)
	subjectID := "task4-concurrent-inspector"
	membershipID := "membership-task4-concurrent"
	seedTask4Membership(
		t,
		pool,
		subjectID,
		membershipID,
		1,
		"ACTIVE",
		"CAA",
		[]string{"inspector"},
		canonicalNow,
	)
	seedTask4Membership(
		t,
		pool,
		subjectID,
		membershipID,
		2,
		"ACTIVE",
		"CAA",
		[]string{"inspector"},
		canonicalNow.Add(time.Second),
	)
	observer := &task4AuthorityObserver{
		observation: identity.AuthorityObservation{
			SubjectID:      subjectID,
			Enabled:        true,
			OrganizationID: "CAA",
			Roles:          []identity.Role{identity.RoleInspector},
		},
	}
	manager := newTask4SessionManager(t, pool, &now, observer)

	blocker, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin identity-row blocker: %v", err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	if err := blocker.QueryRow(context.Background(), `
		SELECT subject_id
		FROM identity_references
		WHERE subject_id = $1
		FOR UPDATE
	`, subjectID).Scan(&subjectID); err != nil {
		t.Fatalf("lock identity row: %v", err)
	}

	type createResult struct {
		session session.BrowserSession
		err     error
	}
	result := make(chan createResult, 1)
	go func() {
		created, createErr := manager.Create(
			context.Background(),
			session.CreateInput{
				SubjectID:      subjectID,
				Issuer:         "https://identity.example/realms/avia",
				DisplayName:    "Task 4 Concurrent Inspector",
				OrganizationID: "CAA",
				Roles:          []identity.Role{identity.RoleInspector},
			},
		)
		result <- createResult{session: created, err: createErr}
	}()

	blocked := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := pool.QueryRow(context.Background(), `
			SELECT EXISTS (
				SELECT 1
				FROM pg_stat_activity
				WHERE datname = current_database()
				  AND wait_event_type = 'Lock'
				  AND query LIKE '%UPDATE identity_references%'
			)
		`).Scan(&blocked); err != nil {
			t.Fatalf("observe blocked session creation: %v", err)
		}
		if blocked {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("session creation did not reach the controlled revision race")
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE desired_membership_sync
		SET desired_revision = 1
		WHERE membership_id = $1
		  AND desired_revision = 2
	`, membershipID); err != nil {
		t.Fatalf("change current membership revision: %v", err)
	}
	if err := blocker.Rollback(context.Background()); err != nil {
		t.Fatalf("release identity-row blocker: %v", err)
	}

	created := <-result
	if !errors.Is(created.err, session.ErrUnauthenticated) {
		t.Fatalf(
			"concurrent stale session creation = id %q, error %v",
			created.session.ID,
			created.err,
		)
	}
	var staleSessionCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM session_references
		WHERE membership_id = $1
		  AND membership_revision = 2
	`, membershipID).Scan(&staleSessionCount); err != nil {
		t.Fatalf("count concurrent stale sessions: %v", err)
	}
	if staleSessionCount != 0 {
		t.Fatalf("concurrent stale session count = %d", staleSessionCount)
	}
}

func TestTask4FirstLoginReconcilesInvitationBeforeCreatingSession(
	t *testing.T,
) {
	pool := canonicalDatabase(t, "task4_session_activation")
	now := canonicalNow
	subjectID := "task4-invited-inspector"
	membershipID := "membership-task4-invited"
	seedTask4Membership(
		t,
		pool,
		subjectID,
		membershipID,
		1,
		"INVITED",
		"CAA",
		[]string{"inspector"},
		now,
	)
	requestID := fmt.Sprintf("task4-membership-%s-1", subjectID)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_action_facts (
			id, request_id, fact_sequence, membership_id, subject_id,
			action_kind, state, delivery_attempt, expires_at,
			provider_acknowledged_at, reason, created_at
		) VALUES (
			'task4-invitation-delivered', $1, 1, $2, $3,
			'INVITATION', 'DELIVERY_ACCEPTED', 1, $4, $5,
			'Task 4 first-login fixture.', $5
		)
	`, requestID, membershipID, subjectID, now.Add(24*time.Hour),
		now); err != nil {
		t.Fatalf("seed Task 4 invitation fact: %v", err)
	}
	observer := &task4AuthorityObserver{
		observation: identity.AuthorityObservation{
			SubjectID:      subjectID,
			Enabled:        true,
			OrganizationID: "CAA",
			Roles:          []identity.Role{identity.RoleInspector},
		},
	}
	activation := administration.NewUserService(
		pool,
		administration.UserServiceDependencies{
			Clock: func() time.Time { return now },
			IDGenerator: func(prefix string) string {
				return prefix + "-task4-activation"
			},
		},
	)
	manager := newTask4SessionManagerWithActivation(
		t,
		pool,
		&now,
		observer,
		activation,
	)
	created, err := manager.Create(context.Background(), session.CreateInput{
		SubjectID:      subjectID,
		Issuer:         "https://identity.example/realms/avia",
		DisplayName:    "Task 4 Invited Inspector",
		Email:          "task4.invited@example.test",
		OrganizationID: "CAA",
		Roles:          []identity.Role{identity.RoleInspector},
	})
	if err != nil {
		t.Fatalf("create first-login session: %v", err)
	}
	var state string
	var revision, sessionRevision int64
	if err := pool.QueryRow(context.Background(), `
		SELECT version.membership_state, version.revision,
		       session.membership_revision
		FROM desired_membership_versions version
		JOIN session_references session
		  ON session.membership_id = version.membership_id
		 AND session.membership_revision = version.revision
		WHERE version.membership_id = $1
		  AND session.id = $2
	`, membershipID, created.ID).Scan(
		&state,
		&revision,
		&sessionRevision,
	); err != nil {
		t.Fatalf("read activated session authority: %v", err)
	}
	if state != "ACTIVE" || revision != 2 || sessionRevision != 2 {
		t.Fatalf(
			"activated session authority = state %q revision %d/%d",
			state,
			revision,
			sessionRevision,
		)
	}
}

func newTask4SessionManager(
	t *testing.T,
	pool *database.Pool,
	now *time.Time,
	observer identity.AuthorityObserver,
) *session.Manager {
	return newTask4SessionManagerWithActivation(
		t,
		pool,
		now,
		observer,
		nil,
	)
}

func newTask4SessionManagerWithActivation(
	t *testing.T,
	pool *database.Pool,
	now *time.Time,
	observer identity.AuthorityObserver,
	activation session.ActivationReconciler,
) *session.Manager {
	t.Helper()
	sequence := byte(0)
	manager, err := session.NewManager(
		pool,
		[]byte("0123456789abcdef0123456789abcdef"),
		session.ManagerDependencies{
			Clock: func() time.Time { return *now },
			IDGenerator: func(prefix string) string {
				sequence++
				return fmt.Sprintf("%s-task4-%d", prefix, sequence)
			},
			RandomBytes: func(size int) ([]byte, error) {
				sequence++
				return bytes.Repeat([]byte{sequence}, size), nil
			},
			AuthorityObserver:    observer,
			ActivationReconciler: activation,
		},
	)
	if err != nil {
		t.Fatalf("new Task 4 session manager: %v", err)
	}
	return manager
}

func seedTask4Membership(
	t *testing.T,
	pool *database.Pool,
	subjectID,
	membershipID string,
	revision int64,
	state,
	organizationID string,
	roles []string,
	observedAt time.Time,
) {
	t.Helper()
	ctx := context.Background()
	if revision == 1 {
		if _, err := pool.Exec(ctx, `
			INSERT INTO identity_references (
				subject_id, issuer, display_name, email, created_at
			) VALUES (
				'task4-admin', 'urn:aviasurveil360:task4',
				'Task 4 Administrator', 'task4.admin@example.test', $1
			)
			ON CONFLICT (subject_id) DO NOTHING
		`, observedAt); err != nil {
			t.Fatalf("seed Task 4 administrator identity: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO identity_references (
				subject_id, issuer, display_name, email, created_at
			) VALUES ($1, 'https://identity.example/realms/avia',
				'Task 4 Identity', lower($1) || '@example.test', $2)
		`, subjectID, observedAt); err != nil {
			t.Fatalf("seed Task 4 identity: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO user_profiles (
				subject_id, display_name, organization_id,
				revision, created_at, updated_at
			) VALUES ($1, 'Task 4 Identity', $2, 1, $3, $3)
		`, subjectID, organizationID, observedAt); err != nil {
			t.Fatalf("seed Task 4 profile: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO user_settings (
				subject_id, notification_preferences, locale, timezone,
				revision, updated_at
			) VALUES ($1, '{}'::jsonb, 'en', 'UTC', 1, $2)
		`, subjectID, observedAt); err != nil {
			t.Fatalf("seed Task 4 settings: %v", err)
		}
	}
	requestID := fmt.Sprintf("task4-membership-%s-%d", subjectID, revision)
	if _, err := pool.Exec(ctx, `
		INSERT INTO user_lifecycle_requests (
			id, subject_id, requested_action, requested_roles,
			requested_organization_id, status, idempotency_key,
			expected_membership_revision, reason, requested_by_subject_id,
			resulting_membership_revision
		) VALUES (
			$1, $2, 'UPDATE_ROLES', $3, $4, 'SUCCEEDED', $1,
			$5, 'Task 4 authority fixture.', 'task4-admin', $6
		)
	`, requestID, subjectID, roles, organizationID, revision-1, revision); err != nil {
		t.Fatalf("seed Task 4 lifecycle request: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO desired_membership_versions (
			membership_id, subject_id, revision, membership_state,
			organization_id, roles, requested_by_subject_id, reason,
			source_request_id, requested_at, effective_at,
			observed_provider_enabled, observed_organization_id,
			observed_roles, observed_at, drift_state
		) VALUES (
			$1, $2, $3, $4, $5, $6, 'task4-admin',
			'Task 4 authority fixture.', $7, $8, $8,
			true, $5, $6, $8, 'IN_SYNC'
		)
	`, membershipID, subjectID, revision, state, organizationID, roles,
		requestID, observedAt); err != nil {
		t.Fatalf("seed Task 4 membership version: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO desired_membership_sync (
			membership_id, subject_id, desired_revision,
			observed_provider_enabled, observed_organization_id,
			observed_roles, observed_at, drift_state
		) VALUES ($1, $2, $3, true, $4, $5, $6, 'IN_SYNC')
		ON CONFLICT (membership_id) DO UPDATE
		SET desired_revision = EXCLUDED.desired_revision,
		    observed_provider_enabled = EXCLUDED.observed_provider_enabled,
		    observed_organization_id = EXCLUDED.observed_organization_id,
		    observed_roles = EXCLUDED.observed_roles,
		    observed_at = EXCLUDED.observed_at,
		    drift_state = EXCLUDED.drift_state
	`, membershipID, subjectID, revision, organizationID, roles,
		observedAt); err != nil {
		t.Fatalf("seed Task 4 membership sync: %v", err)
	}
}
