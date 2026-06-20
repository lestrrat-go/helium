package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// countWhitespaceTextNodes returns the number of whitespace-only text nodes
// anywhere in the tree rooted at n.
func countWhitespaceTextNodes(n helium.Node) int {
	count := 0
	for child := range helium.Children(n) {
		if child.Type() == helium.TextNode {
			if strings.TrimSpace(string(child.Content())) == "" {
				count++
			}
		}
		count += countWhitespaceTextNodes(child)
	}
	return count
}

// TestStripSpaceDoesNotMutateSource verifies that running a transform whose
// stylesheet declares xsl:strip-space does NOT mutate the caller-owned source
// document. Whitespace-only text nodes must be stripped on a private copy used
// only inside the transform; the original tree the caller passed in must be left
// untouched so it can be reused (e.g. for a subsequent XPath query or a second
// transform). See finding A-004.
func TestStripSpaceDoesNotMutateSource(t *testing.T) {
	t.Parallel()

	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, ss)

	source, err := helium.NewParser().Parse(t.Context(),
		[]byte("<doc>\n  <item>x</item>\n</doc>"))
	require.NoError(t, err)

	before := countWhitespaceTextNodes(source)
	require.Positive(t, before,
		"test fixture must contain whitespace-only text nodes to be meaningful")

	out, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)

	// The transform output must reflect strip-space: the whitespace-only text
	// nodes between elements are gone in the result.
	require.NotContains(t, out, "<doc>\n",
		"strip-space must remove whitespace-only text nodes from the transform result; got %q", out)
	require.Contains(t, out, "<item>x</item>",
		"non-whitespace content must survive; got %q", out)

	// The caller-owned source DOM must be untouched: its whitespace-only text
	// nodes are still present after the transform.
	after := countWhitespaceTextNodes(source)
	require.Equal(t, before, after,
		"xsl:strip-space must not mutate the caller's source document (had %d whitespace text nodes, now %d)", before, after)

	// A second transform of the same source must still see the whitespace, i.e.
	// produce the same stripped output (it would differ if the first run had
	// destructively stripped the shared tree).
	out2, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Equal(t, out, out2,
		"repeated transforms of the same reused source must produce identical output")
}
