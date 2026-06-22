package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// Atomizing a node annotated as xs:QName resolves the lexical QName against the
// node's in-scope namespaces via resolveQNameFromNode (inside AtomizeItem).
func TestAtomize_QNameAnnotatedNode(t *testing.T) {
	doc := mustParseXML(t, `<root xmlns:p="urn:x">p:foo</root>`)
	root := doc.DocumentElement()

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		TypeAnnotations(map[helium.Node]string{
			root: xpath3.TypeQName,
		})

	// fn:prefix-from-QName over the atomized QName value.
	res, err := eval.Evaluate(t.Context(), mustCompile(t, `prefix-from-QName(data(.))`), root)
	require.NoError(t, err)
	require.Equal(t, "p", res.StringValue())

	// namespace-uri-from-QName resolves the bound URI.
	res, err = eval.Evaluate(t.Context(), mustCompile(t, `namespace-uri-from-QName(data(.))`), root)
	require.NoError(t, err)
	require.Equal(t, "urn:x", res.StringValue())

	// local-name-from-QName.
	res, err = eval.Evaluate(t.Context(), mustCompile(t, `local-name-from-QName(data(.))`), root)
	require.NoError(t, err)
	require.Equal(t, "foo", res.StringValue())
}

// A default-namespace lexical (no prefix) resolves against the in-scope default.
func TestAtomize_QNameDefaultNamespace(t *testing.T) {
	doc := mustParseXML(t, `<root xmlns="urn:def">bare</root>`)
	root := doc.DocumentElement()

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		TypeAnnotations(map[helium.Node]string{
			root: xpath3.TypeQName,
		})

	res, err := eval.Evaluate(t.Context(), mustCompile(t, `namespace-uri-from-QName(data(.))`), root)
	require.NoError(t, err)
	require.Equal(t, "urn:def", res.StringValue())
}
