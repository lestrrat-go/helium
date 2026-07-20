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

const adaptiveMethod = "adaptive"

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

func TestSerializeItemsNormalizationWithCharacterMap(t *testing.T) {
	decomposed := "e\u0301"
	composed := "é"
	replacement := "a\u030a"
	value := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "x" + decomposed}
	mapItem := xpath3.NewMap([]xpath3.MapEntry{{
		Key:   xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "key"},
		Value: xpath3.ItemSlice{value},
	}})
	arrayItem := xpath3.NewArray([]xpath3.Sequence{xpath3.ItemSlice{value}})

	tests := []struct {
		name   string
		method string
		item   xpath3.Item
		want   string
	}{
		{
			name:   "JSONAtomic",
			method: "json",
			item:   value,
			want:   `"` + replacement + composed + `"`,
		},
		{
			name:   "JSONMap",
			method: "json",
			item:   mapItem,
			want:   `{"key":"` + replacement + composed + `"}`,
		},
		{
			name:   "JSONArray",
			method: "json",
			item:   arrayItem,
			want:   `["` + replacement + composed + `"]`,
		},
		{
			name:   "AdaptiveAtomic",
			method: adaptiveMethod,
			item:   value,
			want:   `"` + replacement + composed + `"`,
		},
		{
			name:   "AdaptiveMap",
			method: adaptiveMethod,
			item:   mapItem,
			want:   `map{"key":"` + replacement + composed + `"}`,
		},
		{
			name:   "AdaptiveArray",
			method: adaptiveMethod,
			item:   arrayItem,
			want:   `["` + replacement + composed + `"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := xslt3.SerializeItems(&buf, xpath3.ItemSlice{tt.item}, nil, &xslt3.OutputDef{
				Method:            tt.method,
				NormalizationForm: normalizationFormNFC,
				ResolvedCharMap:   map[rune]string{'x': replacement},
			})
			require.NoError(t, err)
			require.Equal(t, tt.want, buf.String())
		})
	}
}

func TestSerializeItemsAdaptiveMapKeyNormalizationWithCharacterMap(t *testing.T) {
	decomposed := "e\u0301"
	composed := "é"
	replacement := "\"a\u030a"
	key := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "x" + decomposed}
	inner := xpath3.NewMap([]xpath3.MapEntry{{
		Key:   key,
		Value: xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "value"}},
	}})
	outer := xpath3.NewMap([]xpath3.MapEntry{{
		Key:   key,
		Value: xpath3.ItemSlice{inner},
	}})

	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, xpath3.ItemSlice{outer}, nil, &xslt3.OutputDef{
		Method:            adaptiveMethod,
		NormalizationForm: normalizationFormNFC,
		ResolvedCharMap:   map[rune]string{'x': replacement},
	})
	require.NoError(t, err)
	escapedKey := `\"å` + composed
	require.Equal(t, `map{"`+escapedKey+`":map{"`+escapedKey+`":"value"}}`, buf.String())
}

func TestSerializeItemsAdaptiveStringNormalization(t *testing.T) {
	decomposed := "e\u0301"
	composed := "é"
	item := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: decomposed}

	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, xpath3.ItemSlice{item}, nil, &xslt3.OutputDef{
		Method:            adaptiveMethod,
		NormalizationForm: normalizationFormNFC,
	})
	require.NoError(t, err)
	require.Equal(t, `"`+composed+`"`, buf.String())
}

