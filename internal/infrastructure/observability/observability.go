package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/logging"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/tracing"
)

type Observability struct {
	logger    *logging.StructuredLogger
	tracer    tracing.TraceProvider
	startTime time.Time
}

func NewObservability(logger *logging.StructuredLogger, tracer tracing.TraceProvider) *Observability {
	return &Observability{
		logger:    logger,
		tracer:    tracer,
		startTime: time.Now(),
	}
}

func (o *Observability) Logger() services.Logger {
	return o.logger
}

func (o *Observability) StartSpan(ctx context.Context, operationName string) (context.Context, SpanContext) {
	newCtx, span := o.tracer.Start(ctx, operationName, map[string]interface{}{
		"operation": operationName,
		"start_time": time.Now().Unix(),
	})

	return newCtx, &spanContextImpl{
		ctx:    newCtx,
		span:   span,
		logger: o.logger.WithField("operation", operationName),
		start:  time.Now(),
	}
}

func (o *Observability) Uptime() time.Duration {
	return time.Since(o.startTime)
}

func (o *Observability) Shutdown(ctx context.Context) error {
	return o.tracer.Shutdown(ctx)
}

type SpanContext interface {
	Context() context.Context
	Logger() services.Logger
	RecordError(err error)
	RecordEvent(eventName string, attributes map[string]interface{})
	SetAttribute(key string, value interface{})
	End()
	Duration() time.Duration
}

type spanContextImpl struct {
	ctx    context.Context
	span   tracing.Span
	logger *logging.StructuredLogger
	start  time.Time
}

func (s *spanContextImpl) Context() context.Context {
	return s.ctx
}

func (s *spanContextImpl) Logger() services.Logger {
	return s.logger
}

func (s *spanContextImpl) RecordError(err error) {
	if err != nil {
		s.logger.Error("span error", err)
		s.span.AddEvent("error", map[string]interface{}{
			"message": err.Error(),
		})
	}
}

func (s *spanContextImpl) RecordEvent(eventName string, attributes map[string]interface{}) {
	s.logger.WithFields(attributes).Info(fmt.Sprintf("event: %s", eventName))
	s.span.AddEvent(eventName, attributes)
}

func (s *spanContextImpl) SetAttribute(key string, value interface{}) {
	s.logger.WithField(key, value).Debug("attribute set")
	s.span.SetAttribute(key, value)
}

func (s *spanContextImpl) End() {
	duration := time.Since(s.start)
	s.logger.WithField("duration_ms", duration.Milliseconds()).Info("span completed")
	s.span.End(nil)
}

func (s *spanContextImpl) Duration() time.Duration {
	return time.Since(s.start)
}

type UseCaseObserver struct {
	obs *Observability
}

func NewUseCaseObserver(obs *Observability) *UseCaseObserver {
	return &UseCaseObserver{obs: obs}
}

func (u *UseCaseObserver) ObserveExecution(ctx context.Context, useCaseName string, fn func(SpanContext) error) error {
	_, span := u.obs.StartSpan(ctx, useCaseName)
	defer span.End()

	span.Logger().Info("use case started")

	if err := fn(span); err != nil {
		span.RecordError(err)
		span.Logger().Error("use case failed", err)
		return err
	}

	span.Logger().Info(fmt.Sprintf("use case completed in %dms", span.Duration().Milliseconds()))
	return nil
}
