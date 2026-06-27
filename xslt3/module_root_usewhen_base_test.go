package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestModuleRootUseWhenResolvesAgainstEffectiveBase is a regression for a root
// use-when on an included/imported stylesheet module whose ROOT element carries
// xml:base. The root use-when must be evaluated against the module's EFFECTIVE
// static base (its root xml:base folded into the module URI), exactly like the
// module's own globals and templates — not the including module's base nor the
// bare module URI.
//
// The module's root use-when is doc-available('flag.xml') with root
// xml:base="sub/". Only mem://pkg/sub/flag.xml exists. With the correct base the
// reference resolves to mem://pkg/sub/flag.xml (module included); with the wrong
// base it resolves to mem://pkg/flag.xml (or against the including module's
// base), which does not exist, so the whole module would be wrongly excluded.
func TestModuleRootUseWhenResolvesAgainstEffectiveBase(t *testing.T) {
	const (
		mainBase  = "mem://pkg/main.xsl"
		moduleURI = "mem://pkg/inc.xsl"
		flagURI   = "mem://pkg/sub/flag.xml" // effective base (module URI + xml:base="sub/")
		module    = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xml:base="sub/"
  use-when="doc-available('flag.xml')">
  <xsl:template match="item"><out>MODULE</out></xsl:template>
</xsl:stylesheet>`
	)

	for _, link := range []struct {
		name    string
		linkElt string
	}{
		{"import", `<xsl:import href="inc.xsl"/>`},
		{"include", `<xsl:include href="inc.xsl"/>`},
	} {
		for _, flag := range []struct {
			name        string
			flagPresent bool
		}{
			{"flag_present_module_included", true},
			{"flag_absent_module_excluded", false},
		} {
			t.Run(link.name+"/"+flag.name, func(t *testing.T) {
				ctx := t.Context()
				main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  ` + link.linkElt + `
  <xsl:template match="/"><result><xsl:apply-templates select="//item"/></result></xsl:template>
</xsl:stylesheet>`

				files := map[string]string{moduleURI: module}
				if flag.flagPresent {
					files[flagURI] = `<flag/>`
				}
				resolver := &exactURIResolver{files: files}

				doc, err := helium.NewParser().Parse(ctx, []byte(main))
				require.NoError(t, err)

				ss, err := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(resolver).Compile(ctx, doc)
				require.NoError(t, err)

				source, err := helium.NewParser().Parse(ctx, []byte(`<doc><item>BUILTIN</item></doc>`))
				require.NoError(t, err)

				result, err := ss.Transform(source).Serialize(ctx)
				require.NoError(t, err)

				if flag.flagPresent {
					require.Contains(t, result, "<out>MODULE</out>",
						"flag present: module's root use-when must resolve against effective base %q and include the module", flagURI)
					require.True(t, resolver.askedFor(flagURI),
						"root use-when doc-available must probe the effective-base URI %q; asked=%v", flagURI, resolver.asked)
				} else {
					require.NotContains(t, result, "<out>MODULE</out>",
						"flag absent: module's root use-when must evaluate false and exclude the module")
					require.Contains(t, result, "BUILTIN",
						"flag absent: with the module excluded the built-in template emits the item text")
				}
			})
		}
	}
}
