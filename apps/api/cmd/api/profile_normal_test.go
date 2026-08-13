//go:build !canonicaltest

package main

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/platform/config"
)

func TestNormalRuntimeProfileRejectsCanonicalSeedAndHeaderAuthority(t *testing.T) {
	t.Parallel()

	for _, settings := range []config.Settings{
		{CanonicalSeed: true},
		{CanonicalTestProfile: true},
		{CanonicalSeed: true, CanonicalTestProfile: true},
	} {
		if _, err := activeRuntimeProfile(settings); err == nil {
			t.Fatalf("normal runtime profile accepted canonical settings: %+v", settings)
		}
	}
}

func TestNormalRuntimeProfileHasNoSeedResetOrTestAuthorityHooks(t *testing.T) {
	t.Parallel()

	profile, err := activeRuntimeProfile(config.Settings{})
	if err != nil {
		t.Fatalf("activeRuntimeProfile() error = %v", err)
	}
	if profile.bootstrap != nil ||
		profile.seed != nil ||
		profile.protect != nil ||
		profile.skipMigrations ||
		profile.clock == nil {
		t.Fatalf("normal runtime profile exposes a test hook: %+v", profile)
	}
}
