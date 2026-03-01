package helium

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDumpQuotedString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no quotes",
			input:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "double quotes only",
			input:    `say "hello"`,
			expected: `'say "hello"'`,
		},
		{
			name:     "single quotes only",
			input:    "it's",
			expected: `"it's"`,
		},
		{
			name:     "both quotes",
			input:    `it's a "test"`,
			expected: `"it's a &quot;test&quot;"`,
		},
		{
			name:     "double quote at start",
			input:    `"hello' world`,
			expected: `"&quot;hello' world"`,
		},
		{
			name:     "double quote at end",
			input:    `hello' world"`,
			expected: `"hello' world&quot;"`,
		},
		{
			name:     "consecutive double quotes",
			input:    `a'b""c`,
			expected: `"a'b&quot;&quot;c"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := dumpQuotedString(&buf, tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, buf.String())
		})
	}
}
