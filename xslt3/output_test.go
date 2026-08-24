package xslt3_test

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

const adaptiveMethod = "adaptive"

type oneByteWriter struct {
	buf bytes.Buffer
}

func (w *oneByteWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return w.buf.Write(p[:1])
}

type adaptiveResultDocSerializer struct {
	serialized string
}

func (h *adaptiveResultDocSerializer) HandleResultDocument(_ string, doc *helium.Document, outDef *xslt3.OutputDef) error {
	var out bytes.Buffer
	if err := xslt3.SerializeResult(&out, doc, outDef); err != nil {
		return err
	}
	h.serialized = out.String()
	return nil
}

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

func TestSerializeResultXMLOmitsWriterTerminators(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/"><root>hello</root></xsl:template>
</xsl:stylesheet>`)

	doc, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.NoError(t, err)

	var buf bytes.Buffer
	err = xslt3.SerializeResult(&buf, doc, ss.DefaultOutputDef())
	require.NoError(t, err)
	require.Equal(t, "<root>hello</root>", buf.String())
}

func TestSerializeResultReportsShortWrite(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>content</root>`))
	require.NoError(t, err)
	nonASCIIDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>あ</root>`))
	require.NoError(t, err)

	for _, tc := range []struct {
		name   string
		doc    *helium.Document
		outDef *xslt3.OutputDef
	}{
		{
			name:   "DirectWriter",
			outDef: &xslt3.OutputDef{Method: outMethodXML, OmitDeclaration: true},
		},
		{
			name:   "PostProcessedDeclaration",
			outDef: &xslt3.OutputDef{Method: outMethodXML, Standalone: "yes"},
		},
		{
			name: "PostProcessedCharacterMap",
			outDef: &xslt3.OutputDef{
				Method:          outMethodXML,
				Standalone:      "yes",
				ResolvedCharMap: map[rune]string{'c': "C"},
			},
		},
		{
			name: "Normalized",
			outDef: &xslt3.OutputDef{
				Method:            outMethodXML,
				OmitDeclaration:   true,
				NormalizationForm: "NFC",
			},
		},
		{
			name: "UTF16",
			outDef: &xslt3.OutputDef{
				Method:          outMethodXML,
				OmitDeclaration: true,
				Encoding:        "UTF-16",
			},
		},
		{
			name: "EncodedBytes",
			doc:  nonASCIIDoc,
			outDef: &xslt3.OutputDef{
				Method:          outMethodXML,
				OmitDeclaration: true,
				Encoding:        "Shift_JIS",
			},
		},
		{
			name: "EncodedCharacterReference",
			doc:  nonASCIIDoc,
			outDef: &xslt3.OutputDef{
				Method:          outMethodXML,
				OmitDeclaration: true,
				Encoding:        "US-ASCII",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testDoc := doc
			if tc.doc != nil {
				testDoc = tc.doc
			}
			var dst oneByteWriter
			err := xslt3.SerializeResult(&dst, testDoc, tc.outDef)
			require.ErrorIs(t, err, io.ErrShortWrite)
			require.NotEmpty(t, dst.buf.String())
			if tc.outDef.Encoding == "" {
				require.True(t, strings.HasPrefix(`<root>content</root>`, dst.buf.String()))
			}
		})
	}
}

func TestSerializeResultXMLPreservesExplicitTrailingTextNewline(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" encoding="UTF-8" omit-xml-declaration="yes"/>
  <xsl:template match="/"><out/><xsl:text>&#10;</xsl:text></xsl:template>
</xsl:stylesheet>`)

	doc, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.NoError(t, err)

	var buf bytes.Buffer
	err = xslt3.SerializeResult(&buf, doc, ss.DefaultOutputDef())
	require.NoError(t, err)
	require.Equal(t, "<out/>\n", buf.String())
}

func TestSerializeResultXMLIndentationDiscardsWhitespaceOnlyTopLevelText(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" encoding="iso-8859-1" indent="yes" omit-xml-declaration="yes"/>
  <xsl:template match="/"><out/><xsl:text>&#10;</xsl:text></xsl:template>
</xsl:stylesheet>`)

	doc, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.NoError(t, err)

	var buf bytes.Buffer
	err = xslt3.SerializeResult(&buf, doc, ss.DefaultOutputDef())
	require.NoError(t, err)
	require.Equal(t, "\n<out/>", buf.String())
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

// Top-level xsl:comment and xsl:processing-instruction output uses standalone
// adaptive-item serialization, and no XML declaration.
func TestPrimaryAdaptiveCommentAndProcessingInstruction(t *testing.T) {
	tests := []struct {
		name        string
		instruction string
		want        string
	}{
		{
			name:        "Comment",
			instruction: `<xsl:comment select="'comment'"/>`,
			want:        "<!--comment-->",
		},
		{
			name:        "ProcessingInstruction",
			instruction: `<xsl:processing-instruction name="target" select="'data'"/>`,
			want:        "<?target data?>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive"/>
  <xsl:template match="/">`+tt.instruction+`</xsl:template>
</xsl:stylesheet>`)

			out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			require.NoError(t, err)
			require.Equal(t, tt.want, out)
			require.NotContains(t, out, "<?xml")
		})
	}
}

