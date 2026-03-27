package xpath3_test

import (
	"fmt"
	"io"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

const testXML = `<library>
  <book id="1" lang="en">
    <title>Go Programming</title>
    <author>Alice</author>
    <year>2020</year>
    <price>29.99</price>
  </book>
  <book id="2" lang="ja">
    <title>XML Handbook</title>
    <author>Bob</author>
    <year>2019</year>
    <price>39.99</price>
  </book>
  <book id="3" lang="en">
    <title>XPath Mastery</title>
    <author>Carol</author>
    <year>2021</year>
    <price>24.99</price>
  </book>
</library>`

type testURIResolver map[string]string

func (r testURIResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	body, ok := r[uri]
	if !ok {
		return nil, io.EOF
	}
	return io.NopCloser(strings.NewReader(body)), nil
}

type testCollectionResolver struct {
	sequences map[string]xpath3.Sequence
	uris      map[string][]string
}

func (r testCollectionResolver) ResolveCollection(uri string) (xpath3.Sequence, error) {
	return r.sequences[uri], nil
}

func (r testCollectionResolver) ResolveURICollection(uri string) ([]string, error) {
	return r.uris[uri], nil
}

func parseTestDoc(t *testing.T) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(testXML))
	require.NoError(t, err)
	return doc
}

func TestCompile(t *testing.T) {
	t.Parallel()
	expr, err := xpath3.NewCompiler().Compile(`/library/book`)
	require.NoError(t, err)
	require.Equal(t, `/library/book`, expr.String())
}

func TestCompileError(t *testing.T) {
	t.Parallel()
	_, err := xpath3.NewCompiler().Compile(`][`)
	require.Error(t, err)
}

func TestMustCompile(t *testing.T) {
	t.Parallel()
	require.NotPanics(t, func() {
		xpath3.NewCompiler().MustCompile(`/library/book`)
	})
	require.Panics(t, func() {
		xpath3.NewCompiler().MustCompile(`][`)
	})
}

func TestFind(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	nodes, err := find(t.Context(), doc, `/library/book`)
	require.NoError(t, err)
	require.Len(t, nodes, 3)
}

func TestFindError(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	_, err := find(t.Context(), doc, `count(/library/book)`)
	require.Error(t, err)
}

func TestEvaluateConvenience(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `count(/library/book)`)
	require.NoError(t, err)
	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 3.0, n)
}

func TestWithDefaultLanguage(t *testing.T) {
	t.Parallel()
	compiled, err := xpath3.NewCompiler().Compile(`default-language()`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		DefaultLanguage("fr-CA").
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	atomics, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 1)
	require.Equal(t, xpath3.TypeLanguage, atomics[0].TypeName)
	require.Equal(t, "fr-CA", atomics[0].StringVal())
}

func TestInlineFunctionPreservesDefaultLanguage(t *testing.T) {
	t.Parallel()
	compiled, err := xpath3.NewCompiler().Compile(`let $f := function() { default-language() } return $f()`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		DefaultLanguage("fr-CA").
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	atomics, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 1)
	require.Equal(t, "fr-CA", atomics[0].StringVal())
}

func TestPrefixedVariableRequiresDeclaredNamespace(t *testing.T) {
	t.Parallel()
	compiled, err := xpath3.NewCompiler().Compile(`$p:v`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
			"p:v": xpath3.SingleInteger(1),
		})).
		Evaluate(t.Context(), compiled, nil)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "XPST0081", xpErr.Code)
}

func TestInlineFunctionDoesNotInheritFocus(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	_, err := evaluate(t.Context(), doc, `let $f := function() { boolean(.) } return $f()`)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "XPDY0002", xpErr.Code)
}

func TestNilContextRangeExpr(t *testing.T) {
	t.Parallel()
	// "1 to 10" doesn't require a context item; evaluation with nil node must succeed.
	result, err := evaluate(t.Context(), nil, `1 to 10`)
	require.NoError(t, err)
	require.Equal(t, 10, result.Sequence().Len())
}

func TestNilContextWithContextItem(t *testing.T) {
	t.Parallel()
	// Evaluating "." with a context item (atomic value) and nil node must succeed.
	compiled, err := xpath3.NewCompiler().Compile(".")
	require.NoError(t, err)
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		ContextItem(xpath3.AtomicValue{TypeName: "xs:integer", Value: int64(42)}).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)
	require.Equal(t, 1, result.Sequence().Len())
}