func TestSerializeItemsAdaptiveSingletonElementNormalization(t *testing.T) {
	decomposed := "e\u0301"
	composed := "é"
	tests := []struct {
		name    string
		charMap map[rune]string
	}{
		{name: "NoCharacterMap"},
		{name: "UnrelatedCharacterMap", charMap: map[rune]string{'x': "unused"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			elem, err := doc.CreateElement("out")
			require.NoError(t, err)
			require.NoError(t, elem.AddChild(doc.CreateText([]byte(decomposed))))

			var buf bytes.Buffer
			err = xslt3.SerializeItems(&buf, xpath3.ItemSlice{xpath3.NodeItem{Node: elem}}, nil, &xslt3.OutputDef{
				Method:            adaptiveMethod,
				NormalizationForm: normalizationFormNFC,
				ResolvedCharMap:   tt.charMap,
			})
			require.NoError(t, err)
			require.Contains(t, buf.String(), "<out>"+composed+"</out>")
			require.NotContains(t, buf.String(), decomposed)
		})
	}
}

func TestSerializeItemsAdaptiveFullyNormalizedNodeCharacterMap(t *testing.T) {
	decomposed := "e\u0301"
	mappedNFC := "mapped&#xE9;"
	doc, err := helium.NewParser().Parse(t.Context(), []byte("<out>x"+decomposed+"</out>"))
	require.NoError(t, err)

	root := doc.DocumentElement()
	tests := []struct {
		name  string
		items xpath3.Sequence
		want  string
	}{
		{
			name:  "Document",
			items: xpath3.ItemSlice{xpath3.NodeItem{Node: doc}},
			want:  "<out>" + mappedNFC + "</out>\n",
		},
		{
			name: "MultiItemElement",
			items: xpath3.ItemSlice{
				xpath3.NodeItem{Node: root},
				xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "tail"},
			},
			want: "<out>" + mappedNFC + "</out>\n\"tail\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := xslt3.SerializeItems(&buf, tt.items, nil, &xslt3.OutputDef{
				Method:            adaptiveMethod,
				NormalizationForm: "FULLY-NORMALIZED",
				ResolvedCharMap:   map[rune]string{'x': "mapped"},
			})
			require.NoError(t, err)
			require.Equal(t, tt.want, buf.String())
		})
	}
}

func TestSerializeItemsAdaptiveNodeCharacterDataTransformations(t *testing.T) {
	decomposed := "e\u0301"
	replacement := "a\u030a"
	doc := helium.NewDefaultDocument()
	elem, err := doc.CreateElement("x" + decomposed)
	require.NoError(t, err)
	require.NoError(t, elem.SetAttribute("a"+decomposed, "x"+decomposed))
	require.NoError(t, elem.AddChild(doc.CreateText([]byte("x"+decomposed))))

	node := xpath3.NodeItem{Node: elem}
	nested := xpath3.NewMap([]xpath3.MapEntry{{
		Key: xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "key"},
		Value: xpath3.ItemSlice{xpath3.NewArray([]xpath3.Sequence{
			xpath3.ItemSlice{node},
		})},
	}})

	var buf bytes.Buffer
	err = xslt3.SerializeItems(&buf, xpath3.ItemSlice{node, node, nested}, nil, &xslt3.OutputDef{
		Method:            adaptiveMethod,
		NormalizationForm: normalizationFormNFC,
		ResolvedCharMap:   map[rune]string{'x': replacement},
	})
	require.NoError(t, err)

	serializedContent := replacement + "&#xE9;"
	wantNode := "<x" + decomposed + " a" + decomposed + `="` + serializedContent + `">` + serializedContent + "</x" + decomposed + ">"
	require.Equal(t, wantNode+"\n"+wantNode+"\n"+`map{"key":[`+wantNode+"]}", buf.String())
}

func newAdaptiveCommentAndProcessingInstruction(t *testing.T, data string) (*helium.Comment, *helium.ProcessingInstruction) {
	t.Helper()
	doc := helium.NewDefaultDocument()
	return doc.CreateComment([]byte(data)), doc.CreatePI("p", data)
}

