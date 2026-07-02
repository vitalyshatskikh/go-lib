package restapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/http/restapi/middlewares"

	"github.com/vitalyshatskikh/go-lib/config"
)

const (
	pingPath  = "/ping"
	debugPath = "/debug"
	docsPath  = "/docs"
)

// Server wraps an HTTP server with a chi router, built-in middleware
// (zap request logging, Prometheus metrics, recoverer), a /ping health
// endpoint, and optional /debug pprof when Debug is enabled.
type Server struct {
	apiServer       *http.Server
	router          *chi.Mux
	logger          *zap.Logger
	openapiJSON     []byte
	userMiddlewares []func(http.Handler) http.Handler
}

// ServerOption configures a Server during construction.
type ServerOption func(s *Server) error

// SubRoute describes a sub-router to mount on the server.
//
// Note: ensure Handler fills `http.Request.Pattern` with actual route pattern
// - to log/meter/trace requests properly
type SubRoute struct {
	Prefix  string
	Handler http.Handler
}

// New creates a Server from config. It builds a chi router with built-in middleware.
// Returns an error if cfg is nil or any option failed to apply
func New(cfg *config.Config, options ...ServerOption) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}

	router := chi.NewRouter()

	apiHTTPServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	srv := &Server{
		apiServer: apiHTTPServer,
		router:    router,
		logger:    zap.NewNop(),
	}

	for _, opt := range options {
		err := opt(srv)
		if err != nil {
			return nil, fmt.Errorf("error applying options: %w", err)
		}
	}

	skipPathPrefixes := []string{pingPath, debugPath}
	skipFn := func(r *http.Request) bool {
		for _, prefix := range skipPathPrefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				return true
			}
		}
		return false
	}

	srv.router.NotFound(NotFoundHandler)
	srv.router.MethodNotAllowed(MethodNotAllowedHandler)
	srv.router.Use(
		middleware.RequestLogger(LogFormatter{Logger: srv.logger, Skip: skipFn}),
		middlewares.NewPrometheusMiddleware(middlewares.PrometheusMiddlewareConfig{Skip: skipFn}),
		middlewares.NewOtelTracingMiddleware(middlewares.OtelTracingMiddlewareConfig{
			ServiceName: cfg.Telemetry.ServiceName,
			Skip:        skipFn,
		}),
		middleware.Recoverer,
	)
	srv.router.Use(srv.userMiddlewares...)

	srv.router.Get(pingPath, PingHandler(cfg))

	if cfg.Debug {
		srv.logger.Info("enabling profiler")
		router.Mount(debugPath, middleware.Profiler())
	}

	if srv.openapiJSON != nil {
		srv.logger.Info("enabling openapi endpoint")
		srv.router.Mount(docsPath, OpenAPIHandler(srv.openapiJSON, srv.router.NotFoundHandler()))
	}

	return srv, nil
}

// Mount adds provided subroutes. Returns an error if subroute prefix is empty or handler is nil
func (s *Server) Mount(subroutes ...SubRoute) error {
	for _, route := range subroutes {
		if route.Prefix == "" {
			return fmt.Errorf("subroute prefix must not be empty")
		}
		if route.Handler == nil {
			return fmt.Errorf("subroute handler must not be nil, prefix: %s", route.Prefix)
		}
		s.router.Mount(route.Prefix, route.Handler)
	}
	return nil
}

// Run begins listening and serving HTTP requests. Blocks until the
// server is shut down or an error occurs.
func (s *Server) Run() error {
	s.logger.Info("starting api server",
		zap.String("addr", s.apiServer.Addr),
	)

	if err := s.apiServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("api server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down api server")

	err := s.apiServer.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	return nil
}