func TestNilContextElementKindTest(t *testing.T) {
	t.Parallel()
	// "element()" as standalone expression parses as child::element() step.
	// With a valid context node, it should select child elements.
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `element()`)
	require.NoError(t, err)
	// The doc has a root element "library"
	require.True(t, result.Sequence().Len() > 0)
}

func TestResultIsNodeSet(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `/library/book`)
	require.NoError(t, err)
	require.True(t, result.IsNodeSet())

	nodes, err := result.Nodes()
	require.NoError(t, err)
	require.Len(t, nodes, 3)
}

func TestPrefixedFunctionMissingDoesNotFallBackToFnNamespace(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	compiled, err := xpath3.NewCompiler().Compile(`p:count(/library/book)`)
	require.NoError(t, err)
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Namespaces(map[string]string{"p": "urn:other"}).
		Evaluate(t.Context(), compiled, doc)
	require.Error(t, err)
	require.ErrorIs(t, err, xpath3.ErrUnknownFunction)
}

func TestURIQualifiedFunctionMissingDoesNotFallBackToFnNamespace(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	_, err := evaluate(t.Context(), doc, `Q{urn:other}substring("XPath", 2, 3)`)
	require.Error(t, err)
	require.ErrorIs(t, err, xpath3.ErrUnknownFunction)
}

func TestStringCoercionRejectsMultiItemSequence(t *testing.T) {
	t.Parallel()
	_, err := evaluate(t.Context(), nil, `string-to-codepoints(("ab", "cd"))`)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "XPTY0004", xpErr.Code)
}

func TestIntegerCoercionRejectsMultiItemSequence(t *testing.T) {
	t.Parallel()
	_, err := evaluate(t.Context(), nil, `remove((1, 2, 3), (2, 3))`)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "XPTY0004", xpErr.Code)
}

func TestPublicStringArgsRejectMultiItemSequence(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	tests := []struct {
		name string
		expr string
	}{
		{"parse-json", `parse-json(("{}", "{}"))`},
		{"json-doc", `json-doc(("data.json", "other.json"))`},
		{"json-to-xml", `json-to-xml(('{"a":1}', '{"b":2}'))`},
		{"unparsed-text", `unparsed-text(("a.txt", "b.txt"))`},
		{"unparsed-text-available", `unparsed-text-available(("a.txt", "b.txt"))`},
		{"unparsed-text-lines", `unparsed-text-lines(("a.txt", "b.txt"))`},
		{"QName", `QName("urn:test", ("p:name", "q:name"))`},
		{"namespace-uri-for-prefix", `namespace-uri-for-prefix(("xml", "p"), /library/book[1])`},
		{"resolve-QName", `resolve-QName(("xml:lang", "p:name"), /library/book[1])`},
		{"lang", `lang(("en", "fr"), /library/book[1])`},
		{"parse-xml", `parse-xml(("<root/>", "<other/>"))`},
		{"parse-xml-fragment", `parse-xml-fragment(("<a/>", "<b/>"))`},
		{"parse-ietf-date", `parse-ietf-date(("Wed, 06 Jun 1994 07:29:35 GMT", "Thu, 07 Jun 1994 07:29:35 GMT"))`},
		{"contains-token", `contains-token(("a b"), ("a", "b"))`},
		{"sort", `sort((2, 1), ("http://www.w3.org/2005/xpath-functions/collation/codepoint", "dup"))`},
		{"array-sort", `array:sort([2, 1], ("http://www.w3.org/2005/xpath-functions/collation/codepoint", "dup"))`},
		{"distinct-values-collation", `distinct-values((1, 2), ("http://www.w3.org/2005/xpath-functions/collation/codepoint", "dup"))`},
		{"format-number", `format-number(1, "0", ("fmt", "other"))`},
		{"map-merge", `map:merge((map{"a": 1}), map{"duplicates": ("use-first", "reject")})`},
		{"error-description", `error((), ("a", "b"))`},
		{"trace-label", `trace(1, ("a", "b"))`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := evaluate(t.Context(), doc, tc.expr)
			require.Error(t, err)

			var xpErr *xpath3.XPathError
			require.ErrorAs(t, err, &xpErr)
			require.Equal(t, "XPTY0004", xpErr.Code)
		})
	}
}

