package helium_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestXMLCharValidation(t *testing.T) {
	t.Parallel()

	// character-validity enforcement across the
	// scan paths that previously missed it: CDATA sections never validated the
	// XML Char production, and the slow attribute/comment/PI paths rejected a
	// valid U+FFFD because they treated every utf8.RuneError as invalid without
	// distinguishing genuinely-invalid UTF-8 (width 1) from a real U+FFFD
	// (width 3).
	t.Run("validation", func(t *testing.T) {
		t.Run("cdata with U+0001 is rejected", func(t *testing.T) {
			t.Parallel()
			data := []byte("<root><![CDATA[" + "\x01" + "]]></root>")
			_, err := helium.NewParser().Parse(t.Context(), data)
			require.Error(t, err)
		})

		t.Run("cdata with invalid UTF-8 byte 0xFF is rejected", func(t *testing.T) {
			t.Parallel()
			data := []byte("<root><![CDATA[" + "\xff" + "]]></root>")
			_, err := helium.NewParser().Parse(t.Context(), data)
			require.Error(t, err)
		})

		t.Run("cdata with valid content is accepted", func(t *testing.T) {
			t.Parallel()
			data := []byte("<root><![CDATA[ok]]></root>")
			_, err := helium.NewParser().Parse(t.Context(), data)
			require.NoError(t, err)
		})

		t.Run("comment with U+FFFD is accepted", func(t *testing.T) {
			t.Parallel()
			// U+FFFD (EF BF BD) is a valid XML Char and must survive the slow
			// comment scan path.
			data := []byte("<root><!--" + "\uFFFD" + "--></root>")
			_, err := helium.NewParser().Parse(t.Context(), data)
			require.NoError(t, err)
		})

		t.Run("comment with U+0001 is rejected", func(t *testing.T) {
			t.Parallel()
			data := []byte("<root><!--" + "\x01" + "--></root>")
			_, err := helium.NewParser().Parse(t.Context(), data)
			require.Error(t, err)
		})

		t.Run("slow-path attribute value with U+FFFD is accepted", func(t *testing.T) {
			t.Parallel()
			// The entity reference forces the slow attribute-value scan path
			// (the fast scanner bails on '&'). U+FFFD in that value is valid.
			data := []byte(`<root a="x&amp;` + "\uFFFD" + `"></root>`)
			_, err := helium.NewParser().Parse(t.Context(), data)
			require.NoError(t, err)
		})
	})

	// the width-aware U+FFFD handling
	// in the PI and DTD entity-value scan paths.
	t.Run("other slow paths", func(t *testing.T) {
		t.Run("PI content with U+FFFD is accepted", func(t *testing.T) {
			_, err := helium.NewParser().Parse(t.Context(), []byte("<root><?pi a\uFFFDb?></root>"))
			require.NoError(t, err)
		})

		t.Run("PI content with U+0001 is rejected", func(t *testing.T) {
			_, err := helium.NewParser().Parse(t.Context(), []byte("<root><?pi a\x01b?></root>"))
			require.Error(t, err)
		})

		t.Run("entity declaration value with U+FFFD is accepted", func(t *testing.T) {
			// The entity-declaration value scanner must accept the valid U+FFFD.
			doc := "<!DOCTYPE root [<!ENTITY e \"a\uFFFDb\">]><root/>"
			_, err := helium.NewParser().Parse(t.Context(), []byte(doc))
			require.NoError(t, err)
		})

		t.Run("PUBLIC pubid literal with U+FFFD is rejected", func(t *testing.T) {
			// U+FFFD is a valid XML Char but not a PubidChar, so a pubid
			// literal containing it must be rejected.
			doc := "<!DOCTYPE root PUBLIC \"\uFFFD\" \"sys\"><root/>"
			_, err := helium.NewParser().Parse(t.Context(), []byte(doc))
			require.Error(t, err)
		})

		t.Run("PUBLIC pubid literal with valid PubidChars is accepted", func(t *testing.T) {
			doc := "<!DOCTYPE root PUBLIC \"-//W3C//DTD//EN\" \"sys\"><root/>"
			_, err := helium.NewParser().Parse(t.Context(), []byte(doc))
			require.NoError(t, err)
		})
	})

	// U+FFFD (a valid XML NameStartChar/NameChar)
	// is accepted in element and attribute names, while genuinely-invalid UTF-8 in
	// a name is still rejected.
	t.Run("an invalid UTF-8 name yields U+FFFD", func(t *testing.T) {
		for _, in := range []string{
			"<\uFFFD/>",             // element name starting with U+FFFD
			"<a\uFFFD/>",            // U+FFFD inside element name
			"<root \uFFFD=\"v\"/>",  // attr name starting with U+FFFD
			"<root x\uFFFD=\"v\"/>", // U+FFFD inside attr name (ASCII-first fast path)
		} {
			_, err := helium.NewParser().Parse(t.Context(), []byte(in))
			require.NoError(t, err, "valid U+FFFD in name must parse: %q", in)
		}
		_, err := helium.NewParser().Parse(t.Context(), []byte{'<', 0xFF, '/', '>'})
		require.Error(t, err, "invalid UTF-8 lead byte in a name must be rejected")
	})

	// XML-forbidden Unicode scalars in
	// text content (XML 1.0 §2.2 Char production) are rejected by the parser,
	// while valid characters in the same neighborhood still parse.
	t.Run("non-XML characters are rejected", func(t *testing.T) {
		invalid := []struct {
			name string
			r    rune
		}{
			{"U+FFFE", 0xFFFE},
			{"U+FFFF", 0xFFFF},
		}
		for _, tt := range invalid {
			t.Run("invalid "+tt.name, func(t *testing.T) {
				t.Parallel()
				input := "<r>" + string(tt.r) + "</r>"
				p := helium.NewParser()
				_, err := p.Parse(t.Context(), []byte(input))
				require.Error(t, err, "parsing forbidden char %s must fail", tt.name)
			})
		}

		valid := []struct {
			name string
			r    rune
		}{
			{"U+009F", 0x009F},     // C1 control, but a valid XML Char
			{"U+E000", 0xE000},     // first after surrogate range
			{"U+FFFD", 0xFFFD},     // replacement char — valid XML Char, decodes as RuneError
			{"U+10FFFF", 0x10FFFF}, // last valid code point
			{"U+1FFFE", 0x1FFFE},   // non-character per Unicode, but valid XML Char
		}
		for _, tt := range valid {
			t.Run("valid "+tt.name, func(t *testing.T) {
				t.Parallel()
				input := "<r>" + string(tt.r) + "</r>"
				p := helium.NewParser()
				_, err := p.Parse(t.Context(), []byte(input))
				require.NoError(t, err, "parsing valid char %s must succeed", tt.name)
			})
		}
	})

	// the attribute-value fast path, which
	// must reject XML-forbidden chars just like text content.
	t.Run("non-XML characters in an attribute are rejected", func(t *testing.T) {
		for _, r := range []rune{0xFFFE, 0xFFFF} {
			input := `<r a="` + string(r) + `"/>`
			p := helium.NewParser()
			_, err := p.Parse(t.Context(), []byte(input))
			require.Error(t, err, "forbidden char U+%04X in attribute value must fail", r)
		}
		// A valid multibyte char in an attribute must still parse.
		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(`<r a="`+string(rune(0x4E2D))+`"/>`))
		require.NoError(t, err)
	})
}

