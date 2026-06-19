package helium_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestParseContextCancelledUpFront verifies that a context cancelled before
// Parse runs aborts immediately with the context error instead of doing work.
func TestParseContextCancelledUpFront(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := helium.NewParser().Parse(ctx, []byte(`<?xml version="1.0"?><root><a/></root>`))
	require.ErrorIs(t, err, context.Canceled, "Parse must return the context error")
}

// TestParseContextCancelledDuringParse verifies that cancelling the context
// while Parse is running on a large, deeply repetitive input aborts promptly
// with the context error rather than running to completion.
func TestParseContextCancelledDuringParse(t *testing.T) {
	// Build a large document with many sibling elements so the content loop
	// iterates enough times to observe the cancellation.
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><root>`)
	for range 200000 {
		b.WriteString(`<a>x</a>`)
	}
	b.WriteString(`</root>`)
	input := []byte(b.String())

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel shortly after parsing begins.
	go func() {
		time.Sleep(2 * time.Millisecond)
		cancel()
	}()

	done := make(chan error, 1)
	go func() {
		_, err := helium.NewParser().Parse(ctx, input)
		done <- err
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled, "Parse must return the context error")
	case <-time.After(10 * time.Second):
		t.Fatal("Parse did not return promptly after context cancellation")
	}
}