func TestPublicStringArgsPreserveAtomizationErrors(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	tests := []struct {
		name string
		expr string
	}{
		{"parse-json", `parse-json(map{})`},
		{"unparsed-text", `unparsed-text(map{})`},
		{"parse-xml", `parse-xml(map{})`},
		{"trace-label", `trace(1, map{})`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := evaluate(t.Context(), doc, tc.expr)
			require.Error(t, err)

			var xpErr *xpath3.XPathError
			require.ErrorAs(t, err, &xpErr)
			require.Equal(t, "FOTY0013", xpErr.Code)
		})
	}
}

func TestResolveQNameRejectsMalformedQName(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	_, err := evaluate(t.Context(), doc, `resolve-QName("pre:thi:ng", /library/book[1])`)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FOCA0002", xpErr.Code)
}

func TestJSONDocUsesURIResolverAndBaseURI(t *testing.T) {
	t.Parallel()
	compiled, err := xpath3.NewCompiler().Compile(`json-doc("data.json")?name`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		BaseURI("http://example.com/base/").
		URIResolver(testURIResolver{
			"http://example.com/base/data.json": `{"name":"helium"}`,
		}).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "helium", s)
}

func TestDocUsesURIResolverAndBaseURI(t *testing.T) {
	t.Parallel()
	compiled, err := xpath3.NewCompiler().Compile(`string(doc("data.xml")/root/name)`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		BaseURI("http://example.com/base/").
		URIResolver(testURIResolver{
			"http://example.com/base/data.xml": `<root><name>helium</name></root>`,
		}).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "helium", s)
}

func TestNodeComparisonAfterPrimaryDocOrderBuild(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	compiled, err := xpath3.NewCompiler().Compile(`let $warm := (/library | /library/book[1]) return doc("other.xml")/other/item[1] << doc("other.xml")/other/item[2]`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		BaseURI("http://example.com/base/").
		URIResolver(testURIResolver{
			"http://example.com/base/other.xml": `<other><item id="1"/><item id="2"/></other>`,
		}).
		Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)

	value, ok := result.IsBoolean()
	require.True(t, ok)
	require.True(t, value)
}

func TestCollectionUsesBaseURIResolution(t *testing.T) {
	t.Parallel()
	compiled, err := xpath3.NewCompiler().Compile(`collection("data")`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		BaseURI("http://example.com/base/").
		CollectionResolver(testCollectionResolver{
			sequences: map[string]xpath3.Sequence{
				"http://example.com/base/data": xpath3.SingleString("helium"),
			},
		}).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "helium", s)
}

func TestEmptySequenceResultIsNodeSet(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `/library/missing`)
	require.NoError(t, err)
	require.True(t, result.IsNodeSet())

	nodes, err := result.Nodes()
	require.NoError(t, err)
	require.Empty(t, nodes)
}

func TestResultIsBoolean(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `count(/library/book) > 2`)
	require.NoError(t, err)

	b, ok := result.IsBoolean()
	require.True(t, ok)
	require.True(t, b)
}

func TestResultIsString(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `string(/library/book[1]/title)`)
	require.NoError(t, err)

	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "Go Programming", s)
}

func TestResultIsNumber(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `sum(/library/book/price)`)
	require.NoError(t, err)

	n, ok := result.IsNumber()
	require.True(t, ok)
	require.InDelta(t, 94.97, n, 0.01)
}

func TestResultIsAtomic(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `42`)
	require.NoError(t, err)
	require.True(t, result.IsAtomic())

	avs, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, avs, 1)
}

func TestResultSequence(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `(1, 2, 3)`)
	require.NoError(t, err)
	require.Equal(t, 3, result.Sequence().Len())
}

// --- Location paths ---

func TestDescendantAxis(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	nodes, err := find(t.Context(), doc, `//title`)
	require.NoError(t, err)
	require.Len(t, nodes, 3)
}

func TestPredicateFilter(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	nodes, err := find(t.Context(), doc, `/library/book[@lang="en"]`)
	require.NoError(t, err)
	require.Len(t, nodes, 2)
}

func TestPositionalPredicate(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	nodes, err := find(t.Context(), doc, `/library/book[2]`)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
}

func TestAttributeAccess(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `string(/library/book[1]/@id)`)
	require.NoError(t, err)

	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "1", s)
}

