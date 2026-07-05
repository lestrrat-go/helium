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

func TestHasURIScheme(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"http://host/p", true},
		{"https://host/p", true},
		{"file:///x", true},
		{"urn:isbn:0", true},
		{"a+b-c.d:rest", true},
		{`C:\x`, false},   // single-letter scheme is a Windows drive letter
		{`C:/x`, false},   // same, with forward slash
		{`D:`, false},     // bare drive
		{"a.dtd", false},  // relative reference
		{"/abs/x", false}, // POSIX absolute path, no scheme
		{`\\srv\s`, false},
		{"", false},
		{"http", false}, // no colon
		{":nohead", false},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, uripath.URIScheme(tc.in) != "", "HasURIScheme(%q)", tc.in)
		require.Equalf(t, tc.want, uripath.HasURIScheme(tc.in), "in=%q", tc.in)
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

// SlashRel must compute a forward-slash relative reference (RFC 3986
// dot-segment semantics) identically on POSIX and Windows, NEVER using
// filepath.Rel. Windows drive-rooted and mixed-separator inputs are plain
// strings, so the Windows behavior is exercised on any GOOS.
func TestSlashRel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		baseDir string
		target  string
		want    string
	}{
		// xml:base relativization cases mirrored from the XInclude base.xml golden.
		{"docs/one", "docs/one/two/three/four", "two/three/four"},
		{"docs/one", "ents/one/two2", "../../ents/one/two2"},
		{"docs/one", "ents/one2/two", "../../ents/one2/two"},
		// Absolute POSIX inputs keep their root through segment comparison.
		{"/a/b/docs/one", "/a/b/ents/one/two2", "../../ents/one/two2"},
		// Mixed separators (forward-slash URI ref joined against a backslash OS
		// base, as happens on Windows) must still relativize correctly.
		{`docs\one`, "docs/one/two/three/four", "two/three/four"},
		{`D:/a/b/docs/one`, `D:\a\b\ents\one\two2`, "../../ents/one/two2"},
		// Identical paths collapse to ".".
		{"a/b", "a/b", "."},
		// Rootedness mismatch falls back to the cleaned target (filepath.Rel error path).
		{"/a/b", "c/d", "c/d"},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, uripath.SlashRel(tc.baseDir, tc.target),
			"baseDir=%q target=%q", tc.baseDir, tc.target)
	}
}

// SlashCommonDir must find the longest common directory prefix in forward-slash
// form, preserving the leading-slash root (path.Join would otherwise drop it).
func TestSlashCommonDir(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a    string
		b    string
		want string
	}{
		{"/x/y/ents/inc.xml", "/x/y/docs/doc.xml", "/x/y"},
		{"x/y/ents/inc.xml", "x/y/docs/doc.xml", "x/y"},
		// Windows drive paths share the drive segment; backslashes are normalized.
		{`D:\x\y\ents\inc.xml`, "D:/x/y/docs/doc.xml", "D:/x/y"},
		// No shared directory prefix.
		{"/a/x.xml", "/b/y.xml", "/"},
		{"a/x.xml", "b/y.xml", "."},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, uripath.SlashCommonDir(tc.a, tc.b), "a=%q b=%q", tc.a, tc.b)
	}
}

func TestSlashClean(t *testing.T) {
	t.Parallel()
	require.Equal(t, "/a/c", uripath.SlashClean("/a/b/../c"))
	require.Equal(t, "a/b", uripath.SlashClean(`a\b`))
	require.Equal(t, "C:/a/b", uripath.SlashClean(`C:\a\.\b`))
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

func TestURIScheme(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"http://host/p", "http"},
		{"HTTPS://host/p", "https"}, // lowercased
		{"file:///x", "file"},
		{"urn:isbn:0", "urn"},
		{"a+b-c.d:rest", "a+b-c.d"},
		{`C:\x`, ""},    // single-letter scheme is a Windows drive letter
		{`C:/x`, ""},    // same, with forward slash
		{`D:`, ""},      // bare drive
		{"rel.dtd", ""}, // relative reference
		{"/abs/x", ""},  // POSIX absolute path, no scheme
		{`\\srv\s`, ""},
		{"", ""},
		{"http", ""}, // no colon
		{":nohead", ""},
	}
	for _, tc := range cases {
		require.Equalf(t, tc.want, uripath.URIScheme(tc.in), "in=%q", tc.in)
	}
}
