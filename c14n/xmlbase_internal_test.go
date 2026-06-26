package c14n

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	pathUpX   = "../x"
	pathUpUpX = "../../x"
	urnBase   = "urn:base"
)

func TestJoinURIReference(t *testing.T) {
	// Every `want` below was produced by the real libxml2 xmlBuildURI (v2.9.14),
	// invoked exactly as xmlC14NFixupBaseAttr does (with the trailing-"." → "/"
	// append on the base), so this is a byte-for-byte conformance check.
	tests := []struct {
		name string
		base string
		ref  string
		want string
	}{
		// W3C xml-c14n11 §2.4 / libxml2 spec vectors.
		{"spec-102-e3", "../bar/", "foo", "../bar/foo"},
		{"spec3-d-inner", "..", "x", pathUpX},
		{"spec3-d-outer", "..", pathUpX, pathUpUpX},
		{"spec2-102", "bar/", "foo", "bar/foo"},
		// Absolute base resolves abs-path reference (xmlbase-prop-2).
		{"prop2-e1", "http://xmlbase.example.org/xmlbase0/", "/xmlbase1/", "http://xmlbase.example.org/xmlbase1/"},
		// Absolute reference dominates.
		{"abs-ref-wins", "../bar/", "http://example.com/x", "http://example.com/x"},
		{"urn-ref-wins", "../bar/", "urn:foo", "urn:foo"},
		// Trailing-dot append (libxml2 forces upward traversal).
		{"dotdot-base-keeps-slash", "..", "y/", "../y/"},
		// Empty-path (query/fragment-only) reference keeps the base path.
		{"query-only-ref", "a/b", "?q=1", "a/b?q=1"},
		{"fragment-only-ref", "a/b", "#f", "a/b#f"},
		// Relative path that fully cancels yields empty, not "/".
		{"relative-cancels-to-empty", "abc/", "../", ""},
		// Network-path reference keeps its authority.
		{"network-path-ref", "a/", "//h/x", "//h/x"},
		// Absolute base merge collapses consecutive slashes.
		{"absolute-base-double-slash", "http://h/a//b/", "c", "http://h/a/b/c"},
		// The first path segment survives a trailing ".." (libxml2 quirk).
		{"first-segment-survives-trailing-dotdot", "a/b/", "../..", "a/.."},
		{"foo-bar-up-up", "foo/bar", pathUpUpX, pathUpX},
		{"abc-up-up-d", "a/b/c/", "../../d", "a/d"},
		// A space round-trips to %20, but an encoded delimiter is decoded: libxml2
		// unescapes the path, normalizes, then re-escapes.
		{"space-roundtrips", "a%20b/", "c", "a%20b/c"},
		{"encoded-slash-decoded", "a%2fb/", "c", "a/b/c"},
		{"encoded-dot-decoded", "a%2Eb/", "c", "a.b/c"},
		{"encoded-slash-dotdot", "%2F..%2Fx/", "c", "/x/c"},
		// Empty-but-present authority (file:///) is preserved.
		{"file-empty-authority", "file:///a/b", "c", "file:///a/c"},
		// Opaque base (urn:) payload is treated as the path.
		{"opaque-base-empty-ref", urnBase, "", urnBase},
		{"opaque-base-query", urnBase, "?q=1", "urn:base?q=1"},
		{"opaque-base-relative", urnBase, "x", "urn:x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, faithful := joinURIReference(tt.base, tt.ref)
			require.Equal(t, tt.want, got)
			require.True(t, faithful, "well-formed input must join faithfully")
		})
	}
}

func TestFaithfulXMLBaseValue(t *testing.T) {
	// Degenerate / malformed standalone values are rejected, even as a lone term.
	// Includes raw whitespace, which url.Parse tolerates but libxml2 rejects.
	for _, v := range []string{"//", "///", "urn://", "http://%", "a b", "urn:foo bar", "http://h/a b"} {
		require.False(t, faithfulXMLBaseValue(v), "%q should be unfaithful", v)
	}
	// Well-formed values (including empty-authority file:/// and protocol-relative
	// //host) are accepted.
	for _, v := range []string{"", "a/b", "../x", "/abs/", "http://h/p", "file:///a", "//host/p", urnBase} {
		require.True(t, faithfulXMLBaseValue(v), "%q should be faithful", v)
	}
}

func TestReduceXMLBaseUnfaithful(t *testing.T) {
	// A lone degenerate term (no join) is flagged.
	_, ok := reduceXMLBase([]string{"//"})
	require.False(t, ok, "lone degenerate term must be unfaithful")
	// A degenerate innermost absolute-scheme term (join fast-path) is flagged.
	_, ok = reduceXMLBase([]string{"a/", "urn://"})
	require.False(t, ok, "degenerate innermost term must be unfaithful")
	// Well-formed chains stay faithful.
	_, ok = reduceXMLBase([]string{"a/b/", "../c"})
	require.True(t, ok)
}

func TestReduceXMLBase(t *testing.T) {
	tests := []struct {
		name  string
		chain []string
		want  string
	}{
		{"single-absolute-path", []string{"/c/"}, "/c/"},
		{"single-relative", []string{"foo/bar"}, "foo/bar"},
		{"spec-102-e3", []string{"../bar/", "foo"}, "../bar/foo"},
		{"spec3-d", []string{"..", "..", "x"}, pathUpUpX},
		{"prop2-e1", []string{"http://xmlbase.example.org/xmlbase0/", "/xmlbase1/"}, "http://xmlbase.example.org/xmlbase1/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := reduceXMLBase(tt.chain)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeURIPath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"../bar/foo", "../bar/foo"},
		{pathUpUpX, pathUpUpX},
		{pathUpX, pathUpX},
		{"/c/", "/c/"},
		{"a/b/../c", "a/c"},
		{"foo/./bar", "foo/bar"},
		{"a//b", "a/b"},
		{"abc/../", ""},       // relative cancels to empty, not "/"
		{"/a//b/c", "/a/b/c"}, // collapse consecutive slashes
		{"foo/..", ""},
		{"a/b/..", "a/"},
		{"abc/", "abc/"},
		{"a/b/../..", "a/.."}, // first segment survives trailing ".."
		{"a/../..", ".."},
		{"foo/../../x", pathUpX},
		{"/../x", "/x"}, // leading "/../" discarded on absolute path
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			require.Equal(t, tt.want, normalizeURIPath(tt.in))
		})
	}
}
