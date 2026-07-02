# Go-lib

[![codecov](https://codecov.io/gh/vitalyshatskikh/go-lib/branch/main/graph/badge.svg)](https://codecov.io/gh/vitalyshatskikh/go-lib)
[![Go](https://img.shields.io/github/go-mod/go-version/vitalyshatskikh/go-lib)](https://github.com/vitalyshatskikh/go-lib)

Shared Go library providing common utilities for Go services: environment-based configuration, graceful shutdown, structured logging, Prometheus metrics, OpenTelemetry tracing, a chi-based HTTP server, and PostgreSQL connection pooling with observability.

## Example application

To run example setup:

```shell
git clone github.com/vitalyshatskikh/go-lib
cd go-lib
docker compose up -d
```

URLs:
- [Swagger (OpenAPI)](http://localhost:18080/docs)
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

Application configuration loaded from environment variables using `cleanenv` struct tags. Sub-configs use env prefixes (`APP_`, `API_`, `METRICS_`, `LOGGING_`, `TELEMETRY_`, `POSTGRES_`) with sensible defaults.

Includes `SecretStr` — a string type that masks its value in logs and serialization (`"******"`) while exposing the actual value via `.Value()`.

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

All metrics are partitioned by `status_code`, `method`, `host`, and `path` labels:

| Metric | Type | Description |
|--------|------|-------------|
| `http_server_requests_total` | Counter | Total HTTP requests processed |
| `http_server_request_duration_seconds` | Histogram | Request latency in seconds |
| `http_server_response_size_bytes` | Histogram | Response body size in bytes |
| `http_server_request_size_bytes` | Histogram | Request body size in bytes |

```go
srv, _ := restapi.New(cfg, logger, restapi.SubRoute{
    Prefix: "/api",
    Handler: myRouter,
})
go srv.Start()
// ...
srv.Shutdown(ctx)
```

### `database/postgres`

Observable PostgreSQL connection pool using `pgx/v5`. Creates a [`pgxpool.Pool`](https://github.com/jackc/pgx) with built-in:

- **OpenTelemetry tracing** — spans for queries with semantic attributes
- **Slow query logging** — configurable threshold logs warnings for queries/batches/copies
- **Prometheus metrics** — custom collector exposing pool stats with a `db_name` label:

| Metric | Type | Description |
|--------|------|-------------|
| `pg_pool_max_conns` | Gauge | Maximum number of connections |
| `pg_pool_total_conns` | Gauge | Total number of connections |
| `pg_pool_acquired_conns` | Gauge | Currently acquired connections |
| `pg_pool_idle_conns` | Gauge | Idle connections |
| `pg_pool_constructing_conns` | Gauge | Connections being established |
| `pg_pool_acquire_total` | Counter | Total acquire operations |
| `pg_pool_acquire_duration_seconds_total` | Counter | Cumulative acquire wait time |

```go
pool, err := postgres.NewPGXPool(cfg.Postgres, logger)
if err != nil {
    logger.Fatal("failed to create pool", zap.Error(err))
}
defer pool.Close()
```

Config supports structured fields (`Hosts`, `User`, `Password`, `Database`, `SSLMode`, `TargetSessionAttrs`) or a raw `DSN`. The `ConnString()` method builds a PostgreSQL URL from the structured fields.

## Quick Start

See [`examples/restapi/`](examples/restapi/) for a complete application lifecycle example including database setup with `database/postgres`.

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

| Env Var                         | Default                           | Description                                                        |
|---------------------------------|-----------------------------------|--------------------------------------------------------------------|
| `APP_NAME`                      | `my-app`                          | Service name                                                       |
| `APP_VERSION`                   | `0.1.0`                           | Service version                                                    |
| `APP_ENVIRONMENT`               | `development`                     | Environment name                                                   |
| `API_HOST`                      | `0.0.0.0`                         | HTTP server host                                                   |
| `API_PORT`                      | `8080`                            | HTTP server port                                                   |
| `METRICS_ENABLED`               | `true`                            | Enable Prometheus metrics server                                   |
| `METRICS_HOST`                  | `0.0.0.0`                         | Metrics server host                                                |
| `METRICS_PORT`                  | `8081`                            | Metrics server port                                                |
| `METRICS_PATH`                  | `/metrics`                        | Metrics endpoint path                                              |
| `LOGGING_LEVEL`                 | `info`                            | Log level (debug, info, warn, error)                               |
| `LOGGING_ADD_CALLER`            | `false`                           | Annotate log message with the filename, line and function name     |
| `TELEMETRY_ENABLED`             | `false`                           | Enable OpenTelemetry tracing                                       |
| `TELEMETRY_SERVICE_NAME`        | `my-app`                          | OTLP service name                                                  |
| `TELEMETRY_TRACING_ENDPOINT`    | `localhost:4317`                  | OTLP gRPC endpoint                                                 |
| `TELEMETRY_SAMPLE_RATE`         | `1.0`                             | Trace sampling rate (0.0–1.0)                                      |
| `POSTGRES_DSN`                  | `""`                              | Raw DSN (overrides structured fields)                              |
| `POSTGRES_HOSTS`                | `localhost:15432,localhost:15433` | Comma-separated host:port list                                     |
| `POSTGRES_USER`                 | `postgres`                        | Database user                                                      |
| `POSTGRES_PASSWORD`             | `postgres`                        | Database password (masked as `SecretStr`)                          |
| `POSTGRES_DATABASE`             | `postgres`                        | Database name                                                      |
| `POSTGRES_SSLMODE`              | `prefer`                          | SSL mode (disable, allow, prefer, require, verify-ca, verify-full) |
| `POSTGRES_TARGET_SESSION_ATTRS` | `primary`                         | Session target (primary, standby, prefer-standby)                  |
| `POSTGRES_MAX_CONNS`            | `10`                              | Max pool connections                                               |
| `POSTGRES_MIN_CONNS`            | `0`                               | Min pool connections                                               |
| `POSTGRES_MAX_CONN_LIFETIME`    | `1h`                              | Max connection lifetime                                            |
| `POSTGRES_MAX_CONN_IDLE_TIME`   | `30m`                             | Max connection idle time                                           |
| `POSTGRES_HEALTH_CHECK_PERIOD`  | `1m`                              | Health check interval                                              |
| `POSTGRES_SLOW_QUERY_THRESHOLD` | `0`                               | Slow query log threshold (0 = disabled)                            |
| `DEBUG`                         | `false`                           | Enable debug endpoints (pprof)                                     |

## View Metrics and Traces

### OTel-LGTM Stack

The repository includes a `docker-compose.yml` that runs the full observability stack using `grafana/otel-lgtm` — a single-image bundle of Tempo (traces), Loki (logs), Prometheus (metrics), and Grafana (visualization). A custom Prometheus scrape config targets the example app's metrics endpoint. Grafana dashboards are auto-provisioned.

### Custom Dashboards

Four dashboards are mounted into Grafana automatically:

- **HTTP Server Metrics** — request rate, error ratio (5xx/total), latency percentiles (p50/p95/p99), and request/response sizes, broken down by method and path.
- **Go Runtime Overview** — goroutines, OS threads, RSS/CPU/open FDs, heap in-use/idle/sys, allocation rate, GC pause duration (p50/p95/p99), next GC threshold.
- **PostgreSQL Pool Metrics** — pool connection stats from the `pgxpool` collector: max/total/acquired/idle/constructing connections, acquire rate, and acquire wait duration.
- **Service Traces** — Tempo service node graph showing inter-service dependencies, plus a recent traces table queryable by TraceQL (`{ resource.service.name = "restapi-example" }`).

### Example Application

[`examples/restapi/`](examples/restapi/) implements a complete service lifecycle using this library. It connects to PostgreSQL via `database/postgres`, exposes a `/api/hello` endpoint that queries the database and simulates real-world behavior with random latency (0–1s) and varied response codes (~1% 500, ~9% 400, ~90% 200). It sends traces to the OTLP endpoint and exposes Prometheus metrics.

[`examples/loadgen/`](examples/loadgen/) is a configurable HTTP load generator that sends requests to a target URL at a given rate and reports live latency stats.

### Running the Stack

```shell
docker compose up -d
```

This starts:
1. **postgres-primary** — primary PostgreSQL on `:15432` (WAL-configured for replication)
2. **postgres-replica** — streaming replica on `:15433`
3. **otel-lgtm** — the observability backend (Grafana at `http://localhost:13000`)
4. **restapi** — the example service on `:18080` (API) and `:18081` (metrics)
5. **loadgen** — sends 3 RPS to `/api/hello` for 10 minutes

Open Grafana at `http://localhost:13000` (anonymous login enabled). Navigate to **Dashboards** to view the four custom dashboards. Traces appear in the **Service Traces** dashboard or under **Explore > Tempo**.

To run load generator next time:
```shell
docker compose start loadgen
```

## License

MIT
