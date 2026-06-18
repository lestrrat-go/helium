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
