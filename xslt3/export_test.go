package xslt3

import "io"

// ReadResourceBoundedForTest exposes the internal bounded-read helper to the
// external test package so the size cap can be exercised directly.
func ReadResourceBoundedForTest(r io.Reader) ([]byte, error) {
	return readResourceBounded(r)
}
