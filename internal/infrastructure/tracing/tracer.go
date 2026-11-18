package tracing

import (
	"context"
	"fmt"
)

type TraceProvider interface {
	Start(ctx context.Context, spanName string, attributes map[string]interface{}) (context.Context, Span)
	Shutdown(ctx context.Context) error
}

type Span interface {
	End(err error)
	AddEvent(name string, attributes map[string]interface{})
	SetAttribute(key string, value interface{})
}

type NoOpTracer struct{}

func NewNoOpTracer() TraceProvider {
	return &NoOpTracer{}
}

func (n *NoOpTracer) Start(ctx context.Context, spanName string, attributes map[string]interface{}) (context.Context, Span) {
	return ctx, &NoOpSpan{}
}

func (n *NoOpTracer) Shutdown(ctx context.Context) error {
	return nil
}

type NoOpSpan struct{}

func (n *NoOpSpan) End(err error) {
	// No-op
}

func (n *NoOpSpan) AddEvent(name string, attributes map[string]interface{}) {
	// No-op
}

func (n *NoOpSpan) SetAttribute(key string, value interface{}) {
	// No-op
}

type SimpleTracer struct {
	serviceName string
}

func NewSimpleTracer(serviceName string) TraceProvider {
	return &SimpleTracer{
		serviceName: serviceName,
	}
}

func (s *SimpleTracer) Start(ctx context.Context, spanName string, attributes map[string]interface{}) (context.Context, Span) {
	span := &SimpleSpan{
		serviceName: s.serviceName,
		spanName:    spanName,
		attributes:  attributes,
	}

	newCtx := context.WithValue(ctx, "span_name", spanName)
	return newCtx, span
}

func (s *SimpleTracer) Shutdown(ctx context.Context) error {
	return nil
}

type SimpleSpan struct {
	serviceName string
	spanName    string
	attributes  map[string]interface{}
	events      []string
}

func (s *SimpleSpan) End(err error) {
	status := "OK"
	if err != nil {
		status = "ERROR"
	}
	fmt.Printf("[TRACE] span=%s service=%s status=%s attributes=%v\n",
		s.spanName, s.serviceName, status, s.attributes)
}

func (s *SimpleSpan) AddEvent(name string, attributes map[string]interface{}) {
	s.events = append(s.events, name)
	fmt.Printf("[TRACE] event=%s attributes=%v\n", name, attributes)
}

func (s *SimpleSpan) SetAttribute(key string, value interface{}) {
	s.attributes[key] = value
}
