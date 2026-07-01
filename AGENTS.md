# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## ! Important !

Never mention agent's name, model and vendor in commit messages or generated code or any other materials

## Project

- Module: `github.com/vitalyshatskikh/go-lib` (Go 1.26.3)
- Shared Go library providing common utilities for typical project: graceful shutdown, env-based config, structured logging (zap), Prometheus metrics, OpenTelemetry tracing, chi-based HTTP server

## Packages

- `config/` — application config via `cleanenv` struct tags. Sub-configs use env prefixes (`APP_`, `API_`, `METRICS_`, `LOGGING_`, `TELEMETRY_`) with `env-default` values. `Load()` reads env vars; `LoadFromEnvFile(path)` reads a `.env` file and updates env vars.
- `closer/` — graceful shutdown of multiple `func(ctx) error` closers concurrently with a configurable timeout, coalesces errors via `errors.Join`, guards against double-close. Uses `wg.Go()` — verify `sync.WaitGroup.Go` semantics with Go 1.26.
- `observability/` — `InitLogger` (zap JSON encoder to stdout, ISO8601 timestamps), `InitMetrics` (optional Prometheus HTTP server on separate port, returns shutdown func), `InitTelemetry` (optional OTLP gRPC exporter with sampling, sets global `otel.SetTracerProvider`, returns shutdown func).
- `http/restapi/` — chi HTTP server wrapped in a `Server` struct with `Start()`/`Shutdown(ctx)` methods. Built-in middleware stack: zap request logger (skips `/ping`, `/debug`), Prometheus metrics (request count/duration/size), recoverer. `/ping` health endpoint. Optional `/debug` pprof when `cfg.Debug == true`.

## Commands

### Build
- `go build ./...` — build all packages
- `go mod tidy` — clean dependencies
- `go mod download` — download dependencies

### Test
- `go test ./...` — run all tests
- `go test -v ./...` — verbose
- `go test -race ./...` — with race detector
- `go test -cover ./...` — with coverage
- `go test -bench=. ./...` — benchmarks

### Lint/Format
- `go fmt ./...` — format
- `go fix ./...` — rewrite to modern Go
- `go vet ./...` — static analysis
- `golangci-lint run ./...` — if installed

## Testing Conventions

- Table-driven tests for multiple test cases
- Use Given-When-Then formula:
    - Given: Setup/preconditions
    - When: Action being tested
    - Then: Expected outcome
- Use the following naming conventions for tests:
    - `func TestFff(t *testing.T)` for functions
    - `func TestTtt_Mmm(t *testing.T)` for type methods
    - add `_WhenXxx_ThenYxx` suffix to describe action and expected outcome
- Use test helpers for common setup/teardown
- Test both success and error cases
- Use `github.com/stretchr/testify/assert` for assertions
- Use `github.com/stretchr/testify/require` for assertions that should stop test execution
- Mock external dependencies with `testify/mock`

Note: No tests currently exist in this repo. First contributions should follow these conventions.

## Architecture Notes

- **Config pattern**: config structs use `env` tags with `env-default` and `env-prefix` for namespacing. See `config/types.go` for the full schema.
- **Observability init pattern**: each `Init*` function accepts config + logger, optionally initializes the subsystem, and returns a `func(context.Context) error` shutdown function safe to pass to `closer`.

## Development Workflow

1. `go mod tidy` after adding dependencies
2. `go fmt ./...` before committing
3. `go fix ./...` then `go vet ./...` then `go test ./...` before push
