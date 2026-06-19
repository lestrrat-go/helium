package helium_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

// largeDoc builds a document with many sibling elements so the content loop
// iterates enough times to observe a mid-parse cancellation.
func largeDoc() []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><root>`)
	for range 200000 {
		b.WriteString(`<a>x</a>`)
	}
	b.WriteString(`</root>`)
	return []byte(b.String())
}

// cancellingSAX embeds the default tree builder and cancels the parse context
// after a fixed number of StartElementNS callbacks. This makes cancellation
// deterministic: the parser is guaranteed to have entered the content loop and
// processed a known number of elements before the context is cancelled, so the
// next loop iteration observes the cancellation regardless of machine speed. It
// also records every SAX Error callback so a test can assert that a clean
// cancellation surfaces no error to the SAX handler.
type cancellingSAX struct {
	*helium.TreeBuilder
	cancel   context.CancelFunc
	cancelAt int
	mu       sync.Mutex
	starts   int
	errors   []error
}

func (s *cancellingSAX) StartElementNS(ctx context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
	s.mu.Lock()
	s.starts++
	fire := s.cancel != nil && s.starts == s.cancelAt
	s.mu.Unlock()
	if fire {
		s.cancel()
	}
	return s.TreeBuilder.StartElementNS(ctx, localname, prefix, uri, namespaces, attrs)
}

func (s *cancellingSAX) Error(ctx context.Context, err error) error {
	s.mu.Lock()
	s.errors = append(s.errors, err)
	s.mu.Unlock()
	return s.TreeBuilder.Error(ctx, err)
}

func (s *cancellingSAX) recorded() []error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]error(nil), s.errors...)
}

// TestParseContextCancelledUpFront verifies that a context cancelled before
// Parse runs aborts immediately with the context error instead of doing work.
func TestParseContextCancelledUpFront(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := helium.NewParser().Parse(ctx, []byte(`<?xml version="1.0"?><root><a/></root>`))
	require.ErrorIs(t, err, context.Canceled, "Parse must return the context error")
}

// TestParseContextCancelledDuringParse verifies that cancelling the context
// while Parse is running aborts promptly with the context error rather than
// running to completion. Cancellation is triggered deterministically from a SAX
// handler after a known number of elements have been parsed.
func TestParseContextCancelledDuringParse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	handler := &cancellingSAX{TreeBuilder: helium.NewTreeBuilder(), cancel: cancel, cancelAt: 100}

	done := make(chan error, 1)
	go func() {
		_, err := helium.NewParser().SAXHandler(handler).Parse(ctx, largeDoc())
		done <- err
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled, "Parse must return the context error")
	case <-time.After(10 * time.Second):
		t.Fatal("Parse did not return promptly after context cancellation")
	}
}

// TestParseContextCancelDoesNotFireSAXError verifies that when a context is
// cancelled mid-parse, Parse returns the context error and the SAX Error
// handler is NOT invoked at all: a clean cancellation must not look like a
// malformed document to the handler.
func TestParseContextCancelDoesNotFireSAXError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	handler := &cancellingSAX{TreeBuilder: helium.NewTreeBuilder(), cancel: cancel, cancelAt: 100}

	done := make(chan error, 1)
	go func() {
		_, err := helium.NewParser().SAXHandler(handler).Parse(ctx, largeDoc())
		done <- err
	}()

	var err error
	select {
	case err = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Parse did not return promptly after context cancellation")
	}

	require.ErrorIs(t, err, context.Canceled, "Parse must return the context error")
	require.Empty(t, handler.recorded(), "SAX Error handler must not be invoked on a clean cancellation")
}

// TestParseContextCancelWithRecoverOnError verifies that a mid-parse
// cancellation is not treated as a recoverable parse error: even with
// RecoverOnError(true) enabled, Parse must return the context error AND a nil
// document, never a partial tree.
func TestParseContextCancelWithRecoverOnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	handler := &cancellingSAX{TreeBuilder: helium.NewTreeBuilder(), cancel: cancel, cancelAt: 100}

	done := make(chan struct {
		doc *helium.Document
		err error
	}, 1)
	go func() {
		doc, err := helium.NewParser().RecoverOnError(true).SAXHandler(handler).Parse(ctx, largeDoc())
		done <- struct {
			doc *helium.Document
			err error
		}{doc, err}
	}()

	select {
	case res := <-done:
		require.ErrorIs(t, res.err, context.Canceled, "Parse must return the context error even with RecoverOnError")
		require.Nil(t, res.doc, "cancelled parse must not return a partial document")
	case <-time.After(10 * time.Second):
		t.Fatal("Parse did not return promptly after context cancellation")
	}
}

// malformedRecoverableDoc builds a large document that becomes malformed right
// after the root start tag: the content is a huge run of plain text with no
// closing tag. With RecoverOnError the parser fails on the unterminated content
// and then sits in its skip-to-recover-point loop scanning the long tail. This
// keeps recovery active long enough for a concurrently-cancelled context to be
// observed inside the recovery/skip path.
func malformedRecoverableDoc() []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><root>`)
	// A stray '<' with no following name forces a parse error; the long run of
	// text after it must be skipped during recovery (skipToRecoverPoint), which
	// is where the cancellation needs to be observed.
	b.WriteString(`<`)
	b.WriteString(strings.Repeat("x", 10_000_000))
	return []byte(b.String())
}

// TestParseContextCancelDuringRecovery verifies that cancellation observed
// while the parser is in its recovery / skip-to-recover-point path returns
// promptly with the context error and a nil document, and never blocks. The
// input is malformed so recovery is active and RecoverOnError is enabled so the
// parser would otherwise keep scanning the long tail to the end of input.
func TestParseContextCancelDuringRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct {
		doc *helium.Document
		err error
	}, 1)
	go func() {
		doc, err := helium.NewParser().RecoverOnError(true).Parse(ctx, malformedRecoverableDoc())
		done <- struct {
			doc *helium.Document
			err error
		}{doc, err}
	}()

	// Give the parser a moment to enter the recovery/skip loop, then cancel.
	time.AfterFunc(20*time.Millisecond, cancel)

	select {
	case res := <-done:
		require.ErrorIs(t, res.err, context.Canceled, "Parse must return the context error when cancelled during recovery")
		require.Nil(t, res.doc, "cancelled parse must not return a partial document")
	case <-time.After(10 * time.Second):
		t.Fatal("Parse did not return promptly after cancellation during recovery")
	}
}
