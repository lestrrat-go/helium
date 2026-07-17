package iofs

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lestrrat-go/helium/internal/uripath"
)

// ConfinedDir is an [fs.FS] that opens files only at or below a single root
// directory. Unlike [PermissiveRoot] it refuses any name that resolves outside
// root, so even with external-resource loading enabled an attacker-supplied
// SYSTEM identifier ("/etc/passwd", "../../secret") cannot exfiltrate arbitrary
// local files.
//
// It accepts the names helium's parser hands an fs.FS: a plain relative or
// absolute filesystem path, or a "file:" URI. A name carrying any other URI
// scheme (http, https, ...) is refused, so this FS never reaches the network.
// An absolute in-root name is served directly (no reliance on the parser's
// base-relative retry), which is what lets ConfinedDir be rooted at an
// ARBITRARY directory rather than only the document's own directory.
//
// Confinement is enforced with [os.Root] (os.OpenRoot, Go 1.24+): a lexical
// within-root check rejects a "../"- or absolute-path escape, and os.Root then
// refuses any open whose path component is a symlink resolving outside root. So
// ConfinedDir is both path-escape-safe AND a symlink sandbox — stronger than a
// bare [os.DirFS], which follows an in-root symlink out of its root.
type ConfinedDir struct {
	root string // absolute, cleaned directory path
	// err records a failure to resolve root to an absolute path at
	// construction (filepath.Abs fails when os.Getwd fails, e.g. the working
	// directory was removed). It fails closed: every Open returns it rather
	// than opening against a relative root that would resolve against whatever
	// working directory happens to be current at Open time.
	err error
}

// NewConfinedDir returns a ConfinedDir rooted at dir. dir is resolved to an
// absolute, cleaned path; a relative dir is taken relative to the process
// working directory at call time. If that resolution fails (filepath.Abs
// returns an error, e.g. os.Getwd fails because the working directory was
// removed) the error is retained and every Open returns it — the FS fails
// closed rather than fall back to a relative root that would be resolved
// against a possibly-different working directory at Open time.
func NewConfinedDir(dir string) ConfinedDir {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ConfinedDir{err: err}
	}
	return ConfinedDir{root: abs}
}

// Open implements [fs.FS].
func (c ConfinedDir) Open(name string) (fs.File, error) {
	if c.err != nil {
		return nil, &fs.PathError{Op: opOpen, Path: name, Err: c.err}
	}
	p, err := c.localPath(name)
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(c.root, p)
	}
	p = filepath.Clean(p)
	if !withinDir(c.root, p) {
		return nil, &fs.PathError{Op: opOpen, Path: name, Err: fs.ErrPermission}
	}
	rel, err := filepath.Rel(c.root, p)
	if err != nil {
		return nil, &fs.PathError{Op: opOpen, Path: name, Err: fs.ErrPermission}
	}

	// The lexical withinDir check only constrains the CLEANED path; os.Open would
	// then follow symlinks, so a "leak -> /etc/passwd" link living INSIDE root
	// would escape it. os.Root confines traversal: it refuses any path component
	// that is a symlink resolving outside root. Links that stay within root still
	// resolve.
	root, err := os.OpenRoot(c.root)
	if err != nil {
		return nil, err //nolint:wrapcheck // caller reports raw error
	}
	f, err := root.Open(rel)
	if err != nil {
		_ = root.Close()
		return nil, err //nolint:wrapcheck // caller reports raw error
	}
	return &rootFile{File: f, root: root}, nil
}

