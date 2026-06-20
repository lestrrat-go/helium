package strcursor

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUTF8CursorZeroProgressReaderDoesNotHang(t *testing.T) {
	cur := NewUTF8Cursor(zeroProgressReader{})

	done := make(chan struct{})
	go func() {
		defer close(done)
		require.True(t, cur.Done(), "a zero-progress reader must terminate fill, not spin")
		require.ErrorIs(t, cur.Err(), io.ErrNoProgress, "a zero-progress reader must surface io.ErrNoProgress after the bounded retry count")
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("UTF8Cursor fillBuffer hung on a zero-progress reader")
	}
}

func TestUTF8CursorSlowSplitReaderMakesProgress(t *testing.T) {
	cur := NewUTF8Cursor(&slowSplitReader{data: []byte("héllo")})

	done := make(chan struct{})
	go func() {
		defer close(done)
		require.Equal(t, "héllo", cur.PeekString(len("héllo")), "a slow reader that emits (0, nil) between bytes must still be consumed")
		require.NoError(t, cur.Err(), "a progressing reader must not surface io.ErrNoProgress")
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("UTF8Cursor fillBuffer hung on a slow split reader")
	}
}

type chunkedReader struct {
	data  []byte
	chunk int
}

func (r *chunkedReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := min(r.chunk, len(r.data), len(p))
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

func TestUTF8CursorScanCharDataSliceSpansBufferEdge(t *testing.T) {
	cur := NewUTF8Cursor(&chunkedReader{
		data:  []byte("    <"),
		chunk: 2,
	})

	data, n := cur.ScanCharDataSlice(nil)
	require.Equal(t, 4, n)
	require.Equal(t, "    ", string(data))
}

func TestUTF8CursorScanCharDataSliceConsumesCRLFAcrossBufferEdge(t *testing.T) {
	cur := NewUTF8Cursor(&chunkedReader{
		data:  []byte("\r\n<"),
		chunk: 1,
	})

	data, n := cur.ScanCharDataSlice(nil)
	require.Equal(t, 2, n)
	require.Equal(t, "\n", string(data))
}

func TestUTF8CursorScanCharDataSlicePreservesWhitespaceRunAcrossBufferEdge(t *testing.T) {
	cur := NewUTF8Cursor(&chunkedReader{
		data:  []byte(strings.Repeat(" ", 7) + "<"),
		chunk: 3,
	})

	data, n := cur.ScanCharDataSlice(nil)
	require.Equal(t, 7, n)
	require.Equal(t, strings.Repeat(" ", 7), string(data))
}

func TestUTF8CursorScanQNameBytesASCIIUnprefixed(t *testing.T) {
	cur := NewUTF8Cursor(strings.NewReader("root attr"))

	prefix, local, n, ok := cur.ScanQNameBytes()
	require.True(t, ok)
	require.Nil(t, prefix)
	require.Equal(t, "root", string(local))
	require.Equal(t, 4, n)
}

func TestUTF8CursorScanQNameBytesASCIIPrefixed(t *testing.T) {
	cur := NewUTF8Cursor(strings.NewReader("x:item attr"))

	prefix, local, n, ok := cur.ScanQNameBytes()
	require.True(t, ok)
	require.Equal(t, "x", string(prefix))
	require.Equal(t, "item", string(local))
	require.Equal(t, 6, n)
}

func TestUTF8CursorScanQNameBytesSpansBufferEdge(t *testing.T) {
	cur := NewUTF8Cursor(&chunkedReader{
		data:  []byte("x:item attr"),
		chunk: 2,
	})

	prefix, local, n, ok := cur.ScanQNameBytes()
	require.True(t, ok)
	require.Equal(t, "x", string(prefix))
	require.Equal(t, "item", string(local))
	require.Equal(t, 6, n)
}

func TestUTF8CursorScanQNameBytesRejectsSecondColon(t *testing.T) {
	cur := NewUTF8Cursor(strings.NewReader("a:b:c"))

	prefix, local, n, ok := cur.ScanQNameBytes()
	require.False(t, ok)
	require.Nil(t, prefix)
	require.Nil(t, local)
	require.Zero(t, n)
	require.Equal(t, byte('a'), cur.Peek())
}

func TestRuneCursorReadShortBufferBufferedRune(t *testing.T) {
	cur := NewRuneCursor(strings.NewReader("é"))
	// Buffer the multibyte rune in the ring.
	require.Equal(t, 'é', cur.Peek())

	// A 1-byte destination cannot hold the 2-byte rune. Read must not panic
	// and must not corrupt or drop the buffered rune.
	first := make([]byte, 1)
	n, err := cur.Read(first)
	require.NoError(t, err)
	require.Zero(t, n, "no full rune fits in a 1-byte buffer")

	// The rune must still be deliverable on a subsequent read.
	rest := make([]byte, 8)
	var got []byte
	for {
		m, rerr := cur.Read(rest)
		got = append(got, rest[:m]...)
		if rerr == io.EOF {
			break
		}
		require.NoError(t, rerr)
		if m == 0 {
			break
		}
	}
	require.Equal(t, "é", string(got), "rune delivered intact across reads")
}

func TestRuneCursorReadShortBufferPartialRuneFit(t *testing.T) {
	cur := NewRuneCursor(strings.NewReader("aé"))
	// Buffer both runes in the ring.
	require.Equal(t, 'a', cur.Peek())
	require.Equal(t, 'é', cur.PeekN(2))

	// A 2-byte buffer fits 'a' (1 byte) but not the following 2-byte 'é'.
	// Read must emit only 'a' as a short read with no EOF, leaving 'é'
	// buffered rather than reordering bytes from the underlying reader.
	buf := make([]byte, 2)
	n, err := cur.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 1, n, "only the 1-byte rune fits")
	require.Equal(t, "a", string(buf[:n]))

	// The buffered 'é' is delivered intact on the next read.
	rest := make([]byte, 8)
	m, err := cur.Read(rest)
	require.Equal(t, "é", string(rest[:m]))
	if err != nil {
		require.Equal(t, io.EOF, err)
	}
}
