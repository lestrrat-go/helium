package xslt3

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

// TestNormalizeXMLContentCDATA verifies that Unicode normalization is applied
// to the contents of a CDATA section while the surrounding markup is preserved
// verbatim. The decomposed form of "e" + combining acute accent
// (U+0065 U+0301) inside a CDATA section must be normalized to the composed
// form "é" (U+00E9) under NFC.
func TestNormalizeXMLContentCDATA(t *testing.T) {
	decomposed := "é" // "e" + combining acute accent
	composed := "é"    // precomposed "é"

	input := []byte("<e><![CDATA[" + decomposed + "]]></e>")
	want := "<e><![CDATA[" + composed + "]]></e>"

	got := string(normalizeXMLContent(input, norm.NFC))
	require.Equal(t, want, got, "CDATA section content should be normalized")
}

// invalidNameElement builds an element whose name is not a well-formed XML
// name. CreateElement accepts it, but the writer rejects it at emit time with
// ErrWriterInvalidElementName, giving a node whose serialization must fail.
func invalidNameElement(t *testing.T) *helium.Element {
	t.Helper()
	doc := helium.NewDefaultDocument()
	el, err := doc.CreateElement(`bad name="x"`)
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(el))
	return el
}

// serializeNodeWithMethod must surface a serialization failure, and must never
// discard it. Each output method whose branch drives a serializer (html, xhtml,
// and the default xml path) must return the serializer's error for an
// unserializable node. The xhtml and xml paths run the core writer, whose
// failure is the matchable ErrWriterInvalidElementName sentinel; the html path
// uses the html serializer, which reports its own (unmatched) error.
func TestSerializeNodeWithMethodPropagatesWriterError(t *testing.T) {
	for _, tc := range []struct {
		method       string
		writerBacked bool
	}{
		{methodHTML, false},
		{methodXHTML, true},
		{"xml", true},
		{"", true},
	} {
		t.Run(tc.method, func(t *testing.T) {
			el := invalidNameElement(t)
			_, err := serializeNodeWithMethod(el, tc.method, "")
			require.Error(t, err, "serializeNodeWithMethod must not discard the serializer error")
			if tc.writerBacked {
				require.ErrorIs(t, err, helium.ErrWriterInvalidElementName)
			}
		})
	}
}

// The json-node-output-method path routes a node through serializeNodeWithMethod;
// a serialization failure there must propagate out of serializeItemJSON instead
// of yielding a silently truncated JSON string.
func TestSerializeItemJSONPropagatesNodeSerializationError(t *testing.T) {
	el := invalidNameElement(t)
	outDef := &OutputDef{JSONNodeOutputMethod: methodXHTML}
	_, err := serializeItemJSON(xpath3.NodeItem{Node: el}, outDef)
	require.Error(t, err, "serializeItemJSON must surface the node serialization error")
	require.ErrorIs(t, err, helium.ErrWriterInvalidElementName)
}

// serializeMessageSequence backs xsl:message rendering; an unserializable node
// in the message body must return the writer error so execMessage can raise a
// dynamic error instead of reporting silent success.
func TestSerializeMessageSequencePropagatesWriterError(t *testing.T) {
	el := invalidNameElement(t)
	_, err := serializeMessageSequence(xpath3.ItemSlice{xpath3.NodeItem{Node: el}})
	require.Error(t, err, "serializeMessageSequence must not discard the writer error")
	require.ErrorIs(t, err, helium.ErrWriterInvalidElementName)
}

func TestFrenchCardinalTens(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{40, "quarante"},
		{41, "quarante et un"},
		{42, "quarante-deux"},
		{45, "quarante-cinq"},
		{49, "quarante-neuf"},
		{71, "soixante et onze"},
		{80, "quatre-vingts"},
		{91, "quatre-vingt-onze"},
	}
	for _, tc := range cases {
		if got := frenchCardinal(tc.n); got != tc.want {
			t.Errorf("frenchCardinal(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}
