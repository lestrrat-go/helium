package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedModuleWrapperXMLBase covers an EMBEDDED stylesheet module (selected
// by a fragment identifier, whose root element's parent is another element) whose
// effective static base must fold the FULL xml:base ancestor chain — the wrapper
// element's xml:base AND the embedded xsl:stylesheet's own xml:base — onto the
// module document URI.
//
// Module at mem://pkg/emb.xsl:
//
//	<wrapper xml:base="outer/">
//	  <xsl:stylesheet id="style1" xml:base="inner/" ...>
//
// so the embedded module's effective base is mem://pkg/outer/inner/. The root
// use-when and every template/global in the embedded module must resolve relative
// references against that base, not mem://pkg/inner/ (wrapper xml:base dropped)
// nor the bare mem://pkg/emb.xsl.
func TestEmbeddedModuleWrapperXMLBase(t *testing.T) {
	const (
		mainBase    = "mem://pkg/main.xsl"
		moduleURI   = "mem://pkg/emb.xsl"
		flagURI     = "mem://pkg/outer/inner/flag.xml" // wrapper "outer/" + stylesheet "inner/" folded onto module URI
		effectiveBU = "mem://pkg/outer/inner/"
		module      = `<?xml version="1.0"?>
<wrapper xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xml:base="outer/">
  <xsl:stylesheet id="style1" version="3.0"
    xml:base="inner/"
    use-when="doc-available('flag.xml')">
    <xsl:template match="item"><base><xsl:value-of select="static-base-uri()"/></base></xsl:template>
  </xsl:stylesheet>
</wrapper>`
	)

	for _, link := range []struct {
		name    string
		linkElt string
	}{
		{declImport, `<xsl:import href="emb.xsl#style1"/>`},
		{declInclude, `<xsl:include href="emb.xsl#style1"/>`},
	} {
		t.Run(link.name, func(t *testing.T) {
			ctx := t.Context()
			main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  ` + link.linkElt + `
  <xsl:template match="/"><result><xsl:apply-templates select="//item"/></result></xsl:template>
</xsl:stylesheet>`

			resolver := &exactURIResolver{files: map[string]string{
				moduleURI: module,
				flagURI:   `<flag/>`,
			}}

			doc, err := helium.NewParser().Parse(ctx, []byte(main))
			require.NoError(t, err)

			ss, err := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(resolver).Compile(ctx, doc)
			require.NoError(t, err)

			source, err := helium.NewParser().Parse(ctx, []byte(`<doc><item>x</item></doc>`))
			require.NoError(t, err)

			result, err := ss.Transform(source).Serialize(ctx)
			require.NoError(t, err)

			// (a-root-use-when) The root use-when doc-available('flag.xml') must
			// resolve against the embedded module's effective base (wrapper +
			// stylesheet xml:base), so the module is INCLUDED and its template runs.
			require.True(t, resolver.askedFor(flagURI),
				"root use-when must probe the full-chain effective base %q; asked=%v", flagURI, resolver.asked)
			// (a-template-base) The template's static base URI must equal the
			// embedded module's effective base — the wrapper xml:base must not be
			// dropped.
			require.Contains(t, result, "<base>"+effectiveBU+"</base>",
				"template static base must fold the wrapper + stylesheet xml:base to %q", effectiveBU)
		})
	}
}

// TestEmbeddedModuleWrapperDocSelfResolves covers doc(”) / document(”) called
// from inside an EMBEDDED stylesheet module whose effective base folds a wrapper
// xml:base and the embedded stylesheet's own xml:base. The module's templates
// compile under that folded effective base, so the module document must be cached
// under that same key for doc(”)/document(”) to resolve to the module's OWN
// document rather than falling back to the principal stylesheet.
func TestEmbeddedModuleWrapperDocSelfResolves(t *testing.T) {
	const (
		mainBase  = "mem://pkg/main.xsl"
		moduleURI = "mem://pkg/emb.xsl"
	)

	for _, fn := range []struct {
		name string
		expr string
	}{
		{"doc", `doc('')`},
		{"document", `document('')`},
	} {
		for _, link := range []struct {
			name    string
			linkElt string
		}{
			{declImport, `<xsl:import href="emb.xsl#style1"/>`},
			{declInclude, `<xsl:include href="emb.xsl#style1"/>`},
		} {
			t.Run(fn.name+"/"+link.name, func(t *testing.T) {
				ctx := t.Context()

				module := `<?xml version="1.0"?>
<wrapper xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:d="urn:test" xml:base="outer/">
  <xsl:stylesheet id="style1" version="3.0" xml:base="inner/">
    <d:data>MODULE-DATA</d:data>
    <xsl:template match="item"><out><xsl:value-of select="` + fn.expr + `//*[local-name()='data']"/></out></xsl:template>
  </xsl:stylesheet>
</wrapper>`

				main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:d="urn:test">
  ` + link.linkElt + `
  <d:data>MAIN-DATA</d:data>
  <xsl:template match="/"><result><xsl:apply-templates select="//item"/></result></xsl:template>
</xsl:stylesheet>`

				resolver := &exactURIResolver{files: map[string]string{moduleURI: module}}

				doc, err := helium.NewParser().Parse(ctx, []byte(main))
				require.NoError(t, err)

				ss, err := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(resolver).Compile(ctx, doc)
				require.NoError(t, err)

				source, err := helium.NewParser().Parse(ctx, []byte(`<doc><item>x</item></doc>`))
				require.NoError(t, err)

				result, err := ss.Transform(source).Serialize(ctx)
				require.NoError(t, err)

				require.Contains(t, result, "<out>MODULE-DATA</out>",
					"%s from an embedded module with wrapper xml:base must resolve to the module's own document", fn.expr)
				require.NotContains(t, result, "MAIN-DATA",
					"%s must not fall back to the principal stylesheet document", fn.expr)
			})
		}
	}
}
