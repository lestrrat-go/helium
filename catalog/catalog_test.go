package catalog_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/catalog"
	icatalog "github.com/lestrrat-go/helium/internal/catalog"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/stretchr/testify/require"
)

const (
	testIDToto        = "toto"
	docbookDbcentxMod = "http://www.oasis-open.org/docbook/xml/4.1.2/dbcentx.mod"
	docbookXDtd       = "http://www.oasis-open.org/docbook/xml/4.1.2/docbookx.dtd"
)

func testdataDir() string {
	return heliumtest.TestDir("testdata", "libxml2-compat", "catalogs")
}

func loadTestCatalog(t *testing.T, name string) *catalog.Catalog {
	t.Helper()
	dir := testdataDir()
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/libxml2-compat/catalogs not found; run testdata/libxml2/generate.sh first")
	}
	cat, err := catalog.Load(context.Background(), filepath.Join(dir, name))
	require.NoError(t, err, "loading catalog %s", name)
	return cat
}

func TestDocbook(t *testing.T) {
	cat := loadTestCatalog(t, "docbook.xml")

	tests := []struct {
		name   string
		pubID  string
		sysID  string
		expect string
	}{
		{
			name:   "resolve with rewriteSystem",
			pubID:  testIDToto,
			sysID:  "http://www.oasis-open.org/docbook/xml/4.1.2/dbpoolx.mod",
			expect: "/usr/share/xml/docbook/xml/4.1.2/dbpoolx.mod",
		},
		{
			name:   "public match",
			pubID:  "-//OASIS//ENTITIES DocBook XML Character Entities V4.1.2//EN",
			expect: docbookDbcentxMod,
		},
		{
			name:   "system URN unwrap",
			sysID:  "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN",
			expect: docbookXDtd,
		},
		{
			name:   "public URN unwrap",
			pubID:  "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN",
			expect: docbookXDtd,
		},
		{
			name:   "nextCatalog public match",
			pubID:  testIDToto,
			sysID:  testIDToto,
			expect: "file:///usr/share/xml/toto/toto.dtd",
		},
		{
			name:   "no match",
			pubID:  "nonexistent",
			sysID:  "nonexistent",
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cat.Resolve(t.Context(), tt.pubID, tt.sysID)
			require.Equal(t, tt.expect, got)
		})
	}
}

func TestRegistry(t *testing.T) {
	cat := loadTestCatalog(t, "registry.xml")

	tests := []struct {
		name   string
		pubID  string
		sysID  string
		expect string
	}{
		{
			name:   "delegateSystem with rewriteSystem in delegate",
			pubID:  testIDToto,
			sysID:  "http://www.oasis-open.org/docbook/xml/4.1.2/dbpoolx.mod",
			expect: "/usr/share/xml/docbook/xml/4.1.2/dbpoolx.mod",
		},
		{
			name:   "delegatePublic",
			pubID:  "-//OASIS//ENTITIES DocBook XML Character Entities V4.1.2//EN",
			expect: docbookDbcentxMod,
		},
		{
			name:   "delegateSystem exact",
			sysID:  "http://www.oasis-open.org/docbook/xml/4.1.2/dbpoolx.mod",
			expect: "/usr/share/xml/docbook/xml/4.1.2/dbpoolx.mod",
		},
		{
			name:   "system URN unwrap through delegate",
			sysID:  "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN",
			expect: docbookXDtd,
		},
		{
			name:   "no match",
			pubID:  "nonexistent",
			sysID:  "nonexistent",
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cat.Resolve(t.Context(), tt.pubID, tt.sysID)
			require.Equal(t, tt.expect, got)
		})
	}
}

