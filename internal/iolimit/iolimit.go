// Package iolimit provides a bounded read-all helper shared by the parser,
// xinclude, and xslt3 resource loaders. It is a stdlib-only leaf package and
// never imports back into helium.
package iolimit

import (
	"io"
	"math"
)

// ReadAll reads from r through a LimitReader sized limit+1 (overflow-guarded for
// limit==math.MaxInt64), so a source exactly at the cap is accepted while
// anything larger is detected. It returns the raw bytes, exceeded=true when more
// than limit bytes were read, and any read error. It formats no error and picks
// no sentinel: callers map exceeded to their own error and choose the
// error-vs-size ordering.
func ReadAll(r io.Reader, limit int64) (data []byte, exceeded bool, err error) {
	readLimit := limit
	if readLimit < math.MaxInt64 {
		readLimit++
	}
	data, err = io.ReadAll(io.LimitReader(r, readLimit))
	return data, int64(len(data)) > limit, err
}
