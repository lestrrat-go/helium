package xslt3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEvaluateWithParamsQNameNamespacedKey verifies that xsl:evaluate's
// with-params map stores QName keys under their full Clark-name
// representation, so a namespaced key QName('urn:p','x') is resolvable as
// $p:x inside the dynamically-evaluated expression (with p->urn:p in scope)
// and does not collide with a no-namespace variable of the same local name.
//
// The xpath attribute is itself an XPath expression whose string value is the
// dynamically-evaluated expression, hence the string literal '$p:x'.
func TestEvaluateWithParamsQNameNamespacedKey(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:p="urn:p">
  <xsl:template match="/">
    <out><xsl:evaluate xpath="'$p:x'" with-params="map{ QName('urn:p','x') : 42 }"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, ">42</out>")
}

// TestEvaluateWithParamsQNameNoCollision verifies that a namespaced QName key
// and a no-namespace key with the same local name do not collide: $x resolves
// to the no-namespace value, $p:x to the namespaced value.
func TestEvaluateWithParamsQNameNoCollision(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:p="urn:p">
  <xsl:template match="/">
    <out><xsl:evaluate xpath="'concat($x, &quot;|&quot;, $p:x)'"
        with-params="map{ QName('','x') : 'plain', QName('urn:p','x') : 'namespaced' }"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, ">plain|namespaced</out>")
}
