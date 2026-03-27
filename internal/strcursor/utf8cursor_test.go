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
	n := r.chunk
	if n > len(r.data) {
		n = len(r.data)
	}
	if n > len(p) {
		n = len(p)
	}
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
