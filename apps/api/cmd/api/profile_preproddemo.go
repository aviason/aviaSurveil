//go:build preproddemo

package main

import (
	"fmt"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
)

// The tagged artifact is deliberately opt-in. It is retained only as a
// deletion-gated source boundary; the canonical cutover no longer wires donor
// reader or command services into any runtime.
func activeRuntimeProfile(settings config.Settings) (runtimeProfile, error) {
	if settings.Environment != "development" {
		return runtimeProfile{}, fmt.Errorf("preprod AGA demo API requires development local-preprod configuration")
	}
	if settings.CanonicalSeed || settings.CanonicalTestProfile || settings.CanonicalTestToken != "" {
		return runtimeProfile{}, fmt.Errorf("preprod AGA demo API rejects canonical test authority")
	}
	// The old AGA donor profile remains only as a tagged source boundary for
	// deletion-gated provenance. It deliberately has no donor service wiring;
	// main.go turns agaDemoOnly into a fail-closed readiness state.
	return runtimeProfile{skipMigrations: true, agaDemoOnly: true, clock: time.Now}, nil
}
