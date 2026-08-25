package xslt3_test

import (
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// memResolver (defined in fn_transform_test.go) serves stylesheet modules from
// an in-memory URI->content map.

// importedModuleURI is the URI under which strip-space precedence tests serve
// their imported stylesheet module.
const importedModuleURI = "mem:/imported.xsl"

func TestStripSpace(t *testing.T) {
	// TestStripSpaceImportPrecedence verifies that a conflicting strip-space /
	// preserve-space NameTest across an import boundary is resolved by import
	// precedence (the importing module wins), raising no false XTSE0270,
	// and that the higher-precedence rule governs whitespace stripping at runtime.
	t.Run("import precedence", func(t *testing.T) {
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
  <xsl:import href="` + importedModuleURI + `"/>
  <xsl:preserve-space elements="item"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
</xsl:stylesheet>`

		resolver := &memResolver{files: map[string]string{
			importedModuleURI: imported,
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
	})

	// TestStripSpaceSamePrecedenceConflict verifies that a genuine same-precedence
	// conflict (the same NameTest declared both strip and preserve in one module)
	// still raises XTSE0270.
	t.Run("same precedence conflict", func(t *testing.T) {
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
	})

	// TestStripSpaceLowerPrecedenceConflictNotMasked verifies that a genuine
	// same-precedence strip/preserve conflict in a LOWER-precedence imported module
	// is still reported, even when the importing (higher-precedence) module adds an
	// UNRELATED higher-precedence strip-space rule. The higher-precedence rule for
	// name "b" does not overlap the "a" conflict and therefore must not suppress it.
	// Previously the check filtered by each kind's globally-highest precedence, so
	// the higher "strip b" raised the strip threshold and the real "a" vs "a"
	// conflict at the lower precedence was silently dropped.
	t.Run("lower precedence conflict not masked", func(t *testing.T) {
		t.Parallel()

		// Imported module: a genuine same-precedence conflict over "a".
		imported := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="a"/>
  <xsl:preserve-space elements="a"/>
</xsl:stylesheet>`

		// Importing module adds a higher-precedence, unrelated strip-space for "b".
		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:import href="` + importedModuleURI + `"/>
  <xsl:strip-space elements="b"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

		resolver := &memResolver{files: map[string]string{
			importedModuleURI: imported,
		}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)

		_, err = xslt3.NewCompiler().
			BaseURI("mem:/main.xsl").
			URIResolver(resolver).
			Compile(t.Context(), doc)
		require.Error(t, err,
			"same-precedence strip/preserve conflict over \"a\" must still raise XTSE0270 despite an unrelated higher-precedence strip-space for \"b\"")
		require.Contains(t, err.Error(), "XTSE0270")
	})

	// TestStripSpaceHigherPrecedenceCoversConflict verifies that a genuine
	// same-precedence strip/preserve overlap in a LOWER-precedence imported module is
	// NOT reported as XTSE0270 when the importing (higher-precedence) module declares
	// a rule that COVERS the conflicting name. The imported module has both
	// strip-space="a" and preserve-space="a" (a same-precedence conflict over "a"),
	// but the importing module's higher-precedence strip-space="*" matches "a", so
	// import precedence resolves it per-name and no conflict survives.
	t.Run("higher precedence covers conflict", func(t *testing.T) {
		t.Parallel()

		// Imported module: a same-precedence overlap over "a".
		imported := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="a"/>
  <xsl:preserve-space elements="a"/>
</xsl:stylesheet>`

		// Importing module: higher-precedence universal strip-space covers "a".
		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:import href="` + importedModuleURI + `"/>
  <xsl:strip-space elements="*"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

		resolver := &memResolver{files: map[string]string{
			importedModuleURI: imported,
		}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)

		ss, err := xslt3.NewCompiler().
			BaseURI("mem:/main.xsl").
			URIResolver(resolver).
			Compile(t.Context(), doc)
		require.NoError(t, err,
			"higher-precedence strip-space=\"*\" covers the imported \"a\" overlap; XTSE0270 must NOT fire")
		require.NotNil(t, ss)
	})

	// TestStripSpaceHigherPrecedencePartialCoverConflict verifies that a
	// higher-precedence rule which covers only PART of a same-precedence overlap does
	// NOT suppress XTSE0270 for the uncovered part. The imported module has a genuine
	// "a" vs "a" overlap; the importing module's higher-precedence strip-space="b"
	// matches "b" only, leaving "a" uncovered, so the conflict still fires.
	t.Run("higher precedence partial cover conflict", func(t *testing.T) {
		t.Parallel()

		imported := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="a"/>
  <xsl:preserve-space elements="a"/>
</xsl:stylesheet>`

		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:import href="` + importedModuleURI + `"/>
  <xsl:strip-space elements="b"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

		resolver := &memResolver{files: map[string]string{
			importedModuleURI: imported,
		}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)

		_, err = xslt3.NewCompiler().
			BaseURI("mem:/main.xsl").
			URIResolver(resolver).
			Compile(t.Context(), doc)
		require.Error(t, err,
			"higher-precedence strip-space=\"b\" does not cover \"a\"; the \"a\" overlap must still raise XTSE0270")
		require.Contains(t, err.Error(), "XTSE0270")
	})

	// TestStripSpacePrefixNamespaceContext verifies that a prefixed element name in
	// a strip-space rule is resolved using the namespace context in scope at the
	// declaration, not by local name alone. The same prefix "p" is bound to a
	// different URI in the imported module than in the importing module, so a
	// declaration-local resolution is required to pick the correct namespace.
	t.Run("prefix namespace context", func(t *testing.T) {
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
  <xsl:import href="` + importedModuleURI + `"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
</xsl:stylesheet>`

		resolver := &memResolver{files: map[string]string{
			importedModuleURI: imported,
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
	})

	// TestStripSpaceUndeclaredPrefix verifies that a prefix used in a strip-space
	// elements NameTest that is NOT in scope at the declaration raises XTSE0280,
	// and is never accepted via a compiler-wide binding leaked from an
	// imported module.
	t.Run("undeclared prefix", func(t *testing.T) {
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
  <xsl:import href="` + importedModuleURI + `"/>
  <xsl:strip-space elements="p:item"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

		resolver := &memResolver{files: map[string]string{
			importedModuleURI: imported,
		}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)

		_, err = xslt3.NewCompiler().
			BaseURI("mem:/main.xsl").
			URIResolver(resolver).
			Compile(t.Context(), doc)
		require.Error(t, err, "undeclared prefix in strip-space elements must raise XTSE0280")
		require.Contains(t, err.Error(), "XTSE0280")
	})

	// TestStripSpaceWildcardKindsNoConflict verifies that strip/preserve NameTests
	// of DIFFERENT kinds at the same import precedence do not raise a false
	// XTSE0270: their match priorities differ, so the conflict is resolved at
	// runtime by priority, and is no genuine same-priority conflict.
	t.Run("wildcard kinds no conflict", func(t *testing.T) {
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
	})

	// TestStripSpaceSameKindWildcardConflict verifies that a genuine same-kind,
	// same-name wildcard conflict at the same precedence still raises XTSE0270.
	t.Run("same kind wildcard conflict", func(t *testing.T) {
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
	})

	// TestStripSpaceWildcardOverlapConflict verifies that a namespace wildcard
	// "Q{urn:A}*" and a local-name wildcard "*:item" at the SAME import precedence
	// raise a static XTSE0270. Both NameTests have the same match priority (-0.25)
	// and both match Q{urn:A}item, so neither outranks the other for that name and
	// the strip/preserve outcome is undecidable. Per XSLT 3.0 this is a static error,
	// not a runtime tiebreak. Both orderings are checked because the rule is
	// symmetric.
	t.Run("wildcard overlap conflict", func(t *testing.T) {
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
					"equal-priority overlapping wildcards strip=%q preserve=%q must raise a static XTSE0270", tc.strip, tc.preserve)
				require.Contains(t, err.Error(), "XTSE0270")
			})
		}
	})

	// TestStripSpaceWildcardOverlapResolvedByPrecedence verifies that an
	// equal-priority wildcard overlap is NOT a conflict when a strictly
	// higher-precedence rule covers the overlap region. "*" (universal) at higher
	// import precedence covers Q{urn:A}item, so it decides the outcome and no
	// XTSE0270 fires for the lower-precedence "*:item" vs "Q{urn:A}*" pair.
	t.Run("wildcard overlap resolved by precedence", func(t *testing.T) {
		t.Parallel()

		imported := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*:item"/>
  <xsl:preserve-space elements="Q{urn:A}*"/>
</xsl:stylesheet>`

		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:import href="` + importedModuleURI + `"/>
  <xsl:preserve-space elements="*"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

		resolver := &memResolver{files: map[string]string{
			importedModuleURI: imported,
		}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)

		ss, err := xslt3.NewCompiler().
			BaseURI("mem:/main.xsl").
			URIResolver(resolver).
			Compile(t.Context(), doc)
		require.NoError(t, err,
			"a higher-precedence rule covering the overlap region must resolve the conflict, not raise XTSE0270")
		require.NotNil(t, ss)
	})

	// TestStripSpaceDisjointNamespaceWildcards verifies that two namespace wildcards
	// of DIFFERENT namespaces do not overlap and never raise XTSE0270 even at equal
	// import precedence and priority.
	t.Run("disjoint namespace wildcards", func(t *testing.T) {
		t.Parallel()

		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="Q{urn:A}*"/>
  <xsl:preserve-space elements="Q{urn:B}*"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)

		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err,
			"disjoint namespace wildcards must NOT raise XTSE0270")
		require.NotNil(t, ss)
	})

	// TestStripSpaceNamespaceWildcardPriority verifies that a namespace wildcard
	// (Q{uri}*) outranks the universal wildcard (*) at equal import precedence, so
	// strip-space="Q{urn:A}*" wins over preserve-space="*" for an element in urn:A.
	t.Run("namespace wildcard priority", func(t *testing.T) {
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
	})

	// TestStripSpaceRemapsAttributeSelection verifies that when an initial match
	// selection contains an attribute node, strip-space remaps the attribute onto
	// the stripped copy: a template matched on the attribute, navigating up to its
	// parent via XPath, must observe the STRIPPED parent (no whitespace-only text
	// nodes). See finding 664-1.
	t.Run("remaps attribute selection", func(t *testing.T) {
		t.Parallel()

		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="@id"><out><xsl:value-of select="count(../text())"/></out></xsl:template>
</xsl:stylesheet>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		// The root element has id="x" and two whitespace-only text nodes around the
		// <child> element. Without strip-space, count(../text()) would be 2.
		source, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root id="x">`+"\n  <child/>\n"+`</root>`))
		require.NoError(t, err)

		sel := evalSelection(t, "/*/@id", source)

		out, err := ss.ApplyTemplates(source).
			Selection(sel).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, out, "<out>0</out>",
			"attribute selection must remap to the stripped copy; got %q", out)
		require.NotContains(t, out, "<out>2</out>",
			"attribute selection must not see the unstripped original parent; got %q", out)
	})

	// TestStripSpaceRemapsNamespaceSelection verifies that when an initial match
	// selection contains a namespace node, strip-space remaps it onto the stripped
	// copy so that XPath navigation from the matched namespace node sees the
	// stripped tree.
	t.Run("remaps namespace selection", func(t *testing.T) {
		t.Parallel()

		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="node()|@*"/>
</xsl:stylesheet>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		source, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root xmlns:a="urn:a">`+"\n  <child/>\n"+`</root>`))
		require.NoError(t, err)

		sel := evalSelection(t, "/*/namespace::*", source)
		require.Positive(t, sel.Len(), "fixture must select at least one namespace node")

		// The built-in template for namespace nodes does nothing; the key assertion
		// is that the transform runs without panicking and remapping succeeds.
		out, err := ss.ApplyTemplates(source).
			Selection(sel).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Empty(t, out)
	})

	// TestStripSpaceRemapsImplicitXMLNamespaceSelection verifies that the
	// SYNTHESIZED implicit `xml` namespace node (which the XPath namespace axis
	// fabricates and which is NOT present in the owner element's Namespaces())
	// is remapped onto the stripped copy. A template matched on it that navigates
	// to its owner element's text() must observe the STRIPPED parent (0 text nodes),
	// matching the declared-namespace case. See finding 664-perf.
	t.Run("remaps implicit XML namespace selection", func(t *testing.T) {
		t.Parallel()

		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="namespace-node()"><out><xsl:value-of select="count(../text())"/></out></xsl:template>
  <xsl:template match="node()|@*"/>
</xsl:stylesheet>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		// The root element has two whitespace-only text nodes around <child>. Without
		// remapping, the implicit xml namespace node's owner points at the unstripped
		// original, so count(../text()) would be 2.
		source, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root>`+"\n  <child/>\n"+`</root>`))
		require.NoError(t, err)

		sel := evalSelection(t, "/*/namespace::xml", source)
		require.Equal(t, 1, sel.Len(), "fixture must select exactly the implicit xml namespace node")

		out, err := ss.ApplyTemplates(source).
			Selection(sel).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, out, "<out>0</out>",
			"implicit xml namespace selection must remap to the stripped copy; got %q", out)
		require.NotContains(t, out, "<out>2</out>",
			"implicit xml namespace selection must not see the unstripped original owner; got %q", out)
	})

	// TestStripSpaceSourceSchemaLocationAnnotationPreserved is the regression test
	// for finding 664-9: when the stylesheet is NOT schema-aware (no
	// xsl:import-schema) but the source document is typed solely via
	// xsi:schemaLocation, AND xsl:strip-space rules are active, the strip copy that
	// the transform runs against must still receive the type annotations gathered
	// during source validation.
	//
	// Before the fix, the schemaActive gate that decided whether to build the
	// original->copy node map only looked at ss.schemaAware / ss.schemas /
	// cfg.sourceSchemas — none of which are set for the xsi:schemaLocation-discovered
	// path. So the map stayed nil, remapValidationNode left the annotations on the
	// ORIGINAL nodes, and the transform (navigating the COPY) saw only untyped
	// nodes. "n instance of element(*, xs:integer)" therefore returned false.
	//
	// With the broadened gate the map is built whenever strip rules exist and the
	// source could be typed (here: it declares xsi:schemaLocation), so the annotation
	// rides onto the copy and the instance-of test passes. The no-strip control and
	// a no-schemaLocation control bracket the behaviour.
	t.Run("source schema location annotation preserved", func(t *testing.T) {
		t.Parallel()

		const schemaLoc = "mem:/ssa/schema.xsd"

		stylesheet := func(strip bool) string {
			stripDecl := ""
			if strip {
				stripDecl = `<xsl:strip-space elements="*"/>`
			}
			return `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="urn:ssa"
  version="3.0">
  ` + stripDecl + `
  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:value-of select="if (/t:doc/t:n instance of element(*, xs:integer)) then 'TYPED' else 'UNTYPED'"/>
  </xsl:template>
</xsl:stylesheet>`
		}

		const src = `<?xml version="1.0"?>
<doc xmlns="urn:ssa"
     xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xsi:schemaLocation="urn:ssa mem:/ssa/schema.xsd">
  <n>42</n>
</doc>`

		run := func(t *testing.T, strip bool) string {
			t.Helper()
			ctx := t.Context()
			ssDoc, err := helium.NewParser().Parse(ctx, []byte(stylesheet(strip)))
			require.NoError(t, err)
			ss, err := xslt3.NewCompiler().Compile(ctx, ssDoc)
			require.NoError(t, err)

			resolver := &exactRuntimeURIResolver{files: map[string]string{
				schemaLoc: ssaSchema,
			}}
			source, err := helium.NewParser().Parse(ctx, []byte(src))
			require.NoError(t, err)
			out, err := ss.Transform(source).URIResolver(resolver).Serialize(ctx)
			require.NoError(t, err)
			require.True(t, resolver.askedFor(schemaLoc),
				"resolver must be asked for the source schema %q; got %v", schemaLoc, resolver.asked)
			return out
		}

		// Control: with no strip-space rule the transform runs on the original
		// (validated) tree, so the annotation is naturally present.
		t.Run("no-strip control is typed", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, "TYPED", run(t, false),
				"without strip-space the validated source must carry the xs:integer annotation")
		})

		// The regression: strip-space active must NOT lose the annotation on the copy.
		t.Run("strip-space preserves annotation on copy", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, "TYPED", run(t, true),
				"strip-space must remap source type annotations onto the copy the transform runs on")
		})
	})

	// TestStripSpaceCopyExternalSubsetIsolated verifies that the strip-space copy
	// owns an INDEPENDENT external DTD subset: mutating the copy's external subset
	// (via a *DTD mutator) must NOT affect the source document's external subset.
	//
	// Before the fix, copyAndStrip shared the source's extSubset by pointer into the
	// copy. Because the copy can be exposed to user code (raw-result capture) and
	// *DTD has mutators, a handler mutating the copy's ExtSubset would corrupt the
	// source. The fix deep-copies the external subset so the two are fully isolated.
	// See finding codex 664-2 (extSubset aliasing).
	t.Run("copy external subset isolated", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"ext.dtd": {Data: []byte(
				`<!ELEMENT doc (item*)>` + "\n" +
					`<!ELEMENT item (#PCDATA)>` + "\n" +
					`<!ATTLIST item eid ID #IMPLIED>`)},
		}
		const source = `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc>
  <item eid="x">item</item>
</doc>`

		src, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(source))
		require.NoError(t, err)
		require.NotNil(t, src.ExtSubset(), "source must have an external subset")

		// Sanity: the source resolves the external-DTD-declared ID.
		require.NotNil(t, src.GetElementByID("x"))

		srcNotationsBefore := countNotations(src.ExtSubset())

		// Produce the strip-space copy and confirm it has its own external subset
		// that still resolves the external-DTD ID (round-3 behavior preserved).
		cp, err := xslt3.CopyAndStripForTest(src)
		require.NoError(t, err)
		require.NotNil(t, cp.ExtSubset(), "copy must carry over an external subset")
		require.NotSame(t, src.ExtSubset(), cp.ExtSubset(),
			"copy's external subset must be an independent *DTD, not the shared source pointer")
		require.NotNil(t, cp.GetElementByID("x"),
			"external-DTD ID must still resolve on the strip-space copy")

		// Mutate the COPY's external subset via a *DTD mutator. This must not touch
		// the source's external subset.
		_, err = cp.ExtSubset().AddNotation("injected", "", "injected.dtd")
		require.NoError(t, err)
		require.Positive(t, countNotations(cp.ExtSubset()),
			"mutation must register on the copy's external subset")

		require.Equal(t, srcNotationsBefore, countNotations(src.ExtSubset()),
			"mutating the copy's external subset must NOT change the source's external subset")
		_, found := src.ExtSubset().LookupNotation("injected")
		require.False(t, found,
			"notation added to the copy must not appear in the source's external subset")

		// And the deep copy must reproduce the ID-typed attribute declaration so id()
		// resolution truly comes from the copy's OWN subset (not residual sharing).
		adecls := cp.ExtSubset().AttributesForElement("item")
		require.NotEmpty(t, adecls, "copy's external subset must contain the item attribute decls")
		foundID := false
		for _, a := range adecls {
			if a.AType() == enum.AttrID {
				foundID = true
			}
		}
		require.True(t, foundID, "copy's external subset must contain the ID-typed attribute declaration")
	})

	// TestStripSpacePreservesExternalDTDIDs verifies that running a transform whose
	// stylesheet declares xsl:strip-space keeps id()/GetElementByID working for IDs
	// declared in an EXTERNAL DTD subset.
	//
	// The lazy GetElementByID fallback (document.GetElementByID) walks BOTH the
	// internal AND external DTD subsets when resolving ID-typed attributes. Without
	// strip-space the transform runs over the original source, whose external subset
	// (extSubset) is present, so id('x') resolves to the <item> element. With
	// strip-space the transform runs over copyAndStrip's private copy; that copy
	// drops the source ID table, so id() must fall back to the DTD walk. Before the
	// fix copyAndStrip only carried over the INTERNAL subset (via CopyDTDInfo) and
	// lost extSubset, so the copy's id('x') resolved to nothing and the two paths
	// disagreed. See finding codex 664-3.
	t.Run("preserves external DTD IDs", func(t *testing.T) {
		t.Parallel()

		// The ID attribute is declared ONLY in the external DTD, so resolving id('x')
		// requires consulting extSubset.
		fsys := fstest.MapFS{
			"ext.dtd": {Data: []byte(
				`<!ELEMENT doc (item*)>` + "\n" +
					`<!ELEMENT item (#PCDATA)>` + "\n" +
					`<!ATTLIST item eid ID #IMPLIED>`)},
		}
		const source = `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc>
  <item eid="x">item</item>
</doc>`

		parseSource := func() *helium.Document {
			src, err := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				FS(fsys).
				Parse(t.Context(), []byte(source))
			require.NoError(t, err)
			return src
		}

		// Sanity: the external subset really is what carries the ID declaration, so
		// the source itself resolves id('x') to the <item> element.
		require.NotNil(t, parseSource().GetElementByID("x"),
			"external-DTD-declared ID must resolve on the source document")

		// The stylesheet emits the local name of id('x') (or "none"), revealing the
		// ID semantics of the (possibly copied) source the transform runs over.
		stylesheet := func(withStrip bool) string {
			strip := ""
			if withStrip {
				strip = `  <xsl:strip-space elements="*"/>` + "\n"
			}
			return `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
` + strip + `  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:value-of select="if (id('x')) then local-name(id('x')) else 'none'"/>
  </xsl:template>
</xsl:stylesheet>`
		}

		// Baseline: without strip-space the transform runs over the original source,
		// whose extSubset resolves id('x') to <item>.
		noStripSS, err := xslt3.NewCompiler().Compile(t.Context(),
			mustParse(t, stylesheet(false)))
		require.NoError(t, err)
		noStripOut, err := xslt3.TransformString(t.Context(), parseSource(), noStripSS)
		require.NoError(t, err)
		require.Equal(t, "item", noStripOut,
			"baseline: external-DTD ID resolves without strip-space")

		// With strip-space the transform runs over copyAndStrip's copy, which must
		// carry over extSubset so id('x') resolves identically to the baseline.
		stripSS, err := xslt3.NewCompiler().Compile(t.Context(),
			mustParse(t, stylesheet(true)))
		require.NoError(t, err)
		stripOut, err := xslt3.TransformString(t.Context(), parseSource(), stripSS)
		require.NoError(t, err)
		require.Equal(t, "item", stripOut,
			"xsl:strip-space copy must preserve the source external DTD subset so id('x') still resolves")
	})

	// TestStripSpacePreservesPrefixedDTDIDs verifies that running a transform whose
	// stylesheet declares xsl:strip-space keeps id() resolving an ID-typed attribute
	// declared for a PREFIXED element (a:item) identically to the no-strip baseline.
	//
	// Without strip-space the transform runs over the original source, whose parser
	// ID table maps "x" to the <a:item> element. With strip-space the transform runs
	// over copyAndStrip's private copy. The copy used to drop the source ID table and
	// rely on GetElementByID's lazy fallback, which looked up the DTD ATTLIST by the
	// element's LocalName ("item") only and therefore missed the qualified ATTLIST
	// for "a:item" — so id('x') resolved to <a:item> without strip-space but to
	// nothing with strip-space. copyAndStrip now rebuilds the copy's ID table from
	// the source's, so both paths agree. See finding codex 664-6.
	t.Run("preserves prefixed DTD IDs", func(t *testing.T) {
		t.Parallel()

		const source = `<?xml version="1.0"?>
<!DOCTYPE a:doc [
<!ELEMENT a:doc (a:item*)>
<!ELEMENT a:item (#PCDATA)>
<!ATTLIST a:item eid ID #IMPLIED>
<!ATTLIST a:doc xmlns:a CDATA #IMPLIED>
]>
<a:doc xmlns:a="urn:a">
  <a:item eid="x">item</a:item>
</a:doc>`

		parseSource := func() *helium.Document {
			src, err := helium.NewParser().Parse(t.Context(), []byte(source))
			require.NoError(t, err)
			return src
		}

		// Sanity: the source itself resolves the prefixed-element ID to <a:item>.
		require.NotNil(t, parseSource().GetElementByID("x"),
			"prefixed-element ID must resolve on the source document")

		// The stylesheet emits the local name of id('x') (or "none"), revealing the
		// ID semantics of the (possibly copied) source the transform runs over.
		stylesheet := func(withStrip bool) string {
			strip := ""
			if withStrip {
				strip = `  <xsl:strip-space elements="*"/>` + "\n"
			}
			return `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
` + strip + `  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:value-of select="if (id('x')) then local-name(id('x')) else 'none'"/>
  </xsl:template>
</xsl:stylesheet>`
		}

		noStripSS, err := xslt3.NewCompiler().Compile(t.Context(),
			mustParse(t, stylesheet(false)))
		require.NoError(t, err)
		noStripOut, err := xslt3.TransformString(t.Context(), parseSource(), noStripSS)
		require.NoError(t, err)
		require.Equal(t, "item", noStripOut,
			"baseline: prefixed-element ID resolves without strip-space")

		stripSS, err := xslt3.NewCompiler().Compile(t.Context(),
			mustParse(t, stylesheet(true)))
		require.NoError(t, err)
		stripOut, err := xslt3.TransformString(t.Context(), parseSource(), stripSS)
		require.NoError(t, err)
		require.Equal(t, "item", stripOut,
			"xsl:strip-space copy must resolve the prefixed-element ID identically to the baseline")
	})

	// TestStripSpacePreservesIDsSkip verifies that running a transform whose
	// stylesheet declares xsl:strip-space preserves the source document's
	// ID-skip state. A source parsed with SkipIDs(true) must NOT register its
	// xml:id values, so fn:id('x') (and GetElementByID) returns nothing — both
	// without strip-space (which transforms the original source directly) AND
	// with strip-space (which transforms a private copy produced by copyAndStrip).
	//
	// Before the fix, copyAndStrip created a fresh document that dropped the
	// source's idsSkip flag, so the copy fell back to an O(n) xml:id walk and
	// fn:id('x') wrongly matched. See finding codex 664-2.
	t.Run("preserves IDs under skip", func(t *testing.T) {
		t.Parallel()

		// The stylesheet emits "found" when id('x') resolves to an element and
		// "none" otherwise, so the transform output reveals the ID semantics of
		// the (possibly copied) source the transform actually runs over.
		idLookup := func(withStrip bool) string {
			strip := ""
			if withStrip {
				strip = `  <xsl:strip-space elements="*"/>` + "\n"
			}
			return `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
` + strip + `  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:value-of select="if (id('x')) then 'found' else 'none'"/>
  </xsl:template>
</xsl:stylesheet>`
		}

		parseSource := func() *helium.Document {
			// SkipIDs(true) means xml:id values are NOT interned, so id('x') must
			// not find the element.
			src, err := helium.NewParser().SkipIDs(true).Parse(t.Context(),
				[]byte(`<doc>
  <item xml:id="x">hello</item>
</doc>`))
			require.NoError(t, err)
			return src
		}

		// Baseline: without strip-space, the transform runs over the original
		// SkipIDs source directly, so id('x') finds nothing.
		noStripSS, err := xslt3.NewCompiler().Compile(t.Context(),
			mustParse(t, idLookup(false)))
		require.NoError(t, err)
		noStripOut, err := xslt3.TransformString(t.Context(), parseSource(), noStripSS)
		require.NoError(t, err)
		require.Equal(t, "none", noStripOut,
			"baseline: SkipIDs source without strip-space must not resolve id('x')")

		// With strip-space, the transform runs over copyAndStrip's copy. The copy
		// must inherit idsSkip so id('x') still finds nothing — matching baseline.
		stripSS, err := xslt3.NewCompiler().Compile(t.Context(),
			mustParse(t, idLookup(true)))
		require.NoError(t, err)
		stripOut, err := xslt3.TransformString(t.Context(), parseSource(), stripSS)
		require.NoError(t, err)
		require.Equal(t, "none", stripOut,
			"xsl:strip-space copy must preserve the source's SkipIDs state so id('x') stays unresolved")
	})

	// TestStripSpaceCopyNoEncoding verifies that the single-pass strip-space copy
	// faithfully reproduces a source document's encoding state. A source whose XML
	// declaration omitted an encoding must yield a copy that ALSO has no encoding
	// declaration: neither the copy's recorded encoding nor its serialized form may
	// carry a synthesized encoding="utf8" the source never had. See finding 664-5.
	t.Run("copy no encoding", func(t *testing.T) {
		t.Parallel()

		src, err := helium.NewParser().Parse(t.Context(),
			[]byte("<doc>\n  <item>x</item>\n</doc>"))
		require.NoError(t, err)
		require.Empty(t, src.RawEncoding(),
			"fixture source must have no encoding declaration to be meaningful")

		dst, err := xslt3.CopyAndStripForTest(src)
		require.NoError(t, err)

		require.Empty(t, dst.RawEncoding(),
			"strip-space copy of an encoding-less source must have no encoding (got %q)", dst.RawEncoding())

		srcOut, err := helium.WriteString(src)
		require.NoError(t, err)
		dstOut, err := helium.WriteString(dst)
		require.NoError(t, err)

		require.NotContains(t, srcOut, "encoding=",
			"sanity: encoding-less source must not serialize an encoding= attribute (got %q)", srcOut)
		require.NotContains(t, dstOut, "encoding=",
			"strip-space copy must not serialize a spurious encoding= attribute (got %q)", dstOut)
	})

	// TestStripSpaceCopyEncodingFaithful verifies the converse: when the source DOES
	// declare an encoding, version, or standalone, the strip-space copy reproduces
	// each exactly.
	t.Run("copy encoding faithful", func(t *testing.T) {
		t.Parallel()

		src, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<?xml version="1.1" encoding="UTF-8" standalone="yes"?>`+"\n<doc><item>x</item></doc>"))
		require.NoError(t, err)
		require.Equal(t, "UTF-8", src.RawEncoding())
		require.Equal(t, "1.1", src.Version())

		dst, err := xslt3.CopyAndStripForTest(src)
		require.NoError(t, err)

		require.Equal(t, src.RawEncoding(), dst.RawEncoding(),
			"strip-space copy must reproduce the source encoding exactly")
		require.Equal(t, src.Version(), dst.Version(),
			"strip-space copy must reproduce the source version exactly")
		require.Equal(t, src.Standalone(), dst.Standalone(),
			"strip-space copy must reproduce the source standalone exactly")

		dstOut, err := helium.WriteString(dst)
		require.NoError(t, err)
		require.Contains(t, dstOut, `encoding="UTF-8"`,
			"strip-space copy of an encoded source must serialize its encoding (got %q)", dstOut)
	})

	// TestStripSpaceEmptyRemappedSelectionProducesNoOutput verifies that an initial
	// match selection that is ENTIRELY removed by strip-space remaps to an empty
	// sequence and produces NO output — it must not fall through to applying
	// templates to the source document.
	//
	// The selection /*/text() picks the single whitespace-only text node under
	// <root>. With xsl:strip-space that node has no copy, so the remapped selection
	// becomes empty (length 0). apply-templates over an empty sequence emits
	// nothing. Before the fix, the zero-length remapped selection was treated as
	// "no selection supplied", so the transform fell through to the source document
	// and wrongly invoked the "/" root template, emitting <wrong/>. See finding
	// 664-1.
	t.Run("empty remapped selection produces no output", func(t *testing.T) {
		t.Parallel()

		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/"><wrong/></xsl:template>
  <xsl:template match="text()"><kept/></xsl:template>
</xsl:stylesheet>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		// <root> contains only a whitespace-only text node, which strip-space removes.
		source, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root>`+"\n  \n"+`</root>`))
		require.NoError(t, err)

		sel := evalSelection(t, "/*/text()", source)
		require.Equal(t, 1, sel.Len(), "fixture must select the single whitespace-only text node")

		out, err := ss.ApplyTemplates(source).
			Selection(sel).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Empty(t, out,
			"a fully-stripped initial selection must produce no output; got %q", out)
		require.NotContains(t, out, "<wrong/>",
			"must not fall through to the source-document root template; got %q", out)
	})

	// TestStripSpaceDropsOmittedSelectionNodes verifies that when an initial match
	// selection mixes a whitespace-only text node (which strip-space OMITS from the
	// copy) with a real element child, the omitted node is DROPPED from the remapped
	// selection, and never passed through pointing at the unstripped original. The
	// apply loop must then compute position()/last() from the filtered sequence.
	//
	// The selection /*/text()[1] | /*/child selects the leading whitespace-only text
	// node and the <child> element. After strip-space the whitespace text node has no
	// copy, so the remapped selection should contain only <child>: position()=1,
	// last()=1. Before the fix the omitted node was passed through, so the selection
	// kept length 2 and <child> reported position()=2, last()=2. See finding 664-6.
	t.Run("drops omitted selection nodes", func(t *testing.T) {
		t.Parallel()

		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="child"><out><xsl:value-of select="position()"/>/<xsl:value-of select="last()"/></out></xsl:template>
  <xsl:template match="text()"/>
</xsl:stylesheet>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		// The root element has a leading whitespace-only text node (stripped) followed
		// by the <child> element.
		source, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root>`+"\n  <child/>\n"+`</root>`))
		require.NoError(t, err)

		sel := evalSelection(t, "/*/text()[1] | /*/child", source)
		require.Equal(t, 2, sel.Len(), "fixture must select the whitespace text node and the child")

		out, err := ss.ApplyTemplates(source).
			Selection(sel).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, out, "<out>1/1</out>",
			"omitted whitespace node must be dropped so child is position 1 of 1; got %q", out)
		require.NotContains(t, out, "<out>2/2</out>",
			"the stripped selection must not retain the omitted whitespace node; got %q", out)
	})

	// TestStripSpaceDoesNotMutateSource verifies that running a transform whose
	// stylesheet declares xsl:strip-space does NOT mutate the caller-owned source
	// document. Whitespace-only text nodes must be stripped on a private copy used
	// only inside the transform; the original tree the caller passed in must be left
	// untouched so it can be reused (e.g. for a subsequent XPath query or a second
	// transform). See finding A-004.
	t.Run("does not mutate source", func(t *testing.T) {
		t.Parallel()

		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
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

		source, err := helium.NewParser().Parse(t.Context(),
			[]byte("<doc>\n  <item>x</item>\n</doc>"))
		require.NoError(t, err)

		before := countWhitespaceTextNodes(source)
		require.Positive(t, before,
			"test fixture must contain whitespace-only text nodes to be meaningful")

		out, err := xslt3.TransformString(t.Context(), source, ss)
		require.NoError(t, err)

		// The transform output must reflect strip-space: the whitespace-only text
		// nodes between elements are gone in the result.
		require.NotContains(t, out, "<doc>\n",
			"strip-space must remove whitespace-only text nodes from the transform result; got %q", out)
		require.Contains(t, out, "<item>x</item>",
			"non-whitespace content must survive; got %q", out)

		// The caller-owned source DOM must be untouched: its whitespace-only text
		// nodes are still present after the transform.
		after := countWhitespaceTextNodes(source)
		require.Equal(t, before, after,
			"xsl:strip-space must not mutate the caller's source document (had %d whitespace text nodes, now %d)", before, after)

		// A second transform of the same source must still see the whitespace, i.e.
		// produce the same stripped output (it would differ if the first run had
		// destructively stripped the shared tree).
		out2, err := xslt3.TransformString(t.Context(), source, ss)
		require.NoError(t, err)
		require.Equal(t, out, out2,
			"repeated transforms of the same reused source must produce identical output")
	})

	// TestStripSpaceValidatesBeforeStripping verifies that strict source-schema
	// validation runs on the ORIGINAL (un-stripped) source tree, BEFORE xsl:strip-space
	// removes whitespace-only text nodes. If validation ran after stripping, a
	// whitespace-only element that the schema requires to be empty would have its
	// content removed first, masking the validation error.
	//
	// The schema declares <s> as a simpleType restricting xs:string to length 0, so
	// non-empty content (including a single space) is invalid. The source <s> </s>
	// holds a whitespace-only text node that xsl:strip-space elements="s" would remove.
	// With strip-before-validate (the regression), the whitespace node is gone before
	// validation, <s> looks empty, and validation wrongly passes. With validate-before-
	// strip (correct, matching the no-strip control), validation sees the space and
	// fails. See finding 664-8.
	t.Run("validates before stripping", func(t *testing.T) {
		t.Parallel()

		stylesheet := func(strip bool) string {
			stripDecl := ""
			if strip {
				stripDecl = `<xsl:strip-space elements="s"/>`
			}
			return `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  version="3.0" default-validation="strict">
  ` + stripDecl + `
  <xsl:import-schema>
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
      <xs:simpleType name="emptyString">
        <xs:restriction base="xs:string">
          <xs:length value="0"/>
        </xs:restriction>
      </xs:simpleType>
      <xs:element name="s" type="emptyString"/>
      <xs:complexType name="docType">
        <xs:sequence>
          <xs:element ref="s"/>
        </xs:sequence>
      </xs:complexType>
      <xs:element name="doc" type="docType"/>
    </xs:schema>
  </xsl:import-schema>
  <xsl:output method="text"/>
  <xsl:template match="/">done</xsl:template>
</xsl:stylesheet>`
		}

		const src = `<doc><s> </s></doc>`

		run := func(t *testing.T, strip bool) error {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(stylesheet(strip)))
			require.NoError(t, err)
			ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
			require.NoError(t, err)
			source, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)
			_, err = xslt3.TransformString(t.Context(), source, ss)
			return err
		}

		// Control: without strip-space the whitespace-only <s> content is present at
		// validation time and must fail strict validation.
		t.Run("no-strip control fails validation", func(t *testing.T) {
			t.Parallel()
			err := run(t, false)
			require.Error(t, err, "whitespace-only <s> must fail length-0 validation")
			require.Contains(t, err.Error(), "validation failed")
		})

		// With strip-space, validation must STILL fail: it runs on the original tree
		// before the whitespace node is removed. Before the fix this wrongly passed.
		t.Run("strip-space still fails validation", func(t *testing.T) {
			t.Parallel()
			err := run(t, true)
			require.Error(t, err, "strip-space must not mask the validation error")
			require.Contains(t, err.Error(), "validation failed")
		})
	})

	// TestStripSpaceReorderPinning pins shouldStripWhitespace's behavior across
	// the reorder that evaluates inheritedXMLSpace LAST (after the schema/rule/DTD
	// verdict) instead of first. xml:space="preserve" can only ever convert a
	// strip verdict into a preserve, never the reverse, so these cases must pass
	// identically before and after the reorder.
	//
	// (a)-(c) use DTD element-only content (not xsl:strip-space) to drive the
	// strip verdict: an explicit xsl:strip-space rule makes the WHOLE source tree
	// go through the unrelated copyAndStrip pre-copy (strip_space_copy.go)
	// instead of the runtime shouldStripWhitespace call sites this reorder
	// touches (applyTemplates / onNoMatchTextOnlyCopy / the apply-templates
	// selection filter), so they would not exercise this change. All four cases
	// use xsl:output method="text" with built-in template rules (no xsl:copy-of,
	// which never consults whitespace stripping at all) so the string-value
	// output directly reveals which whitespace-only text nodes survived.
	t.Run("reorder pinning", func(t *testing.T) {
		const dtdOneChild = `<!DOCTYPE doc [<!ELEMENT doc (n)*><!ELEMENT n (#PCDATA)>]>`
		const textCopyStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:apply-templates/>
  </xsl:template>
</xsl:stylesheet>`

		// (a) xml:space="preserve" on the element itself beats a DTD element-only
		// strip verdict.
		t.Run("xml:space preserve beats DTD element-only content", func(t *testing.T) {
			t.Parallel()
			ss := compileStylesheetString(t, textCopyStylesheet)
			source, err := helium.NewParser().Parse(t.Context(),
				[]byte(dtdOneChild+`<doc xml:space="preserve"> <n>1</n> </doc>`))
			require.NoError(t, err)
			out, err := xslt3.TransformString(t.Context(), source, ss)
			require.NoError(t, err)
			require.Equal(t, " 1 ", out,
				"xml:space=\"preserve\" must override a DTD element-only strip verdict")
		})

		// (b) an inner xml:space="default" re-enables a DTD element-only strip
		// verdict under an outer xml:space="preserve" ancestor.
		t.Run("inner xml:space default re-enables stripping", func(t *testing.T) {
			t.Parallel()
			ss := compileStylesheetString(t, textCopyStylesheet)
			source, err := helium.NewParser().Parse(t.Context(), []byte(
				`<!DOCTYPE doc [<!ELEMENT doc (mid)*><!ELEMENT mid (n)*><!ELEMENT n (#PCDATA)>]>`+
					`<doc xml:space="preserve"> <mid xml:space="default"> <n>1</n> </mid> </doc>`))
			require.NoError(t, err)
			out, err := xslt3.TransformString(t.Context(), source, ss)
			require.NoError(t, err)
			require.Equal(t, " 1 ", out,
				"inner xml:space=\"default\" must re-enable the DTD strip verdict despite the outer preserve")
		})

		// (c) DTD element-only content strips whitespace with NO xsl:strip-space
		// rules at all, so the reordered function must still reach the DTD check.
		t.Run("DTD element-only content strips with no strip rules", func(t *testing.T) {
			t.Parallel()
			ss := compileStylesheetString(t, textCopyStylesheet)
			source, err := helium.NewParser().Parse(t.Context(),
				[]byte(dtdOneChild+`<doc> <n>1</n> </doc>`))
			require.NoError(t, err)
			out, err := xslt3.TransformString(t.Context(), source, ss)
			require.NoError(t, err)
			require.Equal(t, "1", out,
				"DTD element-only content must strip whitespace with no explicit strip rules")
		})

		// (d) a schema element-only type strips whitespace despite an explicit
		// xsl:preserve-space rule covering the element. The schemaLocation
		// reference makes this go through the schema-validated in-place strip
		// pre-pass (stripWhitespaceFromNodeInto), so this also pins stripVerdict
		// as called from change 2.
		t.Run("schema element-only type strips despite preserve-space", func(t *testing.T) {
			t.Parallel()
			const schemaLoc = "mem:/reorder/schema.xsd"
			ss := compileStylesheetString(t, `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:preserve-space elements="*"/>
  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:apply-templates/>
  </xsl:template>
</xsl:stylesheet>`)

			resolver := &exactRuntimeURIResolver{files: map[string]string{schemaLoc: ssaSchema}}
			source, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<doc xmlns="urn:ssa"
     xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xsi:schemaLocation="urn:ssa `+schemaLoc+`"> <n>42</n> </doc>`))
			require.NoError(t, err)
			out, err := ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
			require.NoError(t, err)
			require.True(t, resolver.askedFor(schemaLoc), "resolver must be asked for the source schema")
			require.Equal(t, "42", out,
				"an element-only schema type must strip whitespace despite xsl:preserve-space")
		})

		// (e) a doc()-loaded tree keeps an inner xml:space="preserve" subtree's
		// whitespace across more than one nesting level (the pushed-frame
		// recompute-or-inherit step) while stripping whitespace elsewhere in the
		// same document. This exercises stripWhitespaceFromNodeInto's threaded
		// preserve flag, the strip pre-pass body of shouldStripWhitespace.
		t.Run("doc()-loaded tree keeps a nested xml:space preserve subtree", func(t *testing.T) {
			t.Parallel()
			const otherURI = "mem:/reorder/other.xml"
			resolver := &exactRuntimeURIResolver{files: map[string]string{
				otherURI: `<outer xml:space="preserve"><mid><inner>   </inner></mid></outer>`,
			}}

			ctx := t.Context()
			ssDoc, err := helium.NewParser().Parse(ctx, []byte(`<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <result>
      <local><xsl:copy-of select="/wrap/local"/></local>
      <remote><xsl:copy-of select="doc('`+otherURI+`')/outer"/></remote>
    </result>
  </xsl:template>
</xsl:stylesheet>`))
			require.NoError(t, err)
			ss, err := xslt3.NewCompiler().Compile(ctx, ssDoc)
			require.NoError(t, err)

			source, err := helium.NewParser().Parse(ctx, []byte(`<wrap><local>   </local></wrap>`))
			require.NoError(t, err)
			out, err := ss.Transform(source).URIResolver(resolver).Serialize(ctx)
			require.NoError(t, err)

			require.Contains(t, out, "<inner>   </inner>",
				"xml:space=\"preserve\" declared two levels up must still preserve a nested "+
					"whitespace-only text node in a doc()-loaded tree; got %q", out)
			require.Contains(t, out, "<local/>",
				"whitespace in the directly-sourced document must still be stripped; got %q", out)
		})
	})
}

// evalSelection evaluates an XPath expression against the source document and
// returns the resulting sequence, suitable for feeding to Invocation.Selection.
func evalSelection(t *testing.T, expr string, doc *helium.Document) xpath3.Sequence {
	t.Helper()
	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)
	res, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)
	return res.Sequence()
}

// ssaSchema types <doc>'s <n> child as xs:integer so a source validated against
// it carries an xs:integer annotation on <n>. The source references this schema
// purely through xsi:schemaLocation (NOT via xsl:import-schema), so the
// stylesheet itself is NOT schema-aware.
const ssaSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="urn:ssa"
           xmlns:t="urn:ssa"
           elementFormDefault="qualified">
  <xs:element name="doc" type="t:docType"/>
  <xs:complexType name="docType">
    <xs:sequence>
      <xs:element name="n" type="xs:integer"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

// countNotations returns the number of notation-declaration children directly
// under the given DTD.
func countNotations(dtd *helium.DTD) int {
	if dtd == nil {
		return 0
	}
	count := 0
	for c := dtd.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.NotationNode {
			count++
		}
	}
	return count
}

func mustParse(t *testing.T, doc string) *helium.Document {
	t.Helper()
	d, err := helium.NewParser().Parse(t.Context(), []byte(doc))
	require.NoError(t, err)
	return d
}

// countWhitespaceTextNodes returns the number of whitespace-only text nodes
// anywhere in the tree rooted at n.
func countWhitespaceTextNodes(n helium.Node) int {
	count := 0
	for child := range helium.Children(n) {
		if child.Type() == helium.TextNode {
			if strings.TrimSpace(string(child.Content())) == "" {
				count++
			}
		}
		count += countWhitespaceTextNodes(child)
	}
	return count
}

// buildLargeStripSpaceSource generates an XML document with many elements and
// abundant whitespace-only text nodes between them (indentation/newlines), plus
// a couple of namespace declarations so the namespace-handling path is exercised.
// The shape is a few thousand elements so the per-node cost of the source copy is
// visible in the benchmark.
func buildLargeStripSpaceSource(sections, itemsPerSection int) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?>` + "\n")
	b.WriteString(`<catalog xmlns="urn:bench:cat" xmlns:m="urn:bench:meta">` + "\n")
	for s := range sections {
		b.WriteString("  <section id=\"")
		b.WriteString(strconv.Itoa(s))
		b.WriteString("\">\n")
		for i := range itemsPerSection {
			b.WriteString("    <item m:rank=\"")
			b.WriteString(strconv.Itoa(i))
			b.WriteString("\">\n")
			b.WriteString("      <name>Item ")
			b.WriteString(strconv.Itoa(i))
			b.WriteString("</name>\n")
			b.WriteString("      <m:note>note</m:note>\n")
			b.WriteString("    </item>\n")
		}
		b.WriteString("  </section>\n")
	}
	b.WriteString("</catalog>\n")
	return []byte(b.String())
}

const stripBenchStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:c="urn:bench:cat" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
</xsl:stylesheet>`

const noStripBenchStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:c="urn:bench:cat" version="3.0">
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
</xsl:stylesheet>`

func compileBenchStylesheet(b *testing.B, src string) *xslt3.Stylesheet {
	b.Helper()
	doc, err := helium.NewParser().Parse(b.Context(), []byte(src))
	require.NoError(b, err)
	ss, err := xslt3.NewCompiler().Compile(b.Context(), doc)
	require.NoError(b, err)
	return ss
}

// BenchmarkStripSpaceTransform measures the strip-space transform path (which
// triggers the single-pass source copy) against an identical transform with no
// strip-space rules (no copy at all), so the copy overhead is directly visible.
func BenchmarkStripSpaceTransform(b *testing.B) {
	srcBytes := buildLargeStripSpaceSource(40, 30) // ~5k+ elements with whitespace

	source, err := helium.NewParser().Parse(b.Context(), srcBytes)
	require.NoError(b, err)

	stripSS := compileBenchStylesheet(b, stripBenchStylesheet)
	noStripSS := compileBenchStylesheet(b, noStripBenchStylesheet)

	b.Run("strip-space", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, err := xslt3.Transform(b.Context(), source, stripSS)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("no-strip-space", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, err := xslt3.Transform(b.Context(), source, noStripSS)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
