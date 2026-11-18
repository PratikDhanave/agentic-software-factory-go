package logging

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type StructuredLogger struct {
	level  string
	fields map[string]interface{}
	mu     sync.RWMutex
}

type LogLevel string

const (
	DEBUG LogLevel = "debug"
	INFO  LogLevel = "info"
	WARN  LogLevel = "warn"
	ERROR LogLevel = "error"
)

func NewStructuredLogger(level string) *StructuredLogger {
	if level == "" {
		level = "info"
	}
	return &StructuredLogger{
		level:  level,
		fields: make(map[string]interface{}),
	}
}

func (l *StructuredLogger) Info(msg string, args ...interface{}) {
	if shouldLog(l.level, "info") {
		l.logWithFields("INFO", msg, args...)
	}
}

func (l *StructuredLogger) Error(msg string, err error, args ...interface{}) {
	if shouldLog(l.level, "error") {
		allArgs := append(args, "error", err.Error())
		l.logWithFields("ERROR", msg, allArgs...)
	}
}

func (l *StructuredLogger) Debug(msg string, args ...interface{}) {
	if shouldLog(l.level, "debug") {
		l.logWithFields("DEBUG", msg, args...)
	}
}

func (l *StructuredLogger) Warn(msg string, args ...interface{}) {
	if shouldLog(l.level, "warn") {
		l.logWithFields("WARN", msg, args...)
	}
}

func (l *StructuredLogger) WithContext(ctx context.Context) *StructuredLogger {
	nl := &StructuredLogger{
		level:  l.level,
		fields: make(map[string]interface{}),
	}

	l.mu.RLock()
	for k, v := range l.fields {
		nl.fields[k] = v
	}
	l.mu.RUnlock()

	if traceID := ctx.Value("trace_id"); traceID != nil {
		nl.fields["trace_id"] = traceID
	}
	if spanID := ctx.Value("span_id"); spanID != nil {
		nl.fields["span_id"] = spanID
	}

	return nl
}

func (l *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	nl := &StructuredLogger{
		level:  l.level,
		fields: make(map[string]interface{}),
	}

	l.mu.RLock()
	for k, v := range l.fields {
		nl.fields[k] = v
	}
	l.mu.RUnlock()

	for k, v := range fields {
		nl.fields[k] = v
	}

	return nl
}

func (l *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	return l.WithFields(map[string]interface{}{key: value})
}

func (l *StructuredLogger) logWithFields(level, msg string, args ...interface{}) {
	l.mu.RLock()
	fields := l.fields
	l.mu.RUnlock()

	output := fmt.Sprintf("[%s] %s", level, msg)

	if len(fields) > 0 {
		output += " |"
		for k, v := range fields {
			output += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	if len(args) > 0 {
		output += fmt.Sprintf(" | %v", args)
	}

	log.Println(output)
}

func shouldLog(currentLevel, messageLevel string) bool {
	levels := map[string]int{
		"debug": 0,
		"info":  1,
		"warn":  2,
		"error": 3,
	}

	currentLevelInt := levels[currentLevel]
	messageLevelInt := levels[messageLevel]

	return messageLevelInt >= currentLevelInt
}
