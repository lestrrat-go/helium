// Package catalog implements OASIS XML Catalog resolution for public/system
// identifiers and URIs. It is used by the helium parser to transparently
// resolve external DTDs and entities via catalog files.
//
// The primary entry point for users is through the parser's SetCatalog method.
// This package is public so it can be used by other helium subsystems if needed.
package catalog

import (
	helium "github.com/lestrrat-go/helium"
	icatalog "github.com/lestrrat-go/helium/internal/catalog"
)

// Catalog holds parsed catalog entries and provides resolution.
// It wraps the internal catalog implementation and serves as the
// public API entry point.
type Catalog struct {
	cat *icatalog.Catalog
}

// Resolve resolves an external identifier (pubID and/or sysID) to a URI.
// Returns the resolved URI or "" if not found.
// A nil receiver is safe and always returns "".
func (c *Catalog) Resolve(pubID, sysID string) string {
	if c == nil {
		return ""
	}
	return c.cat.Resolve(pubID, sysID)
}

// ResolveURI resolves a URI reference.
// Returns the resolved URI or "" if not found.
// A nil receiver is safe and always returns "".
func (c *Catalog) ResolveURI(uri string) string {
	if c == nil {
		return ""
	}
	return c.cat.ResolveURI(uri)
}

// loaderConfig holds configuration for a Loader.
type loaderConfig struct {
	errorHandler helium.ErrorHandler
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
