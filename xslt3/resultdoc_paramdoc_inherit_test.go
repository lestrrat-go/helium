package xslt3_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A primary xsl:result-document whose parameter-document OMITS a serialization
// parameter must inherit that parameter's value from the unnamed default
// xsl:output, not silently reset it to the Go zero value. Before the fix
// evalResultDocOutputDef took the parameter-document OutputDef as the WHOLE base
// when a parameter-document was present, dropping the default xsl:output: an
// omitted plain boolean (indent, byte-order-mark, allow-duplicate-names,
// undeclare-prefixes) then overwrote an inherited true with false.
func TestResultDocParamDocInheritsDefaultOutput(t *testing.T) {
	const paramDocURI = "http://example.invalid/params.xml"

	// Serialization parameter document that deliberately OMITS indent and
	// allow-duplicate-names; it only sets an unrelated parameter (encoding).
	const paramDoc = `<serialization-parameters xmlns="http://www.w3.org/2010/xslt-xquery-serialization">` +
		`<encoding value="utf-8"/>` +
		`</serialization-parameters>`

	resolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != paramDocURI {
			return nil, errors.New("not found: " + uri)
		}
		return io.NopCloser(strings.NewReader(paramDoc)), nil
	})

	t.Run("indent inherited", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output indent="yes"/>
  <xsl:template match="/">
    <xsl:result-document parameter-document="{'`+paramDocURI+`'}">
      <root><child>x</child></root>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

		out, err := ss.Transform(parseTransformSource(t)).
			URIResolver(resolver).
			Serialize(t.Context())
		require.NoError(t, err)
		// indent="yes" inherited from the default xsl:output must still pretty-print
		// the nested element across lines ("<root>\n  <child>..."); a clobbered
		// indent=false would emit "<root><child>x</child></root>" on one line.
		require.Contains(t, out, "<root>\n", "inherited indent=yes must survive a parameter-document that omits indent")
	})

	t.Run("allow-duplicate-names inherited", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="json" allow-duplicate-names="yes"/>
  <xsl:template match="/">
    <xsl:result-document parameter-document="{'`+paramDocURI+`'}">
      <xsl:sequence select="map{1:'a','1':'b'}"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

		// The map's integer key 1 and string key "1" both become the JSON name
		// "1": a duplicate name that SERE0022 rejects unless allow-duplicate-names
		// ="yes". That value is set only on the default xsl:output and OMITTED by
		// the parameter-document, so it must be inherited rather than reset to false.
		inv := ss.Transform(parseTransformSource(t)).URIResolver(resolver)
		_, err := inv.Do(t.Context())
		require.NoError(t, err, "inherited allow-duplicate-names=yes must survive a parameter-document that omits it (no SERE0022)")
		od := inv.ResolvedOutputDef()
		require.NotNil(t, od)
		require.True(t, od.AllowDuplicateNames,
			"the default xsl:output allow-duplicate-names=yes must survive a parameter-document that omits it")
	})
}
