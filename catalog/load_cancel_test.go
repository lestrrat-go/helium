//go:build unix

package catalog_test

import (
	"context"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/catalog"
	"github.com/stretchr/testify/require"
)

// A catalog read against a FIFO with no writer blocks indefinitely. Load must
// honor ctx and abort promptly on cancellation instead of hanging forever.
func TestLoadCancelsOnBlockingFIFO(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fifo := filepath.Join(dir, "catalog.xml")
	require.NoError(t, syscall.Mkfifo(fifo, 0o600), "mkfifo")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, err := catalog.Load(ctx, fifo)
		done <- err
	}()

	// Let the load block on the open/read of the writer-less FIFO, then cancel.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.Error(t, err, "cancelled FIFO load must return an error")
	case <-time.After(3 * time.Second):
		t.Fatal("Load did not return after cancellation of a blocking FIFO read")
	}
}

// A writer-less FIFO whose open and read both block must not leave a goroutine
// (and the OS thread it occupies) parked forever after Load returns on
// cancellation. On unix the file is opened with O_NONBLOCK and the blocking read
// is interrupted via a read deadline, so no reader goroutine survives. This is
// the leak the CAT-004 residual finding described: a count taken after the load
// has drained must return to its pre-load baseline.
func TestLoadFIFONoGoroutineLeak(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fifo := filepath.Join(dir, "catalog.xml")
	require.NoError(t, syscall.Mkfifo(fifo, 0o600), "mkfifo")

	// Settle any goroutines from prior work so the baseline is meaningful.
	settle()
	before := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, err := catalog.Load(ctx, fifo)
		done <- err
	}()

	// Let the load block on the open/read of the writer-less FIFO, then cancel.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.Error(t, err, "cancelled FIFO load must return an error")
	case <-time.After(3 * time.Second):
		t.Fatal("Load did not return after cancellation of a blocking FIFO read")
	}

	// Allow the (now-unblocked) reader and watcher goroutines to exit, then
	// assert the count returned to baseline. Without the O_NONBLOCK open + read
	// deadline this stays elevated because a reader is stuck in os.Open/ReadAll.
	settle()
	after := runtime.NumGoroutine()
	require.LessOrEqual(t, after, before,
		"goroutine leak after cancelled FIFO load: before=%d after=%d", before, after)
}

// settle gives recently-unblocked goroutines a chance to exit before a
// NumGoroutine snapshot, polling instead of sleeping a fixed duration.
func settle() {
	for range 50 {
		runtime.Gosched()
		time.Sleep(10 * time.Millisecond)
	}
}

// An already-cancelled context must make Load fail fast without blocking on a
// pathological source.
func TestLoadAlreadyCancelled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fifo := filepath.Join(dir, "catalog.xml")
	require.NoError(t, syscall.Mkfifo(fifo, 0o600), "mkfifo")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() {
		_, err := catalog.Load(ctx, fifo)
		done <- err
	}()

	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Load did not fail fast on an already-cancelled context")
	}
}
