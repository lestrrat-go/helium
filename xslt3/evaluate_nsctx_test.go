package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// XSLT3-ADV-004: xsl:evaluate's namespace-context attribute must produce a
// single node. A non-node value (or an empty sequence) is a type error
// XTTE3170 rather than being silently ignored.
func TestEvaluateNamespaceContextTypeError(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	for _, tc := range []struct {
		name         string
		namespaceCtx string
	}{
		{name: "non-node string", namespaceCtx: "'not-a-node'"},
		{name: "non-node number", namespaceCtx: "42"},
		{name: "empty sequence", namespaceCtx: "()"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:evaluate xpath="'hello'" namespace-context="`+tc.namespaceCtx+`"/>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

			_, err := ss.Transform(doc).Serialize(t.Context())
			require.Error(t, err)
			require.ErrorContains(t, err, "XTTE3170")
		})
	}
}
