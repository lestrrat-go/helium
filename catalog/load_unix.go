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

// Bounds of the wait between read attempts on a descriptor that the Go runtime
// poller does not manage (see waitingReader). The wait doubles from the lower
// bound after each empty attempt, is capped at the upper bound, and drops back
// to the lower bound once data arrives.
const (
	minReadRetryWait = time.Millisecond
	maxReadRetryWait = 100 * time.Millisecond
)

// errReadCancelled reports that waitingReader stopped waiting because ctx was
// done. readCatalogBytes replaces it with ctx.Err().
var errReadCancelled = errors.New("catalog: read cancelled")

// readCatalogBytes opens absPath and reads up to readLimit bytes, honoring ctx.
//
// The file is opened with O_NONBLOCK so a pathological source whose open itself
// blocks uninterruptibly — most notably a FIFO with no writer — returns
// immediately instead of parking a goroutine forever in the open syscall. This
// is what makes cancellation leak-free on unix: there is never a goroutine
// stuck inside os.OpenFile.
//
// Where the Go runtime poller manages the descriptor (a FIFO on Linux, for
// example), a read with no data available parks in the poller, and a blocking
// read can be interrupted by setting a read deadline in the past. A watcher
// goroutine does exactly that on ctx cancellation, which unblocks the in-flight
// read; the reader then returns and nothing leaks.
//
// Where the poller does not manage the descriptor (Go never registers a FIFO
// with kqueue on darwin and ios), the non-blocking read reports EAGAIN instead
// of waiting. waitingReader turns that into a wait: it sleeps and retries until
// data or end-of-file arrives, and stops as soon as ctx is done. No extra
// goroutine is involved, so nothing leaks there either.
//
// For a regular file the open never blocks, the read never reports EAGAIN, and
// SetReadDeadline is a harmless no-op (it returns os.ErrNoDeadline, which is
// ignored).
func readCatalogBytes(ctx context.Context, absPath string, readLimit int64) ([]byte, error) {
	f, err := os.OpenFile(absPath, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("catalog: failed to read %q: %w", absPath, err)
	}
	defer f.Close()

	// Stop the watcher when we are done so it does not outlive the read.
	done := make(chan struct{})
	defer close(done)
	go interruptReadOnCancel(ctx, f, done)

	r := &waitingReader{f: f, cancelled: ctx.Done()}
	data, err := io.ReadAll(io.LimitReader(r, readLimit))
	if err != nil {
		// A deadline-interrupted read surfaces as os.ErrDeadlineExceeded, and an
		// abandoned wait as errReadCancelled; report the cancellation cause,
		// which a generic read error would hide.
		if errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, errReadCancelled) {
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

// interruptReadOnCancel sets a read deadline in the past on f once ctx is done,
// which unblocks a read parked in the runtime poller. It returns when either
// ctx is done or done is closed. On a descriptor the poller does not manage,
// SetReadDeadline returns os.ErrNoDeadline and waitingReader handles the
// cancellation instead.
func interruptReadOnCancel(ctx context.Context, f *os.File, done <-chan struct{}) {
	select {
	case <-ctx.Done():
		_ = f.SetReadDeadline(time.Now().Add(-time.Second))
	case <-done:
	}
}

// waitingReader reads from a file opened with O_NONBLOCK. When a read reports
// EAGAIN, which only happens where the runtime poller does not manage the
// descriptor, it waits and retries until data or end-of-file arrives, or until
// cancelled is closed.
type waitingReader struct {
	f         *os.File
	cancelled <-chan struct{}
	wait      time.Duration
}

func (r *waitingReader) Read(p []byte) (int, error) {
	for {
		n, err := r.f.Read(p)
		if !errors.Is(err, syscall.EAGAIN) {
			r.wait = 0
			return n, err
		}
		if n > 0 {
			r.wait = 0
			return n, nil
		}
		if err := r.sleep(); err != nil {
			return 0, err
		}
	}
}

// sleep waits before the next read attempt, backing off from minReadRetryWait
// to maxReadRetryWait. It returns errReadCancelled as soon as cancelled closes.
func (r *waitingReader) sleep() error {
	switch {
	case r.wait < minReadRetryWait:
		r.wait = minReadRetryWait
	case r.wait < maxReadRetryWait:
		r.wait = min(2*r.wait, maxReadRetryWait)
	}

	t := time.NewTimer(r.wait)
	defer t.Stop()
	select {
	case <-r.cancelled:
		return errReadCancelled
	case <-t.C:
		return nil
	}
}