// localPath converts a name handed to Open into a local filesystem path. A
// plain relative or absolute path passes through unchanged; a "file:" URI is
// converted to a local path (percent-decoded, UNC and non-local host rejected);
// any other URI scheme is refused so the FS never reaches the network.
//
// The scheme is detected with the full RFC 3986 grammar INCLUDING a one-letter
// scheme (scheme = ALPHA *( ALPHA / DIGIT / "+" / "-" / "." )). This is stricter
// than [uripath.URIScheme], which returns "" for a one-letter scheme to keep a
// Windows drive letter ("C:\\x") out of URI space: relying on that here let a
// one-letter scheme like "x:dtd" slip past the non-"file" refusal and be opened
// as an in-root name. A native Windows drive path is still allowed, but ONLY on
// Windows and ONLY when it has the drive-letter shape ("C:\\x", "C:/x", "C:");
// on any other platform such a prefix is not a valid local path and is treated
// as a one-letter scheme (and refused unless it is "file").
func (c ConfinedDir) localPath(name string) (string, error) {
	if runtime.GOOS == goosWindows && uripath.HasWindowsDrivePrefix(name) {
		return name, nil
	}
	scheme := rfc3986Scheme(name)
	if scheme == "" {
		return name, nil
	}
	if scheme != "file" {
		return "", &fs.PathError{Op: opOpen, Path: name, Err: fs.ErrPermission}
	}
	return c.fileURILocalPath(name)
}

// fileURILocalPath converts a "file:" URI into a local filesystem path, matching
// the private confinedDirFS this shared type replaces. It differs from
// [FileURIToPath] in ONE respect: an opaque or empty-path form ("file:inside",
// "file:") is converted to an empty local path rather than rejected. Open then
// joins that empty path onto root and reads the root directory, which is the
// observable CLI behavior (an "is a directory" read error) that the promotion
// must preserve. FileURIToPath instead rejects the opaque form, because its
// other callers (XInclude, catalog) must not silently read a directory.
func (c ConfinedDir) fileURILocalPath(name string) (string, error) {
	u, err := url.Parse(name)
	if err != nil {
		return "", &fs.PathError{Op: opOpen, Path: name, Err: err}
	}
	if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
		return "", &fs.PathError{Op: opOpen, Path: name, Err: fs.ErrPermission}
	}
	// A "file:////server/share" URI (or a %5C-encoded backslash form) parses to
	// an empty host with a UNC-shaped path; on Windows that reaches a remote SMB
	// host, defeating the local-only policy. Reject every UNC form outright.
	if IsUNCFileURIPath(u.Path) {
		return "", &fs.PathError{Op: opOpen, Path: name, Err: fs.ErrPermission}
	}
	return fileURIPathFor(runtime.GOOS, u.Path), nil
}

// rfc3986Scheme returns the lowercased URI scheme of name — the leading token of
// an absolute-URI reference (scheme = ALPHA *( ALPHA / DIGIT / "+" / "-" / "." ))
// up to its delimiting ":" — or "" when name carries no such token. Unlike
// [uripath.URIScheme] it recognizes a ONE-letter scheme, so a caller that needs
// to refuse every non-"file" scheme (and has already excluded a Windows
// drive-letter path) cannot be bypassed by a one-letter scheme like "x:dtd".
func rfc3986Scheme(name string) string {
	if len(name) == 0 || !uripath.IsWindowsDriveLetter(name[0]) {
		return ""
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		switch {
		case uripath.IsWindowsDriveLetter(c) || (c >= '0' && c <= '9') ||
			c == '+' || c == '-' || c == '.':
			continue
		case c == ':':
			return strings.ToLower(name[:i])
		default:
			return ""
		}
	}
	return ""
}

// rootFile keeps the owning [os.Root] alive for the lifetime of the open file
// and closes it when the file is closed.
type rootFile struct {
	*os.File
	root *os.Root
}

func (f *rootFile) Close() error {
	err := f.File.Close()
	if cerr := f.root.Close(); err == nil {
		err = cerr
	}
	return err //nolint:wrapcheck // caller reports raw error
}

// withinDir reports whether path is the directory root itself or lies beneath
// it. Both arguments must be absolute and cleaned.
func withinDir(root, path string) bool {
	if path == root {
		return true
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(path, prefix)
}
