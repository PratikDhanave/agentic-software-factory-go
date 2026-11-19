package logging

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

type LogLevel string

const (
	DebugLevel LogLevel = "DEBUG"
	InfoLevel  LogLevel = "INFO"
	WarnLevel  LogLevel = "WARN"
	ErrorLevel LogLevel = "ERROR"
)

type Field struct {
	Key   string
	Value interface{}
}

func NewField(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

type Logger struct {
	level LogLevel
}

func New(level LogLevel) *Logger {
	if level == "" {
		level = InfoLevel
	}
	return &Logger{level: level}
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...Field) {
	if l.shouldLog(DebugLevel) {
		l.log(ctx, DebugLevel, msg, fields...)
	}
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...Field) {
	if l.shouldLog(InfoLevel) {
		l.log(ctx, InfoLevel, msg, fields...)
	}
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...Field) {
	if l.shouldLog(WarnLevel) {
		l.log(ctx, WarnLevel, msg, fields...)
	}
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...Field) {
	if l.shouldLog(ErrorLevel) {
		l.log(ctx, ErrorLevel, msg, fields...)
	}
}

func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		DebugLevel: 0,
		InfoLevel:  1,
		WarnLevel:  2,
		ErrorLevel: 3,
	}
	return levels[level] >= levels[l.level]
}

func (l *Logger) log(ctx context.Context, level LogLevel, msg string, fields ...Field) {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	sb := fmt.Sprintf("[%s] [%s] %s", timestamp, level, msg)

	if len(fields) > 0 {
		sb += " {"
		for i, f := range fields {
			if i > 0 {
				sb += ", "
			}
			sb += fmt.Sprintf("%s=%v", f.Key, f.Value)
		}
		sb += "}"
	}

	if reqID := ctx.Value("request_id"); reqID != nil {
		sb += fmt.Sprintf(" [req_id=%v]", reqID)
	}

	if userID := ctx.Value("user_id"); userID != nil {
		sb += fmt.Sprintf(" [user_id=%v]", userID)
	}

	log.New(os.Stdout, "", 0).Println(sb)
}

var defaultLogger = New(InfoLevel)

func SetDefault(logger *Logger) {
	defaultLogger = logger
}

func Debug(ctx context.Context, msg string, fields ...Field) {
	defaultLogger.Debug(ctx, msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...Field) {
	defaultLogger.Info(ctx, msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...Field) {
	defaultLogger.Warn(ctx, msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...Field) {
	defaultLogger.Error(ctx, msg, fields...)
}
