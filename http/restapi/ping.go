package restapi

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/vitalyshatskikh/go-lib/config"
)

// PingHandler returns an HTTP handler that responds with service status,
// name, version and hostname as JSON. Useful for health checks.
func PingHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "undefined"
		}
		pong := map[string]any{
			"status":   "available",
			"service":  cfg.App.Name,
			"version":  cfg.App.Version,
			"hostname": hostname,
		}
		response, err := json.Marshal(pong)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(response)
	}
}
