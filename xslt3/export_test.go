package xslt3

import "io"

// ReadResourceBoundedForTest exposes the internal bounded-read helper to the
// external test package so the size cap can be exercised directly. A limit of 0
// selects the default [MaxResourceBytes]; a negative limit disables the bound.
func ReadResourceBoundedForTest(r io.Reader, limit int64) ([]byte, error) {
	return readResourceBounded(r, limit)
}
