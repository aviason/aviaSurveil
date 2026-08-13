//go:build canonicaltest

package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/session"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
)

// canonicalTestDirectoryProvider is the provider-neutral directory seam for
// the header-authenticated canonical HTTP fixture. The normal API profile
// always receives its directory from the configured first-party provider
// admin endpoint; this in-process fixture exists only because the canonical
// HTTP contract deliberately runs without an external identity service.
type canonicalTestDirectoryProvider struct{}

func (canonicalTestDirectoryProvider) ListDirectory(
	_ context.Context,
	query identity.ProviderDirectoryQuery,
) (identity.ProviderDirectoryPage, error) {
	if query.First < 0 || query.Limit < 1 || query.Limit > 25 {
		return identity.ProviderDirectoryPage{}, identity.ErrProviderPermanent
	}
	user := identity.ProviderDirectoryUser{
		SubjectID:       "USR-INSPECTOR-DAVID",
		Email:           "david.inspector@example.test",
		DisplayName:     "David Inspector",
		OrganizationID:  "CAA",
		Enabled:         true,
		TOTPConfigured:  false,
		RequiredActions: []string{},
		Roles:           []identity.Role{identity.RoleInspector},
		State:           "ACTIVE",
	}
	search := strings.ToLower(strings.TrimSpace(query.Search))
	if search != "" && !strings.Contains(strings.ToLower(user.SubjectID+" "+user.Email+" "+user.DisplayName), search) {
		return identity.ProviderDirectoryPage{ProviderCalls: 1}, nil
	}
	if query.First > 0 {
		return identity.ProviderDirectoryPage{ProviderCalls: 1}, nil
	}
	return identity.ProviderDirectoryPage{
		Users:         []identity.ProviderDirectoryUser{user},
		ProviderCalls: 1,
	}, nil
}

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
		directoryProvider:         canonicalTestDirectoryProvider{},
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
