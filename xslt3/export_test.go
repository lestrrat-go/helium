package xslt3

import (
	"io"

	"github.com/lestrrat-go/helium"
)

// ReadResourceBoundedForTest exposes the internal bounded-read helper to the
// external test package so the size cap can be exercised directly. A limit of 0
// selects the default [MaxResourceBytes]; a negative limit disables the bound.
func ReadResourceBoundedForTest(r io.Reader, limit int64) ([]byte, error) {
	return readResourceBounded(r, limit)
}

// CopyAndStripForTest exposes the single-pass strip-space copy helper to the
// external test package with the default (no) strip/preserve rules and no node
// map, so the produced copy's independence from the source can be asserted.
func CopyAndStripForTest(src *helium.Document) (*helium.Document, error) {
	dst, _, err := copyAndStrip(src, nil, nil, false, nil)
	return dst, err
}
