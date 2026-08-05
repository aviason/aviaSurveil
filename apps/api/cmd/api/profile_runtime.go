package main

import (
	"context"
	"net/http"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agacandidatedemo"
	workspace "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agademoworkspace"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
)

type runtimeProfile struct {
	skipMigrations            bool
	agaDemoOnly               bool
	clock                     func() time.Time
	idGenerator               func(string) string
	findingReferenceGenerator func() string
	bootstrap                 func(
		context.Context,
		*database.Pool,
		config.Settings,
		time.Time,
	) error
	seed    func(context.Context, *database.Pool, time.Time) error
	protect func(
		config.Settings,
		http.Handler,
		*database.Pool,
		objectstore.Store,
		[]string,
	) (http.Handler, http.Handler, error)
	agaDemoService      func(context.Context, config.Settings) (*aga.Service, func(), error)
	agaWorkspaceService func(context.Context, config.Settings) (*workspace.Service, func(), error)
}
