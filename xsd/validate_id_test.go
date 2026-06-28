package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestValidateIDIDREF covers XSD 1.1 document-wide xs:ID / xs:IDREF / xs:IDREFS
// validation: ID values must be unique across the document, except that the same
// value may identify a single element more than once (multiple ID attributes of
// one element, or multiple ID children of one parent); every IDREF token must
// resolve to some ID.
func TestValidateIDIDREF(t *testing.T) {
	compileValidate := func(t *testing.T, version xsd.Version, schemaXML, instanceXML string) error {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(version).Compile(t.Context(), sdoc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	// An element type carrying two xs:ID attributes (legal only in XSD 1.1).
	const multiIDSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="para" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="para">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:string">
          <xs:attribute name="id-one" type="xs:ID"/>
          <xs:attribute name="id-two" type="xs:ID"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("duplicate ID across different elements is invalid", func(t *testing.T) {
		t.Parallel()
		inst := `<doc><para id-one="aaa" id-two="bbb"/><para id-one="ccc" id-two="aaa"/></doc>`
		require.Error(t, compileValidate(t, xsd.Version11, multiIDSchema, inst))
	})

	t.Run("same ID on two attributes of one element is valid", func(t *testing.T) {
		t.Parallel()
		inst := `<doc><para id-one="eee" id-two="eee"/></doc>`
		require.NoError(t, compileValidate(t, xsd.Version11, multiIDSchema, inst))
	})

	t.Run("whitespace-collapsed duplicate ID is invalid", func(t *testing.T) {
		t.Parallel()
		inst := `<doc><para id-one="aaa" id-two="bbb"/><para id-one="ccc" id-two=" aaa "/></doc>`
		require.Error(t, compileValidate(t, xsd.Version11, multiIDSchema, inst))
	})

	t.Run("XSD 1.0 does not enforce ID uniqueness", func(t *testing.T) {
		t.Parallel()
		// The same duplicate that is invalid in 1.1 stays accepted in 1.0 mode,
		// which keeps helium byte-identical with the libxml2-compat goldens.
		inst := `<doc><para id-one="aaa" id-two="bbb"/><para id-one="ccc" id-two="aaa"/></doc>`
		require.NoError(t, compileValidate(t, xsd.Version10, multiIDSchema, inst))
	})

	// Element-content ID is owned by the PARENT element, so two <id> children of
	// one parent may share a value, but the same value under two parents collides.
	const elemIDSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="node" maxOccurs="unbounded">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="id" type="xs:ID" maxOccurs="unbounded"/>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("two ID children of one parent may share a value", func(t *testing.T) {
		t.Parallel()
		inst := `<root><node><id>zzz</id><id>zzz</id></node></root>`
		require.NoError(t, compileValidate(t, xsd.Version11, elemIDSchema, inst))
	})

	t.Run("same ID under two different parents is invalid", func(t *testing.T) {
		t.Parallel()
		inst := `<root><node><id>zzz</id></node><node><id>zzz</id></node></root>`
		require.Error(t, compileValidate(t, xsd.Version11, elemIDSchema, inst))
	})

	// IDREF / IDREFS resolution.
	const idrefSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:ID"/>
            <xs:attribute name="ref" type="xs:IDREF"/>
            <xs:attribute name="refs" type="xs:IDREFS"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("IDREF resolving to an existing ID is valid", func(t *testing.T) {
		t.Parallel()
		inst := `<root><a id="x"/><a ref="x"/></root>`
		require.NoError(t, compileValidate(t, xsd.Version11, idrefSchema, inst))
	})

	t.Run("dangling IDREF is invalid", func(t *testing.T) {
		t.Parallel()
		inst := `<root><a id="x"/><a ref="y"/></root>`
		require.Error(t, compileValidate(t, xsd.Version11, idrefSchema, inst))
	})

	t.Run("IDREFS all resolving is valid, one dangling is invalid", func(t *testing.T) {
		t.Parallel()
		ok := `<root><a id="x"/><a id="y"/><a refs="x y"/></root>`
		require.NoError(t, compileValidate(t, xsd.Version11, idrefSchema, ok))
		bad := `<root><a id="x"/><a id="y"/><a refs="x y z"/></root>`
		require.Error(t, compileValidate(t, xsd.Version11, idrefSchema, bad))
	})
}
