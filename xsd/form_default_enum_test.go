package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// The schema-level elementFormDefault / attributeFormDefault attributes are of
// the schema-for-schemas `formChoice` type (§3.1 Layer 3), an enumeration over
// {qualified, unqualified}. Any other value — the empty string, a capitalized
// "Qualified"/"Unqualified", or a two-token "qualified unqualified" — is a
// schema-representation error. The rule is version-independent, so it is
// enforced under XSD 1.0 (default) and 1.1. Covers W3C msMeta/Element_w3c
// elemH003-elemH006.
func TestSchema_FormDefaultEnum(t *testing.T) {
	t.Parallel()

	const shell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" %s="%s">
  <xs:element name="myElem" type="xs:string"/>
</xs:schema>`

	const elemForm = "elementFormDefault"
	const attrForm = "attributeFormDefault"

	invalid := []struct {
		name string
		attr string
		val  string
	}{
		{"element-empty", elemForm, ""},              // elemH003
		{"element-Qualified", elemForm, "Qualified"}, // elemH004
		{"element-Unqualified", elemForm, "Unqualified"},
		{"element-two-token", elemForm, "qualified unqualified"}, // elemH006
		{"element-foo", elemForm, "foo"},
		{"attribute-empty", attrForm, ""},
		{"attribute-Qualified", attrForm, "Qualified"},
		{"attribute-two-token", attrForm, "qualified unqualified"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.attr, tc.val)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject invalid %s", v, tc.attr)
				require.Nil(t, schema)
				require.Contains(t, errs, "Expected is '(qualified | unqualified)'.", "version=%v", v)
			}
		})
	}

	valid := []struct {
		name string
		attr string
		val  string
	}{
		{"element-qualified", elemForm, "qualified"},
		{"element-unqualified", elemForm, "unqualified"},
		{"attribute-qualified", attrForm, "qualified"},
		{"attribute-unqualified", attrForm, "unqualified"},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.attr, tc.val)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept valid %s: %s", v, tc.attr, errs)
			}
		})
	}
}
