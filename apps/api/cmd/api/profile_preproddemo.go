//go:build preproddemo

package main

import (
	"context"
	"fmt"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agacandidatedemo"
	workspace "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agademoworkspace"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
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
	if settings.AGADemoWorkspaceReaderURL == "" || settings.AGADemoWorkspaceCommandURL == "" {
		return runtimeProfile{}, fmt.Errorf("preprod AGA demo API requires separate workspace reader and command database URLs")
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
	}, agaWorkspaceService: func(ctx context.Context, settings config.Settings) (*workspace.Service, func(), error) {
		readerPool, err := database.Open(ctx, settings.AGADemoWorkspaceReaderURL)
		if err != nil {
			return nil, nil, err
		}
		readerStore, err := preprod.NewPostgresReader(readerPool)
		if err != nil {
			readerPool.Close()
			return nil, nil, err
		}
		overlayPool, err := database.Open(ctx, settings.AGADemoDatabaseURL)
		if err != nil {
			readerPool.Close()
			return nil, nil, err
		}
		questionBodies, err := workspace.NewPostgresQuestionBodyResolver(overlayPool)
		if err != nil {
			overlayPool.Close()
			readerPool.Close()
			return nil, nil, err
		}
		commandPool, err := database.Open(ctx, settings.AGADemoWorkspaceCommandURL)
		if err != nil {
			overlayPool.Close()
			readerPool.Close()
			return nil, nil, err
		}
		commandStore, err := preprod.NewPostgresCommandStore(commandPool)
		if err != nil {
			overlayPool.Close()
			readerPool.Close()
			commandPool.Close()
			return nil, nil, err
		}
		resolver, err := workspace.NewPostgresBindingResolver(readerPool)
		if err != nil {
			overlayPool.Close()
			readerPool.Close()
			commandPool.Close()
			return nil, nil, err
		}
		service := workspace.NewService(workspace.ServiceConfig{
			ReaderStore:          readerStore,
			CommandStore:         commandStore,
			Resolver:             resolver,
			QuestionBodies:       questionBodies,
			QuestionTextSearch:   questionBodies,
			RecommendationScopes: workspace.NewPostgresRecommendationScopeResolver(readerPool),
			SimulationSetup:      workspace.NewPostgresSimulationSetupResolver(readerPool),
			LifecycleBindings:    workspace.NewFixtureLifecycleBindingResolver(),
		})
		return service, func() { overlayPool.Close(); readerPool.Close(); commandPool.Close() }, nil
	}}, nil
}
