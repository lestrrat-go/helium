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

// TestStripSpaceUndeclaredPrefix verifies that a prefix used in a strip-space
// elements NameTest that is NOT in scope at the declaration raises XTSE0280,
// rather than being silently accepted via a compiler-wide binding leaked from an
// imported module.
func TestStripSpaceUndeclaredPrefix(t *testing.T) {
	t.Parallel()

	// Imported module binds prefix "p" -> urn:A.
	imported := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:p="urn:A" version="3.0">
  <xsl:strip-space elements="p:item"/>
</xsl:stylesheet>`

	// Importing module does NOT bind "p" anywhere in scope at its own
	// strip-space declaration. Using "p:item" here must raise XTSE0280, not be
	// accepted because the imported module happened to bind "p".
	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:import href="mem:/imported.xsl"/>
  <xsl:strip-space elements="p:item"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	resolver := &memResolver{files: map[string]string{
		"mem:/imported.xsl": imported,
	}}

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	_, err = xslt3.NewCompiler().
		BaseURI("mem:/main.xsl").
		URIResolver(resolver).
		Compile(t.Context(), doc)
	require.Error(t, err, "undeclared prefix in strip-space elements must raise XTSE0280")
	require.Contains(t, err.Error(), "XTSE0280")
}

// TestStripSpaceWildcardKindsNoConflict verifies that strip/preserve NameTests
// of DIFFERENT kinds at the same import precedence do not raise a false
// XTSE0270: their match priorities differ, so the conflict is resolved at
// runtime by priority rather than being a genuine same-priority conflict.
func TestStripSpaceWildcardKindsNoConflict(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		strip    string
		preserve string
	}{
		{
			// "*:item" (local-name wildcard, priority -0.25) vs "item"
			// (exact, priority 0): distinct kinds, no conflict.
			name:     "local-wildcard vs exact",
			strip:    "*:item",
			preserve: "item",
		},
		{
			// "Q{}*" (namespace wildcard, empty ns, priority -0.25) vs "*"
			// (universal, priority -0.5): distinct kinds, no conflict.
			name:     "namespace-wildcard vs universal",
			strip:    "Q{}*",
			preserve: "*",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="` + tc.strip + `"/>
  <xsl:preserve-space elements="` + tc.preserve + `"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

			doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
			require.NoError(t, err)

			ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
			require.NoError(t, err,
				"strip=%q preserve=%q are different NameTest kinds and must not raise XTSE0270", tc.strip, tc.preserve)
			require.NotNil(t, ss)
		})
	}
}

// TestStripSpaceSameKindWildcardConflict verifies that a genuine same-kind,
// same-name wildcard conflict at the same precedence still raises XTSE0270.
func TestStripSpaceSameKindWildcardConflict(t *testing.T) {
	t.Parallel()

	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*:item"/>
  <xsl:preserve-space elements="*:item"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
	require.Error(t, err, "same-kind same-name wildcard conflict must raise XTSE0270")
	require.Contains(t, err.Error(), "XTSE0270")
}

// TestStripSpaceWildcardOverlapConflict verifies that two wildcard NameTests of
// DIFFERENT shapes but the SAME match priority whose match SETS genuinely
// overlap raise XTSE0270 at the same import precedence — even though their
// canonical keys differ. "*:item" (local-name wildcard) and "Q{urn:A}*"
// (namespace wildcard) both match Q{urn:A}item at priority -0.25, so declaring
// one strip and the other preserve is a genuine conflict. Both orderings are
// checked because conflict detection must be symmetric.
func TestStripSpaceWildcardOverlapConflict(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		strip    string
		preserve string
	}{
		{
			name:     "local-wildcard strip vs namespace-wildcard preserve",
			strip:    "*:item",
			preserve: "Q{urn:A}*",
		},
		{
			name:     "namespace-wildcard strip vs local-wildcard preserve",
			strip:    "Q{urn:A}*",
			preserve: "*:item",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="` + tc.strip + `"/>
  <xsl:preserve-space elements="` + tc.preserve + `"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

			doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
			require.NoError(t, err)

			_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
			require.Error(t, err,
				"overlapping same-priority strip=%q preserve=%q must raise XTSE0270", tc.strip, tc.preserve)
			require.Contains(t, err.Error(), "XTSE0270")
		})
	}
}

// TestStripSpaceNamespaceWildcardPriority verifies that a namespace wildcard
// (Q{uri}*) outranks the universal wildcard (*) at equal import precedence, so
// strip-space="Q{urn:A}*" wins over preserve-space="*" for an element in urn:A.
func TestStripSpaceNamespaceWildcardPriority(t *testing.T) {
	t.Parallel()

	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="Q{urn:A}*"/>
  <xsl:preserve-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, ss)

	// <a:item> in urn:A has whitespace-only content. The namespace wildcard
	// strip-space rule must outrank the universal preserve-space wildcard, so the
	// whitespace is stripped.
	source, err := helium.NewParser().Parse(t.Context(), []byte(
		`<doc xmlns:a="urn:A"><a:item>   </a:item></doc>`))
	require.NoError(t, err)

	out, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)

	require.NotContains(t, out, "   </a:item>",
		"Q{urn:A}* strip-space must outrank * preserve-space; got %q", out)
}
