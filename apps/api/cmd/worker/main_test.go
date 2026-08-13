package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/config"
	"github.com/aviason/aviaSurveil/internal/platform/telemetry"
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

func TestNewIdentityAdminClientRequiresProductionWorkerConfiguration(t *testing.T) {
	if client, err := newIdentityAdminClient(config.Settings{
		Environment: "test",
	}); err != nil || client != nil {
		t.Fatalf("test worker identity client = %v, err = %v", client, err)
	}
	if _, err := newIdentityAdminClient(config.Settings{
		Environment: "production",
	}); err == nil || !strings.Contains(err.Error(), "first-party") {
		t.Fatalf("missing production first-party config error = %v", err)
	}
	secretFile := filepath.Join(t.TempDir(), "admin-secret")
	if err := os.WriteFile(secretFile, []byte("0123456789abcdef0123456789abcdef"), 0o400); err != nil {
		t.Fatalf("write first-party admin secret: %v", err)
	}
	client, err := newIdentityAdminClient(config.Settings{
		Environment:               "production",
		FirstPartyAdminURL:        "http://auth:8081",
		FirstPartyAdminSecretFile: secretFile,
	})
	if err != nil || client == nil {
		t.Fatalf("production first-party client = %v, err = %v", client, err)
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

type reminderControllerScheduler struct {
	calls  atomic.Int32
	active atomic.Int32
	max    atomic.Int32
	block  <-chan struct{}
}

func (scheduler *reminderControllerScheduler) ScheduleDueReminders(ctx context.Context) (int, error) {
	active := scheduler.active.Add(1)
	defer scheduler.active.Add(-1)
	for {
		current := scheduler.max.Load()
		if active <= current || scheduler.max.CompareAndSwap(current, active) {
			break
		}
	}
	scheduler.calls.Add(1)
	if scheduler.block != nil {
		select {
		case <-scheduler.block:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	return 1, nil
}

func TestReminderControllerRunsStartupAndInjectedTicksWithoutOverlap(t *testing.T) {
	block := make(chan struct{})
	scheduler := &reminderControllerScheduler{block: block}
	ticks := make(chan time.Time, 2)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runReminderController(ctx, reminderControllerConfig{
			Interval: time.Minute, Deadline: time.Second, Ticks: ticks, Schedule: scheduler,
		})
		close(done)
	}()
	deadline := time.After(500 * time.Millisecond)
	for scheduler.calls.Load() < 1 {
		select {
		case <-deadline:
			t.Fatal("startup reminder cycle did not execute")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	ticks <- time.Now()
	if scheduler.calls.Load() != 1 {
		t.Fatal("tick overlapped the startup cycle")
	}
	close(block)
	for scheduler.calls.Load() < 2 {
		select {
		case <-deadline:
			t.Fatal("injected tick did not execute after startup cycle")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if scheduler.max.Load() != 1 {
		t.Fatalf("reminder cycles overlapped: max active %d", scheduler.max.Load())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reminder controller did not stop")
	}
}

func TestReminderControllerBoundsHungCycleAndKeepsRunning(t *testing.T) {
	scheduler := &reminderControllerScheduler{block: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runReminderController(ctx, reminderControllerConfig{
			Interval: 20 * time.Millisecond, Deadline: 5 * time.Millisecond,
			Ticks: make(chan time.Time), Schedule: scheduler,
		})
		close(done)
	}()
	select {
	case <-time.After(time.Second):
		t.Fatal("bounded reminder cycle did not reach its deadline")
	default:
	}
	time.Sleep(20 * time.Millisecond)
	if scheduler.calls.Load() != 1 {
		t.Fatalf("hung cycle unexpectedly overlapped/repeated: %d calls", scheduler.calls.Load())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("hung reminder controller did not shut down")
	}
}
