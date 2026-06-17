package xslt3_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestPrimaryResultDocumentUnlinksChildren guards against a tree-corruption bug
// where a primary xsl:result-document (validation="strict") moved children from
// a temporary document into the primary document via AddChild without unlinking
// them first. The temporary document's sibling links remained attached, which
// corrupted the result tree (and could hang sibling traversal) when the body
// produced a comment followed by the root element.
func TestPrimaryResultDocumentUnlinksChildren(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="" validation="strict">
      <xsl:comment>lead</xsl:comment>
      <out>body</out>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)

	done := make(chan struct{})
	var result string
	var err error
	go func() {
		result, err = ss.Transform(source).Serialize(t.Context())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Serialize hung: primary result-document corrupted the sibling chain")
	}

	require.NoError(t, err)
	require.Contains(t, result, "<!--lead-->")
	require.Contains(t, result, "<out>body</out>")
	require.Less(t, strings.Index(result, "<!--lead-->"), strings.Index(result, "<out>body</out>"),
		"comment must precede the root element")
}
