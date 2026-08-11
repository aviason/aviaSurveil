package graphql

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	apperrors "kindred_server/internal/platform/errors"
)

type ResolverFunc func(context.Context, ResolverEvent) (any, error)

type Handler struct {
	resolvers map[resolverKey]ResolverFunc
	logger    ResolverLogger
}

type ResolverLogger interface {
	Info(message string, fields map[string]any)
}

type HandlerOption func(*Handler)

type resolverKey struct {
	parentType string
	fieldName  string
}

type AppSyncError struct {
	ErrorMessage string            `json:"errorMessage"`
	ErrorType    string            `json:"errorType"`
	ErrorInfo    *AppSyncErrorInfo `json:"errorInfo,omitempty"`
}

type AppSyncErrorInfo struct {
	Status int    `json:"status"`
	Code   string `json:"code"`
}

func NewHandler(opts ...HandlerOption) *Handler {
	h := &Handler{resolvers: make(map[resolverKey]ResolverFunc)}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func WithLogger(logger ResolverLogger) HandlerOption {
	return func(h *Handler) {
		h.logger = logger
	}
}

func (h *Handler) Register(parentTypeName, fieldName string, resolver ResolverFunc) {
	if h.resolvers == nil {
		h.resolvers = make(map[resolverKey]ResolverFunc)
	}
	h.resolvers[resolverKey{parentType: parentTypeName, fieldName: fieldName}] = resolver
}

func (h *Handler) Handle(ctx context.Context, event ResolverEvent) (payload any, err error) {
	parentType, fieldName := operation(event)
	start := time.Now()
	stats := newResolverStats()
	ctx = contextWithResolverStats(ctx, stats)
	defer func() {
		h.logResolverSummary(event, parentType, fieldName, start, stats, err)
	}()

	resolver := h.resolvers[resolverKey{parentType: parentType, fieldName: fieldName}]
	if resolver == nil {
		return nil, NewAppSyncError(
			http.StatusNotFound,
			"unknown_field",
			fmt.Sprintf("unknown GraphQL resolver field %s.%s", parentType, fieldName),
		)
	}

	payload, err = resolver(ctx, event)
	if err != nil {
		var appSyncErr *AppSyncError
		if errors.As(err, &appSyncErr) {
			return nil, appSyncErr
		}
		return nil, AppSyncErrorFromError(err)
	}
	return payload, nil
}

func (h *Handler) logResolverSummary(
	event ResolverEvent,
	parentType string,
	fieldName string,
	start time.Time,
	stats *resolverStats,
	err error,
) {
	if h.logger == nil {
		return
	}
	fields := map[string]any{
		"parent_type":     parentType,
		"field_name":      fieldName,
		"duration_ms":     time.Since(start).Milliseconds(),
		"success":         err == nil,
		"selection_count": len(event.Info.SelectionSetList),
	}
	if counters := stats.snapshot(); len(counters) > 0 {
		fields["counters"] = counters
	}
	if err != nil {
		var appErr *AppSyncError
		if errors.As(err, &appErr) {
			fields["error_code"] = appErr.ErrorType
			if appErr.ErrorInfo != nil {
				fields["error_status"] = appErr.ErrorInfo.Status
			}
		}
	}
	h.logger.Info("graphql resolver completed", fields)
}

func NewAppSyncError(status int, code, message string) *AppSyncError {
	return &AppSyncError{
		ErrorMessage: message,
		ErrorType:    code,
		ErrorInfo: &AppSyncErrorInfo{
			Status: status,
			Code:   code,
		},
	}
}

func AppSyncErrorFromError(err error) *AppSyncError {
	appErr := apperrors.ToAppError(err)
	return NewAppSyncError(appErr.Status, appErr.Code, appErr.Message)
}

func (e *AppSyncError) Error() string {
	return e.ErrorMessage
}
