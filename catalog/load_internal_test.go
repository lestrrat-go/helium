package catalog

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// fileURIPathFor must strip the leading slash before a drive letter only on
// Windows. On POSIX "/C:/tmp/x" is a legitimate absolute path and must be left
// unchanged. goos is passed explicitly so both branches run deterministically
// on the Linux CI host.
func TestFileURIPath(t *testing.T) {
	tests := []struct {
		name   string
		goos   string
		path   string // path component of a file: URI
		expect string
	}{
		{
			name:   "posix absolute path unchanged",
			goos:   "linux",
			path:   "/usr/share/xml/catalog.xml",
			expect: filepath.FromSlash("/usr/share/xml/catalog.xml"),
		},
		{
			// Regression: on POSIX "/C:/tmp/..." is an absolute path and the
			// leading slash must NOT be stripped.
			name:   "posix drive-letter-like path stays absolute",
			goos:   "linux",
			path:   "/C:/tmp/catalog.xml",
			expect: filepath.FromSlash("/C:/tmp/catalog.xml"),
		},
		{
			name:   "windows drive letter strips leading slash",
			goos:   goosWindows,
			path:   "/C:/tmp/catalog.xml",
			expect: filepath.FromSlash("C:/tmp/catalog.xml"),
		},
		{
			name:   "windows drive letter lowercase",
			goos:   goosWindows,
			path:   "/d:/data/cat.xml",
			expect: filepath.FromSlash("d:/data/cat.xml"),
		},
		{
			name:   "windows non drive colon path unchanged",
			goos:   goosWindows,
			path:   "/ab:/x",
			expect: filepath.FromSlash("/ab:/x"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expect, fileURIPathFor(tc.goos, tc.path))
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
			got, isFileURI, err := catalogFilePath(ref)
			require.NoError(t, err)
			require.Equal(t, ref, got)
			require.False(t, isFileURI)
		})
	}
}

// catalogFilePath must treat the "localhost" host of a file: URI as the local
// machine regardless of case, since URI hosts are case-insensitive. An empty
// host denotes the local machine as well.
func TestCatalogFilePathLocalHost(t *testing.T) {
	tests := []struct {
		name   string
		ref    string
		expect string
	}{
		{
			name:   "empty host",
			ref:    "file:///etc/xml/catalog.xml",
			expect: filepath.FromSlash("/etc/xml/catalog.xml"),
		},
		{
			name:   "lowercase localhost",
			ref:    "file://localhost/etc/xml/catalog.xml",
			expect: filepath.FromSlash("/etc/xml/catalog.xml"),
		},
		{
			name:   "uppercase LOCALHOST",
			ref:    "file://LOCALHOST/etc/xml/catalog.xml",
			expect: filepath.FromSlash("/etc/xml/catalog.xml"),
		},
		{
			name:   "mixed case LocalHost",
			ref:    "file://LocalHost/etc/xml/catalog.xml",
			expect: filepath.FromSlash("/etc/xml/catalog.xml"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, isFileURI, err := catalogFilePath(tc.ref)
			require.NoError(t, err)
			require.Equal(t, tc.expect, got)
			require.True(t, isFileURI)
		})
	}
}

// catalogFilePath must reject a genuinely remote (non-local) host.
func TestCatalogFilePathRemoteHostRejected(t *testing.T) {
	_, _, err := catalogFilePath("file://example.com/etc/xml/catalog.xml")
	require.Error(t, err)
}

// catalogFilePath must reject an opaque file: URI such as "file:next.xml"
// (url.URL.Opaque is set, Path is empty). Such a reference has no local path,
// so it must error rather than silently fall through to reading the process
// working directory.
func TestCatalogFilePathOpaqueRejected(t *testing.T) {
	tests := []string{
		"file:next.xml",
		"file:sub/next.xml",
	}

	for _, ref := range tests {
		t.Run(ref, func(t *testing.T) {
			_, _, err := catalogFilePath(ref)
			require.Error(t, err)
		})
	}
}

// catalogFilePath must reject a file: URI whose path component is empty, such
// as "file://localhost" with no path. There is no local path to read.
func TestCatalogFilePathEmptyPathRejected(t *testing.T) {
	_, _, err := catalogFilePath("file://localhost")
	require.Error(t, err)
}

// catalogFilePath must reject a "file:////server/share" UNC URI. Its path
// component begins with "//"; on Windows fileURIPath would produce the UNC path
// \\server\share reaching a remote SMB host, defeating the local-only policy.
func TestCatalogFilePathUNCRejected(t *testing.T) {
	_, _, err := catalogFilePath("file:////server/share/catalog.xml")
	require.Error(t, err)
}

// url.Parse percent-decodes u.Path, so a "%5C"/"%5c" encoded backslash decodes
// to "/\server/share" which still becomes the UNC path \\server\share on
// Windows. catalogFilePath must reject these encoded forms too.
func TestCatalogFilePathEncodedBackslashUNCRejected(t *testing.T) {
	for _, ref := range []string{
		"file:///%5Cserver/share/catalog.xml",
		"file:///%5cserver/share/catalog.xml",
		"file:///%5C%5Cserver/share/catalog.xml",
	} {
		_, _, err := catalogFilePath(ref)
		require.Error(t, err, ref)
	}
}

// catalogFilePath must keep a POSIX "file:///C:/..." path absolute. The
// drive-letter slash strip is Windows-only; on POSIX "/C:/tmp/..." is a valid
// absolute path, so this asserts deterministically on the Linux CI host.
func TestCatalogFilePathPOSIXDriveLetterStaysAbsolute(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("POSIX-specific behavior")
	}
	got, isFileURI, err := catalogFilePath("file:///C:/tmp/catalog.xml")
	require.NoError(t, err)
	require.Equal(t, "/C:/tmp/catalog.xml", got)
	require.True(t, isFileURI)
}