func TestPrimaryAdaptiveCommentAndProcessingInstructionSequenceItemSeparator(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive" item-separator=" | "/>
  <xsl:template match="/">
    <xsl:comment select="'first'"/>
    <xsl:processing-instruction name="target" select="'data'"/>
    <xsl:comment select="'last'"/>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Equal(t, "<!--first--> | <?target data?> | <!--last-->", out)
	require.NotContains(t, out, "<?xml")
}

func TestPrimaryAdaptiveMarkupThenElementKeepsDeferredSeparators(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive" item-separator=" | "/>
  <xsl:template match="/">
    <xsl:comment select="'first'"/>
    <xsl:processing-instruction name="target" select="'data'"/>
    <result/>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, `<?xml version="1.0"`)
	require.Contains(t, out, "<!--first--> | <?target data?> | <result/>")
}

func TestPrimaryAdaptiveMarkupSeparatorAppliesCharacterMapAndNormalization(t *testing.T) {
	decomposed := "e\u0301"
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:character-map name="map">
    <xsl:output-character character="x" string="mapped"/>
  </xsl:character-map>
  <xsl:output method="adaptive" item-separator="x`+decomposed+`" normalization-form="NFC" use-character-maps="map"/>
  <xsl:template match="/">
    <xsl:comment select="'x`+decomposed+`'"/>
    <xsl:processing-instruction name="target" select="'x`+decomposed+`'"/>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Equal(t, "<!--x"+decomposed+"-->mapped&#xE9;<?target x"+decomposed+"?>", out)
}

func TestSecondaryAdaptiveMarkupSequenceItemSeparatorHasNoDeclaration(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="secondary.xml" method="adaptive" item-separator=" | ">
      <xsl:comment select="'first'"/>
      <xsl:processing-instruction name="target" select="'data'"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	handler := &adaptiveResultDocSerializer{}
	_, err := ss.Transform(parseTransformSource(t)).ResultDocumentHandler(handler).Do(t.Context())
	require.NoError(t, err)
	require.Equal(t, "<!--first--> | <?target data?>", handler.serialized)
	require.NotContains(t, handler.serialized, "<?xml")
}

func TestPrimaryAdaptiveCommentAndProcessingInstructionSequenceItemSeparatorTextFallback(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive" item-separator=" | "/>
  <xsl:template match="/">
    <xsl:comment select="'first'"/>
    <xsl:text>text</xsl:text>
    <xsl:comment select="'last'"/>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, `<?xml version="1.0"`)
	require.Contains(t, out, "<!--first-->text | <!--last-->")
}

func TestPrimaryAdaptiveCommentAndProcessingInstructionInvalidCharacter(t *testing.T) {
	tests := []struct {
		name        string
		instruction string
	}{
		{
			name:        "Comment",
			instruction: `<xsl:comment select="codepoints-to-string(1)"/>`,
		},
		{
			name:        "ProcessingInstruction",
			instruction: `<xsl:processing-instruction name="target" select="codepoints-to-string(1)"/>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive"/>
  <xsl:template match="/">`+tt.instruction+`</xsl:template>
</xsl:stylesheet>`)

			_, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			requireSERE0006(t, err)
		})
	}
}

