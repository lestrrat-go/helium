package main

import (
	"encoding/xml"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainedPath(t *testing.T) {
	root := filepath.Join("/tmp", "assets")

	t.Run("normal relative path", func(t *testing.T) {
		got, err := containedPath(root, "tests/insn/foo.xsl")
		require.NoError(t, err)
		require.Equal(t, filepath.Join(root, "tests", "insn", "foo.xsl"), got)
	})

	t.Run("absolute path rejected", func(t *testing.T) {
		_, err := containedPath(root, "/etc/passwd")
		require.Error(t, err)
	})

	t.Run("dot-dot escape rejected", func(t *testing.T) {
		_, err := containedPath(root, "../../xslt3/pwn.go")
		require.Error(t, err)
	})

	t.Run("interior dot-dot that stays inside is allowed", func(t *testing.T) {
		got, err := containedPath(root, "tests/insn/../foo.xsl")
		require.NoError(t, err)
		require.Equal(t, filepath.Join(root, "tests", "foo.xsl"), got)
	})

	t.Run("interior dot-dot that escapes is rejected", func(t *testing.T) {
		_, err := containedPath(root, "tests/../../foo.xsl")
		require.Error(t, err)
	})
}

func TestResolveQNameWithAttrs(t *testing.T) {
	t.Run("prefixed name resolves to Clark notation", func(t *testing.T) {
		attrs := []xml.Attr{
			{Name: xml.Name{Space: "xmlns", Local: "p"}, Value: "urn:p"},
		}
		got := resolveQNameWithAttrs("p:x", attrs)
		require.Equal(t, "{urn:p}x", got)
	})

	t.Run("unprefixed name kept as-is", func(t *testing.T) {
		got := resolveQNameWithAttrs("x", nil)
		require.Equal(t, "x", got)
	})

	t.Run("unbound prefix kept as-is", func(t *testing.T) {
		got := resolveQNameWithAttrs("p:x", nil)
		require.Equal(t, "p:x", got)
	})
}
