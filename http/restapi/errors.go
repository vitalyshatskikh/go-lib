package restapi

import (
	"encoding/json"
	"net/http"
)

// Error represents a JSON-encoded API error response.
type Error struct {
	Error  string         `json:"error"`
	Detail map[string]any `json:"detail,omitempty"`
}

// NotFoundHandler writes a 404 JSON response.
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(Error{Error: "not found"})
}

// MethodNotAllowedHandler writes a 405 JSON response.
func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(Error{Error: "method not allowed"})
}
