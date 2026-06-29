package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestVersion11DefaultAttributesApply(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t"
  elementFormDefault="qualified" attributeFormDefault="qualified"
  defaultAttributes="t:defaults">
  <xs:attributeGroup name="defaults">
    <xs:attribute name="defaultAttr" type="xs:boolean" use="required"/>
  </xs:attributeGroup>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="implicit">
          <xs:complexType/>
        </xs:element>
        <xs:element name="explicitTrue">
          <xs:complexType defaultAttributesApply="true"/>
        </xs:element>
        <xs:element name="explicitFalse">
          <xs:complexType defaultAttributesApply="false"/>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<t:root xmlns:t="urn:t" t:defaultAttr="true">
  <t:implicit t:defaultAttr="true"/>
  <t:explicitTrue t:defaultAttr="false"/>
  <t:explicitFalse/>
</t:root>`))

	require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<t:root xmlns:t="urn:t" t:defaultAttr="true">
  <t:implicit/>
  <t:explicitTrue t:defaultAttr="false"/>
  <t:explicitFalse/>
</t:root>`), xsd.ErrValidationFailed)
}

func TestVersion11DefaultAttributesDuplicateWithExplicitGroup(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t"
  attributeFormDefault="qualified" defaultAttributes="t:defaults">
  <xs:attributeGroup name="defaults">
    <xs:attribute name="a" type="xs:boolean"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="more">
    <xs:attribute name="a" type="xs:boolean" use="required"/>
  </xs:attributeGroup>
  <xs:element name="root" type="t:T"/>
  <xs:complexType name="T" defaultAttributesApply="true">
    <xs:attributeGroup ref="t:more"/>
  </xs:complexType>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)

	_, err = xsd.NewCompiler().Version(xsd.Version11).Label("test.xsd").Compile(t.Context(), doc)
	require.ErrorIs(t, err, xsd.ErrCompilationFailed)
}

func TestVersion11DefaultAttributesMissingGroupFailsCompile(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t" defaultAttributes="t:missing">
  <xs:element name="root">
    <xs:complexType/>
  </xs:element>
</xs:schema>`

	_, err := compileDefaultAttributesSchema(t, xsd.NewCompiler().Version(xsd.Version11).Label("test.xsd"), schemaXML)
	require.ErrorIs(t, err, xsd.ErrCompilationFailed)
}

func TestVersion11DefaultAttributesInvalidQNameFailsCompile(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		ref       string
		rootAttrs string
		want      string
	}{
		{name: "leading colon", ref: ":defaults", want: "is not a valid QName"},
		{name: "empty", ref: "", want: "is not a valid QName"},
		{name: "contains whitespace", ref: "bad name", want: "is not a valid QName"},
		{name: "unbound prefix", ref: "p:missing", want: "not bound to a namespace"},
		{
			name:      "deprecated datatypes namespace",
			ref:       "old:missing",
			rootAttrs: ` xmlns:old="http://www.w3.org/2001/XMLSchema-datatypes"`,
			want:      "has been deprecated",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` + tt.rootAttrs + ` defaultAttributes="` + tt.ref + `">
  <xs:attributeGroup name="defaults">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
  <xs:complexType name="T"/>
</xs:schema>`

			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			_, err := compileDefaultAttributesSchema(t, xsd.NewCompiler().
				Version(xsd.Version11).
				Label("test.xsd").
				ErrorHandler(collector), schemaXML)
			_ = collector.Close()

			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
			errs := compileErrorsString(collector.Errors())
			require.Contains(t, errs, tt.want)
			require.NotContains(t, errs, "does not resolve to a(n) attribute group definition")
		})
	}
}

func TestVersion11DefaultAttributesMissingGroupFailsWithoutApplyingType(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		schema string
	}{
		{
			name: "no complex types",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  defaultAttributes="missing"/>`,
		},
		{
			name: "all complex types opt out",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  defaultAttributes="missing">
  <xs:complexType name="T" defaultAttributesApply="false"/>
</xs:schema>`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			_, err := compileDefaultAttributesSchema(t, xsd.NewCompiler().
				Version(xsd.Version11).
				Label("test.xsd").
				ErrorHandler(collector), tt.schema)
			_ = collector.Close()

			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
			errs := compileErrorsString(collector.Errors())
			require.Contains(t, errs, "defaultAttributes")
			require.Contains(t, errs, "does not resolve to a(n) attribute group definition")
		})
	}
}

func TestVersion11DefaultAttributesMissingGroupInIncludeFailsCompile(t *testing.T) {
	t.Parallel()

	const mainXSD = "main.xsd"
	const incXSD = "inc.xsd"
	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="t:Included"/>
</xs:schema>`)},
		incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t" defaultAttributes="t:missing">
  <xs:complexType name="Included"/>
</xs:schema>`)},
	}
	data, err := fsys.ReadFile(mainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = xsd.NewCompiler().Version(xsd.Version11).Label(mainXSD).FS(fsys).ErrorHandler(collector).Compile(t.Context(), doc)
	requireCompileResultErr(t, err)
	errs := compileErrorsString(collector.Errors())
	require.Contains(t, errs, incXSD)
	require.Contains(t, errs, "defaultAttributes")
}

func TestVersion11DefaultAttributesAfterExplicitGroups(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" defaultAttributes="defaults">
  <xs:attributeGroup name="explicit">
    <xs:attribute name="explicit" type="xs:string" default="yes"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="defaults">
    <xs:attribute name="schemaDefault" type="xs:string" default="yes"/>
  </xs:attributeGroup>
  <xs:complexType name="Base" defaultAttributesApply="false"/>
  <xs:complexType name="ComplexExt">
    <xs:complexContent>
      <xs:extension base="Base">
        <xs:attributeGroup ref="explicit"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="SimpleExt">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attributeGroup ref="explicit"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType defaultAttributesApply="false">
      <xs:sequence>
        <xs:element name="direct">
          <xs:complexType>
            <xs:attributeGroup ref="explicit"/>
          </xs:complexType>
        </xs:element>
        <xs:element name="complexExt" type="ComplexExt"/>
        <xs:element name="simpleExt" type="SimpleExt"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	schema, err := compileDefaultAttributesSchema(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)

	idoc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><direct/><complexExt/><simpleExt>x</simpleExt></root>`))
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), idoc))
	out, err := helium.WriteString(idoc)
	require.NoError(t, err)

	requireAttrOrder(t, out, "direct", "explicit", "schemaDefault")
	requireAttrOrder(t, out, "complexExt", "explicit", "schemaDefault")
	requireAttrOrder(t, out, "simpleExt", "explicit", "schemaDefault")
}

func compileDefaultAttributesSchema(t *testing.T, c xsd.Compiler, schemaXML string) (*xsd.Schema, error) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	return c.Compile(t.Context(), doc)
}

func requireAttrOrder(t *testing.T, out, elem, before, after string) {
	t.Helper()
	start := strings.Index(out, "<"+elem)
	require.NotEqual(t, -1, start, "element %s not found in %q", elem, out)
	tagEnd := strings.Index(out[start:], ">")
	require.NotEqual(t, -1, tagEnd, "element %s start tag not closed in %q", elem, out)
	tag := out[start : start+tagEnd]
	beforeIndex := strings.Index(tag, before+`=`)
	afterIndex := strings.Index(tag, after+`=`)
	require.NotEqual(t, -1, beforeIndex, "attribute %s not found in %q", before, tag)
	require.NotEqual(t, -1, afterIndex, "attribute %s not found in %q", after, tag)
	require.Less(t, beforeIndex, afterIndex, "attribute %s should appear before %s in %q", before, after, tag)
}
