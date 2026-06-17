package strcursor

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
