package catalog_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium/catalog"
	icatalog "github.com/lestrrat-go/helium/internal/catalog"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/stretchr/testify/require"
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
			pubID:  "toto",
			sysID:  "http://www.oasis-open.org/docbook/xml/4.1.2/dbpoolx.mod",
			expect: "/usr/share/xml/docbook/xml/4.1.2/dbpoolx.mod",
		},
		{
			name:   "public match",
			pubID:  "-//OASIS//ENTITIES DocBook XML Character Entities V4.1.2//EN",
			expect: "http://www.oasis-open.org/docbook/xml/4.1.2/dbcentx.mod",
		},
		{
			name:   "system URN unwrap",
			sysID:  "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN",
			expect: "http://www.oasis-open.org/docbook/xml/4.1.2/docbookx.dtd",
		},
		{
			name:   "public URN unwrap",
			pubID:  "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN",
			expect: "http://www.oasis-open.org/docbook/xml/4.1.2/docbookx.dtd",
		},
		{
			name:   "nextCatalog public match",
			pubID:  "toto",
			sysID:  "toto",
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
			pubID:  "toto",
			sysID:  "http://www.oasis-open.org/docbook/xml/4.1.2/dbpoolx.mod",
			expect: "/usr/share/xml/docbook/xml/4.1.2/dbpoolx.mod",
		},
		{
			name:   "delegatePublic",
			pubID:  "-//OASIS//ENTITIES DocBook XML Character Entities V4.1.2//EN",
			expect: "http://www.oasis-open.org/docbook/xml/4.1.2/dbcentx.mod",
		},
		{
			name:   "delegateSystem exact",
			sysID:  "http://www.oasis-open.org/docbook/xml/4.1.2/dbpoolx.mod",
			expect: "/usr/share/xml/docbook/xml/4.1.2/dbpoolx.mod",
		},
		{
			name:   "system URN unwrap through delegate",
			sysID:  "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN",
			expect: "http://www.oasis-open.org/docbook/xml/4.1.2/docbookx.dtd",
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
			expect: "http://www.oasis-open.org/docbook/xml/4.1.2/dbcentx.mod",
		},
		{
			name:   "public with leading space (normalized to match)",
			pubID:  " -//OASIS//ENTITIES DocBook XML Character Entities V4.1.2//EN",
			expect: "http://www.oasis-open.org/docbook/xml/4.1.2/dbcentx.mod",
		},
		{
			name:   "public with trailing space (normalized to match)",
			pubID:  "-//OASIS//ENTITIES DocBook XML Character Entities V4.1.2//EN ",
			expect: "http://www.oasis-open.org/docbook/xml/4.1.2/dbcentx.mod",
		},
		{
			name:   "system URN with leading/trailing spaces in unwrapped ID",
			sysID:  "urn:publicid:+-:OASIS:DTD+++DocBook+XML+V4.1.2:EN+",
			expect: "http://www.oasis-open.org/docbook/xml/4.1.2/docbookx.dtd",
		},
		{
			name:   "public URN with multiple spaces in unwrapped ID",
			pubID:  "urn:publicid:+-:OASIS:DTD+DocBook+XML+++V4.1.2:EN+",
			expect: "http://www.oasis-open.org/docbook/xml/4.1.2/docbookx.dtd",
		},
		{
			name:   "nextCatalog resolve with whitespace pubID",
			pubID:  "\ttoto\t",
			sysID:  "toto",
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
	require.Equal(t, "http://www.oasis-open.org/docbook/xml/4.1.2/dbcentx.mod", got)
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

func TestLoadError(t *testing.T) {
	_, err := catalog.Load(context.Background(), "/nonexistent/catalog.xml")
	require.Error(t, err)
}

func TestNilCatalog(t *testing.T) {
	var c *catalog.Catalog
	require.Equal(t, "", c.Resolve(t.Context(), "foo", "bar"))
	require.Equal(t, "", c.ResolveURI(t.Context(), "foo"))
}
