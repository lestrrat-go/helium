package main

import (
	"encoding/xml"
	"os"
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

func TestCollectTransitiveDepsContainment(t *testing.T) {
	// root/ is the containment boundary. tests/ holds the entry stylesheet,
	// shared/ holds an in-tree dependency, and an out-of-tree file sits
	// outside root entirely.
	root := t.TempDir()
	outside := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(root, "tests"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "shared"), 0o755))

	insideDep := filepath.Join(root, "shared", "common.xsl")
	require.NoError(t, os.WriteFile(insideDep,
		[]byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>`), 0o644))

	outsideDep := filepath.Join(outside, "evil.xsl")
	require.NoError(t, os.WriteFile(outsideDep,
		[]byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>`), 0o644))

	entry := filepath.Join(root, "tests", "main.xsl")
	entrySrc := `<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:include href="../shared/common.xsl"/>
  <xsl:import href="../../` + filepath.Base(outside) + `/evil.xsl"/>
  <xsl:import href="` + outsideDep + `"/>
</xsl:stylesheet>`
	require.NoError(t, os.WriteFile(entry, []byte(entrySrc), 0o644))

	deps := collectTransitiveDeps(root, entry)

	require.Contains(t, deps, insideDep, "in-tree dependency must be discovered")
	for _, d := range deps {
		require.NotEqual(t, outsideDep, d, "absolute escaping dependency must be rejected")
		rel, err := filepath.Rel(root, d)
		require.NoError(t, err)
		require.False(t, rel == ".." || filepath.IsAbs(rel) ||
			len(rel) >= 2 && rel[:2] == "..",
			"dependency %q escapes root", d)
	}
}

func TestResolveDepRejectsEscape(t *testing.T) {
	root := filepath.Join("/tmp", "assets")
	dir := filepath.Join(root, "tests", "insn")

	t.Run("in-tree relative reference is resolved", func(t *testing.T) {
		got, ok := resolveDep(root, dir, "../shared/common.xsl")
		require.True(t, ok)
		require.Equal(t, filepath.Join(root, "tests", "shared", "common.xsl"), got)
	})

	t.Run("dot-dot escape rejected", func(t *testing.T) {
		_, ok := resolveDep(root, dir, "../../../etc/passwd")
		require.False(t, ok)
	})

	t.Run("absolute reference rejected", func(t *testing.T) {
		_, ok := resolveDep(root, dir, "/etc/passwd")
		require.False(t, ok)
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
