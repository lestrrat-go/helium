package catalog_test

import (
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium/internal/catalog"
	"github.com/stretchr/testify/require"
)

// ResolveURI must resolve references in URI space when the base carries a URI
// scheme. A path-absolute reference such as "/abs/asset.xml" against a
// "file:///..." base must stay in "file:" URI space ("file:///abs/asset.xml"),
// not collapse to the bare local path "/abs/asset.xml".
const fileCatalogURI = "file:///tmp/catalog.xml"

func TestResolveURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		base    string
		value   string
		expect  string
		wantErr bool
	}{
		{
			name:   "empty value",
			base:   fileCatalogURI,
			value:  "",
			expect: "",
		},
		{
			name:   "value already has scheme",
			base:   fileCatalogURI,
			value:  "http://example.com/x.xsd",
			expect: "http://example.com/x.xsd",
		},
		{
			// Regression: a path-absolute reference against a file: base must
			// resolve in URI space, keeping the file: scheme.
			name:   "path-absolute ref against file base",
			base:   fileCatalogURI,
			value:  "/abs/asset.xml",
			expect: "file:///abs/asset.xml",
		},
		{
			name:   "relative ref against file base",
			base:   fileCatalogURI,
			value:  "asset.xml",
			expect: "file:///tmp/asset.xml",
		},
		{
			name:   "absolute-uri ref against file base",
			base:   fileCatalogURI,
			value:  "http://example.com/asset.xml",
			expect: "http://example.com/asset.xml",
		},
		{
			// Non-URI local-path base keeps the original OS-path behavior: a
			// path-absolute reference is returned unchanged.
			name:   "path-absolute ref against local-path base",
			base:   filepath.FromSlash("/tmp/catalog.xml"),
			value:  filepath.FromSlash("/abs/asset.xml"),
			expect: filepath.FromSlash("/abs/asset.xml"),
		},
		{
			// Non-URI local-path base: a relative reference joins against the
			// base directory as an OS path.
			name:   "relative ref against local-path base",
			base:   filepath.FromSlash("/tmp/catalog.xml"),
			value:  "asset.xml",
			expect: filepath.FromSlash("/tmp/asset.xml"),
		},
		{
			name:   "empty base relative value",
			base:   "",
			value:  "asset.xml",
			expect: "asset.xml",
		},
		{
			// A malformed reference (invalid percent-encoding) against a URI
			// base must surface an error and yield no usable mapping, rather
			// than silently returning the raw value.
			name:    "malformed value against file base",
			base:    fileCatalogURI,
			value:   "%zz",
			expect:  "",
			wantErr: true,
		},
		{
			// A malformed ABSOLUTE URI carrying a scheme must not bypass
			// validation. url.Parse rejects the invalid percent-encoding, so an
			// error is surfaced and no usable mapping is produced.
			name:    "malformed absolute uri with scheme",
			base:    fileCatalogURI,
			value:   "http://example.com/%zz",
			expect:  "",
			wantErr: true,
		},
		{
			// A well-formed absolute URI with a scheme still resolves, even with
			// an empty base.
			name:   "well-formed absolute uri empty base",
			base:   "",
			value:  "http://example.com/asset.xml",
			expect: "http://example.com/asset.xml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := catalog.ResolveURI(tc.base, tc.value)
			if tc.wantErr {
				require.Error(t, err)
				require.Equal(t, "", got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expect, got)
		})
	}
}

// HasScheme must not misread a single-letter Windows drive (e.g. "D:") as a URI
// scheme. These string inputs are GOOS-independent so the Windows behavior is
// covered on Linux CI.
func TestHasScheme(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"http://example.com/x", true},
		{"file:///tmp/x", true},
		{"urn:isbn:0", true},
		{`C:\catalog.xml`, false},       // Windows drive, not a scheme
		{`D:/share/catalog.xml`, false}, // Windows drive, forward slash
		{"C:", false},
		{"/tmp/catalog.xml", false},
		{"relative.xml", false},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, catalog.HasScheme(tc.in), "in=%q", tc.in)
	}
}

// ResolveURI with a Windows drive-letter base must treat the base as a local
// path (not a URI), and a Windows-absolute value must be returned unchanged.
// These string inputs exercise the Windows path on any GOOS.
func TestResolveURIWindowsShapes(t *testing.T) {
	t.Parallel()

	t.Run("windows-drive base resolves relative ref as local path", func(t *testing.T) {
		got, err := catalog.ResolveURI(`C:\dir\catalog.xml`, "asset.xml")
		require.NoError(t, err)
		// filepath.Join/Dir use the host separator; on POSIX this is
		// "C:\\dir/asset.xml". The key property is that the drive letter is
		// NOT leaked into URI space ("c:///...").
		require.NotContains(t, got, "c://")
		require.NotContains(t, got, "C://")
		require.Contains(t, got, "asset.xml")
	})

	t.Run("windows-absolute value returned unchanged", func(t *testing.T) {
		got, err := catalog.ResolveURI("/tmp/catalog.xml", `C:\abs\asset.xml`)
		require.NoError(t, err)
		require.Equal(t, `C:\abs\asset.xml`, got)
	})

	t.Run("posix-absolute value against local base returned unchanged on any OS", func(t *testing.T) {
		// On Windows filepath.IsAbs("/abs/x") is false; uripath.IsAbsolutePath
		// keeps it absolute so it is not mis-joined against the base dir.
		got, err := catalog.ResolveURI(`C:\dir\catalog.xml`, "/abs/asset.xml")
		require.NoError(t, err)
		require.Equal(t, "/abs/asset.xml", got)
	})
}