func TestWhitex(t *testing.T) {
	cat := loadTestCatalog(t, "whitex.xml")

	tests := []struct {
		name   string
		pubID  string
		sysID  string
		expect string
	}{
		{
			name:   "resolve with whitespace in pubID",
			pubID:  "toto  ",
			sysID:  "http://www.oasis-open.org/docbook/xml/4.1.2/dbpoolx.mod",
			expect: "/usr/share/xml/docbook/xml/4.1.2/dbpoolx.mod",
		},
		{
			name:   "public with tab in ID (normalized to match)",
			pubID:  "-//OASIS//ENTITIES\tDocBook XML Character Entities V4.1.2//EN",
			expect: docbookDbcentxMod,
		},
		{
			name:   "public with leading space (normalized to match)",
			pubID:  " -//OASIS//ENTITIES DocBook XML Character Entities V4.1.2//EN",
			expect: docbookDbcentxMod,
		},
		{
			name:   "public with trailing space (normalized to match)",
			pubID:  "-//OASIS//ENTITIES DocBook XML Character Entities V4.1.2//EN ",
			expect: docbookDbcentxMod,
		},
		{
			name:   "system URN with leading/trailing spaces in unwrapped ID",
			sysID:  "urn:publicid:+-:OASIS:DTD+++DocBook+XML+V4.1.2:EN+",
			expect: docbookXDtd,
		},
		{
			name:   "public URN with multiple spaces in unwrapped ID",
			pubID:  "urn:publicid:+-:OASIS:DTD+DocBook+XML+++V4.1.2:EN+",
			expect: docbookXDtd,
		},
		{
			name:   "nextCatalog resolve with whitespace pubID",
			pubID:  "\ttoto\t",
			sysID:  testIDToto,
			expect: "file:///usr/share/xml/toto/toto.dtd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cat.Resolve(t.Context(), tt.pubID, tt.sysID)
			require.Equal(t, tt.expect, got)
		})
	}
}

func TestRecursive(t *testing.T) {
	cat := loadTestCatalog(t, "catalog-recursive.xml")

	// Resolving a URI that triggers the recursive catalog should return ""
	// (recursion limit prevents infinite loop).
	got := cat.ResolveURI(t.Context(), "/foo/bar")
	require.Equal(t, "", got)
}

func TestRepeatedNextCatalog(t *testing.T) {
	// repeated-next-catalog.xml has multiple nextCatalog entries pointing
	// to registry.xml with various relative paths. After dedup during
	// parsing, only unique entries remain.
	cat := loadTestCatalog(t, "repeated-next-catalog.xml")

	// Should still resolve correctly through the deduplicated nextCatalogs.
	got := cat.Resolve(t.Context(), "-//OASIS//ENTITIES DocBook XML Character Entities V4.1.2//EN", "")
	require.Equal(t, docbookDbcentxMod, got)
}

func TestNormalizePublicID(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"hello", "hello"},
		{"  hello  ", "hello"},
		{"hello\tworld", "hello world"},
		{"  hello \t\n world  ", "hello world"},
		{"\t\n", ""},
		{"", ""},
		{"a  b  c", "a b c"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := icatalog.NormalizePublicID(tt.input)
			require.Equal(t, tt.expect, got)
		})
	}
}

func TestUnwrapURN(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN", "-//OASIS//DTD DocBook XML V4.1.2//EN"},
		{"urn:publicid:+-:OASIS:DTD+DocBook+XML+V4.1.2:EN+", " -//OASIS//DTD DocBook XML V4.1.2//EN "},
		{"urn:publicid:+-:OASIS:DTD+++DocBook+XML+V4.1.2:EN+", " -//OASIS//DTD   DocBook XML V4.1.2//EN "},
		{"not-a-urn", ""},
		{"", ""},
		{"urn:publicid:test%2Bvalue", "test+value"},
		{"urn:publicid:test%3Avalue", "test:value"},
		{"urn:publicid:test%2Fvalue", "test/value"},
		{"urn:publicid:test%3Bvalue", "test;value"},
		{"urn:publicid:test%27value", "test'value"},
		{"urn:publicid:test%3Fvalue", "test?value"},
		{"urn:publicid:test%23value", "test#value"},
		{"urn:publicid:test%25value", "test%value"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := icatalog.UnwrapURN(tt.input)
			require.Equal(t, tt.expect, got)
		})
	}
}

func TestResolveURI(t *testing.T) {
	cat := loadTestCatalog(t, "stylesheet.xml")

	got := cat.ResolveURI(t.Context(), "http://www.oasis-open.org/committes/tr.xsl")
	require.Equal(t, "http://www.oasis-open.org/committes/entity/stylesheets/base/tr.xsl", got)

	// Non-matching URI should return "".
	got = cat.ResolveURI(t.Context(), "http://example.com/nonexistent")
	require.Equal(t, "", got)
}

