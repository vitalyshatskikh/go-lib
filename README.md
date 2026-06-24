# Go-lib

[![codecov](https://codecov.io/gh/vitalyshatskikh/go-lib/branch/main/graph/badge.svg)](https://codecov.io/gh/vitalyshatskikh/go-lib)
[![Go](https://img.shields.io/github/go-mod/go-version/vitalyshatskikh/go-lib)](https://github.com/vitalyshatskikh/go-lib)

Shared Go library providing common utilities for Go services: environment-based configuration, graceful shutdown, structured logging, Prometheus metrics, OpenTelemetry tracing, and a chi-based HTTP server.

## Example application

To run example setup:

```shell
git clone github.com/vitalyshatskikh/go-lib
cd go-lib
docker compose up -d
```

URLs:
- [Example endpoint](http://localhost:18080/api/hello)
- [Health check](http://localhost:18080/ping)
- [Metrics](http://localhost:18081/metrics)
- [Grafana](http://localhost:13000/dashboards)

## Installation

```shell
go get github.com/vitalyshatskikh/go-lib
```

## Packages

### `config`

Application configuration loaded from environment variables using `cleanenv` struct tags. Sub-configs use env prefixes (`APP_`, `API_`, `METRICS_`, `LOGGING_`, `TELEMETRY_`) with sensible defaults.

```go
cfg, err := config.Load()
// or with .env file
cfg, err := config.LoadFromEnvFile(".env")
```

### `closer`

Graceful shutdown manager. Runs multiple `func(ctx) error` closers concurrently with a configurable timeout, coalesces errors, and guards against double-close.

```go
cl := closer.New(5 * time.Second)
cl.Add(srv.Shutdown)
cl.Add(metricsShutdown)
// ...
// On SIGINT/SIGTERM:
err := cl.Close()
```

### `observability`

Factory functions to initialize observability subsystems. Each returns a shutdown function suitable for `closer.Closer`.

- `InitLogger(cfg)` — creates a zap JSON logger (stdout, ISO8601 timestamps)
- `InitMetrics(cfg, logger)` — optional Prometheus HTTP server on a separate port
- `InitTelemetry(ctx, cfg, logger)` — optional OTLP gRPC exporter with trace sampling

```go
logger, _ := observability.InitLogger(cfg)
metricsCleanup, _ := observability.InitMetrics(cfg, logger)
telemetryCleanup, _ := observability.InitTelemetry(ctx, cfg, logger)
```

### `http/restapi`

Chi-based HTTP server with a built-in middleware stack: zap request logging, Prometheus metrics, OpenTelemetry tracing, and panic recovery. Includes a `/ping` health endpoint and optional `/debug/pprof`.

```go
srv, _ := restapi.New(cfg, logger, restapi.SubRoute{
    Prefix: "/api",
    Handler: myRouter,
})
go srv.Start()
// ...
srv.Shutdown(ctx)
```

## Quick Start

See [`examples/restapi/`](examples/restapi/) for a complete application lifecycle example.

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os/signal"
    "syscall"
    "time"

    "go.uber.org/zap"

    "github.com/vitalyshatskikh/go-lib/closer"
    "github.com/vitalyshatskikh/go-lib/config"
    "github.com/vitalyshatskikh/go-lib/http/restapi"
    "github.com/vitalyshatskikh/go-lib/observability"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    logger, err := observability.InitLogger(cfg)
    if err != nil {
        log.Fatal(err)
    }

    cl := closer.New(5 * time.Second)

    metricsCleanup, _ := observability.InitMetrics(cfg, logger)
    cl.Add(metricsCleanup)

    telemetryCleanup, _ := observability.InitTelemetry(ctx, cfg, logger)
    cl.Add(telemetryCleanup)

    srv, _ := restapi.New(cfg, logger)
    cl.Add(srv.Shutdown)

    go func() {
        if err := srv.Start(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("server error", zap.Error(err))
        }
    }()

    <-ctx.Done()
    logger.Info("shutting down...")
    _ = cl.Close()
}
```

## Configuration Reference

| Env Var | Default | Description |
|---|---|---|
| `APP_NAME` | `app` | Service name |
| `APP_VERSION` | `dev` | Service version |
| `APP_ENVIRONMENT` | `development` | Environment name |
| `API_HOST` | `0.0.0.0` | HTTP server host |
| `API_PORT` | `8080` | HTTP server port |
| `METRICS_ENABLED` | `false` | Enable Prometheus metrics server |
| `METRICS_HOST` | `0.0.0.0` | Metrics server host |
| `METRICS_PORT` | `9090` | Metrics server port |
| `METRICS_PATH` | `/metrics` | Metrics endpoint path |
| `LOGGING_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `TELEMETRY_ENABLED` | `false` | Enable OpenTelemetry tracing |
| `TELEMETRY_SERVICE_NAME` | `app` | OTLP service name |
| `TELEMETRY_TRACING_ENDPOINT` | `localhost:4317` | OTLP gRPC endpoint |
| `TELEMETRY_SAMPLE_RATE` | `1.0` | Trace sampling rate (0.0–1.0) |
| `DEBUG` | `false` | Enable debug endpoints (pprof) |

## View Metrics and Traces

### OTel-LGTM Stack

The repository includes a `docker-compose.yml` that runs the full observability stack using `grafana/otel-lgtm` — a single-image bundle of Tempo (traces), Loki (logs), Prometheus (metrics), and Grafana (visualization). A custom Prometheus scrape config targets the example app's metrics endpoint. Grafana dashboards are auto-provisioned.

### Custom Dashboards

Three dashboards are mounted into Grafana automatically:

- **HTTP Server Metrics** — request rate, error ratio (5xx/total), latency percentiles (p50/p95/p99), and request/response sizes, broken down by method and path.
- **Go Runtime Overview** — goroutines, OS threads, RSS/CPU/open FDs, heap in-use/idle/sys, allocation rate, GC pause duration (p50/p95/p99), next GC threshold.
- **Service Traces** — Tempo service node graph showing inter-service dependencies, plus a recent traces table queryable by TraceQL (`{ resource.service.name = "restapi-example" }`).

### Example Application

[`examples/restapi/`](examples/restapi/) implements a complete service lifecycle using this library. It exposes a `/api/hello` endpoint that simulates real-world behavior with random latency (0–1s) and varied response codes (~1% 500, ~9% 400, ~90% 200). It sends traces to the OTLP endpoint and exposes Prometheus metrics.

[`examples/loadgen/`](examples/loadgen/) is a configurable HTTP load generator that sends requests to a target URL at a given rate and reports live latency stats.

### Running the Stack

```shell
docker compose up -d
```

This starts:
1. **otel-lgtm** — the observability backend (Grafana at `http://localhost:13000`)
2. **restapi** — the example service on `:18080` (API) and `:18081` (metrics)
3. **loadgen** — sends 3 RPS to `/api/hello` for 10 minutes

Open Grafana at `http://localhost:13000` (anonymous login enabled). Navigate to **Dashboards** to view the three custom dashboards. Traces appear in the **Service Traces** dashboard or under **Explore > Tempo**.

To run load generator next time:
```shell
docker compose start loadgen
```

## License

MIT
