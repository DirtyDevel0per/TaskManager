package logger

import (
	"context"
	"fmt"
	"os"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Logger struct {
	level Level
}

func New(level string) *Logger {
	var logLevel Level

	switch level {
	case "debug":
		logLevel = DebugLevel
	case "info":
		logLevel = InfoLevel
	case "warn":
		logLevel = WarnLevel
	case "error":
		logLevel = ErrorLevel
	default:
		logLevel = InfoLevel
	}

	return &Logger{
		level: logLevel,
	}
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.level <= DebugLevel {
		fmt.Printf("[DEBUG] %s %v\n", msg, args)
	}
}

func (l *Logger) Info(msg string, args ...interface{}) {
	if l.level <= InfoLevel {
		fmt.Printf("[INFO] %s %v\n", msg, args)
	}
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.level <= WarnLevel {
		fmt.Printf("[WARN] %s %v\n", msg, args)
	}
}

func (l *Logger) Error(msg string, args ...interface{}) {
	if l.level <= ErrorLevel {
		fmt.Printf("[ERROR] %s %v\n", msg, args)
	}
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.Error(msg, args...)
	os.Exit(1)
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	return l
}