func TestSerializeItemsAdaptiveCommentAndProcessingInstruction(t *testing.T) {
	decomposed := "e\u0301"
	data := "x" + decomposed
	comment, pi := newAdaptiveCommentAndProcessingInstruction(t, data)
	outDef := &xslt3.OutputDef{
		Method:            adaptiveMethod,
		NormalizationForm: normalizationFormNFC,
		ResolvedCharMap:   map[rune]string{'x': "mapped"},
	}

	var topLevel bytes.Buffer
	err := xslt3.SerializeItems(&topLevel, xpath3.ItemSlice{
		xpath3.NodeItem{Node: comment},
		xpath3.NodeItem{Node: pi},
	}, nil, outDef)
	require.NoError(t, err)
	require.Equal(t, "<!--"+data+"-->\n<?p "+data+"?>", topLevel.String())

	nestedMap := xpath3.NewMap([]xpath3.MapEntry{
		{
			Key:   xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "comment"},
			Value: xpath3.ItemSlice{xpath3.NodeItem{Node: comment}},
		},
		{
			Key:   xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "pi"},
			Value: xpath3.ItemSlice{xpath3.NodeItem{Node: pi}},
		},
	})
	nestedArray := xpath3.NewArray([]xpath3.Sequence{
		xpath3.ItemSlice{xpath3.NodeItem{Node: comment}},
		xpath3.ItemSlice{xpath3.NodeItem{Node: pi}},
	})

	var nested bytes.Buffer
	err = xslt3.SerializeItems(&nested, xpath3.ItemSlice{nestedMap, nestedArray}, nil, outDef)
	require.NoError(t, err)
	require.Equal(t, `map{"comment":<!--`+data+`-->,"pi":<?p `+data+`?>}`+"\n[<!--"+data+"-->,<?p "+data+"?>]", nested.String())
}

func TestSerializeItemsAdaptiveCommentAndProcessingInstructionInvalidChars(t *testing.T) {
	tests := []struct {
		name  string
		item  func(*testing.T) xpath3.Item
		ver   string
		valid bool
	}{
		{
			name: "CommentControlDefault",
			item: func(t *testing.T) xpath3.Item {
				comment, _ := newAdaptiveCommentAndProcessingInstruction(t, "a\x01b")
				return xpath3.NodeItem{Node: comment}
			},
		},
		{
			name: "ProcessingInstructionControlXML11",
			item: func(t *testing.T) xpath3.Item {
				_, pi := newAdaptiveCommentAndProcessingInstruction(t, "a\x01b")
				return xpath3.NodeItem{Node: pi}
			},
			ver: xmlVersion11,
		},
		{
			name: "CommentNELXML10",
			item: func(t *testing.T) xpath3.Item {
				comment, _ := newAdaptiveCommentAndProcessingInstruction(t, "a\u0085b")
				return xpath3.NodeItem{Node: comment}
			},
			ver:   "1.0",
			valid: true,
		},
		{
			name: "ProcessingInstructionNELDefault",
			item: func(t *testing.T) xpath3.Item {
				_, pi := newAdaptiveCommentAndProcessingInstruction(t, "a\u0085b")
				return xpath3.NodeItem{Node: pi}
			},
			valid: true,
		},
		{
			name: "CommentNELXML11",
			item: func(t *testing.T) xpath3.Item {
				comment, _ := newAdaptiveCommentAndProcessingInstruction(t, "a\u0085b")
				return xpath3.NodeItem{Node: comment}
			},
			ver: xmlVersion11,
		},
		{
			name: "ProcessingInstructionNELXML11",
			item: func(t *testing.T) xpath3.Item {
				_, pi := newAdaptiveCommentAndProcessingInstruction(t, "a\u0085b")
				return xpath3.NodeItem{Node: pi}
			},
			ver: xmlVersion11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := xslt3.SerializeItems(&buf, xpath3.ItemSlice{tt.item(t)}, nil, &xslt3.OutputDef{
				Method:  adaptiveMethod,
				Version: tt.ver,
			})
			if tt.valid {
				require.NoError(t, err)
				return
			}
			requireSERE0006(t, err)
		})
	}
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

// outMethodXHTML is the XHTML output method. Its version parameter uses an XML
// VersionNum, just like the XML output method's version parameter.
const outMethodXHTML = "xhtml"

