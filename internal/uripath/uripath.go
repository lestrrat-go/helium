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

import (
	"path"
	"strings"
)

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

// HasURIScheme reports whether s begins with an absolute-URI scheme followed by
// a colon, e.g. "http://host/p", "file:///x", "urn:isbn:0". The scheme syntax
// is RFC 3986's ALPHA *( ALPHA / DIGIT / "+" / "-" / "." ), but a SINGLE-letter
// scheme is deliberately NOT recognized so that a Windows drive-letter prefix
// ("C:\\x", "D:/x") is never mistaken for a URI scheme. Callers that need to
// treat a drive letter as a path must gate on HasWindowsDrivePrefix first; this
// helper exists so an absolute URI can be distinguished from a relative
// reference independent of runtime.GOOS.
func HasURIScheme(s string) bool {
	return URIScheme(s) != ""
}

// URIScheme returns the LOWERCASED absolute-URI scheme of s ("http", "https",
// "file", "urn", ...) or "" when s carries no scheme. The grammar mirrors
// [HasURIScheme] (RFC 3986 ALPHA *( ALPHA / DIGIT / "+" / "-" / "." )), and a
// SINGLE-letter scheme is deliberately NOT recognized so a Windows drive-letter
// prefix ("C:\\x", "D:/x") is never mistaken for a URI scheme. The result is
// lowercased because URI schemes are case-insensitive (RFC 3986 §3.1), so a
// caller can compare against a fixed scheme name without re-normalizing.
func URIScheme(s string) string {
	if len(s) < 2 || !IsWindowsDriveLetter(s[0]) {
		return ""
	}
	i := 1
	for i < len(s) {
		c := s[i]
		switch {
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.':
			i++
		case c == ':':
			// A single-character scheme ("C:") is a Windows drive letter, not a
			// URI scheme. Require at least two scheme characters.
			if i < 2 {
				return ""
			}
			return strings.ToLower(s[:i])
		default:
			return ""
		}
	}
	return ""
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
	slashed := ToSlash(s)
	if HasWindowsDrivePrefix(s) {
		return "file:///" + slashed
	}
	// UNC: "//server/share" already begins with the two slashes after
	// normalization, so "file:" + slashed yields "file://server/share".
	return "file:" + slashed
}

// ToSlash replaces every backslash with a forward slash. Unlike
// filepath.ToSlash, the replacement is unconditional (not gated on
// runtime.GOOS), so a Windows path is normalized — and the normalization is
// testable — on POSIX as well.
func ToSlash(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] == '\\' {
			b[i] = '/'
		}
	}
	return string(b)
}

// JoinLocalBaseDir resolves a relative local-filesystem ref against the
// directory of a local-filesystem base path, returning a FORWARD-SLASH result
// on every OS. Both inputs are normalized with [ToSlash] first, then joined
// with path.Join (slash semantics), so the resolution never depends on
// runtime.GOOS or filepath.Separator. This keeps the documented forward-slash
// contract of helium's URI-style resolvers on Windows, where filepath.Join
// would otherwise emit backslashes and corrupt a POSIX-shaped base.
//
// The base may be a full file path ("/a/b/style.xsl") or a directory-like path
// ("/a/b"); the caller is responsible for choosing baseDir accordingly (see the
// resolvers that pass a directory derived from the base). ref must already be
// known not to be absolute under either OS convention; an absolute ref should
// be returned verbatim by the caller before reaching here.
func JoinLocalBaseDir(baseDir, ref string) string {
	return path.Join(ToSlash(baseDir), ToSlash(ref))
}

// LocalBaseDir derives the containing directory of a local-filesystem base
// path, in FORWARD-SLASH form. If the base already names a directory (it ends
// in a separator) the trailing separator is trimmed; if the base's last segment
// carries no '.', the whole base is treated as a directory; otherwise the last
// segment is dropped (path.Dir). The result is always slash-separated so it is
// stable across OSes.
func LocalBaseDir(base string) string {
	slashed := ToSlash(base)
	if len(slashed) > 0 && slashed[len(slashed)-1] == '/' {
		trimmed := slashed
		for len(trimmed) > 1 && trimmed[len(trimmed)-1] == '/' {
			trimmed = trimmed[:len(trimmed)-1]
		}
		return trimmed
	}
	last := path.Base(slashed)
	if !containsDot(last) {
		return slashed
	}
	return path.Dir(slashed)
}

// SlashDir returns the directory portion of a forward-slash path, equivalent to
// path.Dir but provided here so callers that already work in slash space (e.g.
// keys into an fs.FS) can derive a parent directory without reaching for the
// OS-dependent filepath.Dir, which would reintroduce backslashes on Windows.
func SlashDir(p string) string {
	return path.Dir(ToSlash(p))
}

