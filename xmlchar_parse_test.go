package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestParseRejectsNonXMLChars verifies that XML-forbidden Unicode scalars in
// text content (XML 1.0 §2.2 Char production) are rejected by the parser,
// while valid characters in the same neighborhood still parse.
func TestParseRejectsNonXMLChars(t *testing.T) {
	t.Parallel()

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
}

// TestParseRejectsNonXMLCharsInAttr covers the attribute-value fast path, which
// must reject XML-forbidden chars just like text content.
func TestParseRejectsNonXMLCharsInAttr(t *testing.T) {
	t.Parallel()

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
}

// TestParseAttrSlowPathXMLChars covers the attribute-value slow path
// (parseAttributeValueInternal). The slow path is forced by including an
// entity reference or a tab (which needs whitespace normalization) in the
// same attribute. A real U+FFFD (valid XML Char, encoded as 3-byte UTF-8)
// must parse, while XML-forbidden chars must still be rejected.
func TestParseAttrSlowPathXMLChars(t *testing.T) {
	t.Parallel()

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
}

// TestParseAttrWhitespaceNormalization checks the attribute-value fast path
// normalizes a literal tab to a space (XML 1.0 §3.3.3), matching newline/CR.
func TestParseAttrWhitespaceNormalization(t *testing.T) {
	t.Parallel()

	for _, ws := range []string{"\t", "\n", "\r"} {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r a="x`+ws+`y"/>`))
		require.NoError(t, err)
		attrs := doc.DocumentElement().Attributes()
		require.Len(t, attrs, 1)
		require.Equal(t, "x y", attrs[0].Value(), "whitespace %q must normalize to space", ws)
	}
}
