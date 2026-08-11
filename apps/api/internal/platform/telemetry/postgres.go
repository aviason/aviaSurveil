package telemetry

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type postgresTraceKey struct{}

type postgresTrace struct {
	ctx       context.Context
	span      trace.Span
	startedAt time.Time
	operation string
	module    string
}

type postgresTracer struct {
	runtime *Runtime
	module  string
}

func (runtime *Runtime) PostgresTracer(module string) pgx.QueryTracer {
	return postgresTracer{runtime: runtime, module: boundedModule(module)}
}

func (tracer postgresTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	operation := firstSQLToken(data.SQL)
	spanContext, span := tracer.runtime.tracer.Start(
		ctx,
		"db.client.operation",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(
		attribute.String("db.system.name", "postgresql"),
		attribute.String("db.operation.name", operation),
		attribute.String("module", tracer.module),
	)
	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		span.SetAttributes(attribute.String("correlation.id", correlationID))
	}
	return context.WithValue(spanContext, postgresTraceKey{}, postgresTrace{
		ctx:       spanContext,
		span:      span,
		startedAt: time.Now(),
		operation: operation,
		module:    tracer.module,
	})
}

func (tracer postgresTracer) TraceQueryEnd(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	query, ok := ctx.Value(postgresTraceKey{}).(postgresTrace)
	if !ok {
		return
	}
	outcome := "succeeded"
	if data.Err != nil {
		outcome = "failed"
		query.span.SetStatus(codes.Error, ErrorClass(data.Err))
	}
	query.span.SetAttributes(attribute.String("outcome.class", outcome))
	query.span.End()
	tracer.runtime.dbDuration.Record(
		query.ctx,
		float64(time.Since(query.startedAt).Microseconds())/1000,
		metric.WithAttributes(
			attribute.String("db.system.name", "postgresql"),
			attribute.String("db.operation.name", query.operation),
			attribute.String("module", query.module),
			attribute.String("outcome.class", outcome),
		),
	)
}

func (runtime *Runtime) TracePostgres(
	ctx context.Context,
	operation string,
	module string,
	execute func(context.Context) error,
) error {
	startedAt := time.Now()
	ctx, span := runtime.tracer.Start(
		ctx,
		"db.client.operation",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(
		attribute.String("db.system.name", "postgresql"),
		attribute.String("db.operation.name", boundedDatabaseOperation(operation)),
		attribute.String("module", boundedModule(module)),
	)
	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		span.SetAttributes(attribute.String("correlation.id", correlationID))
	}
	err := execute(ctx)
	outcome := "succeeded"
	if err != nil {
		outcome = "failed"
		span.SetStatus(codes.Error, ErrorClass(err))
	}
	span.SetAttributes(attribute.String("outcome.class", outcome))
	span.End()
	runtime.dbDuration.Record(
		ctx,
		float64(time.Since(startedAt).Microseconds())/1000,
		metric.WithAttributes(
			attribute.String("db.system.name", "postgresql"),
			attribute.String("db.operation.name", boundedDatabaseOperation(operation)),
			attribute.String("module", boundedModule(module)),
			attribute.String("outcome.class", outcome),
		),
	)
	return err
}

func boundedDatabaseOperation(operation string) string {
	switch strings.ToUpper(strings.TrimSpace(operation)) {
	case "SELECT":
		return "select"
	case "INSERT":
		return "insert"
	case "UPDATE":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return "other"
	}
}

func boundedModule(module string) string {
	switch module {
	case "application", "worker", "findings", "planning", "evidence", "reports", "notifications", "identity", "outbox":
		return module
	default:
		return "other"
	}
}

func firstSQLToken(sql string) string {
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return "other"
	}
	return boundedDatabaseOperation(fields[0])
}
