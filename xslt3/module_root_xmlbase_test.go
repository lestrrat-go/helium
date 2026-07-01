package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestModuleRootXMLBaseResolvesGlobals is an end-to-end regression for an
// included/imported stylesheet module whose ROOT element carries xml:base.
//
// The main module gets root xml:base folded into its effective static base in
// compile(), but an external module is loaded with c.baseURI set to the bare
// module URI and compiled directly (loadExternalStylesheet → compileTopLevel
// for xsl:import; the two-phase collect/compile path for xsl:include). Before
// the fix, the module root's xml:base was silently dropped, so a global
// variable in that module resolved doc()/document() against the bare module URI
// ("mem://pkg/data.xml") instead of the declaration-site effective base under
// xml:base="sub/" ("mem://pkg/sub/data.xml").
func TestModuleRootXMLBaseResolvesGlobals(t *testing.T) {
	const (
		mainBase   = "mem://pkg/main.xsl"
		moduleURI  = "mem://pkg/inc.xsl"
		wantURI    = "mem://pkg/sub/data.xml" // effective base (module URI + xml:base="sub/")
		preFixURI  = "mem://pkg/data.xml"     // bare module URI (the dropped-xml:base bug)
		moduleTmpl = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xml:base="sub/">
  <xsl:variable name="g" select="doc('data.xml')/data/@v"/>
</xsl:stylesheet>`
	)

	for _, tc := range []struct {
		name    string
		linkElt string
	}{
		{"import", `<xsl:import href="inc.xsl"/>`},
		{"include", `<xsl:include href="inc.xsl"/>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  ` + tc.linkElt + `
  <xsl:template match="/"><out><xsl:value-of select="$g"/></out></xsl:template>
</xsl:stylesheet>`

			compileResolver := fileMapResolver{files: map[string]string{moduleURI: moduleTmpl}}
			doc, err := helium.NewParser().Parse(ctx, []byte(main))
			require.NoError(t, err)

			ss, err := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(compileResolver).Compile(ctx, doc)
			require.NoError(t, err)

			runtimeResolver := &recordingURIResolver{files: map[string][]byte{
				wantURI: []byte(`<data v="DEEP"/>`),
			}}
			source := parseTransformSource(t)
			result, err := ss.Transform(source).URIResolver(runtimeResolver).Serialize(ctx)
			require.NoError(t, err)

			require.True(t, runtimeResolver.seen(wantURI),
				"global doc() must resolve against the module's effective base %q; got %v", wantURI, runtimeResolver.requests)
			require.False(t, runtimeResolver.seen(preFixURI),
				"global doc() must NOT resolve against the bare module URI %q (dropped xml:base); got %v", preFixURI, runtimeResolver.requests)
			require.Contains(t, result, "<out>DEEP</out>")
		})
	}
}
