package xslt3

import (
	"errors"
	"io"

	"github.com/lestrrat-go/helium/internal/iolimit"
)

// MaxResourceBytes is the default maximum number of bytes read from a single
// external resource loaded through the configured URIResolver / HTTPClient /
// PackageResolver — fn:doc / document(), xsl:import / xsl:include, xsl:use-package
// package loads, xsl:import-schema, fn:transform stylesheet sources, and
// serialization parameter documents. It guards against a hostile or pathological
// resource (e.g. an effectively unbounded HTTP body or a "/dev/zero"-style
// stream) exhausting memory via an unbounded io.ReadAll. It mirrors the bounds
// already enforced by the parser (external entities) and xinclude.
const MaxResourceBytes = 10 << 20 // 10 MiB

// ErrResourceTooLarge is returned when an external resource exceeds
// [MaxResourceBytes]. The cap is enforced against the actual number of bytes
// read, not a Content-Length header, so a server that lies about its size
// cannot bypass it. When a runtime read (fn:doc / document(), fn:transform
// stylesheet / package sources) trips the cap, the resulting error wraps both
// this sentinel and [ErrDynamicError]: errors.Is matches either one.
var ErrResourceTooLarge = errors.New("xslt3: external resource exceeds maximum allowed size")

// resolveResourceLimit maps a configured cap to the value actually enforced:
// 0 selects the default [MaxResourceBytes] and a negative value disables the
// bound. Call sites pass the value configured on the Compiler / Invocation.
func resolveResourceLimit(configured int64) int64 {
	if configured == 0 {
		return MaxResourceBytes
	}
	return configured
}

// readResourceBounded reads from r through an [io.LimitReader] capped at limit,
// returning [ErrResourceTooLarge] when the source is larger. A limit of 0
// selects the default [MaxResourceBytes]; a negative limit disables the bound.
// It replaces unbounded io.ReadAll calls on resolver / HTTP bodies so a single
// external resource cannot exhaust process memory. The default-permitted set of
// resources is unchanged; only the read size is bounded.
func readResourceBounded(r io.Reader, limit int64) ([]byte, error) {
	limit = resolveResourceLimit(limit)
	if limit < 0 {
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, err //nolint:wrapcheck // callers wrap with the URI for context
		}
		return data, nil
	}

	data, exceeded, err := iolimit.ReadAll(r, limit)
	if exceeded {
		return nil, ErrResourceTooLarge
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // callers wrap with the URI for context
	}
	return data, nil
}
