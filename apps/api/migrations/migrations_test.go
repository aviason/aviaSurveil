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
	if LatestVersion != 47 {
		t.Fatalf("latest API migration = %d, want 47", LatestVersion)
	}
}