func TestPrimaryAdaptiveCommentAndProcessingInstructionSequenceItemSeparatorInvalidCharacter(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive" item-separator=" | "/>
  <xsl:template match="/">
    <xsl:comment select="'first'"/>
    <xsl:processing-instruction name="target" select="codepoints-to-string(1)"/>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	requireSERE0006(t, err)
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

func TestSerializeDocumentItemsOmitWriterTerminators(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<source/>`))
	require.NoError(t, err)
	items := xpath3.ItemSlice{xpath3.NodeItem{Node: doc}}

	for _, tc := range []struct {
		name   string
		outDef *xslt3.OutputDef
		want   string
	}{
		{
			name:   "Adaptive",
			outDef: &xslt3.OutputDef{Method: adaptiveMethod},
			want:   `<source/>`,
		},
		{
			name:   "XMLSequence",
			outDef: &xslt3.OutputDef{Method: outMethodXML},
			want:   `<source/>`,
		},
		{
			name: "JSONNodeXML",
			outDef: &xslt3.OutputDef{
				Method:               outMethodJSON,
				JSONNodeOutputMethod: outMethodXML,
			},
			want: `"<source\/>"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var dst strings.Builder
			require.NoError(t, xslt3.SerializeItems(&dst, items, doc, tc.outDef))
			require.Equal(t, tc.want, dst.String())
		})
	}
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
			method: outMethodJSON,
			item:   value,
			want:   `"` + replacement + composed + `"`,
		},
		{
			name:   "JSONMap",
			method: outMethodJSON,
			item:   mapItem,
			want:   `{"key":"` + replacement + composed + `"}`,
		},
		{
			name:   "JSONArray",
			method: outMethodJSON,
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
			want:  "<out>" + mappedNFC + "</out>",
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
			ver:   xmlVersion10,
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

// Output methods shared by serialization tests.
const (
	outMethodJSON = "json"
	outMethodXML  = "xml"
)

// outMethodXHTML is the XHTML output method. Its version parameter uses an XML
// VersionNum, just like the XML output method's version parameter.
const outMethodXHTML = "xhtml"

// XML output versions used by the invalid-character serialization tests.
const (
	xmlVersion10 = "1.0"
	xmlVersion11 = "1.1"
)

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
// rejection as SERE0006, truncating no output. Under an
// XML 1.1 OutputDef version, U+0001 is a legal character reference, so the same
// item serializes with nil error and a char reference instead.
func TestSerializeItemsXMLInvalidChar(t *testing.T) {
	root := newBadCharElement(t)
	items := xpath3.ItemSlice{xpath3.NodeItem{Node: root}}

	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, items, nil, &xslt3.OutputDef{Method: outMethodXML})
	requireSERE0006(t, err)

	var buf10 bytes.Buffer
	err = xslt3.SerializeItems(&buf10, items, nil, &xslt3.OutputDef{Method: outMethodXML, Version: xmlVersion10})
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
	err = xslt3.SerializeItems(&buf10, items, nil, &xslt3.OutputDef{Method: outMethodXHTML, Version: xmlVersion10})
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
	err := xslt3.SerializeItems(&buf, items, nil, &xslt3.OutputDef{Method: outMethodJSON, JSONNodeOutputMethod: outMethodXML})
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

func TestMessageDocumentPreservesTopLevelWhitespace(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="message" as="document-node()">
      <xsl:document><xsl:text>&#10;</xsl:text><out/><xsl:text>&#10;</xsl:text></xsl:document>
    </xsl:variable>
    <xsl:message select="$message"/>
    <result/>
  </xsl:template>
</xsl:stylesheet>`)

	handler := &messageRecordingHandler{}
	_, err := ss.Transform(parseTransformSource(t)).MessageHandler(handler).Do(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"\n<out/>\n"}, handler.messages)
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
	err = xslt3.SerializeItems(&buf10, items, nil, &xslt3.OutputDef{Method: adaptiveMethod, Version: xmlVersion10})
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
		{name: "XML10", version: xmlVersion10},
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

// mutatedMarker is a sentinel written into derived/snapshot OutputDef fields to
// prove that mutating them never reaches compiled or shared stylesheet state.
const mutatedMarker = "MUTATED"

// A-003: serializeResult must not mutate the caller-supplied OutputDef during
// html/xhtml auto-detection. Reusing one OutputDef{Method:"xml"} across an
// <html> doc and a non-html doc must not turn the second into HTML output.
func TestSerializeResultDoesNotMutateOutputDef(t *testing.T) {
	parse := func(src string) *helium.Document {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		return doc
	}

	outDef := &xslt3.OutputDef{Method: outMethodXML}

	htmlDoc := parse(`<html><body><br/></body></html>`)
	var htmlBuf strings.Builder
	require.NoError(t, xslt3.SerializeResult(&htmlBuf, htmlDoc, outDef))

	// outDef must still be xml-method after auto-detecting HTML.
	require.Equal(t, outMethodXML, outDef.Method, "outDef.Method must not be mutated")
	require.False(t, outDef.OmitDeclaration, "outDef.OmitDeclaration must not be mutated")

	xmlDoc := parse(`<root><br/></root>`)
	var xmlBuf strings.Builder
	require.NoError(t, xslt3.SerializeResult(&xmlBuf, xmlDoc, outDef))

	// The second (non-html) doc must serialize as XML, not HTML.
	out := xmlBuf.String()
	require.Contains(t, out, "<br/>", "second doc must be XML-serialized with self-closing br")
	require.NotContains(t, out, "<br>", "second doc must not be HTML-serialized")
}

// A-002 / A-004: DefaultOutputDef must return a deep clone, not the internal
// pointer. Mutating the clone — including its pointer fields — must never reach
// through to the stylesheet's internal state.
func TestDefaultOutputDefReturnsClone(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" item-separator="|" build-tree="yes"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	d1 := ss.DefaultOutputDef()
	require.NotNil(t, d1)
	d2 := ss.DefaultOutputDef()
	require.NotNil(t, d2)
	require.NotSame(t, d1, d2, "DefaultOutputDef must return a fresh clone each call")

	// The pointer fields must themselves be fresh allocations, not aliases.
	require.NotNil(t, d1.ItemSeparator)
	require.NotNil(t, d2.ItemSeparator)
	require.NotSame(t, d1.ItemSeparator, d2.ItemSeparator, "ItemSeparator pointer must be deep-cloned")
	require.NotNil(t, d1.BuildTree)
	require.NotNil(t, d2.BuildTree)
	require.NotSame(t, d1.BuildTree, d2.BuildTree, "BuildTree pointer must be deep-cloned")

	// Mutating the returned def — scalar and pointee — must not affect internal state.
	d1.Method = "html"
	*d1.ItemSeparator = mutatedMarker
	*d1.BuildTree = false

	d3 := ss.DefaultOutputDef()
	require.Equal(t, outMethodXML, d3.Method, "mutating a returned def must not corrupt internal state")
	require.NotNil(t, d3.ItemSeparator)
	require.Equal(t, "|", *d3.ItemSeparator, "mutating clone *ItemSeparator must not corrupt internal state")
	require.NotNil(t, d3.BuildTree)
	require.True(t, *d3.BuildTree, "mutating clone *BuildTree must not corrupt internal state")
}

// A-006: Do/WriteTo must not mutate the shared invocationConfig; ResolvedOutputDef
// must return an independent deep-cloned snapshot.
func TestResolvedOutputDefIsSnapshot(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" item-separator="|"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	_, err := inv.Do(t.Context())
	require.NoError(t, err)

	r1 := inv.ResolvedOutputDef()
	require.NotNil(t, r1)
	require.NotNil(t, r1.ItemSeparator)

	// Mutating the snapshot — scalar and pointee — must not affect a later read.
	r1.Method = "html"
	*r1.ItemSeparator = mutatedMarker

	r2 := inv.ResolvedOutputDef()
	require.NotNil(t, r2)
	require.Equal(t, outMethodXML, r2.Method, "ResolvedOutputDef must return an independent snapshot")
	require.NotNil(t, r2.ItemSeparator)
	require.Equal(t, "|", *r2.ItemSeparator, "mutating snapshot *ItemSeparator must not affect a later read")
}

// W01: a ResultDocumentHandler must receive an OutputDef whose pointer/slice/map
// fields are independent of the compiled stylesheet. Mutating those fields from
// the handler must not corrupt the compiled named format shared across runs.
type captureResultDocHandler struct {
	outDef *xslt3.OutputDef
}

func (h *captureResultDocHandler) HandleResultDocument(_ string, _ *helium.Document, outDef *xslt3.OutputDef) error {
	h.outDef = outDef
	return nil
}

func TestResultDocumentHandlerOutputDefIsIsolated(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output name="fmt" method="xml" item-separator="|" build-tree="yes"
              suppress-indentation="a b"/>
  <xsl:template match="/">
    <xsl:result-document href="secondary.xml" format="fmt"><secondary/></xsl:result-document>
    <out/>
  </xsl:template>
</xsl:stylesheet>`)

	run := func() *xslt3.OutputDef {
		h := &captureResultDocHandler{}
		_, err := ss.Transform(parseTransformSource(t)).ResultDocumentHandler(h).Do(t.Context())
		require.NoError(t, err)
		require.NotNil(t, h.outDef, "handler must receive an OutputDef")
		return h.outDef
	}

	first := run()
	require.NotNil(t, first.ItemSeparator)
	require.Equal(t, "|", *first.ItemSeparator)
	require.NotNil(t, first.BuildTree)
	require.True(t, *first.BuildTree)
	require.Equal(t, []string{"a", "b"}, first.SuppressIndentation)

	// Mutate every pointer/slice/map field on the delivered def.
	*first.ItemSeparator = mutatedMarker
	*first.BuildTree = false
	first.SuppressIndentation[0] = mutatedMarker
	first.SuppressIndentation = append(first.SuppressIndentation, "MUTATED2")
	if first.ResolvedCharMap == nil {
		first.ResolvedCharMap = map[rune]string{}
	}
	first.ResolvedCharMap['x'] = mutatedMarker

	// A second run must observe the original compiled values, proving the first
	// delivered def did not alias compiled/shared state.
	second := run()
	require.NotNil(t, second.ItemSeparator)
	require.Equal(t, "|", *second.ItemSeparator, "compiled item-separator must be unaffected")
	require.NotNil(t, second.BuildTree)
	require.True(t, *second.BuildTree, "compiled build-tree must be unaffected")
	require.Equal(t, []string{"a", "b"}, second.SuppressIndentation, "compiled suppress-indentation must be unaffected")
	require.Empty(t, second.ResolvedCharMap, "compiled char map must be unaffected")
}

