//go:build unix

package catalog_test

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/catalog"
	"github.com/stretchr/testify/require"
)

// newBlockingFIFO creates a FIFO and holds it open for writing for the rest of
// the test without ever writing to it, so a read of the FIFO blocks until the
// reader gives up. (With no writer at all, a non-blocking read of a FIFO
// reports end-of-file at once and nothing would block.) O_RDWR opens a FIFO
// without waiting for the other end.
func newBlockingFIFO(t *testing.T) string {
	t.Helper()
	fifo := filepath.Join(t.TempDir(), "catalog.xml")
	require.NoError(t, syscall.Mkfifo(fifo, 0o600), "mkfifo")
	w, err := os.OpenFile(fifo, os.O_RDWR, 0)
	require.NoError(t, err, "open the FIFO's write end")
	t.Cleanup(func() { _ = w.Close() })
	return fifo
}

// watchContext wraps a context and closes watched the first time code asks for
// its Done channel. Load reads the catalog while a watcher goroutine waits on
// Done to interrupt that read, so watched closing means the file is open and the
// interrupt is armed: a cancellation from then on must be delivered through it.
type watchContext struct {
	context.Context //nolint:containedctx // the wrapper IS the context handed to Load
	once            sync.Once
	watched         chan struct{}
}

func newWatchContext(parent context.Context) *watchContext {
	return &watchContext{Context: parent, watched: make(chan struct{})}
}

func (c *watchContext) Done() <-chan struct{} {
	c.once.Do(c.markWatched)
	return c.Context.Done()
}

func (c *watchContext) markWatched() {
	close(c.watched)
}

// loadInto runs catalog.Load and reports its error on done.
func loadInto(ctx context.Context, filename string, done chan<- error) {
	_, err := catalog.Load(ctx, filename)
	done <- err
}

// cancelWatchedLoad waits until the load in flight on done is waiting on ctx,
// cancels it, and returns the load's error. The timeouts only keep a broken
// Load from hanging the test.
func cancelWatchedLoad(t *testing.T, ctx *watchContext, cancel context.CancelFunc, done <-chan error) error {
	t.Helper()
	select {
	case <-ctx.watched:
	case err := <-done:
		t.Fatalf("Load returned %v before it waited on its context", err)
	case <-time.After(10 * time.Second):
		t.Fatal("Load never waited on its context")
	}
	cancel()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("Load did not return after cancellation of a blocking FIFO read")
		return nil
	}
}

// A catalog read against a FIFO whose writer never writes blocks indefinitely.
// Load must honor ctx and abort on cancellation instead of hanging forever.
func TestLoadCancelsOnBlockingFIFO(t *testing.T) {
	t.Parallel()

	fifo := newBlockingFIFO(t)
	parent, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := newWatchContext(parent)

	done := make(chan error, 1)
	go loadInto(ctx, fifo, done)

	err := cancelWatchedLoad(t, ctx, cancel, done)
	require.ErrorIs(t, err, context.Canceled, "cancelled FIFO load must return the context error")
}

// A blocking FIFO read must not leave a goroutine (and the OS thread it
// occupies) parked forever after Load returns on cancellation. On unix the file
// is opened with O_NONBLOCK and the blocking read is interrupted via a read
// deadline, so no reader goroutine survives.
//
// The load runs on a goroutine carrying a pprof label, which every goroutine it
// starts inherits, so the test counts exactly the goroutines this load created,
// whatever else runs in parallel. Once Load has returned they must all exit.
func TestLoadFIFONoGoroutineLeak(t *testing.T) {
	t.Parallel()

	fifo := newBlockingFIFO(t)
	parent, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := newWatchContext(parent)

	const labelKey, labelValue = "helium-catalog-test", "fifo-leak"
	labels := pprof.WithLabels(context.Background(), pprof.Labels(labelKey, labelValue))
	done := make(chan error, 1)
	go loadLabeled(labels, ctx, fifo, done)

	err := cancelWatchedLoad(t, ctx, cancel, done)
	require.ErrorIs(t, err, context.Canceled, "cancelled FIFO load must return the context error")

	waitLabeledGoroutinesExit(t, labelKey, labelValue)
}

// loadLabeled applies the pprof labels in labels to the current goroutine, so
// every goroutine Load starts inherits them, then runs loadInto.
func loadLabeled(labels, ctx context.Context, filename string, done chan<- error) {
	pprof.SetGoroutineLabels(labels)
	loadInto(ctx, filename, done)
}

// waitLabeledGoroutinesExit waits until no goroutine carries the pprof label
// key=value. A goroutine that has been told to stop can take a moment to exit,
// so this polls; a goroutine that never exits fails the test once the hang
// guard expires, with the goroutine profile in the message.
func waitLabeledGoroutinesExit(t *testing.T, key, value string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		n, profile := labeledGoroutines(t, key, value)
		if n == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d goroutine(s) started by the load never exited:\n%s", n, profile)
		}
		runtime.Gosched()
	}
}

var goroutineRecord = regexp.MustCompile(`^(\d+) @`)

// labeledGoroutines counts the goroutines carrying the pprof label key=value in
// the debug=1 goroutine profile, where each record opens with "<count> @ ..."
// and a labelled record's next line is "# labels: {...}".
func labeledGoroutines(t *testing.T, key, value string) (int, string) {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, pprof.Lookup("goroutine").WriteTo(&buf, 1))
	want := strconv.Quote(key) + ":" + strconv.Quote(value)

	total := 0
	count := 0
	sc := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	for sc.Scan() {
		line := sc.Text()
		if m := goroutineRecord.FindStringSubmatch(line); m != nil {
			count, _ = strconv.Atoi(m[1])
			continue
		}
		if strings.HasPrefix(line, "# labels: ") && strings.Contains(line, want) {
			total += count
		}
		count = 0
	}
	return total, buf.String()
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
