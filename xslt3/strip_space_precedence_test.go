package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// memResolver (defined in fn_transform_test.go) serves stylesheet modules from
// an in-memory URI->content map.

// TestStripSpaceImportPrecedence verifies that a conflicting strip-space /
// preserve-space NameTest across an import boundary is resolved by import
// precedence (the importing module wins) rather than raising a false XTSE0270,
// and that the higher-precedence rule governs whitespace stripping at runtime.
func TestStripSpaceImportPrecedence(t *testing.T) {
	t.Parallel()

	// Imported module strips whitespace in <item>.
	imported := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="item"/>
</xsl:stylesheet>`

	// Importing (higher-precedence) module preserves whitespace in <item>.
	// Without import-precedence handling the overlapping NameTest "item" in both
	// strip-space and preserve-space would falsely raise XTSE0270.
	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:import href="mem:/imported.xsl"/>
  <xsl:preserve-space elements="item"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
</xsl:stylesheet>`

	resolver := &memResolver{files: map[string]string{
		"mem:/imported.xsl": imported,
	}}

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		BaseURI("mem:/main.xsl").
		URIResolver(resolver).
		Compile(t.Context(), doc)
	require.NoError(t, err, "overlapping strip/preserve at different import precedence must not raise XTSE0270")
	require.NotNil(t, ss)

	source, err := helium.NewParser().Parse(t.Context(),
		[]byte("<doc><item>   </item></doc>"))
	require.NoError(t, err)

	out, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)

	// The higher-precedence preserve-space rule wins: the whitespace-only text
	// node inside <item> survives. A self-closing <item/> would mean the
	// lower-precedence strip-space rule incorrectly won.
	require.Contains(t, out, "<item>   </item>",
		"higher import-precedence preserve-space must override imported strip-space; got %q", out)
}

// TestStripSpaceSamePrecedenceConflict verifies that a genuine same-precedence
// conflict (the same NameTest declared both strip and preserve in one module)
// still raises XTSE0270.
func TestStripSpaceSamePrecedenceConflict(t *testing.T) {
	t.Parallel()

	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="item"/>
  <xsl:preserve-space elements="item"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
	require.Error(t, err, "same-precedence strip/preserve conflict must raise XTSE0270")
	require.Contains(t, err.Error(), "XTSE0270")
}

// TestStripSpacePrefixNamespaceContext verifies that a prefixed element name in
// a strip-space rule is resolved using the namespace context in scope at the
// declaration, not by local name alone. The same prefix "p" is bound to a
// different URI in the imported module than in the importing module, so a
// declaration-local resolution is required to pick the correct namespace.
func TestStripSpacePrefixNamespaceContext(t *testing.T) {
	t.Parallel()

	// Imported module binds p -> urn:A and strips p:item (i.e. urn:A item).
	imported := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:p="urn:A" version="3.0">
  <xsl:strip-space elements="p:item"/>
</xsl:stylesheet>`

	// Importing module rebinds p -> urn:B. Its own rules do not mention p:item,
	// so the imported strip rule must still target urn:A item (resolved at the
	// import's declaration), NOT urn:B item.
	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:p="urn:B" version="3.0">
  <xsl:import href="mem:/imported.xsl"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
</xsl:stylesheet>`

	resolver := &memResolver{files: map[string]string{
		"mem:/imported.xsl": imported,
	}}

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		BaseURI("mem:/main.xsl").
		URIResolver(resolver).
		Compile(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, ss)

	// <a:item> in urn:A has whitespace-only content (should be stripped).
	// <b:item> in urn:B has whitespace-only content (must NOT be stripped,
	// because the strip rule resolves to urn:A, not urn:B).
	source, err := helium.NewParser().Parse(t.Context(), []byte(
		`<doc xmlns:a="urn:A" xmlns:b="urn:B"><a:item>   </a:item><b:item>   </b:item></doc>`))
	require.NoError(t, err)

	out, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)

	// urn:A item stripped (empty element, no whitespace text node); urn:B item
	// retains its whitespace.
	require.NotContains(t, out, "   </a:item>",
		"urn:A item should be stripped; got %q", out)
	require.Contains(t, out, "   </b:item>",
		"urn:B item must not be stripped (prefix p resolves to urn:A at the import declaration); got %q", out)
}
