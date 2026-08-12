package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/session"
)

func TestBrowserSessionHashesOpaqueCredentialsEncryptsProviderTokensAndEnforcesPolicy(t *testing.T) {
	pool := canonicalDatabase(t, "browser_session")
	now := canonicalNow
	randomCall := byte(0)
	idCounts := map[string]int{}
	seedTask4Membership(
		t,
		pool,
		"oidc-inspector",
		"membership-oidc-inspector",
		1,
		"ACTIVE",
		"CAA",
		[]string{"inspector"},
		now,
	)
	observer := &task4AuthorityObserver{
		observation: identity.AuthorityObservation{
			SubjectID:      "oidc-inspector",
			Enabled:        true,
			OrganizationID: "CAA",
			Roles:          []identity.Role{identity.RoleInspector},
			ObservedAt:     now,
		},
	}
	manager, err := session.NewManager(pool, []byte("0123456789abcdef0123456789abcdef"), session.ManagerDependencies{
		Clock: func() time.Time { return now },
		IDGenerator: func(prefix string) string {
			idCounts[prefix]++
			return fmt.Sprintf("%s-auth-%03d", prefix, idCounts[prefix])
		},
		RandomBytes: func(size int) ([]byte, error) {
			randomCall++
			return bytes.Repeat([]byte{randomCall}, size), nil
		},
		AuthorityObserver: observer,
	})
	if err != nil {
		t.Fatalf("new session manager: %v", err)
	}

	created, err := manager.Create(context.Background(), session.CreateInput{
		SubjectID: "oidc-inspector", Issuer: "https://identity.example/realms/avia", DisplayName: "OIDC Inspector",
		OrganizationID: "CAA", Roles: []identity.Role{identity.RoleInspector}, ProviderSessionID: "provider-session-001",
		ProviderTokens: identity.ProviderTokens{AccessToken: "plain-access-secret", RefreshToken: "plain-refresh-secret", IDToken: "plain-id-secret", Expiry: now.Add(time.Hour)},
	})
	if err != nil {
		t.Fatalf("create browser session: %v", err)
	}
	if created.Token == "" || created.CSRFToken == "" || created.Token == created.CSRFToken {
		t.Fatalf("opaque browser credentials = %+v", created)
	}
	if created.ExpiresAt != now.Add(30*time.Minute) || created.AbsoluteExpiresAt != now.Add(8*time.Hour) {
		t.Fatalf("session expiry = idle %s absolute %s", created.ExpiresAt, created.AbsoluteExpiresAt)
	}

	var tokenHash, csrfHash string
	var providerCiphertext []byte
	if err := pool.QueryRow(context.Background(), `
		SELECT session_token_hash, csrf_token_hash, provider_tokens_ciphertext
		FROM session_references WHERE id = $1
	`, created.ID).Scan(&tokenHash, &csrfHash, &providerCiphertext); err != nil {
		t.Fatalf("read persisted session: %v", err)
	}
	if tokenHash == created.Token || csrfHash == created.CSRFToken || bytes.Contains(providerCiphertext, []byte("plain-access-secret")) || bytes.Contains(providerCiphertext, []byte("plain-refresh-secret")) {
		t.Fatal("raw browser/provider secret was persisted")
	}
	var profileRevision, settingsRevision int64
	if err := pool.QueryRow(context.Background(), `
		SELECT profile.revision, settings.revision
		FROM user_profiles profile
		JOIN user_settings settings ON settings.subject_id = profile.subject_id
		WHERE profile.subject_id = $1
	`, created.Principal.SubjectID).Scan(&profileRevision, &settingsRevision); err != nil {
		t.Fatalf("read bootstrapped profile/settings: %v", err)
	}
	if profileRevision != 1 || settingsRevision != 1 {
		t.Fatalf("bootstrapped revisions = profile %d settings %d", profileRevision, settingsRevision)
	}

	principal, err := manager.Authenticate(context.Background(), created.Token)
	if err != nil {
		t.Fatalf("authenticate session: %v", err)
	}
	if principal.SubjectID != "oidc-inspector" || principal.OrganizationID != "CAA" || principal.SessionID != created.ID || !principal.HasRole(identity.RoleInspector) {
		t.Fatalf("authenticated principal = %+v", principal)
	}
	if err := manager.ValidateCSRF(context.Background(), principal.SessionID, created.CSRFToken); err != nil {
		t.Fatalf("validate CSRF: %v", err)
	}
	if err := manager.ValidateCSRF(context.Background(), principal.SessionID, "wrong-csrf"); !errors.Is(err, session.ErrCSRF) {
		t.Fatalf("wrong CSRF error = %v", err)
	}

	now = canonicalNow.Add(29 * time.Minute)
	if _, err := manager.Authenticate(context.Background(), created.Token); err != nil {
		t.Fatalf("refresh idle session: %v", err)
	}
	now = canonicalNow.Add(58 * time.Minute)
	if _, err := manager.Authenticate(context.Background(), created.Token); err != nil {
		t.Fatalf("rolling idle session: %v", err)
	}
	now = canonicalNow.Add(8 * time.Hour)
	if _, err := manager.Authenticate(context.Background(), created.Token); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("absolute-expired session error = %v", err)
	}

	now = canonicalNow
	second, err := manager.Create(context.Background(), session.CreateInput{
		SubjectID: "oidc-inspector", Issuer: "https://identity.example/realms/avia", DisplayName: "OIDC Inspector",
		OrganizationID: "CAA", Roles: []identity.Role{identity.RoleInspector},
		ProviderTokens: identity.ProviderTokens{IDToken: "server-held-revocation-id-token"},
	})
	if err != nil {
		t.Fatalf("create revocable session: %v", err)
	}
	providerLogoutTicket, err := manager.Revoke(context.Background(), second.ID)
	if err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if providerLogoutTicket == "" || providerLogoutTicket == "server-held-revocation-id-token" || strings.Contains(providerLogoutTicket, "server-held-revocation-id-token") {
		t.Fatalf("provider logout ticket was not opaque")
	}
	providerIDToken, err := manager.RedeemProviderLogout(context.Background(), providerLogoutTicket)
	if err != nil {
		t.Fatalf("redeem provider logout: %v", err)
	}
	if providerIDToken != "server-held-revocation-id-token" {
		t.Fatalf("revoke provider ID token = %q", providerIDToken)
	}
	now = canonicalNow.Add(providerLogoutExpiryForTest)
	if _, err := manager.RedeemProviderLogout(context.Background(), providerLogoutTicket); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("expired provider logout ticket error = %v", err)
	}
	if _, err := manager.Authenticate(context.Background(), second.Token); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("revoked session error = %v", err)
	}
	var revokeAuditCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM audit_events
		WHERE action = 'SESSION_REVOKED'
		  AND entity_type = 'SESSION'
		  AND entity_id = $1
		  AND actor_subject_id = $2
	`, second.ID, second.Principal.SubjectID).Scan(&revokeAuditCount); err != nil {
		t.Fatalf("count session revoke audit: %v", err)
	}
	if revokeAuditCount != 1 {
		t.Fatalf("session revoke audit count = %d", revokeAuditCount)
	}

	if _, err := pool.Exec(context.Background(), `
		UPDATE user_profiles
		SET tombstoned_at = $2
		WHERE subject_id = $1
	`, second.Principal.SubjectID, now); err != nil {
		t.Fatalf("tombstone user profile: %v", err)
	}
	if _, err := manager.Create(context.Background(), session.CreateInput{
		SubjectID: "oidc-inspector", Issuer: "https://identity.example/realms/avia", DisplayName: "OIDC Inspector",
		OrganizationID: "CAA", Roles: []identity.Role{identity.RoleInspector},
	}); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("tombstoned profile session-create error = %v", err)
	}
}

const providerLogoutExpiryForTest = 3 * time.Minute

func TestSessionManagerInjectsOnlyCurrentEffectiveDepartmentAssignments(t *testing.T) {
	pool := canonicalDatabase(t, "session_department_assignments")
	now := canonicalNow
	seedTask4Membership(t, pool, "manager-session", "membership-manager-session", 1, "ACTIVE", "CAA", []string{"manager"}, now)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('manager-session', 'test', 'Manager Session') ON CONFLICT DO NOTHING;
		INSERT INTO caa_department_memberships (id, root_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from)
		VALUES ('manager-session-assignment', 'manager-session-assignment', 'manager-session', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01')
	`); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	manager, err := session.NewManager(pool, []byte("0123456789abcdef0123456789abcdef"), session.ManagerDependencies{Clock: func() time.Time { return now }, AuthorityObserver: &task4AuthorityObserver{observation: identity.AuthorityObservation{SubjectID: "manager-session", Enabled: true, OrganizationID: "CAA", Roles: []identity.Role{identity.RoleDepartmentManager}, ObservedAt: now}}})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	created, err := manager.Create(context.Background(), session.CreateInput{SubjectID: "manager-session", Issuer: "https://identity.example/realms/avia", DisplayName: "Manager", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleDepartmentManager}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	principal, err := manager.Authenticate(context.Background(), created.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if !identity.CanTechnicallyReviewUnit(principal, "FLIGHT_OPERATIONS_INSPECTORATE", "FLIGHT_OPERATIONS_INSPECTORATE") {
		t.Fatalf("assignment missing: %+v", principal.DepartmentAssignments)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO caa_organizational_unit_status_facts (
			id, root_id, supersedes_id, organizational_unit_id, status, effective_from
		) VALUES (
			'session-inactive-unit', 'seed-unit-status-FLIGHT_OPERATIONS_INSPECTORATE',
			'seed-unit-status-FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'INACTIVE', '2026-01-01'
		)
	`); err != nil {
		t.Fatalf("append inactive unit status: %v", err)
	}
	principal, err = manager.Authenticate(context.Background(), created.Token)
	if err != nil {
		t.Fatalf("authenticate inactive assignment: %v", err)
	}
	if identity.CanTechnicallyReview(principal, "FLIGHT_OPERATIONS_INSPECTORATE") || len(principal.DepartmentAssignments) != 0 {
		t.Fatalf("inactive assignment retained authority: %+v", principal.DepartmentAssignments)
	}
}

func TestOIDCLoginStateIsOneTimeHashedAndRejectsUnsafeReturnTargets(t *testing.T) {
	pool := canonicalDatabase(t, "oidc_login_state")
	manager, err := session.NewManager(pool, []byte("0123456789abcdef0123456789abcdef"), session.ManagerDependencies{
		Clock:       func() time.Time { return canonicalNow },
		IDGenerator: func(prefix string) string { return prefix + "-login-001" },
		RandomBytes: func(size int) ([]byte, error) { return bytes.Repeat([]byte{7}, size), nil },
	})
	if err != nil {
		t.Fatalf("new session manager: %v", err)
	}
	request, err := manager.NewLoginState(context.Background(), "https://attacker.example/phish")
	if err != nil {
		t.Fatalf("new login state: %v", err)
	}
	if request.ReturnTo != "/" || request.State == "" || request.Nonce == "" || request.PKCEChallenge == "" {
		t.Fatalf("login request = %+v", request)
	}
	var storedState string
	if err := pool.QueryRow(context.Background(), "SELECT state_hash FROM oidc_login_states").Scan(&storedState); err != nil {
		t.Fatalf("read login state: %v", err)
	}
	if storedState == request.State {
		t.Fatal("raw OIDC state was persisted")
	}
	consumed, err := manager.ConsumeLoginState(context.Background(), request.State)
	if err != nil {
		t.Fatalf("consume login state: %v", err)
	}
	if consumed.Nonce != request.Nonce || consumed.PKCEVerifier == "" || consumed.ReturnTo != "/" {
		t.Fatalf("consumed state = %+v", consumed)
	}
	if _, err := manager.ConsumeLoginState(context.Background(), request.State); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("replayed OIDC state error = %v", err)
	}
}
