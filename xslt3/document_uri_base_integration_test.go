package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// docURIBaseStylesheet calls doc()/document() with a relative href so the
// runtime must resolve it against the stylesheet's base URI.
const docURIBaseStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <a><xsl:value-of select="doc('doc.xml')/data/@v"/></a>
      <b><xsl:value-of select="document('doc.xml')/data/@v"/></b>
    </out>
  </xsl:template>
</xsl:stylesheet>`

// TestDocumentResolutionPreservesURIBase is an end-to-end regression for the
// base-URI corruption fixed in this change. When the stylesheet is compiled
// with Compiler.BaseURI("mem://pkg/main.xsl"), doc('doc.xml') and
// document('doc.xml') must reach the runtime URIResolver with the URI
// "mem://pkg/doc.xml" — i.e. the sibling reference resolved against the FULL
// URI base preserving scheme+authority.
//
// Before the fix, ec.baseDir() ran filepath.Dir over the URI base first,
// collapsing "mem://pkg/main.xsl" to "mem:/pkg", so the resolver was instead
// asked for "mem:/pkg/doc.xml" (host dropped). This test fails on that path.
func TestDocumentResolutionPreservesURIBase(t *testing.T) {
	const wantURI = "mem://pkg/doc.xml"

	resolver := &recordingURIResolver{files: map[string][]byte{
		wantURI: []byte(`<data v="hello"/>`),
	}}

	doc, err := helium.NewParser().Parse(t.Context(), []byte(docURIBaseStylesheet))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().BaseURI("mem://pkg/main.xsl").Compile(t.Context(), doc)
	require.NoError(t, err)

	source := parseTransformSource(t)
	result, err := ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
	require.NoError(t, err)

	require.True(t, resolver.seen(wantURI),
		"runtime resolver must receive %q; got %v", wantURI, resolver.requests)
	// Confirm the host was not dropped (the pre-fix bug produced mem:/pkg/...).
	require.False(t, resolver.seen("mem:/pkg/doc.xml"),
		"resolver must not receive the host-collapsed URI; got %v", resolver.requests)

	require.Contains(t, result, "<a>hello</a>")
	require.Contains(t, result, "<b>hello</b>")
}

// TestDocumentResolutionLocalBaseUnchanged guards against a regression in
// local-filesystem document resolution: a relative doc()/document() href under
// a plain local base must still resolve against the containing directory via
// filepath, NOT be treated as a URI.
func TestDocumentResolutionLocalBaseUnchanged(t *testing.T) {
	const wantURI = "/styles/doc.xml"

	resolver := &recordingURIResolver{files: map[string][]byte{
		wantURI: []byte(`<data v="local"/>`),
	}}

	doc, err := helium.NewParser().Parse(t.Context(), []byte(docURIBaseStylesheet))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().BaseURI("/styles/main.xsl").Compile(t.Context(), doc)
	require.NoError(t, err)

	source := parseTransformSource(t)
	result, err := ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
	require.NoError(t, err)

	require.True(t, resolver.seen(wantURI),
		"runtime resolver must receive %q; got %v", wantURI, resolver.requests)
	require.Contains(t, result, "<a>local</a>")
}
