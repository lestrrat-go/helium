package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestNamespace2614ExcludedLiteralPrefixCanBeRebound covers W3C
// namespace-2614. An excluded literal result element prefix is available to
// xsl:namespace, and namespace fixup gives the element's original namespace a
// generated prefix.
func TestNamespace2614ExcludedLiteralPrefixCanBeRebound(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="2.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <p:item xmlns:p="http://p.uri/" xsl:exclude-result-prefixes="p">
      <xsl:namespace name="p">http://q.uri/</xsl:namespace>
    </p:item>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := xslt3.TransformString(t.Context(), parseTransformSource(t), ss)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(out))
	require.NoError(t, err)
	elem := doc.DocumentElement()
	require.Equal(t, "item", elem.LocalName())
	require.Equal(t, "http://p.uri/", elem.URI())
	require.Equal(t, "p_0", elem.Prefix())
	require.Equal(t, "http://q.uri/", namespaceURI(elem, "p"))
	require.Equal(t, "http://p.uri/", namespaceURI(elem, elem.Prefix()))
}

// TestNamespaceRebindOnNonExcludedLiteralResultElementRaisesXTDE0430 keeps
// the normal conflict rule for a literal result element whose prefix was not
// excluded from the result tree.
func TestNamespaceRebindOnNonExcludedLiteralResultElementRaisesXTDE0430(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <p:item xmlns:p="http://p.uri/">
      <xsl:namespace name="p">http://q.uri/</xsl:namespace>
    </p:item>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := xslt3.TransformString(t.Context(), parseTransformSource(t), ss)
	require.ErrorContains(t, err, "XTDE0430")
}

func namespaceURI(elem *helium.Element, prefix string) string {
	for _, ns := range elem.Namespaces() {
		if ns.Prefix() == prefix {
			return ns.URI()
		}
	}
	return ""
}
