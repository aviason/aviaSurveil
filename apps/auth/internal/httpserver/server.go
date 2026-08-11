package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/config"
)

const notReadyReason = "provider_runtime_not_initialized"

type Server struct {
	httpServer *http.Server
	ready      *atomic.Bool
}

func New(settings config.Settings, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	ready := &atomic.Bool{}
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writeHealth(writer, http.StatusOK, "alive", "")
	})
	mux.HandleFunc("/health/ready", func(writer http.ResponseWriter, _ *http.Request) {
		if !serverReady(writer, ready) {
			return
		}
		writeHealth(writer, http.StatusOK, "ready", "")
	})
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writeHealth(writer, http.StatusOK, "alive", "")
	})
	// No OIDC or administrative routes are mounted in Task 2. A route can be
	// added only after its storage, identity, and negative protocol contract is
	// implemented in a later task.
	mux.HandleFunc("/", func(writer http.ResponseWriter, _ *http.Request) {
		http.NotFound(writer, nil)
	})

	return &Server{ready: ready, httpServer: &http.Server{
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

func serverReady(writer http.ResponseWriter, ready *atomic.Bool) bool {
	if ready.Load() {
		return true
	}
	// The Task 2 provider has no database/storage runtime yet. Keep readiness
	// fail-closed even though the liveness endpoint is available.
	writeHealth(writer, http.StatusServiceUnavailable, "not_ready", notReadyReason)
	return false
}

func writeHealth(writer http.ResponseWriter, status int, state, reason string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(struct {
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	}{Status: state, Reason: reason})
}
