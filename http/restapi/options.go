package restapi

import (
	"io"
	"net/http"

	"go.uber.org/zap"
)

// WithLogger sets the logger used by the server.
func WithLogger(logger *zap.Logger) ServerOption {
	return func(srv *Server) error {
		if logger != nil {
			srv.logger = logger
		}
		return nil
	}
}

// WithOpenAPI mounts an OpenAPI/Swagger UI endpoint. It accepts an OpenAPI
// spec in JSON or YAML format and serves it at /docs with Swagger UI.
func WithOpenAPI(spec io.Reader) ServerOption {
	return func(s *Server) error {
		jsonSpec, err := parseSpec(spec)
		if err != nil {
			return err
		}
		s.openapiJSON = jsonSpec
		return nil
	}
}

// WithMiddleWares appends user-defined HTTP middlewares to the server's
// middleware stack. They run after the built-in middlewares (logging,
// metrics, tracing, recoverer).
func WithMiddleWares(mws ...func(http.Handler) http.Handler) ServerOption {
	return func(srv *Server) error {
		srv.userMiddlewares = append(srv.userMiddlewares, mws...)
		return nil
	}
}