// W01: a primary xsl:result-document delivers its effective output def via
// ResolvedOutputDef. Mutating that snapshot's pointer/slice/map fields must not
// corrupt the compiled stylesheet across runs.
func TestPrimaryResultDocumentResolvedOutputDefIsIsolated(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" item-separator="|" build-tree="yes"/>
  <xsl:template match="/">
    <xsl:result-document item-separator=";"><out/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	run := func() *xslt3.OutputDef {
		inv := ss.Transform(parseTransformSource(t))
		_, err := inv.Do(t.Context())
		require.NoError(t, err)
		r := inv.ResolvedOutputDef()
		require.NotNil(t, r)
		return r
	}

	first := run()
	require.NotNil(t, first.ItemSeparator)
	require.Equal(t, ";", *first.ItemSeparator, "result-document override must apply")

	*first.ItemSeparator = mutatedMarker
	if first.BuildTree != nil {
		*first.BuildTree = false
	}

	second := run()
	require.NotNil(t, second.ItemSeparator)
	require.Equal(t, ";", *second.ItemSeparator, "compiled/override state must be unaffected across runs")
}

// A primary xsl:result-document with an explicit false boolean serialization
// AVT must override an inherited true from xsl:output. Before the fix the merge
// OR-ed the override with the inherited value, so an explicit false could never
// turn an inherited true back off.
func TestPrimaryResultDocBooleanFalseOverridesInheritedTrue(t *testing.T) {
	resolve := func(t *testing.T, output, resultDocAttrs string) *xslt3.OutputDef {
		t.Helper()
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  `+output+`
  <xsl:template match="/">
    <xsl:result-document `+resultDocAttrs+`><out/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)
		inv := ss.Transform(parseTransformSource(t))
		_, err := inv.Do(t.Context())
		require.NoError(t, err)
		r := inv.ResolvedOutputDef()
		require.NotNil(t, r)
		return r
	}

	t.Run("indent false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="xml" indent="yes"/>`, `indent="{false()}"`)
		require.False(t, r.Indent, "explicit indent=false must override inherited indent=yes")
	})

	t.Run("indent inherited yes stays on when not overridden", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="xml" indent="yes"/>`, `method="xml"`)
		require.True(t, r.Indent, "inherited indent=yes must remain on when not overridden")
	})

	t.Run("omit-xml-declaration false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="xml" omit-xml-declaration="yes"/>`, `omit-xml-declaration="{false()}"`)
		require.False(t, r.OmitDeclaration, "explicit omit-xml-declaration=false must override inherited yes")
	})

	t.Run("byte-order-mark false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="xml" byte-order-mark="yes"/>`, `byte-order-mark="{false()}"`)
		require.False(t, r.ByteOrderMark, "explicit byte-order-mark=false must override inherited yes")
	})

	t.Run("escape-uri-attributes false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="html" escape-uri-attributes="yes"/>`, `escape-uri-attributes="{false()}"`)
		require.NotNil(t, r.EscapeURIAttributes)
		require.False(t, *r.EscapeURIAttributes, "explicit escape-uri-attributes=false must override inherited yes")
	})

	t.Run("include-content-type false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="html" include-content-type="yes"/>`, `include-content-type="{false()}"`)
		require.NotNil(t, r.IncludeContentType)
		require.False(t, *r.IncludeContentType, "explicit include-content-type=false must override inherited yes")
	})

	t.Run("undeclare-prefixes false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="xml" version="1.1" undeclare-prefixes="yes"/>`, `undeclare-prefixes="{false()}"`)
		require.False(t, r.UndeclarePrefixes, "explicit undeclare-prefixes=false must override inherited yes")
	})
}