func TestParseAttrValue(t *testing.T) {
	t.Parallel()

	// the attribute-value slow path
	// (parseAttributeValueInternal). The slow path is forced by including an
	// entity reference or a tab (which needs whitespace normalization) in the
	// same attribute. A real U+FFFD (valid XML Char, encoded as 3-byte UTF-8)
	// must parse, while XML-forbidden chars must still be rejected.
	t.Run("slow-path XML characters", func(t *testing.T) {
		// Triggers that force the slow path: an entity ref and a normalizable tab.
		triggers := []struct {
			name string
			// before/after wrap the test char in the attribute value.
			before string
			after  string
		}{
			{"entity-after", "", "&amp;"},
			{"entity-before", "&amp;", ""},
			{"tab-after", "", "\tx"},
		}

		for _, tr := range triggers {
			t.Run("valid U+FFFD "+tr.name, func(t *testing.T) {
				t.Parallel()
				input := `<r a="` + tr.before + string(rune(0xFFFD)) + tr.after + `"/>`
				_, err := helium.NewParser().Parse(t.Context(), []byte(input))
				require.NoError(t, err, "real U+FFFD on slow path must parse (%s)", tr.name)
			})
		}

		t.Run("invalid characters", func(t *testing.T) {
			t.Parallel()
			for _, r := range []rune{0xFFFE, 0xFFFF} {
				input := `<r a="` + string(r) + `&amp;"/>`
				_, err := helium.NewParser().Parse(t.Context(), []byte(input))
				require.Error(t, err, "forbidden char U+%04X on slow path must fail", r)
			}
		})
	})

	// the attribute-value fast path
	// normalizes a literal tab to a space (XML 1.0 §3.3.3), matching newline/CR.
	t.Run("whitespace normalization", func(t *testing.T) {
		for _, ws := range []string{"\t", "\n", "\r"} {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r a="x`+ws+`y"/>`))
			require.NoError(t, err)
			attrs := doc.DocumentElement().Attributes()
			require.Len(t, attrs, 1)
			require.Equal(t, "x y", attrs[0].Value(), "whitespace %q must normalize to space", ws)
		}
	})

	// An attribute value holding a C0 control, DEL, a C1 control, or another
	// non-ASCII character, after ASCII runs of lengths around the 16-byte scan
	// stride, in both XML versions and both quote styles. XML 1.0 accepts every
	// Char literally; XML 1.1 rejects its RestrictedChar (C0 controls other than
	// tab, LF and CR, DEL, and U+0080-U+0084, U+0086-U+009F).
	t.Run("XML 1.0 and 1.1 characters", func(t *testing.T) {
		t.Parallel()

		const run = "abcdefghijklmnopqrstuvwxyz0123456789"
		for _, tc := range attrCharCases {
			for _, version := range []string{"1.0", ver11} {
				wantErr := tc.err10
				if version == ver11 {
					wantErr = tc.err11
				}
				for _, n := range []int{0, 1, 7, 8, 15, 16, 17, 31, 32, 33} {
					for _, quote := range []string{`"`, `'`} {
						value := run[:n] + string(tc.r) + "z"
						decl := `<?xml version="` + version + `"?>`
						src := decl + `<r a=` + quote + value + quote + `/>`
						doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
						if wantErr {
							// A C0 control ends the fast scan and the slow path stops
							// at it, so the value is reported unclosed where the
							// control sits; an XML 1.1 RestrictedChar is reported
							// at the start of the value. The column and the context
							// line count the XML declaration on the same line.
							want := fmt.Sprintf("invalid char at line 1, column %d\n -> '%s<r a=%s' <-- around here",
								len(decl)+7, decl, quote)
							if tc.r < 0x20 {
								want = fmt.Sprintf("string not closed at line 1, column %d\n -> '%s<r a=%s%s' <-- around here",
									len(decl)+7+n, decl, quote, run[:n])
							}
							require.EqualError(t, err, want, "U+%04X, XML %s, %d leading bytes", tc.r, version, n)
							continue
						}
						require.NoError(t, err, "U+%04X, XML %s, %d leading bytes", tc.r, version, n)
						attrs := doc.DocumentElement().Attributes()
						require.Len(t, attrs, 1)
						require.Equal(t, value, attrs[0].Value(), "U+%04X, XML %s, %d leading bytes", tc.r, version, n)
					}
				}
			}
		}
	})

	// The tree the parser builds for each attribute value: one Text child for
	// a value without references, the Text/EntityRef list for a value whose
	// entity references are kept, no child for an empty value, the namespace of
	// a prefixed attribute, and the default and type flags from the DTD.
	t.Run("node shape", func(t *testing.T) {
		t.Parallel()

		for _, tc := range attrShapeCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				p := helium.NewParser().SubstituteEntities(tc.substitute).DefaultDTDAttributes(tc.defaults)
				doc, err := p.Parse(t.Context(), []byte(tc.src))
				require.NoError(t, err)
				require.Equal(t, tc.want, describeAttributes(doc))
				for _, id := range tc.ids {
					require.NotNil(t, doc.GetElementByID(id), "ID %q registered", id)
				}
			})
		}
	})
}

