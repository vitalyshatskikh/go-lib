package restapi

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Error  string         `json:"error"`
	Detail map[string]any `json:"detail,omitempty"`
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(Error{Error: "not found"})
}

func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(Error{Error: "method not allowed"})
}
