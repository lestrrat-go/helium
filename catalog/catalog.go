package catalog

import (
	"context"
	"errors"

	helium "github.com/lestrrat-go/helium"
	icatalog "github.com/lestrrat-go/helium/internal/catalog"
)

// MaxCatalogSize is the default maximum number of bytes read from a catalog
// file. Catalog files are loaded from XML_CATALOG_FILES or the API, so an
// unbounded read of a hostile or pathological source (e.g. /dev/zero) could
// exhaust memory before parsing applies any limits. The file is read through a
// strict byte cap and rejected with [ErrCatalogTooLarge] when it is exceeded.
const MaxCatalogSize = 10 << 20 // 10 MiB

// ErrCatalogTooLarge is returned when a catalog file exceeds the configured
// byte cap (set via [Loader.MaxBytes]), or [MaxCatalogSize] when no cap is
// configured. The cap is enforced against the actual number of bytes read.
var ErrCatalogTooLarge = errors.New("catalog file exceeds maximum allowed size")

// Catalog holds parsed catalog entries and provides resolution.
// It wraps the internal catalog implementation and serves as the
// public API entry point.
type Catalog struct {
	cat *icatalog.Catalog
}

// Resolve resolves an external identifier (pubID and/or sysID) to a URI.
// Returns the resolved URI or "" if not found.
// A nil receiver is safe and always returns "".
func (c *Catalog) Resolve(ctx context.Context, pubID, sysID string) string {
	if c == nil {
		return ""
	}
	return c.cat.Resolve(ctx, pubID, sysID)
}

// ResolveResult is like Resolve but also reports whether resolution ended in a
// catalog break. A break is the OASIS/libxml2 "cut" signal: delegate or
// nextCatalog entries were consulted and all of them failed, so the search must
// STOP rather than continue to later catalogs in a chain.
//
// When broke is true the caller must not consult any further catalog, even
// though uri is "". When broke is false a "" uri means "no match here, keep
// searching". A non-empty uri is a successful match. A nil receiver is safe and
// returns ("", false).
func (c *Catalog) ResolveResult(ctx context.Context, pubID, sysID string) (uri string, broke bool) {
	if c == nil {
		return "", false
	}
	return c.cat.ResolveResult(ctx, pubID, sysID)
}

// ResolveURI resolves a URI reference.
// Returns the resolved URI or "" if not found.
// A nil receiver is safe and always returns "".
func (c *Catalog) ResolveURI(ctx context.Context, uri string) string {
	if c == nil {
		return ""
	}
	return c.cat.ResolveURI(ctx, uri)
}

// ResolveURIResult is like ResolveURI but also reports whether resolution ended
// in a catalog break (see [Catalog.ResolveResult]). When broke is true a chain
// caller must stop searching rather than fall through to later catalogs. A nil
// receiver is safe and returns ("", false).
func (c *Catalog) ResolveURIResult(ctx context.Context, uri string) (resolved string, broke bool) {
	if c == nil {
		return "", false
	}
	return c.cat.ResolveURIResult(ctx, uri)
}

// loaderConfig holds configuration for a Loader.
type loaderConfig struct {
	errorHandler helium.ErrorHandler
	maxBytes     int
}

// Loader loads OASIS XML Catalog files. It is a value-style wrapper:
// fluent methods return updated copies and the original is never mutated.
// The terminal method is Load.
type Loader struct {
	cfg *loaderConfig
}

// NewLoader creates a new Loader with default settings.
func NewLoader() Loader {
	return Loader{cfg: &loaderConfig{}}
}

func (l Loader) clone() Loader {
	if l.cfg == nil {
		return Loader{cfg: &loaderConfig{}}
	}
	cp := *l.cfg
	return Loader{cfg: &cp}
}

// ErrorHandler returns a copy of the Loader that delivers warnings
// to the given error handler during catalog parsing.
func (l Loader) ErrorHandler(h helium.ErrorHandler) Loader {
	l = l.clone()
	l.cfg.errorHandler = h
	return l
}

// MaxBytes returns a copy of the Loader that caps the number of bytes read
// from a catalog file at n. The cap guards against hostile or pathological
// sources (e.g. /dev/zero) that could otherwise exhaust memory. When a catalog
// file exceeds the cap, Load fails with [ErrCatalogTooLarge]. A value less than
// or equal to zero (the default) means [MaxCatalogSize] (10 MiB) is used.
func (l Loader) MaxBytes(n int) Loader {
	l = l.clone()
	l.cfg.maxBytes = n
	return l
}
