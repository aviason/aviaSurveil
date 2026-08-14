package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aviason/aviaSurveil/internal/administration"
	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/assistant"
	"github.com/aviason/aviaSurveil/internal/documents"
	"github.com/aviason/aviaSurveil/internal/evidence"
	"github.com/aviason/aviaSurveil/internal/httpapi"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/inspections/attachments"
	"github.com/aviason/aviaSurveil/internal/planning"
	"github.com/aviason/aviaSurveil/internal/platform/config"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	platformhealth "github.com/aviason/aviaSurveil/internal/platform/health"
	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
	"github.com/aviason/aviaSurveil/internal/platform/session"
	"github.com/aviason/aviaSurveil/internal/platform/telemetry"
	"github.com/aviason/aviaSurveil/internal/risk"
	fieldsync "github.com/aviason/aviaSurveil/internal/sync"
	"github.com/aviason/aviaSurveil/migrations"
)

type unavailableReadiness struct {
	err error
}

type objectStoreReadiness struct {
	store objectstore.Store
}

func (readiness objectStoreReadiness) Ready(ctx context.Context) error {
	return readiness.store.Check(ctx)
}

type combinedReadiness []httpapi.ReadinessProbe

const (
	oidcLoginAdmissionCleanupBatch  = 1000
	oidcLoginAdmissionCleanupPasses = 2
)

