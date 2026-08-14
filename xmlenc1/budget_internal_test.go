package xmlenc1

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEncryptedKeyBudgetRemaining pins remaining()'s running-spend semantics:
// it starts at the limit, decreases as charge succeeds, and reports -1 for an
// unlimited or nil budget.
func TestEncryptedKeyBudgetRemaining(t *testing.T) {
	t.Parallel()

	b := &encryptedKeyBudget{limit: 10, left: 10}
	require.Equal(t, 10, b.remaining())

	require.NoError(t, b.charge(4))
	require.Equal(t, 6, b.remaining())

	require.ErrorIs(t, b.charge(7), ErrEncryptedKeyBytesExceeded)
	require.Equal(t, 6, b.remaining(), "a rejected charge must not decrement remaining")

	unlimited := &encryptedKeyBudget{limit: -1, left: -1}
	require.Equal(t, -1, unlimited.remaining())

	var nilBudget *encryptedKeyBudget
	require.Equal(t, -1, nilBudget.remaining())
}

// TestPayloadCipherValueBudgetRemaining pins remaining()'s one-shot semantics:
// unlike encryptedKeyBudget, a successful charge does NOT decrease it, because
// exactly one payload CipherValue exists per EncryptedData.
func TestPayloadCipherValueBudgetRemaining(t *testing.T) {
	t.Parallel()

	b := &payloadCipherValueBudget{limit: 10}
	require.Equal(t, 10, b.remaining())

	require.NoError(t, b.charge(4))
	require.Equal(t, 10, b.remaining(), "a one-shot charge must not decrease remaining")

	unlimited := &payloadCipherValueBudget{limit: -1}
	require.Equal(t, -1, unlimited.remaining())

	var nilBudget *payloadCipherValueBudget
	require.Equal(t, -1, nilBudget.remaining())
}

// TestBudgetWriter proves budgetWriter forwards bytes while the running total
// stays within budget.remaining(), and fails closed with the budget's own
// over-limit error the moment a Write would exceed it, without forwarding any
// byte from that call or mutating the budget's state.
func TestBudgetWriter(t *testing.T) {
	t.Parallel()

	t.Run("within limit forwards every byte", func(t *testing.T) {
		t.Parallel()

		var dst bytes.Buffer
		budget := &payloadCipherValueBudget{limit: 10}
		w := newBudgetWriter(t.Context(), &dst, budget)

		n, err := w.Write([]byte("hello"))
		require.NoError(t, err)
		require.Equal(t, 5, n)

		n, err = w.Write([]byte("world"))
		require.NoError(t, err)
		require.Equal(t, 5, n)

		require.Equal(t, "helloworld", dst.String())
		// budgetWriter itself never charges a within-limit write; the
		// budget's own state is untouched until the caller charges it.
		require.Equal(t, 10, budget.remaining())
	})

	t.Run("over limit fails closed with the budget's own error", func(t *testing.T) {
		t.Parallel()

		var dst bytes.Buffer
		budget := &payloadCipherValueBudget{limit: 4}
		w := newBudgetWriter(t.Context(), &dst, budget)

		n, err := w.Write([]byte("hello"))
		require.Zero(t, n)
		require.ErrorIs(t, err, ErrCipherValueBytesExceeded)
		require.Empty(t, dst.String(), "nothing from an over-limit write reaches dst")
	})

	t.Run("unlimited budget never fails", func(t *testing.T) {
		t.Parallel()

		var dst bytes.Buffer
		budget := &payloadCipherValueBudget{limit: -1}
		w := newBudgetWriter(t.Context(), &dst, budget)

		n, err := w.Write(bytes.Repeat([]byte("x"), 1024))
		require.NoError(t, err)
		require.Equal(t, 1024, n)
	})

	t.Run("nil budget is unlimited", func(t *testing.T) {
		t.Parallel()

		var dst bytes.Buffer
		w := newBudgetWriter(t.Context(), &dst, nil)

		n, err := w.Write([]byte("hello"))
		require.NoError(t, err)
		require.Equal(t, 5, n)
	})

	t.Run("total accumulates across writes", func(t *testing.T) {
		t.Parallel()

		var dst bytes.Buffer
		budget := &payloadCipherValueBudget{limit: 8}
		w := newBudgetWriter(t.Context(), &dst, budget)

		n, err := w.Write([]byte("hello")) // 5 bytes, within limit
		require.NoError(t, err)
		require.Equal(t, 5, n)

		// The next write's 4 bytes would push the cumulative total to 9,
		// over the 8 byte limit.
		n, err = w.Write([]byte("moar"))
		require.Zero(t, n)
		require.ErrorIs(t, err, ErrCipherValueBytesExceeded)
		require.Equal(t, "hello", dst.String(), "the earlier accepted write is unaffected")
	})

	t.Run("a done context fails closed without forwarding or charging", func(t *testing.T) {
		t.Parallel()

		var dst bytes.Buffer
		budget := &payloadCipherValueBudget{limit: 10}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		w := newBudgetWriter(ctx, &dst, budget)

		n, err := w.Write([]byte("hello"))
		require.Zero(t, n)
		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, dst.String(), "nothing from a Write on a done context reaches dst")
	})

	// A Write that is simultaneously over budget and past a done context
	// reports the context, not the budget: the context is polled first, so
	// budget.charge is never even called, which is why this is the branch
	// that wins and not merely the value abort() would have produced anyway.
	t.Run("a done context wins over an over-limit write", func(t *testing.T) {
		t.Parallel()

		var dst bytes.Buffer
		budget := &payloadCipherValueBudget{limit: 4}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		w := newBudgetWriter(ctx, &dst, budget)

		n, err := w.Write([]byte("hello"))
		require.Zero(t, n)
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, ErrCipherValueBytesExceeded)
	})

	t.Run("a nil context never stops a write", func(t *testing.T) {
		t.Parallel()

		var dst bytes.Buffer
		w := newBudgetWriter(nil, &dst, nil) //nolint:staticcheck // exercising contextErr's documented nil handling

		n, err := w.Write([]byte("hello"))
		require.NoError(t, err)
		require.Equal(t, 5, n)
	})
}