// A-006 race: concurrent Serialize/ResolvedOutputDef on the SAME Invocation
// value must be safe. Run under -race to catch data races on the shared config.
func TestConcurrentSerializeAndResolvedOutputDef(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" item-separator="|"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	ctx := t.Context()

	const n = 16
	var wg sync.WaitGroup
	wg.Add(n * 2)
	for range n {
		go func() {
			defer wg.Done()
			_, err := inv.Serialize(ctx)
			require.NoError(t, err)
		}()
		go func() {
			defer wg.Done()
			// Reading the resolved def concurrently with Serialize must not race.
			_ = inv.ResolvedOutputDef()
		}()
	}
	wg.Wait()

	out, err := inv.Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "<out/>")
}

type shortWriteAtCallWriter struct {
	buf     bytes.Buffer
	call    int
	shortAt int
}

func (w *shortWriteAtCallWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	w.call++
	if w.call == w.shortAt {
		if len(p) == 1 {
			return 0, nil
		}
		return w.buf.Write(p[:1])
	}
	return w.buf.Write(p)
}

func requireExactOutputAndShortWrite(t *testing.T, want []byte, shortAt int, serialize func(io.Writer) error) {
	t.Helper()

	var full bytes.Buffer
	require.NoError(t, serialize(&full))
	require.Equal(t, want, full.Bytes())

	dst := &shortWriteAtCallWriter{shortAt: shortAt}
	require.ErrorIs(t, serialize(dst), io.ErrShortWrite)
	require.Less(t, dst.buf.Len(), len(want))
	require.Equal(t, string(want[:dst.buf.Len()]), dst.buf.String())
}

