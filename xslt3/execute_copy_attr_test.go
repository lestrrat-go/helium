package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestCopyOfAttributeValuesNotReparsed guards against the result-tree copy path
// re-parsing an already-resolved attribute value. xsl:copy-of of an element (and
// xsl:copy) duplicates the source element into the result tree via a deep copy;
// the attribute value returned by the parser is already entity-resolved, so it
// must be stored LITERALLY (and re-escaped by the serializer) rather than fed
// back through the value-parsing setter, which would choke on a bare '&' (an
// "entity was unterminated" error) or silently double-resolve '&amp;amp;'.
func TestCopyOfAttributeValuesNotReparsed(t *testing.T) {
	t.Parallel()

	// srcAttr is the lexical attribute value as authored in the source XML;
	// wantAttr is its expected serialization after an entity-resolved round trip.
	cases := []struct {
		name     string
		srcAttr  string
		wantAttr string
	}{
		{"ampersand", "x&amp;y", "x&amp;y"},
		{"less-than", "a&lt;b", "a&lt;b"},
		{"greater-than", "a&gt;b", "a&gt;b"},
		{"quote", "&quot;q&quot;", "&quot;q&quot;"},
		{"numeric-ref", "&#65;&#66;", "AB"},
		{"double-escaped", "&amp;amp;", "&amp;amp;"},
		{"mixed", "p?a=1&amp;b=2&lt;3", "p?a=1&amp;b=2&lt;3"},
	}

	copyOfSheet := compileSheet(t, `<out><xsl:copy-of select="."/></out>`)
	copySheet := compileSheet(t, `<xsl:copy select="."><xsl:copy-of select="@*"/></xsl:copy>`)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<e a="`+tc.srcAttr+`"/>`))
			require.NoError(t, err)

			wantElem := `<e a="` + tc.wantAttr + `"/>`

			// xsl:copy-of of the element: the previously-broken path.
			out, err := xslt3.TransformString(t.Context(), src, copyOfSheet)
			require.NoError(t, err, "xsl:copy-of must not re-parse the resolved value")
			require.Equal(t, "<out>"+wantElem+"</out>\n", out)

			// xsl:copy of the element plus xsl:copy-of of its attributes.
			out2, err := xslt3.TransformString(t.Context(), src, copySheet)
			require.NoError(t, err, "xsl:copy must not re-parse the resolved value")
			require.Equal(t, wantElem+"\n", out2)
		})
	}
}

// TestCopyOfNamespacedAttributeNotReparsed exercises the namespaced-attribute
// branch of the deep-copy attribute loop (SetLiteralAttributeNS).
func TestCopyOfNamespacedAttributeNotReparsed(t *testing.T) {
	t.Parallel()

	src, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<e xmlns:p="urn:p" p:a="x&amp;y&lt;z"/>`))
	require.NoError(t, err)

	ss := compileSheet(t, `<out><xsl:copy-of select="."/></out>`)
	out, err := xslt3.TransformString(t.Context(), src, ss)
	require.NoError(t, err)
	require.Equal(t, `<out><e xmlns:p="urn:p" p:a="x&amp;y&lt;z"/></out>`+"\n", out)
}

// compileSheet compiles a minimal stylesheet whose single template body (matched
// on the /e source element) is the provided sequence constructor.
func compileSheet(t *testing.T, body string) *xslt3.Stylesheet {
	t.Helper()
	sheet := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:output method="xml" omit-xml-declaration="yes"/>
<xsl:template match="/e">` + body + `</xsl:template>
</xsl:stylesheet>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(sheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)
	return ss
}
