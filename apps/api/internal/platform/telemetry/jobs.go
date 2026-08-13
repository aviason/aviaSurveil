package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type TraceCarrier map[string]string
type runtimeContextKey struct{}

func (runtime *Runtime) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, runtimeContextKey{}, runtime)
}

func TraceParentFromContext(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return ""
	}
	return fmt.Sprintf(
		"00-%s-%s-%02x",
		spanContext.TraceID(),
		spanContext.SpanID(),
		byte(spanContext.TraceFlags()),
	)
}

func StartPersistedJob(
	ctx context.Context,
	traceParent string,
	correlationID string,
	jobKind string,
	adapter string,
) (context.Context, trace.Span) {
	runtime, ok := ctx.Value(runtimeContextKey{}).(*Runtime)
	if !ok || runtime == nil {
		return noop.NewTracerProvider().
			Tracer(instrumentationName).
			Start(ctx, "worker.job.process")
	}
	carrier := TraceCarrier{}
	if strings.TrimSpace(traceParent) != "" {
		carrier["traceparent"] = strings.TrimSpace(traceParent)
	}
	return runtime.StartJob(
		ContextWithCorrelationID(ctx, correlationID),
		carrier,
		jobKind,
		adapter,
	)
}

func FinishPersistedJob(
	ctx context.Context,
	span trace.Span,
	jobKind string,
	adapter string,
	err error,
) {
	outcome := "succeeded"
	if err != nil {
		outcome = "failed"
		span.SetStatus(codes.Error, ErrorClass(err))
	}
	span.SetAttributes(attribute.String("outcome.class", outcome))
	if runtime, ok := ctx.Value(runtimeContextKey{}).(*Runtime); ok && runtime != nil {
		runtime.RecordJobAttempt(ctx, jobKind, adapter, outcome)
	}
	span.End()
}

func (runtime *Runtime) Inject(ctx context.Context) TraceCarrier {
	carrier := propagation.MapCarrier{}
	runtime.propagator.Inject(ctx, carrier)
	return TraceCarrier(carrier)
}

func (runtime *Runtime) StartJob(
	ctx context.Context,
	carrier TraceCarrier,
	jobKind string,
	adapter string,
) (context.Context, trace.Span) {
	linkedContext := runtime.propagator.Extract(
		ctx,
		propagation.MapCarrier(carrier),
	)
	options := []trace.SpanStartOption{
		trace.WithNewRoot(),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("job.kind", boundedJobKind(jobKind)),
			attribute.String("adapter", boundedAdapter(adapter)),
		),
	}
	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		options = append(options, trace.WithAttributes(
			attribute.String("correlation.id", correlationID),
		))
	}
	if trace.SpanContextFromContext(linkedContext).IsValid() {
		options = append(options, trace.WithLinks(trace.LinkFromContext(linkedContext)))
	}
	return runtime.tracer.Start(ctx, "worker.job.process", options...)
}

func (runtime *Runtime) RecordJobAttempt(
	ctx context.Context,
	jobKind string,
	adapter string,
	outcome string,
) {
	runtime.jobAttempts.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("job.kind", boundedJobKind(jobKind)),
			attribute.String("adapter", boundedAdapter(adapter)),
			attribute.String("outcome.class", boundedOutcome(outcome)),
		),
	)
}

func (runtime *Runtime) RecordOutboxReadyAge(
	ctx context.Context,
	jobKind string,
	queue string,
	outcome string,
	age time.Duration,
) {
	seconds := age.Seconds()
	if seconds < 0 {
		seconds = 0
	}
	runtime.outboxReadyAge.Record(
		ctx,
		seconds,
		metric.WithAttributes(
			attribute.String("job.kind", boundedJobKind(jobKind)),
			attribute.String("queue", boundedQueue(queue)),
			attribute.String("outcome.class", boundedOutcome(outcome)),
		),
	)
}

func RecordPersistedOutboxReadyAge(
	ctx context.Context,
	jobKind string,
	queue string,
	availableAt time.Time,
	now time.Time,
) {
	runtime, ok := ctx.Value(runtimeContextKey{}).(*Runtime)
	if !ok || runtime == nil {
		return
	}
	runtime.RecordOutboxReadyAge(
		ctx,
		jobKind,
		queue,
		"ready",
		now.Sub(availableAt),
	)
}

func boundedJobKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "scan", "email", "document", "identity", "reminder":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "other"
	}
}

func boundedAdapter(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "disabled", "mailpit", "native-pdf", "postgresql":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "other"
	}
}

func boundedQueue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "evidence", "attachment", "notification", "document", "identity", "reminder":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "other"
	}
}

func boundedOutcome(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ready", "acknowledged", "succeeded", "failed", "retrying", "dead_lettered", "accepted", "duplicate", "conflict", "retryable", "retry_exhausted", "validation_rejected":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "other"
	}
}
