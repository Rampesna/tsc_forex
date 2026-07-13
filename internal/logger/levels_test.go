package logger_test

import (
	"testing"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/logger"
)

func TestNewLoggerLevels(t *testing.T) {
	t.Parallel()
	for _, lvl := range []string{"debug", "info", "warn", "error"} {
		l, err := logger.New(config.LogConfig{Level: lvl, Encoding: "json"})
		if err != nil {
			t.Fatalf("level %s: %v", lvl, err)
		}
		_ = l.Sync()
	}
}

func TestNewLoggerConsoleDevelopment(t *testing.T) {
	t.Parallel()
	l, err := logger.New(config.LogConfig{Level: "debug", Encoding: "console", Development: true})
	if err != nil {
		t.Fatal(err)
	}
	logger.WithProvider(logger.WithRequestID(l, "r1"), "mock").Debug("dev log")
}
