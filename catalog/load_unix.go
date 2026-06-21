//go:build unix

package catalog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

// readCatalogBytes opens absPath and reads up to readLimit bytes, honoring ctx.
//
// The file is opened with O_NONBLOCK so a pathological source whose open itself
// blocks uninterruptibly — most notably a FIFO with no writer — returns
// immediately instead of parking a goroutine forever in the open syscall. This
// is what makes cancellation leak-free on unix: there is never a goroutine
// stuck inside os.OpenFile.
//
// A FIFO (and other pollable sources) opened non-blocking is registered with
// the Go runtime poller, so a blocking read can be interrupted by setting a
// read deadline in the past. A watcher goroutine does exactly that on ctx
// cancellation, which unblocks the in-flight read; the reader goroutine then
// returns and nothing leaks. For a regular file the open never blocks, the read
// never parks, and SetReadDeadline is a harmless no-op (it returns
// os.ErrNoDeadline, which is ignored).
func readCatalogBytes(ctx context.Context, absPath string, readLimit int64) ([]byte, error) {
	f, err := os.OpenFile(absPath, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to read %q: %w", absPath, err)
	}
	defer f.Close()

	// Stop the watcher and clear the deadline when we are done so a deadline
	// armed just as the read completes does not leak into anything reusing the
	// fd (it is closed here anyway, but keep the state tidy).
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			// Interrupt a blocking read on a pollable source. Regular files
			// are not pollable; SetReadDeadline returns os.ErrNoDeadline there,
			// which is fine because their reads never block.
			_ = f.SetReadDeadline(time.Now().Add(-time.Second))
		case <-done:
		}
	}()

	data, err := io.ReadAll(io.LimitReader(f, readLimit))
	if err != nil {
		// A deadline-interrupted read surfaces as os.ErrDeadlineExceeded; report
		// the cancellation cause rather than a generic read error.
		if errors.Is(err, os.ErrDeadlineExceeded) {
			if cerr := ctx.Err(); cerr != nil {
				return nil, cerr
			}
		}
		return nil, fmt.Errorf("catalog: failed to read %q: %w", absPath, err)
	}

	// The read completed, but ctx may have been cancelled in a way that did not
	// interrupt it (e.g. a regular file that finished first). Honor ctx anyway.
	if cerr := ctx.Err(); cerr != nil {
		return nil, cerr
	}
	return data, nil
}
