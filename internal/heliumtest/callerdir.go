// Package heliumtest provides test helpers shared across helium packages.
package heliumtest

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
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

var (
	repoRoot     string
	repoRootOnce sync.Once
)

func findRepoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("cannot find repo root (go.mod)")
		}
		dir = parent
	}
}

// RepoRoot returns the absolute path to the repository root (the directory
// containing go.mod). The result is cached after the first call.
func RepoRoot() string {
	repoRootOnce.Do(func() { repoRoot = findRepoRoot() })
	return repoRoot
}

// TestDir returns an absolute path under the repository root.
// The path elements are joined after the root, e.g.
// TestDir("testdata", "libxml2-compat") → "<repo>/testdata/libxml2-compat".
func TestDir(path ...string) string {
	return filepath.Join(append([]string{RepoRoot()}, path...)...)
}
