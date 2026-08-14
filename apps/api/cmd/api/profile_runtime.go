package main

import (
	"context"
	"net/http"
	"time"

	"github.com/aviason/aviaSurveil/internal/administration"
	"github.com/aviason/aviaSurveil/internal/platform/config"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
)

type runtimeProfile struct {
	applyMigrations           bool
	clock                     func() time.Time
	idGenerator               func(string) string
	findingReferenceGenerator func() string
	bootstrap                 func(
		context.Context,
		*database.Pool,
		config.Settings,
		time.Time,
	) error
	seed              func(context.Context, *database.Pool, time.Time) error
	directoryProvider administration.AccessDirectoryProvider
	protect           func(
		config.Settings,
		http.Handler,
		*database.Pool,
		objectstore.Store,
		[]string,
	) (http.Handler, http.Handler, error)
}