func parseShortWriteDocument(t *testing.T) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)
	return doc
}

func TestSerializeResultBOMReportsShortWrite(t *testing.T) {
	doc := helium.NewDefaultDocument()

	for _, tc := range []struct {
		name   string
		want   []byte
		outDef *xslt3.OutputDef
	}{
		{
			name: "UTF8",
			want: []byte{0xEF, 0xBB, 0xBF},
			outDef: &xslt3.OutputDef{
				Method:        "text",
				ByteOrderMark: true,
			},
		},
		{
			name: "UTF16",
			want: []byte{0xFE, 0xFF},
			outDef: &xslt3.OutputDef{
				Method:   "text",
				Encoding: "UTF-16",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireExactOutputAndShortWrite(t, tc.want, 1, func(w io.Writer) error {
				return xslt3.SerializeResult(w, doc, tc.outDef)
			})
		})
	}
}

func TestSerializeResultDirectCharacterMapReportsShortWrite(t *testing.T) {
	doc := parseShortWriteDocument(t)
	outDef := &xslt3.OutputDef{
		Method:          outMethodXML,
		OmitDeclaration: true,
		ResolvedCharMap: map[rune]string{'x': "x"},
	}

	requireExactOutputAndShortWrite(t, []byte(`<root/>`), 1, func(w io.Writer) error {
		return xslt3.SerializeResult(w, doc, outDef)
	})
}

func TestSerializeItemsXMLReportsShortWrite(t *testing.T) {
	doc := parseShortWriteDocument(t)
	separator := "||"
	stringItem := func(s string) xpath3.AtomicValue {
		return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}
	}

	for _, tc := range []struct {
		name    string
		want    string
		items   xpath3.Sequence
		outDef  *xslt3.OutputDef
		shortAt int
	}{
		{
			name:    "DocumentItem",
			want:    `<root/>`,
			items:   xpath3.ItemSlice{xpath3.NodeItem{Node: doc}},
			outDef:  &xslt3.OutputDef{Method: outMethodXML},
			shortAt: 1,
		},
		{
			name:    "AtomicItem",
			want:    "alpha",
			items:   xpath3.ItemSlice{stringItem("alpha")},
			outDef:  &xslt3.OutputDef{Method: outMethodXML},
			shortAt: 1,
		},
		{
			name:  "Separator",
			want:  "a||bravo",
			items: xpath3.ItemSlice{stringItem("a"), stringItem("bravo")},
			outDef: &xslt3.OutputDef{
				Method:        outMethodXML,
				ItemSeparator: &separator,
			},
			shortAt: 2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireExactOutputAndShortWrite(t, []byte(tc.want), tc.shortAt, func(w io.Writer) error {
				return xslt3.SerializeItems(w, tc.items, nil, tc.outDef)
			})
		})
	}
}

func TestSerializeItemsAdaptiveReportsShortWrite(t *testing.T) {
	doc := parseShortWriteDocument(t)
	separator := "||"
	stringItem := func(s string) xpath3.AtomicValue {
		return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}
	}

	for _, tc := range []struct {
		name    string
		want    string
		items   xpath3.Sequence
		outDef  *xslt3.OutputDef
		shortAt int
	}{
		{
			name:    "DocumentItem",
			want:    `<root/>`,
			items:   xpath3.ItemSlice{xpath3.NodeItem{Node: doc}},
			outDef:  &xslt3.OutputDef{Method: adaptiveMethod},
			shortAt: 1,
		},
		{
			name:  "Separator",
			want:  `"a"||"bravo"`,
			items: xpath3.ItemSlice{stringItem("a"), stringItem("bravo")},
			outDef: &xslt3.OutputDef{
				Method:        adaptiveMethod,
				ItemSeparator: &separator,
			},
			shortAt: 2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireExactOutputAndShortWrite(t, []byte(tc.want), tc.shortAt, func(w io.Writer) error {
				return xslt3.SerializeItems(w, tc.items, nil, tc.outDef)
			})
		})
	}
}

func TestSerializeItemsJSONReportsShortWrite(t *testing.T) {
	doc := parseShortWriteDocument(t)
	node := xpath3.NodeItem{Node: doc}

	for _, tc := range []struct {
		name  string
		want  string
		items xpath3.Sequence
	}{
		{
			name:  "SingleValue",
			want:  `"<root\/>"`,
			items: xpath3.ItemSlice{node},
		},
		{
			name:  "Array",
			want:  `["<root\/>","<root\/>"]`,
			items: xpath3.ItemSlice{node, node},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outDef := &xslt3.OutputDef{
				Method:               outMethodJSON,
				JSONNodeOutputMethod: outMethodXML,
			}
			requireExactOutputAndShortWrite(t, []byte(tc.want), 1, func(w io.Writer) error {
				return xslt3.SerializeItems(w, tc.items, nil, outDef)
			})
		})
	}
}

