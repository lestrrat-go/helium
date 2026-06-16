package xmlchar_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/stretchr/testify/require"
)

func TestIsValidNCName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"foo", true},
		{"_bar", true},
		{"café", true},
		{"a123", true},
		{"a.b", true},
		{"a-b", true},
		{"_", true},
		{"A", true},
		{"z", true},
		{"", false},
		{"1foo", false},
		{"foo:bar", false},
		{"-bar", false},
		{".bar", false},
		{" foo", false},
		{"foo bar", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, xmlchar.IsValidNCName(tt.input))
		})
	}
}

func TestIsValidPITarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"target", true},
		{"xml-stylesheet", true},
		{"_pi", true},
		{"café", true},
		{"�", true}, // genuinely encoded U+FFFD is a valid NCName char
		{"", false},
		{"xml", false},                // reserved (any case)
		{"XML", false},                // reserved (any case)
		{"Xml", false},                // reserved (any case)
		{"a:b", false},                // colons forbidden, matching the parser
		{":", false},                  // colons forbidden
		{"a:", false},                 // colons forbidden
		{":a", false},                 // colons forbidden
		{"1bad", false},               // must start with NCNameStartChar
		{string([]byte{0xff}), false}, // invalid UTF-8
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, xmlchar.IsValidPITarget(tt.input))
		})
	}
}

func TestIsChar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		r    rune
		want bool
	}{
		{0x8, false},      // backspace, control
		{0x9, true},       // tab
		{0xA, true},       // LF
		{0xD, true},       // CR
		{0x1F, false},     // unit separator, control
		{0x20, true},      // space
		{0xD7FF, true},    // last before surrogate range
		{0xD800, false},   // surrogate
		{0xDFFF, false},   // surrogate
		{0xE000, true},    // first after surrogate range
		{0xFFFD, true},    // replacement char (valid Char)
		{0xFFFE, false},   // non-character
		{0xFFFF, false},   // non-character
		{0x10000, true},   // first supplementary
		{0x10FFFF, true},  // last valid code point
		{0x110000, false}, // beyond Unicode range
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("U+%04X", tt.r), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, xmlchar.IsChar(tt.r), "IsChar(%#x)", tt.r)
		})
	}
}

func TestIsNCNameStartChar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		r    rune
		want bool
	}{
		{'A', true},
		{'z', true},
		{'_', true},
		{'\u00C0', true}, // Latin capital A with grave
		{'\u3001', true}, // CJK
		{'0', false},
		{'-', false},
		{'.', false},
		{':', false},
		{' ', false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("U+%04X", tt.r), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, xmlchar.IsNCNameStartChar(tt.r))
		})
	}
}

func TestIsNCNameChar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		r    rune
		want bool
	}{
		{'A', true},
		{'z', true},
		{'_', true},
		{'0', true},
		{'9', true},
		{'-', true},
		{'.', true},
		{'\u00B7', true}, // middle dot (combining)
		{':', false},
		{' ', false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("U+%04X", tt.r), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, xmlchar.IsNCNameChar(tt.r))
		})
	}
}
