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
// Callers handling untrusted input should substitute a stricter fs.FS
// on the relevant builder, for example one produced by [os.DirFS] or
// [os.OpenRoot].
type PermissiveRoot struct{}

// Open implements [fs.FS].
func (PermissiveRoot) Open(name string) (fs.File, error) {
	return os.Open(name) //nolint:gosec,wrapcheck // intentional passthrough; see type doc
}
