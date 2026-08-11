package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/config"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/httpserver"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/provider"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/telemetry"
)

var version = "candidate"

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
	protocol := provider.NewProtocolBoundary()
	server := httpserver.New(settings, logger)
	logger.Info("isolated auth scaffold starting",
		slog.String("service", "auth"),
		slog.String("version", version),
		slog.String("environment", settings.Environment),
		slog.String("profile", settings.Profile),
		slog.String("address", settings.HTTPAddress),
		slog.String("issuer_host", issuerHost(settings.IssuerURL)),
		slog.String("selected_library", provider.SelectedLibrary),
		slog.String("selected_version", provider.SelectedVersion),
		slog.String("authorization_endpoint", protocol.AuthorizationEndpoint),
	)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(shutdownContext); shutdownErr != nil {
			return fmt.Errorf("shutdown auth scaffold: %w", shutdownErr)
		}
		return nil
	case serverErr := <-serverErrors:
		if errors.Is(serverErr, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen on auth scaffold: %w", serverErr)
	}
}

func issuerHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "[invalid]"
	}
	return parsed.Host
}
