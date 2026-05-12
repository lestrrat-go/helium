// Package iofs provides the default [fs.FS] used by helium for opening
// external resources (DTDs, entities, schemas, XInclude targets).
package iofs

import (
	"io/fs"
	"os"
)

// Root is an [fs.FS] backed by direct calls to [os.Open]. Unlike most
// fs.FS implementations it does not enforce [fs.ValidPath]; names are
// passed straight through to the OS. This preserves helium's historical
// behavior of opening any path (absolute, traversal-containing, etc.)
// supplied to the parser, schema compilers, and XInclude processor.
//
// Callers handling untrusted input should substitute a stricter fs.FS
// on the relevant builder (e.g. one produced by [os.DirFS]).
type Root struct{}

// Open implements [fs.FS].
func (Root) Open(name string) (fs.File, error) {
	return os.Open(name) //nolint:gosec,wrapcheck // intentional passthrough; see type doc
}
