# Observability Guide: Logging & Tracing

This document describes the observability infrastructure integrated throughout the agentic-software-factory codebase. It provides structured logging with log levels and distributed tracing capabilities.

## Architecture Overview

The observability system is built on three core abstractions:

### 1. StructuredLogger
Provides structured logging with configurable log levels (DEBUG, INFO, WARN, ERROR).

```go
logger := logging.NewStructuredLogger("info")
logger.Info("user created", "user_id", "123", "email", "test@example.com")
logger.Error("operation failed", err, "operation", "sync")
logger.Debug("internal state", "count", 42)
```

**Features:**
- Log level filtering (only logs at or above configured level)
- Structured fields for context-aware logging
- WithContext() for automatic trace ID and span ID injection
- WithFields() and WithField() for adding contextual data
- Outputs: `[LEVEL] message | field1=value1 field2=value2`

**Log Levels:**
- DEBUG (0): Detailed debugging information
- INFO (1): General informational messages
- WARN (2): Warning messages (default threshold)
- ERROR (3): Error messages

### 2. TraceProvider
Abstracts distributed tracing with two implementations:

**NoOpTracer**: No-op implementation for disabling tracing
```go
tracer := tracing.NewNoOpTracer()
```

**SimpleTracer**: Outputs trace information to stdout for development
```go
tracer := tracing.NewSimpleTracer("agentic-factory")
ctx, span := tracer.Start(ctx, "operation-name", map[string]interface{}{
    "user_id": "123",
})
span.SetAttribute("key", "value")
span.AddEvent("event-name", map[string]interface{}{
    "data": "value",
})
span.End(nil) // or pass error
```

### 3. Observability Facade
Binds logger and tracer together, providing unified observability across the application.

```go
obs := observability.NewObservability(logger, tracer)

// Start a traced operation
ctx, span := obs.StartSpan(context.Background(), "process-job")
defer span.End()

// Log through the span context
span.Logger().Info("processing started")
span.SetAttribute("job_id", jobID)
```

### 4. UseCaseObserver
Automatically instruments use case execution with tracing and logging. This is the primary way observability is integrated into the application.

```go
observer := observability.NewUseCaseObserver(obs)

err := observer.ObserveExecution(ctx, "submit-job", func(span observability.SpanContext) error {
    span.SetAttribute("job_id", job.ID)
    return submitJobUC.Execute(ctx, job)
})
```

**What ObserveExecution does:**
1. Creates a new traced span for the operation
2. Logs operation start
3. Automatically catches and logs errors
4. Records execution duration
5. Logs operation completion

## Integration Points

### HTTP Handlers
All HTTP handlers are instrumented with observability through the UseCaseObserver:

```go
// WebhookHandler automatically traces use case execution
err := h.observer.ObserveExecution(ctx, "submit-job", func(span observability.SpanContext) error {
    span.SetAttribute("job_id", job.ID)
    return h.submitJobUC.Execute(ctx, job)
})

// JobHandler automatically traces use case execution
err := h.observer.ObserveExecution(r.Context(), "get-job-status", func(span observability.SpanContext) error {
    span.SetAttribute("job_id", jobID)
    var execErr error
    job, execErr = h.getJobStatusUC.Execute(r.Context(), jobID)
    return execErr
})
```

### Bootstrap Initialization
The observability infrastructure is fully initialized during application bootstrap:

```go
// internal/bootstrap/bootstrap.go
logger := logging.NewStructuredLogger(logLevel)
tracer := tracing.NewSimpleTracer("agentic-factory")
obs := observability.NewObservability(logger, tracer)
observer := observability.NewUseCaseObserver(obs)

// Handlers receive observer for automatic instrumentation
webhookHandler := http.NewWebhookHandler(submitJobUC, logger, observer)
jobHandler := http.NewJobHandler(getJobStatusUC, logger, observer)
```

## Configuration

### Log Level
Control logging verbosity via environment variable:

```bash
# Set to debug for detailed logging
LOG_LEVEL=debug go run ./cmd/main.go

# Set to info for standard logging (default)
LOG_LEVEL=info go run ./cmd/main.go

# Set to warn or error to reduce verbosity
LOG_LEVEL=error go run ./cmd/main.go
```

## Example Output

### With submit-job use case:
```
[INFO] initializing application | log_level=info
[INFO] use case started | operation=submit-job
[INFO] job submitted | id=gh-webhook-1234567890 type=issue-triage
[INFO] use case completed in 0ms | operation=submit-job
[INFO] span completed | operation=submit-job duration_ms=0
```

### With error:
```
[INFO] use case started | operation=get-job-status job_id=not-found
[ERROR] span error | operation=get-job-status | [error job not found]
[ERROR] use case failed | operation=get-job-status | [error job not found]
[INFO] span completed | operation=get-job-status duration_ms=0
```

## Future Enhancements

The observability infrastructure is designed for future expansion:

1. **OpenTelemetry OTLP Exporter**: Replace SimpleTracer with OTLP exporter for production tracing
2. **Span Sampling**: Configure which spans to record based on trace ID or attributes
3. **Metric Collection**: Add latency and error rate metrics
4. **Log Aggregation**: Send logs to centralized logging systems (ELK, Datadog, etc.)
5. **Distributed Context Propagation**: Support W3C Trace Context headers for microservices

## Testing

Observability is thoroughly tested:

```go
func TestUseCaseObserverSuccess(t *testing.T) {
    logger := logging.NewStructuredLogger("info")
    tracer := tracing.NewNoOpTracer()
    obs := observability.NewObservability(logger, tracer)
    observer := observability.NewUseCaseObserver(obs)

    ctx := context.Background()
    err := observer.ObserveExecution(ctx, "test-use-case", func(span SpanContext) error {
        span.SetAttribute("test", "value")
        return nil
    })

    if err != nil {
        t.Errorf("expected no error, got %v", err)
    }
}
```

## Best Practices

1. **Use SpanContext attributes for high-cardinality data**: Job IDs, user IDs, etc.
2. **Keep log messages short**: Details go in structured fields
3. **Don't log sensitive data**: Passwords, tokens, PII
4. **Use appropriate log levels**: DEBUG for internal state, INFO for business events, WARN/ERROR for issues
5. **Wrap use cases with ObserveExecution**: Ensures consistent tracing and logging

## References

- **StructuredLogger**: `internal/infrastructure/logging/structured_logger.go`
- **TraceProvider**: `internal/infrastructure/tracing/tracer.go`
- **Observability**: `internal/infrastructure/observability/observability.go`
- **Bootstrap**: `internal/bootstrap/bootstrap.go`
- **HTTP Handlers**: `internal/adapters/http/webhook_handler.go`, `internal/adapters/http/job_handler.go`
