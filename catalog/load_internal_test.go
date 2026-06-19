package catalog

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// fileURIPath must convert a Windows "file:///C:/..." URI path into a
// drive-letter filesystem path while leaving POSIX absolute paths unchanged.
// The conversion logic is exercised directly so it runs on Linux CI.
func TestFileURIPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string // path component of a file: URI
		expect string
	}{
		{
			name:   "posix absolute path unchanged",
			path:   "/usr/share/xml/catalog.xml",
			expect: filepath.FromSlash("/usr/share/xml/catalog.xml"),
		},
		{
			name:   "windows drive letter strips leading slash",
			path:   "/C:/tmp/catalog.xml",
			expect: filepath.FromSlash("C:/tmp/catalog.xml"),
		},
		{
			name:   "windows drive letter lowercase",
			path:   "/d:/data/cat.xml",
			expect: filepath.FromSlash("d:/data/cat.xml"),
		},
		{
			name:   "non drive colon path unchanged",
			path:   "/ab:/x",
			expect: filepath.FromSlash("/ab:/x"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expect, fileURIPath(tc.path))
		})
	}
}

// catalogFilePath must treat a bare Windows drive-letter path as a local OS
// path rather than rejecting "C" as an unsupported URI scheme.
func TestCatalogFilePathDriveLetterIsLocal(t *testing.T) {
	tests := []string{
		`C:\tmp\catalog.xml`,
		`C:/tmp/catalog.xml`,
	}

	for _, ref := range tests {
		t.Run(ref, func(t *testing.T) {
			got, err := catalogFilePath(ref)
			require.NoError(t, err)
			require.Equal(t, ref, got)
		})
	}
}
