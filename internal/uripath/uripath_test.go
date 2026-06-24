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
