// Command sentry-mock is a minimal HTTP server that simulates a Sentry
// ingestion endpoint. It accepts Sentry envelope POST requests at /api/,
// pretty-prints their JSON payloads for inspection, and returns a static
// success response. Useful for local testing without a real Sentry instance.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "19000"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/", handleEnvelope)

	addr := ":" + port
	log.Printf("Sentry mock server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleEnvelope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR reading body: %v", err)
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	_ = r.Body.Close()

	log.Printf("=== Sentry Envelope [%s] ===", r.URL.Path)

	for i, line := range strings.Split(strings.TrimRight(string(body), "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(line), "  ", "  "); err != nil {
			log.Printf("  [%d] %s", i, line)
		} else {
			log.Printf("  [%d]%s", i, buf.String())
		}
	}

	log.Printf("=== End Envelope ===")
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, `{"status":"ok"}`)
}
