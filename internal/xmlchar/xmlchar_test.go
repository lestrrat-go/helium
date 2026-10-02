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

func TestIsValidEncName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"UTF-8", true},
		{"utf-8", true},
		{"ISO-8859-1", true},
		{"Shift_JIS", true},
		{"a", true},
		{"a1._-", true},
		{"", false},
		{"utf 8", false},
		{"1utf8", false},
		{".utf8", false},
		{"-utf8", false},
		{"_utf8", false},
		{"utf+8", false},
		{"utf8\"?><x/>", false},
		{"カタカナ", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, xmlchar.IsValidEncName(tt.input))
		})
	}
}

func TestIsValidQName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"foo", true},
		{"p:foo", true},
		{"xml:space", true},
		{"_a:_b", true},
		{"", false},
		{":foo", false},
		{"foo:", false},
		{"p:q:r", false},
		{"1p:foo", false},
		{"p:1foo", false},
		{`root injected="1"`, false},
		{"foo bar", false},
		{"x>y", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, xmlchar.IsValidQName(tt.input))
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

func TestIsLiteralChar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		r      rune
		want10 bool
		want11 bool
	}{
		{0x0, false, false},      // NUL: never a Char
		{0x1, false, false},      // C0: XML 1.1 Char, but RestrictedChar
		{0x9, true, true},        // tab
		{0xA, true, true},        // LF
		{0xB, false, false},      // C0
		{0xD, true, true},        // CR
		{0x1F, false, false},     // C0
		{0x20, true, true},       // space
		{0x7E, true, true},       // tilde
		{0x7F, true, false},      // DEL: RestrictedChar
		{0x80, true, false},      // C1: RestrictedChar
		{0x84, true, false},      // C1: RestrictedChar
		{0x85, true, true},       // NEL: not restricted
		{0x86, true, false},      // C1: RestrictedChar
		{0x9F, true, false},      // C1: RestrictedChar
		{0xA0, true, true},       // no-break space
		{0xD7FF, true, true},     // last before surrogates
		{0xD800, false, false},   // surrogate
		{0xFFFD, true, true},     // replacement char
		{0xFFFE, false, false},   // non-character
		{0x10FFFF, true, true},   // last code point
		{0x110000, false, false}, // beyond Unicode
		{-1, false, false},       // negative
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("U+%04X", tt.r), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want10, xmlchar.IsLiteralChar(tt.r, false), "XML 1.0")
			require.Equal(t, tt.want11, xmlchar.IsLiteralChar(tt.r, true), "XML 1.1")
			require.Equal(t, tt.r > 0 && tt.r <= 0x10FFFF && (tt.r < 0xD800 || tt.r > 0xDFFF) && tt.r != 0xFFFE && tt.r != 0xFFFF,
				xmlchar.IsXML11Char(tt.r), "IsXML11Char")
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

func TestIsValidName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"foo", true},
		{"_bar", true},
		{"café", true},
		{"a123", true},
		{"a:b", true},   // single colon (QName-shaped) is a valid Name
		{"a:b:c", true}, // multiple colons: valid Name but NOT a valid QName
		{":foo", true},  // leading colon: valid Name, invalid QName/NCName
		{"foo:", true},  // trailing colon: valid Name, invalid QName/NCName
		{":", true},     // a bare colon is a valid NameStartChar
		{"a-b.c", true},
		{"", false},                   // empty
		{"1abc", false},               // must start with NameStartChar
		{"-abc", false},               // '-' is not a NameStartChar
		{".abc", false},               // '.' is not a NameStartChar
		{"a b", false},                // space is not a NameChar
		{"a<b", false},                // '<' must be rejected (injection guard)
		{string([]byte{0xff}), false}, // invalid UTF-8
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, xmlchar.IsValidName(tt.input))
		})
	}
}

func TestIsValidNameWidthAware(t *testing.T) {
	t.Parallel()
	require.True(t, xmlchar.IsValidName("a�b"), "valid U+FFFD must be accepted")
	require.True(t, xmlchar.IsValidName("�"), "U+FFFD is a valid NameStartChar")
	require.False(t, xmlchar.IsValidName(string([]byte{0xff})), "invalid UTF-8 must be rejected")
}

func TestIsValidNCNameWidthAware(t *testing.T) {
	t.Parallel()
	require.True(t, xmlchar.IsValidNCName("a�b"), "valid U+FFFD must be accepted")
	require.True(t, xmlchar.IsValidNCName("�"), "U+FFFD is a valid NCNameStartChar")
	require.False(t, xmlchar.IsValidNCName(string([]byte{0xff})), "invalid UTF-8 must be rejected")
	require.True(t, xmlchar.IsValidNCName("abc"))
	require.False(t, xmlchar.IsValidNCName("1abc"))
}