// attrCharCases lists characters whose literal use in an attribute value
// differs between XML 1.0 and XML 1.1, with whether each version rejects it.
var attrCharCases = []struct {
	r     rune
	err10 bool
	err11 bool
}{
	{r: 0x01, err10: true, err11: true},
	{r: 0x08, err10: true, err11: true},
	{r: 0x0B, err10: true, err11: true},
	{r: 0x1F, err10: true, err11: true},
	{r: 0x7E},
	{r: 0x7F, err11: true},
	{r: 0x80, err11: true},
	{r: 0x84, err11: true},
	{r: 0x85},
	{r: 0x86, err11: true},
	{r: 0x9F, err11: true},
	{r: 0xA0},
	{r: 0xE9},
	{r: 0x2028},
	{r: 0xFFFD},
	{r: 0x1F600},
}

// attrShapeNoDTD and attrShapeDTD are documents whose attributes cover the
// values the parser's tree fast path builds differently: with and without '&',
// with entity and character references, empty, holding the other quote,
// prefixed, defaulted from the DTD, and ID-typed.
const (
	attrShapeNoDTD = `<r xmlns:p="urn:p" plain="abc" amp="a&amp;b" lt="a&lt;b" cref="&#65;z&#x42;" p:pre="pv" ` +
		`p:amp="1&amp;2" empty="" p:empty="" dq='say "hi"' sq="it's" ` +
		`long="0123456789abcdefghijklmnopqrstuvwxyz" u="é中😀"><c p:x="y" z=""/></r>`
	attrShapeDTD = `<!DOCTYPE r [
<!ENTITY e "ent">
<!ATTLIST r d CDATA "def&e;x" p:d CDATA "pd" id ID #IMPLIED plain CDATA "dflt">
<!ATTLIST c xml:id ID #IMPLIED>
]>
<r xmlns:p="urn:p" ref="x&e;y" p:ref="1&e;2" id="i1" mixed="&e;&amp;&e;" plain="abc"><c xml:id="c1" v="w"/></r>`
)

