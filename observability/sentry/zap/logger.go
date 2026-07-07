package zap

import (
	"context"

	sentryzap "github.com/getsentry/sentry-go/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/vitalyshatskikh/go-lib/config"
)

// WrapLogger wraps the given zap.Logger with a Sentry core that forwards
// WarnLevel and ErrorLevel log entries to Sentry. If cfg.Sentry.Dsn is empty,
// the logger is returned unchanged.
func WrapLogger(cfg *config.Config, logger *zap.Logger) *zap.Logger {
	if cfg.Sentry.DSN == "" {
		return logger
	}

	var levels []zapcore.Level
	for _, lvl := range cfg.Sentry.Levels {
		parsed, err := zapcore.ParseLevel(lvl)
		if err != nil {
			continue
		}
		levels = append(levels, parsed)
	}
	if len(levels) == 0 {
		levels = append(levels, zapcore.ErrorLevel)
	}

	sentryCore := sentryzap.NewSentryCore(
		context.Background(),
		sentryzap.Option{
			Level:        levels,
			AddCaller:    cfg.Logging.AddCaller,
			FlushTimeout: cfg.Sentry.FlushTimeout,
		},
	)

	tee := zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, sentryCore)
	})

	return logger.WithOptions(tee)
}
