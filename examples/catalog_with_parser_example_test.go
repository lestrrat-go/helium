package examples_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/catalog"
)

func Example_catalog_with_parser() {
	// Create a catalog that maps an external system ID to a local file.
	// When the parser encounters a reference to "http://example.com/schema.dtd",
	// it will use "local-schema.dtd" from the catalog directory instead.
	const catalogSrc = `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="http://example.com/schema.dtd" uri="local-schema.dtd"/>
</catalog>`

	// Write the catalog to a temporary file.
	dir, err := os.MkdirTemp(".", ".tmp-catalog-*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %s\n", err)
		return
	}
	defer os.RemoveAll(dir)

	catalogPath := filepath.Join(dir, "catalog.xml")
	if err := os.WriteFile(catalogPath, []byte(catalogSrc), 0644); err != nil {
		fmt.Printf("failed to write catalog: %s\n", err)
		return
	}

	cat, err := catalog.Load(context.Background(), catalogPath)
	if err != nil {
		fmt.Printf("failed to load catalog: %s\n", err)
		return
	}

	// Catalog attaches a loaded catalog to the parser. When the parser needs
	// to resolve a system ID or public ID (e.g., from a DOCTYPE declaration),
	// it will consult this catalog first.
	p := helium.NewParser().
		Catalog(cat)

	// Parse a simple document. The catalog is available for any external
	// entity references the parser might encounter during parsing.
	doc, err := p.Parse(context.Background(), []byte(`<root/>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	s, err := helium.WriteString(doc)
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Print(s)
	// Output:
	// <?xml version="1.0"?>
	// <root/>
}
