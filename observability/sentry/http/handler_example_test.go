package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
	"github.com/vitalyshatskikh/go-lib/observability/sentry"
	sentrymock "github.com/vitalyshatskikh/go-lib/observability/sentry/mock"
)

func ExampleWrapHandler() {
	ms, stopMock := sentrymock.RunMockSentry("testkey", "testproject")
	defer stopMock()

	cfg := &config.Config{
		Sentry: config.SentryConfig{
			DSN: config.SecretURL(ms.DSN),
		},
	}

	stopSentry, err := sentry.InitSentry(context.Background(), cfg, zap.NewNop())
	if err != nil {
		fmt.Println(err)
		return
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sentry.CaptureError(r.Context(), fmt.Errorf("some error"))
		_, _ = io.WriteString(w, "ok")
	})

	wrapped := WrapHandler(cfg, handler)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	body, _ := io.ReadAll(rr.Body)

	_ = ms.WaitCalled(1, 100*time.Millisecond)
	_ = stopSentry(context.Background())

	fmt.Printf("body: %s\n", body)
	fmt.Printf("sentry events: %d\n", len(ms.Calls))

	// Output:
	// body: ok
	// sentry events: 1
}

func ExampleWrapHandler_emptyDsn() {
	ms, stopMock := sentrymock.RunMockSentry("testkey", "testproject")
	defer stopMock()

	cfg := &config.Config{
		Sentry: config.SentryConfig{},
	}

	stopSentry, err := sentry.InitSentry(context.Background(), cfg, zap.NewNop())
	if err != nil {
		fmt.Println(err)
		return
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sentry.CaptureError(r.Context(), fmt.Errorf("some error"))
		_, _ = io.WriteString(w, "ok")
	})

	wrapped := WrapHandler(cfg, handler)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	body, _ := io.ReadAll(rr.Body)

	_ = stopSentry(context.Background())

	fmt.Printf("body: %s\n", body)
	fmt.Printf("sentry events: %d\n", len(ms.Calls))

	// Output:
	// body: ok
	// sentry events: 0
}
