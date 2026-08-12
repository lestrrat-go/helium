package helium_test

import (
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
			t.Run("invalid/"+tt.name, func(t *testing.T) {
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
			t.Run("valid/"+tt.name, func(t *testing.T) {
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

		t.Run("valid-U+FFFD", func(t *testing.T) {
			t.Parallel()
			for _, tr := range triggers {
				t.Run(tr.name, func(t *testing.T) {
					t.Parallel()
					input := `<r a="` + tr.before + string(rune(0xFFFD)) + tr.after + `"/>`
					_, err := helium.NewParser().Parse(t.Context(), []byte(input))
					require.NoError(t, err, "real U+FFFD on slow path must parse (%s)", tr.name)
				})
			}
		})

		t.Run("invalid", func(t *testing.T) {
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
}
