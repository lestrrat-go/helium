package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// The QName-accessor functions take a singleton xs:QName? (or element() for the
// node-based ones). A 2-item argument must raise XPTY0004, not silently use the
// first item.
func TestQNameFunctionsRejectMultiItemArg(t *testing.T) {
	t.Parallel()

	doc := mustParseXML(t, `<root xmlns:p="urn:p"><a/><b/></root>`)

	cases := []string{
		`prefix-from-QName((QName("urn:p", "p:a"), QName("urn:p", "p:b")))`,
		`local-name-from-QName((QName("urn:p", "p:a"), QName("urn:p", "p:b")))`,
		`namespace-uri-from-QName((QName("urn:p", "p:a"), QName("urn:p", "p:b")))`,
		`resolve-QName("p:a", //root/*)`,
		`in-scope-prefixes(//root/*)`,
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(t.Context(), doc, expr)
			require.Error(t, err, expr)
			var xpErr *xpath3.XPathError
			require.ErrorAs(t, err, &xpErr)
			require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code, expr)
		})
	}
}

// namespace-uri-for-prefix and in-scope-prefixes take a REQUIRED singleton
// element() second/first argument. An empty sequence or a multi-element
// sequence must raise XPTY0004, not be silently accepted.
func TestQNameElementArgRequiresSingleton(t *testing.T) {
	t.Parallel()

	doc := mustParseXML(t, `<root xmlns:p="urn:p"><a/><b/></root>`)

	cases := []string{
		`namespace-uri-for-prefix("p", ())`,
		`namespace-uri-for-prefix("p", //root/*)`,
		`in-scope-prefixes(())`,
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(t.Context(), doc, expr)
			require.Error(t, err, expr)
			var xpErr *xpath3.XPathError
			require.ErrorAs(t, err, &xpErr)
			require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code, expr)
		})
	}
}

// resolve-QName's second argument is a REQUIRED singleton element(). An empty,
// multi-element, or non-element second argument must raise XPTY0004 even when
// the first ($qname) argument is the empty sequence.
func TestResolveQNameElementArgValidatedFirst(t *testing.T) {
	t.Parallel()

	doc := mustParseXML(t, `<root xmlns:p="urn:p"><a/><b/></root>`)

	cases := []string{
		`resolve-QName((), ())`,
		`resolve-QName((), //root/*)`,
		`resolve-QName((), "x")`,
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(t.Context(), doc, expr)
			require.Error(t, err, expr)
			var xpErr *xpath3.XPathError
			require.ErrorAs(t, err, &xpErr)
			require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code, expr)
		})
	}
}

// A single array/list argument that atomizes to multiple values must raise
// XPTY0004 (a cardinality error), not FOTY0013. The accessors must atomize
// the argument first and then enforce 0-or-1 cardinality on the result.
func TestQNameAccessorsArrayArgRejected(t *testing.T) {
	t.Parallel()

	doc := mustParseXML(t, `<root xmlns:p="urn:p"><a/><b/></root>`)

	cases := []string{
		`prefix-from-QName([QName("urn:p", "p:a"), QName("urn:p", "p:b")])`,
		`local-name-from-QName([QName("urn:p", "p:a"), QName("urn:p", "p:b")])`,
		`namespace-uri-from-QName([QName("urn:p", "p:a"), QName("urn:p", "p:b")])`,
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(t.Context(), doc, expr)
			require.Error(t, err, expr)
			var xpErr *xpath3.XPathError
			require.ErrorAs(t, err, &xpErr)
			require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code, expr)
		})
	}
}