func (readiness combinedReadiness) Ready(ctx context.Context) error {
	for _, probe := range readiness {
		if err := probe.Ready(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (readiness unavailableReadiness) Ready(context.Context) error {
	return readiness.err
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	settings, err := config.LoadAPI(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	profile, err := activeRuntimeProfile(settings)
	if err != nil {
		return fmt.Errorf("select API artifact profile: %w", err)
	}
	var directoryProvider administration.AccessDirectoryProvider
	var authorityObserver identity.AuthorityObserver
	if settings.FirstPartyAdminURL != "" {
		firstPartyClient, firstPartyErr := identity.NewFirstPartyAdminClient(identity.FirstPartyAdminConfig{
			BaseURL: settings.FirstPartyAdminURL, SecretFile: settings.FirstPartyAdminSecretFile,
		})
		if firstPartyErr != nil {
			return fmt.Errorf("configure AviaAuth identity provider: %w", firstPartyErr)
		}
		directoryProvider = firstPartyClient
		authorityObserver = firstPartyClient
	}
	if directoryProvider == nil {
		directoryProvider = profile.directoryProvider
	}
	telemetryRuntime, err := telemetry.NewRuntime(ctx, telemetry.Config{
		ServiceName:      "api",
		ServiceVersion:   "candidate",
		Environment:      settings.Environment,
		OTLPHTTPEndpoint: settings.OTLPHTTPEndpoint,
	})
	if err != nil {
		return fmt.Errorf("configure telemetry: %w", err)
	}
	slog.SetDefault(telemetry.NewJSONLogger(nil, "api"))
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := telemetryRuntime.Shutdown(shutdownContext); shutdownErr != nil {
			slog.Warn("telemetry shutdown incomplete", "errorClass", telemetry.ErrorClass(shutdownErr))
		}
	}()

	var probe httpapi.ReadinessProbe = unavailableReadiness{err: errors.New("PostgreSQL initialization has not completed")}
	var databaseHealth httpapi.ReadinessProbe = unavailableReadiness{err: errors.New("PostgreSQL initialization has not completed")}
	var objectStoreHealth httpapi.ReadinessProbe
	var scannerHealth httpapi.ReadinessProbe
	var authentication http.Handler
	var authenticatedAPI http.Handler
	var testAdministration http.Handler
	pool, databaseErr := database.OpenWithTracer(
		ctx,
		settings.DatabaseURL,
		telemetryRuntime.PostgresTracer("application"),
	)
	if databaseErr == nil {
		var migrationErr error
		if profile.applyMigrations {
			migrationErr = migrations.Apply(ctx, pool)
		}
		if migrationErr == nil {
			var bootstrapErr error
			if profile.bootstrap != nil {
				bootstrapErr = profile.bootstrap(
					ctx,
					pool,
					settings,
					profile.clock(),
				)
			}
			if bootstrapErr == nil {
				runtimeClock := profile.clock
				userLifecycleService := administration.NewUserService(
					pool,
					administration.UserServiceDependencies{
						Clock:       runtimeClock,
						IDGenerator: profile.idGenerator,
					},
				)
				databaseProbe := database.Readiness{Pool: pool, RequiredMigrationVersion: migrations.LatestVersion}
				databaseHealth = databaseProbe
				probe = databaseProbe
				exerciseProfileEnabled := canonicalAGAExerciseProfileEnabled(settings.Environment, os.LookupEnv)
				if !exerciseProfileEnabled && canonicalTestExerciseProfileEnabled(settings) {
					// The canonical HTTP contract harness is an explicit disposable
					// test profile. It uses the same namespaced exercise content as
					// local-preprod, while the strict production/default guard above
					// remains unchanged.
					exerciseProfileEnabled = true
				}
				if exerciseProfileEnabled && !canonicalTestExerciseProfileEnabled(settings) {
					if namespaceErr := verifyCanonicalAGAExerciseDatabase(ctx, pool); namespaceErr != nil {
						exerciseProfileEnabled = false
						probe = unavailableReadiness{err: namespaceErr}
						slog.Error("canonical AGA exercise namespace does not match the connected database; readiness will fail closed", "error", namespaceErr)
					}
				}
				var authBoundary *httpapi.AuthBoundary
				if settings.OIDCIssuerURL != "" {
					var sessionManager *session.Manager
					var managerErr error
					if authorityObserver == nil {
						managerErr = errors.New(
							"identity provider authority observer is required for OIDC sessions",
						)
					} else {
						sessionManager, managerErr = session.NewManager(
							pool,
							settings.SessionEncryptionKey,
							session.ManagerDependencies{
								AuthorityObserver:                 authorityObserver,
								ActivationReconciler:              userLifecycleService,
								RequireProviderAuthorityRevisions: true,
								LoginBindingKey:                   []byte(settings.OIDCClientSecret),
							},
						)
					}
					if managerErr != nil {
						probe = unavailableReadiness{err: managerErr}
						slog.Error("session manager unavailable; readiness will fail closed", "error", managerErr)
					} else {
						go maintainOIDCLoginState(ctx, sessionManager)
						provider, providerErr := identity.NewRemoteOIDCProvider(ctx, identity.RemoteOIDCConfig{
							IssuerURL: settings.OIDCIssuerURL, DiscoveryURL: settings.OIDCDiscoveryURL,
							ClientID:     settings.OIDCClientID,
							ClientSecret: settings.OIDCClientSecret, RedirectURL: settings.OIDCRedirectURL,
							RequireAuthorityClaims: true,
							DisableRefreshToken:    true,
						})
						if providerErr != nil {
							probe = unavailableReadiness{err: providerErr}
							slog.Error("OIDC provider unavailable; readiness will fail closed", "error", providerErr)
						} else {
							authBoundary = httpapi.NewAuthBoundaryWithCookieSecure(provider, sessionManager, settings.CookieSecure)
							authBoundary.RequireFreshAuthorityOnEveryRequest()
							authentication = authBoundary.Handler()
						}
					}
				}
				if settings.ObjectStoreMode != "" || settings.ObjectStoreEndpoint != "" {
					var objects objectstore.Store
					var objectErr error
					switch settings.ObjectStoreMode {
					case "aws-s3":
						objects, objectErr = objectstore.NewAWSStore(objectstore.AWSConfig{
							Region: settings.ObjectStoreRegion, HealthBucket: settings.QuarantineBucket, Clock: runtimeClock,
						})
					case "", "minio":
						var localObjects *objectstore.MinIOStore
						localObjects, objectErr = objectstore.NewMinIOStore(objectstore.MinIOConfig{
							Endpoint: settings.ObjectStoreEndpoint, PublicEndpoint: settings.ObjectStorePublicEndpoint,
							AccessKey: settings.ObjectStoreAccessKey, SecretKey: settings.ObjectStoreSecretKey,
							UseTLS: settings.ObjectStoreTLS, PublicUseTLS: settings.ObjectStorePublicTLS,
							Region: settings.ObjectStoreRegion, AllowServerManagedCORS: settings.AllowServerManagedCORS,
							Clock: runtimeClock,
						})
						objects = localObjects
						if objectErr == nil && settings.Environment != "production" && settings.Environment != "demo" {
							objectErr = localObjects.EnsurePrivateBuckets(ctx, []string{
								settings.QuarantineBucket,
								settings.CanonicalBucket,
								settings.AttachmentBucket,
								settings.DocumentBucket,
							}, settings.ObjectStoreCORSOrigins)
						}
					default:
						objectErr = fmt.Errorf("unsupported AVIA_OBJECT_STORE_MODE %q", settings.ObjectStoreMode)
					}
					var scannerProbe httpapi.ReadinessProbe
					if objectErr == nil {
						scannerProbe, objectErr = newScannerReadiness(settings)
					}
					if objectErr != nil {
						probe = unavailableReadiness{err: objectErr}
						slog.Error("object store unavailable; readiness will fail closed", "error", objectErr)
					} else {
						objectStoreHealth = objectStoreReadiness{store: objects}
						if settings.ScannerMode == "guardduty-s3" {
							// GuardDuty result processing reads exact-version S3 object tags.
							// The same scoped S3 probe therefore verifies both required paths.
							scannerHealth = objectStoreHealth
						} else {
							scannerHealth = scannerProbe
						}
						readinessProbes := combinedReadiness{
							probe,
							objectStoreHealth,
						}
						if scannerProbe != nil {
							readinessProbes = append(readinessProbes, scannerProbe)
						}
						probe = readinessProbes
						appDependencies := application.Dependencies{
							Clock:                     runtimeClock,
							IDGenerator:               profile.idGenerator,
							FindingReferenceGenerator: profile.findingReferenceGenerator,
						}
						if profile.seed != nil {
							if resetErr := profile.seed(ctx, pool, runtimeClock()); resetErr != nil {
								probe = unavailableReadiness{err: resetErr}
								slog.Error("canonical test seed failed; readiness will fail closed", "error", resetErr)
							}
						}
						applicationService := application.NewService(pool, appDependencies)
						grantService := fieldsync.NewGrantService(pool, fieldsync.GrantDependencies{
							Clock: runtimeClock, IDGenerator: profile.idGenerator,
						})
						syncOperations := fieldsync.NewOperationService(pool, fieldsync.OperationDependencies{
							Clock: runtimeClock, IDGenerator: profile.idGenerator,
						})
						evidenceUploadConfig, attachmentUploadConfig := uploadServiceConfigs(
							settings,
							runtimeClock,
							profile.idGenerator,
						)
						evidenceUploads := evidence.NewUploadService(pool, objects, evidenceUploadConfig)
						attachmentUploads := attachments.NewUploadService(pool, objects, attachmentUploadConfig)
						planningService := planning.NewService(
							pool,
							planningServiceDependencies(runtimeClock, profile.idGenerator),
						)
						riskService := risk.NewService(pool, risk.Dependencies{
							Clock: runtimeClock, IDGenerator: profile.idGenerator,
						})
						administrationService := administration.NewProjectionService(
							pool,
							administration.ProjectionDependencies{
								Clock: runtimeClock, DirectoryProvider: directoryProvider,
							},
						)
						assistantService := assistant.NewService(pool, assistant.Dependencies{
							Clock: runtimeClock, IDGenerator: profile.idGenerator,
							Provider: assistant.NewDeterministicProvider(),
						})
						communicationsWorkflow := application.NewCommunicationsWorkflow(
							pool,
							communicationsWorkflowDependencies(
								runtimeClock,
								profile.idGenerator,
							),
						)
						documentAccess := documents.NewService(
							pool,
							objects,
							documents.Dependencies{
								Bucket: settings.DocumentBucket, Clock: runtimeClock,
							},
						)
						apiHandler := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
							Pool: pool, Application: applicationService, GrantService: grantService,
							SyncOperations:  syncOperations,
							EvidenceUploads: evidenceUploads, AttachmentUploads: attachmentUploads,
							Planning: planningService,
							Risk:     riskService, Administration: administrationService,
							Assistant:              assistantService,
							Communications:         communicationsWorkflow,
							Documents:              documentAccess,
							DirectoryProvider:      directoryProvider,
							Users:                  userLifecycleService,
							PreprodExerciseProfile: exerciseProfileEnabled,
							PreprodIdentityNamespace: func() string {
								value, _ := os.LookupEnv("AVIA_PREPROD_IDENTITY_NAMESPACE")
								return value
							}(),
							Clock: runtimeClock,
						}).Handler()
						if profile.protect != nil {
							protectedAPI, testAdmin, profileErr := profile.protect(
								settings,
								apiHandler,
								pool,
								objects,
								[]string{
									settings.QuarantineBucket,
									settings.CanonicalBucket,
									settings.AttachmentBucket,
									settings.DocumentBucket,
								},
							)
							if profileErr != nil {
								probe = unavailableReadiness{err: profileErr}
								slog.Error(
									"canonical test authority unavailable; readiness will fail closed",
									"error",
									profileErr,
								)
							} else {
								authenticatedAPI = protectedAPI
								testAdministration = testAdmin
							}
						} else if authBoundary != nil {
							authenticatedAPI = authBoundary.Protect(apiHandler)
						}
					}
				}
			} else {
				probe = unavailableReadiness{err: bootstrapErr}
				slog.Error("test profile bootstrap failed; readiness will fail closed", "error", bootstrapErr)
			}
		} else {
			probe = unavailableReadiness{err: migrationErr}
			slog.Error("database migrations unavailable; readiness will fail closed", "error", migrationErr)
		}
	} else {
		probe = unavailableReadiness{err: databaseErr}
		slog.Error("database unavailable; readiness will fail closed", "error", databaseErr)
	}
	if pool != nil {
		defer pool.Close()
	}
	runtimeReadiness, readinessErr := newRuntimeReadiness(
		settings,
		probe,
		databaseHealth,
		objectStoreHealth,
		scannerHealth,
	)
	if readinessErr != nil {
		probe = unavailableReadiness{err: readinessErr}
		slog.Error(
			"runtime readiness configuration unavailable; readiness will fail closed",
			"error",
			readinessErr,
		)
	} else {
		probe = runtimeReadiness
	}

	server := &http.Server{
		Addr: settings.HTTPAddress,
		Handler: httpapi.NewInstrumentedApplicationHandler(
			probe,
			authentication,
			authenticatedAPI,
			testAdministration,
			telemetryRuntime.HTTPMiddleware,
		),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("API listening", "address", settings.HTTPAddress, "environment", settings.Environment)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownContext)
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	}
}

