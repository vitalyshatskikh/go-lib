package mock

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// MockSentry records HTTP requests made by the Sentry SDK to a local test
// server. It exposes the generated DSN and a log of all captured requests.
type MockSentry struct {
	DSN   string
	Calls []string

	mx sync.RWMutex
}

// WaitCalled blocks until ms.Calls exceed 'n' or timeout is reached.
func (ms *MockSentry) WaitCalled(n int, timeout time.Duration) error {
	if n <= 0 {
		return fmt.Errorf("calls must be greater than zero")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond):
			ms.mx.RLock()
			called := len(ms.Calls)
			ms.mx.RUnlock()
			if called >= n {
				return nil
			}
		}
	}
}

// RunMockSentry starts a local HTTP test server that acts as a Sentry
// endpoint. It returns a MockSentry that records incoming requests and a
// close function to shut down the server when the test finishes.
func RunMockSentry(key, project string) (*MockSentry, func()) {
	ms := &MockSentry{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ms.mx.Lock()
		defer ms.mx.Unlock()

		call := fmt.Sprintf("%s %s", r.Method, r.RequestURI)
		ms.Calls = append(ms.Calls, call)
	})

	srv := httptest.NewServer(handler)

	ms.DSN = fmt.Sprintf("http://%s@%s/%s", key, srv.Listener.Addr().String(), project)

	return ms, srv.Close
}