// A root catalog whose nextCatalog references a downstream catalog via a
// "file://" URI must resolve the downstream mapping. The file: URI has to be
// converted to a local filesystem path before opening.
func TestNextCatalogFileURI(t *testing.T) {
	dir := t.TempDir()

	nextPath, err := filepath.Abs(filepath.Join(dir, "next.xml"))
	require.NoError(t, err)

	nextXML := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <uri name="http://example.com/asset" uri="file:///downstream/asset.xml"/>
</catalog>`
	require.NoError(t, os.WriteFile(nextPath, []byte(nextXML), 0o600))

	// Reference the downstream catalog via a file:// URI (not a bare path).
	// Build the URI portably: slash-normalize the absolute path and ensure a
	// leading slash so the authority component is empty. On POSIX nextPath is
	// "/abs/next.xml" -> "file:///abs/next.xml"; on Windows it is
	// "C:\...\next.xml" -> "C:/.../next.xml" -> "file:///C:/.../next.xml".
	// This round-trips through the production catalogFilePath/fileURIPath code
	// back to nextPath on the current OS.
	slashPath := filepath.ToSlash(nextPath)
	if !strings.HasPrefix(slashPath, "/") {
		slashPath = "/" + slashPath
	}
	nextURI := (&url.URL{Scheme: "file", Path: slashPath}).String()
	rootXML := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <nextCatalog catalog="` + nextURI + `"/>
</catalog>`
	rootPath := filepath.Join(dir, "root.xml")
	require.NoError(t, os.WriteFile(rootPath, []byte(rootXML), 0o600))

	cat, err := catalog.Load(context.Background(), rootPath)
	require.NoError(t, err)

	got := cat.ResolveURI(context.Background(), "http://example.com/asset")
	require.Equal(t, "file:///downstream/asset.xml", got)
}

// A downstream catalog reached via a "file:" nextCatalog whose own entry uses a
// RELATIVE uri must resolve that uri against the catalog's "file:" URI, yielding
// a "file:" URI — not a bare filesystem path. Regression for the baseURI being
// overwritten with the decoded local path in loadInternal.
func TestNextCatalogFileURIRelativeEntry(t *testing.T) {
	dir := t.TempDir()

	nextPath, err := filepath.Abs(filepath.Join(dir, "next.xml"))
	require.NoError(t, err)

	// The downstream entry's uri is relative ("asset.xml"); it must resolve
	// against the downstream catalog's own file: URI.
	nextXML := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <uri name="http://example.com/asset" uri="asset.xml"/>
</catalog>`
	require.NoError(t, os.WriteFile(nextPath, []byte(nextXML), 0o600))

	// Reference the downstream catalog via a portable file:// URI (see
	// TestNextCatalogFileURI for the construction rationale).
	slashPath := filepath.ToSlash(nextPath)
	if !strings.HasPrefix(slashPath, "/") {
		slashPath = "/" + slashPath
	}
	nextURI := (&url.URL{Scheme: "file", Path: slashPath}).String()
	rootXML := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <nextCatalog catalog="` + nextURI + `"/>
</catalog>`
	rootPath := filepath.Join(dir, "root.xml")
	require.NoError(t, os.WriteFile(rootPath, []byte(rootXML), 0o600))

	cat, err := catalog.Load(context.Background(), rootPath)
	require.NoError(t, err)

	got := cat.ResolveURI(context.Background(), "http://example.com/asset")

	// The relative "asset.xml" must resolve to a file: URI in the same
	// directory as next.xml, not to the bare local path nextPath's directory.
	dirSlash := filepath.ToSlash(filepath.Dir(nextPath))
	if !strings.HasPrefix(dirSlash, "/") {
		dirSlash = "/" + dirSlash
	}
	want := (&url.URL{Scheme: "file", Path: dirSlash + "/asset.xml"}).String()
	require.Equal(t, want, got)
	require.True(t, strings.HasPrefix(got, "file:"),
		"relative downstream uri must resolve to a file: URI, got %q", got)
}

func TestLoadError(t *testing.T) {
	_, err := catalog.Load(context.Background(), "/nonexistent/catalog.xml")
	require.Error(t, err)
}

func TestNilCatalog(t *testing.T) {
	var c *catalog.Catalog
	require.Equal(t, "", c.Resolve(t.Context(), "foo", "bar"))
	require.Equal(t, "", c.ResolveURI(t.Context(), "foo"))
}