func TestSerializeItemsJSONPostProcessingReportsShortWrite(t *testing.T) {
	item := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "alpha"}
	outDef := &xslt3.OutputDef{
		Method:          outMethodJSON,
		ResolvedCharMap: map[rune]string{'a': "A"},
	}

	requireExactOutputAndShortWrite(t, []byte(`"AlphA"`), 1, func(w io.Writer) error {
		return xslt3.SerializeItems(w, xpath3.ItemSlice{item}, nil, outDef)
	})
}

func TestSerializeResultHTMLCharacterMapReportsShortWrite(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>alpha</root>`))
	require.NoError(t, err)
	includeContentType := false
	outDef := &xslt3.OutputDef{
		Method:             "html",
		IncludeContentType: &includeContentType,
		ResolvedCharMap:    map[rune]string{'a': "A"},
	}

	requireExactOutputAndShortWrite(t, []byte(`<root>AlphA</root>`), 1, func(w io.Writer) error {
		return xslt3.SerializeResult(w, doc, outDef)
	})
}

func TestSerializeResultJSONCharacterMapReportsShortWrite(t *testing.T) {
	doc := parseShortWriteDocument(t)
	outDef := &xslt3.OutputDef{
		Method:               outMethodJSON,
		JSONNodeOutputMethod: outMethodXML,
		ResolvedCharMap:      map[rune]string{'r': "R"},
	}

	requireExactOutputAndShortWrite(t, []byte(`<?xml veRsion="1.0" encoding="UTF-8"?><Root/>`), 1, func(w io.Writer) error {
		return xslt3.SerializeResult(w, doc, outDef)
	})
}

func TestSerializeResultXHTMLReportsShortWrite(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>alpha</root>`))
	require.NoError(t, err)
	includeContentType := false
	outDef := &xslt3.OutputDef{
		Method:             outMethodXHTML,
		OmitDeclaration:    true,
		IncludeContentType: &includeContentType,
	}

	requireExactOutputAndShortWrite(t, []byte(`<root>alpha</root>`), 1, func(w io.Writer) error {
		return xslt3.SerializeResult(w, doc, outDef)
	})
}

