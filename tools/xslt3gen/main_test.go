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

func TestCatalogRelPath(t *testing.T) {
	sourceDir := filepath.Join("/tmp", "source")

	t.Run("in-tree file resolves to slash-normalized rel path", func(t *testing.T) {
		got, ok := catalogRelPath(sourceDir, "tests/insn", "foo.xsl")
		require.True(t, ok)
		require.Equal(t, filepath.Join("tests", "insn", "foo.xsl"), got)
	})

	t.Run("empty file rejected", func(t *testing.T) {
		_, ok := catalogRelPath(sourceDir, "tests/insn", "")
		require.False(t, ok)
	})

	t.Run("dot-dot escape rejected", func(t *testing.T) {
		_, ok := catalogRelPath(sourceDir, "tests/insn", "../../../xslt3/pwn.go")
		require.False(t, ok)
	})

	t.Run("absolute file rejected", func(t *testing.T) {
		_, ok := catalogRelPath(sourceDir, "tests/insn", "/etc/passwd")
		require.False(t, ok)
	})

	t.Run("escaping tsDir rejected", func(t *testing.T) {
		_, ok := catalogRelPath(sourceDir, "../../etc", "passwd")
		require.False(t, ok)
	})

	t.Run("fragment stripped before containment", func(t *testing.T) {
		got, ok := catalogRelPath(sourceDir, "tests/insn", "foo.xml#frag")
		require.True(t, ok)
		require.Equal(t, filepath.Join("tests", "insn", "foo.xml"), got)
	})

	t.Run("interior dot-dot staying inside is allowed", func(t *testing.T) {
		got, ok := catalogRelPath(sourceDir, "tests/insn", "../shared/common.xsl")
		require.True(t, ok)
		require.Equal(t, filepath.Join("tests", "shared", "common.xsl"), got)
	})
}

// TestAddTransitiveDepsContainment verifies that the dependency-collection entry
// point used while populating the asset set never records a dependency that
// escapes sourceDir, even when the entry stylesheet imports an outside file.
func TestAddTransitiveDepsContainment(t *testing.T) {
	sourceDir := t.TempDir()
	outside := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "tests", "insn"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "tests", "shared"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "tests", "shared", "common.xsl"),
		[]byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "evil.xsl"),
		[]byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>`), 0o644))

	entrySrc := `<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:include href="../shared/common.xsl"/>
  <xsl:import href="../../` + filepath.Base(outside) + `/evil.xsl"/>
</xsl:stylesheet>`
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "tests", "insn", "main.xsl"),
		[]byte(entrySrc), 0o644))

	assetFiles := make(map[string]struct{})
	// main.xsl is the contained entry path (as produced by catalogRelPath).
	addTransitiveDeps(assetFiles, sourceDir, filepath.Join("tests", "insn", "main.xsl"))

	require.Contains(t, assetFiles, filepath.Join("tests", "shared", "common.xsl"))
	for rel := range assetFiles {
		require.False(t, filepath.IsAbs(rel) || rel == ".." ||
			(len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator)),
			"asset %q escapes sourceDir", rel)
	}
}

// TestReadContainedAssertionFile verifies the assert-xml/assert-serialization
// "file" read site rejects escaping references before reading.
func TestReadContainedAssertionFile(t *testing.T) {
	sourceDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "tests", "insn"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "tests", "insn", "expected.out"),
		[]byte("hello"), 0o644))

	t.Run("in-tree file read", func(t *testing.T) {
		data, err := readContainedAssertionFile(sourceDir, "tests/insn", "expected.out")
		require.NoError(t, err)
		require.Equal(t, "hello", string(data))
	})

	t.Run("escaping file rejected without read", func(t *testing.T) {
		_, err := readContainedAssertionFile(sourceDir, "tests/insn", "../../../etc/passwd")
		require.Error(t, err)
	})

	t.Run("absolute file rejected", func(t *testing.T) {
		_, err := readContainedAssertionFile(sourceDir, "tests/insn", "/etc/passwd")
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
