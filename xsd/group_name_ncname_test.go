package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// The {name} of a global model group definition is of type xs:NCName (XSD
// Structures §3.7.2), so a colon-bearing or otherwise non-NCName value is a
// schema-representation error. This is version-independent, so it is enforced
// under both XSD 1.0 (default) and XSD 1.1. A valid NCName must still compile.
func TestGroup_NameMustBeNCName(t *testing.T) {
	t.Parallel()

	const shell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="urn:a">
  <xs:group name=%q>
    <xs:sequence>
      <xs:element name="a"/>
    </xs:sequence>
  </xs:group>
</xs:schema>`

	invalid := []struct {
		name  string
		gname string
	}{
		{"leading-digit", "1"},
		{"colon-declared-prefix", "a:b"},
		{"colon-undeclared-prefix", "z:b"},
		{"two-colons", "a:b:b"},
		{"leading-colon", ":_"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.gname)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject non-NCName group name", v)
				require.Nil(t, schema)
				require.Contains(t, errs, "is not a valid 'xs:NCName'", "version=%v", v)
			}
		})
	}

	valid := []struct {
		name  string
		gname string
	}{
		{"simple", "grp1"},
		{"underscore-start", "_grp"},
		{"dots-dashes", "a.b-c_d"},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.gname)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept valid NCName group name: %s", v, errs)
			}
		})
	}
}
