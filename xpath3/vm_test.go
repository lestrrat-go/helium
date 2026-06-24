package xpath3_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestPartialApplicationPlaceholderVM(t *testing.T) {
	result, err := evaluate(t.Context(), nil, `let $f := concat("a", ?, "c") return $f("b")`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "abc", value)
}

func TestPartialApplicationMapArrayVM(t *testing.T) {
	t.Run("map placeholder", func(t *testing.T) {
		result, err := evaluate(t.Context(), nil, `let $m := map{"a":1,"b":2} return $m(?)("b")`)
		require.NoError(t, err)
		value, ok := result.IsNumber()
		require.True(t, ok)
		require.Equal(t, 2.0, value)
	})

	t.Run("array placeholder", func(t *testing.T) {
		result, err := evaluate(t.Context(), nil, `let $a := array{10,20,30} return $a(?)(2)`)
		require.NoError(t, err)
		value, ok := result.IsNumber()
		require.True(t, ok)
		require.Equal(t, 20.0, value)
	})
}

func TestGeneralComparisonAgainstLargeRangeVM(t *testing.T) {
	result, err := evaluate(t.Context(), nil, `1000000000000000020001 = 1000000000000000000000 to 1000000000000010000003`)
	require.NoError(t, err)

	value, ok := result.IsBoolean()
	require.True(t, ok)
	require.True(t, value)
}

func TestLiteralArgsFunctionCallVM(t *testing.T) {
	result, err := evaluate(t.Context(), nil, `concat("go", "-", "vm")`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "go-vm", value)
}

func TestLocationPathPredicateVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := evaluate(t.Context(), doc, `/library/book[@lang="en"]/title/string()`)
	require.NoError(t, err)

	atomics, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 2)
	require.Equal(t, "Go Programming", atomics[0].StringVal())
	require.Equal(t, "XPath Mastery", atomics[1].StringVal())
}

func TestCountPathWithPredicateWhitespaceVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := evaluate(t.Context(), doc, `count( /library/book [ @lang = "en" ] /title )`)
	require.NoError(t, err)

	value, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 2.0, value)
}

func TestPositionalPredicateVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := evaluate(t.Context(), doc, `string(/library/book[1]/@id)`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "1", value)
}

func TestPositionFunctionPredicateVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := evaluate(t.Context(), doc, `string(/library/book[position() = 2]/title)`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "XML Handbook", value)
}

func TestAttributeExistencePredicateVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := evaluate(t.Context(), doc, `count(/library/book[@lang])`)
	require.NoError(t, err)

	value, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 3.0, value)
}

func TestContextItemCompiledIsContextItem(t *testing.T) {
	doc := parseTestDoc(t)
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile(".")
	require.NoError(t, err)

	state := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).NewEvalState(root)
	result, err := compiled.EvaluateReuse(t.Context(), state, root)
	require.NoError(t, err)

	nodes, err := result.Nodes()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, root, nodes[0])
}

func TestAttributeEqualityPredicateReverseOperandsVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := evaluate(t.Context(), doc, `string(/library/book["en" = @lang][2]/title)`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "XPath Mastery", value)
}

func TestDumpVM(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile(`count(/library/book[@lang="en"])`)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = compiled.DumpVM(&buf)
	require.NoError(t, err)

	dump := buf.String()
	require.Contains(t, dump, "root @3\n")
	require.Contains(t, dump, " 0000 location-path      attribute::lang\n")
	require.Contains(t, dump, " 0001 binary             binary(=, @0, \"en\")\n")
	require.Contains(t, dump, " 0002 location-path      /child::library/child::book[attribute-equals(lang, \"en\")]\n")
	require.Contains(t, dump, "*0003 function-call      count(@2)\n")
}

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
