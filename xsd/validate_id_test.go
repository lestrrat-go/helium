package xsd_test

import (
	"fmt"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestValidateIDIDREF covers document-wide xs:ID / xs:IDREF / xs:IDREFS
// validation (cvc-id, version-independent): ID values must be unique across the
// document and every IDREF token must resolve to some ID. XSD 1.1 relaxes
// uniqueness so the same value may identify a single element more than once
// (multiple ID attributes of one element, or multiple ID children of one parent);
// XSD 1.0 has no such relaxation — any repeat is a duplicate.
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

	t.Run("XSD 1.0 enforces ID uniqueness across elements", func(t *testing.T) {
		t.Parallel()
		// cvc-id is version-independent: a duplicate ID value across two different
		// elements is invalid in 1.0 too. Each <a> carries a single xs:ID attribute,
		// legal in 1.0.
		inst := `<root><a id="x"/><a id="x"/></root>`
		require.Error(t, compileValidate(t, xsd.Version10, idrefSchema, inst))
	})

	t.Run("XSD 1.0 resolves IDREF referential integrity", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileValidate(t, xsd.Version10, idrefSchema, `<root><a id="x"/><a ref="x"/></root>`))
		require.Error(t, compileValidate(t, xsd.Version10, idrefSchema, `<root><a id="x"/><a ref="y"/></root>`))
	})

	t.Run("XSD 1.0 rejects same-owner same-value ID recurrence", func(t *testing.T) {
		t.Parallel()
		// Two <id> element-content children of one <node> share a value (both
		// identify the same parent element). XSD 1.1 accepts this (the multiple-ID
		// relaxation); XSD 1.0 has no such relaxation, so it is a duplicate (W3C
		// elemZ016 / idconstrdefs00301m2_n).
		inst := `<root><node><id>zzz</id><id>zzz</id></node></root>`
		require.NoError(t, compileValidate(t, xsd.Version11, elemIDSchema, inst))
		require.Error(t, compileValidate(t, xsd.Version10, elemIDSchema, inst))
	})

	t.Run("XSD 1.0 rejects same value across two distinct elements", func(t *testing.T) {
		t.Parallel()
		// Distinct owner elements (two separate <node>s) with the same value are a
		// uniqueness violation in BOTH versions.
		inst := `<root><node><id>zzz</id></node><node><id>zzz</id></node></root>`
		require.Error(t, compileValidate(t, xsd.Version10, elemIDSchema, inst))
		require.Error(t, compileValidate(t, xsd.Version11, elemIDSchema, inst))
	})

	// A complex type admitting two global xs:ID attributes on one element via
	// anyAttribute. XSD 1.0 caps an element at one ID-typed attribute; XSD 1.1
	// removed the cap.
	const twoIDAttrSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:anyAttribute processContents="strict"/>
  </xs:complexType>
  <xs:element name="doc" type="base"/>
  <xs:attribute name="a" type="xs:ID"/>
  <xs:attribute name="b" type="xs:ID"/>
</xs:schema>`

	t.Run("XSD 1.0 caps one ID-typed attribute per element", func(t *testing.T) {
		t.Parallel()
		// Two DISTINCT-valued ID attributes on one element: rejected in 1.0 (the
		// one-ID-per-element cardinality rule, W3C attZ014a/attZ014b), accepted in
		// 1.1. Values differ, so this is the cardinality rule, not value-uniqueness.
		inst := `<doc a="x" b="y"/>`
		require.Error(t, compileValidate(t, xsd.Version10, twoIDAttrSchema, inst))
		require.NoError(t, compileValidate(t, xsd.Version11, twoIDAttrSchema, inst))
	})

	// Two attributes of type union(xs:int, xs:ID). The union is not itself
	// ID-typed, so the cap must detect ID-ness VALUE-dependently, the same way the
	// uniqueness collection does — an attribute counts only when its value selects
	// the xs:ID member.
	const unionIDAttrSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrID">
    <xs:union memberTypes="xs:int xs:ID"/>
  </xs:simpleType>
  <xs:complexType name="base">
    <xs:attribute name="u1" type="intOrID"/>
    <xs:attribute name="u2" type="intOrID"/>
  </xs:complexType>
  <xs:element name="doc" type="base"/>
</xs:schema>`

	t.Run("XSD 1.0 caps two union(int,ID) attributes both carrying IDs", func(t *testing.T) {
		t.Parallel()
		// "aaa"/"bbb" are not valid xs:int, so each resolves to the xs:ID member —
		// two ID-bearing attributes → rejected in 1.0, accepted in 1.1.
		inst := `<doc u1="aaa" u2="bbb"/>`
		require.Error(t, compileValidate(t, xsd.Version10, unionIDAttrSchema, inst))
		require.NoError(t, compileValidate(t, xsd.Version11, unionIDAttrSchema, inst))
	})

	t.Run("XSD 1.0 union int value does not count toward the ID cap", func(t *testing.T) {
		t.Parallel()
		// u1="5" resolves to xs:int (no ID leaf, doesn't count); only u2="aaa" is
		// ID-bearing → one ID attribute → valid in both versions.
		inst := `<doc u1="5" u2="aaa"/>`
		require.NoError(t, compileValidate(t, xsd.Version10, unionIDAttrSchema, inst))
		require.NoError(t, compileValidate(t, xsd.Version11, unionIDAttrSchema, inst))
	})

	// Two attributes of type list-of-xs:ID. A list is not itself ID-typed either,
	// so the cap must count via the same decomposition (a list contributes ID
	// leaves for its tokens).
	const listIDAttrSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="idList">
    <xs:list itemType="xs:ID"/>
  </xs:simpleType>
  <xs:complexType name="base">
    <xs:attribute name="l1" type="idList"/>
    <xs:attribute name="l2" type="idList"/>
  </xs:complexType>
  <xs:element name="doc" type="base"/>
