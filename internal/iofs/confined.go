package iofs

import (
	"io/fs"
	"os"
	"path/filepath"
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
}

// NewConfinedDir returns a ConfinedDir rooted at dir. dir is resolved to an
// absolute, cleaned path; a relative dir is taken relative to the process
// working directory at call time.
func NewConfinedDir(dir string) ConfinedDir {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = filepath.Clean(dir)
	}
	return ConfinedDir{root: abs}
}

// Open implements [fs.FS].
func (c ConfinedDir) Open(name string) (fs.File, error) {
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
// converted via [FileURIToPath] (percent-decoded, UNC and non-local host
// rejected); any other URI scheme is refused so the FS never reaches the
// network.
func (c ConfinedDir) localPath(name string) (string, error) {
	scheme := uripath.URIScheme(name)
	if scheme == "" {
		return name, nil
	}
	if scheme != "file" {
		return "", &fs.PathError{Op: opOpen, Path: name, Err: fs.ErrPermission}
	}
	p, err := FileURIToPath(name)
	if err != nil {
		return "", &fs.PathError{Op: opOpen, Path: name, Err: err}
	}
	return p, nil
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
