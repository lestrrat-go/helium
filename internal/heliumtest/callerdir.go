// Package heliumtest provides test helpers shared across helium packages.
package heliumtest

import (
	"path/filepath"
	"runtime"
)

// CallerDir returns the directory containing the source file of the caller.
// skip is the number of additional stack frames to skip (0 = direct caller).
func CallerDir(skip int) string {
	_, file, _, ok := runtime.Caller(skip + 1)
	if !ok {
		panic("runtime.Caller failed")
	}
	return filepath.Dir(file)
}