</xs:schema>`

	t.Run("XSD 1.0 caps two list-of-ID attributes", func(t *testing.T) {
		t.Parallel()
		// Each list contributes xs:ID leaves, so both attributes are ID-bearing →
		// rejected in 1.0, accepted in 1.1. Tokens are all distinct (no uniqueness
		// violation), isolating the cardinality cap.
		inst := `<doc l1="a b" l2="c d"/>`
		require.Error(t, compileValidate(t, xsd.Version10, listIDAttrSchema, inst))
		require.NoError(t, compileValidate(t, xsd.Version11, listIDAttrSchema, inst))
	})
}

// TestIDConstraintRef covers XSD 1.1 identity-constraint @ref: a key/unique/
// keyref may reference another constraint of the SAME kind, reusing its
// selector/fields (and, for keyref, its refer). A reference to a missing
// constraint, or to a constraint of a different kind, is a schema error.
func TestIDConstraintRef(t *testing.T) {
	compile := func(t *testing.T, schemaXML string) (*xsd.Schema, error) {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
	}
	validate := func(t *testing.T, schema *xsd.Schema, instanceXML string) error {
		t.Helper()
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	const uniqueRefSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:element name="chap" type="chap">
          <xs:unique name="u"><xs:selector xpath="section"/><xs:field xpath="@nr"/></xs:unique>
        </xs:element>
        <xs:element name="appx" type="chap">
          <xs:unique ref="u"/>
        </xs:element>
      </xs:choice>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="chap">
    <xs:sequence maxOccurs="unbounded">
      <xs:element name="section">
        <xs:complexType><xs:attribute name="nr" type="xs:int"/></xs:complexType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	t.Run("unique ref applies the referenced constraint at the new host", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, uniqueRefSchema)
		require.NoError(t, err)
		// Duplicate @nr inside appx (which uses the ref'd unique) is invalid.
		bad := `<doc><appx><section nr="1"/><section nr="1"/></appx></doc>`
		require.Error(t, validate(t, schema, bad))
		ok := `<doc><appx><section nr="1"/><section nr="2"/></appx></doc>`
		require.NoError(t, validate(t, schema, ok))
	})

	t.Run("ref to a nonexistent constraint is a schema error", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
    <xs:unique ref="missing"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, schemaXML)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("ref to a different kind of constraint is a schema error", func(t *testing.T) {
		t.Parallel()
		// A key referencing a unique is a kind mismatch.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:element name="chap" type="chap">
          <xs:unique name="u"><xs:selector xpath="section"/><xs:field xpath="@nr"/></xs:unique>
        </xs:element>
        <xs:element name="appx" type="chap">
          <xs:key ref="u"/>
        </xs:element>
      </xs:choice>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="chap">
    <xs:sequence maxOccurs="unbounded">
      <xs:element name="section">
        <xs:complexType><xs:attribute name="nr" type="xs:int"/></xs:complexType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`
		_, err := compile(t, schemaXML)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}

