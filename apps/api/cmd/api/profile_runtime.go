package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
)

func configuredDataFeedWriter() (*datafeed.Writer, error) {
	tenantID := strings.TrimSpace(os.Getenv("AVIA_DATA_FEED_TENANT_ID"))
	keyFile := strings.TrimSpace(os.Getenv("AVIA_DATA_FEED_PAYLOAD_KEY_FILE"))
	keyRef := strings.TrimSpace(os.Getenv("AVIA_DATA_FEED_PAYLOAD_KEY_REF"))
	if tenantID == "" && keyFile == "" && keyRef == "" {
		return nil, nil
	}
	if tenantID == "" || keyFile == "" {
		return nil, fmt.Errorf("AVIA_DATA_FEED_TENANT_ID and AVIA_DATA_FEED_PAYLOAD_KEY_FILE must be configured together")
	}
	key, err := datafeed.LoadPayloadKeyFile(keyFile)
	if err != nil {
		return nil, err
	}
	return datafeed.NewWriter(datafeed.WriterConfig{TenantID: tenantID, PayloadKey: key, PayloadKeyRef: keyRef})
}

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
}
