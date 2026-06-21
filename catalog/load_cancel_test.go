//go:build unix

package catalog_test

import (
	"context"
	"path/filepath"
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