func maintainOIDCLoginState(ctx context.Context, manager *session.Manager) {
	if manager == nil {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupContext, cancel := context.WithTimeout(ctx, 5*time.Second)
			if removed, err := manager.CleanupLoginState(cleanupContext, 100); err != nil {
				slog.Warn("OIDC login-state cleanup deferred", "errorClass", "oidc_login_state_cleanup_failure")
			} else if removed > 0 {
				slog.Info("OIDC login-state cleanup completed", "cleanupClass", "oidc_login_state", "removedCount", removed)
			}
			// The global login-admission ceiling is 600 starts per minute. Drain
			// two bounded batches so attacker-variable buckets cannot outgrow
			// retention while cleanup remains bounded per tick.
			for pass := 0; pass < oidcLoginAdmissionCleanupPasses; pass++ {
				removed, err := manager.CleanupLoginAdmission(cleanupContext, oidcLoginAdmissionCleanupBatch)
				if err != nil {
					slog.Warn("OIDC login-admission cleanup deferred", "errorClass", "oidc_login_admission_cleanup_failure")
					break
				}
				if removed > 0 {
					slog.Info("OIDC login-admission cleanup completed", "cleanupClass", "oidc_login_admission", "removedCount", removed)
				}
				if removed < oidcLoginAdmissionCleanupBatch {
					break
				}
			}
			cancel()
		}
	}
}

