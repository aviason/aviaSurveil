package main

import (
	"net/url"
	"testing"
)

func TestMigrationDatabaseURLUsesTheOwningRoleForDefaultPrivileges(t *testing.T) {
	raw := "postgres://avia_master:master-secret@example.test:5432/postgres?sslmode=require"
	parsed, err := url.Parse(migrationDatabaseURL(raw, "migration-secret"))
	if err != nil {
		t.Fatalf("parse migration database URL: %v", err)
	}
	if got := parsed.User.Username(); got != "surveil_migration" {
		t.Fatalf("migration database URL user = %q, want surveil_migration", got)
	}
	if got := parsed.Path; got != "/surveil" {
		t.Fatalf("migration database URL path = %q, want /surveil", got)
	}
	if got := parsed.Query().Get("sslmode"); got != "require" {
		t.Fatalf("migration database URL sslmode = %q, want require", got)
	}
}
