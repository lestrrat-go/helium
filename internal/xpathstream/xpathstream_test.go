package xpathstream_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/xpathstream"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// fnNS is the XPath functions namespace, used to build EQName spellings.
const fnNS = "http://www.w3.org/2005/xpath-functions"

// TestPrefixedFnStreamability verifies that fn:-prefixed and EQName-spelled
// (Q{...}local) calls to the special-cased XPath functions (position, last, ...)
// are analyzed for streamability identically to their unprefixed forms.
func TestPrefixedFnStreamability(t *testing.T) {
	for _, tc := range []struct {
		name      string
		uneprefix string
		other     string
	}{
		{name: "position fn-prefix", uneprefix: "a[position() = 1]", other: "a[fn:position() = 1]"},
		{name: "last fn-prefix", uneprefix: "a[last()]", other: "a[fn:last()]"},
		{name: "position eqname", uneprefix: "a[position() = 1]", other: "a[Q{" + fnNS + "}position() = 1]"},
		{name: "last eqname", uneprefix: "a[last()]", other: "a[Q{" + fnNS + "}last()]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plain, err := xpath3.NewCompiler().Compile(tc.uneprefix)
			require.NoError(t, err, "compile unprefixed")
			other, err := xpath3.NewCompiler().Compile(tc.other)
			require.NoError(t, err, "compile alternate spelling")

			// Sanity: the unprefixed predicate is non-motionless, so the
			// equality assertion below is meaningful.
			require.True(t, xpathstream.ExprTreeHasNonMotionlessPredicate(plain.AST()),
				"unprefixed %q must be non-motionless", tc.uneprefix)

			require.Equal(t,
				xpathstream.ExprTreeHasNonMotionlessPredicate(plain.AST()),
				xpathstream.ExprTreeHasNonMotionlessPredicate(other.AST()),
				"non-motionless classification must match for %q vs %q", tc.uneprefix, tc.other)
		})
	}
}