// attrShapeCases pairs each attribute-shape document and parser setting with
// describeAttributes' rendering of the tree the default parser builds.
var attrShapeCases = []struct {
	name       string
	src        string
	substitute bool
	defaults   bool
	ids        []string
	want       string
}{
	{name: "no DTD", src: attrShapeNoDTD, want: `<r>
  plain prefix="" uri="" default=false atype=0 value="abc" children=[ 3:"abc" ]
  amp prefix="" uri="" default=false atype=0 value="a&b" children=[ 3:"a&b" ]
  lt prefix="" uri="" default=false atype=0 value="a<b" children=[ 3:"a<b" ]
  cref prefix="" uri="" default=false atype=0 value="AzB" children=[ 3:"AzB" ]
  p:pre prefix="p" uri="urn:p" default=false atype=0 value="pv" children=[ 3:"pv" ]
  p:amp prefix="p" uri="urn:p" default=false atype=0 value="1&2" children=[ 3:"1&2" ]
  empty prefix="" uri="" default=false atype=0 value="" children=[ ]
  p:empty prefix="p" uri="urn:p" default=false atype=0 value="" children=[ ]
  dq prefix="" uri="" default=false atype=0 value="say \"hi\"" children=[ 3:"say \"hi\"" ]
  sq prefix="" uri="" default=false atype=0 value="it's" children=[ 3:"it's" ]
  long prefix="" uri="" default=false atype=0 value="0123456789abcdefghijklmnopqrstuvwxyz" children=[ 3:"0123456789abcdefghijklmnopqrstuvwxyz" ]
  u prefix="" uri="" default=false atype=0 value="é中😀" children=[ 3:"é中😀" ]
<c>
  p:x prefix="p" uri="urn:p" default=false atype=0 value="y" children=[ 3:"y" ]
  z prefix="" uri="" default=false atype=0 value="" children=[ ]
`},
	{name: "no DTD substituted", src: attrShapeNoDTD, substitute: true, want: `<r>
  plain prefix="" uri="" default=false atype=0 value="abc" children=[ 3:"abc" ]
  amp prefix="" uri="" default=false atype=0 value="a&b" children=[ 3:"a&b" ]
  lt prefix="" uri="" default=false atype=0 value="a<b" children=[ 3:"a<b" ]
  cref prefix="" uri="" default=false atype=0 value="AzB" children=[ 3:"AzB" ]
  p:pre prefix="p" uri="urn:p" default=false atype=0 value="pv" children=[ 3:"pv" ]
  p:amp prefix="p" uri="urn:p" default=false atype=0 value="1&2" children=[ 3:"1&2" ]
  empty prefix="" uri="" default=false atype=0 value="" children=[ ]
  p:empty prefix="p" uri="urn:p" default=false atype=0 value="" children=[ ]
  dq prefix="" uri="" default=false atype=0 value="say \"hi\"" children=[ 3:"say \"hi\"" ]
  sq prefix="" uri="" default=false atype=0 value="it's" children=[ 3:"it's" ]
  long prefix="" uri="" default=false atype=0 value="0123456789abcdefghijklmnopqrstuvwxyz" children=[ 3:"0123456789abcdefghijklmnopqrstuvwxyz" ]
  u prefix="" uri="" default=false atype=0 value="é中😀" children=[ 3:"é中😀" ]
<c>
  p:x prefix="p" uri="urn:p" default=false atype=0 value="y" children=[ 3:"y" ]
  z prefix="" uri="" default=false atype=0 value="" children=[ ]
`},
	{name: "DTD", src: attrShapeDTD, ids: []string{"i1", "c1"}, want: `<r>
  ref prefix="" uri="" default=false atype=0 value="xenty" children=[ 3:"x" 5:"ent" 3:"y" ]
  p:ref prefix="p" uri="urn:p" default=false atype=0 value="1ent2" children=[ 3:"1" 5:"ent" 3:"2" ]
  id prefix="" uri="" default=false atype=2 value="i1" children=[ 3:"i1" ]
  mixed prefix="" uri="" default=false atype=0 value="ent&ent" children=[ 5:"ent" 3:"&" 5:"ent" ]
  plain prefix="" uri="" default=false atype=1 value="abc" children=[ 3:"abc" ]
<c>
  xml:id prefix="xml" uri="http://www.w3.org/XML/1998/namespace" default=false atype=2 value="c1" children=[ 3:"c1" ]
  v prefix="" uri="" default=false atype=0 value="w" children=[ 3:"w" ]
`},
	{name: "DTD defaulted", src: attrShapeDTD, defaults: true, ids: []string{"i1", "c1"}, want: `<r>
  ref prefix="" uri="" default=false atype=0 value="xenty" children=[ 3:"x" 5:"ent" 3:"y" ]
  p:ref prefix="p" uri="urn:p" default=false atype=0 value="1ent2" children=[ 3:"1" 5:"ent" 3:"2" ]
  id prefix="" uri="" default=false atype=2 value="i1" children=[ 3:"i1" ]
  mixed prefix="" uri="" default=false atype=0 value="ent&ent" children=[ 5:"ent" 3:"&" 5:"ent" ]
  plain prefix="" uri="" default=false atype=1 value="abc" children=[ 3:"abc" ]
  d prefix="" uri="" default=true atype=1 value="defentx" children=[ 3:"def" 5:"ent" 3:"x" ]
  p:d prefix="p" uri="urn:p" default=true atype=1 value="pd" children=[ 3:"pd" ]
<c>
  xml:id prefix="xml" uri="http://www.w3.org/XML/1998/namespace" default=false atype=2 value="c1" children=[ 3:"c1" ]
  v prefix="" uri="" default=false atype=0 value="w" children=[ 3:"w" ]
`},
	{name: "DTD substituted", src: attrShapeDTD, substitute: true, ids: []string{"i1", "c1"}, want: `<r>
  ref prefix="" uri="" default=false atype=0 value="xenty" children=[ 3:"xenty" ]
  p:ref prefix="p" uri="urn:p" default=false atype=0 value="1ent2" children=[ 3:"1ent2" ]
  id prefix="" uri="" default=false atype=2 value="i1" children=[ 3:"i1" ]
  mixed prefix="" uri="" default=false atype=0 value="ent&ent" children=[ 3:"ent&ent" ]
  plain prefix="" uri="" default=false atype=1 value="abc" children=[ 3:"abc" ]
<c>
  xml:id prefix="xml" uri="http://www.w3.org/XML/1998/namespace" default=false atype=2 value="c1" children=[ 3:"c1" ]
  v prefix="" uri="" default=false atype=0 value="w" children=[ 3:"w" ]
`},
	{name: "DTD substituted and defaulted", src: attrShapeDTD, substitute: true, defaults: true, ids: []string{"i1", "c1"},
		want: `<r>
  ref prefix="" uri="" default=false atype=0 value="xenty" children=[ 3:"xenty" ]
  p:ref prefix="p" uri="urn:p" default=false atype=0 value="1ent2" children=[ 3:"1ent2" ]
  id prefix="" uri="" default=false atype=2 value="i1" children=[ 3:"i1" ]
  mixed prefix="" uri="" default=false atype=0 value="ent&ent" children=[ 3:"ent&ent" ]
  plain prefix="" uri="" default=false atype=1 value="abc" children=[ 3:"abc" ]
  d prefix="" uri="" default=true atype=1 value="defentx" children=[ 3:"defentx" ]
  p:d prefix="p" uri="urn:p" default=true atype=1 value="pd" children=[ 3:"pd" ]
<c>
  xml:id prefix="xml" uri="http://www.w3.org/XML/1998/namespace" default=false atype=2 value="c1" children=[ 3:"c1" ]
  v prefix="" uri="" default=false atype=0 value="w" children=[ 3:"w" ]
`},
}

// describeAttributes renders every attribute in doc's element tree: its name,
// namespace, default and type flags, value, and the node list holding the
// value.
func describeAttributes(doc *helium.Document) string {
	var sb strings.Builder
	for n := range helium.Descendants(doc) {
		elem, ok := n.(*helium.Element)
		if !ok {
			continue
		}
		fmt.Fprintf(&sb, "<%s>\n", elem.Name())
		for _, attr := range elem.Attributes() {
			fmt.Fprintf(&sb, "  %s prefix=%q uri=%q default=%t atype=%d value=%q children=[",
				attr.Name(), attr.Prefix(), attr.URI(), attr.IsDefault(), attr.AType(), attr.Value())
			for child := range helium.Children(attr) {
				fmt.Fprintf(&sb, " %d:%q", child.Type(), child.Content())
			}
			sb.WriteString(" ]\n")
		}
	}
	return sb.String()
}
