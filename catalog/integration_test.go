package catalog_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/catalog"
	"github.com/stretchr/testify/require"
)

func TestCatalogExternalSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// DTD that declares a default attribute.
	dtdContent := `<!ATTLIST doc status CDATA "active">`
	dtdPath := filepath.Join(dir, "test.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(dtdContent), 0644))

	// Catalog mapping the system ID to our local DTD.
	catContent := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="http://example.com/test.dtd" uri="` + dtdPath + `"/>
</catalog>`
	catPath := filepath.Join(dir, "catalog.xml")
	require.NoError(t, os.WriteFile(catPath, []byte(catContent), 0644))

	// XML referencing the system ID — without catalog, DTD won't be found.
	xmlContent := `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "http://example.com/test.dtd">
<doc/>`

	cat, err := catalog.Load(context.Background(), catPath)
	require.NoError(t, err)

	p := helium.NewParser()
	p.SetOption(helium.ParseDTDLoad)
	p.SetOption(helium.ParseDTDAttr)
	p.SetCatalog(cat)

	doc, err := p.Parse(t.Context(), []byte(xmlContent))
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Verify the external DTD was loaded: the default attribute should be applied.
	root := doc.FirstChild()
	for root != nil && root.Type() != helium.ElementNode {
		root = root.NextSibling()
	}
	require.NotNil(t, root, "should have root element")

	elem := root.(*helium.Element)
	attrs := elem.Attributes()
	require.Len(t, attrs, 1, "default attribute from DTD should be applied")
	require.Equal(t, "status", attrs[0].LocalName())
	require.Equal(t, "active", attrs[0].Value())
}

func TestCatalogPublicIDResolution(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	dtdContent := `<!ATTLIST item category CDATA "general">`
	dtdPath := filepath.Join(dir, "item.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(dtdContent), 0644))

	catContent := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <public publicId="-//TEST//DTD Item 1.0//EN" uri="` + dtdPath + `"/>
</catalog>`
	catPath := filepath.Join(dir, "catalog.xml")
	require.NoError(t, os.WriteFile(catPath, []byte(catContent), 0644))

	xmlContent := `<?xml version="1.0"?>
<!DOCTYPE item PUBLIC "-//TEST//DTD Item 1.0//EN" "http://bogus.example.com/item.dtd">
<item/>`

	cat, err := catalog.Load(context.Background(), catPath)
	require.NoError(t, err)

	p := helium.NewParser()
	p.SetOption(helium.ParseDTDLoad)
	p.SetOption(helium.ParseDTDAttr)
	p.SetCatalog(cat)

	doc, err := p.Parse(t.Context(), []byte(xmlContent))
	require.NoError(t, err)
	require.NotNil(t, doc)

	root := doc.FirstChild()
	for root != nil && root.Type() != helium.ElementNode {
		root = root.NextSibling()
	}
	require.NotNil(t, root)

	elem := root.(*helium.Element)
	attrs := elem.Attributes()
	require.Len(t, attrs, 1)
	require.Equal(t, "category", attrs[0].LocalName())
	require.Equal(t, "general", attrs[0].Value())
}

func TestCatalogNoCatalog(t *testing.T) {
	t.Parallel()

	xmlContent := `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "http://example.com/nonexistent.dtd">
<doc/>`

	p := helium.NewParser()
	p.SetOption(helium.ParseDTDLoad)

	doc, err := p.Parse(t.Context(), []byte(xmlContent))
	require.NoError(t, err)
	require.NotNil(t, doc)

	root := doc.FirstChild()
	for root != nil && root.Type() != helium.ElementNode {
		root = root.NextSibling()
	}
	require.NotNil(t, root)
	elem := root.(*helium.Element)
	require.Empty(t, elem.Attributes())
}
