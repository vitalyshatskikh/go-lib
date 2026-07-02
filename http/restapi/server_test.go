package restapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vitalyshatskikh/go-lib/config"
)

func setupTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name:    "test-app",
			Version: "1.0.0",
		},
		API: config.APIConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
	}
}

func resetPrometheusRegistry() {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
}

func setupTestServer(t *testing.T, cfg *config.Config, subroutes ...SubRoute) (*Server, *httptest.Server) {
	t.Helper()

	srv, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, srv)
	err = srv.Mount(subroutes...)
	require.NoError(t, err)

	ts := httptest.NewServer(srv.apiServer.Handler)
	t.Cleanup(ts.Close)

	return srv, ts
}

func TestNew_DefaultConfig(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()

	srv, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, "127.0.0.1:8080", srv.apiServer.Addr)
}

func TestNew_WithSubroutes(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	subRouter := chi.NewRouter()
	subRouter.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("world"))
	})

	_, ts := setupTestServer(t, cfg, SubRoute{Prefix: "/api", Handler: subRouter})

	t.Run("known subroute responds correctly", func(t *testing.T) {
		resp, err := ts.Client().Get(ts.URL + "/api/hello")
		require.NoError(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "world", string(body))
	})

	t.Run("unknown subroute returns 404", func(t *testing.T) {
		resp, err := ts.Client().Get(ts.URL + "/api/nonexistent")
		require.NoError(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestNew_DebugModeEnabled(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	cfg.Debug = true

	_, ts := setupTestServer(t, cfg)

	resp, err := ts.Client().Get(ts.URL + "/debug/pprof/")
	require.NoError(t, err)
	_ = resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNew_DebugModeDisabled(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	cfg.Debug = false

	_, ts := setupTestServer(t, cfg)

	resp, err := ts.Client().Get(ts.URL + "/debug/pprof/")
	require.NoError(t, err)
	_ = resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestPingHandler_ReturnsCorrectResponse(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()

	_, ts := setupTestServer(t, cfg)

	resp, err := ts.Client().Get(ts.URL + "/ping")
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var body map[string]any
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "available", body["status"])
	assert.Equal(t, "test-app", body["service"])
	assert.Equal(t, "1.0.0", body["version"])
	assert.Equal(t, "127.0.0.1", body["hostname"])
}

func TestServer_Start_WhenServerStarted_ThenServesRequests(t *testing.T) {
	resetPrometheusRegistry()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg := setupTestConfig()
	cfg.API.Host = "127.0.0.1"
	cfg.API.Port = port

	srv, err := New(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Go(func() {
		startErr := srv.Run()
		assert.NoError(t, startErr)
	})

	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			return true
		}
		return false
	}, 3*time.Second, 50*time.Millisecond, "server failed to start in time")

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/ping", port))
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownErr := srv.Shutdown(shutdownCtx)
	assert.NoError(t, shutdownErr)
	wg.Wait()
}

func TestServer_StartAndShutdown_WhenServerRunning_ThenShutsDownGracefully(t *testing.T) {
	resetPrometheusRegistry()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg := setupTestConfig()
	cfg.API.Host = "127.0.0.1"
	cfg.API.Port = port

	srv, err := New(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = srv.Run()
	})

	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			return true
		}
		return false
	}, 3*time.Second, 50*time.Millisecond, "server failed to start in time")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	require.NoError(t, err)
	wg.Wait()

	_, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/ping", port))
	assert.Error(t, err)
}

func TestNew_WithOpenAPI_WhenInvalidSpec_ThenReturnsError(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	_, err := New(cfg, WithOpenAPI(strings.NewReader("key:\n\tvalue")))

	require.Error(t, err)
}

func TestNew_WithOpenAPI_WhenServerCreated_ThenServesSpecEndpoint(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	spec := `{"openapi":"3.0.0","info":{"title":"Test","version":"1.0.0"}}`
	srv, err := New(cfg, WithOpenAPI(strings.NewReader(spec)))
	require.NoError(t, err)

	ts := httptest.NewServer(srv.apiServer.Handler)
	t.Cleanup(ts.Close)

	resp, err := ts.Client().Get(ts.URL + "/docs/openapi.json")
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, spec, string(body))
}

func TestNew_WithOpenAPI_WhenServerCreated_ThenServesDocsUI(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	srv, err := New(cfg, WithOpenAPI(strings.NewReader(`{"openapi":"3.0.0"}`)))
	require.NoError(t, err)

	ts := httptest.NewServer(srv.apiServer.Handler)
	t.Cleanup(ts.Close)

	resp, err := ts.Client().Get(ts.URL + "/docs/")
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
}

func TestNew_WithOpenAPI_WhenNoSpec_ThenDocsNotServed(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	srv, err := New(cfg)
	require.NoError(t, err)

	ts := httptest.NewServer(srv.apiServer.Handler)
	t.Cleanup(ts.Close)

	resp, err := ts.Client().Get(ts.URL + "/docs/")
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