// TestIDCXPathDefaultNamespace verifies that @xpathDefaultNamespace on an
// identity-constraint selector resolves unprefixed element name tests against
// the given namespace, so a uniqueness violation in a namespaced document is
// detected (it would be missed if the unprefixed name matched no-namespace).
func TestIDCXPathDefaultNamespace(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:x" xmlns:s="urn:x" elementFormDefault="qualified">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence><xs:element name="emp" type="s:emp" maxOccurs="unbounded"/></xs:sequence>
    </xs:complexType>
    <xs:unique name="u">
      <xs:selector xpath="emp" xpathDefaultNamespace="urn:x"/>
      <xs:field xpath="nr" xpathDefaultNamespace="urn:x"/>
    </xs:unique>
  </xs:element>
  <xs:complexType name="emp">
    <xs:sequence><xs:element name="nr" type="xs:int"/></xs:sequence>
  </xs:complexType>
</xs:schema>`
	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
	require.NoError(t, err)

	dup := `<doc xmlns="urn:x"><emp><nr>1</nr></emp><emp><nr>1</nr></emp></doc>`
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(dup))
	require.NoError(t, err)
	require.Error(t, xsd.NewValidator(schema).Validate(t.Context(), idoc))

	ok := `<doc xmlns="urn:x"><emp><nr>1</nr></emp><emp><nr>2</nr></emp></doc>`
	idoc2, err := helium.NewParser().Parse(t.Context(), []byte(ok))
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), idoc2))
}

// TestIDCXPathDefaultNamespaceEmptyOverride covers PR860-IDC-006: an explicit
// xpathDefaultNamespace="" on a selector/field is a real value (xs:anyURI admits
// the empty string) meaning "no default element namespace", and must NOT inherit
// the schema-level default. Here the root sets ##targetNamespace but the selector/
// field override it to empty, so unprefixed name tests must match no-namespace
// elements (the unqualified local emp/nr), catching the duplicate.
func TestIDCXPathDefaultNamespaceEmptyOverride(t *testing.T) {
	t.Parallel()
	// elementFormDefault defaults to unqualified, so local emp/nr are no-namespace
	// while the global doc is in urn:x.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:x" xmlns:s="urn:x" xpathDefaultNamespace="##targetNamespace">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence><xs:element name="emp" type="empType" maxOccurs="unbounded"/></xs:sequence>
    </xs:complexType>
    <xs:unique name="u">
      <xs:selector xpath="emp" xpathDefaultNamespace=""/>
      <xs:field xpath="nr" xpathDefaultNamespace=""/>
    </xs:unique>
  </xs:element>
  <xs:complexType name="empType">
    <xs:sequence><xs:element name="nr" type="xs:int"/></xs:sequence>
  </xs:complexType>
</xs:schema>`
	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
	require.NoError(t, err)

	// doc is {urn:x}doc; the local emp/nr are unqualified (no namespace). With the
	// empty selector/field default the unprefixed "emp"/"nr" match these
	// no-namespace nodes, so the duplicate nr is caught. (Inheriting ##targetNamespace
	// would resolve "emp" to {urn:x}emp, match nothing, and miss the duplicate.)
	dup := `<s:doc xmlns:s="urn:x"><emp><nr>1</nr></emp><emp><nr>1</nr></emp></s:doc>`
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(dup))
	require.NoError(t, err)
	require.Error(t, xsd.NewValidator(schema).Validate(t.Context(), idoc),
		"explicit xpathDefaultNamespace=\"\" must not inherit the schema-level default")

	ok := `<s:doc xmlns:s="urn:x"><emp><nr>1</nr></emp><emp><nr>2</nr></emp></s:doc>`
	idoc2, err := helium.NewParser().Parse(t.Context(), []byte(ok))
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), idoc2))
}

