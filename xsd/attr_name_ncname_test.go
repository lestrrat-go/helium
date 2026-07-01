package xsd_test

import (
	"fmt"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// The {name} of an xs:attribute declaration is of type xs:NCName (XSD
// Structures §3.2.2), so an empty, colon-bearing, or otherwise non-NCName value
// is a schema-representation error. This is version-independent, so it is
// enforced under both XSD 1.0 (default) and XSD 1.1. A valid NCName — including
// dots, dashes, underscores, and non-ASCII NameChars — must still compile.
func TestAttribute_NameMustBeNCName(t *testing.T) {
	t.Parallel()

	const shell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="urn:a">
  <xs:complexType name="ct">
    %s
  </xs:complexType>
</xs:schema>`

	invalid := []struct {
		name string
		attr string
	}{
		{"empty", `<xs:attribute name=""/>`},
		{"leading-digit", `<xs:attribute name="0"/>`},
		{"apostrophe", `<xs:attribute name="&apos;"/>`},
		{"colon-declared-prefix", `<xs:attribute name="a:b"/>`},
		{"colon-undeclared-prefix", `<xs:attribute name="z:b"/>`},
		{"two-colons", `<xs:attribute name="a:b:b"/>`},
		{"leading-colon", `<xs:attribute name=":_"/>`},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.attr)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject non-NCName attribute name", v)
				require.Nil(t, schema)
				require.Contains(t, errs, "is not a valid 'NCName'", "version=%v", v)
			}
		})
	}

	valid := []struct {
		name string
		attr string
	}{
		{"simple", `<xs:attribute name="att1"/>`},
		{"underscore-start", `<xs:attribute name="_att"/>`},
		{"dots-dashes", `<xs:attribute name="a.b-c_d"/>`},
		{"non-ascii", `<xs:attribute name="naïve"/>`},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.attr)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept valid NCName attribute name: %s", v, errs)
			}
		})
	}
}

func compileWith(t *testing.T, v xsd.Version, schemaXML string) (*xsd.Schema, string, error) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, cerr := xsd.NewCompiler().
		Version(v).
		Label("test.xsd").
		ErrorHandler(collector).
		Compile(t.Context(), doc)
	_ = collector.Close()
	return schema, compileErrorsString(collector.Errors()), cerr
}
