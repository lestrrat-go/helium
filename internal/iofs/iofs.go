// Package iofs provides the default [fs.FS] used by helium for opening
// external resources (DTDs, entities, schemas, XInclude targets).
package iofs

import (
	"io/fs"
	"os"
)

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