// TestIDCXPathDefaultNamespaceInheritedDefaultNS covers PR860-REVIEW-NS-001: an
// inherited schema-level xpathDefaultNamespace="##defaultNamespace" must resolve
// against the SCHEMA ROOT's default namespace, NOT against a selector/field that
// redeclares xmlns. Here the root default ns is urn:A but the selector/field
// redeclare xmlns="urn:B"; the inherited default must still be urn:A so the
// duplicate in the urn:A instance is caught.
func TestIDCXPathDefaultNamespaceInheritedDefaultNS(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns="urn:A" targetNamespace="urn:A" elementFormDefault="qualified"
    xpathDefaultNamespace="##defaultNamespace">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence><xs:element name="emp" type="empType" maxOccurs="unbounded"/></xs:sequence>
    </xs:complexType>
    <xs:unique name="u">
      <xs:selector xmlns="urn:B" xpath="emp"/>
      <xs:field xmlns="urn:B" xpath="nr"/>
    </xs:unique>
  </xs:element>
  <xs:complexType name="empType">
    <xs:sequence><xs:element name="nr" type="xs:int"/></xs:sequence>
  </xs:complexType>
</xs:schema>`
	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
	require.NoError(t, err)

	// emp/nr are {urn:A} (qualified). The inherited ##defaultNamespace must resolve
	// to the ROOT default ns urn:A (not the selector's redeclared urn:B), so "emp"
	// matches {urn:A}emp and the duplicate nr=1 is caught.
	dup := `<doc xmlns="urn:A"><emp><nr>1</nr></emp><emp><nr>1</nr></emp></doc>`
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(dup))
	require.NoError(t, err)
	require.Error(t, xsd.NewValidator(schema).Validate(t.Context(), idoc),
		"inherited ##defaultNamespace must resolve against the schema root, not the selector's redeclared xmlns")

	ok := `<doc xmlns="urn:A"><emp><nr>1</nr></emp><emp><nr>2</nr></emp></doc>`
	idoc2, err := helium.NewParser().Parse(t.Context(), []byte(ok))
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), idoc2))
}

// TestIDSkipWildcardNotAssessed verifies that elements/attributes admitted
// through a processContents="skip" wildcard are NOT treated as xs:ID/xs:IDREF by
// the document-wide ID pass: skip content is not schema-assessed, so duplicate
// "ID" values there must not be flagged even when a global declaration of the
// same name would otherwise classify them as xs:ID.
func TestIDSkipWildcardNotAssessed(t *testing.T) {
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

	t.Run("skip-wildcard attributes are not global-classified as ID", func(t *testing.T) {
		t.Parallel()
		// A GLOBAL attribute named "id" of type xs:ID exists, so a naive global
		// fallback would type the skip-content @id attributes as xs:ID and reject
		// the duplicate. Under skip they are unassessed, so this is valid.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="id" type="xs:ID"/>
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:any processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		inst := `<doc><a id="dup"/><b id="dup"/></doc>`
		require.NoError(t, compileValidate(t, schemaXML, inst))
	})

	t.Run("skip-wildcard element content is not global-classified as ID", func(t *testing.T) {
		t.Parallel()
		// A GLOBAL element "n" of type xs:ID exists; under skip the <n> elements are
		// unassessed, so duplicate text content must not be flagged.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="n" type="xs:ID"/>
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:any processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		inst := `<doc><wrap><n>dup</n></wrap><wrap><n>dup</n></wrap></doc>`
		require.NoError(t, compileValidate(t, schemaXML, inst))
	})

	// A skip subtree is annotated (actualElemType) for pass-2 IDC canonicalization
	// when it carries xsi:type, but that annotation must NOT leak into the
	// document-wide ID/IDREF pass: skip content is not schema-assessed.
	const skipAnySchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:any processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("skip-wildcard xsi:type=xs:ID duplicates are not flagged", func(t *testing.T) {
		t.Parallel()
		inst := `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<wrap><id xsi:type="xs:ID">dup</id></wrap>` +
			`<wrap><id xsi:type="xs:ID">dup</id></wrap></doc>`
		require.NoError(t, compileValidate(t, skipAnySchema, inst))
	})

	t.Run("skip-wildcard xsi:type=xs:IDREF dangling ref is not flagged", func(t *testing.T) {
		t.Parallel()
		inst := `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<wrap><r xsi:type="xs:IDREF">nope</r></wrap></doc>`
		require.NoError(t, compileValidate(t, skipAnySchema, inst))
	})
}

// TestIDConstraintRefUnboundPrefix verifies that an identity-constraint @ref
// using a namespace prefix that is not bound in scope is a fatal schema error
// rather than silently resolving to the no-namespace constraint set.
func TestIDConstraintRefUnboundPrefix(t *testing.T) {
	t.Parallel()
	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:element name="chap" type="chap">
          <xs:unique name="u"><xs:selector xpath="section"/><xs:field xpath="@nr"/></xs:unique>
        </xs:element>
        <xs:element name="appx" type="chap">
          <xs:unique ref="bad:u"/>
        </xs:element>
      </xs:choice>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="chap">
    <xs:sequence maxOccurs="unbounded">
      <xs:element name="section">
        <xs:complexType><xs:attribute name="nr" type="xs:int"/></xs:complexType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`
	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
	require.ErrorIs(t, err, xsd.ErrCompilationFailed)
}

// TestIDConstraintRefConflictingChildren verifies that an identity-constraint
// using @ref must not also carry name / selector / field / (keyref) refer — the
// ref form is mutually exclusive with the full form.
func TestIDConstraintRefConflictingChildren(t *testing.T) {
	compile := func(t *testing.T, appxIDC string) error {
		t.Helper()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:element name="chap" type="chap">
          <xs:unique name="u"><xs:selector xpath="section"/><xs:field xpath="@nr"/></xs:unique>
        </xs:element>
        <xs:element name="appx" type="chap">
          ` + appxIDC + `
        </xs:element>
      </xs:choice>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="chap">
    <xs:sequence maxOccurs="unbounded">
      <xs:element name="section">
        <xs:complexType><xs:attribute name="nr" type="xs:int"/></xs:complexType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
		return err
	}

	t.Run("ref with name is rejected", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compile(t, `<xs:unique ref="u" name="dup"/>`), xsd.ErrCompilationFailed)
	})
	t.Run("ref with empty-but-present name is rejected", func(t *testing.T) {
		t.Parallel()
		// PR860-IDC-004: a present name="" must be rejected like any name companion;
		// detection is by attribute PRESENCE, not value.
		require.ErrorIs(t, compile(t, `<xs:unique ref="u" name=""/>`), xsd.ErrCompilationFailed)
	})
	t.Run("ref with selector is rejected", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compile(t, `<xs:unique ref="u"><xs:selector xpath="section"/></xs:unique>`), xsd.ErrCompilationFailed)
	})
	t.Run("ref with field is rejected", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compile(t, `<xs:unique ref="u"><xs:field xpath="@nr"/></xs:unique>`), xsd.ErrCompilationFailed)
	})
	t.Run("ref with malformed QName is rejected", func(t *testing.T) {
		t.Parallel()
		// ":u" is not a valid xs:QName (empty prefix); it must be a fatal error, not
		// silently resolved as an unprefixed/default-namespace reference.
		require.ErrorIs(t, compile(t, `<xs:unique ref=":u"/>`), xsd.ErrCompilationFailed)
	})
	t.Run("keyref refer with malformed QName is rejected", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compile(t, `<xs:keyref name="kr" refer=":k"><xs:selector xpath="section"/><xs:field xpath="@nr"/></xs:keyref>`), xsd.ErrCompilationFailed)
	})
	t.Run("plain ref is accepted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compile(t, `<xs:unique ref="u"/>`))
	})
}

