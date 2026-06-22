package helium_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// writeDTD writes a DTD file into a fresh temp dir and returns its path.
func writeDTD(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(p, []byte(body), 0600))
	return p
}

// TestExternalDTDConditionalSections exercises INCLUDE/IGNORE conditional
// sections, which only appear in the external subset.
func TestExternalDTDConditionalSections(t *testing.T) {
	t.Parallel()

	const dtd = `<!ELEMENT root (#PCDATA)>
<![INCLUDE[
<!ENTITY included "in">
]]>
<![IGNORE[
<!ENTITY ignored "out">
]]>`
	path := writeDTD(t, dtd)

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + path + `">
<root/>`

	doc, err := helium.NewParser().LoadExternalDTD(true).Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, found := doc.GetEntity("included")
	require.True(t, found, "entity inside INCLUDE section must be declared")

	_, found = doc.GetEntity("ignored")
	require.False(t, found, "entity inside IGNORE section must be skipped")
}

// TestExternalDTDNotationsAndEntities exercises notation and external entity
// declarations resolved from the external subset.
func TestExternalDTDNotationsAndEntities(t *testing.T) {
	t.Parallel()

	const dtd = `<!ELEMENT root (#PCDATA)>
<!NOTATION gif SYSTEM "viewer.exe">
<!NOTATION png PUBLIC "-//N//EN" "png.exe">
<!ENTITY img SYSTEM "img.gif" NDATA gif>
<!ENTITY ext SYSTEM "data.xml">
<!ENTITY pub PUBLIC "-//P//EN" "pub.xml">`
	path := writeDTD(t, dtd)

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + path + `">
<root/>`

	doc, err := helium.NewParser().LoadExternalDTD(true).Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	require.NotNil(t, doc.ExtSubset(), "external subset must be present")
	// The external general entities are resolvable from the document.
	_, ok := doc.GetEntity("ext")
	require.True(t, ok, "external SYSTEM entity declared in ext subset")
	_, ok = doc.GetEntity("pub")
	require.True(t, ok, "external PUBLIC entity declared in ext subset")
}

// TestExternalDTDPublicIdentifier exercises a DOCTYPE that declares a PUBLIC
// external identifier (with both public and system IDs).
func TestExternalDTDPublicIdentifier(t *testing.T) {
	t.Parallel()

	const dtd = `<!ELEMENT root (#PCDATA)>
<!ENTITY who "world">`
	path := writeDTD(t, dtd)

	xml := `<?xml version="1.0"?>
<!DOCTYPE root PUBLIC "-//Example//DTD root//EN" "` + path + `">
<root/>`

	doc, err := helium.NewParser().LoadExternalDTD(true).Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	_, found := doc.GetEntity("who")
	require.True(t, found)
}

// TestExternalDTDMissingFile exercises the not-found branch of external DTD
// resolution: a SYSTEM id pointing at a non-existent file must not crash.
func TestExternalDTDMissingFile(t *testing.T) {
	t.Parallel()

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "/nonexistent/path/to.dtd">
<root>content</root>`

	// The document body is still well-formed; the external DTD simply cannot be
	// loaded. Parsing should not panic.
	doc, _ := helium.NewParser().LoadExternalDTD(true).Parse(t.Context(), []byte(xml))
	if doc != nil {
		require.NotNil(t, doc.DocumentElement())
	}
}

// TestInternalSubsetParameterEntityInclusion exercises a parameter entity used
// inside the internal subset to pull in further declarations.
func TestInternalSubsetParameterEntityInclusion(t *testing.T) {
	t.Parallel()

	const xml = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ENTITY % decls "<!ELEMENT root (#PCDATA)><!ENTITY inner 'inner-value'>">
%decls;
]>
<root/>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	_, found := doc.GetEntity("inner")
	require.True(t, found, "entity declared via internal-subset PE inclusion must be present")
}
