package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/config"
)

const (
	notReadyReason              = "provider_runtime_not_initialized"
	dependencyUnavailableReason = "provider_dependency_unavailable"
)

type Server struct {
	httpServer     *http.Server
	ready          *atomic.Bool
	readinessCheck func(context.Context) error
}

func New(settings config.Settings, logger *slog.Logger) *Server {
	return NewWithRuntime(settings, logger, nil)
}

// NewWithRuntime mounts a fully initialized isolated candidate handler while
// retaining the fail-closed health-only constructor used before dependencies
// are available. The caller must not call Ready until every dependency has
// been checked successfully.
func NewWithRuntime(settings config.Settings, logger *slog.Logger, runtime http.Handler) *Server {
	return NewWithRuntimeReadiness(settings, logger, runtime, nil)
}

// NewWithRuntimeReadiness keeps the provider fail-closed after startup: every
// readiness request verifies its critical dependencies rather than treating a
// successful boot as permanent availability.
func NewWithRuntimeReadiness(settings config.Settings, logger *slog.Logger, runtime http.Handler, readinessCheck func(context.Context) error) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	ready := &atomic.Bool{}
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writeHealth(writer, http.StatusOK, "alive", "")
	})
	mux.HandleFunc("/health/ready", func(writer http.ResponseWriter, _ *http.Request) {
		if !serverReady(writer, ready, readinessCheck) {
			return
		}
		writeHealth(writer, http.StatusOK, "ready", "")
	})
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writeHealth(writer, http.StatusOK, "alive", "")
	})
	if runtime == nil {
		mux.HandleFunc("/", func(writer http.ResponseWriter, _ *http.Request) { http.NotFound(writer, nil) })
	} else {
		mux.Handle("/", runtime)
	}

	return &Server{ready: ready, readinessCheck: readinessCheck, httpServer: &http.Server{
		Addr:              settings.HTTPAddress,
		Handler:           mux,
		ReadHeaderTimeout: settings.ReadHeaderTimeout,
		ReadTimeout:       settings.ReadTimeout,
		WriteTimeout:      settings.WriteTimeout,
		IdleTimeout:       settings.IdleTimeout,
		MaxHeaderBytes:    settings.MaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}}
}

func (server *Server) ListenAndServe() error {
	return server.httpServer.ListenAndServe()
}

func (server *Server) Shutdown(ctx context.Context) error {
	return server.httpServer.Shutdown(ctx)
}

func (server *Server) Ready() {
	server.ready.Store(true)
}

func (server *Server) SetReady(ready bool) {
	server.ready.Store(ready)
}

func serverReady(writer http.ResponseWriter, ready *atomic.Bool, readinessCheck func(context.Context) error) bool {
	if !ready.Load() {
		// The Task 2 provider has no database/storage runtime yet. Keep readiness
		// fail-closed even though the liveness endpoint is available.
		writeHealth(writer, http.StatusServiceUnavailable, "not_ready", notReadyReason)
		return false
	}
	if readinessCheck != nil {
		checkContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if readinessCheck(checkContext) != nil {
			writeHealth(writer, http.StatusServiceUnavailable, "not_ready", dependencyUnavailableReason)
			return false
		}
	}
	return true
}

func writeHealth(writer http.ResponseWriter, status int, state, reason string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(struct {
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	}{Status: state, Reason: reason})
}
