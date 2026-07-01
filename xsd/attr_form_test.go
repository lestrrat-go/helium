package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// The `form` attribute of <xs:attribute> is governed by the schema for schemas
// (XSD Structures §3.2.2): a top-level (global) attribute declaration has the
// `topLevelAttribute` type, which omits `form`, so `form` on a global attribute
// is a schema-representation error; a local attribute's `form` must be one of
// the `formChoice` enumeration {qualified, unqualified}. Both rules are
// version-independent, so they are enforced under XSD 1.0 (default) and 1.1.
func TestAttribute_FormRepresentation(t *testing.T) {
	t.Parallel()

	const localShell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="ct">
    %s
  </xs:complexType>
</xs:schema>`

	const globalShell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  %s
</xs:schema>`

	invalid := []struct {
		name   string
		shell  string
		attr   string
		expect string
	}{
		// @form on a global attribute declaration is not allowed (attA001/attA002).
		{"global-qualified", globalShell, `<xs:attribute name="test" form="qualified"/>`, "The attribute 'form' is not allowed."},
		{"global-unqualified", globalShell, `<xs:attribute name="test" form="unqualified"/>`, "The attribute 'form' is not allowed."},
		// Bad @form enum values on a local attribute (attA003-006).
		{"local-foo", localShell, `<xs:attribute name="test" form="foo"/>`, "The value must be one of 'qualified' or 'unqualified'."},
		{"local-empty", localShell, `<xs:attribute name="test" form=""/>`, "The value must be one of 'qualified' or 'unqualified'."},
		{"local-Qualified", localShell, `<xs:attribute name="test" form="Qualified"/>`, "The value must be one of 'qualified' or 'unqualified'."},
		{"local-Unqualified", localShell, `<xs:attribute name="test" form="Unqualified"/>`, "The value must be one of 'qualified' or 'unqualified'."},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(tc.shell, tc.attr)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject invalid @form", v)
				require.Nil(t, schema)
				require.Contains(t, errs, tc.expect, "version=%v", v)
			}
		})
	}

	valid := []struct {
		name string
		attr string
	}{
		{"qualified", `<xs:attribute name="test" form="qualified"/>`},
		{"unqualified", `<xs:attribute name="test" form="unqualified"/>`},
		{"absent", `<xs:attribute name="test"/>`},
	}
	for _, tc := range valid {
		t.Run("valid/local/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(localShell, tc.attr)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept valid local @form: %s", v, errs)
			}
		})
	}
}
