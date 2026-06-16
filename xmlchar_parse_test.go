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
		{"U+009F", 0x009F},   // C1 control, but a valid XML Char
		{"U+E000", 0xE000},   // first after surrogate range
		{"U+10FFFF", 0x10FFFF}, // last valid code point
		{"U+1FFFE", 0x1FFFE},  // non-character per Unicode, but valid XML Char
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
