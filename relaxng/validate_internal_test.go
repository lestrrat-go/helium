package relaxng

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNormalizeTokenFastPath checks that normalizeToken's already-collapsed
// shortcut returns exactly what the full split-and-join collapse returns.
func TestNormalizeTokenFastPath(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"", " ", "  ", "a", "a b", "a  b", " a", "a ", " a b ", "a\tb", "a\nb", "a\rb",
		"\t", "a b c", "a b  c", "a b", "   ", " ", "ab cd ef",
	}
	for _, in := range inputs {
		want := strings.Join(xmlFields(in), " ")
		require.Equal(t, want, normalizeToken(in), "normalizeToken(%q)", in)
		if isCollapsedToken(in) {
			require.Equal(t, in, want, "isCollapsedToken(%q) is true but the collapse changes it", in)
		}
	}
}
