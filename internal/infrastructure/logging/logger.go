package logging

import (
	"fmt"
	"log"
	"os"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type SimpleLogger struct {
	level string
}

func NewSimpleLogger(level string) services.Logger {
	if level == "" {
		level = "info"
	}
	return &SimpleLogger{level: level}
}

func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	if l.level != "debug" && l.level != "info" {
		return
	}
	log.Printf("[INFO] %s %v", msg, args)
}

func (l *SimpleLogger) Error(msg string, err error, args ...interface{}) {
	log.Printf("[ERROR] %s: %v %v", msg, err, args)
}

func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	if l.level != "debug" {
		return
	}
	log.Printf("[DEBUG] %s %v", msg, args)
}

func (l *SimpleLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[WARN] %s %v", msg, args)
}

type StdLogger struct{}

func NewStdLogger() services.Logger {
	return &StdLogger{}
}

func (l *StdLogger) Info(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, "[INFO] %s\n", fmt.Sprintf(msg, args...))
}

func (l *StdLogger) Error(msg string, err error, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] %s: %v\n", fmt.Sprintf(msg, args...), err)
}

func (l *StdLogger) Debug(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, "[DEBUG] %s\n", fmt.Sprintf(msg, args...))
}

func (l *StdLogger) Warn(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, "[WARN] %s\n", fmt.Sprintf(msg, args...))
}
