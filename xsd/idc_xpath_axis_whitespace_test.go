package xsd_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIDCXPathAxisWhitespace verifies that an identity-constraint selector/field
// @xpath tolerates insignificant whitespace around the '::' axis separator, which
// XPath permits (ExprWhitespace) — e.g. 'child ::item' / 'attribute ::id'. The
// restricted IDC XPath subset (XSD Structures 3.11.6) constrains the grammar
// (no node-type tests, limited '//', bound prefixes) but does NOT forbid
// whitespace around '::'; libxml2/Xerces accept these. Mirrors the W3C xsd10
// msMeta cases idI024 (selector 'child ::imp:iid') and idJ033 (field
// 'attribute ::imp:sid').
func TestIDCXPathAxisWhitespace(t *testing.T) {
	t.Parallel()

	const tmpl = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="u">
      <xs:selector xpath="%s"/>
      <xs:field xpath="%s"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	t.Run("whitespace around :: compiles clean", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ selector, field string }{
			{"child ::item", "@id"},
			{"child:: item", "@id"},
			{"child :: item", "@id"},
			{"item", "attribute ::id"},
			{"item", "attribute :: id"},
		} {
			errs := compileSchemaErrors(t, fmt.Sprintf(tmpl, tc.selector, tc.field))
			require.Empty(t, errs, "selector=%q field=%q", tc.selector, tc.field)
		}
	})

	// The whitespace relaxation must not loosen the genuine §3.11.6 subset
	// restrictions: a node-type test is still rejected.
	t.Run("node-type test still rejected", func(t *testing.T) {
		t.Parallel()
		errs := compileSchemaErrors(t, fmt.Sprintf(tmpl, "child ::node()", "@id"))
		require.NotEmpty(t, errs)
	})
}
