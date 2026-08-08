package main

import (
	"strings"
	"testing"
)

func TestNormalRuntimeRoleContractIsNonOwnerAndExplicit(t *testing.T) {
	t.Parallel()

	for _, required := range []string{
		"CREATE ROLE preprod_normal_api LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS",
		"REVOKE aviasurveil360_preprod_loader FROM preprod_normal_api",
		"REVOKE ALL PRIVILEGES ON DATABASE aviasurveil360_local_preprod FROM PUBLIC",
		"REVOKE ALL ON SCHEMA public FROM PUBLIC",
		"REVOKE ALL ON SCHEMA public FROM preprod_normal_api",
		"GRANT CONNECT ON DATABASE aviasurveil360_local_preprod TO preprod_normal_api",
		"GRANT USAGE ON SCHEMA public TO preprod_normal_api",
		"GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO preprod_normal_api",
		"GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO preprod_normal_api",
		"REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC",
		"GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO preprod_normal_api",
		"ALTER DEFAULT PRIVILEGES FOR ROLE aviasurveil360_preprod_loader IN SCHEMA public",
	} {
		if !strings.Contains(normalRuntimeRoleDDL, required) {
			t.Fatalf("normal runtime role contract is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"GRANT CREATE",
		"GRANT ALL",
		"WITH SUPERUSER",
		"WITH CREATEROLE",
		"WITH CREATEDB",
	} {
		if strings.Contains(normalRuntimeRoleDDL, forbidden) {
			t.Fatalf("normal runtime role contract grants forbidden capability %q", forbidden)
		}
	}
}

func TestParseCommandAllowsOnlyProvisionOrLeastPrivilegeVerification(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		arguments []string
		want      string
		wantError bool
	}{
		{arguments: nil, want: "provision"},
		{arguments: []string{"verify-least-privilege"}, want: "verify-least-privilege"},
		{arguments: []string{"provision"}, wantError: true},
		{arguments: []string{"verify-least-privilege", "unexpected"}, wantError: true},
	} {
		got, err := parseCommand(testCase.arguments)
		if (err != nil) != testCase.wantError {
			t.Fatalf("parseCommand(%q) error = %v, wantError %v", testCase.arguments, err, testCase.wantError)
		}
		if got != testCase.want {
			t.Fatalf("parseCommand(%q) = %q, want %q", testCase.arguments, got, testCase.want)
		}
	}
}

func TestDatabaseURLUsesDedicatedNormalRuntimeRole(t *testing.T) {
	t.Parallel()

	value := databaseURL(normalRuntimeRole, "private password")
	if !strings.Contains(value, "preprod_normal_api:private%20password@preprod-postgres:5432/") {
		t.Fatalf("database URL does not use the dedicated normal runtime role: %q", value)
	}
	if strings.Contains(value, bootstrapOwnerRole+":") {
		t.Fatalf("database URL unexpectedly uses bootstrap owner credentials: %q", value)
	}
}
