//go:build !canonicaltest

package main

import (
	"fmt"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/config"
)

func activeRuntimeProfile(settings config.Settings) (runtimeProfile, error) {
	if settings.CanonicalSeed ||
		settings.CanonicalTestProfile ||
		settings.CanonicalTestToken != "" {
		return runtimeProfile{}, fmt.Errorf(
			"normal API artifact rejects canonical seed, reset, and header authority",
		)
	}
	return runtimeProfile{
		clock: time.Now,
	}, nil
}
