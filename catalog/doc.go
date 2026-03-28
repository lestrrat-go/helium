// Package catalog implements OASIS XML Catalog resolution for public/system
// identifiers and URIs.
//
// Use [NewLoader] to load a catalog file, then pass the resulting [*Catalog]
// to the parser via [helium.Parser.Catalog]:
//
//	cat, err := catalog.NewLoader().Load(ctx, "catalog.xml")
//	doc, err := helium.NewParser().Catalog(cat).Parse(ctx, xmlBytes)
//
// The [*Catalog] type satisfies the [helium.CatalogResolver] interface,
// which the parser uses for resolution. Custom resolver implementations
// can also be passed to [helium.Parser.Catalog].
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with catalog_). Because examples are in
// a separate test module they do not appear in the generated documentation.
package catalog
