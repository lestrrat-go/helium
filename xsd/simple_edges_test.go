package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileV11 compiles schemaXML under XSD 1.1 and returns the resolved schema
// (nil on failure), the collected diagnostic text, and the compile error.
func compileV11(t *testing.T, schemaXML string) (*xsd.Schema, string, error) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, cerr := xsd.NewCompiler().
		Version(xsd.Version11).
		Label("test.xsd").
		ErrorHandler(collector).
		Compile(t.Context(), doc)
	_ = collector.Close()
	return schema, compileErrorsString(collector.Errors()), cerr
}

// compileV10 compiles schemaXML under the default XSD 1.0 semantics.
func compileV10(t *testing.T, schemaXML string) (*xsd.Schema, error) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	return xsd.NewCompiler().Label("test.xsd").Compile(t.Context(), doc)
}

// simple005: finalDefault="extension" applies to a simpleType in XSD 1.1, so a
// simpleContent extension of that type is forbidden. In XSD 1.0 the extension
// bit does not apply to simple types, so the same schema is valid.
func TestSimpleEdge_FinalExtensionOnSimpleType(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:s" xmlns:s="urn:s" finalDefault="extension">
  <xs:simpleType name="pubDate">
    <xs:restriction base="xs:date">
      <xs:pattern value="2012.*"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:complexType name="pubType">
    <xs:simpleContent>
      <xs:extension base="s:pubDate">
        <xs:attribute name="country" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`

	_, errs, cerr := compileV11(t, schemaXML)
	require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "1.1: extension-final simpleType must block simpleContent extension")
	require.Contains(t, errs, "extension is forbidden")

	// XSD 1.0: finalDefault extension does not reach a simple type, so valid.
	schema10, cerr10 := compileV10(t, schemaXML)
	require.NoError(t, cerr10, "1.0: extension bit must not apply to simpleType")
	require.NotNil(t, schema10)
}

// simple010/012/013: a complexContent restriction may narrow a child element's
// type to a type validly derived from a member of the base element's union type
// (transitively through nested unions).
func TestSimpleEdge_UnionMemberSubstitutabilityInRestriction(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		// simple010: chap = union(date,dateTime,time); sub-chap restricts date.
		"direct member": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:s" xmlns:s="urn:s">
  <xs:complexType name="doc-type">
    <xs:sequence maxOccurs="unbounded">
      <xs:element name="chap" type="s:chap"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="sub-doc-type">
    <xs:complexContent>
      <xs:restriction base="s:doc-type">
        <xs:sequence maxOccurs="unbounded">
          <xs:element name="chap" type="s:sub-chap"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:simpleType name="chap"><xs:union memberTypes="xs:date xs:dateTime xs:time"/></xs:simpleType>
  <xs:simpleType name="sub-chap"><xs:restriction base="xs:date"/></xs:simpleType>
</xs:schema>`,
		// simple013: chap = union(dt,time), dt = union(date,dateTime); sub-chap
		// restricts date — a member of a nested union member.
		"nested union member": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:s" xmlns:s="urn:s">
  <xs:complexType name="doc-type">
    <xs:sequence maxOccurs="unbounded">
      <xs:element name="chap" type="s:chap"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="sub-doc-type">
    <xs:complexContent>
      <xs:restriction base="s:doc-type">
        <xs:sequence maxOccurs="unbounded">
          <xs:element name="chap" type="s:sub-chap"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:simpleType name="chap"><xs:union memberTypes="s:dt xs:time"/></xs:simpleType>
  <xs:simpleType name="dt"><xs:union memberTypes="xs:date xs:dateTime"/></xs:simpleType>
  <xs:simpleType name="sub-chap"><xs:restriction base="xs:date"/></xs:simpleType>
</xs:schema>`,
	}

	for name, schemaXML := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, errs, cerr := compileV11(t, schemaXML)
			require.NoError(t, cerr, "schema should compile: %s", errs)
		})
	}
}

// simple051/052/053: xs:anyAtomicType must not be used as the base type of a
// user-defined simple type, nor as a list item type, nor as a union member type.
func TestSimpleEdge_AnyAtomicTypeUsageRejected(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"restriction base": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:simpleType><xs:restriction base="xs:anyAtomicType"/></xs:simpleType>
  </xs:element>
</xs:schema>`,
		"list item type": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:simpleType><xs:list itemType="xs:anyAtomicType"/></xs:simpleType>
  </xs:element>
</xs:schema>`,
		"union member type": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:simpleType><xs:union memberTypes="xs:anyAtomicType xs:string"/></xs:simpleType>
  </xs:element>
</xs:schema>`,
	}

	for name, schemaXML := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, _, cerr := compileV11(t, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed)
		})
	}
}

// simple050: xs:anyAtomicType is a legitimate type for an element declaration.
func TestSimpleEdge_AnyAtomicTypeAsElementType(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:anyAtomicType"/>
</xs:schema>`
	schema, errs, cerr := compileV11(t, schemaXML)
	require.NoError(t, cerr, "anyAtomicType is a valid element type: %s", errs)
	require.NotNil(t, schema)
}

// simple095: an xs:NOTATION restriction's enumeration values must name notations
// declared in the schema. "png" is not declared, so the schema is invalid.
func TestSimpleEdge_NotationEnumerationMustBeDeclared(t *testing.T) {
	t.Parallel()
	const badXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
  <xs:simpleType name="restrictedNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="png"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:attribute name="a" type="restrictedNotation"/>
</xs:schema>`
	_, _, cerr := compileV11(t, badXML)
	require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "undeclared notation enumeration must fail")

	const goodXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
  <xs:simpleType name="restrictedNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="jpeg"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:attribute name="a" type="restrictedNotation"/>
</xs:schema>`
	schema, errs, cerr := compileV11(t, goodXML)
	require.NoError(t, cerr, "declared notation enumeration must compile: %s", errs)
	require.NotNil(t, schema)
}
