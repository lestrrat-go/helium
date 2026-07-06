package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestVersion10AttributeGroupWildcardCompletesTypeWildcard(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attributeGroup name="targetOnly">
    <xs:anyAttribute namespace="##targetNamespace" processContents="skip"/>
  </xs:attributeGroup>
  <xs:complexType name="doc">
    <xs:sequence/>
    <xs:attributeGroup ref="t:targetOnly"/>
    <xs:anyAttribute namespace="##any" processContents="skip"/>
  </xs:complexType>
  <xs:element name="doc" type="t:doc"/>
</xs:schema>`

	t.Run("admits attribute in the group wildcard namespace", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schema,
			`<t:doc xmlns:t="urn:t" t:a="1"/>`)
		require.NoError(t, err)
	})

	t.Run("rejects attribute outside the complete wildcard intersection", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schema,
			`<t:doc xmlns:t="urn:t" xmlns:x="urn:x" x:a="1"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})
}

func TestVersion10AttributeGroupWildcardExtensionUnion(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attributeGroup name="other">
    <xs:anyAttribute namespace="##other" processContents="skip"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="withLocal">
    <xs:anyAttribute namespace="##targetNamespace ##local urn:b" processContents="skip"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="withoutLocal">
    <xs:anyAttribute namespace="##targetNamespace urn:b" processContents="skip"/>
  </xs:attributeGroup>

  <xs:complexType name="base">
    <xs:sequence/>
    <xs:attributeGroup ref="t:other"/>
  </xs:complexType>
  <xs:complexType name="anyDerived">
    <xs:complexContent>
      <xs:extension base="t:base">
        <xs:sequence/>
        <xs:attributeGroup ref="t:withLocal"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="notAbsentDerived">
    <xs:complexContent>
      <xs:extension base="t:base">
        <xs:sequence/>
        <xs:attributeGroup ref="t:withoutLocal"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="any" type="t:anyDerived"/>
  <xs:element name="notAbsent" type="t:notAbsentDerived"/>
</xs:schema>`

	t.Run("union admits any namespace when the set contributes target and local", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schema,
			`<t:any xmlns:t="urn:t" xmlns:b="urn:b" xmlns:x="urn:x" t:a="1" local="1" b:a="1" x:a="1"/>`)
		require.NoError(t, err)
	})

	t.Run("union without local rejects absent-namespace attributes", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schema,
			`<t:notAbsent xmlns:t="urn:t" local="1"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("union without local admits namespaced attributes", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schema,
			`<t:notAbsent xmlns:t="urn:t" xmlns:x="urn:x" x:a="1"/>`)
		require.NoError(t, err)
	})
}

func TestVersion10AttributeGroupWildcardExtensionUnionNotExpressible(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attributeGroup name="other">
    <xs:anyAttribute namespace="##other" processContents="skip"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="localAndB">
    <xs:anyAttribute namespace="##local urn:b" processContents="skip"/>
  </xs:attributeGroup>
  <xs:complexType name="base">
    <xs:sequence/>
    <xs:attributeGroup ref="t:other"/>
  </xs:complexType>
  <xs:complexType name="badDerived">
    <xs:complexContent>
      <xs:extension base="t:base">
        <xs:sequence/>
        <xs:attributeGroup ref="t:localAndB"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="bad" type="t:badDerived"/>
</xs:schema>`

	require.Contains(t, compileFatalErrors(t, schema), "not expressible")
}

func TestVersion10StrictWildcardLoadsInstanceSchemaLocation(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "main.xsd"
		hintXSD = "hint.xsd"
	)
	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="t:itemType"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="itemType">
    <xs:attributeGroup ref="t:strictOther"/>
  </xs:complexType>
  <xs:attributeGroup name="strictOther">
    <xs:anyAttribute namespace="##other" processContents="strict"/>
  </xs:attributeGroup>
</xs:schema>`)},
		hintXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:hint" attributeFormDefault="qualified">
  <xs:attribute name="att" type="xs:string"/>
</xs:schema>`)},
	}

	schemaSrc, err := fsys.ReadFile(mainXSD)
	require.NoError(t, err)
	schemaDoc, err := helium.NewParser().BaseURI(mainXSD).Parse(t.Context(), schemaSrc)
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().FS(fsys).BaseDir(".").Compile(t.Context(), schemaDoc)
	require.NoError(t, err)

	inst, err := helium.NewParser().BaseURI("doc.xml").Parse(t.Context(), []byte(`<t:doc
  xmlns:t="urn:t"
  xmlns:h="urn:hint"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:schemaLocation="urn:hint hint.xsd">
  <t:item h:att="ok"/>
</t:doc>`))
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), inst))
}
