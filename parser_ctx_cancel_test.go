package helium_test

import (
	"context"
	"errors"
	"strings"
	"sync"
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

// errRecordingSAX embeds the default tree builder and records every SAX Error
// callback so the test can assert that context cancellation does NOT surface as
// a parse error to the SAX handler.
type errRecordingSAX struct {
	*helium.TreeBuilder
	mu     sync.Mutex
	errors []error
}

func (s *errRecordingSAX) Error(ctx context.Context, err error) error {
	s.mu.Lock()
	s.errors = append(s.errors, err)
	s.mu.Unlock()
	return s.TreeBuilder.Error(ctx, err)
}

func (s *errRecordingSAX) recorded() []error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]error(nil), s.errors...)
}

// TestParseContextCancelDoesNotFireSAXError verifies that when a context is
// cancelled mid-parse, Parse returns the context error and the SAX Error
// handler is NOT invoked as if the document were malformed.
func TestParseContextCancelDoesNotFireSAXError(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><root>`)
	for range 200000 {
		b.WriteString(`<a>x</a>`)
	}
	b.WriteString(`</root>`)
	input := []byte(b.String())

	handler := &errRecordingSAX{TreeBuilder: helium.NewTreeBuilder()}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(2 * time.Millisecond)
		cancel()
	}()

	done := make(chan error, 1)
	go func() {
		_, err := helium.NewParser().SAXHandler(handler).Parse(ctx, input)
		done <- err
	}()

	var err error
	select {
	case err = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Parse did not return promptly after context cancellation")
	}

	require.ErrorIs(t, err, context.Canceled, "Parse must return the context error")

	for _, e := range handler.recorded() {
		require.False(t,
			errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded),
			"SAX Error handler must not be invoked for context cancellation, got: %v", e)
	}
}
