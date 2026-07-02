package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// XSD 1.0 §3.3.6 (the element analog of au-props-correct.3 §3.2.6): if an
// element's {type definition} is or is derived from xs:ID there must NOT be a
// {value constraint} (default or fixed). XSD 1.1 removed this restriction (W3C
// bug 4077), so the check is version-gated: 1.0 rejects, 1.1 accepts.
//
// W3C msMeta/Element_w3c coverage: elemZ032a (xs:ID element with default) and
// elemZ032b (an element typed by a user type derived from xs:ID).
func TestElement_IDMustNotHaveValueConstraint(t *testing.T) {
	t.Parallel()

	const shell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="myID">
    <xs:restriction base="xs:ID"/>
  </xs:simpleType>
  %s
</xs:schema>`

	rejected := []struct {
		name string
		elem string
	}{
		{"default-on-id", `<xs:element name="e" type="xs:ID" default="a"/>`},
		{"fixed-on-id", `<xs:element name="e" type="xs:ID" fixed="a"/>`},
		{"default-on-derived-id", `<xs:element name="e" type="myID" default="a"/>`},
		{"fixed-on-derived-id", `<xs:element name="e" type="myID" fixed="a"/>`},
	}
	for _, tc := range rejected {
		t.Run("invalid10/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.elem)
			schema, errs, cerr := compileWith(t, xsd.Version10, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "XSD 1.0 must reject an ID element with a value constraint")
			require.Nil(t, schema)
			require.Contains(t, errs, "there must not be a value constraint")
		})
		t.Run("valid11/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.elem)
			schema, _, cerr := compileWith(t, xsd.Version11, schemaXML)
			require.NoError(t, cerr, "XSD 1.1 must accept an ID element with a value constraint (bug 4077)")
			require.NotNil(t, schema)
		})
	}

	// A plain xs:ID element with no value constraint, and a non-ID element
	// carrying a fixed/default, must still compile under both versions.
	accepted := []struct {
		name string
		elem string
	}{
		{"id-no-constraint", `<xs:element name="e" type="xs:ID"/>`},
		{"string-fixed", `<xs:element name="e" type="xs:string" fixed="v"/>`},
		{"idref-default", `<xs:element name="e" type="xs:IDREF" default="a"/>`},
	}
	for _, tc := range accepted {
		t.Run("accepted/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.elem)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, _, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept %s", v, tc.name)
				require.NotNil(t, schema)
			}
		})
	}
}
