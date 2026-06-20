package xpathstream_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/xpathstream"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// TestPrefixedFnStreamability verifies that fn:-prefixed calls to the
// special-cased XPath functions (position, last, ...) are analyzed for
// streamability identically to their unprefixed forms.
func TestPrefixedFnStreamability(t *testing.T) {
	for _, tc := range []struct {
		name      string
		uneprefix string
		prefixed  string
	}{
		{name: "position", uneprefix: "a[position() = 1]", prefixed: "a[fn:position() = 1]"},
		{name: "last", uneprefix: "a[last()]", prefixed: "a[fn:last()]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plain, err := xpath3.NewCompiler().Compile(tc.uneprefix)
			require.NoError(t, err, "compile unprefixed")
			pref, err := xpath3.NewCompiler().Compile(tc.prefixed)
			require.NoError(t, err, "compile fn-prefixed")

			require.Equal(t,
				xpathstream.ExprTreeHasNonMotionlessPredicate(plain.AST()),
				xpathstream.ExprTreeHasNonMotionlessPredicate(pref.AST()),
				"non-motionless classification must match for %q vs %q", tc.uneprefix, tc.prefixed)

			require.Equal(t,
				xpathstream.ExprUsesFunction(plain, tc.name),
				xpathstream.ExprUsesFunction(pref, tc.name),
				"ExprUsesFunction must match for %q vs %q", tc.uneprefix, tc.prefixed)
		})
	}
}
