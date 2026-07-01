package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// XSD 1.0 "Attribute Declaration Properties Correct" clause 3 (§3.2.6): if an
// attribute's {type definition} is or is derived from xs:ID there must NOT be a
// {value constraint} (default or fixed). XSD 1.1 removed this restriction (W3C
// bug 4077), so the check is version-gated: 1.0 rejects, 1.1 accepts.
//
// W3C msMeta/Additional coverage: addB078/A/B (fixed xs:ID attribute) and
// isDefault060_2/069 (fixed xs:ID attribute inside a named complex type).
func TestAttribute_IDMustNotHaveValueConstraint(t *testing.T) {
	t.Parallel()

	const shell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="myID">
    <xs:restriction base="xs:ID"/>
  </xs:simpleType>
  <xs:complexType name="ct">
    %s
  </xs:complexType>
</xs:schema>`

	rejected := []struct {
		name string
		attr string
	}{
		{"fixed-on-id", `<xs:attribute name="a" type="xs:ID" fixed="A1"/>`},
		{"default-on-id", `<xs:attribute name="a" type="xs:ID" default="A1"/>`},
		{"fixed-on-derived-id", `<xs:attribute name="a" type="myID" fixed="A1"/>`},
		{"required-fixed-on-id", `<xs:attribute name="a" type="xs:ID" use="required" fixed="A1"/>`},
	}
	for _, tc := range rejected {
		t.Run("invalid10/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.attr)
			schema, errs, cerr := compileWith(t, xsd.Version10, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "XSD 1.0 must reject an ID attribute with a value constraint")
			require.Nil(t, schema)
			require.Contains(t, errs, "there must not be a value constraint")
		})
		t.Run("valid11/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.attr)
			schema, _, cerr := compileWith(t, xsd.Version11, schemaXML)
			require.NoError(t, cerr, "XSD 1.1 must accept an ID attribute with a value constraint (bug 4077)")
			require.NotNil(t, schema)
		})
	}

	// A plain xs:ID attribute with no value constraint, and a non-ID attribute
	// carrying a fixed/default, must still compile under both versions.
	accepted := []struct {
		name string
		attr string
	}{
		{"id-no-constraint", `<xs:attribute name="a" type="xs:ID"/>`},
		{"string-fixed", `<xs:attribute name="a" type="xs:string" fixed="v"/>`},
		{"idref-fixed", `<xs:attribute name="a" type="xs:IDREF" fixed="A1"/>`},
	}
	for _, tc := range accepted {
		t.Run("accepted/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.attr)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, _, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept %s", v, tc.name)
				require.NotNil(t, schema)
			}
		})
	}
}
