package examples_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium/catalog"
)

func Example_catalog_load() {
	// An OASIS XML Catalog maps external identifiers (system IDs, public IDs)
	// to local URIs. This is commonly used to redirect network-based DTD/schema
	// references to local copies, enabling offline validation.
	const catalogSrc = `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="http://example.com/schema.dtd" uri="local-schema.dtd"/>
  <public publicId="-//Example//DTD Test//EN" uri="local-schema.dtd"/>
</catalog>`

	// Write the catalog to a temporary file. In a real application, the
	// catalog file would typically live at a well-known location on disk.
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

	// catalog.Load parses the catalog XML and returns a Catalog object
	// that can resolve identifiers.
	cat, err := catalog.Load(context.Background(), catalogPath)
	if err != nil {
		fmt.Printf("failed to load catalog: %s\n", err)
		return
	}

	// Resolve looks up an identifier pair (publicID, systemID) and returns
	// the local URI from the catalog. The resolved path is absolute.
	// Here we check the suffix since the absolute path varies by environment.

	// Resolve a system identifier (by systemId).
	resolved := cat.Resolve(context.Background(), "", "http://example.com/schema.dtd")
	fmt.Printf("system resolved: %t\n", strings.HasSuffix(resolved, "local-schema.dtd"))

	// Resolve a public identifier (by publicId).
	resolved = cat.Resolve(context.Background(), "-//Example//DTD Test//EN", "")
	fmt.Printf("public resolved: %t\n", strings.HasSuffix(resolved, "local-schema.dtd"))

	// When neither publicId nor systemId matches any catalog entry,
	// Resolve returns an empty string.
	resolved = cat.Resolve(context.Background(), "", "http://unknown.com/other.dtd")
	fmt.Printf("unknown: %q\n", resolved)
	// Output:
	// system resolved: true
	// public resolved: true
	// unknown: ""
}
