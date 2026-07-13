package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

// New builds a production-oriented Zap logger from config.
// Callers own the lifecycle and must Sync on shutdown.
func New(cfg config.LogConfig) (*zap.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	var zcfg zap.Config
	if cfg.Development {
		zcfg = zap.NewDevelopmentConfig()
	} else {
		zcfg = zap.NewProductionConfig()
	}
	zcfg.Level = zap.NewAtomicLevelAt(level)
	zcfg.Encoding = cfg.Encoding
	if cfg.Encoding == "" {
		zcfg.Encoding = "json"
	}
	zcfg.EncoderConfig.TimeKey = "timestamp"
	zcfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zcfg.EncoderConfig.MessageKey = "message"
	zcfg.EncoderConfig.LevelKey = "level"
	zcfg.EncoderConfig.CallerKey = "caller"

	log, err := zcfg.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}
	return log, nil
}

func parseLevel(s string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info", "":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown log level %q", s)
	}
}

// WithRequestID returns a child logger enriched with request_id.
func WithRequestID(log *zap.Logger, requestID string) *zap.Logger {
	return log.With(zap.String("request_id", requestID))
}

// WithProvider returns a child logger enriched with provider context.
func WithProvider(log *zap.Logger, provider string) *zap.Logger {
	return log.With(zap.String("provider", provider))
}