// --- String functions ---

func TestStringFunctions(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	tests := []struct {
		expr string
		want string
	}{
		{`upper-case("hello")`, "HELLO"},
		{`lower-case("WORLD")`, "world"},
		{`concat(string(/library/book[1]/author), " & ", string(/library/book[2]/author))`, "Alice & Bob"},
		{`substring("XPath", 2, 3)`, "Pat"},
		{`normalize-space("  hello   world  ")`, "hello world"},
		{`translate("abc", "abc", "ABC")`, "ABC"},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			result, err := evaluate(t.Context(), doc, tt.expr)
			require.NoError(t, err)
			s, ok := result.IsString()
			require.True(t, ok)
			require.Equal(t, tt.want, s)
		})
	}
}

func TestStringFunctionsRejectMultiItemSequences(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	tests := []struct {
		name string
		expr string
		code string
	}{
		{name: "upper-case", expr: `upper-case(("a", "b"))`, code: "XPTY0004"},
		{name: "substring source", expr: `substring(("abc", "def"), 2)`, code: "XPTY0004"},
		{name: "substring position", expr: `substring("abc", (1, 2))`, code: "XPTY0004"},
		{name: "resolve-uri", expr: `resolve-uri(("a", "b"), "http://example.com/")`, code: "XPTY0004"},
		{name: "regex input", expr: `matches(("abc", "def"), "a")`, code: "XPTY0004"},
		{name: "concat operator map", expr: `map{} || "x"`, code: "FOTY0014"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := evaluate(t.Context(), doc, tt.expr)
			require.Error(t, err)

			var xpErr *xpath3.XPathError
			require.ErrorAs(t, err, &xpErr)
			require.Equal(t, tt.code, xpErr.Code)
		})
	}
}

// --- Numeric functions ---

func TestNumericFunctions(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	tests := []struct {
		expr string
		want float64
	}{
		{`abs(-5)`, 5},
		{`ceiling(4.2)`, 5},
		{`floor(4.8)`, 4},
		{`round(4.5)`, 5},
		{`count(/library/book)`, 3},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			result, err := evaluate(t.Context(), doc, tt.expr)
			require.NoError(t, err)
			n, ok := result.IsNumber()
			require.True(t, ok)
			require.Equal(t, tt.want, n)
		})
	}
}

// --- Boolean functions ---

func TestBooleanFunctions(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	tests := []struct {
		expr string
		want bool
	}{
		{`true()`, true},
		{`false()`, false},
		{`boolean(1)`, true},
		{`boolean(0)`, false},
		{`not(false())`, true},
		{`exists(/library/book)`, true},
		{`empty(/library/nonexistent)`, true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			result, err := evaluate(t.Context(), doc, tt.expr)
			require.NoError(t, err)
			b, ok := result.IsBoolean()
			require.True(t, ok)
			require.Equal(t, tt.want, b)
		})
	}
}

// --- Aggregate functions ---

func TestAggregateFunctions(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	t.Run("min", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `min(/library/book/year)`)
		require.NoError(t, err)
		n, ok := result.IsNumber()
		require.True(t, ok)
		require.Equal(t, 2019.0, n)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `max(/library/book/year)`)
		require.NoError(t, err)
		n, ok := result.IsNumber()
		require.True(t, ok)
		require.Equal(t, 2021.0, n)
	})
}

// --- Sequence operations ---

func TestSequenceOperations(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	t.Run("range", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `1 to 5`)
		require.NoError(t, err)
		require.Equal(t, 5, result.Sequence().Len())
	})

	t.Run("reverse", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `reverse((1, 2, 3))`)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 3, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, int64(3), av.IntegerVal())
	})

	t.Run("distinct-values", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `distinct-values((1, 2, 2, 3, 3))`)
		require.NoError(t, err)
		require.Equal(t, 3, result.Sequence().Len())
	})

	t.Run("subsequence", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `subsequence((1, 2, 3, 4, 5), 2, 3)`)
		require.NoError(t, err)
		require.Equal(t, 3, result.Sequence().Len())
	})
}

// --- Arithmetic ---

