package c14n

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJoinURIReference(t *testing.T) {
	tests := []struct {
		name string
		base string
		ref  string
		want string
	}{
		// W3C xml-c14n11 §2.4 / libxml2 spec vectors.
		{"spec-102-e3", "../bar/", "foo", "../bar/foo"},
		{"spec3-d-inner", "..", "x", "../x"},
		{"spec3-d-outer", "..", "../x", "../../x"},
		{"spec2-102", "bar/", "foo", "bar/foo"},
		// Absolute base resolves abs-path reference (xmlbase-prop-2).
		{"prop2-e1", "http://xmlbase.example.org/xmlbase0/", "/xmlbase1/", "http://xmlbase.example.org/xmlbase1/"},
		// Absolute reference dominates.
		{"abs-ref-wins", "../bar/", "http://example.com/x", "http://example.com/x"},
		{"urn-ref-wins", "../bar/", "urn:foo", "urn:foo"},
		// Trailing-dot append (libxml2 forces upward traversal).
		{"dotdot-base-keeps-slash", "..", "y/", "../y/"},
		// Empty-path (query/fragment-only) reference keeps the base path (BASE-001).
		{"query-only-ref", "a/b", "?q=1", "a/b?q=1"},
		{"fragment-only-ref", "a/b", "#f", "a/b#f"},
		// Relative path that fully cancels yields empty, not "/" (BASE-002).
		{"relative-cancels-to-empty", "abc/", "../", ""},
		// Network-path reference keeps its authority (BASE-003).
		{"network-path-ref", "a/", "//h/x", "//h/x"},
		// Absolute base merge collapses consecutive slashes (BASE-004).
		{"absolute-base-double-slash", "http://h/a//b/", "c", "http://h/a/b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, joinURIReference(tt.base, tt.ref))
		})
	}
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
		{"spec3-d", []string{"..", "..", "x"}, "../../x"},
		{"prop2-e1", []string{"http://xmlbase.example.org/xmlbase0/", "/xmlbase1/"}, "http://xmlbase.example.org/xmlbase1/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, reduceXMLBase(tt.chain))
		})
	}
}

func TestNormalizeURIPath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"../bar/foo", "../bar/foo"},
		{"../../x", "../../x"},
		{"../x", "../x"},
		{"/c/", "/c/"},
		{"a/b/../c", "a/c"},
		{"foo/./bar", "foo/bar"},
		{"a//b", "a/b"},
		{"abc/../", ""},       // relative cancels to empty, not "/"
		{"/a//b/c", "/a/b/c"}, // collapse consecutive slashes
		{"foo/..", ""},
		{"a/b/..", "a/"},
		{"abc/", "abc/"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			require.Equal(t, tt.want, normalizeURIPath(tt.in))
		})
	}
}
