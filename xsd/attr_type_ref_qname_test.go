package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// The `type` and `ref` attributes of <xs:attribute> are of type xs:QName
// (XSD Structures §3.2.2). A present value that is not a lexically valid QName
// (a leading colon `:_`, a leading digit `123`, or the empty string on `ref`)
// is a schema-representation error. Additionally, a top-level (global)
// attribute declaration is typed by the schema-for-schemas `topLevelAttribute`,
// which omits `ref`, so a `ref` on a global attribute declaration is a schema
// error. All of these are version-independent, enforced under both XSD 1.0
// (default) and XSD 1.1.
func TestAttribute_TypeRefMustBeQName(t *testing.T) {
	t.Parallel()

	// %s is spliced inside an <xs:complexType> so a local attribute use is legal.
	const localShell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="urn:a" targetNamespace="urn:a">
  <xs:attribute name="ga" type="xs:string"/>
  <xs:complexType name="ct">
    %s
  </xs:complexType>
</xs:schema>`

	// %s is spliced directly under <xs:schema> to exercise a global declaration.
	const globalShell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="urn:a" targetNamespace="urn:a">
  <xs:attribute name="ga" type="xs:string"/>
  %s
</xs:schema>`

	const notQName = "not a valid QName"

	invalid := []struct {
		name   string
		shell  string
		attr   string
		expect string
	}{
		// @type an invalid QName (attD005/attD006).
		{"type-leading-colon", localShell, `<xs:attribute name="x" type=":_"/>`, notQName},
		{"type-leading-digit", localShell, `<xs:attribute name="x" type="123"/>`, notQName},
		{"type-leading-colon-global", globalShell, `<xs:attribute name="x" type=":_"/>`, notQName},
		// @ref an invalid QName (attE005) or present-but-empty (attE007).
		{"ref-leading-colon", localShell, `<xs:attribute ref=":_"/>`, notQName},
		{"ref-empty", localShell, `<xs:attribute ref=""/>`, notQName},
		// ref on a global attribute declaration (attKa010/attP003).
		{"ref-on-global", globalShell, `<xs:attribute ref="a:ga"/>`, "The attribute 'ref' is not allowed."},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(tc.shell, tc.attr)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject", v)
				require.Nil(t, schema)
				require.Contains(t, errs, tc.expect, "version=%v", v)
			}
		})
	}

	valid := []struct {
		name  string
		shell string
		attr  string
	}{
		{"prefixed-builtin-type", localShell, `<xs:attribute name="x" type="xs:integer"/>`},
		{"unprefixed-builtin-type", localShell, `<xs:attribute name="x" type="string" xmlns="http://www.w3.org/2001/XMLSchema"/>`},
		{"local-ref-to-global", localShell, `<xs:attribute ref="a:ga"/>`},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(tc.shell, tc.attr)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept: %s", v, errs)
				require.NotNil(t, schema)
			}
		})
	}
}
