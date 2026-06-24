package uripath_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/stretchr/testify/require"
)

func TestHasWindowsDrivePrefix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{`C:\x`, true},
		{`c:/x`, true},
		{`D:`, true},
		{`Z:\a\b\c`, true},
		{`C:foo`, false}, // drive-relative, no separator
		{`/usr/x`, false},
		{`\\server\share`, false},
		{`http://x`, false}, // multi-letter scheme
		{`1:\x`, false},     // non-alpha drive
		{``, false},
		{`C`, false},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, uripath.HasWindowsDrivePrefix(tc.in), "in=%q", tc.in)
	}
}

func TestIsWindowsAbsolute(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{`C:\x`, true},
		{`C:/x`, true},
		{`D:`, true},
		{`\\server\share`, true},
		{`\rooted`, true},
		{`/usr/x`, false},
		{`relative\path`, false},
		{`C:foo`, false},
		{``, false},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, uripath.IsWindowsAbsolute(tc.in), "in=%q", tc.in)
	}
}

func TestIsPOSIXAbsolute(t *testing.T) {
	t.Parallel()
	require.True(t, uripath.IsPOSIXAbsolute("/usr/x"))
	require.True(t, uripath.IsPOSIXAbsolute("/"))
	require.False(t, uripath.IsPOSIXAbsolute("usr/x"))
	require.False(t, uripath.IsPOSIXAbsolute(`C:\x`))
	require.False(t, uripath.IsPOSIXAbsolute(""))
}

// IsAbsolutePath must be true for an absolute path under EITHER convention,
// regardless of the host OS. This is what makes a containment guard reject a
// POSIX "/etc/passwd" on Windows and a Windows "C:\..." on POSIX.
func TestIsAbsolutePath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{`/etc/passwd`, true},  // POSIX-absolute
		{`C:\windows`, true},   // Windows drive-absolute
		{`D:/data`, true},      // Windows drive-absolute, forward slash
		{`E:`, true},           // bare drive
		{`\\host\share`, true}, // UNC
		{`\abs`, true},         // backslash-rooted
		{`relative/x`, false},
		{`a.dtd`, false},
		{`C:foo`, false},
		{``, false},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, uripath.IsAbsolutePath(tc.in), "in=%q", tc.in)
	}
}

// JoinLocalBaseDir must always emit forward slashes, normalizing a Windows
// (backslash) base or ref so the result is identical on every OS. This is what
// keeps helium's URI-style resolvers from leaking '\' into their documented
// forward-slash output on Windows.
func TestJoinLocalBaseDir(t *testing.T) {
	t.Parallel()
	cases := []struct {
		baseDir string
		ref     string
		want    string
	}{
		{"/dir", "a.dtd", "/dir/a.dtd"},
		{"/dir/", "a.dtd", "/dir/a.dtd"},
		{`C:\dir`, "child.xml", "C:/dir/child.xml"},
		{`C:/dir`, `sub\child.xml`, "C:/dir/sub/child.xml"},
		{"/a/b", "../c/x", "/a/c/x"},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, uripath.JoinLocalBaseDir(tc.baseDir, tc.ref),
			"baseDir=%q ref=%q", tc.baseDir, tc.ref)
	}
}

// LocalBaseDir distinguishes a file base (drop the last segment) from a
// directory base (keep it), always in forward-slash form.
func TestLocalBaseDir(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"/work/main.xsl", "/work"},
		{"/work/schemas", "/work/schemas"}, // extensionless last segment = directory
		{"/work/schemas/", "/work/schemas"},
		{`C:\work\main.xsl`, "C:/work"},
		{`C:\work\schemas`, "C:/work/schemas"},
		{`C:\work\schemas\`, "C:/work/schemas"},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, uripath.LocalBaseDir(tc.in), "in=%q", tc.in)
	}
}

// SlashDir is path.Dir over a backslash-normalized input, so a Windows path
// yields a forward-slash parent on any OS.
func TestSlashDir(t *testing.T) {
	t.Parallel()
	require.Equal(t, "/a/b", uripath.SlashDir("/a/b/c.xml"))
	require.Equal(t, "schemas", uripath.SlashDir(`schemas\x.rng`))
	require.Equal(t, "C:/a", uripath.SlashDir(`C:\a\b.xsl`))
}

func TestWindowsToFileURI(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{`C:\a\b`, "file:///C:/a/b"},
		{`D:/ents/x.xml`, "file:///D:/ents/x.xml"},
		{`c:\x`, "file:///c:/x"},
		{`\\server\share\f.xml`, "file://server/share/f.xml"},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, uripath.WindowsToFileURI(tc.in), "in=%q", tc.in)
	}
}
