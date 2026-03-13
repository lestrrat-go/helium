package xpath3_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXPathErrorErrorNilReceiver(t *testing.T) {
	var err *xpath3.XPathError
	require.Equal(t, "<nil XPathError>", err.Error())
}

func TestXPathErrorIs(t *testing.T) {
	t.Run("same code matches", func(t *testing.T) {
		err := &xpath3.XPathError{Code: "XPTY0004", Message: "type mismatch"}
		assert.True(t, errors.Is(err, &xpath3.XPathError{Code: "XPTY0004"}))
	})

	t.Run("different code does not match", func(t *testing.T) {
		err := &xpath3.XPathError{Code: "XPTY0004", Message: "type mismatch"}
		assert.False(t, errors.Is(err, &xpath3.XPathError{Code: "FOAR0002"}))
	})

	t.Run("empty code target does not match", func(t *testing.T) {
		err := &xpath3.XPathError{Code: "XPTY0004", Message: "type mismatch"}
		assert.False(t, errors.Is(err, &xpath3.XPathError{}))
	})

	t.Run("nil receiver does not match", func(t *testing.T) {
		var err *xpath3.XPathError
		assert.False(t, errors.Is(err, &xpath3.XPathError{Code: "XPTY0004"}))
	})

	t.Run("wrapped XPathError matches via errors.Is", func(t *testing.T) {
		inner := &xpath3.XPathError{Code: "FOAR0002", Message: "division by zero"}
		wrapped := fmt.Errorf("eval failed: %w", inner)
		assert.True(t, errors.Is(wrapped, &xpath3.XPathError{Code: "FOAR0002"}))
	})

	t.Run("errors.As extracts XPathError", func(t *testing.T) {
		inner := &xpath3.XPathError{Code: "FORG0001", Message: "invalid value"}
		wrapped := fmt.Errorf("eval failed: %w", inner)
		var xpErr *xpath3.XPathError
		require.True(t, errors.As(wrapped, &xpErr))
		assert.Equal(t, "FORG0001", xpErr.Code)
	})
}
