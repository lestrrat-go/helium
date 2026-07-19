package xslt3_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestSerializeResultXML(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><root>hello</root></xsl:template>
</xsl:stylesheet>`)

	doc, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.NoError(t, err)

	var buf bytes.Buffer
	err = xslt3.SerializeResult(&buf, doc, ss.DefaultOutputDef())
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<root>hello</root>")
}

func TestSerializeResultNilOutputDef(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><root>hello</root></xsl:template>
</xsl:stylesheet>`)

	doc, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.NoError(t, err)

	// nil OutputDef should use defaults.
	var buf bytes.Buffer
	err = xslt3.SerializeResult(&buf, doc, nil)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<root>hello</root>")
}

func TestSerializeResultText(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/">hello world</xsl:template>
</xsl:stylesheet>`)

	doc, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.NoError(t, err)

	var buf bytes.Buffer
	err = xslt3.SerializeResult(&buf, doc, ss.DefaultOutputDef())
	require.NoError(t, err)
	require.Equal(t, "hello world", strings.TrimSpace(buf.String()))
}

func TestSerializeItemsAtomics(t *testing.T) {
	items := xpath3.ItemSlice{
		xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "alpha"},
		xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "bravo"},
	}

	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, items, nil, nil)
	require.NoError(t, err)
	result := buf.String()
	require.Contains(t, result, "alpha")
	require.Contains(t, result, "bravo")
}

func TestSerializeItemsWithDocument(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<data>content</data>`))
	require.NoError(t, err)

	var buf bytes.Buffer
	err = xslt3.SerializeItems(&buf, nil, doc, nil)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "content")
}

func TestDefaultOutputDef(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text" encoding="UTF-8"/>
  <xsl:template match="/">hello</xsl:template>
</xsl:stylesheet>`)

	outDef := ss.DefaultOutputDef()
	require.NotNil(t, outDef)
}

func TestDefaultOutputDefNilStylesheet(t *testing.T) {
	var ss *xslt3.Stylesheet
	outDef := ss.DefaultOutputDef()
	require.Nil(t, outDef)
}

// outMethodXML is the "xml" output method, held as a const so these invalid-char
// tests do not add repeated string literals (goconst).
const outMethodXML = "xml"

// newBadCharElement builds a small <r> element whose text content carries an
// XML-invalid control character (U+0001), via the public DOM API. The DOM
// accepts the control byte; the writer is the enforcement point.
func newBadCharElement(t *testing.T) *helium.Element {
	t.Helper()
	doc := helium.NewDefaultDocument()
	root, err := doc.CreateElement("r")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))
	require.NoError(t, root.AddChild(doc.CreateText([]byte("a\x01b"))))
	return root
}

// requireSERE0006 asserts err is the XSLT serialization error SERE0006 that the
// serializer raises when the writer rejects an XML-invalid character.
func requireSERE0006(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var xe *xslt3.XSLTError
	require.ErrorAs(t, err, &xe)
	require.Equal(t, "SERE0006", xe.Code)
}

// SerializeItems with method="xml" must propagate the writer's invalid-char
// rejection as SERE0006 rather than silently truncating the output.
func TestSerializeItemsXMLInvalidChar(t *testing.T) {
	root := newBadCharElement(t)
	items := xpath3.ItemSlice{xpath3.NodeItem{Node: root}}
	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, items, nil, &xslt3.OutputDef{Method: outMethodXML})
	requireSERE0006(t, err)
}

// SerializeItems with method="json" and json-node-output-method="xml" must
// propagate the writer's invalid-char rejection as SERE0006.
func TestSerializeItemsJSONNodeXMLInvalidChar(t *testing.T) {
	root := newBadCharElement(t)
	items := xpath3.ItemSlice{xpath3.NodeItem{Node: root}}
	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, items, nil, &xslt3.OutputDef{Method: "json", JSONNodeOutputMethod: outMethodXML})
	requireSERE0006(t, err)
}

// SerializeItems with method="adaptive" over a multi-item sequence containing a
// node with an invalid character must propagate SERE0006 (the single-element
// path already propagates via serializeXML; this exercises the per-item path).
func TestSerializeItemsAdaptiveInvalidChar(t *testing.T) {
	root := newBadCharElement(t)
	items := xpath3.ItemSlice{xpath3.NodeItem{Node: root}, xpath3.NodeItem{Node: root}}
	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, items, nil, &xslt3.OutputDef{Method: "adaptive"})
	requireSERE0006(t, err)
}
