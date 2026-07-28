package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/telemetry"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type scriptedProcessor struct {
	results []bool
	errAt   int
	calls   int
}

type persistedJobProcessor struct {
	traceParent string
	correlation string
	processed   bool
}

func (processor *persistedJobProcessor) ProcessNext(ctx context.Context) (bool, error) {
	if processor.processed {
		return false, nil
	}
	processor.processed = true
	_, span := telemetry.StartPersistedJob(
		ctx,
		processor.traceParent,
		processor.correlation,
		"scan",
		"clamav",
	)
	span.End()
	return true, nil
}

func (processor *scriptedProcessor) ProcessNext(context.Context) (bool, error) {
	processor.calls++
	if processor.errAt > 0 && processor.calls == processor.errAt {
		return false, errors.New("scanner unavailable")
	}
	if len(processor.results) == 0 {
		return false, nil
	}
	result := processor.results[0]
	processor.results = processor.results[1:]
	return result, nil
}

func TestProcessAvailableReportsBatchCountAndFailure(t *testing.T) {
	t.Parallel()
	processor := &scriptedProcessor{results: []bool{true, true, false}}
	processed, err := processAvailable(context.Background(), processor)
	if err != nil || processed != 2 || processor.calls != 3 {
		t.Fatalf("processAvailable() = (%d, %v), calls %d", processed, err, processor.calls)
	}

	failing := &scriptedProcessor{results: []bool{true}, errAt: 2}
	processed, err = processAvailable(context.Background(), failing)
	if err == nil || processed != 1 || failing.calls != 2 {
		t.Fatalf("failing processAvailable() = (%d, %v), calls %d", processed, err, failing.calls)
	}
}

