package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/logging"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/tracing"
)

func TestObservabilityStartSpan(t *testing.T) {
	logger := logging.NewStructuredLogger("debug")
	tracer := tracing.NewNoOpTracer()
	obs := NewObservability(logger, tracer)

	ctx := context.Background()
	_, span := obs.StartSpan(ctx, "test-operation")

	if span == nil {
		t.Error("expected span, got nil")
	}
}

func TestSpanContextRecordError(t *testing.T) {
	logger := logging.NewStructuredLogger("debug")
	tracer := tracing.NewSimpleTracer("test-service")
	obs := NewObservability(logger, tracer)

	ctx := context.Background()
	_, span := obs.StartSpan(ctx, "test-op")

	testErr := errors.New("test error")
	span.RecordError(testErr)
	span.End()
}

func TestSpanContextRecordEvent(t *testing.T) {
	logger := logging.NewStructuredLogger("debug")
	tracer := tracing.NewSimpleTracer("test-service")
	obs := NewObservability(logger, tracer)

	ctx := context.Background()
	_, span := obs.StartSpan(ctx, "test-op")

	span.RecordEvent("test-event", map[string]interface{}{
		"value": 42,
	})
	span.End()
}

func TestUseCaseObserverSuccess(t *testing.T) {
	logger := logging.NewStructuredLogger("info")
	tracer := tracing.NewNoOpTracer()
	obs := NewObservability(logger, tracer)
	observer := NewUseCaseObserver(obs)

	ctx := context.Background()
	err := observer.ObserveExecution(ctx, "test-use-case", func(span SpanContext) error {
		span.SetAttribute("test", "value")
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestUseCaseObserverError(t *testing.T) {
	logger := logging.NewStructuredLogger("info")
	tracer := tracing.NewNoOpTracer()
	obs := NewObservability(logger, tracer)
	observer := NewUseCaseObserver(obs)

	ctx := context.Background()
	expectedErr := errors.New("test error")
	err := observer.ObserveExecution(ctx, "test-use-case", func(span SpanContext) error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestStructuredLoggerWithFields(t *testing.T) {
	logger := logging.NewStructuredLogger("debug")
	loggerWithFields := logger.WithFields(map[string]interface{}{
		"user_id": "123",
		"action":  "create",
	})

	// Should not error
	loggerWithFields.Info("test message")
}

func TestUptime(t *testing.T) {
	logger := logging.NewStructuredLogger("info")
	tracer := tracing.NewNoOpTracer()
	obs := NewObservability(logger, tracer)

	uptime := obs.Uptime()
	if uptime < 0 {
		t.Errorf("expected positive uptime, got %v", uptime)
	}
}