func TestArithmetic(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	tests := []struct {
		expr string
		want float64
	}{
		{`2 + 3`, 5},
		{`10 - 4`, 6},
		{`3 * 4`, 12},
		{`10 div 3`, 10.0 / 3.0},
		{`10 idiv 3`, 3},
		{`10 mod 3`, 1},
		{`-5`, -5},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			result, err := evaluate(t.Context(), doc, tt.expr)
			require.NoError(t, err)
			n, ok := result.IsNumber()
			require.True(t, ok)
			require.InDelta(t, tt.want, n, 1e-10)
		})
	}
}

// --- Comparisons ---

func TestComparisons(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	tests := []struct {
		expr string
		want bool
	}{
		{`1 = 1`, true},
		{`1 != 2`, true},
		{`1 < 2`, true},
		{`2 > 1`, true},
		{`1 <= 1`, true},
		{`1 >= 1`, true},
		{`1 eq 1`, true},
		{`1 ne 2`, true},
		{`1 lt 2`, true},
		{`2 gt 1`, true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			result, err := evaluate(t.Context(), doc, tt.expr)
			require.NoError(t, err)
			b, ok := result.IsBoolean()
			require.True(t, ok)
			require.Equal(t, tt.want, b)
		})
	}
}

// --- Logic ---

func TestLogic(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	tests := []struct {
		expr string
		want bool
	}{
		{`true() and true()`, true},
		{`true() and false()`, false},
		{`false() or true()`, true},
		{`false() or false()`, false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			result, err := evaluate(t.Context(), doc, tt.expr)
			require.NoError(t, err)
			b, ok := result.IsBoolean()
			require.True(t, ok)
			require.Equal(t, tt.want, b)
		})
	}
}

// --- FLWOR ---

func TestFLWOR(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	t.Run("for", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `for $x in (1, 2, 3) return $x * 2`)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 3, seq.Len())
		av, ok := seq.Get(2).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, int64(6), av.IntegerVal())
	})

	t.Run("let", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `let $x := 42 return $x`)
		require.NoError(t, err)
		n, ok := result.IsNumber()
		require.True(t, ok)
		require.Equal(t, 42.0, n)
	})
}

// --- Quantified expressions ---

func TestQuantified(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	t.Run("some", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `some $x in (1, 2, 3) satisfies $x > 2`)
		require.NoError(t, err)
		b, ok := result.IsBoolean()
		require.True(t, ok)
		require.True(t, b)
	})

	t.Run("every", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `every $x in (1, 2, 3) satisfies $x > 0`)
		require.NoError(t, err)
		b, ok := result.IsBoolean()
		require.True(t, ok)
		require.True(t, b)
	})
}

// --- If-then-else ---

func TestIfThenElse(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `if (count(/library/book) > 2) then "many" else "few"`)
	require.NoError(t, err)
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "many", s)
}

// --- Cast / instance of ---

func TestCast(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `"42" cast as xs:integer`)
	require.NoError(t, err)
	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 42.0, n)
}

func TestCastable(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `"42" castable as xs:integer`)
	require.NoError(t, err)
	b, ok := result.IsBoolean()
	require.True(t, ok)
	require.True(t, b)
}

// --- Map constructor + functions ---

func TestMapConstructor(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `map:size(map { "a": 1, "b": 2 })`)
	require.NoError(t, err)
	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 2.0, n)
}

// --- Array constructor + functions ---

func TestArrayConstructor(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `array:size([1, 2, 3])`)
	require.NoError(t, err)
	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 3.0, n)
}

// --- Math functions ---

func TestMathFunctions(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	t.Run("pi", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `math:pi()`)
		require.NoError(t, err)
		n, ok := result.IsNumber()
		require.True(t, ok)
		require.InDelta(t, math.Pi, n, 1e-10)
	})

	t.Run("sqrt", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `math:sqrt(16)`)
		require.NoError(t, err)
		n, ok := result.IsNumber()
		require.True(t, ok)
		require.Equal(t, 4.0, n)
	})
}

// --- Higher-order functions ---

