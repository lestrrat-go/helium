package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestPartialApplicationPlaceholderVM(t *testing.T) {
	result, err := xpath3.Evaluate(t.Context(), nil, `let $f := concat("a", ?, "c") return $f("b")`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "abc", value)
}

func TestGeneralComparisonAgainstLargeRangeVM(t *testing.T) {
	result, err := xpath3.Evaluate(t.Context(), nil, `1000000000000000020001 = 1000000000000000000000 to 1000000000000010000003`)
	require.NoError(t, err)

	value, ok := result.IsBoolean()
	require.True(t, ok)
	require.True(t, value)
}

func TestLiteralArgsFunctionCallVM(t *testing.T) {
	result, err := xpath3.Evaluate(t.Context(), nil, `concat("go", "-", "vm")`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "go-vm", value)
}

func TestLocationPathPredicateVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := xpath3.Evaluate(t.Context(), doc, `/library/book[@lang="en"]/title/string()`)
	require.NoError(t, err)

	atomics, err := result.Atomics()
	require.NoError(t, err)
	require.Len(t, atomics, 2)
	require.Equal(t, "Go Programming", atomics[0].StringVal())
	require.Equal(t, "XPath Mastery", atomics[1].StringVal())
}

func TestCountPathWithPredicateWhitespaceVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := xpath3.Evaluate(t.Context(), doc, `count( /library/book [ @lang = "en" ] /title )`)
	require.NoError(t, err)

	value, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 2.0, value)
}

func TestPositionalPredicateVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := xpath3.Evaluate(t.Context(), doc, `string(/library/book[1]/@id)`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "1", value)
}

func TestPositionFunctionPredicateVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := xpath3.Evaluate(t.Context(), doc, `string(/library/book[position() = 2]/title)`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "XML Handbook", value)
}

func TestAttributeExistencePredicateVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := xpath3.Evaluate(t.Context(), doc, `count(/library/book[@lang])`)
	require.NoError(t, err)

	value, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, 3.0, value)
}

func TestContextItemCompiledIsContextItem(t *testing.T) {
	doc := parseTestDoc(t)
	root := doc.DocumentElement()

	compiled, err := xpath3.Compile(".")
	require.NoError(t, err)

	state := xpath3.NewEvalState(t.Context(), root)
	result, err := compiled.EvaluateReuse(state, root)
	require.NoError(t, err)

	nodes, err := result.Nodes()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, root, nodes[0])
}

func TestAttributeEqualityPredicateReverseOperandsVM(t *testing.T) {
	doc := parseTestDoc(t)

	result, err := xpath3.Evaluate(t.Context(), doc, `string(/library/book["en" = @lang][2]/title)`)
	require.NoError(t, err)

	value, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "XPath Mastery", value)
}
