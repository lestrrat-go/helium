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

// TestEmbeddedModuleRootUseWhenResolvesAgainstEmbeddedBase is a regression for a
// root use-when on an EMBEDDED stylesheet module (one selected by a fragment
// identifier, whose root element's parent is another element, not the Document).
//
// moduleEffectiveBaseURI deliberately returns the bare module URI for an
// embedded root (its xml:base is normally re-applied by the later
// stylesheetBaseURI ancestor walk during child compilation). But the ROOT
// use-when has no such later walk, so its doc-available()/doc() must apply the
// embedded root's own xml:base here. The embedded stylesheet carries
// xml:base="sub/" and use-when="doc-available('flag.xml')"; only
// mem://pkg/sub/flag.xml exists. With the correct embedded base the reference
// resolves there and the module is included; with the bare module URI it
// resolves to mem://pkg/flag.xml, which is absent, wrongly excluding the module.
func TestEmbeddedModuleRootUseWhenResolvesAgainstEmbeddedBase(t *testing.T) {
	const (
		mainBase  = "mem://pkg/main.xsl"
		moduleURI = "mem://pkg/emb.xsl"
		flagURI   = "mem://pkg/sub/flag.xml" // embedded root xml:base="sub/" folded onto module URI
		module    = `<?xml version="1.0"?>
<wrapper xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:stylesheet id="style1" version="3.0"
    xml:base="sub/"
    use-when="doc-available('flag.xml')">
    <xsl:template match="item"><out>EMBEDDED</out></xsl:template>
  </xsl:stylesheet>
</wrapper>`
	)

	for _, link := range []struct {
		name    string
		linkElt string
	}{
		{"import", `<xsl:import href="emb.xsl#style1"/>`},
		{"include", `<xsl:include href="emb.xsl#style1"/>`},
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

			source, err := helium.NewParser().Parse(ctx, []byte(`<doc><item>BUILTIN</item></doc>`))
			require.NoError(t, err)

			result, err := ss.Transform(source).Serialize(ctx)
			require.NoError(t, err)

			require.Contains(t, result, "<out>EMBEDDED</out>",
				"embedded module's root use-when must resolve against the embedded root's xml:base (effective %q) and include the module", flagURI)
			require.True(t, resolver.askedFor(flagURI),
				"root use-when doc-available must probe the embedded-base URI %q; asked=%v", flagURI, resolver.asked)
		})
	}
}

// TestUseWhenDocAvailableHonorsMaxResourceBytes is a regression for compile-time
// use-when reads ignoring Compiler.MaxResourceBytes. The compile-time use-when
// evaluator routes doc()/doc-available() through the compiler URIResolver; it
// must also enforce the compiler's per-resource read cap, matching the runtime
// evaluator. With a 1-byte cap, doc-available() on an over-cap resource must be
// false; with the default cap the small resource loads.
func TestUseWhenDocAvailableHonorsMaxResourceBytes(t *testing.T) {
	const (
		mainBase = "mem://pkg/main.xsl"
		flagURI  = "mem://pkg/flag.xml"
	)
	main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/" use-when="doc-available('flag.xml')"><result>WITH-FLAG</result></xsl:template>
  <xsl:template match="/" use-when="not(doc-available('flag.xml'))"><result>NO-FLAG</result></xsl:template>
</xsl:stylesheet>`

	for _, tc := range []struct {
		name   string
		cap    int64
		capSet bool
		want   string
	}{
		{"default_cap_flag_loads", 0, false, "WITH-FLAG"},
		{"tiny_cap_flag_over_limit", 1, true, "NO-FLAG"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			// flag.xml is comfortably larger than the 1-byte cap.
			resolver := &exactURIResolver{files: map[string]string{flagURI: `<flag>data</flag>`}}

			doc, err := helium.NewParser().Parse(ctx, []byte(main))
			require.NoError(t, err)

			compiler := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(resolver)
			if tc.capSet {
				compiler = compiler.MaxResourceBytes(tc.cap)
			}
			ss, err := compiler.Compile(ctx, doc)
			require.NoError(t, err)

			source, err := helium.NewParser().Parse(ctx, []byte(`<doc/>`))
			require.NoError(t, err)

			result, err := ss.Transform(source).Serialize(ctx)
			require.NoError(t, err)
			require.Contains(t, result, tc.want)
		})
	}
}
