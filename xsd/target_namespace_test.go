package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestVersion11LocalAttributeTargetNamespace(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="x">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="TEST_TYPE">
          <xs:attribute name="a" type="xs:integer" targetNamespace="http://test1"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="TEST_TYPE">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:anyAttribute/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`

	require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<x xmlns:test1="http://test1" test1:a="100">Hello World</x>`))
	require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<x a="100">Hello World</x>`), xsd.ErrValidationFailed)
}

func TestVersion11LocalElementTargetNamespace(t *testing.T) {
	t.Parallel()

	t.Run("schema namespace overrides unqualified form default", func(t *testing.T) {
		t.Parallel()

		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:root" xmlns:r="urn:root">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" targetNamespace="urn:root" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
			`<r:root xmlns:r="urn:root"><r:item>100</r:item></r:root>`))
		require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
			`<r:root xmlns:r="urn:root"><item>100</item></r:root>`), xsd.ErrValidationFailed)
	})

	t.Run("cross namespace allowed in non-anyType restriction", func(t *testing.T) {
		t.Parallel()

		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:root" xmlns:r="urn:root">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:any namespace="##any" processContents="skip"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="restricted">
    <xs:complexContent>
      <xs:restriction base="r:base">
        <xs:sequence>
          <xs:element name="item" targetNamespace="urn:item" type="xs:int"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="r:restricted"/>
</xs:schema>`

		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
			`<r:root xmlns:r="urn:root" xmlns:i="urn:item"><i:item>100</i:item></r:root>`))
		require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
			`<r:root xmlns:r="urn:root"><item>100</item></r:root>`), xsd.ErrValidationFailed)
	})

	t.Run("empty explicit namespace overrides qualified form default in restriction", func(t *testing.T) {
		t.Parallel()

		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:root" xmlns:r="urn:root" elementFormDefault="qualified">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:any namespace="##any" processContents="skip"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="restricted">
    <xs:complexContent>
      <xs:restriction base="r:base">
        <xs:sequence>
          <xs:element name="item" targetNamespace="" type="xs:int"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="r:restricted"/>
</xs:schema>`

		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
			`<r:root xmlns:r="urn:root"><item>100</item></r:root>`))
		require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
			`<r:root xmlns:r="urn:root"><r:item>100</r:item></r:root>`), xsd.ErrValidationFailed)
	})

	t.Run("chameleon include uses effective target namespace", func(t *testing.T) {
		t.Parallel()

		const mainXSD = "main.xsd"
		const incXSD = "inc.xsd"
		const ns = "urn:t"
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="` + ns + `" xmlns:t="` + ns + `" elementFormDefault="qualified">
  <xs:include schemaLocation="inc.xsd"/>
</xs:schema>`)},
			incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="` + ns + `" elementFormDefault="qualified">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" targetNamespace="` + ns + `" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`)},
		}

		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().
			Version(xsd.Version11).
			Label(mainXSD).
			BaseDir(".").
			FS(fsys).
			Compile(t.Context(), doc)
		require.NoError(t, err)

		idoc, err := helium.NewParser().Parse(t.Context(), []byte(`<t:root xmlns:t="`+ns+`"><t:item>100</t:item></t:root>`))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), idoc))
	})
}

func TestVersion11LocalElementTargetNamespaceRepresentationConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name: "requires name",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element targetNamespace="urn:item" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "not allowed with ref",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item" targetNamespace="urn:item"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "not allowed with form",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" targetNamespace="urn:item" form="qualified"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed outside restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" targetNamespace="urn:item"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "empty target namespace not allowed without schema target namespace outside restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" targetNamespace=""/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "empty target namespace not allowed with explicit empty schema target namespace outside restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" targetNamespace=""/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed in anyType restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="restricted">
    <xs:complexContent>
      <xs:restriction base="xs:anyType">
        <xs:sequence>
          <xs:element name="item" targetNamespace="urn:item"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed on nested local element inside restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:element name="item" type="xs:anyType"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="restricted">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:element name="item">
            <xs:complexType>
              <xs:sequence>
                <xs:element name="nested" targetNamespace="urn:nested"/>
              </xs:sequence>
            </xs:complexType>
          </xs:element>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
			require.NoError(t, err)

			_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		})
	}
}

func TestVersion11LocalElementTargetNamespaceRefSource(t *testing.T) {
	t.Parallel()

	const mainXSD = "main.xsd"
	const incXSD = "inc.xsd"
	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="inc.xsd"/>
</xs:schema>`)},
		incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item" targetNamespace="urn:item"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`)},
	}
	data, err := fsys.ReadFile(mainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = xsd.NewCompiler().
		Version(xsd.Version11).
		Label(mainXSD).
		BaseDir(".").
		FS(fsys).
		ErrorHandler(collector).
		Compile(t.Context(), doc)
	require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	errs := compileErrorsString(collector.Errors())
	require.Contains(t, errs, incXSD+":")
	require.Contains(t, errs, "attribute 'targetNamespace'")
}

func TestVersion11LocalAttributeTargetNamespaceRepresentationConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name: "requires name",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="x">
    <xs:complexType>
      <xs:attribute targetNamespace="urn:other" type="xs:string"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "not allowed with ref",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="x">
    <xs:complexType>
      <xs:attribute ref="a" targetNamespace="urn:other"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "not allowed with form",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:anyAttribute/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="x">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="base">
          <xs:attribute name="a" type="xs:string" targetNamespace="urn:other" form="qualified"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed in extension",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="x">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="base">
          <xs:attribute name="a" type="xs:string" targetNamespace="urn:other"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed in attribute group",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="attrs">
    <xs:attribute name="a" type="xs:string" targetNamespace="urn:other"/>
  </xs:attributeGroup>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed in anyType restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="restricted">
    <xs:complexContent>
      <xs:restriction base="xs:anyType">
        <xs:attribute name="a" type="xs:string" targetNamespace="urn:other"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed on nested element attribute inside restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:element name="item" type="xs:anyType"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="restricted">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:element name="item">
            <xs:complexType>
              <xs:attribute name="a" type="xs:string" targetNamespace="urn:other"/>
            </xs:complexType>
          </xs:element>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`,
		},
		{
			name: "xsi namespace not allowed in valid restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:anyAttribute/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="x">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="base">
          <xs:attribute name="local" type="xs:string" targetNamespace="http://www.w3.org/2001/XMLSchema-instance"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
			require.NoError(t, err)

			_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		})
	}
}