// verifyCanonicalAGAExerciseDatabase checks the connected PostgreSQL
// identity, rather than trusting process environment variables.  A copied
// profile env file on a shared database must fail readiness before exercise
// catalogs can become visible.
func verifyCanonicalAGAExerciseDatabase(ctx context.Context, pool *database.Pool) error {
	if pool == nil {
		return errors.New("PostgreSQL pool is required for exercise namespace verification")
	}
	var databaseName, databaseOwner string
	if err := pool.QueryRow(ctx, `
		SELECT current_database(), pg_get_userbyid(datdba)
		FROM pg_database
		WHERE datname = current_database()
	`).Scan(&databaseName, &databaseOwner); err != nil {
		return fmt.Errorf("verify connected exercise database identity: %w", err)
	}
	if databaseName != "aviasurveil360_local_preprod" || databaseOwner != "aviasurveil360_preprod_loader" {
		return fmt.Errorf("connected database is not the dedicated exercise namespace: got %s owned by %s", databaseName, databaseOwner)
	}
	return nil
}

// canonicalAGAExerciseProfileEnabled is deliberately stricter than a single
// feature flag. Exercise content is enabled only for the local-preprod
// runtime, the sealed versioned profile, an explicit qualification opt-in,
// the dedicated whole-namespace identity, and the disposable database owner
// envelope. A shared/default database can therefore never expose exercise
// question versions merely because a profile variable was inherited.
// Production/default API processes therefore fail closed even if a profile
// variable is accidentally inherited.
func canonicalAGAExerciseProfileEnabled(environment string, lookup func(string) (string, bool)) bool {
	profile, profileOK := lookup("AVIA_PREPROD_PROFILE")
	qualification, qualificationOK := lookup("AVIA_PREPROD_PROFILE_QUALIFICATION")
	namespace, namespaceOK := lookup("AVIA_PREPROD_IDENTITY_NAMESPACE")
	databaseName, databaseNameOK := lookup("AVIA_PREPROD_DATABASE_NAME")
	databaseOwner, databaseOwnerOK := lookup("AVIA_PREPROD_DATABASE_OWNER")
	return environment == "local-preprod" && profileOK && profile == "aga-preprod@1.0.0" &&
		qualificationOK && strings.EqualFold(strings.TrimSpace(qualification), "true") &&
		namespaceOK && strings.TrimSpace(namespace) == "canonical-aga-preprod-exercise-v1" &&
		databaseNameOK && strings.TrimSpace(databaseName) == "aviasurveil360_local_preprod" &&
		databaseOwnerOK && strings.TrimSpace(databaseOwner) == "aviasurveil360_preprod_loader"
}

