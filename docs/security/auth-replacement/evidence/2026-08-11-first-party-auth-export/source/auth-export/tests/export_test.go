package tests

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestForwardOnlyMigrationOrder(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrations = append(migrations, entry.Name())
		}
	}
	if len(migrations) != 1 {
		t.Fatalf("expected one export migration, got %v", migrations)
	}
	sorted := append([]string(nil), migrations...)
	sort.Strings(sorted)
	if strings.Join(migrations, "\n") != strings.Join(sorted, "\n") {
		t.Fatalf("migration files are not lexically ordered: %v", migrations)
	}
	data, err := os.ReadFile(filepath.Join("..", "migrations", migrations[0]))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{
		"app.users",
		"app.auth_accounts",
		"app.auth_sessions",
		"app.auth_refresh_tokens",
		"app.auth_security_events",
		"app.staff_permissions",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("migration does not define %s", required)
		}
	}
	if strings.Contains(strings.ToUpper(text), "DROP TABLE") {
		t.Error("export migration contains a rollback/destructive DROP TABLE")
	}
}

func TestFixtureUsesReservedDomain(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "fixtures", "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "example.invalid") {
		t.Fatal("fixture must use the reserved example.invalid domain")
	}
}
