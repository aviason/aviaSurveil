//go:build preproddemo

package main

import (
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"testing"
)

func TestPreprodDemoProfileRejectsNonDevelopment(t *testing.T) {
	if _, err := activeRuntimeProfile(config.Settings{Environment: "production", AGADemoDatabaseURL: "postgres://reader"}); err == nil {
		t.Fatal("expected tagged profile to reject production")
	}
}

func TestPreprodDemoProfileRequiresSeparateReaderURL(t *testing.T) {
	if _, err := activeRuntimeProfile(config.Settings{Environment: "development"}); err == nil {
		t.Fatal("expected missing reader URL rejection")
	}
}

func TestPreprodDemoProfileIsMigrationFreeAndAGAOnly(t *testing.T) {
	profile, err := activeRuntimeProfile(config.Settings{
		Environment:        "development",
		AGADemoDatabaseURL: "postgres://reader",
	})
	if err != nil {
		t.Fatalf("activeRuntimeProfile() error = %v", err)
	}
	if !profile.skipMigrations || !profile.agaDemoOnly || profile.agaDemoService == nil {
		t.Fatalf("tagged profile is not the isolated AGA runtime: %+v", profile)
	}
}
