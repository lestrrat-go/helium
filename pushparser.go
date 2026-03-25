package helium

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

// pushStream is a concurrent buffer that bridges Push() calls and the parser's
// io.Reader interface. Its Read method blocks until enough bytes are available
// (or the stream is closed), guaranteeing that ByteCursor.fillBuffer always
// receives the number of bytes it requests.
type pushStream struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    bytes.Buffer
	closed bool
	wrErr  error // returned to future Write calls when parser has failed
}

func newPushStream() *pushStream {
	ps := &pushStream{}
	ps.cond = sync.NewCond(&ps.mu)
	return ps
}

// Read blocks until len(p) bytes are available or the stream is closed.
// When closed, it returns whatever remains in the buffer. Returns io.EOF
// when the buffer is empty and the stream is closed.
func (ps *pushStream) Read(p []byte) (int, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for ps.buf.Len() < len(p) && !ps.closed {
		ps.cond.Wait()
	}

	if ps.buf.Len() == 0 && ps.closed {
		return 0, io.EOF
	}

	return ps.buf.Read(p)
}

// Write appends data to the buffer and signals waiting readers.
// It never blocks. Returns wrErr if the parser has already failed.
func (ps *pushStream) Write(p []byte) (int, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.wrErr != nil {
		return 0, ps.wrErr
	}
	if ps.closed {
		return 0, io.ErrClosedPipe
	}

	n, err := ps.buf.Write(p)
	ps.cond.Signal()
	return n, err
}

// Close marks the stream as closed, waking any blocked reader.
func (ps *pushStream) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.closed = true
	ps.cond.Broadcast()
	return nil
}

// closeWithWriteError sets wrErr so that future Write calls return err,
// and closes the stream.
func (ps *pushStream) closeWithWriteError(err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.wrErr = err
	ps.closed = true
	ps.cond.Broadcast()
}

type pushResult struct {
	doc *Document
	err error
}

// PushParser provides an incremental XML parsing interface
// (libxml2: xmlParserCtxt in push mode).
// Data is pushed via Push or Write, and the parser processes tokens as
// they become available in a background goroutine. Call Close to signal
// end-of-input and retrieve the parsed Document.
type PushParser struct {
	stream    *pushStream
	done      chan pushResult
	closeOnce sync.Once
	result    pushResult
}

// NewPushParser creates a PushParser using the given Parser's configuration.
// The parser runs in a background goroutine, reading from an internal buffer
// as data is pushed.
func (p Parser) NewPushParser(ctx context.Context) *PushParser {
	stream := newPushStream()
	pp := &PushParser{
		stream: stream,
		done:   make(chan pushResult, 1),
	}

	go func() {
		var res pushResult
		defer func() {
			if r := recover(); r != nil {
				if res.err == nil {
					res.err = fmt.Errorf("parser panic: %v", r)
				}
			}
			stream.closeWithWriteError(res.err)
			pp.done <- res
		}()

		doc, err := p.ParseReader(ctx, stream)
		res = pushResult{doc: doc, err: err}
	}()

	return pp
}

// Push sends a chunk of XML data to the parser. It returns an error if the
// parser has already failed or if the stream has been closed.
func (pp *PushParser) Push(chunk []byte) error {
	_, err := pp.stream.Write(chunk)
	return err
}

// Write implements io.Writer, allowing use with io.Copy and similar functions.
func (pp *PushParser) Write(p []byte) (int, error) {
	return pp.stream.Write(p)
}

// Close signals end-of-input, waits for the parser goroutine to finish, and
// returns the parsed Document. It is idempotent: subsequent calls return the
// same result.
func (pp *PushParser) Close() (*Document, error) {
	pp.closeOnce.Do(func() {
		if err := pp.stream.Close(); err != nil {
			pp.result = pushResult{err: err}
			return
		}
		pp.result = <-pp.done
	})
	return pp.result.doc, pp.result.err
}
