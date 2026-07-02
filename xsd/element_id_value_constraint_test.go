package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// XSD 1.0 "Element Declaration Properties Correct" clause 4 (§3.3.6): if an
// element's {type definition} is or is derived from xs:ID there must NOT be a
// {value constraint} (default or fixed). This is the element counterpart of the
// attribute rule au-props-correct.3. XSD 1.1 removed the restriction (W3C bug
// 4077), so the check is version-gated: 1.0 rejects, 1.1 accepts.
//
// W3C sunData/ElemDecl/valueConstraint/valueConstraint01001m coverage:
// m2/m3 (fixed/default on an xs:ID element) and m5/m6 (fixed/default on an
// element whose type restricts xs:ID); Saxon Id id014/id015 confirm the 1.1
// relaxation.
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
		{"fixed-on-id", `<xs:element name="e" type="xs:ID" fixed="A1"/>`},
		{"default-on-id", `<xs:element name="e" type="xs:ID" default="A1"/>`},
		{"fixed-on-derived-id", `<xs:element name="e" type="myID" fixed="A1"/>`},
		{"default-on-derived-id", `<xs:element name="e" type="myID" default="A1"/>`},
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
		{"idref-fixed", `<xs:element name="e" type="xs:IDREF" fixed="A1"/>`},
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
