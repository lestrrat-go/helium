package catalog_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/internal/catalog"
	"github.com/stretchr/testify/require"
)

// blockingLoader blocks inside Load until released, so a test can hold a load
// in flight and exercise the cancellation path of a waiting resolve.
type blockingLoader struct {
	started  chan struct{} // closed-ish: signalled once Load has been entered
	release  chan struct{} // Load returns once this is closed
	cat      *catalog.Catalog
	calls    atomic.Int32
	signalMu sync.Mutex
	signaled bool
}

func newBlockingLoader(cat *catalog.Catalog) *blockingLoader {
	return &blockingLoader{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
		cat:     cat,
	}
}

func (l *blockingLoader) Load(ctx context.Context, _ string) (*catalog.Catalog, error) {
	l.calls.Add(1)
	l.signalMu.Lock()
	if !l.signaled {
		l.signaled = true
		close(l.started)
	}
	l.signalMu.Unlock()

	select {
	case <-l.release:
		return l.cat, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// An already-cancelled context must make Resolve return promptly instead of
// invoking the loader and blocking on a slow load.
func TestResolveAlreadyCancelledReturnsPromptly(t *testing.T) {
	t.Parallel()

	leaf := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: fooDTDSystemID, URL: "file:///foo.dtd"},
		},
	}
	loader := newBlockingLoader(leaf)

	root := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryNextCatalog, URL: "next.xml"},
		},
		Loader: loader,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before the call

	done := make(chan string, 1)
	go func() {
		done <- root.Resolve(ctx, "", fooDTDSystemID)
	}()

	select {
	case got := <-done:
		require.Equal(t, "", got, "cancelled resolve must not resolve")
	case <-time.After(2 * time.Second):
		t.Fatal("Resolve did not return promptly on an already-cancelled context")
	}
}

// A second resolution waiting on the same entry's in-flight load must return on
// context cancellation rather than blocking until the load completes.
func TestResolveWaiterReturnsOnCancellation(t *testing.T) {
	t.Parallel()

	leaf := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: fooDTDSystemID, URL: "file:///foo.dtd"},
		},
	}
	loader := newBlockingLoader(leaf)

	root := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryNextCatalog, URL: "next.xml"},
		},
		Loader: loader,
	}

	// Goroutine 1: starts the load and holds it in flight (never cancelled).
	holderCtx := t.Context()
	holderDone := make(chan struct{})
	go func() {
		defer close(holderDone)
		root.Resolve(holderCtx, "", fooDTDSystemID)
	}()

	// Wait until the load is actually in flight.
	select {
	case <-loader.started:
	case <-time.After(2 * time.Second):
		t.Fatal("loader was never entered")
	}

	// Goroutine 2: waits on the same entry's load, but its context is cancelled.
	waiterCtx, waiterCancel := context.WithCancel(t.Context())
	waiterDone := make(chan string, 1)
	go func() {
		waiterDone <- root.Resolve(waiterCtx, "", fooDTDSystemID)
	}()

	// Give the waiter a moment to actually start blocking on the load, then
	// cancel it. It must return promptly even though the load is still in
	// flight.
	time.Sleep(50 * time.Millisecond)
	waiterCancel()

	select {
	case got := <-waiterDone:
		require.Equal(t, "", got, "cancelled waiter must not resolve")
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not return on cancellation while a load was in flight")
	}

	// Release the in-flight load and let the holder finish cleanly.
	close(loader.release)
	select {
	case <-holderDone:
	case <-time.After(2 * time.Second):
		t.Fatal("holder did not finish after release")
	}

	// The single-load dedup must still hold: the entry was loaded at most once.
	require.LessOrEqual(t, loader.calls.Load(), int32(1), "entry loaded more than once")
}
