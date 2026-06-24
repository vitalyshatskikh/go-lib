package restapi

import "go.uber.org/zap"

func WithLogger(logger *zap.Logger) ServerOption {
	return func(srv *Server) error {
		if logger != nil {
			srv.logger = logger
		}
		return nil
	}
}