func TestProcessAvailableInstrumentedLinksEachPersistedJob(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	runtime, err := telemetry.NewRuntime(context.Background(), telemetry.Config{
		ServiceName:    "worker",
		ServiceVersion: "test",
		Environment:    "test",
		SpanExporter:   exporter,
		MetricReader:   sdkmetric.NewManualReader(),
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	commandContext, commandSpan := runtime.Tracer().Start(
		context.Background(),
		"command.accept",
	)
	traceParent := telemetry.TraceParentFromContext(commandContext)
	commandSpan.End()

	processed, err := processAvailableInstrumented(
		context.Background(),
		runtime,
		&persistedJobProcessor{
			traceParent: traceParent,
			correlation: "CORRELATION-WORKER-001",
		},
	)
	if err != nil || processed != 1 {
		t.Fatalf("processAvailableInstrumented() = (%d, %v)", processed, err)
	}
	if err := runtime.ForceFlush(context.Background()); err != nil {
		t.Fatalf("ForceFlush() error = %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("job spans = %+v", spans)
	}
	var commandTraceID string
	var jobLinks int
	for _, span := range spans {
		switch span.Name {
		case "command.accept":
			commandTraceID = span.SpanContext.TraceID().String()
		case "worker.job.process":
			if len(span.Links) == 1 &&
				span.Links[0].SpanContext.TraceID().String() == commandTraceID {
				jobLinks++
			}
		}
	}
	if commandTraceID == "" || jobLinks != 1 {
		t.Fatalf("command trace %q, linked jobs %d, spans %+v", commandTraceID, jobLinks, spans)
	}
}

func TestNewKeycloakAdminClientRequiresProductionWorkerConfiguration(t *testing.T) {
	t.Parallel()
	if client, err := newKeycloakAdminClient(config.Settings{
		Environment: "test",
	}); err != nil || client != nil {
		t.Fatalf("test worker Keycloak client = %v, err = %v", client, err)
	}
	if _, err := newKeycloakAdminClient(config.Settings{
		Environment: "production",
	}); err == nil || !strings.Contains(err.Error(), "Keycloak") {
		t.Fatalf("missing production Keycloak config error = %v", err)
	}
	client, err := newKeycloakAdminClient(config.Settings{
		Environment:                 "production",
		KeycloakAdminURL:            "http://keycloak:8080/identity",
		KeycloakRealm:               "aviasurveil360",
		KeycloakServiceClientID:     "aviasurveil360-lifecycle",
		KeycloakServiceClientSecret: "lifecycle-client-secret",
	})
	if err != nil || client == nil {
		t.Fatalf("production Keycloak client = %v, err = %v", client, err)
	}
}

func TestNewEvidenceScannerKeepsDeterministicModeOutOfProduction(t *testing.T) {
	t.Parallel()

	deterministic, err := newEvidenceScanner(config.Settings{
		Environment: "test",
		ScannerMode: "deterministic-test",
	})
	if err != nil || deterministic == nil {
		t.Fatalf("test scanner = %T, err = %v", deterministic, err)
	}

	clamAV, err := newEvidenceScanner(config.Settings{
		Environment:               "production",
		ScannerMode:               "clamav",
		ClamAVAddress:             "clamav:3310",
		ClamAVMaximumSignatureAge: 48 * time.Hour,
	})
	if err != nil || clamAV == nil {
		t.Fatalf("production scanner = %T, err = %v", clamAV, err)
	}

	if candidate, err := newEvidenceScanner(config.Settings{
		Environment: "production",
		ScannerMode: "deterministic-test",
	}); err == nil || candidate != nil {
		t.Fatalf("production deterministic scanner = %T, err = %v", candidate, err)
	}
}

func TestNewDocumentRendererRequiresGotenbergInProduction(t *testing.T) {
	t.Parallel()

	deterministic, err := newDocumentRenderer(config.Settings{
		Environment: "test",
		ScannerMode: "deterministic-test",
	})
	if err != nil || deterministic == nil {
		t.Fatalf("test document renderer = %T, err = %v", deterministic, err)
	}

	renderer, err := newDocumentRenderer(config.Settings{
		Environment:           "production",
		GotenbergURL:          "http://gotenberg:3000",
		GotenbergTimeout:      30 * time.Second,
		GotenbergRendererHash: testGotenbergRendererHash,
	})
	if err != nil || renderer == nil {
		t.Fatalf("production Gotenberg renderer = %T, err = %v", renderer, err)
	}

	if renderer, err := newDocumentRenderer(config.Settings{
		Environment: "production",
	}); err == nil || renderer != nil {
		t.Fatalf("missing production Gotenberg renderer = %T, err = %v", renderer, err)
	}

	if renderer, err := newDocumentRenderer(config.Settings{
		Environment: "development",
	}); err != nil || renderer != nil {
		t.Fatalf("optional development renderer = %T, err = %v", renderer, err)
	}
}

func TestNewNotificationSenderRequiresSMTPInProduction(t *testing.T) {
	t.Parallel()

	if sender, err := newNotificationSender(config.Settings{
		Environment: "test",
	}); err != nil || sender != nil {
		t.Fatalf("optional test SMTP sender = %T, err = %v", sender, err)
	}
	if sender, err := newNotificationSender(config.Settings{
		Environment: "production",
	}); err == nil || sender != nil ||
		!strings.Contains(err.Error(), "SMTP") {
		t.Fatalf("missing production SMTP sender = %T, err = %v", sender, err)
	}
	sender, err := newNotificationSender(config.Settings{
		Environment:        "production",
		SMTPAddress:        "mailpit:1025",
		SMTPFrom:           "no-reply@aviasurveil360.local",
		SMTPUsername:       "aviasurveil360",
		SMTPPassword:       "smtp-secret",
		SMTPTimeout:        10 * time.Second,
		SMTPPrivateNetwork: true,
	})
	if err != nil || sender == nil {
		t.Fatalf("production SMTP sender = %T, err = %v", sender, err)
	}
}

const testGotenbergRendererHash = "sha256:56c47f7b913f3b978554115a0191c4a9dcc2558f9090f27f3f13f28a7c2f8329"
