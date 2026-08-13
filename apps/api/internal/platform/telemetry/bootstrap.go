package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/netpolicy"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/aviason/aviaSurveil"

type Config struct {
	ServiceName      string
	ServiceVersion   string
	ServiceInstance  string
	Environment      string
	OTLPHTTPEndpoint string
	ExportInterval   time.Duration
	ExportTimeout    time.Duration
	SpanExporter     sdktrace.SpanExporter
	MetricReader     sdkmetric.Reader
}

type Runtime struct {
	tracerProvider *sdktrace.TracerProvider
	metricProvider *sdkmetric.MeterProvider
	propagator     propagation.TextMapPropagator
	tracer         trace.Tracer
	httpDuration   metric.Float64Histogram
	dbDuration     metric.Float64Histogram
	outboxReadyAge metric.Float64Histogram
	jobAttempts    metric.Int64Counter
}

func NewRuntime(ctx context.Context, config Config) (*Runtime, error) {
	if err := validateRuntimeConfig(config); err != nil {
		return nil, err
	}
	if config.ExportTimeout <= 0 {
		config.ExportTimeout = 3 * time.Second
	}
	if config.ExportInterval <= 0 {
		config.ExportInterval = 10 * time.Second
	}
	if config.ServiceInstance == "" {
		config.ServiceInstance = newCorrelationID()
	}
	runtimeResource := resource.NewSchemaless(
		attribute.String("service.name", config.ServiceName),
		attribute.String("service.version", config.ServiceVersion),
		attribute.String("service.instance.id", config.ServiceInstance),
		attribute.String("deployment.environment.name", config.Environment),
	)

	traceOptions := []sdktrace.TracerProviderOption{sdktrace.WithResource(runtimeResource)}
	if config.SpanExporter != nil {
		traceOptions = append(
			traceOptions,
			sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(config.SpanExporter)),
		)
	} else if config.OTLPHTTPEndpoint != "" {
		exporter, err := otlptracehttp.New(
			ctx,
			otlptracehttp.WithEndpointURL(config.OTLPHTTPEndpoint),
			otlptracehttp.WithTimeout(config.ExportTimeout),
		)
		if err != nil {
			return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
		}
		traceOptions = append(traceOptions, sdktrace.WithBatcher(exporter))
	}
	tracerProvider := sdktrace.NewTracerProvider(traceOptions...)

	metricOptions := []sdkmetric.Option{sdkmetric.WithResource(runtimeResource)}
	if config.MetricReader != nil {
		metricOptions = append(metricOptions, sdkmetric.WithReader(config.MetricReader))
	} else if config.OTLPHTTPEndpoint != "" {
		exporter, err := otlpmetrichttp.New(
			ctx,
			otlpmetrichttp.WithEndpointURL(config.OTLPHTTPEndpoint),
			otlpmetrichttp.WithTimeout(config.ExportTimeout),
		)
		if err != nil {
			_ = tracerProvider.Shutdown(ctx)
			return nil, fmt.Errorf("create OTLP metric exporter: %w", err)
		}
		metricOptions = append(
			metricOptions,
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(config.ExportInterval),
				sdkmetric.WithTimeout(config.ExportTimeout),
			)),
		)
	}
	metricProvider := sdkmetric.NewMeterProvider(metricOptions...)
	meter := metricProvider.Meter(instrumentationName)
	httpDuration, err := meter.Float64Histogram(
		"http.server.duration",
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(5, 10, 25, 50, 100, 250, 500, 1000, 2500),
	)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		_ = metricProvider.Shutdown(ctx)
		return nil, fmt.Errorf("create HTTP duration histogram: %w", err)
	}
	dbDuration, err := meter.Float64Histogram(
		"db.client.operation.duration",
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 2.5, 5, 10, 25, 50, 100, 250, 500),
	)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		_ = metricProvider.Shutdown(ctx)
		return nil, fmt.Errorf("create database duration histogram: %w", err)
	}
	outboxReadyAge, err := meter.Float64Histogram(
		"outbox.ready.age",
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(5, 15, 30, 60, 120, 300, 600),
	)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		_ = metricProvider.Shutdown(ctx)
		return nil, fmt.Errorf("create outbox ready age histogram: %w", err)
	}
	jobAttempts, err := meter.Int64Counter("worker.job.attempts", metric.WithUnit("attempt"))
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		_ = metricProvider.Shutdown(ctx)
		return nil, fmt.Errorf("create job attempts counter: %w", err)
	}
	return &Runtime{
		tracerProvider: tracerProvider,
		metricProvider: metricProvider,
		propagator: propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
		tracer:         tracerProvider.Tracer(instrumentationName),
		httpDuration:   httpDuration,
		dbDuration:     dbDuration,
		outboxReadyAge: outboxReadyAge,
		jobAttempts:    jobAttempts,
	}, nil
}

