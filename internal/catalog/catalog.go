// Package catalog implements OASIS XML Catalog resolution for public/system
// identifiers and URIs. This is an internal package — use the public
// github.com/lestrrat-go/helium/catalog package instead.
package catalog

const (
	// CatalogNamespace is the OASIS XML Catalog namespace.
	CatalogNamespace = "urn:oasis:names:tc:entity:xmlns:xml:catalog"

	// MaxDepth is the maximum recursion depth for catalog resolution.
	MaxDepth = 50

	// MaxDelegates is the maximum number of delegate catalogs per resolution.
	MaxDelegates = 50
)

// Prefer controls whether public or system identifiers take precedence
// when both are available in a catalog entry.
type Prefer int

const (
	PreferNone   Prefer = iota
	PreferPublic        // prefer="public" (default per OASIS spec)
	PreferSystem        // prefer="system"
)

// EntryType identifies the kind of catalog entry (libxml2: xmlCatalogEntryType).
type EntryType int

const (
	EntryPublic         EntryType = iota // (libxml2: XML_CATA_PUBLIC)
	EntrySystem                          // (libxml2: XML_CATA_SYSTEM)
	EntryRewriteSystem                   // (libxml2: XML_CATA_REWRITE_SYSTEM)
	EntryDelegatePublic                  // (libxml2: XML_CATA_DELEGATE_PUBLIC)
	EntryDelegateSystem                  // (libxml2: XML_CATA_DELEGATE_SYSTEM)
	EntryURI                             // (libxml2: XML_CATA_URI)
	EntryRewriteURI                      // (libxml2: XML_CATA_REWRITE_URI)
	EntryDelegateURI                     // (libxml2: XML_CATA_DELEGATE_URI)
	EntryNextCatalog                     // (libxml2: XML_CATA_NEXT_CATALOG)
)

// Entry represents a single catalog entry parsed from an XML catalog file.
type Entry struct {
	Type EntryType
	Name string // match key (publicId, systemId prefix, URI, name)
	URL  string // resolved URL (value resolved against xml:base)

	Prefer  Prefer   // inherited or overridden prefer attribute
	Catalog *Catalog // for nextCatalog/delegate entries (lazy-loaded)
}

// Loader loads a catalog from a file path. This interface decouples
// the resolution logic from the XML parser used to read catalog files.
type Loader interface {
	Load(filename string) (*Catalog, error)
}

// visitedKey identifies a (catalogURL, id1, id2) tuple for the resolve
// visited cache, matching libxml2's xmlCatalogResolveCacheVisited.
type visitedKey struct {
	url string
	id1 string
	id2 string
}

// Catalog holds parsed catalog entries and provides resolution.
type Catalog struct {
	Entries       []Entry
	Prefer        Prefer
	BaseURI       string
	Depth         int // recursion guard (shared across resolution chain)
	Loader  Loader
	visited map[visitedKey]struct{}
}

// Resolver is the interface that the helium parser uses for catalog
// resolution. This avoids the parser depending on the public catalog package.
type Resolver interface {
	Resolve(pubID, sysID string) string
	ResolveURI(uri string) string
}
