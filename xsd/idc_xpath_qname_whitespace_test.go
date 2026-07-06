package xsd_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIDCXPathQNameWhitespace verifies that whitespace adjacent to the SINGLE
// ':' of a name-test QName (Prefix ':' LocalPart) or prefixed wildcard
// (NCName ':' '*') is rejected in an identity-constraint selector/field @xpath.
// A QName is a single lexical token in the restricted §3.11.6 subset, so
// 'xpns: *' (a space after the colon) is outside the grammar even though the
// xpath1 lexer tolerates it. Mirrors the W3C xsd10 msMeta case idJ016
// (field xpath='xpns: *'). The '::' axis separator is a distinct token and its
// surrounding whitespace stays tolerated (idI024/idJ033), covered by
// TestIDCXPathAxisWhitespace.
func TestIDCXPathQNameWhitespace(t *testing.T) {
	t.Parallel()

	const tmpl = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xpns="xpns.org">
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

	const (
		selDesc = ".//item"
		fieldID = "@id"
	)

	t.Run("rejects whitespace after the QName colon", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ selector, field string }{
			{selDesc, "xpns: *"},    // idJ016: space after colon before '*'
			{selDesc, "xpns: id"},   // space after colon before an NCName
			{"xpns: item", fieldID}, // in the selector too
		} {
			errs := compileSchemaErrors(t, fmt.Sprintf(tmpl, tc.selector, tc.field))
			require.NotEmpty(t, errs, "selector=%q field=%q should be rejected", tc.selector, tc.field)
		}
	})

	t.Run("accepts the whitespace-free prefixed name test", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ selector, field string }{
			{selDesc, "xpns:*"},
			{selDesc, fieldID},
			{"item", fieldID},
		} {
			errs := compileSchemaErrors(t, fmt.Sprintf(tmpl, tc.selector, tc.field))
			require.Empty(t, errs, "selector=%q field=%q should compile clean: %s", tc.selector, tc.field, errs)
		}
	})
}
