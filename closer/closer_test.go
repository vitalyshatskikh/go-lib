package closer

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloser_Close_WhenClosersAdded_ThenAllExecuted(t *testing.T) {
	var counter atomic.Int32

	c := New(5 * time.Second)
	c.Add(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})
	c.Add(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})
	c.Add(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})

	err := c.Close()

	assert.NoError(t, err)
	assert.Equal(t, int32(3), counter.Load())
}

func TestCloser_Close_WhenAddedAfterClose_ThenNotExecuted(t *testing.T) {
	var counter atomic.Int32

	c := New(5 * time.Second)
	err := c.Close()
	require.NoError(t, err)

	c.Add(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})
	c.Add(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})
	c.Add(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})

	assert.Equal(t, int32(0), counter.Load())
}

func TestCloser_Close_WhenNilCloserAdded_ThenCallIsNoop(t *testing.T) {
	c := New(5 * time.Second)
	c.Add(nil)
	c.Add(nil)
	c.Add(nil)

	err := c.Close()

	assert.NoError(t, err)
}

func TestCloser_Close_WhenCalledTwice_ThenSecondCallIsNoop(t *testing.T) {
	var counter atomic.Int32

	c := New(5 * time.Second)
	c.Add(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})

	err1 := c.Close()
	err2 := c.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, int32(1), counter.Load(), "closer should only execute once")
}

func TestCloser_Close_WhenSomeClosersFail_ThenErrorsCoalesced(t *testing.T) {
	c := New(5 * time.Second)
	c.Add(func(ctx context.Context) error {
		return nil
	})
	c.Add(func(ctx context.Context) error {
		return errors.New("error from closer 2")
	})
	c.Add(func(ctx context.Context) error {
		return errors.New("error from closer 3")
	})

	err := c.Close()

	require.Error(t, err)
	assert.ErrorContains(t, err, "error from closer 2")
	assert.ErrorContains(t, err, "error from closer 3")
}

func TestCloser_Close_WhenCleanupFnStopsOnTimeout_ThenErrorDeadlineExceeded(t *testing.T) {
	c := New(50 * time.Millisecond)
	c.Add(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	})

	err := c.Close()

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "close fn 0")
}

func TestCloser_Close_WhenCleanupFnIgnoresTimeout_ThenErrorDeadlineExceeded(t *testing.T) {
	c := New(50 * time.Millisecond)
	c.Add(func(ctx context.Context) error {
		time.Sleep(5 * time.Second) // ignores context cancellation
		return nil
	})

	err := c.Close()

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "hard deadline")
}

func TestCloser_Close_WhenCleanupFnPanic_ThenNoPanic(t *testing.T) {
	c := New(50 * time.Millisecond)
	c.Add(func(ctx context.Context) error {
		panic("some-panic-message-1")
	})
	c.Add(func(ctx context.Context) error {
		panic("some-panic-message-2")
	})

	err := c.Close()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "some-panic-message-1")
	assert.Contains(t, err.Error(), "some-panic-message-2")
}

func TestCloser_Close_WhenNonPositiveTimeoutValue_ThenErrorDeadlineExceeded(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"zero", time.Duration(0)},
		{"negative", -1 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.timeout)
			c.Add(func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			})

			err := c.Close()

			require.Error(t, err)
			assert.Contains(t, err.Error(), "hard deadline")
		})
	}
}

func TestCloser_Close_WhenCleanupFnCallsAdd_ThenNoDeadlock(t *testing.T) {
	var counter atomic.Int32

	c := New(5 * time.Second)
	c.Add(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})
	c.Add(func(ctx context.Context) error {
		counter.Add(1)
		// never do this on production!
		// this cleanup fn will be ignored
		c.Add(func(ctx context.Context) error {
			counter.Add(1)
			return nil
		})
		return nil
	})

	err := c.Close()

	assert.NoError(t, err)
	assert.Equal(t, int32(2), counter.Load())
}

func TestCloser_Add_WhenCalledAfterClose_ThenNoPanic(t *testing.T) {
	c := New(5 * time.Second)

	err1 := c.Close()
	assert.NoError(t, err1)

	assert.NotPanics(t, func() {
		c.Add(func(ctx context.Context) error {
			return nil
		})
	})

	err2 := c.Close()
	assert.NoError(t, err2, "second close after add should be a no-op")
}
