// Package uripath provides small, OS-independent helpers for distinguishing
// native filesystem paths from URI references, and for converting native
// Windows/POSIX absolute paths into "file:" URIs.
//
// Every helper here decides purely from the STRING shape of its input — it
// never consults runtime.GOOS or filepath separators. This is deliberate: the
// helium URI/path resolution code must behave the same way on every platform
// (so that, e.g., a Windows drive-letter path embedded in a catalog or a
// stylesheet base URI is recognized even when helium runs on POSIX), and so
// that the Windows-specific behavior is exercised by the Linux test suite.
package uripath

// IsWindowsDriveLetter reports whether b is an ASCII letter usable as a Windows
// drive letter ([A-Za-z]).
func IsWindowsDriveLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// HasWindowsDrivePrefix reports whether s begins with a Windows drive-letter
// specifier: an ASCII letter, a colon, and then either a path separator
// ('/' or '\\') or the end of the string. Examples that match: "C:\\x",
// "c:/x", "D:". A lone "C:foo" (drive-relative, no separator) does NOT match,
// matching the conservative shape used elsewhere in the tree.
func HasWindowsDrivePrefix(s string) bool {
	if len(s) < 2 {
		return false
	}
	if !IsWindowsDriveLetter(s[0]) || s[1] != ':' {
		return false
	}
	if len(s) == 2 {
		return true
	}
	return s[2] == '/' || s[2] == '\\'
}

// IsWindowsAbsolute reports whether s is a Windows-style absolute path: a
// drive-letter path ("C:\\x", "C:/x", "C:") or a UNC / rooted path beginning
// with a backslash ("\\\\server\\share", "\\x").
func IsWindowsAbsolute(s string) bool {
	if HasWindowsDrivePrefix(s) {
		return true
	}
	return len(s) > 0 && s[0] == '\\'
}

// IsPOSIXAbsolute reports whether s is a POSIX-style absolute path: it begins
// with a single forward slash. This is GOOS-independent on purpose so a
// "/etc/passwd"-shaped reference is recognized as absolute even on Windows.
func IsPOSIXAbsolute(s string) bool {
	return len(s) > 0 && s[0] == '/'
}

// IsAbsolutePath reports whether s is absolute under EITHER POSIX or Windows
// conventions, independent of the host OS. Use this in security/containment
// guards so a path that is absolute on the "other" OS is still rejected.
func IsAbsolutePath(s string) bool {
	return IsPOSIXAbsolute(s) || IsWindowsAbsolute(s)
}

// WindowsToFileURI converts a Windows absolute path into a "file:" URI. It
// normalizes backslashes to forward slashes and prefixes "file:///" for a
// drive-letter path ("C:\\a\\b" -> "file:///C:/a/b") or "file://" for a UNC
// path ("\\\\server\\share" -> "file://server/share"). s MUST be a Windows
// absolute path (see IsWindowsAbsolute); callers gate on that first.
func WindowsToFileURI(s string) string {
	slashed := toSlash(s)
	if HasWindowsDrivePrefix(s) {
		return "file:///" + slashed
	}
	// UNC: "//server/share" already begins with the two slashes after
	// normalization, so "file:" + slashed yields "file://server/share".
	return "file:" + slashed
}

// toSlash replaces every backslash with a forward slash. It is the
// OS-independent equivalent of filepath.ToSlash on Windows, applied
// unconditionally so the conversion is testable on POSIX.
func toSlash(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] == '\\' {
			b[i] = '/'
		}
	}
	return string(b)
}
