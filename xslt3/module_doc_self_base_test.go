package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestModuleDocSelfResolvesAgainstEffectiveBase is a regression for doc(”) /
// document(”) called from within an imported/included stylesheet module whose
// ROOT element carries xml:base. Such a module compiles its templates under the
// FOLDED effective module base (module URI + root xml:base), so doc(”) from a
// module template must resolve to the MODULE's OWN document — not the principal
// stylesheet.
//
// The module is at mem://pkg/inc.xsl with root xml:base="sub/", so its effective
// base is mem://pkg/sub/inc.xsl. Its module document is cached only under the
// bare URI before this fix, so the doc(”) lookup (keyed on the template's folded
// base URI) missed and wrongly fell back to the principal stylesheet, returning
// MAIN-DATA instead of MODULE-DATA.
func TestModuleDocSelfResolvesAgainstEffectiveBase(t *testing.T) {
	const (
		mainBase  = "mem://pkg/main.xsl"
		moduleURI = "mem://pkg/inc.xsl"
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
			{declImport, `<xsl:import href="inc.xsl"/>`},
			{declInclude, `<xsl:include href="inc.xsl"/>`},
		} {
			t.Run(fn.name+"/"+link.name, func(t *testing.T) {
				ctx := t.Context()

				module := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:d="urn:test"
  xml:base="sub/">
  <d:data>MODULE-DATA</d:data>
  <xsl:template match="item"><out><xsl:value-of select="` + fn.expr + `//*[local-name()='data']"/></out></xsl:template>
</xsl:stylesheet>`

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
					"%s from a module with root xml:base must resolve to the module's own document, not the principal stylesheet", fn.expr)
				require.NotContains(t, result, "MAIN-DATA",
					"%s must not fall back to the principal stylesheet document", fn.expr)
			})
		}
	}
}
