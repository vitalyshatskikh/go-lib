// Command restapi is an example REST API server that demonstrates
// full integration of config, observability, closer, REST API server,
// and PostgreSQL pool packages.
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

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/closer"
	"github.com/vitalyshatskikh/go-lib/config"
	"github.com/vitalyshatskikh/go-lib/database/postgres"
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

	pool, err := postgres.NewPGXPool(cfg.Postgres, logger)
	if err != nil {
		return fmt.Errorf("failed to create db pool: %w", err)
	}

	handler := NewHandler(&PGRepository{Pool: pool})
	err = srv.Mount(
		restapi.SubRoute{Prefix: "/", Handler: http.RedirectHandler("/docs", http.StatusFound)},
		restapi.SubRoute{Prefix: "/api", Handler: http.StripPrefix("/api", handler)},
	)
	if err != nil {
		return fmt.Errorf("failed to mount routes: %w", err)
	}

	stop := make(chan struct{}, 1)
	go func() {
		logger.Info("starting api server")
		if err := srv.Run(); err != nil {
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

func NewHandler(repo Repository) http.Handler {
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

		name, err := repo.GetName(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "Hello, %s!", name)
	})

	return mux
}

type Repository interface {
	GetName(ctx context.Context) (string, error)
}

type PGRepository struct {
	Pool *pgxpool.Pool
}

func (p *PGRepository) Close(_ context.Context) error {
	p.Pool.Close()
	return nil
}

func (p *PGRepository) GetName(ctx context.Context) (string, error) {
	conn, err := p.Pool.Acquire(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Release()

	var name string
	err = conn.QueryRow(ctx, "SELECT $1", "Kitty").Scan(&name)
	if err != nil {
		return "", err
	}

	return name, nil
}
