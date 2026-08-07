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

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/administration"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/agacandidatedemo"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/assistant"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/documents"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/evidence"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/inspections/attachments"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/planning"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	platformhealth "github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/health"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/scanner"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/session"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/telemetry"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/risk"
	fieldsync "github.com/MarlonJD/aviaSurveil360/apps/api/internal/sync"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
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
	if settings.KeycloakAdminURL != "" {
		keycloakClient, keycloakErr := identity.NewKeycloakAdminClient(
			identity.KeycloakAdminConfig{
				BaseURL:      settings.KeycloakAdminURL,
				Realm:        settings.KeycloakRealm,
				ClientID:     settings.KeycloakServiceClientID,
				ClientSecret: settings.KeycloakServiceClientSecret,
			},
		)
		if keycloakErr != nil {
			return fmt.Errorf("configure Keycloak directory provider: %w", keycloakErr)
		}
		directoryProvider = keycloakClient
		authorityObserver = keycloakClient
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
		if !profile.skipMigrations {
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
				var authBoundary *httpapi.AuthBoundary
				if settings.OIDCIssuerURL != "" {
					var sessionManager *session.Manager
					var managerErr error
					if authorityObserver == nil {
						managerErr = errors.New(
							"Keycloak authority observer is required for OIDC sessions",
						)
					} else {
						sessionManager, managerErr = session.NewManager(
							pool,
							settings.SessionEncryptionKey,
							session.ManagerDependencies{
								AuthorityObserver:    authorityObserver,
								ActivationReconciler: userLifecycleService,
							},
						)
					}
					if managerErr != nil {
						probe = unavailableReadiness{err: managerErr}
						slog.Error("session manager unavailable; readiness will fail closed", "error", managerErr)
					} else {
						provider, providerErr := identity.NewRemoteOIDCProvider(ctx, identity.RemoteOIDCConfig{
							IssuerURL: settings.OIDCIssuerURL, DiscoveryURL: settings.OIDCDiscoveryURL,
							ClientID:     settings.OIDCClientID,
							ClientSecret: settings.OIDCClientSecret, RedirectURL: settings.OIDCRedirectURL,
						})
						if providerErr != nil {
							probe = unavailableReadiness{err: providerErr}
							slog.Error("OIDC provider unavailable; readiness will fail closed", "error", providerErr)
						} else {
							authBoundary = httpapi.NewAuthBoundaryWithCookieSecure(provider, sessionManager, settings.CookieSecure)
							authentication = authBoundary.Handler()
						}
					}
				}
				if profile.agaDemoOnly {
					if authBoundary == nil || profile.agaDemoService == nil || profile.agaWorkspaceService == nil {
						probe = unavailableReadiness{err: errors.New("tagged AGA demo authentication is unavailable")}
					} else {
						service, closeReader, readerErr := profile.agaDemoService(ctx, settings)
						if readerErr != nil {
							probe = unavailableReadiness{err: readerErr}
							slog.Error("AGA demo reader unavailable; capability will fail closed", "error", readerErr)
						} else {
							defer closeReader()
							workspaceService, closeWorkspace, workspaceErr := profile.agaWorkspaceService(ctx, settings)
							if workspaceErr != nil {
								probe = unavailableReadiness{err: workspaceErr}
								slog.Error("AGA demo workspace unavailable; capability will fail closed", "error", workspaceErr)
							} else {
								defer closeWorkspace()
								legacyAPI := httpapi.ProtectAGACandidateDemo(authBoundary, httpapi.NewAGACandidateDemoHandler(service))
								workspaceAPI := httpapi.ProtectAGADemoWorkspace(authBoundary, workspaceService, httpapi.NewAGADemoWorkspaceHandler(workspaceService))
								authenticatedAPI = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
									if strings.HasPrefix(request.URL.Path, "/v1/preprod/aga-demo-workspace/") {
										workspaceAPI.ServeHTTP(writer, request)
										return
									}
									legacyAPI.ServeHTTP(writer, request)
								})
							}
						}
					}
				} else if settings.ObjectStoreEndpoint != "" {
					objects, objectErr := objectstore.NewMinIOStore(objectstore.MinIOConfig{
						Endpoint: settings.ObjectStoreEndpoint, PublicEndpoint: settings.ObjectStorePublicEndpoint,
						AccessKey: settings.ObjectStoreAccessKey, SecretKey: settings.ObjectStoreSecretKey,
						UseTLS: settings.ObjectStoreTLS, PublicUseTLS: settings.ObjectStorePublicTLS,
						Region: settings.ObjectStoreRegion, AllowServerManagedCORS: settings.AllowServerManagedCORS,
						Clock: runtimeClock,
					})
					if objectErr == nil && settings.Environment != "production" {
						objectErr = objects.EnsurePrivateBuckets(ctx, []string{
							settings.QuarantineBucket,
							settings.CanonicalBucket,
							settings.AttachmentBucket,
							settings.DocumentBucket,
						}, settings.ObjectStoreCORSOrigins)
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
						scannerHealth = scannerProbe
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
						var agaDemoService any
						if profile.agaDemoService != nil {
							service, closeReader, readerErr := profile.agaDemoService(ctx, settings)
							if readerErr != nil {
								probe = unavailableReadiness{err: readerErr}
								slog.Error("AGA demo reader unavailable; capability will fail closed", "error", readerErr)
							} else {
								defer closeReader()
								agaDemoService = service
							}
						}
						apiHandler := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
							Pool: pool, Application: applicationService, GrantService: grantService,
							SyncOperations:  syncOperations,
							EvidenceUploads: evidenceUploads, AttachmentUploads: attachmentUploads,
							Planning: planningService,
							Risk:     riskService, Administration: administrationService,
							Assistant:         assistantService,
							Communications:    communicationsWorkflow,
							Documents:         documentAccess,
							DirectoryProvider: directoryProvider,
							Users:             userLifecycleService,
							AGACandidateDemo: func() *agacandidatedemo.Service {
								service, _ := agaDemoService.(*agacandidatedemo.Service)
								return service
							}(),
							PreprodExerciseProfile: os.Getenv("AVIA_PREPROD_PROFILE") == "aga-preprod@1.0.0" && strings.EqualFold(os.Getenv("AVIA_PREPROD_PROFILE_QUALIFICATION"), "true"),
							Clock:                  runtimeClock,
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
	if settings.ObjectStoreEndpoint != "" {
		dependencies = append(dependencies, platformhealth.Dependency{
			Name: "minio", Required: true, Probe: namedProbe(objectStoreProbe, "MinIO"),
			Timeout: settings.RuntimeHealthTimeout,
		})
	}
	if settings.ScannerMode == "clamav" {
		dependencies = append(dependencies, platformhealth.Dependency{
			Name: "clamav", Required: true, Probe: namedProbe(scannerProbe, "ClamAV"),
			Timeout: settings.RuntimeHealthTimeout,
		})
	}
	if settings.GotenbergHealthURL != "" {
		probe, err := platformhealth.NewHTTPProbe(
			settings.GotenbergHealthURL,
			settings.RuntimeHealthTimeout,
		)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, platformhealth.Dependency{
			Name: "gotenberg", Required: false, Probe: probe,
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
	case "", "deterministic-test":
		return nil, nil
	case "clamav":
		return scanner.NewClamAV(scanner.ClamAVConfig{
			Address:             settings.ClamAVAddress,
			MaximumSignatureAge: settings.ClamAVMaximumSignatureAge,
		})
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
