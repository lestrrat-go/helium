package xpath3

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeTimeout mimics regexp2's untyped wall-clock timeout error.
func fakeTimeout() error {
	return fmt.Errorf("match timeout after 5s on input `x`")
}

// withSpuriousTimeoutRetry distinguishes a spurious fastclock timeout (fires
// almost immediately) from a genuine ReDoS timeout (burns ~budget of wall
// time), retrying only the former. These cases pin that behavior without
// depending on regexp2's flaky background clock.
func TestWithSpuriousTimeoutRetry(t *testing.T) {
	t.Run("spurious timeout is retried then succeeds", func(t *testing.T) {
		calls := 0
		v, err := withSpuriousTimeoutRetry(time.Second, func() (int, error) {
			calls++
			if calls == 1 {
				return 0, fakeTimeout() // instant: far below budget/2
			}
			return 42, nil
		})
		require.NoError(t, err)
		require.Equal(t, 42, v)
		require.Equal(t, 2, calls, "should retry exactly once after the spurious timeout")
	})

	t.Run("genuine timeout is propagated without retry", func(t *testing.T) {
		const budget = 40 * time.Millisecond
		calls := 0
		_, err := withSpuriousTimeoutRetry(budget, func() (int, error) {
			calls++
			time.Sleep(budget) // elapsed >= budget/2 => genuine
			return 0, fakeTimeout()
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "match timeout")
		require.Equal(t, 1, calls, "a genuine timeout must not be retried")
	})

	t.Run("persistent spurious timeout gives up after max attempts", func(t *testing.T) {
		calls := 0
		_, err := withSpuriousTimeoutRetry(time.Second, func() (int, error) {
			calls++
			return 0, fakeTimeout() // always instant
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "match timeout")
		require.Equal(t, 3, calls, "should stop after maxAttempts")
	})

	t.Run("non-timeout error is returned immediately", func(t *testing.T) {
		sentinel := errors.New("boom")
		calls := 0
		_, err := withSpuriousTimeoutRetry(time.Second, func() (int, error) {
			calls++
			return 0, sentinel
		})
		require.ErrorIs(t, err, sentinel)
		require.Equal(t, 1, calls)
	})

	t.Run("zero budget disables retry", func(t *testing.T) {
		calls := 0
		_, err := withSpuriousTimeoutRetry(0, func() (int, error) {
			calls++
			return 0, fakeTimeout()
		})
		require.Error(t, err)
		require.Equal(t, 1, calls, "no budget means we cannot tell spurious from genuine")
	})
}