func TestSerializeResultTextReportsShortWrite(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>alpha</root>`))
	require.NoError(t, err)

	for _, tc := range []struct {
		name    string
		want    string
		charMap map[rune]string
	}{
		{
			name: "Content",
			want: "alpha",
		},
		{
			name:    "CharacterMap",
			want:    "[a]lph[a]",
			charMap: map[rune]string{'a': "[a]"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outDef := &xslt3.OutputDef{
				Method:          "text",
				ResolvedCharMap: tc.charMap,
			}
			requireExactOutputAndShortWrite(t, []byte(tc.want), 1, func(w io.Writer) error {
				return xslt3.SerializeResult(w, doc, outDef)
			})
		})
	}
}

const normalizationFormNFC = "NFC"

func TestCharacterMapReplacementSkipsNormalization(t *testing.T) {
	decomposed := "e\u0301"
	composed := "é"
	doc, err := helium.NewParser().Parse(t.Context(), []byte("<out>x"+decomposed+"</out>"))
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, xslt3.SerializeResult(&out, doc, &xslt3.OutputDef{
		Method:            outMethodXML,
		OmitDeclaration:   true,
		NormalizationForm: normalizationFormNFC,
		ResolvedCharMap:   map[rune]string{'x': decomposed},
	}))

	require.Equal(t, "<out>"+decomposed+composed+"</out>", out.String())
}

func TestCharacterMapNormalizationKeepsRawDOEContent(t *testing.T) {
	doc := helium.NewDefaultDocument()
	root, err := doc.CreateElement("out")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	// Build the historical metadata-shaped raw bytes without keeping the old
	// protocol spelling as a source literal.
	rawStart := string([]byte{0, 'C', 'M', 'S', 'T', 'A', 'R', 'T', 0})
	rawEnd := string([]byte{0, 'C', 'M', 'E', 'N', 'D', 0})
	decomposed := "e\u0301"
	composed := "é"
	require.NoError(t, root.AddChild(doc.CreatePI("disable-output-escaping", "")))
	require.NoError(t, root.AddChild(doc.CreateText([]byte(rawStart+decomposed+rawEnd))))
	require.NoError(t, root.AddChild(doc.CreatePI("enable-output-escaping", "")))
	require.NoError(t, root.AddChild(doc.CreateText([]byte("x"))))

	var out bytes.Buffer
	require.NoError(t, xslt3.SerializeResult(&out, doc, &xslt3.OutputDef{
		Method:            outMethodXML,
		OmitDeclaration:   true,
		NormalizationForm: normalizationFormNFC,
		ResolvedCharMap:   map[rune]string{'x': decomposed},
	}))

	// Caller-provided raw content is ordinary output and normalizes to NFC.
	// Only the character-map replacement remains decomposed.
	require.Equal(t, "<out>"+rawStart+composed+rawEnd+decomposed+"</out>", out.String())
}

func TestCharacterMapNormalizationPreservesRawDOEMarkup(t *testing.T) {
	doc := helium.NewDefaultDocument()
	root, err := doc.CreateElement("out")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	decomposed := "e\u0301"
	composed := "é"
	raw := `<inner a="` + decomposed + `">` + decomposed + `</inner>`
	require.NoError(t, root.AddChild(doc.CreatePI("disable-output-escaping", "")))
	require.NoError(t, root.AddChild(doc.CreateText([]byte(raw))))
	require.NoError(t, root.AddChild(doc.CreatePI("enable-output-escaping", "")))

	var out bytes.Buffer
	require.NoError(t, xslt3.SerializeResult(&out, doc, &xslt3.OutputDef{
		Method:            outMethodXML,
		OmitDeclaration:   true,
		NormalizationForm: normalizationFormNFC,
		ResolvedCharMap:   map[rune]string{'x': "x"},
	}))

	require.Equal(t, `<out><inner a="`+composed+`">`+composed+`</inner></out>`, out.String())
}

func TestTextCharacterMapNormalizationCrossesOmittedPI(t *testing.T) {
	composed := "é"
	doc, err := helium.NewParser().Parse(t.Context(), []byte("<out>e<?split?>\u0301</out>"))
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, xslt3.SerializeResult(&out, doc, &xslt3.OutputDef{
		Method:            "text",
		NormalizationForm: normalizationFormNFC,
		ResolvedCharMap:   map[rune]string{'x': "unused"},
	}))

	require.Equal(t, composed, out.String())
}

// TestXMLOutputNeverEmitsXmlnsXml verifies the xslt3 XML output method never
// serializes a redundant xmlns:xml="http://www.w3.org/XML/1998/namespace"
// declaration on a literal result element — the "xml" prefix is predefined by
// the Namespaces in XML spec and bound implicitly everywhere. The regression
// surfaced on literal result elements produced by an imported module's named
// template. A genuine namespace declaration (xmlns:foo) and an xml:lang
// attribute must still serialize normally.
func TestXMLOutputNeverEmitsXmlnsXml(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// The imported module's named template emits a literal result element. In
	// the buggy path the LRE picked up a spurious xml-prefix namespace
	// declaration from the module's in-scope bindings.
	const layoutModule = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">
  <xsl:template name="page-header">
    <header xmlns:foo="urn:example:foo" xml:lang="en">
      <h1>Title</h1>
    </header>
  </xsl:template>
</xsl:stylesheet>`

	const main = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:import href="layout.xsl"/>
  <xsl:output method="xml" indent="no" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <page>
      <xsl:call-template name="page-header"/>
    </page>
  </xsl:template>
</xsl:stylesheet>`

	resolver := &recordingCompileResolver{files: map[string][]byte{
		"mem:/styles/layout.xsl": []byte(layoutModule),
	}}

	doc, err := helium.NewParser().Parse(ctx, []byte(main))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		BaseURI("mem:/styles/main.xsl").
		URIResolver(resolver).
		Compile(ctx, doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)

	out, err := xslt3.TransformString(ctx, source, ss)
	require.NoError(t, err)

	require.NotContains(t, out, "xmlns:xml=",
		"redundant xmlns:xml declaration must never be serialized; got %q", out)
	require.Contains(t, out, `xmlns:foo="urn:example:foo"`,
		"a genuine namespace declaration must still serialize; got %q", out)
	require.Contains(t, out, `xml:lang="en"`,
		"an xml:lang attribute must still serialize; got %q", out)
	require.True(t, strings.Contains(out, "<header"), "got %q", out)
}

// BenchmarkSerializeResultXML measures the default XML-method serialization
// path (version unset → XML 1.0) over a document with many text nodes. It
// guards against reintroducing an extra whole-tree traversal for XML-version
// character validation: the SERE0006 check is folded into the writer's escape
// pass, so this benchmark should match the plain-serialization fast path.
func BenchmarkSerializeResultXML(b *testing.B) {
	const textNodes = 20000
	var sb strings.Builder
	sb.WriteString("<root>")
	for range textNodes {
		sb.WriteString("<item>lorem ipsum dolor sit amet consectetur</item>")
	}
	sb.WriteString("</root>")

	doc, err := helium.NewParser().Parse(b.Context(), []byte(sb.String()))
	if err != nil {
		b.Fatalf("parse: %v", err)
	}
	outDef := &xslt3.OutputDef{Method: outMethodXML}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := xslt3.SerializeResult(io.Discard, doc, outDef); err != nil {
			b.Fatalf("serialize: %v", err)
		}
	}
}