// TestIDConstraintRefFormCompanionSymmetry verifies that a present-but-empty
// @name / @refer companion on the XSD 1.1 identity-constraint @ref form is
// treated like an invalid NCName/QName VALUE — emitting its one value diagnostic
// and NOT the structural ref-conflict — with the literal "" and a whitespace-only
// value ("   ") handled identically (xs:NCName / xs:QName both fix whiteSpace
// "collapse"). A genuinely-present VALID companion still fires the ref-conflict.
func TestIDConstraintRefFormCompanionSymmetry(t *testing.T) {
	t.Parallel()

	// The @ref names an existing same-kind unique so the ref itself resolves
	// cleanly (no dangling-ref noise); appxIDC is the ref-form constraint under test.
	compile := func(t *testing.T, appxIDC string) (string, error) {
		t.Helper()
		schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:element name="chap" type="chap">
          <xs:unique name="u"><xs:selector xpath="section"/><xs:field xpath="@nr"/></xs:unique>
        </xs:element>
        <xs:element name="appx" type="chap">
          %s
        </xs:element>
      </xs:choice>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="chap">
    <xs:sequence maxOccurs="unbounded">
      <xs:element name="section">
        <xs:complexType><xs:attribute name="nr" type="xs:int"/></xs:complexType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`, appxIDC)
		_, errs, cerr := compileWith(t, xsd.Version11, schemaXML)
		return errs, cerr
	}

	const wantNCName = "is not a valid 'xs:NCName'"
	const wantQName = "is not a valid QName"
	const wantConflict = "must not also specify"

	// A collapse-empty companion emits its value diagnostic, never the ref-conflict.
	valueCases := []struct {
		name      string
		idcFmt    string
		wantValue string
		wantOther string
	}{
		{"name", `<xs:unique ref="u" name="%s"/>`, wantNCName, wantQName},
		{"refer", `<xs:unique ref="u" refer="%s"/>`, wantQName, wantNCName},
	}
	for _, tc := range valueCases {
		for _, val := range []struct{ label, v string }{{"literal-empty", ""}, {"whitespace-only", "   "}} {
			t.Run("collapse-empty/"+tc.name+"/"+val.label, func(t *testing.T) {
				t.Parallel()
				errs, cerr := compile(t, fmt.Sprintf(tc.idcFmt, val.v))
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "errs: %s", errs)
				require.Contains(t, errs, tc.wantValue,
					"present-empty @%s must emit its invalid-value diagnostic; got: %s", tc.name, errs)
				require.NotContains(t, errs, wantConflict,
					"present-empty @%s must not fire the structural ref-conflict; got: %s", tc.name, errs)
				require.NotContains(t, errs, tc.wantOther,
					"present-empty @%s must not leak the other value diagnostic; got: %s", tc.name, errs)
			})
		}
	}

	// A genuinely-present VALID companion still fires the ref-conflict.
	validCases := []struct {
		name   string
		idc    string
		compan string
	}{
		{"name", `<xs:unique ref="u" name="dup"/>`, "name"},
		{"refer", `<xs:unique ref="u" refer="u"/>`, "refer"},
	}
	for _, tc := range validCases {
		t.Run("valid-companion-still-conflicts/"+tc.name, func(t *testing.T) {
			t.Parallel()
			errs, cerr := compile(t, tc.idc)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "errs: %s", errs)
			require.Contains(t, errs, "must not also specify '"+tc.compan+"'",
				"valid @%s companion must still fire the ref-conflict; got: %s", tc.compan, errs)
			require.Equal(t, 0, strings.Count(errs, wantNCName)+strings.Count(errs, wantQName),
				"a valid companion is not an invalid NCName/QName value; got: %s", errs)
		})
	}
}

// TestIDConstraintRefValidPrefixed verifies that a valid PREFIXED @ref still
// resolves after the malformed-QName check is added (the check must not reject
// well-formed prefixed references).
func TestIDConstraintRefValidPrefixed(t *testing.T) {
	t.Parallel()
	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="urn:x" targetNamespace="urn:x">
  <xs:element name="doc">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:element name="chap" type="s:chap">
          <xs:unique name="u"><xs:selector xpath="s:section"/><xs:field xpath="@nr"/></xs:unique>
        </xs:element>
        <xs:element name="appx" type="s:chap">
          <xs:unique ref="s:u"/>
        </xs:element>
      </xs:choice>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="chap">
    <xs:sequence maxOccurs="unbounded">
      <xs:element name="section">
        <xs:complexType><xs:attribute name="nr" type="xs:int"/></xs:complexType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`
	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
	require.NoError(t, err, "a valid prefixed @ref must resolve")
	require.NotNil(t, schema)

	// The ref'd unique applies at appx: a duplicate @nr inside appx is invalid.
	bad := `<doc xmlns="urn:x"><appx><section nr="1"/><section nr="1"/></appx></doc>`
	bdoc, err := helium.NewParser().Parse(t.Context(), []byte(bad))
	require.NoError(t, err)
	require.Error(t, xsd.NewValidator(schema).Validate(t.Context(), bdoc))
}

// TestIDLaxWildcardXsiTypeAssessed covers PR860-XSD-ID-001: an element admitted
// through a processContents="lax" wildcard that has no declaration but whose
// xsi:type resolves to a governing type IS schema-assessed (XSD lax) and so
// participates in the document-wide ID/IDREF pass. A processContents="skip"
// wildcard, by contrast, never assesses its content (preserving the earlier fix).
func TestIDLaxWildcardXsiTypeAssessed(t *testing.T) {
	compileValidate := func(t *testing.T, processContents, instanceXML string) error {
		t.Helper()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:any processContents="` + processContents + `" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	// Two xsi:type="xs:ID" elements under DIFFERENT parents (different ID owners)
	// share the value "dup": a genuine duplicate. The xs prefix must be bound in
	// the instance for xsi:type="xs:ID" to resolve.
	const dupDifferentOwners = `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
		`<w1><id xsi:type="xs:ID">dup</id></w1>` +
		`<w2><id xsi:type="xs:ID">dup</id></w2></doc>`

	t.Run("lax wildcard xsi:type=xs:ID duplicate across owners is rejected", func(t *testing.T) {
		t.Parallel()
		require.Error(t, compileValidate(t, "lax", dupDifferentOwners),
			"lax-assessed xsi:type=xs:ID content must participate in ID uniqueness")
	})

	t.Run("skip wildcard xsi:type=xs:ID duplicate is NOT rejected", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileValidate(t, "skip", dupDifferentOwners),
			"skip content is never assessed, so its xsi:type=xs:ID must not be checked")
	})

	t.Run("lax wildcard xsi:type=xs:ID under one owner is valid", func(t *testing.T) {
		t.Parallel()
		// Two xs:ID elements with the same value under the SAME parent identify the
		// same element (parent owner), so this is not a duplicate.
		sameOwner := `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<w><id xsi:type="xs:ID">dup</id><id xsi:type="xs:ID">dup</id></w></doc>`
		require.NoError(t, compileValidate(t, "lax", sameOwner))
	})

	// PR860-XSD-ID-005: a lax no-declaration element is not validly nillable (no
	// nillable declaration), so xsi:nil="true" must NOT exempt its real content
	// from the ID/IDREF pass — raw xsi:nil must not drop it.
	t.Run("lax wildcard xsi:nil=true xs:ID with valid content still collides", func(t *testing.T) {
		t.Parallel()
		inst := `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<w1><id xsi:type="xs:ID" xsi:nil="true">dup</id></w1>` +
			`<w2><id xsi:type="xs:ID" xsi:nil="true">dup</id></w2></doc>`
		require.Error(t, compileValidate(t, "lax", inst),
			"a lax element with xsi:nil but a resolvable xsi:type is not nilled; its ID content must still be checked")
	})

	t.Run("lax wildcard xsi:nil=true xs:IDREF with valid content still dangles", func(t *testing.T) {
		t.Parallel()
		inst := `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<r xsi:type="xs:IDREF" xsi:nil="true">nomatch</r></doc>`
		require.Error(t, compileValidate(t, "lax", inst),
			"a lax element with xsi:nil but a resolvable xsi:type is not nilled; its IDREF must still resolve")
	})
}

// TestIDNilledElementDefaultNotCollected covers PR860-XSD-ID-002: a nilled
// element (xsi:nil="true") has no element value, so its declared default/fixed
// must NOT be substituted into the ID/IDREF pass. Otherwise the fabricated value
// would false-reject a valid document as a duplicate ID (or a dangling IDREF).
func TestIDNilledElementDefaultNotCollected(t *testing.T) {
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

	t.Run("nilled ID element default does not duplicate a sibling's ID", func(t *testing.T) {
		t.Parallel()
		// Each <grp> bears one <id> (an element-content ID owned by its <grp>). The
		// first grp's <id> is nilled; without skipping it, its default "p1" would be
		// collected under grp1 and collide with grp2's legitimate "p1" under grp2.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="grp" maxOccurs="unbounded">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="id" type="xs:ID" nillable="true" default="p1"/>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		inst := `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
			`<grp><id xsi:nil="true"/></grp>` +
			`<grp><id>p1</id></grp></doc>`
		require.NoError(t, compileValidate(t, schemaXML, inst),
			"a nilled element's default must not be collected as an ID")
	})

	t.Run("nilled IDREF element default does not dangle", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="r" type="xs:IDREF" nillable="true" default="nomatch"/>
        <xs:element name="x" type="xs:ID"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		inst := `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
			`<r xsi:nil="true"/><x>realid</x></doc>`
		require.NoError(t, compileValidate(t, schemaXML, inst),
			"a nilled IDREF element's default must not be collected as a reference")
	})
}

// TestIDLaxWildcardNilledStillValidated covers PR860-REVIEW-001: an undeclared
// processContents="lax" element with a resolvable xsi:type and xsi:nil="true"
// must STILL be validated against the governing type — an undeclared element has
// no nillable declaration, so xsi:nil cannot exempt it from type validation.
func TestIDLaxWildcardNilledStillValidated(t *testing.T) {
	compileValidate := func(t *testing.T, instanceXML string) error {
		t.Helper()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType><xs:sequence>
      <xs:any processContents="lax" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	t.Run("nilled lax element with invalid content is rejected", func(t *testing.T) {
		t.Parallel()
		// xsi:nil="true" must not bypass validation: "not-int" is not a valid xs:int.
		inst := `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<id xsi:type="xs:int" xsi:nil="true">not-int</id></doc>`
		require.Error(t, compileValidate(t, inst),
			"nilled undeclared lax element must still be validated against xsi:type")
	})

	t.Run("nilled lax element with empty content the type forbids is rejected", func(t *testing.T) {
		t.Parallel()
		// Empty content is not a valid xs:int, and there is no nillable declaration
		// to make the element legitimately nil.
		inst := `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<id xsi:type="xs:int" xsi:nil="true"></id></doc>`
		require.Error(t, compileValidate(t, inst))
	})

	t.Run("nilled lax element with empty content the type permits is valid", func(t *testing.T) {
		t.Parallel()
		// Empty content IS a valid xs:string, so the element validates.
		inst := `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<id xsi:type="xs:string" xsi:nil="true"></id></doc>`
		require.NoError(t, compileValidate(t, inst))
	})
}

// TestIDConstraintEmptyRef covers PR860-REVIEW-002: an identity-constraint with
// a present-but-empty ref="" is the (invalid) ref form and must be a fatal schema
// error, not silently dropped.
func TestIDConstraintEmptyRef(t *testing.T) {
	t.Parallel()
	compile := func(t *testing.T, idc string) error {
		t.Helper()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
    ` + idc + `
  </xs:element>
</xs:schema>`
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
		return err
	}

	require.ErrorIs(t, compile(t, `<xs:unique ref=""/>`), xsd.ErrCompilationFailed)
	require.ErrorIs(t, compile(t, `<xs:key ref=""/>`), xsd.ErrCompilationFailed)
	require.ErrorIs(t, compile(t, `<xs:keyref ref=""/>`), xsd.ErrCompilationFailed)
}

