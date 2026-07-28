//go:build canonicaltest

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/session"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
)

func activeRuntimeProfile(settings config.Settings) (runtimeProfile, error) {
	if settings.Environment != "test" ||
		!settings.CanonicalSeed ||
		!settings.CanonicalTestProfile {
		return runtimeProfile{}, fmt.Errorf(
			"canonical-test API artifact requires the explicit canonical test profile",
		)
	}
	generator := testprofile.NewGenerator()
	return runtimeProfile{
		clock:                     testprofile.CanonicalScenarioTime,
		idGenerator:               generator.Next,
		findingReferenceGenerator: generator.FindingReference,
		bootstrap:                 session.BootstrapTestProfile,
		seed: func(
			ctx context.Context,
			pool *database.Pool,
			_ time.Time,
		) error {
			return testprofile.Reset(
				ctx,
				pool,
				testprofile.CanonicalScenarioTime(),
			)
		},
		protect: func(
			settings config.Settings,
			api http.Handler,
			pool *database.Pool,
			objects objectstore.Store,
			buckets []string,
		) (http.Handler, http.Handler, error) {
			resetter, ok := objects.(objectstore.TestResetter)
			if !ok {
				return nil, nil, fmt.Errorf(
					"canonical-test object store does not expose reset authority",
				)
			}
			boundary := httpapi.NewCanonicalTestBoundary(
				settings.CanonicalTestToken,
			)
			admin := httpapi.NewCanonicalTestAdmin(
				pool,
				resetter,
				buckets,
				generator,
				testprofile.CanonicalScenarioTime,
			)
			return boundary.Protect(api), boundary.Admin(admin), nil
		},
	}, nil
}
