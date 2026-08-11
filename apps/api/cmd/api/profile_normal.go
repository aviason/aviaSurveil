//go:build !canonicaltest

package main

import (
	"fmt"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
)

func activeRuntimeProfile(settings config.Settings) (runtimeProfile, error) {
	if settings.CanonicalSeed ||
		settings.CanonicalTestProfile ||
		settings.CanonicalTestToken != "" {
		return runtimeProfile{}, fmt.Errorf(
			"normal API artifact rejects canonical seed, reset, and header authority",
		)
	}
	// The disposable local-preprod stack runs migrations in its owner-only
	// one-shot service.  The long-lived normal API uses the provisioned
	// runtime role and must not attempt DDL/forward repair on startup.
	return runtimeProfile{
		skipMigrations: settings.Environment == "local-preprod",
		clock:          time.Now,
	}, nil
}
