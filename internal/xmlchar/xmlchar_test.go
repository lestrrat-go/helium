package xmlchar_test

import (
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

func TestIsNCNameStartChar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		r    rune
		want bool
	}{
		{'A', true},
		{'z', true},
		{'_', true},
		{'\u00C0', true},  // Latin capital A with grave
		{'\u3001', true},  // CJK
		{'0', false},
		{'-', false},
		{'.', false},
		{':', false},
		{' ', false},
	}
	for _, tt := range tests {
		t.Run(string(tt.r), func(t *testing.T) {
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
		{'\u00B7', true},  // middle dot (combining)
		{':', false},
		{' ', false},
	}
	for _, tt := range tests {
		t.Run(string(tt.r), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, xmlchar.IsNCNameChar(tt.r))
		})
	}
}
