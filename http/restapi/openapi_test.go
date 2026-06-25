package restapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, fmt.Errorf("read error")
}

func TestParseSpec_WhenValidJSON_ReturnsJSON(t *testing.T) {
	spec := `{"openapi":"3.0.0","info":{"title":"Test"}}`

	result, err := parseSpec(strings.NewReader(spec))

	require.NoError(t, err)
	assert.JSONEq(t, spec, string(result))
}

func TestParseSpec_WhenValidYAML_ReturnsJSON(t *testing.T) {
	yamlSpec := "openapi: 3.0.0\ninfo:\n  title: Test\n"

	result, err := parseSpec(strings.NewReader(yamlSpec))

	require.NoError(t, err)
	assert.JSONEq(t, `{"openapi":"3.0.0","info":{"title":"Test"}}`, string(result))
}

func TestParseSpec_WhenInvalidSpec_ReturnsError(t *testing.T) {
	_, err := parseSpec(strings.NewReader("key:\n\tvalue"))

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to parse spec")
}

func TestParseSpec_WhenReaderFails_ReturnsError(t *testing.T) {
	_, err := parseSpec(errReader{})

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read spec")
}

func TestOpenAPIHandler_WhenGetSpec_ThenReturnsSpec(t *testing.T) {
	spec := `{"openapi":"3.0.0"}`
	handler := OpenAPIHandler([]byte(spec), nil)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, spec, rec.Body.String())
}

func TestOpenAPIHandler_WhenUnknownPath_ThenReturnsNotFound(t *testing.T) {
	handler := OpenAPIHandler([]byte(`{"openapi":"3.0.0"}`), nil)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
