package otel

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// TracerProviderConfig configures the OpenTelemetry tracer provider
type TracerProviderConfig struct {
	ServiceName   string
	Environment   string
	Insecure     bool
	Endpoint      string
}

// NewTracerProvider creates a new OpenTelemetry tracer provider
func NewTracerProvider(cfg TracerProviderConfig) (trace.TracerProvider, func(context.Context) error) {
	var tp trace.TracerProvider

	if cfg.Insecure {
		// Insecure mode for development/testing without OTLP collector
		tp = trace.NewTracerProvider(
			trace.WithSampler(trace.AlwaysSample()),
		)
	} else {
		// Production mode with OTLP collector
		tp = otel.NewTracerProvider(
			otel.WithSampler(otel.TraceProbability(0.1)),
			otel.WithBatchProcessor(
				otlpExporter(cfg.Endpoint),
			),
		)
	}

	otel.SetTracerProvider(tp)

	return tp, otel.Shutdown
}

// otlpExporter creates an OTLP exporter
func otlpExporter(endpoint string) func(context.Context) ([]byte, error) {
	// Returns a simple exporter function
	// In production, use the actual OTLP exporter
	return func(context.Context) ([]byte, error) {
		return nil, nil
	}
}

// Tracer wraps the OTel tracer
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer creates a new tracer for the given service
func NewTracer(serviceName string) *Tracer {
	tracer := otel.Tracer(serviceName)
	return &Tracer{tracer: tracer}
}

// StartSpan starts a new span with the given name
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) trace.Span {
	return t.tracer.Start(ctx, name, opts...)
}

// LogAttributes logs attributes to the current span
func (t *Tracer) LogAttributes(ctx context.Context, attrs ...attribute.Attribute) {
	_, span := t.tracer.Start(ctx, "attributes")
	defer span.End()
	span.SetAttributes(attrs...)
}

// LogEvent logs an event to the current span
func (t *Tracer) LogEvent(ctx context.Context, msg string, opts ...trace.EventOption) {
	_, span := t.tracer.Start(ctx, "event")
	defer span.End()
	span.AddEvent(msg, opts...)
}

// TraceOperation traces a function execution with full span
func (t *Tracer) TraceOperation(ctx context.Context, name string, fn func(context.Context) error, opts ...attribute.KeyValue) error {
	ctx, span := t.tracer.Start(ctx, name)
	defer span.End()

	// Set status attribute
	span.SetAttributes(
		attribute.String("operation.name", name),
		attribute.String("system", "ASF"),
	)

	if err := fn(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// TraceRequest traces HTTP request execution
func (t *Tracer) TraceRequest(ctx context.Context, method, url string, statusCode int, duration time.Duration, err error) {
	_, span := t.tracer.Start(ctx, "http.request")
	defer span.End()

	span.SetAttributes(
		semconv.HTTPMethodKey.String(method),
		semconv.HTTPTargetKey.String(url),
		semconv.HTTPStatusCodeInt64(statusCode),
		semconv.HTTPDurationKey.Int64(duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(fmt.Errorf("request failed: %w", err))
		span.SetStatus(codes.Error, fmt.Sprintf("request failed: %w", err))
	} else {
		span.SetStatus(codes.Ok, "")
	}
}

// TraceFunction traces function execution with attributes
func (t *Tracer) TraceFunction(ctx context.Context, name string, fn func() error, attrs ...attribute.Attribute) error {
	_, span := t.tracer.Start(ctx, name)
	defer span.End()

	span.SetAttributes(attrs...)

	if err := fn(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// TraceWorkerTask traces agent worker task execution
func (t *Tracer) TraceWorkerTask(ctx context.Context, taskID, specID, taskType string) (trace.Span, context.Context) {
	return t.tracer.Start(ctx,
		"worker.task",
		trace.WithAttributes(
			attribute.String("task.id", taskID),
			attribute.String("spec.id", specID),
			attribute.String("task.type", taskType),
			attribute.String("system", "ASF"),
		),
	)
}

// TraceSpecExecution traces specification execution
func (t *Tracer) TraceSpecExecution(ctx context.Context, specID, step string) (trace.Span, context.Context) {
	return t.tracer.Start(ctx,
		"spec.execution",
		trace.WithAttributes(
			attribute.String("spec.id", specID),
			attribute.String("step", step),
			attribute.String("system", "ASF"),
		),
	)
}

// TraceIntegration traces integration adapter execution
func (t *Tracer) TraceIntegration(ctx context.Context, platform string, adapterEvent string) (trace.Span, context.Context) {
	return t.tracer.Start(ctx,
		"integration.event",
		trace.WithAttributes(
			attribute.String("integration.platform", platform),
			attribute.String("integration.event", adapterEvent),
			attribute.String("system", "ASF"),
		),
	)
}

// ShutdownOTEL shuts down the OpenTelemetry pipeline
func ShutdownOTEL() {
	otel.Shutdown()
}

// TracerKey is a context key for storing tracer instances
type TracerKey struct{}

// TracerFromContext retrieves a tracer from context
func TracerFromContext(ctx context.Context) *Tracer {
	if t, ok := ctx.Value(TracerKey{}).(*Tracer); ok {
		return t
	}
	return nil
}

// ContextWithTracer adds a tracer to context
func ContextWithTracer(ctx context.Context, t *Tracer) context.Context {
	return context.WithValue(ctx, TracerKey{}, t)
}

// TraceMiddleware creates an HTTP middleware that traces requests
func (t *Tracer) TraceMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := t.tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
			defer span.End()

			span.SetAttributes(
				semconv.HTTPMethodKey.String(r.Method),
				semconv.HTTPTargetKey.String(r.URL.Path),
				attribute.String("client.ip", r.RemoteAddr),
				attribute.String("user.agent", r.UserAgent()),
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TraceAgentExecution traces agent execution lifecycle
func (t *Tracer) TraceAgentExecution(ctx context.Context, agentID, specID, taskType string, duration time.Duration, success bool) {
	_, span := t.tracer.Start(ctx,
		"agent.execution",
		trace.WithAttributes(
			attribute.String("agent.id", agentID),
			attribute.String("spec.id", specID),
			attribute.String("task.type", taskType),
			attribute.Int64("duration.ms", duration.Milliseconds()),
			attribute.Bool("success", success),
		),
	)
	span.End()
}