// TestIDConstraintRefForbidsReferAllKinds covers PR860-REVIEW-003: the ref form
// forbids @refer for EVERY kind (key/unique/keyref), not just keyref.
func TestIDConstraintRefForbidsReferAllKinds(t *testing.T) {
	compile := func(t *testing.T, appxIDC string) error {
		t.Helper()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:element name="chap" type="chap">
          <xs:key name="k"><xs:selector xpath="section"/><xs:field xpath="@nr"/></xs:key>
        </xs:element>
        <xs:element name="appx" type="chap">
          ` + appxIDC + `
        </xs:element>
      </xs:choice>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="chap">
    <xs:sequence maxOccurs="unbounded">
      <xs:element name="section">
        <xs:complexType><xs:attribute name="nr" type="xs:int"/></xs:complexType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
		return err
	}

	t.Run("key ref with refer is rejected", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compile(t, `<xs:key ref="k" refer="k"/>`), xsd.ErrCompilationFailed)
	})
	t.Run("unique ref with refer is rejected", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compile(t, `<xs:unique ref="k" refer="k"/>`), xsd.ErrCompilationFailed)
	})
}

// TestIDMinOccursFailureNoSpuriousDangling covers PR860-IDPASS-001: a child that
// MATCHES a particle but is never actually assessed (the particle fails early,
// here an unsatisfied minOccurs) must NOT be classified as ID/IDREF by pass 3.
// recordElemDecl writes actualElemDecl at the match scan BEFORE content
// validation, so relying on it would report a spurious dangling IDREF on top of
// the real structural error. elementTypeForID uses assessedElemType only, which is
// not set for the unassessed child.
func TestIDMinOccursFailureNoSpuriousDangling(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="r" type="xs:IDREF" minOccurs="2" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
	require.NoError(t, err)

	// Only one <r> for a minOccurs=2 element: a structural error. The single <r>
	// matched the particle (so actualElemDecl is recorded) but its occurrence is
	// unsatisfied, so it is never assessed and must not be collected as an IDREF.
	inst := `<doc><r>missing</r></doc>`
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(inst))
	require.NoError(t, err)

	var errs string
	verr := validateWithOutput(t, xsd.NewValidator(schema), idoc, &errs)
	require.Error(t, verr, "the unsatisfied minOccurs must fail validation")
	require.NotContains(t, errs, "There is no ID/IDREF binding",
		"a matched-but-unassessed child must not produce a spurious dangling-IDREF error; got: %q", errs)
}

