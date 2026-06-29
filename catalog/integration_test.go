package catalog_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/catalog"
	"github.com/stretchr/testify/require"
)

// nativePathToFileURI builds a "file:///" URI from a native absolute path on any
// OS. On POSIX the path begins with "/", so "file://" + "/tmp/x" already yields
// the correct "file:///tmp/x". On Windows the slashed path is "C:/..." (no
// leading slash), so a leading slash is prepended to avoid "file://C:/..."
// where "C:" is mis-read as the URI host.
func nativePathToFileURI(p string) string {
	slashed := filepath.ToSlash(p)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	return "file://" + slashed
}

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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).Catalog(cat).FS(helium.PermissiveFS())

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

func TestCatalogExternalSubsetFileURI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// DTD that declares a default attribute.
	dtdContent := `<!ATTLIST doc status CDATA "active">`
	dtdPath := filepath.Join(dir, "test.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(dtdContent), 0644))

	// Catalog mapping the system ID to a "file:" URI rather than a bare path.
	// The resolved value reaches the parser as "file:///...", which must be
	// converted to a local path before being opened (CAT-001).
	fileURI := nativePathToFileURI(dtdPath)
	catContent := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="http://example.com/test.dtd" uri="` + fileURI + `"/>
</catalog>`
	catPath := filepath.Join(dir, "catalog.xml")
	require.NoError(t, os.WriteFile(catPath, []byte(catContent), 0644))

	xmlContent := `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "http://example.com/test.dtd">
<doc/>`

	cat, err := catalog.Load(context.Background(), catPath)
	require.NoError(t, err)

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).Catalog(cat).FS(helium.PermissiveFS())

	doc, err := p.Parse(t.Context(), []byte(xmlContent))
	require.NoError(t, err)
	require.NotNil(t, doc)

	root := doc.FirstChild()
	for root != nil && root.Type() != helium.ElementNode {
		root = root.NextSibling()
	}
	require.NotNil(t, root, "should have root element")

	elem := root.(*helium.Element)
	attrs := elem.Attributes()
	require.Len(t, attrs, 1, "default attribute from catalog file: URI DTD should be applied")
	require.Equal(t, "status", attrs[0].LocalName())
	require.Equal(t, "active", attrs[0].Value())
}

func TestCatalogResolveEntityFileURI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// External parsed entity content, mapped via a "file:" URI in the catalog.
	entContent := `replaced-text`
	entPath := filepath.Join(dir, "ext.ent")
	require.NoError(t, os.WriteFile(entPath, []byte(entContent), 0644))

	fileURI := nativePathToFileURI(entPath)
	catContent := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="http://example.com/ext.ent" uri="` + fileURI + `"/>
</catalog>`
	catPath := filepath.Join(dir, "catalog.xml")
	require.NoError(t, os.WriteFile(catPath, []byte(catContent), 0644))

	xmlContent := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "http://example.com/ext.ent">
]>
<doc>&ext;</doc>`

	cat, err := catalog.Load(context.Background(), catPath)
	require.NoError(t, err)

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).SubstituteEntities(true).Catalog(cat).FS(helium.PermissiveFS())

	doc, err := p.Parse(t.Context(), []byte(xmlContent))
	require.NoError(t, err)
	require.NotNil(t, doc)

	root := doc.FirstChild()
	for root != nil && root.Type() != helium.ElementNode {
		root = root.NextSibling()
	}
	require.NotNil(t, root, "should have root element")
	require.Contains(t, string(root.Content()), "replaced-text",
		"external entity resolved via catalog file: URI should be substituted")
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).Catalog(cat).FS(helium.PermissiveFS())

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

	p := helium.NewParser().LoadExternalDTD(true)

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
