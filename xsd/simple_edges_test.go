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

// TestSimpleEdge_NotationEnumInListUnionCarrier verifies the §3.14.6
// "enumeration value must name a declared notation" rule reaches a NOTATION
// carrier used as a list itemType or a union memberType, not only a direct
// atomic xs:NOTATION restriction. A bare built-in xs:NOTATION list/union carrier
// is permitted in XSD 1.0 (particlesZ007), so the enumeration-value check is what
// rejects an undeclared token there; the same carrier is separately rejected as
// non-enumeration-derived in XSD 1.1.
func TestSimpleEdge_NotationEnumInListUnionCarrier(t *testing.T) {
	t.Parallel()

	// listSchema builds a schema whose enumeration constrains a list of
	// bare xs:NOTATION to the single-token value tok.
	listSchema := func(tok string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
  <xs:simpleType name="notationList">
    <xs:list itemType="xs:NOTATION"/>
  </xs:simpleType>
  <xs:simpleType name="restrictedList">
    <xs:restriction base="notationList">
      <xs:enumeration value="` + tok + `"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:attribute name="a" type="restrictedList"/>
</xs:schema>`
	}

	// unionSchema builds a schema whose enumeration constrains a union whose first
	// member is bare xs:NOTATION to the value tok.
	unionSchema := func(tok string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
  <xs:simpleType name="notationUnion">
    <xs:union memberTypes="xs:NOTATION"/>
  </xs:simpleType>
  <xs:simpleType name="restrictedUnion">
    <xs:restriction base="notationUnion">
      <xs:enumeration value="` + tok + `"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:attribute name="a" type="restrictedUnion"/>
</xs:schema>`
	}

	// XSD 1.0: bare-NOTATION list/union carrier is a permitted use, so the only
	// thing rejecting an undeclared token is the enumeration-value check.
	_, cerr := compileV10(t, listSchema("png"))
	require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "1.0 list: undeclared notation token must fail")

	schema, cerr := compileV10(t, listSchema("jpeg"))
	require.NoError(t, cerr, "1.0 list: declared notation token must compile")
	require.NotNil(t, schema)

	_, cerr = compileV10(t, unionSchema("png"))
	require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "1.0 union: undeclared notation value must fail")

	schema, cerr = compileV10(t, unionSchema("jpeg"))
	require.NoError(t, cerr, "1.0 union: declared notation value must compile")
	require.NotNil(t, schema)

	// XSD 1.1: an undeclared token in either carrier is still rejected (the
	// bare carrier is additionally non-enumeration-derived here).
	_, _, cerr = compileV11(t, listSchema("png"))
	require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "1.1 list: undeclared notation token must fail")

	_, _, cerr = compileV11(t, unionSchema("png"))
	require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "1.1 union: undeclared notation value must fail")
}