func TestHOFFunctions(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	t.Run("for-each", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `for-each((1, 2, 3), function($x) { $x * 10 })`)
		require.NoError(t, err)
		seq := result.Sequence()
		require.Equal(t, 3, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, int64(10), av.IntegerVal())
	})

	t.Run("for-each function coercion", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `let $f := function($ff as (function(item()) as item()), $s as xs:string){$ff($ff($s))} return
			for-each((upper-case#1, lower-case#1, normalize-space#1, concat(?, '!')), $f(?, ' Say NO! '))`)
		require.NoError(t, err)

		atomics, err := result.Atomics()
		require.NoError(t, err)
		require.Len(t, atomics, 4)
		require.Equal(t, " SAY NO! ", atomics[0].StringVal())
		require.Equal(t, " say no! ", atomics[1].StringVal())
		require.Equal(t, "Say NO!", atomics[2].StringVal())
		require.Equal(t, " Say NO! !!", atomics[3].StringVal())
	})

	t.Run("filter", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `filter((1, 2, 3, 4, 5), function($x) { $x > 3 })`)
		require.NoError(t, err)
		require.Equal(t, 2, result.Sequence().Len())
	})

	t.Run("fold-left", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `fold-left((1, 2, 3), 0, function($a, $b) { $a + $b })`)
		require.NoError(t, err)
		n, ok := result.IsNumber()
		require.True(t, ok)
		require.Equal(t, 6.0, n)
	})
}

// --- Simple map operator ---

func TestSimpleMapOperator(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `(1, 2, 3) ! (. * 2)`)
	require.NoError(t, err)
	seq := result.Sequence()
	require.Equal(t, 3, seq.Len())
	av, ok := seq.Get(1).(xpath3.AtomicValue)
	require.True(t, ok)
	require.Equal(t, int64(4), av.IntegerVal())
}

// --- Try-catch ---

func TestTryCatch(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `try { error() } catch * { "caught" }`)
	require.NoError(t, err)
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "caught", s)
}

func TestTryCatchMatchesCustomQNameError(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `
		try { error(QName("urn:test-errors", "t:oops"), "boom") }
		catch Q{urn:test-errors}oops {
			(prefix-from-QName($err:code), local-name-from-QName($err:code), string(namespace-uri-from-QName($err:code)))
		}`)
	require.NoError(t, err)

	atomics, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 3)
	require.Equal(t, "t", atomics[0].StringVal())
	require.Equal(t, "oops", atomics[1].StringVal())
	require.Equal(t, "urn:test-errors", atomics[2].StringVal())
}

func TestErrorRejectsNonQNameCode(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	_, err := evaluate(t.Context(), doc, `error("oops")`)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "XPTY0004", xpErr.Code)
}

func TestApplyRejectsEmptyArrayArgument(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	_, err := evaluate(t.Context(), doc, `apply(function($x) { $x }, ())`)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "XPTY0004", xpErr.Code)
}

// --- Context with variables ---

func TestContextVariables(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	compiled, err := xpath3.NewCompiler().Compile(`count(/library/book[price > $threshold])`)
	require.NoError(t, err)
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
			"threshold": xpath3.SingleDouble(30.0),
		})).
		Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)
	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 1.0, n)
}

func TestWithVariablesCopiesSequences(t *testing.T) {
	t.Parallel()
	seq := xpath3.SingleInteger(1)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
			"x": seq,
		}))

	seq.(xpath3.ItemSlice)[0] = xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(2)}

	compiled, err := xpath3.NewCompiler().Compile(`$x`)
	require.NoError(t, err)
	result, err := eval.Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	atomics, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 1)
	require.Equal(t, int64(1), atomics[0].IntegerVal())
}

// --- Context with namespaces ---

func TestContextNamespaces(t *testing.T) {
	t.Parallel()
	xmlNS := `<root xmlns:ex="http://example.com"><ex:item>hello</ex:item></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xmlNS))
	require.NoError(t, err)

	compiled, err := xpath3.NewCompiler().Compile(`/root/e:item`)
	require.NoError(t, err)
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Namespaces(map[string]string{
			"e": "http://example.com",
		}).
		Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)
	nodes, err := result.Nodes()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
}

func TestUndeclaredPrefixInPathStep(t *testing.T) {
	t.Parallel()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:b="urn:b"><b:item/></root>`))
	require.NoError(t, err)

	// "p" is not declared in namespaces — must produce XPST0081
	compiled, err := xpath3.NewCompiler().Compile(`/root/p:item`)
	require.NoError(t, err)
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Namespaces(map[string]string{
			"a": "urn:a",
		}).
		Evaluate(t.Context(), compiled, doc)
	require.Error(t, err, "undeclared prefix in path step must be rejected")
	require.Contains(t, err.Error(), "XPST0081")
}

// --- Op limit ---