func canonicalTestExerciseProfileEnabled(settings config.Settings) bool {
	return settings.Environment == "test" && settings.CanonicalSeed && settings.CanonicalTestProfile
}

func newRuntimeReadiness(
	settings config.Settings,
	applicationProbe httpapi.ReadinessProbe,
	databaseProbe httpapi.ReadinessProbe,
	objectStoreProbe httpapi.ReadinessProbe,
	scannerProbe httpapi.ReadinessProbe,
) (httpapi.ReadinessProbe, error) {
	namedProbe := func(probe httpapi.ReadinessProbe, name string) httpapi.ReadinessProbe {
		if probe != nil {
			return probe
		}
		return unavailableReadiness{err: fmt.Errorf("%s initialization has not completed", name)}
	}
	dependencies := []platformhealth.Dependency{
		{
			Name: "application", Required: true,
			Probe: namedProbe(applicationProbe, "application"), Timeout: settings.RuntimeHealthTimeout,
		},
		{
			Name: "postgresql", Required: true,
			Probe: namedProbe(databaseProbe, "PostgreSQL"), Timeout: settings.RuntimeHealthTimeout,
		},
	}
	if settings.IdentityHealthURL != "" {
		probe, err := platformhealth.NewHTTPProbe(
			settings.IdentityHealthURL,
			settings.RuntimeHealthTimeout,
		)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, platformhealth.Dependency{
			Name: "identity", Required: true, Probe: probe,
			Timeout: settings.RuntimeHealthTimeout,
		})
	}
	if settings.ObjectStoreMode != "" || settings.ObjectStoreEndpoint != "" {
		name := "minio"
		if settings.ObjectStoreMode == "aws-s3" {
			name = "s3"
		}
		dependencies = append(dependencies, platformhealth.Dependency{
			Name: name, Required: true, Probe: namedProbe(objectStoreProbe, name),
			Timeout: settings.RuntimeHealthTimeout,
		})
	}
	if settings.ScannerMode == "guardduty-s3" {
		dependencies = append(dependencies, platformhealth.Dependency{
			Name: "guardduty-s3-result", Required: true, Probe: namedProbe(scannerProbe, "GuardDuty S3 result"),
			Timeout: settings.RuntimeHealthTimeout,
		})
	}
	if settings.SMTPHealthAddress != "" {
		probe, err := platformhealth.NewTCPProbe(
			settings.SMTPHealthAddress,
			settings.RuntimeHealthTimeout,
		)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, platformhealth.Dependency{
			Name: "mailpit", Required: false, Probe: probe,
			Timeout: settings.RuntimeHealthTimeout,
		})
	}
	return platformhealth.NewDependencies(dependencies...)
}

