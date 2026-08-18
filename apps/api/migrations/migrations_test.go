package migrations

import (
	"strings"
	"testing"
)

func TestOIDCLoginStateSecurityMigrationContainsBrowserBoundAdmission(t *testing.T) {
	migration, err := migrationFiles.ReadFile("000046_oidc_login_state_security.up.sql")
	if err != nil {
		t.Fatalf("read login-state security migration: %v", err)
	}
	for _, required := range []string{
		"DELETE FROM oidc_login_states",
		"browser_binding_hash text",
		"cannot migrate live OIDC login state without browser binding",
		"oidc_login_states_browser_binding_hash_check",
		"oidc_login_admission",
		"request_count integer NOT NULL",
		"oidc_login_admission_cleanup_idx",
	} {
		if !strings.Contains(string(migration), required) {
			t.Errorf("login-state security migration missing %q", required)
		}
	}
	if LatestVersion != 56 {
		t.Fatalf("latest API migration = %d, want 56", LatestVersion)
	}
	focusMigration, err := migrationFiles.ReadFile("000049_canonical_audit_type_focus_policy.up.sql")
	if err != nil {
		t.Fatalf("read audit-type focus migration: %v", err)
	}
	for _, required := range []string{
		"canonical_audit_type_matches_question_focus",
		"RAMP_INSPECTION",
		"CABIN_INSPECTION",
		"PERIODIC_SURVEILLANCE",
		"IMMUTABLE",
	} {
		if !strings.Contains(strings.ToUpper(string(focusMigration)), strings.ToUpper(required)) {
			t.Errorf("audit-type focus migration missing %q", required)
		}
	}
	focusPolicyV2, err := migrationFiles.ReadFile("000056_canonical_audit_type_focus_policy_v2.up.sql")
	if err != nil {
		t.Fatalf("read audit-type focus v2 migration: %v", err)
	}
	for _, required := range []string{"CHANGE_APPROVAL", "INITIAL_CERTIFICATION", "SPECIAL_PURPOSE", "ELSE false"} {
		if !strings.Contains(strings.ToUpper(string(focusPolicyV2)), strings.ToUpper(required)) {
			t.Errorf("audit-type focus v2 migration missing %q", required)
		}
	}
}
