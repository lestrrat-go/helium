package xslt3

import (
	"context"
	"io"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestSchemaRegistrySubstitutionGroupMembersUseEligibleTransitiveSet(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="elem0" type="xs:string"/>
  <xs:element name="elem1" type="xs:string" block="substitution"/>
  <xs:element name="elem2" type="xs:string" substitutionGroup="elem0 elem1"/>
  <xs:element name="elem3" type="xs:string" substitutionGroup="elem2"/>
  <xs:element name="blocked" type="xs:string" substitutionGroup="elem1"/>
</xs:schema>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Label("test.xsd").Compile(t.Context(), doc)
	require.NoError(t, err)

	reg := &schemaRegistry{schemas: []*xsd.Schema{schema}}
	require.True(t, reg.IsSubstitutionGroupMember("elem3", "", "elem0", ""), "schema-element(elem0) should see transitive members")
	require.False(t, reg.IsSubstitutionGroupMember("blocked", "", "elem1", ""), "blocked heads must not expose members to xslt3")
}

// part and sxsd are relative schema-location filenames used across these cases.
const (
	part = "part.xsd"
	sxsd = "s.xsd"
)

// TestResolveSchemaURI exercises the URI-aware resolution of a top-level
// schema-location against a base URI directly, including the "../" dot-segment
// case.
func TestResolveSchemaURI(t *testing.T) {
	for _, tc := range []struct {
		name string
		base string
		ref  string
		want string
	}{
		{"http sibling", "https://example.com/s/main.xsd", part, "https://example.com/s/part.xsd"},
		{"http subdir", "https://example.com/s/main.xsd", "sub/part.xsd", "https://example.com/s/sub/part.xsd"},
		{"http parent", "https://example.com/s/sub/main.xsd", "../part.xsd", "https://example.com/s/part.xsd"},
		// Root-relative ref against a URI base must keep the base
		// scheme+authority and replace the path — NOT be returned verbatim as a
		// local-looking "/schemas/s.xsd".
		{"http root-relative", "https://example.com/style/main.xsl", "/schemas/s.xsd", "https://example.com/schemas/s.xsd"},
		{"file root-relative", "file:///tmp/style/main.xsl", "/schemas/s.xsd", "file:///schemas/s.xsd"},
		{"file sibling", "file:///tmp/s/main.xsd", part, "file:///tmp/s/part.xsd"},
		{"file parent", "file:///tmp/s/sub/main.xsd", "../part.xsd", "file:///tmp/s/part.xsd"},
		{"absolute ref unchanged", "file:///tmp/s/main.xsd", "https://other/x.xsd", "https://other/x.xsd"},
		{"empty base", "", part, part},
		// Absolute URIs without a "//" authority (opaque or single-slash)
		// must be passed through verbatim against a LOCAL filesystem base —
		// never filepath-joined onto it.
		{"mem single-slash uri, local base", "/work/main.xsl", "mem:/schemas/s.xsd", "mem:/schemas/s.xsd"},
		{"urn opaque uri, local base", "/work/main.xsl", "urn:schemas:s", "urn:schemas:s"},
		{"file single-slash uri, local base", "/work/main.xsl", "file:/tmp/s.xsd", "file:/tmp/s.xsd"},
		// No-authority single-slash URI base (OmitHost) + relative ref must
		// preserve the "mem:/..." form, NOT gain an empty "//" authority
		// ("mem:///...") that would miss an exact resolver keyed on "mem:/...".
		{"mem single-slash base, relative ref", "mem:/stylesheets/main.xsl", sxsd, "mem:/stylesheets/s.xsd"},
		{"mem runtime doc base, relative ref", "mem:/docs/input.xml", sxsd, "mem:/docs/s.xsd"},
		// Regression: canonical empty-authority bases keep their "///".
		{"file empty-authority base, relative ref", "file:///tmp/style/main.xsl", sxsd, "file:///tmp/style/s.xsd"},
		{"https authority base, relative ref", "https://example.com/s/main.xsl", part, "https://example.com/s/part.xsd"},
		// Local FILE base (last segment has an extension): filepath.Dir drops
		// the filename, so the ref resolves into the containing directory.
		{"local file base, relative ref", "/work/main.xsl", sxsd, "/work/s.xsd"},
		// Local DIRECTORY-like base (extensionless last segment, e.g. from
		// xml:base): treated as a directory, NOT truncated by filepath.Dir.
		{"local dir base, relative ref", "/work/schemas", sxsd, "/work/schemas/s.xsd"},
		{"local dir base trailing slash, relative ref", "/work/schemas/", sxsd, "/work/schemas/s.xsd"},
		// Windows-shaped local base: resolution stays in forward-slash space on
		// every OS (plain strings, so exercised on Linux). The slashed base's
		// last segment carries an extension, so it is the document and is dropped.
		{"windows file base, relative ref", `C:\work\main.xsl`, sxsd, "C:/work/s.xsd"},
		{"windows dir base, relative ref", `C:\work\schemas`, sxsd, "C:/work/schemas/s.xsd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveSchemaURI(tc.ref, tc.base)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestResolveSchemaURIError verifies that an unresolvable URI reference (an
// invalid RFC 3986 reference such as "%zz.xsd" against an https base) is
// REJECTED with an error, and never filepath-collapsed into a
// corrupted, authority-dropped name like "https:/example.com/s/%zz.xsd".
func TestResolveSchemaURIError(t *testing.T) {
	for _, tc := range []struct {
		name string
		base string
		ref  string
	}{
		{"invalid ref https base", "https://example.com/s/main.xsl", "%zz.xsd"},
		{"invalid ref file base", "file:///tmp/s/main.xsd", "%zz.xsd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveSchemaURI(tc.ref, tc.base)
			require.Error(t, err)
		})
	}
}

// TestSchemaResolverFSForwardsNameVerbatim verifies that the FS adapter passes
// the name it receives straight to the byte-loader. The xsd compiler now hands
// Open the canonical nested-schema URI (resolved URI-aware on the xsd side), so
// the adapter must NOT rewrite it — any rewriting would corrupt an absolute
// cross-host URI such as https://cdn.example.com/part.xsd.
func TestSchemaResolverFSForwardsNameVerbatim(t *testing.T) {
	for _, name := range []string{
		"https://example.com/s/part.xsd",
		"https://cdn.example.com/part.xsd",
		"file:///tmp/s/part.xsd",
		"/tmp/s/part.xsd",
		"part.xsd",
	} {
		t.Run(name, func(t *testing.T) {
			var got string
			s := schemaResolverFS{
				ctx: context.Background(),
				load: func(_ context.Context, uri string) ([]byte, error) {
					got = uri
					return []byte("<x/>"), nil
				},
			}
			f, err := s.Open(name)
			require.NoError(t, err)
			_ = f.Close()
			require.Equal(t, name, got, "FS adapter must forward the canonical name unchanged")
		})
	}
}

var _ io.Closer = (*schemaResolverFile)(nil)
