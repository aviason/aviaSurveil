package integration_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/telemetry"
	"github.com/aviason/aviaSurveil/migrations"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestTelemetryPipelinePropagatesHTTPPostgresAndJobContext(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	reader := metric.NewManualReader()
	runtime, err := telemetry.NewRuntime(context.Background(), telemetry.Config{
		ServiceName:    "api",
		ServiceVersion: "test",
		Environment:    "integration",
		SpanExporter:   exporter,
		MetricReader:   reader,
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	t.Cleanup(func() {
		_ = runtime.Shutdown(context.Background())
	})

	handler := runtime.HTTPMiddleware(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if correlationID := telemetry.CorrelationIDFromContext(request.Context()); correlationID == "" {
			t.Fatal("handler context has no correlation ID")
		}
		if err := runtime.TracePostgres(
			request.Context(),
			"select",
			"findings",
			func(context.Context) error { return nil },
		); err != nil {
			t.Fatalf("TracePostgres() error = %v", err)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/findings/FINDING-PRIVATE-ID", nil)
	request.Header.Set(
		"traceparent",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("response status = %d", response.Code)
	}
	if correlationID := response.Header().Get("X-Correlation-ID"); correlationID == "" {
		t.Fatal("response has no correlation ID")
	}
	httpCorrelationID := response.Header().Get("X-Correlation-ID")

	parentContext, parent := runtime.Tracer().Start(context.Background(), "command.accept")
	carrier := runtime.Inject(parentContext)
	parent.End()
	jobContext, job := runtime.StartJob(
		telemetry.ContextWithCorrelationID(
			context.Background(),
			"CORRELATION-JOB-001",
		),
		carrier,
		"scan",
		"disabled",
	)
	runtime.RecordJobAttempt(jobContext, "scan", "disabled", "succeeded")
	runtime.RecordOutboxReadyAge(
		jobContext,
		"scan",
		"evidence",
		"succeeded",
		42*time.Second,
	)
	job.End()

	queryContext := runtime.PostgresTracer("findings").TraceQueryStart(
		context.Background(),
		nil,
		pgx.TraceQueryStartData{SQL: "SELECT * FROM findings WHERE id = $1"},
	)
	runtime.PostgresTracer("findings").TraceQueryEnd(
		queryContext,
		nil,
		pgx.TraceQueryEndData{},
	)

	if err := runtime.ForceFlush(context.Background()); err != nil {
		t.Fatalf("ForceFlush() error = %v", err)
	}
	spans := exporter.GetSpans()
	byName := make(map[string]tracetest.SpanStub, len(spans))
	postgresWithHTTPCorrelation := false
	for _, span := range spans {
		byName[span.Name] = span
		if span.Name == "db.client.operation" &&
			spanAttribute(span, "correlation.id") == httpCorrelationID {
			postgresWithHTTPCorrelation = true
		}
		for _, attribute := range span.Attributes {
			if strings.Contains(attribute.Value.AsString(), "FINDING-PRIVATE-ID") {
				t.Fatalf("span %q leaked a record ID", span.Name)
			}
		}
	}
	serverSpan, ok := byName["http.server.request"]
	if !ok {
		t.Fatal("HTTP server span is missing")
	}
	if got := serverSpan.Parent.SpanID().String(); got != "00f067aa0ba902b7" {
		t.Fatalf("server parent span ID = %q", got)
	}
	if serverSpan.SpanKind != trace.SpanKindServer {
		t.Fatalf("HTTP span kind = %s", serverSpan.SpanKind)
	}
	databaseSpan, ok := byName["db.client.operation"]
	if !ok {
		t.Fatal("PostgreSQL span is missing")
	}
	if databaseSpan.SpanKind != trace.SpanKindClient {
		t.Fatalf("PostgreSQL span kind = %s", databaseSpan.SpanKind)
	}
	if !postgresWithHTTPCorrelation {
		t.Fatalf("no PostgreSQL span carried correlation.id %q", httpCorrelationID)
	}
	jobSpan, ok := byName["worker.job.process"]
	if !ok || len(jobSpan.Links) != 1 {
		t.Fatalf("job links = %+v", jobSpan.Links)
	}
	if jobSpan.SpanKind != trace.SpanKindConsumer {
		t.Fatalf("job span kind = %s", jobSpan.SpanKind)
	}
	if jobSpan.Links[0].SpanContext.TraceID() != byName["command.accept"].SpanContext.TraceID() {
		t.Fatal("job link does not point at the command trace")
	}
	if got := spanAttribute(jobSpan, "correlation.id"); got != "CORRELATION-JOB-001" {
		t.Fatalf("job correlation.id = %q", got)
	}

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	metricNames := collectedMetricNames(metrics)
	for _, required := range []string{
		"http.server.duration",
		"db.client.operation.duration",
		"outbox.ready.age",
		"worker.job.attempts",
	} {
		if !metricNames[required] {
			t.Fatalf("metric %q is missing", required)
		}
	}
}

func TestTelemetrySanitizesErrorsAndCollectorFailureDoesNotFailRequests(t *testing.T) {
	t.Parallel()

	if got := telemetry.ErrorClass(errors.New("password=secret evidence bytes")); got != "internal" {
		t.Fatalf("ErrorClass() = %q", got)
	}
	if got := telemetry.ErrorClass(context.DeadlineExceeded); got != "deadline_exceeded" {
		t.Fatalf("deadline ErrorClass() = %q", got)
	}

	runtime, err := telemetry.NewRuntime(context.Background(), telemetry.Config{
		ServiceName:      "api",
		ServiceVersion:   "test",
		Environment:      "integration",
		OTLPHTTPEndpoint: "http://127.0.0.1:1",
		ExportTimeout:    20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewRuntime() with unavailable collector error = %v", err)
	}
	handler := runtime.HTTPMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusAccepted)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/findings", nil))
	if response.Code != http.StatusAccepted {
		t.Fatalf("collector failure changed response status to %d", response.Code)
	}
	_ = runtime.Shutdown(context.Background())
}

func TestOutboxPersistsW3CContextAndWorkerCreatesALink(t *testing.T) {
	pool := createTestDatabase(t, "telemetry_outbox")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	exporter := tracetest.NewInMemoryExporter()
	runtime, err := telemetry.NewRuntime(context.Background(), telemetry.Config{
		ServiceName:    "telemetry-pipeline",
		ServiceVersion: "test",
		Environment:    "integration",
		SpanExporter:   exporter,
		MetricReader:   metric.NewManualReader(),
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	commandContext, commandSpan := runtime.Tracer().Start(
		context.Background(),
		"command.accept",
	)
	if err := database.WithinTransaction(
		commandContext,
		pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			_, err := transaction.Exec(ctx, `
				INSERT INTO outbox_messages (
					id, topic, aggregate_type, aggregate_id, payload,
					correlation_id
				) VALUES (
					'OUTBOX-TELEMETRY-001', 'evidence.scan_requested',
					'EVIDENCE_VERSION', 'EV-TELEMETRY-001', '{}'::jsonb,
					'CORRELATION-OUTBOX-001'
				)
			`)
			return err
		},
	); err != nil {
		t.Fatalf("insert traced outbox message: %v", err)
	}
	commandSpan.End()

	var traceParent, correlationID string
	if err := pool.QueryRow(
		context.Background(),
		"SELECT traceparent, correlation_id FROM outbox_messages WHERE id = 'OUTBOX-TELEMETRY-001'",
	).Scan(&traceParent, &correlationID); err != nil {
		t.Fatalf("read outbox traceparent: %v", err)
	}
	if !strings.HasPrefix(traceParent, "00-") {
		t.Fatalf("outbox traceparent = %q", traceParent)
	}

	workerContext, workerSpan := telemetry.StartPersistedJob(
		runtime.Context(context.Background()),
		traceParent,
		correlationID,
		"scan",
		"disabled",
	)
	runtime.RecordJobAttempt(workerContext, "scan", "disabled", "succeeded")
	workerSpan.End()
	if err := runtime.ForceFlush(context.Background()); err != nil {
		t.Fatalf("ForceFlush() error = %v", err)
	}

	var commandTraceID string
	var workerLinks []string
	for _, span := range exporter.GetSpans() {
		switch span.Name {
		case "command.accept":
			commandTraceID = span.SpanContext.TraceID().String()
		case "worker.job.process":
			for _, link := range span.Links {
				workerLinks = append(
					workerLinks,
					link.SpanContext.TraceID().String(),
				)
			}
		}
	}
	if commandTraceID == "" || len(workerLinks) != 1 ||
		workerLinks[0] != commandTraceID {
		t.Fatalf("command trace %q, worker links %v", commandTraceID, workerLinks)
	}
	for _, span := range exporter.GetSpans() {
		if span.Name == "worker.job.process" &&
			spanAttribute(span, "correlation.id") != "CORRELATION-OUTBOX-001" {
			t.Fatalf(
				"worker correlation.id = %q",
				spanAttribute(span, "correlation.id"),
			)
		}
	}
}

func collectedMetricNames(metrics metricdata.ResourceMetrics) map[string]bool {
	names := map[string]bool{}
	for _, scope := range metrics.ScopeMetrics {
		for _, instrument := range scope.Metrics {
			names[instrument.Name] = true
		}
	}
	return names
}

func spanAttribute(span tracetest.SpanStub, name string) string {
	for _, attribute := range span.Attributes {
		if string(attribute.Key) == name {
			return attribute.Value.AsString()
		}
	}
	return ""
}
