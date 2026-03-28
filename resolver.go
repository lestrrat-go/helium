package helium

import "context"

// CatalogResolver resolves external entity identifiers and URIs via an
// XML catalog. Implementations are passed to [Parser.Catalog] to enable
// catalog-based resolution during parsing.
//
// The [github.com/lestrrat-go/helium/catalog] package provides a standard
// implementation backed by OASIS XML Catalog files. Custom implementations
// may also be used.
type CatalogResolver interface {
	// Resolve resolves a public/system identifier pair to a URI.
	// Either pubID or sysID may be empty. Returns "" if no match is found.
	Resolve(ctx context.Context, pubID, sysID string) string

	// ResolveURI resolves a URI reference. Returns "" if no match is found.
	ResolveURI(ctx context.Context, uri string) string
}
