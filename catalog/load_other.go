//go:build !unix

package catalog

import (
	"context"
	"fmt"
	"io"
	"os"
)

// readCatalogBytes opens absPath and reads up to readLimit bytes, honoring ctx.
//
// On non-unix platforms the O_NONBLOCK open trick used by the unix build is not
// portable, so the open+read runs on a helper goroutine and readCatalogBytes
// returns ctx.Err() as soon as ctx is done. An in-flight read is additionally
// unblocked by closing the file, so the helper does not leak once a writer
// appears or the source drains.
//
// Residual limitation: a source whose open syscall blocks uninterruptibly (a
// FIFO/named pipe with no writer) can still leave the helper goroutine parked
// inside os.Open until a writer appears. Regular files — the overwhelmingly
// common case — never block on open, so the helper always returns for them.
func readCatalogBytes(ctx context.Context, absPath string, readLimit int64) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}
	// Buffered so the helper goroutine never blocks sending even if this
	// function has already returned on ctx cancellation.
	resCh := make(chan result, 1)
	// fileCh hands the opened file to the cancellation watcher so it can close
	// an in-flight read; buffered for the same non-blocking reason.
	fileCh := make(chan *os.File, 1)

	go func() {
		f, err := os.Open(absPath)
		if err != nil {
			resCh <- result{err: fmt.Errorf("catalog: failed to read %q: %w", absPath, err)}
			return
		}
		fileCh <- f
		defer f.Close()

		data, err := io.ReadAll(io.LimitReader(f, readLimit))
		if err != nil {
			resCh <- result{err: fmt.Errorf("catalog: failed to read %q: %w", absPath, err)}
			return
		}
		resCh <- result{data: data}
	}()

	select {
	case <-ctx.Done():
		// Unblock an in-flight read by closing the file if it was opened.
		select {
		case f := <-fileCh:
			_ = f.Close()
		default:
		}
		return nil, ctx.Err()
	case res := <-resCh:
		if res.err != nil {
			return nil, res.err
		}
		return res.data, nil
	}
}
