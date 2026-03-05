package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func compileWithErrors(t *testing.T, schemaXML string) (*xsd.Schema, string) {
	t.Helper()
	schemaDOC, err := helium.Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, err := xsd.Compile(schemaDOC, xsd.WithSchemaFilename("test.xsd"), xsd.WithCompileErrorHandler(collector))
	require.NoError(t, err)
	_ = collector.Close()
	_, errs := partitionCompileErrors(collector.Errors())
	return schema, errs
}

func compileAndValidate(t *testing.T, schemaXML, instanceXML string) error {
	t.Helper()
	schema, errs := compileWithErrors(t, schemaXML)
	require.Empty(t, errs, "unexpected compile errors")
	doc, err := helium.Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)
	return xsd.Validate(doc, schema, xsd.WithFilename("test.xml"))
}

func TestBlockDefault(t *testing.T) {
	t.Run("blockDefault=#all blocks substitution group", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  blockDefault="#all">
  <xs:complexType name="baseType">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derivedType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:sequence>
          <xs:element name="extra" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="head" type="baseType"/>
  <xs:element name="member" type="derivedType" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root><member><value>hi</value><extra>more</extra></member></root>`
		err := compileAndValidate(t, schemaXML, instanceXML)
		require.Error(t, err)
	})

	t.Run("blockDefault=#all blocks xsi:type with extension", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  blockDefault="#all">
  <xs:complexType name="baseType">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derivedType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:sequence>
          <xs:element name="extra" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="baseType"/>
</xs:schema>`
		instanceXML := `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:type="derivedType"><value>hi</value><extra>more</extra></root>`
		err := compileAndValidate(t, schemaXML, instanceXML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "blocked by the element declaration")
	})
}

func TestBlockOnElement(t *testing.T) {
	t.Run("block=extension blocks xsi:type with extension-derived type", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derivedType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:sequence>
          <xs:element name="extra" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="baseType" block="extension"/>
</xs:schema>`
		instanceXML := `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:type="derivedType"><value>hi</value><extra>more</extra></root>`
		err := compileAndValidate(t, schemaXML, instanceXML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "blocked by the element declaration")
	})

	t.Run("block=substitution blocks substitution group members", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string" block="substitution"/>
  <xs:element name="member" type="xs:string" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root><member>hello</member></root>`
		err := compileAndValidate(t, schemaXML, instanceXML)
		require.Error(t, err)
	})

	t.Run("explicit block= overrides blockDefault", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  blockDefault="#all">
  <xs:element name="head" type="xs:string" block=""/>
  <xs:element name="member" type="xs:string" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root><member>hello</member></root>`
		err := compileAndValidate(t, schemaXML, instanceXML)
		require.NoError(t, err)
	})
}

func TestFinalDefault(t *testing.T) {
	t.Run("finalDefault=restriction produces compile error for restriction derivation", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  finalDefault="restriction">
  <xs:complexType name="baseType">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derivedType">
    <xs:complexContent>
      <xs:restriction base="baseType">
        <xs:sequence>
          <xs:element name="value" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="baseType"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, "Derivation by restriction is forbidden")
	})

	t.Run("finalDefault=extension produces compile error for extension derivation", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  finalDefault="extension">
  <xs:complexType name="baseType">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derivedType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:sequence>
          <xs:element name="extra" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="baseType"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, "Derivation by extension is forbidden")
	})
}

func TestFinalOnComplexType(t *testing.T) {
	t.Run("final=extension on complexType blocks extension derivation", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType" final="extension">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derivedType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:sequence>
          <xs:element name="extra" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="baseType"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, "Derivation by extension is forbidden")
	})

	t.Run("final=restriction allows extension", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType" final="restriction">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derivedType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:sequence>
          <xs:element name="extra" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="baseType"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Empty(t, errs)
	})
}

func TestFinalOnSubstGroupHead(t *testing.T) {
	t.Run("final=extension on head blocks extension-derived member", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derivedType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:sequence>
          <xs:element name="extra" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="head" type="baseType" final="extension"/>
  <xs:element name="member" type="derivedType" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, "forbidden by the head element's final value")
	})
}

func TestInvalidBlockDefaultFinalDefault(t *testing.T) {
	t.Run("invalid blockDefault produces compile error", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  blockDefault="invalid">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, "blockDefault")
		require.Contains(t, errs, "is not valid")
	})

	t.Run("invalid finalDefault produces compile error", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  finalDefault="invalid">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, "finalDefault")
		require.Contains(t, errs, "is not valid")
	})
}

func TestExplicitEmptyOverridesDefault(t *testing.T) {
	t.Run("explicit block= empty overrides blockDefault for substitution", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  blockDefault="substitution">
  <xs:element name="head" type="xs:string" block=""/>
  <xs:element name="member" type="xs:string" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root><member>hello</member></root>`
		err := compileAndValidate(t, schemaXML, instanceXML)
		require.NoError(t, err)
	})

	t.Run("explicit final= empty overrides finalDefault", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  finalDefault="#all">
  <xs:complexType name="baseType" final="">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derivedType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:sequence>
          <xs:element name="extra" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="baseType"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Empty(t, errs)
	})
}

func TestSimpleTypeFinal(t *testing.T) {
	t.Run("finalDefault=list blocks simpleType list derivation", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  finalDefault="list">
  <xs:simpleType name="myInt">
    <xs:restriction base="xs:integer"/>
  </xs:simpleType>
  <xs:simpleType name="myIntList">
    <xs:list itemType="myInt"/>
  </xs:simpleType>
  <xs:element name="root" type="myIntList"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, "Derivation by list is forbidden")
	})

	t.Run("finalDefault=union blocks simpleType union derivation", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  finalDefault="union">
  <xs:simpleType name="myStr">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="myUnion">
    <xs:union memberTypes="myStr xs:integer"/>
  </xs:simpleType>
  <xs:element name="root" type="myUnion"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, "Derivation by union is forbidden")
	})
}
