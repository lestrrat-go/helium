package xpath3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// TestDumpVM_VariedExpressions exercises the VM dump formatters
// (vmOpcode.String, formatVMExpr, formatNodeTest, formatAxis, etc.) across a
// broad set of expression shapes so the textual rendering of each branch runs.
func TestDumpVM_VariedExpressions(t *testing.T) {
	exprs := []string{
		// literals, variables, arithmetic, comparison.
		`42`,
		`"text"`,
		`-3 + 4 * 2`,
		`1 to 5`,
		`(1, 2, 3)`,
		`1 = 2`,
		`"a" || "b"`,
		// paths across many axes & node tests.
		`/child::a/descendant::b`,
		`//c`,
		`parent::node()`,
		`ancestor::*`,
		`following-sibling::x`,
		`preceding-sibling::y`,
		`following::z`,
		`preceding::w`,
		`attribute::id`,
		`self::node()`,
		`descendant-or-self::node()`,
		`./text()`,
		`./comment()`,
		`./processing-instruction()`,
		`./processing-instruction("xml-stylesheet")`,
		`./element()`,
		`./attribute()`,
		`./document-node()`,
		`./namespace-node()`,
		// predicates.
		`a[1]`,
		`a[@id = "x"]`,
		`a[position() = 2]`,
		`a[@id]`,
		// control flow & quantified.
		`if (1) then 2 else 3`,
		`for $x in (1, 2) return $x`,
		`some $x in (1, 2) satisfies $x = 1`,
		`every $x in (1, 2) satisfies $x = 1`,
		`let $x := 1 return $x`,
		// type expressions.
		`1 instance of xs:integer`,
		`"1" cast as xs:integer`,
		`"1" castable as xs:integer`,
		`1 treat as xs:integer`,
		// functions, maps, arrays, lookups, inline.
		`fn:count((1, 2))`,
		`fn:abs#1`,
		`map { "a": 1 }`,
		`array { 1, 2 }`,
		`[1, 2]`,
		`map { "a": 1 }("a")`,
		`function($x) { $x + 1 }`,
		`(1, 2) ! (. + 1)`,
		`a union b`,
		`a intersect b`,
		`a except b`,
		// kind tests with names / inner tests (formatNodeTest branches).
		`child::element(name)`,
		`child::attribute(id)`,
		`self::document-node(element(root))`,
		`namespace-node()`,
		`processing-instruction("target")`,
		// SequenceType-bearing expressions for instance-of / treat / cast dumps.
		`1 instance of element(x)`,
		`1 instance of map(*)`,
		`1 instance of array(*)`,
		`1 instance of function(*)`,
		`map { "a": 1 } instance of map(xs:string, xs:integer)`,
		`[1] instance of array(xs:integer)`,
		`fn:abs#1 instance of function(xs:double) as xs:double`,
		`Q{urn:x}name(1)`,
		// optimized predicate forms (vmPositionPredicate / attribute-exists /
		// attribute-equals-string) in formatVMExpr.
		`/root/child[5]`,
		`/root/child[@id]`,
		`/root/child[@id = "x"]`,
		`a/b[@n = "1"]/c`,
		// lookups and unary lookups.
		`map { "a": 1 }?*`,
		`[1, 2]?1`,
		`(map { "a": 1 }, map { "b": 2 }) ! ?*`,
	}

	for _, e := range exprs {
		t.Run(e, func(t *testing.T) {
			compiled, err := xpath3.NewCompiler().Compile(e)
			require.NoError(t, err)
			var sb strings.Builder
			require.NoError(t, compiled.DumpVM(&sb))
			require.NotEmpty(t, sb.String())
			require.Contains(t, sb.String(), "root @")
		})
	}
}
