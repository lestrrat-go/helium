package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDOnRootDenotesNoElement covers the XSD 1.1 ID/IDREF rule that an xs:ID
// appearing as the ELEMENT CONTENT of the document root denotes no element (an
// element-content ID identifies its PARENT element, and the root has none). The
// ID is therefore never registered, so an xs:IDREF to it dangles and the instance
// is invalid (W3C ibmData idIDREF s3_3_4ii26/ii27). The same content wrapped in a
// parent element is valid, because then the ID identifies the wrapping parent.
func TestIDOnRootDenotesNoElement(t *testing.T) {
	compileValidate := func(t *testing.T, schemaXML, instanceXML string) error {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	// root: simpleContent that is a LIST of xs:ID plus an xs:IDREF attribute.
	const listSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="listOfIDs">
          <xs:attribute name="idref_attr" type="xs:IDREF"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="listOfIDs"><xs:list itemType="xs:ID"/></xs:simpleType>
</xs:schema>`

	t.Run("list of ID on document root is invalid (ii26)", func(t *testing.T) {
		t.Parallel()
		// IDs b1/b2/b3 are root content but root has no parent, so they denote no
		// element; idref_attr="b2" then has no binding.
		require.Error(t, compileValidate(t, listSchema, `<root idref_attr="b2">b1 b2 b3</root>`),
			"an ID in the document root's content denotes no element, so the IDREF dangles")
	})

	// root: simpleContent that is a UNION involving xs:ID plus an xs:IDREF attribute.
	const unionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="unionOfIDs">
          <xs:attribute name="idref_attr" type="xs:IDREF"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="unionOfIDs"><xs:union memberTypes="xs:integer xs:boolean xs:ID"/></xs:simpleType>
</xs:schema>`

	t.Run("union of ID on document root is invalid (ii27)", func(t *testing.T) {
		t.Parallel()
		require.Error(t, compileValidate(t, unionSchema, `<root idref_attr="b2">b2</root>`),
			"a union ID in the document root's content denotes no element, so the IDREF dangles")
	})

	// Same content, but the ID-bearing element is now a child, so its content ID
	// identifies its PARENT and the IDREF on a sibling resolves.
	const wrappedSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="root">
          <xs:complexType>
            <xs:simpleContent>
              <xs:extension base="listOfIDs">
                <xs:attribute name="idref_attr" type="xs:IDREF"/>
              </xs:extension>
            </xs:simpleContent>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="listOfIDs"><xs:list itemType="xs:ID"/></xs:simpleType>
</xs:schema>`

	t.Run("list of ID on a non-root element is valid (denotes parent)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileValidate(t, wrappedSchema,
			`<doc><root idref_attr="b2">b1 b2 b3</root></doc>`),
			"an element-content ID identifies its parent, so the IDREF on the same element resolves")
	})
}

// TestElementDefaultValidity covers the XSD 1.1 (Element Default Valid) check: an
// element declaration's default/fixed value must be valid against its declared
// simple type, enforced at compile time (W3C ibmData idIDREF s3_3_4si07/si08).
func TestElementDefaultValidity(t *testing.T) {
	compile := func(t *testing.T, schemaXML string) error {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
		return err
	}
	compile10 := func(t *testing.T, schemaXML string) error {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version10).Compile(t.Context(), sdoc)
		return err
	}

	// si07: default "aka" on an element whose type is a list of (xs:ID restricted
	// to the enumeration "ala"). "aka" is not "ala", so the default is invalid.
	const listSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="list_of_ids" type="listOfIDs" default="DEFVAL"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="listOfIDs"><xs:list itemType="listTypeA"/></xs:simpleType>
  <xs:simpleType name="listTypeA">
    <xs:restriction base="xs:ID"><xs:enumeration value="ala"/></xs:restriction>
  </xs:simpleType>
</xs:schema>`

	t.Run("invalid element default on list of ID-enum is a schema error (si07)", func(t *testing.T) {
		t.Parallel()
		schema := strings.ReplaceAll(listSchema, "DEFVAL", "aka")
		require.ErrorIs(t, compile(t, schema), xsd.ErrCompilationFailed)
	})

	t.Run("valid element default on list of ID-enum compiles", func(t *testing.T) {
		t.Parallel()
		schema := strings.ReplaceAll(listSchema, "DEFVAL", "ala")
		require.NoError(t, compile(t, schema))
	})

	// si08: default "id_a1" on an element of union(xs:integer, xs:boolean, xs:ID
	// restricted to enumeration "id_a"). "id_a1" matches no member, so invalid.
	const unionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="union_of_ids" type="unionOfIDs" default="DEFVAL"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="unionOfIDs"><xs:union memberTypes="xs:integer xs:boolean unionTypeA"/></xs:simpleType>
  <xs:simpleType name="unionTypeA">
    <xs:restriction base="xs:ID"><xs:enumeration value="id_a"/></xs:restriction>
  </xs:simpleType>
</xs:schema>`

	t.Run("invalid element default on union with ID is a schema error (si08)", func(t *testing.T) {
		t.Parallel()
		schema := strings.ReplaceAll(unionSchema, "DEFVAL", "id_a1")
		require.ErrorIs(t, compile(t, schema), xsd.ErrCompilationFailed)
	})

	t.Run("valid element default on union with ID compiles", func(t *testing.T) {
		t.Parallel()
		schema := strings.ReplaceAll(unionSchema, "DEFVAL", "id_a")
		require.NoError(t, compile(t, schema))
	})

	// A plain invalid default against a builtin type is also rejected at compile.
	const intSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="n" type="xs:int" default="DEFVAL"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("invalid element default on xs:int is a schema error", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compile(t, strings.ReplaceAll(intSchema, "DEFVAL", "notint")), xsd.ErrCompilationFailed)
	})

	t.Run("valid element default on xs:int compiles", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compile(t, strings.ReplaceAll(intSchema, "DEFVAL", "42")))
	})

	t.Run("XSD 1.0 does not enforce element default validity (byte-identical)", func(t *testing.T) {
		t.Parallel()
		// The element-default check is gated to 1.1; 1.0 stays as before (it does not
		// reject an invalid element default at compile time).
		require.NoError(t, compile10(t, strings.ReplaceAll(intSchema, "DEFVAL", "notint")))
	})
}
