package c14n

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDocumentBaseURIWindowsDrive is the string-shaped (GOOS-independent)
// regression for the Windows C14N 1.1 xml:base fixup failure: a Windows-drive
// base URI must be turned into a proper forward-slash "file:" URI so the
// downstream RFC 3986 ".." relativization in relativizeURI counts segments
// correctly. Before the fix, "file://" was prepended to a backslash path
// ("file://D:\dir\doc.xml"), which url.Parse could not navigate, collapsing
// "../../x" into "..//x". A Windows-absolute base is a plain string, so the
// Windows branch is exercised on any OS; the POSIX cases confirm no regression.
func TestDocumentBaseURIWindowsDrive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		baseURI string
		want    string
	}{
		{"windows drive backslash", `D:\dir\sub\doc.xml`, "file:///D:/dir/sub/doc.xml"},
		{"windows drive forward slash", "D:/dir/sub/doc.xml", "file:///D:/dir/sub/doc.xml"},
		{"unc path", `\\host\share\doc.xml`, "file://host/share/doc.xml"},
		// An already-absolute URI is preserved untouched.
		{"absolute http uri", "http://example.com/dir/doc.xml", "http://example.com/dir/doc.xml"},
		{"empty base", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &canonicalizer{baseURI: tc.baseURI}
			require.Equal(t, tc.want, c.documentBaseURI())
		})
	}
}

// TestRelativizeURIDotDotForwardSlash verifies that relativizeURI emits a
// correct "../../" relative reference from forward-slash "file:" URIs — the
// value that must be byte-identical on POSIX and Windows. The Windows path
// failure ("..//x") only arose because documentBaseURI fed it a backslash base;
// given a proper forward-slash file URI, the relativization itself is platform
// independent.
func TestRelativizeURIDotDotForwardSlash(t *testing.T) {
	t.Parallel()

	// base of element <d>: foo/bar resolved twice up then x — see the
	// xmlbase-c14n11spec3-102 golden.
	base := "file:///work/dir/test/foo/bar"
	target := "file:///work/dir/x"
	require.Equal(t, "../../x", relativizeURI(base, target))
}