// xmlVersion11 is the XML 1.1 output version used by the invalid-character
// serialization tests.
const xmlVersion11 = "1.1"

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

// newBadCharDocument records XML 1.1 so the adaptive XML 1.0 default must
// override the source document's version.
func newBadCharDocument(t *testing.T) *helium.Document {
	t.Helper()
	doc := newBadCharElement(t).OwnerDocument()
	doc.SetVersion(xmlVersion11)
	return doc
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

// requireControlCharRef asserts the serialized output carries a character
// reference to U+0001 (decimal or hex form) around the surrounding text, i.e.
// XML 1.1 char-referenced the restricted control instead of rejecting it.
func requireControlCharRef(t *testing.T, out string) {
	t.Helper()
	hasRef := strings.Contains(out, "&#1;") || strings.Contains(out, "&#x1;")
	require.True(t, hasRef, "expected a U+0001 character reference in %q", out)
	require.Contains(t, out, "a")
	require.Contains(t, out, "b")
}

// SerializeItems with method="xml" must propagate the writer's invalid-char
// rejection as SERE0006 rather than silently truncating the output. Under an
// XML 1.1 OutputDef version, U+0001 is a legal character reference, so the same
// item serializes with nil error and a char reference instead.
func TestSerializeItemsXMLInvalidChar(t *testing.T) {
	root := newBadCharElement(t)
	items := xpath3.ItemSlice{xpath3.NodeItem{Node: root}}

	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, items, nil, &xslt3.OutputDef{Method: outMethodXML})
	requireSERE0006(t, err)

	var buf10 bytes.Buffer
	err = xslt3.SerializeItems(&buf10, items, nil, &xslt3.OutputDef{Method: outMethodXML, Version: "1.0"})
	requireSERE0006(t, err)

	var buf11 bytes.Buffer
	err = xslt3.SerializeItems(&buf11, items, nil, &xslt3.OutputDef{Method: outMethodXML, Version: xmlVersion11})
	require.NoError(t, err)
	requireControlCharRef(t, buf11.String())
}

// SerializeItems with method="xhtml" passes its XML version to the per-item
// writer. Under XML 1.1, U+0001 is a legal character reference; the default
// and XML 1.0 versions reject it as SERE0006.
func TestSerializeItemsXHTMLInvalidChar(t *testing.T) {
	root := newBadCharElement(t)
	items := xpath3.ItemSlice{xpath3.NodeItem{Node: root}}

	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, items, nil, &xslt3.OutputDef{Method: outMethodXHTML})
	requireSERE0006(t, err)

	var buf10 bytes.Buffer
	err = xslt3.SerializeItems(&buf10, items, nil, &xslt3.OutputDef{Method: outMethodXHTML, Version: "1.0"})
	requireSERE0006(t, err)

	var buf11 bytes.Buffer
	err = xslt3.SerializeItems(&buf11, items, nil, &xslt3.OutputDef{Method: outMethodXHTML, Version: xmlVersion11})
	require.NoError(t, err)
	requireControlCharRef(t, buf11.String())
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

// messageRecordingHandler records each xsl:message delivered to it.
type messageRecordingHandler struct {
	messages []string
}

func (h *messageRecordingHandler) HandleMessage(msg string, _ bool) error {
	h.messages = append(h.messages, msg)
	return nil
}

