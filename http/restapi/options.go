package restapi

import (
	"io"

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
