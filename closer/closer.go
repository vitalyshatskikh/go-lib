package closer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

var _ io.Closer = (*Closer)(nil)

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
//
// Note: zero or negative value of timeout means that on Close() call will set already expired deadline,
// so the context passed into cleanup functions will be canceled immediately
func New(timeout time.Duration) *Closer {
	return &Closer{
		timeout: timeout,
	}
}

// Add registers a close function. It is safe for concurrent use.
//
// Note:
//   - no-op if already closed
//   - silently skips nil close function
func (c *Closer) Add(fn func(ctx context.Context) error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	if c.closed {
		return // cannot reuse
	}
	if fn == nil {
		return // just skip
	}
	c.toClose = append(c.toClose, fn)
}

// Close executes all registered close functions concurrently.
// Each function receives a context with the configured timeout.
// On the first call it runs shutdown and returns joined errors.
// Subsequent calls are no-ops.
func (c *Closer) Close() error {
	c.mx.Lock()

	if c.closed {
		c.mx.Unlock()
		return nil
	}

	c.closed = true
	c.mx.Unlock()

	// marked as 'closed' and unlocked now, so:
	// - Add() is no-op, new close functions will never appended
	// - no deadlocks even any close function try to call Add()

	// 10% time lag to allow cleanup functions handle the context cancellation
	hardTimeout := c.timeout / 100 * 110
	globalCtx, globalCancel := context.WithTimeout(context.Background(), hardTimeout)
	defer globalCancel()

	closeErrsChan := make(chan error, len(c.toClose))

	for idx, fn := range c.toClose {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					closeErrsChan <- fmt.Errorf("close fn %d panic: %v", idx, r)
				}
			}()

			ctx, cancel := context.WithTimeout(globalCtx, c.timeout)
			defer cancel()

			err := fn(ctx)
			if err != nil {
				err = fmt.Errorf("close fn %d: %w", idx, err)
			}
			closeErrsChan <- err
		}()
	}

	errs := make([]error, 0, len(c.toClose))

	for range len(c.toClose) {
		select {
		case err := <-closeErrsChan:
			errs = append(errs, err)
		case <-globalCtx.Done():
			// hard deadline! cleanup functions should be finished anyway
			errs = append(errs, fmt.Errorf("hard deadline: %w", globalCtx.Err()))
			break
		}
	}

	return errors.Join(errs...)
}
