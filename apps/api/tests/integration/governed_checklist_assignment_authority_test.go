//go:build canonicaltest

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernedChecklistAssignmentAuthority(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "000028_governed_checklist_intake_and_authoring.up.sql")
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read version-28 migration: %v", err)
	}
	sql := strings.ToLower(string(sqlBytes))
	for _, token := range []string{"reviewed_source_set", "source_owner", "checklist_reviewer", "effective_from", "effective_to"} {
		if !strings.Contains(sql, token) {
			t.Errorf("assignment migration is missing %s", token)
		}
	}
	for _, forbidden := range []string{"grant functional", "admin self", "insert into governed_checklist_functional_assignments select"} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("assignment migration contains prohibited grant path %q", forbidden)
		}
	}
}
