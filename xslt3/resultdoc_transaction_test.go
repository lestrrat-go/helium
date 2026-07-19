package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// Finding 1 (PR #649 round 4): the buffered primary direct-write path must
// preserve the REAL base frame's capture state. When the default output method
// is json/adaptive (an item-serialization method) the base frame has
// captureItems=true, so atomic values produced by xsl:sequence inside a primary
// xsl:result-document MUST be preserved as separate XDM items rather than
// stringified into the DOM as text. With the regression, the buffer frame was
// created without captureItems/sequenceMode, so the atomics were written as a
// single merged text node instead of three integer items.
func TestResultDocumentPrimaryAdaptivePreservesAtomicItems(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive"/>
  <xsl:template match="/">
    <xsl:result-document>
      <xsl:sequence select="(1, 2, 3)"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	capture := &primaryItemsCapture{}
	_, err := ss.Transform(parseTransformSource(t)).
		PrimaryItemsHandler(capture).
		Do(t.Context())
	require.NoError(t, err)

	require.NotNil(t, capture.seq, "expected primary items to be captured")
	require.Equal(t, 3, capture.seq.Len(),
		"three atomic values must be preserved as three XDM items, not stringified into a single text node")
	for i, want := range []string{"1", "2", "3"} {
		av, ok := capture.seq.Get(i).(xpath3.AtomicValue)
		require.True(t, ok, "captured item %d must be an atomic value, not a node/text item", i)
		s, sErr := xpath3.AtomicToString(av)
		require.NoError(t, sErr)
		require.Equal(t, want, s, "captured atomic %d must retain its integer value", i)
	}
}

// Finding 2 (PR #649 round 4): per-href result-document state is a transaction
// with a single commit point. The failed attempt below uses method="adaptive"
// (an item-serialization method), so its body populates resultDocItems[href]
// BEFORE post-body validation. validation="strict" over a two-root document then
// fails (XTTE1550 from validateDocumentStructure). Pre-fix, resultDocItems and
// resultDocOutputDefs were mutated before the validation step and before
// committed=true, so the caught failure left stale entries. The xsl:catch then
// writes the SAME href with a plain XML body (no items): the stale json items
// from the rolled-back attempt would be serialized into the catch's document by
// the end-of-transform materialization loop, contaminating it. The transaction
// fix stages all per-href state and publishes it only at the commit point, so
// the catch's <good/> is the SOLE content for that href.
func TestResultDocumentSecondaryValidationFailRollbackTryCatch(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output name="json" method="adaptive"/>
  <xsl:template match="/">
    <xsl:try>
      <xsl:result-document href="out.xml" format="json" validation="strict">
        <bad/>
        <alsobad/>
      </xsl:result-document>
      <xsl:catch>
        <xsl:result-document href="out.xml"><good/></xsl:result-document>
      </xsl:catch>
    </xsl:try>
  </xsl:template>
</xsl:stylesheet>`)

	collector := &resultDocCollect{docs: map[string]*helium.Document{}}
	_, err := ss.Transform(parseTransformSource(t)).
		ResultDocumentHandler(collector).
		Do(t.Context())
	require.NoError(t, err,
		"the caught secondary result-document must release its URI so the catch reuses the same href")

	doc, ok := collector.docs["out.xml"]
	require.True(t, ok, "the catch must have delivered a result document for out.xml")
	root := findResultRoot(doc)
	require.NotNil(t, root, "delivered result document must have a root element")
	require.Equal(t, "good", root.Name(),
		"only the catch's <good/> may be delivered; the rolled-back attempt must leave no stale state")

	// The stale json items from the failed attempt must NOT have been serialized
	// into the catch's document: it must contain exactly one child, the <good/>
	// element, with no leftover text node from the rolled-back <bad/>/<alsobad/>.
	var childCount int
	for child := range helium.Children(doc) {
		childCount++
		require.Equal(t, helium.ElementNode, child.Type(),
			"no stale serialized text node may be appended to the catch's document")
	}
	require.Equal(t, 1, childCount, "the catch's document must contain only <good/>")
}

// A secondary xsl:result-document with an item-serialization method (adaptive)
// stages its element items into resultDocItems and serializes them in the
// end-of-transform materialization loop. When that serialization fails — here a
// two-element sequence whose text carries U+0001, an XML-invalid character — the
// error MUST surface from the transform rather than being swallowed into a
// silently empty secondary document.
func TestResultDocumentSecondarySerializationErrorSurfaces(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="e" as="element()">
      <r><xsl:value-of select="codepoints-to-string(1)"/></r>
    </xsl:variable>
    <xsl:result-document href="out.txt" method="adaptive">
      <xsl:sequence select="($e, $e)"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	collector := &resultDocCollect{docs: map[string]*helium.Document{}}
	_, err := ss.Transform(parseTransformSource(t)).
		ResultDocumentHandler(collector).
		Do(t.Context())
	requireSERE0006(t, err)
}

type resultDocCollect struct {
	docs map[string]*helium.Document
}

func (c *resultDocCollect) HandleResultDocument(href string, doc *helium.Document, _ *xslt3.OutputDef) error {
	c.docs[href] = doc
	return nil
}

func findResultRoot(doc *helium.Document) helium.Node {
	for child := range helium.Children(doc) {
		if child.Type() == helium.ElementNode {
			return child
		}
	}
	return nil
}
