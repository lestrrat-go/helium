// Package iofs provides the default [fs.FS] used by helium for opening
// external resources (DTDs, entities, schemas, XInclude targets).
package iofs

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

// goosWindows is the runtime.GOOS value for Windows. Drive-letter handling in
// "file:" URIs is gated on this so POSIX behavior is never altered.
const goosWindows = "windows"

// PermissiveRoot is an [fs.FS] backed by direct calls to [os.Open]. It
// is the explicit opposite of [os.Root]: rather than sandboxing access
// to a directory, it accepts any path the caller hands it — absolute,
// containing "..", anywhere on the filesystem — and forwards verbatim
// to os.Open without enforcing [fs.ValidPath].
//
// It exists to preserve helium's historical behavior of opening any
// path supplied to the parser, schema compilers, and XInclude processor.
//
// Note: the helium packages that consume this FS build the names they
// pass to Open via [filepath.Join] against the document's base URI /
// base dir, so those names may be absolute and may use OS-specific
// separators on Windows. A caller-supplied FS that enforces
// [fs.ValidPath] (such as [os.DirFS] or [os.OpenRoot]) will reject
// those names. Sandboxing the loader behind such an FS requires path
// normalization that is not yet performed by helium; until then,
// PermissiveRoot is the only configuration that accepts OS-style
// names end-to-end.
type PermissiveRoot struct{}

// Open implements [fs.FS].
func (PermissiveRoot) Open(name string) (fs.File, error) {
	// A non-file-scheme absolute URI (http://, https://, urn:, ...) is NOT a
	// local filesystem path. Handing it to os.Open opens a file whose NAME is
	// the literal URI, which fails with a platform-DEPENDENT errno — ENOENT on
	// Linux, EINVAL ("invalid argument") on macOS/Windows — that a resolution-
	// miss classifier (e.g. the xsd nested-schema loader's isBenignResolutionMiss)
	// cannot reliably treat as benign, so a legitimately-ABSENT optional
	// include/import hint pointing at a network URI becomes fatal on some hosts.
	// Report it as a canonical fs.ErrNotExist "not found" so it is classified as a
	// resolution miss CONSISTENTLY across platforms; a REQUIRED load still
	// surfaces the miss to its caller. A genuinely-local path (no scheme, or a
	// "file:" URI / Windows drive-letter path) still reaches os.Open and returns
	// its real errno, so a malformed LOCAL path stays fatal/normal.
	if s := uripath.URIScheme(name); s != "" && s != "file" {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return os.Open(name) //nolint:gosec,wrapcheck // intentional passthrough; see type doc
}

// DenyAll is an [fs.FS] that refuses every Open with [fs.ErrNotExist]. It is
// the default FS of a freshly constructed parser: no external resource (DTD,
// entity, ...) referenced by a document is loaded unless the caller explicitly
// supplies an FS via Parser.FS. Making "load nothing" the default keeps
// untrusted input from reaching the host filesystem (XXE / local-file
// disclosure). To restore the historical permissive behavior, pass
// [PermissiveRoot] (exposed publicly as helium.PermissiveFS).
type DenyAll struct{}

// Open implements [fs.FS]. It always fails with [fs.ErrNotExist] so callers
// that treat a missing resource as "skip silently" behave as if no file was
// present.
func (DenyAll) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// FileURIToPath converts a "file:" URI into a local filesystem path. It mirrors
// the conversion performed in package catalog (added in PR #602) so that other
// loaders — notably the XInclude processor — resolve "file:" hrefs identically.
//
// For "file:///abs/path" the host is empty and Path holds the (already
// percent-decoded) absolute path. A "file://host/path" with a non-localhost
// host is not addressable on the local filesystem and is rejected. URI hosts are
// case-insensitive, so an empty host and "localhost" in any case both denote the
// local machine. An opaque "file:" URI such as "file:next.xml" (u.Opaque set,
// empty u.Path) and a "file://localhost" URI with no path are rejected rather
// than letting an empty path read the process working directory.
func FileURIToPath(ref string) (string, error) {
	u, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("failed to parse file URI %q: %w", ref, err)
	}

	if u.Scheme != "file" {
		return "", fmt.Errorf("not a file URI: %q", ref)
	}

	if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
		return "", fmt.Errorf("non-local file URI host %q in %q", u.Host, ref)
	}

	if u.Opaque != "" || u.Path == "" {
		return "", fmt.Errorf("invalid file URI %q: no local path", ref)
	}

	// A "file:////server/share" URI parses to an empty host with a path that
	// begins with two separators; on Windows filepath.FromSlash would turn that
	// into a UNC path (\\server\share) reaching a remote SMB host, defeating the
	// local-only policy. Reject every UNC form outright.
	if IsUNCFileURIPath(u.Path) {
		return "", fmt.Errorf("UNC file URI %q is not a local path", ref)
	}

	return fileURIPathFor(runtime.GOOS, u.Path), nil
}

// IsUNCFileURIPath reports whether p — the (already percent-decoded) path
// component of a "file:" URI — denotes a UNC path. A UNC path begins with two
// path separators ("\\server\share"); on Windows filepath.FromSlash turns such
// a path into a remote SMB reference, defeating the local-only policy applied by
// the "file:" URI loaders.
//
// url.Parse percent-decodes u.Path, so the two leading separators may appear as
// any mix of forward slash and backslash: "//" (from "file:////server/share"),
// "/\" (from "file:///%5Cserver/share", since %5C/%5c decode to a backslash), or
// "\\" (from doubly-encoded forms). All such forms are detected here so a single
// encoded backslash cannot smuggle a UNC path past the "//"-only check.
//
// This is the single source of truth for the UNC rejection shared by
// [FileURIToPath], package catalog, and the helium CLI safety helpers.
func IsUNCFileURIPath(p string) bool {
	return len(p) >= 2 && isPathSep(p[0]) && isPathSep(p[1])
}

// isPathSep reports whether c is a path separator in either POSIX ("/") or
// Windows ("\") form. A decoded "file:" URI path may contain backslashes when
// the URI percent-encoded them as %5C/%5c.
func isPathSep(c byte) bool {
	return c == '/' || c == '\\'
}

// fileURIPathFor is the OS-parameterized conversion of a "file:" URI path
// component into a local filesystem path. The drive-letter slash strip only
// applies on Windows; on POSIX "/C:/tmp/x" is a valid absolute path and is left
// untouched. goos is threaded explicitly so the conversion is deterministically
// testable on a non-Windows CI host.
func fileURIPathFor(goos, p string) string {
	if goos == goosWindows && len(p) >= 3 && p[0] == '/' && xmlchar.IsASCIILetter(p[1]) && p[2] == ':' {
		p = p[1:]
	}
	return filepath.FromSlash(p)
}
