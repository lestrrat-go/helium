package xslt3

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// part is the relative schema-location filename used across these cases.
const part = "part.xsd"

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
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, resolveSchemaURI(tc.ref, tc.base))
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
