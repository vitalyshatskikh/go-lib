package restapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-openapi/runtime/server-middleware/docui"
	"gopkg.in/yaml.v3"
)

const (
	specURL = "/openapi.json"
)

// OpenAPIHandler returns an http.Handler that serves the OpenAPI spec at
// /docs/openapi.json and renders Swagger UI at /docs.
func OpenAPIHandler(jsonSpec []byte, next http.Handler) http.Handler {
	router := chi.NewRouter()

	if next == nil {
		next = http.NotFoundHandler()
	}

	swaggerHandler := docui.SwaggerUI(next, docui.WithSpecURL(docsPath+specURL))

	router.Handle("/", swaggerHandler)
	router.Get(specURL, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonSpec)
	})

	return router
}

func parseSpec(spec io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec: %w", err)
	}

	if json.Valid(raw) {
		return raw, nil
	}

	var parsed any
	err = yaml.Unmarshal(raw, &parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	jsonBytes, err := json.Marshal(parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to convert spec to JSON: %w", err)
	}

	return jsonBytes, nil
}
