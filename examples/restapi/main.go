package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/closer"
	"github.com/vitalyshatskikh/go-lib/config"
	"github.com/vitalyshatskikh/go-lib/http/restapi"
	"github.com/vitalyshatskikh/go-lib/observability"
)

//go:embed openapi.yml
var openapiYML []byte

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config", err.Error())
	}

	logger, err := observability.InitLogger(cfg)
	if err != nil {
		log.Fatal("failed to initialize logger", err.Error())
	}

	err = run(cfg, logger)
	if err != nil {
		_ = logger.Sync()
		os.Exit(1)
	}
	_ = logger.Sync()
}

func run(cfg *config.Config, logger *zap.Logger) error {
	logger.Info("starting service...")

	ctx := context.Background()
	c := closer.New(5 * time.Second) // 5 sec to shutdown
	defer func() {
		logger.Info("shutting down service...")
		if err := c.Close(); err != nil {
			logger.Error("service stopped with errors", zap.Error(err))
		}
		logger.Info("service stopped")
	}()

	shutdownTelemetry, err := observability.InitTelemetry(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	c.Add(shutdownTelemetry)

	shutdownMetrics, err := observability.InitMetrics(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}
	c.Add(shutdownMetrics)

	srv, err := restapi.New(cfg, restapi.WithLogger(logger), restapi.WithOpenAPI(bytes.NewReader(openapiYML)))
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	c.Add(srv.Shutdown)

	err = srv.Mount(
		restapi.SubRoute{Prefix: "/api", Handler: http.StripPrefix("/api", NewHandler())},
	)
	if err != nil {
		return fmt.Errorf("failed to mount routes: %w", err)
	}

	stop := make(chan struct{}, 1)
	go func() {
		logger.Info("starting api server")
		if err := srv.Start(); err != nil {
			logger.Error("api server error", zap.Error(err))
		}
		close(stop)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
	case <-stop:
	}

	return nil
}

func NewHandler() http.Handler {
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))

	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		d := randomizer.Intn(1000)
		time.Sleep(time.Duration(d) * time.Millisecond)

		x := randomizer.Intn(100)
		if x < 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if x < 10 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, Kitty!"))
	})

	return mux
}