func validateRuntimeConfig(config Config) error {
	if config.ServiceName == "" {
		return errors.New("telemetry service name is required")
	}
	if config.ServiceVersion == "" {
		return errors.New("telemetry service version is required")
	}
	if config.Environment == "" {
		return errors.New("telemetry environment is required")
	}
	if config.ServiceInstance != "" &&
		!safeCorrelationID.MatchString(config.ServiceInstance) {
		return errors.New("telemetry service instance must be a bounded opaque identifier")
	}
	if config.OTLPHTTPEndpoint != "" {
		endpoint, err := url.Parse(config.OTLPHTTPEndpoint)
		if err != nil || endpoint.Scheme == "" || endpoint.Host == "" ||
			endpoint.Scheme != "http" || endpoint.User != nil ||
			endpoint.RawQuery != "" || endpoint.Fragment != "" ||
			!netpolicy.IsPrivateHost(endpoint.Hostname()) {
			return errors.New("telemetry OTLP endpoint must be an absolute private HTTP URL without credentials, query, or fragment")
		}
	}
	if err := DefaultContract().Validate(); err != nil {
		return fmt.Errorf("validate telemetry contract: %w", err)
	}
	return nil
}

func (runtime *Runtime) Tracer() trace.Tracer {
	return runtime.tracer
}

func (runtime *Runtime) ForceFlush(ctx context.Context) error {
	return errors.Join(
		runtime.tracerProvider.ForceFlush(ctx),
		runtime.metricProvider.ForceFlush(ctx),
	)
}

func (runtime *Runtime) Shutdown(ctx context.Context) error {
	return errors.Join(
		runtime.tracerProvider.Shutdown(ctx),
		runtime.metricProvider.Shutdown(ctx),
	)
}

func NewJSONLogger(writer io.Writer, serviceName string) *slog.Logger {
	if writer == nil {
		writer = os.Stderr
	}
	return slog.New(redactingHandler{
		next: slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	}).With("service", serviceName)
}

type redactingHandler struct {
	next slog.Handler
}

func (handler redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return handler.next.Enabled(ctx, level)
}

func (handler redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	sanitized := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attribute slog.Attr) bool {
		sanitized.AddAttrs(redactLogAttribute(attribute))
		return true
	})
	return handler.next.Handle(ctx, sanitized)
}

func (handler redactingHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	sanitized := make([]slog.Attr, len(attributes))
	for index, attribute := range attributes {
		sanitized[index] = redactLogAttribute(attribute)
	}
	return redactingHandler{next: handler.next.WithAttrs(sanitized)}
}

func (handler redactingHandler) WithGroup(name string) slog.Handler {
	return redactingHandler{next: handler.next.WithGroup(name)}
}

func redactLogAttribute(attribute slog.Attr) slog.Attr {
	attribute.Value = attribute.Value.Resolve()
	if sensitiveLogKey(attribute.Key) {
		return slog.String(attribute.Key, "[REDACTED]")
	}
	if attribute.Value.Kind() != slog.KindGroup {
		return attribute
	}
	group := attribute.Value.Group()
	for index, child := range group {
		group[index] = redactLogAttribute(child)
	}
	return slog.Group(attribute.Key, groupToAny(group)...)
}

func groupToAny(attributes []slog.Attr) []any {
	values := make([]any, len(attributes))
	for index, attribute := range attributes {
		values[index] = attribute
	}
	return values
}

func sensitiveLogKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "error" {
		return true
	}
	for _, fragment := range []string{
		"password",
		"token",
		"cookie",
		"secret",
		"authorization",
		"evidence.bytes",
		"message.body",
		"internal_caa_note",
	} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
