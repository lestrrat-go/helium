package heliumcmd

import (
	"math"
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
	t.Run("file URI empty host", func(t *testing.T) {
		got, err := localFilePath("file:///tmp/mod.xsl")
		require.NoError(t, err)
		require.Equal(t, "/tmp/mod.xsl", got)
	})
	t.Run("file URI localhost host", func(t *testing.T) {
		got, err := localFilePath("file://localhost/tmp/mod.xsl")
		require.NoError(t, err)
		require.Equal(t, "/tmp/mod.xsl", got)
	})
	t.Run("file URI percent-decoded", func(t *testing.T) {
		got, err := localFilePath("file:///tmp/a%20b/mod.xsl")
		require.NoError(t, err)
		require.Equal(t, "/tmp/a b/mod.xsl", got)
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
