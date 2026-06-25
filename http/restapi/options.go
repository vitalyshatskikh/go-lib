package restapi

import (
	"io"

	"go.uber.org/zap"
)

func WithLogger(logger *zap.Logger) ServerOption {
	return func(srv *Server) error {
		if logger != nil {
			srv.logger = logger
		}
		return nil
	}
}

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
