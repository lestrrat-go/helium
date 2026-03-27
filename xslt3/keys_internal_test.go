package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestCanonicalKeyUsesQNameValueSpace verifies that xsl:key treats QName
// values with the same namespace URI and local name as identical keys,
// regardless of the prefix used.  Two elements declare the same QName key
// value under different prefixes; a single key() lookup must return both.
func TestCanonicalKeyUsesQNameValueSpace(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:key name="byType" match="item" use="resolve-QName(@type, .)"/>
  <xsl:template match="/">
    <result>
      <count><xsl:value-of select="count(key('byType', resolve-QName('one:mp3', /root/item[1])))"/></count>
    </result>
  </xsl:template>
</xsl:stylesheet>`)

	source, err := helium.NewParser().Parse(t.Context(), []byte(
		`<root>` +
			`<item xmlns:one="urn:test" type="one:mp3"/>` +
			`<item xmlns:two="urn:test" type="two:mp3"/>` +
			`</root>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)

	// Both items use the same QName (urn:test, mp3), so key() must return 2.
	require.Contains(t, result, "<count>2</count>")
}
