package logger_test

import (
	"testing"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/logger"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	log, err := logger.New(config.LogConfig{
		Level:       "info",
		Encoding:    "json",
		Development: false,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = log.Sync() }()

	child := logger.WithRequestID(log, "req-1")
	child = logger.WithProvider(child, "mock")
	child.Info("test")
}

func TestNewLoggerInvalidLevel(t *testing.T) {
	t.Parallel()

	_, err := logger.New(config.LogConfig{Level: "nope", Encoding: "json"})
	if err == nil {
		t.Fatal("expected error")
	}
}
