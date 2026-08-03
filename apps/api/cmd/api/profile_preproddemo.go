//go:build preproddemo

package main

import (
	"context"
	"fmt"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agacandidatedemo"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
)

// The tagged artifact is deliberately opt-in. Reader-pool injection is wired
// only by the local-preprod compose profile; normal and canonical builds never
// select this profile.
func activeRuntimeProfile(settings config.Settings) (runtimeProfile, error) {
	if settings.Environment != "development" {
		return runtimeProfile{}, fmt.Errorf("preprod AGA demo API requires development local-preprod configuration")
	}
	if settings.CanonicalSeed || settings.CanonicalTestProfile || settings.CanonicalTestToken != "" {
		return runtimeProfile{}, fmt.Errorf("preprod AGA demo API rejects canonical test authority")
	}
	if settings.AGADemoDatabaseURL == "" {
		return runtimeProfile{}, fmt.Errorf("preprod AGA demo API requires a separate reader database URL")
	}
	return runtimeProfile{skipMigrations: true, agaDemoOnly: true, clock: time.Now, agaDemoService: func(ctx context.Context, settings config.Settings) (*aga.Service, func(), error) {
		pool, err := database.Open(ctx, settings.AGADemoDatabaseURL)
		if err != nil {
			return nil, nil, err
		}
		reader, err := aga.NewPostgresReader(pool)
		if err != nil {
			pool.Close()
			return nil, nil, err
		}
		return aga.NewService(reader), pool.Close, nil
	}}, nil
}
