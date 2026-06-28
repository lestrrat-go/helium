package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAEdges covers the XSD 1.1 conditional-type-assignment edges:
// an inline anonymous complexType / simpleType on xs:alternative, xs:error as the
// selected alternative type (which makes the element invalid), and
// xpathDefaultNamespace affecting an unprefixed @test path.
func TestVersion11CTAEdges(t *testing.T) {
	compile := func(t *testing.T, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	}
	mustCompile := func(t *testing.T, s string) *xsd.Schema {
		t.Helper()
		schema, err := compile(t, s)
		require.NoError(t, err)
		return schema
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	t.Run("inline anonymous complexType", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Pub">
    <xs:sequence><xs:element name="title" type="xs:string"/></xs:sequence>
    <xs:attribute name="kind" type="xs:string"/>
  </xs:complexType>
  <xs:element name="lib">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="pub" type="Pub" maxOccurs="unbounded">
          <xs:alternative test="@kind='book'">
            <xs:complexType>
              <xs:complexContent>
                <xs:extension base="Pub">
                  <xs:sequence><xs:element name="isbn" type="xs:string"/></xs:sequence>
                </xs:extension>
              </xs:complexContent>
            </xs:complexType>
          </xs:alternative>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		// book selects the inline extension type that requires <isbn>; cd falls back
		// to Pub which has no isbn.
		require.NoError(t, validate(t, schema,
			`<lib><pub kind="book"><title>T</title><isbn>1</isbn></pub><pub kind="cd"><title>X</title></pub></lib>`))
		// cd uses Pub, which does not permit <isbn>.
		require.ErrorIs(t, validate(t, schema,
			`<lib><pub kind="cd"><title>X</title><isbn>1</isbn></pub></lib>`), xsd.ErrValidationFailed)
		// book must carry <isbn> (inline type requires it).
		require.ErrorIs(t, validate(t, schema,
			`<lib><pub kind="book"><title>T</title></pub></lib>`), xsd.ErrValidationFailed)
	})

	t.Run("inline anonymous simpleType", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="c">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="val" type="xs:string">
          <xs:alternative test="../@mode='int'">
            <xs:simpleType><xs:restriction base="xs:int"/></xs:simpleType>
          </xs:alternative>
        </xs:element>
      </xs:sequence>
      <xs:attribute name="mode" type="xs:string"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		require.NoError(t, validate(t, schema, `<c mode="int"><val>42</val></c>`))
		require.ErrorIs(t, validate(t, schema, `<c mode="int"><val>abc</val></c>`), xsd.ErrValidationFailed)
		// mode!=int falls back to xs:string, so any text is valid.
		require.NoError(t, validate(t, schema, `<c mode="str"><val>abc</val></c>`))
	})

	t.Run("xs:error alternative makes element invalid", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="MsgT">
    <xs:sequence/>
    <xs:attribute name="kind" type="xs:string"/>
  </xs:complexType>
  <xs:element name="msg" type="MsgT">
    <xs:alternative test="@kind='bad'" type="xs:error"/>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		// kind='bad' selects xs:error: the element is invalid.
		require.ErrorIs(t, validate(t, schema, `<msg kind="bad"/>`), xsd.ErrValidationFailed)
		// otherwise the declared MsgT governs and the element is valid.
		require.NoError(t, validate(t, schema, `<msg kind="ok"/>`))
	})

	t.Run("alternatives apply through an element ref", func(t *testing.T) {
		t.Parallel()
		// Conditional type assignment lives on the GLOBAL element declaration, so a
		// child matched via <xs:element ref="msg"> must still honour the type table.
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="MsgT">
    <xs:sequence/>
    <xs:attribute name="kind" type="xs:string"/>
  </xs:complexType>
  <xs:element name="msg" type="MsgT">
    <xs:alternative test="@kind='bad'" type="xs:error"/>
  </xs:element>
  <xs:element name="msgs">
    <xs:complexType>
      <xs:sequence><xs:element ref="msg" maxOccurs="unbounded"/></xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		require.NoError(t, validate(t, schema, `<msgs><msg kind="ok"/></msgs>`))
		// The ref'd msg with kind='bad' selects xs:error and is invalid.
		require.ErrorIs(t, validate(t, schema, `<msgs><msg kind="ok"/><msg kind="bad"/></msgs>`), xsd.ErrValidationFailed)
	})

	t.Run("xpathDefaultNamespace affects unprefixed test path", func(t *testing.T) {
		t.Parallel()
		// With xpathDefaultNamespace=##targetNamespace, the unprefixed name test
		// `self::root` matches the {urn:t}root element and selects Alt (needs <y>).
		const withXDN = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:complexType name="Base"><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="Alt"><xs:sequence><xs:element name="y" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:element name="root" type="t:Base">
    <xs:alternative test="self::root" type="t:Alt" xpathDefaultNamespace="##targetNamespace"/>
  </xs:element>
</xs:schema>`
		schemaXDN := mustCompile(t, withXDN)
		require.NoError(t, validate(t, schemaXDN, `<root xmlns="urn:t"><y>v</y></root>`))
		require.ErrorIs(t, validate(t, schemaXDN, `<root xmlns="urn:t"><x>v</x></root>`), xsd.ErrValidationFailed)

		// Without xpathDefaultNamespace, `self::root` resolves to {}root which does
		// not match {urn:t}root, so the alternative never fires and Base governs.
		const noXDN = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:complexType name="Base"><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="Alt"><xs:sequence><xs:element name="y" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:element name="root" type="t:Base">
    <xs:alternative test="self::root" type="t:Alt"/>
  </xs:element>
</xs:schema>`
		schemaNo := mustCompile(t, noXDN)
		require.NoError(t, validate(t, schemaNo, `<root xmlns="urn:t"><x>v</x></root>`))
		require.ErrorIs(t, validate(t, schemaNo, `<root xmlns="urn:t"><y>v</y></root>`), xsd.ErrValidationFailed)
	})
}
