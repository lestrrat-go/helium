package heliumtest

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// PollContext is a context.Context that counts how many times its Err method
// is called and turns done at a chosen call. helium's long-running walks poll
// their context through Err, so the count measures how much of that work ran,
// and the chosen call places a cancellation or deadline at a fixed point in it.
// A test built on it depends on the work done, never on how long it took.
//
// Value lookups go to the parent. The parent's own cancellation and deadline
// are ignored: the context turns done only at its chosen poll.
type PollContext struct {
	parent   context.Context //nolint:containedctx // Value lookups are forwarded to it
	expireAt int64
	err      error
	polls    atomic.Int64
	doneOnce sync.Once
	done     chan struct{}
}

// NewPollContext returns a PollContext whose expireAt-th Err call, and every
// later one, returns err and closes Done. An expireAt of zero or less never
// expires, which makes the context a pure poll counter. err is typically
// context.Canceled or context.DeadlineExceeded.
func NewPollContext(parent context.Context, expireAt int, err error) *PollContext {
	return &PollContext{parent: parent, expireAt: int64(expireAt), err: err, done: make(chan struct{})}
}

// Deadline reports no deadline.
func (c *PollContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

// Done returns a channel that is closed at the expiring Err call.
func (c *PollContext) Done() <-chan struct{} {
	return c.done
}

// Err counts the call and returns the configured error from the expiring call
// on, nil before it.
func (c *PollContext) Err() error {
	n := c.polls.Add(1)
	if c.expireAt <= 0 || n < c.expireAt {
		return nil
	}
	c.doneOnce.Do(c.closeDone)
	return c.err
}

func (c *PollContext) closeDone() {
	close(c.done)
}

// Value returns the parent's value for key.
func (c *PollContext) Value(key any) any {
	return c.parent.Value(key)
}

// Polls returns how many times Err has been called.
func (c *PollContext) Polls() int {
	return int(c.polls.Load())
}

// PollsAfterExpiry returns how many Err calls followed the expiring one, or -1
// when the context has not expired.
func (c *PollContext) PollsAfterExpiry() int {
	n := c.polls.Load()
	if c.expireAt <= 0 || n < c.expireAt {
		return -1
	}
	return int(n - c.expireAt)
}