func newScannerReadiness(settings config.Settings) (httpapi.ReadinessProbe, error) {
	switch settings.ScannerMode {
	case "", "disabled", "deterministic-test", "guardduty-s3":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported AVIA_SCANNER_MODE %q", settings.ScannerMode)
	}
}

func uploadServiceConfigs(
	settings config.Settings,
	clock func() time.Time,
	idGenerator func(string) string,
) (evidence.UploadServiceConfig, attachments.UploadServiceConfig) {
	return evidence.UploadServiceConfig{
			QuarantineBucket: settings.QuarantineBucket,
			CanonicalBucket:  settings.CanonicalBucket,
			MaximumByteSize:  25 * 1024 * 1024,
			InstructionTTL:   10 * time.Minute,
			Clock:            clock,
			IDGenerator:      idGenerator,
		}, attachments.UploadServiceConfig{
			QuarantineBucket: settings.QuarantineBucket,
			MaximumByteSize:  25 * 1024 * 1024,
			InstructionTTL:   10 * time.Minute,
			Clock:            clock,
			IDGenerator:      idGenerator,
		}
}

func planningServiceDependencies(
	clock func() time.Time,
	idGenerator func(string) string,
) planning.Dependencies {
	return planning.Dependencies{Clock: clock, IDGenerator: idGenerator}
}

func communicationsWorkflowDependencies(
	clock func() time.Time,
	idGenerator func(string) string,
) application.CommunicationsWorkflowDependencies {
	return application.CommunicationsWorkflowDependencies{
		Clock:       clock,
		IDGenerator: idGenerator,
	}
}
