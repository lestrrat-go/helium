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
	return os.Open(name) //nolint:gosec,wrapcheck // intentional passthrough; see type doc
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

	return fileURIPathFor(runtime.GOOS, u.Path), nil
}

// fileURIPathFor is the OS-parameterized conversion of a "file:" URI path
// component into a local filesystem path. The drive-letter slash strip only
// applies on Windows; on POSIX "/C:/tmp/x" is a valid absolute path and is left
// untouched. goos is threaded explicitly so the conversion is deterministically
// testable on a non-Windows CI host.
func fileURIPathFor(goos, p string) string {
	if goos == goosWindows && len(p) >= 3 && p[0] == '/' && isASCIILetter(p[1]) && p[2] == ':' {
		p = p[1:]
	}
	return filepath.FromSlash(p)
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
