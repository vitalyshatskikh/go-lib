package restapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWithLogger_WhenLoggerProvided_ThenSetsLogger(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	testLogger := zap.NewNop()

	srv, err := New(cfg, WithLogger(testLogger))
	require.NoError(t, err)
	require.NotNil(t, srv)

	assert.Same(t, testLogger, srv.logger)
}

func TestWithLogger_WhenLoggerIsNil_ThenKeepsDefaultLogger(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()

	srv, err := New(cfg, WithLogger(nil))
	require.NoError(t, err)
	require.NotNil(t, srv)

	assert.NotNil(t, srv.logger)
}

func TestWithOpenAPI_WhenValidJSON_ThenSetsSpec(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	spec := `{"openapi":"3.0.0","info":{"title":"Test"}}`

	srv, err := New(cfg, WithOpenAPI(strings.NewReader(spec)))
	require.NoError(t, err)
	require.NotNil(t, srv)

	assert.JSONEq(t, spec, string(srv.openapiJSON))
}

func TestWithOpenAPI_WhenValidYAML_ThenConvertsToJSON(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	yamlSpec := "openapi: 3.0.0\ninfo:\n  title: Test\n"

	srv, err := New(cfg, WithOpenAPI(strings.NewReader(yamlSpec)))
	require.NoError(t, err)
	require.NotNil(t, srv)

	assert.JSONEq(t, `{"openapi":"3.0.0","info":{"title":"Test"}}`, string(srv.openapiJSON))
}

func TestWithOpenAPI_WhenInvalidSpec_ReturnsError(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()

	_, err := New(cfg, WithOpenAPI(strings.NewReader("key:\n\tvalue")))
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to parse spec")
}

func TestWithOpenAPI_WhenReaderFails_ReturnsError(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()

	_, err := New(cfg, WithOpenAPI(errReader{}))
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read spec")
}

func TestWithMiddleWares_WhenNoMiddlewares_ThenUserMiddlewaresEmpty(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()

	srv, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, srv)

	assert.Empty(t, srv.userMiddlewares)
}

func TestWithMiddleWares_WhenOneMiddleware_ThenAppendsMiddleware(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	mw := func(next http.Handler) http.Handler {
		return next
	}

	srv, err := New(cfg, WithMiddleWares(mw))
	require.NoError(t, err)
	require.NotNil(t, srv)

	require.Len(t, srv.userMiddlewares, 1)
}

func TestWithMiddleWares_WhenMultipleMiddlewares_ThenAppendsAll(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	mw1 := func(next http.Handler) http.Handler { return next }
	mw2 := func(next http.Handler) http.Handler { return next }
	mw3 := func(next http.Handler) http.Handler { return next }

	srv, err := New(cfg, WithMiddleWares(mw1, mw2, mw3))
	require.NoError(t, err)
	require.NotNil(t, srv)

	require.Len(t, srv.userMiddlewares, 3)
}

func TestWithMiddleWares_WhenMultipleCalls_ThenAppendsAll(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	mw1 := func(next http.Handler) http.Handler { return next }
	mw2 := func(next http.Handler) http.Handler { return next }

	srv, err := New(cfg, WithMiddleWares(mw1), WithMiddleWares(mw2))
	require.NoError(t, err)
	require.NotNil(t, srv)

	require.Len(t, srv.userMiddlewares, 2)
}

func TestWithMiddleWares_WhenMiddlewareSetsHeader_ThenHeaderPresentInResponse(t *testing.T) {
	resetPrometheusRegistry()

	cfg := setupTestConfig()
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "test-value")
			next.ServeHTTP(w, r)
		})
	}

	srv, err := New(cfg, WithMiddleWares(mw))
	require.NoError(t, err)
	require.NotNil(t, srv)

	subRouter := chi.NewRouter()
	subRouter.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	err = srv.Mount(SubRoute{Prefix: "/api", Handler: subRouter})
	require.NoError(t, err)

	ts := httptest.NewServer(srv.apiServer.Handler)
	t.Cleanup(ts.Close)

	resp, err := ts.Client().Get(ts.URL + "/api/hello")
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "test-value", resp.Header.Get("X-Custom"))
}
