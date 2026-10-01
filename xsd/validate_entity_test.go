package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestValidateEntities checks the XSD 1.1 xs:ENTITY value-space pass: every
// ENTITY token must name an unparsed entity the instance DTD declares, whether
// the ENTITY type comes from a declaration or only from an instance xsi:type.
func TestValidateEntities(t *testing.T) {
	t.Parallel()

	const noEntity = "There is no unparsed entity declared for the ENTITY value"
	const dtd = `<!DOCTYPE root [
<!NOTATION gif SYSTEM "image/gif">
<!ENTITY pic SYSTEM "pic.gif" NDATA gif>
]>
`

	validate := func(t *testing.T, version xsd.Version, schemaXML, instance string) (string, error) {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(version).Compile(t.Context(), sdoc)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		verr := xsd.NewValidator(schema).ErrorHandler(collector).Validate(t.Context(), doc)
		require.NoError(t, collector.Close())
		var msgs strings.Builder
		for _, e := range collector.Errors() {
			msgs.WriteString(e.Error())
		}
		return msgs.String(), verr
	}

	const attrSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:attribute name="refs" type="xs:ENTITIES"/></xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("declared ENTITIES attribute names a declared entity", func(t *testing.T) {
		t.Parallel()
		_, err := validate(t, xsd.Version11, attrSchema, dtd+`<root refs="pic"/>`)
		require.NoError(t, err)
	})

	t.Run("declared ENTITIES attribute names an undeclared entity", func(t *testing.T) {
		t.Parallel()
		msgs, err := validate(t, xsd.Version11, attrSchema, dtd+`<root refs="pic nope"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
		require.Contains(t, msgs, noEntity+" 'nope' (attribute 'refs').")
	})

	// The schema declares no ENTITY-typed component; only the instance's
	// xsi:type makes the element content an xs:ENTITY value.
	const anySchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:anySimpleType"/>
</xs:schema>`
	const xsiEntity = `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:ENTITY">%s</root>`

	t.Run("xsi:type ENTITY names a declared entity", func(t *testing.T) {
		t.Parallel()
		_, err := validate(t, xsd.Version11, anySchema, dtd+strings.Replace(xsiEntity, "%s", "pic", 1))
		require.NoError(t, err)
	})

	t.Run("xsi:type ENTITY names an undeclared entity", func(t *testing.T) {
		t.Parallel()
		msgs, err := validate(t, xsd.Version11, anySchema, strings.Replace(xsiEntity, "%s", "nope", 1))
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
		require.Contains(t, msgs, noEntity+" 'nope'.")
	})

	t.Run("1.0 does not check ENTITY values", func(t *testing.T) {
		t.Parallel()
		_, err := validate(t, xsd.Version10, anySchema, strings.Replace(xsiEntity, "%s", "nope", 1))
		require.NoError(t, err)
	})
}
