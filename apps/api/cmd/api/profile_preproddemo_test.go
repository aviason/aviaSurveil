//go:build preproddemo

package main

import (
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"testing"
)

func TestPreprodDemoProfileRejectsNonDevelopment(t *testing.T) {
	if _, err := activeRuntimeProfile(config.Settings{Environment: "production"}); err == nil {
		t.Fatal("expected tagged profile to reject production")
	}
}

func TestPreprodDemoProfileIsMigrationFreeAndAGAOnly(t *testing.T) {
	profile, err := activeRuntimeProfile(config.Settings{Environment: "development"})
	if err != nil {
		t.Fatalf("activeRuntimeProfile() error = %v", err)
	}
	if !profile.skipMigrations || !profile.agaDemoOnly {
		t.Fatalf("tagged profile is not the isolated AGA runtime: %+v", profile)
	}
}