func TestOpLimit(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	compiled, err := xpath3.NewCompiler().Compile(`/library/book/title`)
	require.NoError(t, err)
	// Op counting triggers on location path node traversal
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		OpLimit(1).
		Evaluate(t.Context(), compiled, doc)
	require.Error(t, err)
}

// --- Regex functions ---

func TestRegexFunctions(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	t.Run("matches", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `matches("hello world", "^hello")`)
		require.NoError(t, err)
		b, ok := result.IsBoolean()
		require.True(t, ok)
		require.True(t, b)
	})

	t.Run("replace", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `replace("hello world", "world", "XPath")`)
		require.NoError(t, err)
		s, ok := result.IsString()
		require.True(t, ok)
		require.Equal(t, "hello XPath", s)
	})

	t.Run("tokenize", func(t *testing.T) {
		t.Parallel()
		result, err := evaluate(t.Context(), doc, `tokenize("a-b-c", "-")`)
		require.NoError(t, err)
		require.Equal(t, 3, result.Sequence().Len())
	})
}

func BenchmarkMatchesLiteralRegex(b *testing.B) {
	doc, err := helium.NewParser().Parse(b.Context(), []byte(testXML))
	require.NoError(b, err)

	expr := xpath3.NewCompiler().MustCompile(`matches("hello world", "^hello")`)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := eval.Evaluate(b.Context(), expr, doc)
		require.NoError(b, err)
		value, ok := result.IsBoolean()
		require.True(b, ok)
		require.True(b, value)
	}
}

// --- Union / intersect / except ---

func TestSetOperations(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)

	t.Run("union", func(t *testing.T) {
		t.Parallel()
		nodes, err := find(t.Context(), doc, `/library/book[1] | /library/book[3]`)
		require.NoError(t, err)
		require.Len(t, nodes, 2)
	})
}

// --- Expression reuse ---

func TestExpressionReuse(t *testing.T) {
	t.Parallel()
	expr := xpath3.NewCompiler().MustCompile(`count(/library/book)`)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	doc1 := parseTestDoc(t)
	r1, err := eval.Evaluate(t.Context(), expr, doc1)
	require.NoError(t, err)
	n1, ok := r1.IsNumber()
	require.True(t, ok)
	require.Equal(t, 3.0, n1)

	doc2, err := helium.NewParser().Parse(t.Context(), []byte(`<library><book/></library>`))
	require.NoError(t, err)
	r2, err := eval.Evaluate(t.Context(), expr, doc2)
	require.NoError(t, err)
	n2, ok := r2.IsNumber()
	require.True(t, ok)
	require.Equal(t, 1.0, n2)
}

// --- Inline function ---

func TestInlineFunction(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc,
		`let $double := function($x) { $x * 2 } return $double(21)`)
	require.NoError(t, err)
	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 42.0, n)
}

// --- String concat operator ---

func TestStringConcatOperator(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `"hello" || " " || "world"`)
	require.NoError(t, err)
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "hello world", s)
}

// --- Arrow operator ---

func TestArrowOperator(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `"hello" => upper-case()`)
	require.NoError(t, err)
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "HELLO", s)
}

// --- Lookup ---

func TestMapLookup(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `map { "x": 42 }?x`)
	require.NoError(t, err)
	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 42.0, n)
}

func TestArrayLookup(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	result, err := evaluate(t.Context(), doc, `[10, 20, 30]?2`)
	require.NoError(t, err)
	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 20.0, n)
}

func TestArrayLookupRejectsOversizedIndex(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	_, err := evaluate(t.Context(), doc, `[10, 20, 30]?9999999999999999999999999`)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FOAY0001", xpErr.Code)
}

func TestConcurrentEvaluate(t *testing.T) {
	t.Parallel()
	doc := parseTestDoc(t)
	expr := xpath3.NewCompiler().MustCompile(`count(/library/book)`)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	const goroutines = 8
	errs := make(chan error, goroutines)
	for range goroutines {
		go func() {
			r, err := eval.Evaluate(t.Context(), expr, doc)
			if err != nil {
				errs <- err
				return
			}
			n, ok := r.IsNumber()
			if !ok || n != 3 {
				errs <- fmt.Errorf("expected 3, got %v (ok=%v)", n, ok)
				return
			}
			errs <- nil
		}()
	}
	for range goroutines {
		require.NoError(t, <-errs)
	}
}