// TestIDSimpleContentWithChildNoSpuriousDangling covers PR860-IDPASS-STRUCTURAL-
// SIMPLE: a simple-typed ID/IDREF element that pass 1 already rejected for having
// CHILD ELEMENTS must not also produce a fabricated ID/IDREF in pass 3.
// elemTextContent ignores child elements, and a default/fixed must not stand in
// for non-empty (children-bearing) content — so collection is skipped when the
// element has child elements. A genuinely-empty element still uses its default.
func TestIDSimpleContentWithChildNoSpuriousDangling(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="r" type="xs:IDREF" default="missing" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
	require.NoError(t, err)

	validate := func(t *testing.T, inst string) (string, error) {
		t.Helper()
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(inst))
		require.NoError(t, err)
		var errs string
		verr := validateWithOutput(t, xsd.NewValidator(schema), idoc, &errs)
		return errs, verr
	}

	t.Run("child element present reports only the structural error", func(t *testing.T) {
		t.Parallel()
		// <r> has a child element: simple content is violated (pass 1). Its default
		// "missing" must NOT be fabricated into a dangling IDREF.
		errs, verr := validate(t, `<doc><r><bad/></r></doc>`)
		require.Error(t, verr, "simple content with a child element must fail validation")
		require.NotContains(t, errs, "There is no ID/IDREF binding",
			"a structurally-invalid simple element must not produce a spurious dangling IDREF; got: %q", errs)
	})

	t.Run("genuinely empty element still uses its default", func(t *testing.T) {
		t.Parallel()
		// An empty <r/> takes its default "missing"; that value IS still collected
		// and (here) reported as a dangling IDREF — confirming the default path works.
		errs, verr := validate(t, `<doc><r/></doc>`)
		require.Error(t, verr)
		require.Contains(t, errs, "There is no ID/IDREF binding",
			"a genuinely-empty element's default must still be collected; got: %q", errs)
	})
}

