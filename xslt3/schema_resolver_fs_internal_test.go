package xslt3

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// part is the relative schema-location filename used across these cases.
const part = "part.xsd"

// TestResolveSchemaURI exercises the URI-aware resolution of a schema-location
// against a base URI directly, including the "../" dot-segment case that the
// xsd compiler's path-traversal sandbox prevents from being reached end to end.
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
		{"file sibling", "file:///tmp/s/main.xsd", part, "file:///tmp/s/part.xsd"},
		{"file parent", "file:///tmp/s/sub/main.xsd", "../part.xsd", "file:///tmp/s/part.xsd"},
		{"absolute ref unchanged", "file:///tmp/s/main.xsd", "https://other/x.xsd", "https://other/x.xsd"},
		{"empty base", "", part, part},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, resolveSchemaURI(tc.ref, tc.base))
		})
	}
}

// TestSchemaResolverFSResolveName verifies that the FS adapter recovers the
// canonical nested-schema URI from the (filepath-collapsed) name the xsd
// compiler hands to Open, using the stored original base URI rather than any
// ambiguous string repair of the collapsed form.
func TestSchemaResolverFSResolveName(t *testing.T) {
	for _, tc := range []struct {
		name    string
		baseURI string
		ref     string // relative schema-location the xsd compiler joins
		want    string
	}{
		// name == filepath.Join(filepath.Dir(baseURI), ref) is how the xsd
		// compiler forms the Open argument; resolveName must invert it.
		{"file sibling", "file:///tmp/s/main.xsd", part, "file:///tmp/s/part.xsd"},
		{"file subdir", "file:///tmp/s/main.xsd", "sub/part.xsd", "file:///tmp/s/sub/part.xsd"},
		{"http sibling", "https://example.com/s/main.xsd", part, "https://example.com/s/part.xsd"},
		{"http subdir", "https://example.com/s/main.xsd", "sub/part.xsd", "https://example.com/s/sub/part.xsd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := schemaResolverFS{baseURI: tc.baseURI}
			name := filepath.Join(filepath.Dir(tc.baseURI), tc.ref)
			require.Equal(t, tc.want, s.resolveName(name))
		})
	}
}

// TestSchemaResolverFSLocalPathVerbatim verifies that when the base URI is a
// genuine local filesystem path (no scheme), the name is forwarded unchanged so
// local-schema resolution and the default-deny behavior are unaffected.
func TestSchemaResolverFSLocalPathVerbatim(t *testing.T) {
	s := schemaResolverFS{baseURI: "/tmp/s/main.xsd"}
	name := filepath.Join("/tmp/s", part)
	require.Equal(t, name, s.resolveName(name))
}
