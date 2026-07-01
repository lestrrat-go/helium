package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// The {name} of an xs:element declaration is of type xs:NCName (XSD Structures
// §3.3.2), so an empty, colon-bearing, whitespace, or otherwise non-NCName value
// is a schema-representation error. This is version-independent, so it is
// enforced under both XSD 1.0 (default) and XSD 1.1. A valid NCName — including
// dots, dashes, underscores, and non-ASCII NameChars — must still compile. The
// rule applies to both global (top-level) and local (particle) declarations.
func TestElement_NameMustBeNCName(t *testing.T) {
	t.Parallel()

	const globalShell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  %s
</xs:schema>`

	const localShell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="ct">
    <xs:sequence>
      %s
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	invalid := []struct {
		name    string
		elem    string
		isLocal bool
	}{
		{"global/qname", `<xs:element name="foo:bar"/>`, false},
		{"global/leading-colon", `<xs:element name=":bar"/>`, false},
		{"global/trailing-colon", `<xs:element name="foo:"/>`, false},
		{"global/whitespace", `<xs:element name=" "/>`, false},
		{"global/leading-dash-digit", `<xs:element name="-2.5foo"/>`, false},
		{"local/qname", `<xs:element name="foo:bar" type="xs:string"/>`, true},
		{"local/leading-digit", `<xs:element name="0" type="xs:string"/>`, true},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			shell := globalShell
			if tc.isLocal {
				shell = localShell
			}
			schemaXML := fmt.Sprintf(shell, tc.elem)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject non-NCName element name", v)
				require.Nil(t, schema)
				require.Contains(t, errs, "is not a valid 'NCName'", "version=%v", v)
			}
		})
	}

	valid := []struct {
		name    string
		elem    string
		isLocal bool
	}{
		{"global/simple", `<xs:element name="foo" type="xs:string"/>`, false},
		{"global/underscore-start", `<xs:element name="_foo"/>`, false},
		{"global/dots-dashes", `<xs:element name="a.b-c_d"/>`, false},
		{"global/non-ascii", `<xs:element name="naïve"/>`, false},
		{"local/simple", `<xs:element name="foo" type="xs:string"/>`, true},
		{"local/dots-dashes", `<xs:element name="a.b-c_d" type="xs:string"/>`, true},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			shell := globalShell
			if tc.isLocal {
				shell = localShell
			}
			schemaXML := fmt.Sprintf(shell, tc.elem)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept valid NCName element name: %s", v, errs)
			}
		})
	}
}
