package helium_test

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

// attrValueElements is the number of <v> elements attrValueDoc holds: enough
// attribute bytes to span many value chunks and many refills of a reader's
// buffer.
const attrValueElements = 3000

// attrValueDoc returns a document whose <v> elements carry attribute values
// of every kind the parser builds: a plain value, values with a predefined
// entity, a character reference and a DTD entity, a value with non-ASCII
// characters, a CDATA value with whitespace to normalize, and an NMTOKENS
// value with whitespace to collapse.
func attrValueDoc() []byte {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE r [
<!ENTITY e "ent">
<!ATTLIST v t NMTOKENS #IMPLIED>
]>
<r>
`)
	for i := range attrValueElements {
		n := strconv.Itoa(i)
		b.WriteString(`<v a="plain-` + n + `" b="x&amp;y-` + n + `" c="&#x41;&e;-` + n +
			`" d="é日-` + n + "\" w=\"t\tn\nx-" + n + `" t="  p   q-` + n + `  "/>` + "\n")
	}
	b.WriteString("</r>\n")
	return []byte(b.String())
}

// wantAttrValues returns the attribute values of the i-th <v> element, in
// document order, as "name=value".
func wantAttrValues(i int) []string {
	n := strconv.Itoa(i)
	return []string{
		"a=plain-" + n,
		"b=x&y-" + n,
		"c=Aent-" + n,
		"d=é日-" + n,
		"w=t n x-" + n,
		"t=p q-" + n,
	}
}

// retainingHandler builds the DOM like the default handler and keeps every
// attribute value string the parser hands to StartElementNS, without copying
// it.
type retainingHandler struct {
	*helium.TreeBuilder
	values [][]string
}

func (h *retainingHandler) StartElementNS(ctx context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
	if localname == "v" {
		vals := make([]string, 0, len(attrs))
		for _, a := range attrs {
			vals = append(vals, a.Name()+"="+a.Value())
		}
		h.values = append(h.values, vals)
	}
	return h.TreeBuilder.StartElementNS(ctx, localname, prefix, uri, namespaces, attrs)
}

func requireAttrValues(t *testing.T, values [][]string) {
	t.Helper()
	require.Len(t, values, attrValueElements)
	for i, got := range values {
		require.Equal(t, wantAttrValues(i), got, "element %d", i)
	}
}

func requireDOMAttrValues(t *testing.T, doc *helium.Document) {
	t.Helper()
	i := 0
	for v := range helium.ChildElements(doc.DocumentElement()) {
		var got []string
		for a := range helium.Attributes(v) {
			got = append(got, a.Name()+"="+a.Value())
		}
		require.Equal(t, wantAttrValues(i), got, "element %d", i)
		i++
	}
	require.Equal(t, attrValueElements, i)
}

func TestParseAttributeValues(t *testing.T) {
	t.Parallel()

	input := attrValueDoc()
	parser := helium.NewParser().SubstituteEntities(true)

	// A SAX handler may keep the attribute value strings it is given. They
	// must still read the same once the parse has gone on to later values,
	// has finished, and other parses have run, whether the input arrives
	// whole, one byte per read, or through the push parser.
	t.Run("SAX handler keeps values", func(t *testing.T) {
		t.Parallel()

		parsers := []struct {
			name  string
			parse func(*testing.T, helium.Parser, []byte) (*helium.Document, error)
		}{
			{"whole input", parseWhole},
			{"ParseReader one byte per read", parseOneBytePerRead},
			{"push parser", parsePushed},
		}
		for _, tc := range parsers {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				h := &retainingHandler{TreeBuilder: helium.NewTreeBuilder()}
				doc, err := tc.parse(t, parser.SAXHandler(h), input)
				require.NoError(t, err)
				requireDOMAttrValues(t, doc)
				doc.Free()

				// Later parses reuse the freed document's slab chunks.
				for range 3 {
					other, err := parser.Parse(t.Context(), input)
					require.NoError(t, err)
					other.Free()
				}
				requireAttrValues(t, h.values)
			})
		}
	})

	// Parses running at once each build their own values; run under -race.
	t.Run("parallel parses", func(t *testing.T) {
		t.Parallel()

		for i := range 4 {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				t.Parallel()

				doc, err := parser.Parse(t.Context(), input)
				require.NoError(t, err)
				requireDOMAttrValues(t, doc)
				doc.Free()
			})
		}
	})

	// Appending to one attribute's value leaves the values parsed after it
	// untouched.
	t.Run("append to a value", func(t *testing.T) {
		t.Parallel()

		doc, err := parser.Parse(t.Context(), input)
		require.NoError(t, err)
		defer doc.Free()

		v := doc.DocumentElement().FirstChild()
		for v != nil && v.Type() != helium.ElementNode {
			v = v.NextSibling()
		}
		e, ok := v.(*helium.Element)
		require.True(t, ok)
		attrs := e.Attributes()
		require.NoError(t, attrs[0].AppendText([]byte("ZZZZ")))
		require.Equal(t, "plain-0ZZZZ", attrs[0].Value())
		var got []string
		for _, a := range attrs[1:] {
			got = append(got, a.Name()+"="+a.Value())
		}
		require.Equal(t, wantAttrValues(0)[1:], got)
	})
}

func parseWhole(t *testing.T, p helium.Parser, input []byte) (*helium.Document, error) {
	return p.Parse(t.Context(), input)
}

func parseOneBytePerRead(t *testing.T, p helium.Parser, input []byte) (*helium.Document, error) {
	return p.ParseReader(t.Context(), iotest.OneByteReader(bytes.NewReader(input)))
}

// parsePushed feeds input to a push parser 64 bytes at a time.
func parsePushed(t *testing.T, p helium.Parser, input []byte) (*helium.Document, error) {
	pp := p.NewPushParser(t.Context())
	for len(input) > 0 {
		n := min(64, len(input))
		if err := pp.Push(input[:n]); err != nil {
			return nil, err
		}
		input = input[n:]
	}
	return pp.Close()
}
