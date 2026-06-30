package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileVer compiles schemaXML at the given version and returns the error.
func compileVer(t *testing.T, schemaXML string, v xsd.Version) (*xsd.Schema, error) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	return xsd.NewCompiler().Version(v).Compile(t.Context(), doc)
}

// TestXSIAttributeReferenceRequired covers saxon Complex.testSet complex009 /
// complex010: an XSD 1.1 schema may reference an xsi: processor attribute as a
// (required) attribute use. A present xsi: attribute must satisfy that use
// instead of being skipped as a special attribute and reported missing.
func TestXSIAttributeReferenceRequired(t *testing.T) {
	t.Parallel()

	const xsiTypeSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <xs:element name="root" type="B"/>
  <xs:complexType name="B">
    <xs:sequence>
      <xs:element name="e" minOccurs="0" maxOccurs="5"/>
    </xs:sequence>
    <xs:attribute ref="xsi:type" use="required"/>
  </xs:complexType>
</xs:schema>`

	schema, cerr := compileVer(t, xsiTypeSchema, xsd.Version11)
	require.NoError(t, cerr, "schema must compile")

	validate := func(t *testing.T, instanceXML string) error {
		t.Helper()
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	// A present xsi:type satisfies the required use.
	require.NoError(t, validate(t, `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="B"><e/></root>`))
	// Missing xsi:type is rejected (the required use is unmet).
	require.Error(t, validate(t, `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><e/></root>`))
}

// TestSchemaComponentIDValidity covers saxon Open.testSet open038 / open039: a
// schema-component @id must be a valid xs:ID (NCName after whitespace collapse)
// and unique within the schema document. Enforced in XSD 1.1; XSD 1.0 keeps the
// historical lenient behavior (byte-identical goldens).
func TestSchemaComponentIDValidity(t *testing.T) {
	t.Parallel()

	// open038: two ids that collapse to the same NCName ("open001").
	const dupID = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" id="open001"/>
  <xs:element name="b" id=" open001 "/>
</xs:schema>`

	// open039: an id that is not a valid NCName ("open001/2").
	const badID = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" id="open001"/>
  <xs:element name="b" id="open001/2"/>
</xs:schema>`

	_, err := compileVer(t, dupID, xsd.Version11)
	require.Error(t, err, "duplicate xs:ID must fail in 1.1")
	_, err = compileVer(t, badID, xsd.Version11)
	require.Error(t, err, "invalid NCName id must fail in 1.1")

	// XSD 1.0 stays lenient (no @id enforcement) to preserve byte-identity.
	_, err = compileVer(t, dupID, xsd.Version10)
	require.NoError(t, err, "duplicate xs:ID is tolerated in 1.0")
	_, err = compileVer(t, badID, xsd.Version10)
	require.NoError(t, err, "invalid NCName id is tolerated in 1.0")
}
