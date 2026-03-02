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
func (c *Catalog) Resolve(pubID, sysID string) string {
	if c == nil {
		return ""
	}
	return c.cat.Resolve(pubID, sysID)
}

// ResolveURI resolves a URI reference.
// Returns the resolved URI or "" if not found.
func (c *Catalog) ResolveURI(uri string) string {
	if c == nil {
		return ""
	}
	return c.cat.ResolveURI(uri)
}

// LoadOption configures the behavior of Load.
type LoadOption func(*loadConfig)

type loadConfig struct {
	errorHandler helium.ErrorHandler
}

// WithErrorHandler sets the error handler that receives warnings
// during catalog parsing (e.g. missing required attributes).
func WithErrorHandler(h helium.ErrorHandler) LoadOption {
	return func(c *loadConfig) { c.errorHandler = h }
}
