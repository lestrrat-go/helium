package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// Compiling a broad range of expression shapes drives the VM compiler's
// per-node prefix-plan construction (appendExprLocalPrefixChecks and friends),
// and Expression.Validate then runs the resulting prefixValidationPlan.
func TestValidate_WalksAllExprShapes(t *testing.T) {
	ns := xpath3.PredeclaredNamespaces()

	exprs := []string{
		`fn:concat("a", "b")`,
		`"1" cast as xs:integer`,
		`"1" castable as xs:integer`,
		`1 instance of xs:integer`,
		`1 treat as xs:integer`,
		`/a/b/c`,
		`a[1][@x]`,
		`-1 + 2 * 3`,
		`"a" || "b"`,
		`(1, 2) ! (. + 1)`,
		`1 to 10`,
		`a union b`,
		`a intersect b`,
		`a except b`,
		`if (1) then 2 else 3`,
		`for $x in (1, 2) return $x`,
		`let $y := 3 return $y`,
		`some $x in (1, 2) satisfies $x = 1`,
		`every $x in (1, 2) satisfies $x = 1`,
		`try { 1 div 0 } catch * { 0 }`,
		`let $f := fn:abs#1 return $f(-1)`,
		`function($x) { $x + 1 }(2)`,
		`map { "a": 1, "b": 2 }?a`,
		`[1, 2, 3]?2`,
		`(1, 2, 3)[. > 1]`,
		`map { "a": 1 }("a")`,
		`fn:count(//x)`,
		`$v[. = "y"]`,
		`fn:for-each((1, 2), function($x) { $x })`,
	}
	for _, e := range exprs {
		compiled, err := xpath3.NewCompiler().Compile(e)
		require.NoError(t, err, e)
		// fn / map / array prefixes are predeclared, $v is unbound but prefix
		// validation only checks namespace prefixes, not variable bindings.
		require.NoError(t, compiled.Validate(ns), e)
	}

	// An undeclared prefix on a function name is caught.
	bad, err := xpath3.NewCompiler().Compile(`undeclared:foo(1)`)
	require.NoError(t, err)
	require.Error(t, bad.Validate(map[string]string{}))

	// An undeclared prefix in a cast target type is caught.
	bad, err = xpath3.NewCompiler().Compile(`"1" cast as undeclared:myType`)
	require.NoError(t, err)
	require.Error(t, bad.Validate(map[string]string{}))
}
