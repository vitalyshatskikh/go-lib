package closer

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Closer manages graceful shutdown of multiple functions.
// It executes all registered close functions concurrently with a configurable timeout
// and coalesces errors via errors.Join. It guards against double-close.
type Closer struct {
	mx      sync.Mutex
	closed  bool
	toClose []func(ctx context.Context) error
	timeout time.Duration
}

// New creates a Closer that waits up to timeout for all close functions to complete.
func New(timeout time.Duration) *Closer {
	return &Closer{
		timeout: timeout,
	}
}

// Add registers a close function. It is safe for concurrent use.
func (c *Closer) Add(fn func(ctx context.Context) error) {
	c.mx.Lock()
	c.toClose = append(c.toClose, fn)
	c.mx.Unlock()
}

// Close executes all registered close functions concurrently.
// Each function receives a context with the configured timeout.
// On the first call it runs shutdown and returns joined errors.
// Subsequent calls are no-ops.
func (c *Closer) Close() error {
	c.mx.Lock()
	defer c.mx.Unlock()

	if c.closed {
		return nil
	}

	wg := sync.WaitGroup{}

	errs := make([]error, len(c.toClose))

	for idx, fn := range c.toClose {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
			defer cancel()

			errs[idx] = fn(ctx)
		})
	}

	wg.Wait()

	c.closed = true
	return errors.Join(errs...)
}
