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
