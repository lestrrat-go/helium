package xsd_test

import (
	"slices"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func compileXSD11Schema(t *testing.T, schemaXML string) (*xsd.Schema, error) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	return xsd.NewCompiler().Version(xsd.Version11).Label("test.xsd").Compile(t.Context(), doc)
}

func validateXSDDocument(t *testing.T, schema *xsd.Schema, instanceXML string) error {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)
	return xsd.NewValidator(schema).Label("test.xml").Validate(t.Context(), doc)
}

func TestVersion11SubstitutionGroupList(t *testing.T) {
	t.Run("direct member can affiliate with multiple heads", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="elem0" type="xs:string"/>
  <xs:element name="elem1" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="elem0"/>
        <xs:element ref="elem1"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="elem2" type="xs:string" substitutionGroup="elem0 elem1"/>
</xs:schema>`
		schema, err := compileXSD11Schema(t, schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateXSDDocument(t, schema, `<root><elem2>a</elem2><elem2>b</elem2></root>`))
	})

	t.Run("transitive member can substitute for every listed head", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="elem0" type="xs:string"/>
  <xs:element name="elem1" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="elem0"/>
        <xs:element ref="elem1"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="elem2" type="xs:string" substitutionGroup="elem0 elem1" abstract="true"/>
  <xs:element name="elem3" type="xs:string" substitutionGroup="elem2"/>
</xs:schema>`
		schema, err := compileXSD11Schema(t, schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateXSDDocument(t, schema, `<root><elem3>a</elem3><elem3>b</elem3></root>`))
	})

	t.Run("member must be validly substitutable for each listed head", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="elem0" type="xs:string"/>
  <xs:element name="elem1" type="xs:integer"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="elem0"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="elem2" type="xs:string" substitutionGroup="elem0 elem1"/>
</xs:schema>`
		_, err := compileXSD11Schema(t, schemaXML)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("member can derive from an unfaceted union head member type", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="headType">
    <xs:union memberTypes="xs:int xs:string"/>
  </xs:simpleType>
  <xs:element name="head" type="headType"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="member" type="xs:int" substitutionGroup="head"/>
</xs:schema>`
		schema, err := compileXSD11Schema(t, schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateXSDDocument(t, schema, `<root><member>42</member></root>`))
	})

	t.Run("member cannot derive through a faceted union head", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseUnion">
    <xs:union memberTypes="xs:int xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="headType">
    <xs:restriction base="baseUnion">
      <xs:enumeration value="ok"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="head" type="headType"/>
  <xs:element name="member" type="xs:int" substitutionGroup="head"/>
</xs:schema>`
		_, err := compileXSD11Schema(t, schemaXML)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("untyped multi-head member inherits type from first head chain", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head0" type="xs:string"/>
  <xs:element name="head1" substitutionGroup="head0"/>
  <xs:element name="head2" type="xs:anySimpleType"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head1"/>
        <xs:element ref="head2"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="member" substitutionGroup="head1 head2"/>
</xs:schema>`
		schema, err := compileXSD11Schema(t, schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateXSDDocument(t, schema, `<root><member>a</member><member>b</member></root>`))
	})

	t.Run("public substitution members are transitive and block filtered", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="elem0" type="xs:string"/>
  <xs:element name="elem1" type="xs:string" block="substitution"/>
  <xs:element name="elem2" type="xs:string" substitutionGroup="elem0 elem1"/>
  <xs:element name="elem3" type="xs:string" substitutionGroup="elem2"/>
  <xs:element name="blocked" type="xs:string" substitutionGroup="elem1"/>
</xs:schema>`
		schema, err := compileXSD11Schema(t, schemaXML)
		require.NoError(t, err)

		names := make([]xsd.QName, 0, 2)
		for _, member := range schema.SubstGroupMembers(xsd.QName{Local: "elem0"}) {
			names = append(names, member.Name)
		}
		require.True(t, slices.Contains(names, xsd.QName{Local: "elem2"}))
		require.True(t, slices.Contains(names, xsd.QName{Local: "elem3"}))
		require.False(t, slices.Contains(names, xsd.QName{Local: "blocked"}))
	})
}
