package observability

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
)

// InitMetrics starts a Prometheus metrics HTTP server on a separate port
// when cfg.Metrics.Enabled is true. It registers the default promhttp handler
// at the configured path (default: /metrics). Returns a shutdown function
// that gracefully stops the server. If metrics are disabled, returns a no-op
// shutdown function.
func InitMetrics(cfg *config.Config, logger *zap.Logger) (func(context.Context) error, error) {
	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}

	if !cfg.Metrics.Enabled {
		return func(ctx context.Context) error { return nil }, nil
	}

	path := cfg.Metrics.Path
	if path == "" {
		path = "/metrics"
	}

	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())

	metricsHTTPServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Metrics.Host, cfg.Metrics.Port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		logger.Info("starting metrics server",
			zap.String("addr", metricsHTTPServer.Addr),
		)
		if err := metricsHTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server error", zap.Error(err))
		}
	}()

	return func(ctx context.Context) error {
		logger.Info("shutting down metrics server")
		err := metricsHTTPServer.Shutdown(ctx)
		if err != nil {
			logger.Error("failed to shutdown metrics server", zap.Error(err))
		}
		return err
	}, nil
}