// SlashClean is path.Clean over a backslash-normalized input, so a Windows path
// (or a path with mixed separators, as produced when a forward-slash URI ref is
// joined with a backslash OS base) is collapsed using forward-slash dot-segment
// semantics on every OS. Unlike filepath.Clean it never emits '\' and never
// applies Windows volume/UNC handling, so the result is byte-identical across
// platforms.
func SlashClean(p string) string {
	return path.Clean(ToSlash(p))
}

// SlashRel computes the relative reference from the directory tree rooted at
// baseDir to target, using pure forward-slash (RFC 3986 dot-segment) semantics
// — the slash-space analogue of filepath.Rel. Both inputs are normalized with
// [ToSlash] and [path.Clean] first, so the computation never depends on
// runtime.GOOS or filepath.Separator (filepath.Rel diverges on Windows: it
// splits on '\', mishandles a path that already uses '/', and applies
// drive-letter rules). baseDir and target must agree on rootedness (both
// absolute or both relative); when they don't, target is returned cleaned and
// unchanged, mirroring filepath.Rel's error path where the caller falls back to
// the absolute target.
//
// The result is the minimal sequence of ".." and forward path segments that,
// resolved against baseDir, yields target. It is the value used verbatim as an
// xml:base relative reference, so it MUST be byte-identical on POSIX and
// Windows.
func SlashRel(baseDir, target string) string {
	base := SlashClean(baseDir)
	targ := SlashClean(target)
	if base == targ {
		return "."
	}

	baseAbs := IsPOSIXAbsolute(base)
	targAbs := IsPOSIXAbsolute(targ)
	if baseAbs != targAbs {
		// Cannot relativize across rootedness; caller treats this like
		// filepath.Rel's error and keeps the (cleaned) target.
		return targ
	}

	baseSeg := splitSlash(base)
	targSeg := splitSlash(targ)

	// Drop the longest common prefix of segments.
	i := 0
	for i < len(baseSeg) && i < len(targSeg) && baseSeg[i] == targSeg[i] {
		i++
	}

	var out []string
	for j := i; j < len(baseSeg); j++ {
		// A ".." in the remaining base means base escaped above a point that
		// target shares, which filepath.Rel reports as impossible; fall back.
		if baseSeg[j] == ".." {
			return targ
		}
		out = append(out, "..")
	}
	out = append(out, targSeg[i:]...)
	if len(out) == 0 {
		return "."
	}
	return path.Join(out...)
}

// SlashCommonDir returns the longest common directory prefix of the directories
// of a and b, in forward-slash form — the slash-space analogue of the previous
// filepath-based commonAncestorDir. Inputs are normalized with [ToSlash] and
// [path.Clean]; the directory of each (its last segment dropped) is compared
// segment by segment. Returns "." when there is no shared directory prefix.
func SlashCommonDir(a, b string) string {
	aDir := SlashDir(SlashClean(a))
	bDir := SlashDir(SlashClean(b))
	aParts := splitSlash(aDir)
	bParts := splitSlash(bDir)

	n := min(len(aParts), len(bParts))
	common := 0
	for i := range n {
		if aParts[i] != bParts[i] {
			break
		}
		common = i + 1
	}
	if common == 0 {
		return "."
	}
	return joinSlashSegments(aParts[:common])
}

// joinSlashSegments rejoins segments produced by splitSlash, preserving the
// leading-slash root marker. path.Join would silently drop a leading "" segment
// (and with it the absolute-path root), so reconstruct it explicitly: an empty
// first segment means the path was rooted ("/"+rest), otherwise it is relative.
func joinSlashSegments(segs []string) string {
	if len(segs) == 0 {
		return "."
	}
	if segs[0] == "" {
		if len(segs) == 1 {
			return "/"
		}
		return "/" + path.Join(segs[1:]...)
	}
	return path.Join(segs...)
}

// splitSlash splits a cleaned forward-slash path into its non-empty segments,
// preserving a leading-slash marker as an empty first segment so an absolute
// path keeps its root during segment comparison.
func splitSlash(p string) []string {
	if p == "" || p == "." {
		return nil
	}
	rooted := p[0] == '/'
	trimmed := p
	for len(trimmed) > 0 && trimmed[0] == '/' {
		trimmed = trimmed[1:]
	}
	var segs []string
	for s := range strings.SplitSeq(trimmed, "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	if rooted {
		return append([]string{""}, segs...)
	}
	return segs
}

func containsDot(s string) bool {
	for i := range len(s) {
		if s[i] == '.' {
			return true
		}
	}
	return false
}