// TestIDStrictWildcardFailureNoSpuriousDuplicate covers the post-merge bug where
// the strict-wildcard / no-global-declaration FAILURE path walked the subtree with
// annotateAnyTypeChildren, which (after the lax-assessment work) laxly ASSESSES
// globally-declared descendants and populates assessedElemType. Pass 3 then
// collected xs:ID values from a subtree whose strict wildcard match already
// FAILED, fabricating a duplicate-ID on top of the real strict errors. The fix
// uses canonicalization-only annotateSkipChildren for the strict failure, so the
// unassessed subtree is never collected.
func TestIDStrictWildcardFailureNoSpuriousDuplicate(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="id" type="xs:ID"/>
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:any processContents="strict" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
	require.NoError(t, err)

	// w1/w2 have no global declaration: the strict wildcard fails for each. Their
	// globally-declared <id> grandchildren carry duplicate xs:ID "dup", but the
	// strict-failed subtree is NOT assessed, so no duplicate-ID must be reported.
	inst := `<doc><w1><id>dup</id></w1><w2><id>dup</id></w2></doc>`
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(inst))
	require.NoError(t, err)

	var errs string
	verr := validateWithOutput(t, xsd.NewValidator(schema), idoc, &errs)
	require.Error(t, verr, "the strict wildcard failure must fail validation")
	require.Contains(t, errs, "demanded by the strict wildcard",
		"the real strict-wildcard error must be reported; got: %q", errs)
	require.NotContains(t, errs, "Duplicate key-sequence",
		"a strict-failed (unassessed) subtree must not produce a spurious duplicate-ID; got: %q", errs)
}
