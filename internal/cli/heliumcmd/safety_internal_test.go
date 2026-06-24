package heliumcmd

import (
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalFilePath(t *testing.T) {
	t.Run("plain absolute path passes through", func(t *testing.T) {
		got, err := localFilePath("/tmp/mod.xsl")
		require.NoError(t, err)
		require.Equal(t, "/tmp/mod.xsl", got)
	})
	t.Run("plain relative path passes through", func(t *testing.T) {
		got, err := localFilePath("sub/mod.xsl")
		require.NoError(t, err)
		require.Equal(t, "sub/mod.xsl", got)
	})
	// A "file:" URI is decoded into a NATIVE local path, so the expected value
	// is wrapped in filepath.FromSlash (POSIX "/tmp/mod.xsl", Windows
	// "\\tmp\\mod.xsl"). The plain-path subtests above pass the string through
	// unchanged, so they are not wrapped.
	t.Run("file URI empty host", func(t *testing.T) {
		got, err := localFilePath("file:///tmp/mod.xsl")
		require.NoError(t, err)
		require.Equal(t, filepath.FromSlash("/tmp/mod.xsl"), got)
	})
	t.Run("file URI localhost host", func(t *testing.T) {
		got, err := localFilePath("file://localhost/tmp/mod.xsl")
		require.NoError(t, err)
		require.Equal(t, filepath.FromSlash("/tmp/mod.xsl"), got)
	})
	t.Run("file URI uppercase localhost host", func(t *testing.T) {
		got, err := localFilePath("file://LOCALHOST/tmp/mod.xsl")
		require.NoError(t, err)
		require.Equal(t, filepath.FromSlash("/tmp/mod.xsl"), got)
	})
	t.Run("file URI percent-decoded", func(t *testing.T) {
		got, err := localFilePath("file:///tmp/a%20b/mod.xsl")
		require.NoError(t, err)
		require.Equal(t, filepath.FromSlash("/tmp/a b/mod.xsl"), got)
	})
	t.Run("remote host rejected", func(t *testing.T) {
		_, err := localFilePath("file://example.com/mod.xsl")
		require.Error(t, err)
		require.Contains(t, err.Error(), "host")
	})
	t.Run("http scheme rejected", func(t *testing.T) {
		_, err := localFilePath("http://example.com/mod.xsl")
		require.Error(t, err)
		require.Contains(t, err.Error(), "scheme")
	})
	t.Run("https scheme rejected", func(t *testing.T) {
		_, err := localFilePath("https://example.com/mod.xsl")
		require.Error(t, err)
		require.Contains(t, err.Error(), "scheme")
	})
	t.Run("file URI drive letter on Windows", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Windows-specific drive-letter handling")
		}
		got, err := localFilePath("file:///C:/tmp/mod.xsl")
		require.NoError(t, err)
		require.Equal(t, `C:\tmp\mod.xsl`, got)
	})
}

func TestFileURIPathToLocal(t *testing.T) {
	// The decode logic is platform-independent: drive the windows flag
	// explicitly so both host conventions are exercised on any GOOS.
	t.Run("non-windows passes through", func(t *testing.T) {
		require.Equal(t, "/tmp/mod.xsl", fileURIPathToLocal("/tmp/mod.xsl", false))
		require.Equal(t, "/C:/tmp/x", fileURIPathToLocal("/C:/tmp/x", false))
	})
	t.Run("windows drive letter strips leading slash", func(t *testing.T) {
		require.Equal(t, `C:\tmp\mod.xsl`, fileURIPathToLocal("/C:/tmp/mod.xsl", true))
		require.Equal(t, `c:\x`, fileURIPathToLocal("/c:/x", true))
		require.Equal(t, `C:`, fileURIPathToLocal("/C:", true))
	})
	t.Run("windows rooted non-drive keeps leading slash", func(t *testing.T) {
		// Must stay rooted, not become relative "tmp\\x".
		require.Equal(t, `\tmp\x`, fileURIPathToLocal("/tmp/x", true))
		require.Equal(t, `\share\mod.xsl`, fileURIPathToLocal("/share/mod.xsl", true))
	})
}

func TestIsDriveLetterPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/C:/x", true},
		{"/c:/x", true},
		{"/C:", true},
		{"/tmp/x", false},
		{"/C:foo", false}, // colon not followed by separator
		{"C:/x", false},   // no leading slash
		{"/1:/x", false},  // non-alpha drive
		{"/", false},
		{"", false},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, isDriveLetterPath(tc.path), "path=%q", tc.path)
	}
}

func TestReadInputMaxInt64NoOverflow(t *testing.T) {
	// maxBytes == math.MaxInt64 must not overflow (the old maxBytes+1 wrapped to
	// a negative LimitReader cap and read nothing). The full input must come
	// back unchanged.
	const data = "<?xml version=\"1.0\"?><root>hello</root>"
	got, err := readInput(strings.NewReader(data), "stdin", math.MaxInt64)
	require.NoError(t, err)
	require.Equal(t, data, string(got))
}

func TestReadInputAtAndOverCap(t *testing.T) {
	t.Run("exactly at cap succeeds", func(t *testing.T) {
		data := "abcde"
		got, err := readInput(strings.NewReader(data), "stdin", int64(len(data)))
		require.NoError(t, err)
		require.Equal(t, data, string(got))
	})
	t.Run("one byte over cap fails", func(t *testing.T) {
		data := "abcdef"
		_, err := readInput(strings.NewReader(data), "stdin", int64(len(data)-1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds maximum size")
	})
}
