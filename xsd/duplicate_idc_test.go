package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestDuplicateIDC verifies that identity-constraint definitions (xs:key,
// xs:unique, xs:keyref) share a single symbol space: two constraints with the
// same {targetNamespace}name anywhere in the schema are a fatal collision, even
// when hosted by different element declarations. Distinct names must still
// compile cleanly.
func TestDuplicateIDC(t *testing.T) {
	t.Parallel()

	t.Run("duplicate key across two elements", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="A">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="x" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k">
      <xs:selector xpath="x"/>
      <xs:field xpath="."/>
    </xs:key>
  </xs:element>
  <xs:element name="B">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="y" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k">
      <xs:selector xpath="y"/>
      <xs:field xpath="."/>
    </xs:key>
  </xs:element>
</xs:schema>`
		require.Contains(t,
			compileErrorsExact(t, schemaXML),
			"element key: Schemas parser error : Element '{http://www.w3.org/2001/XMLSchema}key': An identity-constraint definition ''k does already exist.",
			"duplicate identity-constraint name across element declarations must be rejected")
	})

	t.Run("key and keyref share symbol space", func(t *testing.T) {
		t.Parallel()
		// A keyref colliding with a key on the same name is also a redeclaration:
		// all three IDC kinds occupy one symbol space.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="A">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="x" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k">
      <xs:selector xpath="x"/>
      <xs:field xpath="."/>
    </xs:key>
    <xs:keyref name="k" refer="k">
      <xs:selector xpath="x"/>
      <xs:field xpath="."/>
    </xs:keyref>
  </xs:element>
</xs:schema>`
		require.Contains(t,
			compileErrorsExact(t, schemaXML),
			"An identity-constraint definition ''k does already exist.",
			"a keyref sharing a key's name must be rejected as a duplicate")
	})

	t.Run("distinct names compile", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="A">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="x" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k1">
      <xs:selector xpath="x"/>
      <xs:field xpath="."/>
    </xs:key>
  </xs:element>
  <xs:element name="B">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="y" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k2">
      <xs:selector xpath="y"/>
      <xs:field xpath="."/>
    </xs:key>
  </xs:element>
</xs:schema>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Label("test.xsd").Compile(t.Context(), doc)
		require.NoError(t, err, "distinct identity-constraint names must compile cleanly")
	})
}
