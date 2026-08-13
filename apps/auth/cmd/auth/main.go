package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/admin"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/challenge"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/config"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/httpserver"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/mail"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/mfa"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/password"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/provider"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/telemetry"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

var version = "candidate"

const (
	throttleCleanupBatch   = 10000
	throttleCleanupPasses  = 2
	challengeCleanupBatch  = 1000
	challengeCleanupPasses = 2
	deliveryCleanupBatch   = 1000
	deliveryCleanupPasses  = 2
	providerCleanupBatch   = 1000
	providerCleanupPasses  = 2
)

func main() {
	logger := telemetry.NewRedactedLogger(os.Stderr)
	slog.SetDefault(logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, logger); err != nil {
		logger.Error("auth provider stopped", slog.String("error_class", "startup_or_runtime_failure"))
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	settings, err := config.Load(os.LookupEnv, os.ReadFile)
	if err != nil {
		return fmt.Errorf("load isolated auth configuration: %w", err)
	}
	pool, err := pgxpool.New(ctx, settings.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open isolated auth database: %w", err)
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		return fmt.Errorf("apply isolated auth migrations: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("check isolated auth database: %w", err)
	}
	hasher, err := password.NewDefault()
	if err != nil {
		return fmt.Errorf("initialize password hasher: %w", err)
	}
	limiter, err := throttle.NewPostgresLimiter(pool, time.Now)
	if err != nil {
		return fmt.Errorf("initialize durable login limiter: %w", err)
	}
	identities, err := identity.NewPostgresStore(pool, identity.Config{Hasher: hasher, PasswordPolicy: password.DefaultPolicy(), Limiter: limiter})
	if err != nil {
		return fmt.Errorf("initialize durable identity store: %w", err)
	}
	mfaKey := settings.MFAKey()
	mfaStore, err := mfa.NewPostgresStore(pool, mfa.Config{EncryptionKey: mfaKey[:]})
	if err != nil {
		return fmt.Errorf("initialize durable MFA store: %w", err)
	}
	challenges, err := challenge.NewPostgresStore(pool, challenge.Config{})
	if err != nil {
		return fmt.Errorf("initialize durable challenge store: %w", err)
	}
	dataKey := settings.DataEncryptionKey()
	outbox, err := mail.NewOutbox(mail.OutboxConfig{Pool: pool, EncryptionKey: dataKey[:]})
	if err != nil {
		return fmt.Errorf("initialize durable mail outbox: %w", err)
	}
	smtpHost, _, err := net.SplitHostPort(settings.SMTPAddress)
	if err != nil {
		return fmt.Errorf("parse SMTP address: %w", err)
	}
	sender, err := mail.NewSender(mail.Config{Address: settings.SMTPAddress, Host: smtpHost, From: settings.SMTPFrom, Username: settings.SMTPUsername, Password: settings.SMTPPassword(), TLSMode: mail.TLSMode(settings.SMTPTLSMode), TLSConfig: settings.SMTPTLSConfig(), Timeout: 30 * time.Second})
	if err != nil {
		return fmt.Errorf("initialize verified SMTP sender: %w", err)
	}
	candidateConfig := provider.CandidateConfig{Issuer: settings.IssuerURL, AllowInsecure: issuerAllowsHTTP(settings.IssuerURL), CryptoKey: dataKey, CryptoKeyID: settings.SigningKeyID, SigningKey: settings.SigningKey(), SigningKeyID: settings.SigningKeyID, ClientID: settings.OIDCClientID, ClientSecret: settings.OIDCClientSecret(), RedirectURI: settings.OIDCRedirectURI, PostLogoutRedirectURI: settings.OIDCLogoutURI}
	storage, err := provider.NewPostgresStorage(pool, provider.PostgresStorageConfig{Candidate: candidateConfig, EncryptionKey: dataKey[:], Limiter: limiter})
	if err != nil {
		return fmt.Errorf("initialize durable OIDC storage: %w", err)
	}
	if err := storage.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap isolated OIDC material: %w", err)
	}
	identities.SetSessionRevoker(storage)
	mfaStore.SetSessionRevoker(storage)
	runtime, err := provider.NewRuntimeCandidate(candidateConfig, storage, provider.RuntimeDependencies{Identity: identities, MFA: mfaStore, Challenges: challenges, Outbox: outbox, Logger: logger})
	if err != nil {
		return fmt.Errorf("initialize isolated OIDC runtime: %w", err)
	}
	if err := storage.Health(ctx); err != nil {
		return fmt.Errorf("check durable OIDC storage: %w", err)
	}
	server := httpserver.NewWithRuntimeReadiness(settings, logger, runtime.Handler, func(checkContext context.Context) error {
		if err := storage.Health(checkContext); err != nil {
			return err
		}
		return sender.Probe(checkContext)
	})
	server.Ready()
	var adminServer *httpserver.Server
	if settings.AdminHTTPAddress != "" {
		adminRuntime, adminErr := admin.NewServer(admin.Config{
			Pool: pool, Secret: settings.AdminBearerSecret(), Issuer: settings.IssuerURL,
			Identity: identities, MFA: mfaStore, Challenges: challenges, Outbox: outbox, Provider: storage,
		})
		if adminErr != nil {
			return fmt.Errorf("initialize provider-admin server: %w", adminErr)
		}
		adminSettings := settings
		adminSettings.HTTPAddress = settings.AdminHTTPAddress
		adminServer = httpserver.NewWithRuntimeReadiness(adminSettings, logger, adminRuntime.Handler(), func(checkContext context.Context) error {
			return storage.Health(checkContext)
		})
		adminServer.Ready()
	}
	go deliverOutbox(ctx, outbox, sender, logger)
	go maintainProvider(ctx, storage, challenges, limiter, outbox, logger)
	logger.Info("isolated auth candidate starting",
		slog.String("service", "auth"),
		slog.String("version", version),
		slog.String("environment", settings.Environment),
		slog.String("profile", settings.Profile),
		slog.String("address", settings.HTTPAddress),
		slog.String("issuer_host", issuerHost(settings.IssuerURL)),
		slog.String("selected_library", provider.SelectedLibrary),
		slog.String("selected_version", provider.SelectedVersion),
	)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()
	if adminServer != nil {
		go func() {
			serverErrors <- adminServer.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(shutdownContext); shutdownErr != nil {
			return fmt.Errorf("shutdown auth scaffold: %w", shutdownErr)
		}
		if adminServer != nil {
			if shutdownErr := adminServer.Shutdown(shutdownContext); shutdownErr != nil {
				return fmt.Errorf("shutdown provider-admin server: %w", shutdownErr)
			}
		}
		return nil
	case serverErr := <-serverErrors:
		if errors.Is(serverErr, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen on auth scaffold: %w", serverErr)
	}
}

func maintainProvider(ctx context.Context, storage *provider.PostgresStorage, challenges *challenge.PostgresStore, limiter *throttle.PostgresLimiter, outbox *mail.Outbox, logger *slog.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupContext, cancel := context.WithTimeout(ctx, 5*time.Second)
			// Provider authorization admission allows up to 600 requests per
			// minute. Drain two bounded batches so expired codes and requests do
			// not outgrow retention while cleanup remains bounded per tick.
			for pass := 0; pass < providerCleanupPasses; pass++ {
				removed, err := storage.CleanupExpired(cleanupContext, providerCleanupBatch)
				if err != nil {
					logger.Warn("provider authorization cleanup deferred", slog.String("error_class", "provider_cleanup_failure"))
					break
				}
				if removed > 0 {
					logger.Info("auth cleanup completed", slog.String("cleanup_class", "provider_authorization"), slog.Int64("removed_count", removed))
				}
				if removed < providerCleanupBatch {
					break
				}
			}
			// Recovery admission allows up to 600 requests per minute for each
			// public recovery purpose. Drain two bounded batches so challenge
			// retention stays ahead of that aggregate ceiling without an unbounded
			// delete.
			for pass := 0; pass < challengeCleanupPasses; pass++ {
				removed, err := challenges.Cleanup(cleanupContext, time.Now().UTC(), challengeCleanupBatch)
				if err != nil {
					logger.Warn("identity challenge cleanup deferred", slog.String("error_class", "challenge_cleanup_failure"))
					break
				}
				if removed > 0 {
					logger.Info("auth cleanup completed", slog.String("cleanup_class", "identity_challenge"), slog.Int("removed_count", removed))
				}
				if removed < challengeCleanupBatch {
					break
				}
			}
			// Attacker-variable buckets are admitted behind a global rule. Drain
			// enough bounded batches each minute to keep cleanup ahead of the
			// maximum configured admission rate without an unbounded delete.
			for pass := 0; pass < throttleCleanupPasses; pass++ {
				removed, err := limiter.Cleanup(cleanupContext, time.Now().UTC(), throttleCleanupBatch)
				if err != nil {
					logger.Warn("authentication throttle cleanup deferred", slog.String("error_class", "throttle_cleanup_failure"))
					break
				}
				if removed > 0 {
					logger.Info("auth cleanup completed", slog.String("cleanup_class", "throttle_bucket"), slog.Int64("removed_count", removed))
				}
				if removed < throttleCleanupBatch {
					break
				}
			}
			// Recovery admission permits up to 600 requests per minute for each
			// public recovery purpose. Keep retained delivered/terminal rows from
			// becoming the next durable backlog while retaining bounded deletes.
			for pass := 0; pass < deliveryCleanupPasses; pass++ {
				removed, err := outbox.CleanupRetention(cleanupContext, time.Now().UTC(), deliveryCleanupBatch)
				if err != nil {
					logger.Warn("mail retention cleanup deferred", slog.String("error_class", "mail_retention_cleanup_failure"))
					break
				}
				if removed > 0 {
					logger.Info("auth cleanup completed", slog.String("cleanup_class", "mail_retention"), slog.Int64("removed_count", removed))
				}
				if removed < deliveryCleanupBatch {
					break
				}
			}
			cancel()
		}
	}
}

func deliverOutbox(ctx context.Context, outbox *mail.Outbox, sender *mail.Sender, logger *slog.Logger) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for {
				found, err := outbox.DeliverOnce(ctx, sender)
				if err != nil {
					logger.Warn("candidate outbox delivery deferred", slog.String("error_class", "mail_delivery_failure"))
					break
				}
				if !found {
					break
				}
			}
		}
	}
}

func issuerAllowsHTTP(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "http"
}

func issuerHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "[invalid]"
	}
	return parsed.Host
}