// xsl:message must report a node serialization failure as SERE0006 and avoid
// delivering a partial message to its handler.
func TestMessageInvalidCharSERE0006(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:message select="/*"/>
    <out/>
  </xsl:template>
</xsl:stylesheet>`)

	root := newBadCharElement(t)
	handler := &messageRecordingHandler{}
	_, err := ss.Transform(root.OwnerDocument()).MessageHandler(handler).Do(t.Context())
	requireSERE0006(t, err)
	require.Empty(t, handler.messages)
}

// SerializeItems with method="adaptive" over a multi-item sequence containing a
// node with an invalid character must propagate SERE0006 (the single-element
// path already propagates via serializeXML; this exercises the per-item path).
// Under an XML 1.1 version, adaptive inherits the version parameter for the
// embedded node serialization, so U+0001 becomes a char reference with nil error.
func TestSerializeItemsAdaptiveInvalidChar(t *testing.T) {
	root := newBadCharElement(t)
	items := xpath3.ItemSlice{xpath3.NodeItem{Node: root}, xpath3.NodeItem{Node: root}}

	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, items, nil, &xslt3.OutputDef{Method: adaptiveMethod})
	requireSERE0006(t, err)

	var buf10 bytes.Buffer
	err = xslt3.SerializeItems(&buf10, items, nil, &xslt3.OutputDef{Method: adaptiveMethod, Version: "1.0"})
	requireSERE0006(t, err)

	var buf11 bytes.Buffer
	err = xslt3.SerializeItems(&buf11, items, nil, &xslt3.OutputDef{Method: adaptiveMethod, Version: xmlVersion11})
	require.NoError(t, err)
	requireControlCharRef(t, buf11.String())
}

// Adaptive XML serialization uses its output version consistently for each
// path that delegates element or document items to the XML writer. A source
// document marked XML 1.1 cannot change the default XML 1.0 result.
func TestSerializeItemsAdaptiveXMLVersion(t *testing.T) {
	tests := []struct {
		name               string
		items              func(*testing.T) xpath3.Sequence
		doc                func(*testing.T) *helium.Document
		wantXMLDeclaration bool
	}{
		{
			name:               "NoItemsDocument",
			doc:                newBadCharDocument,
			wantXMLDeclaration: true,
		},
		{
			name: "SingletonElement",
			items: func(t *testing.T) xpath3.Sequence {
				return xpath3.ItemSlice{xpath3.NodeItem{Node: newBadCharElement(t)}}
			},
			wantXMLDeclaration: true,
		},
		{
			name: "SingletonDocument",
			items: func(t *testing.T) xpath3.Sequence {
				return xpath3.ItemSlice{xpath3.NodeItem{Node: newBadCharDocument(t)}}
			},
		},
		{
			name: "MapContainedDocument",
			items: func(t *testing.T) xpath3.Sequence {
				doc := newBadCharDocument(t)
				return xpath3.ItemSlice{xpath3.NewMap([]xpath3.MapEntry{{
					Key:   xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "doc"},
					Value: xpath3.ItemSlice{xpath3.NodeItem{Node: doc}},
				}})}
			},
		},
		{
			name: "ArrayContainedDocument",
			items: func(t *testing.T) xpath3.Sequence {
				doc := newBadCharDocument(t)
				return xpath3.ItemSlice{xpath3.NewArray([]xpath3.Sequence{
					xpath3.ItemSlice{xpath3.NodeItem{Node: doc}},
				})}
			},
		},
	}
	versions := []struct {
		name    string
		version string
	}{
		{name: "Default"},
		{name: "XML10", version: "1.0"},
		{name: "XML11", version: xmlVersion11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, version := range versions {
				t.Run(version.name, func(t *testing.T) {
					var buf bytes.Buffer
					var items xpath3.Sequence
					if tt.items != nil {
						items = tt.items(t)
					}
					var doc *helium.Document
					if tt.doc != nil {
						doc = tt.doc(t)
					}
					err := xslt3.SerializeItems(&buf, items, doc, &xslt3.OutputDef{
						Method:  adaptiveMethod,
						Version: version.version,
					})
					if version.version != xmlVersion11 {
						requireSERE0006(t, err)
						return
					}

					require.NoError(t, err)
					requireControlCharRef(t, buf.String())
					if tt.wantXMLDeclaration {
						require.Contains(t, buf.String(), `<?xml version="`+xmlVersion11+`"`)
					} else {
						require.NotContains(t, buf.String(), "<?xml")
					}
				})
			}
		})
	}
}
