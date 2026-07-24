package xslt3_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

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
