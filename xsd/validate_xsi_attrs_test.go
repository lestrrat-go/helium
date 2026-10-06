package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestNonProcessorXsiAttributes checks that only the four xsi: processor
// attributes (xsi:type, xsi:nil, xsi:schemaLocation,
// xsi:noNamespaceSchemaLocation) are exempt from attribute-use matching
// (cvc-complex-type clause 3, XSD 1.0 and 1.1). Any other attribute in the xsi
// namespace is an ordinary attribute: without an attribute wildcard that admits
// it, it is not allowed. xmllint rejects it the same way.
func TestNonProcessorXsiAttributes(t *testing.T) {
	t.Parallel()

	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	validate := func(t *testing.T, v xsd.Version, schemaXML, instanceXML string) (string, error) {
		t.Helper()
		schema, errs, cerr := compileWith(t, v, schemaXML)
		require.NoError(t, cerr, errs)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		verr := xsd.NewValidator(schema).Label("test.xml").ErrorHandler(collector).Validate(t.Context(), doc)
		_ = collector.Close()
		var b strings.Builder
		for _, e := range collector.Errors() {
			b.WriteString(e.Error())
		}
		return b.String(), verr
	}

	const noAttrs = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" nillable="true"><xs:complexType/></xs:element>
</xs:schema>`
	const withAttr = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="T"/>
  <xs:complexType name="T"><xs:attribute name="a"/></xs:complexType>
</xs:schema>`
	const laxWildcard = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:anyAttribute namespace="##any" processContents="lax"/></xs:complexType></xs:element>
</xs:schema>`
	const strictWildcard = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:anyAttribute namespace="##any" processContents="strict"/></xs:complexType></xs:element>
</xs:schema>`

	for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
		t.Run(v.String(), func(t *testing.T) {
			t.Parallel()

			t.Run("rejected on a type without attribute uses", func(t *testing.T) {
				t.Parallel()
				out, err := validate(t, v, noAttrs, `<root `+xsiNS+` xsi:bogus="1"/>`)
				require.Error(t, err)
				require.Contains(t, out, "bogus' is not allowed")
			})

			t.Run("rejected on a type with attribute uses", func(t *testing.T) {
				t.Parallel()
				out, err := validate(t, v, withAttr, `<root `+xsiNS+` a="x" xsi:bogus="1"/>`)
				require.Error(t, err)
				require.Contains(t, out, "bogus' is not allowed")
			})

			t.Run("processor attributes stay exempt", func(t *testing.T) {
				t.Parallel()
				out, err := validate(t, v, noAttrs, `<root `+xsiNS+` xsi:nil="true" xsi:schemaLocation="urn:a a.xsd" xsi:noNamespaceSchemaLocation="b.xsd"/>`)
				require.NoError(t, err, out)
				out, err = validate(t, v, withAttr, `<root `+xsiNS+` a="x" xsi:type="T"/>`)
				require.NoError(t, err, out)
			})

			t.Run("admitted by a lax wildcard", func(t *testing.T) {
				t.Parallel()
				_, err := validate(t, v, laxWildcard, `<root `+xsiNS+` xsi:bogus="1"/>`)
				require.NoError(t, err)
			})

			t.Run("strict wildcard needs a declaration", func(t *testing.T) {
				t.Parallel()
				_, err := validate(t, v, strictWildcard, `<root `+xsiNS+` xsi:bogus="1"/>`)
				require.Error(t, err)
			})
		})
	}
}
