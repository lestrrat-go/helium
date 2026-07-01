package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileV compiles schemaXML with c and returns the compile error (or nil).
func compileV(t *testing.T, c xsd.Compiler, schemaXML string) error {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	_, err = c.Compile(t.Context(), doc)
	return err
}

// TestVersion11WildcardStaticEDC covers the XSD 1.1 addition to "Element
// Declarations Consistent": a content model containing a local element
// declaration particle AND a (lax/strict) wildcard that ·allows· its name is
// invalid when a same-named GLOBAL element declaration exists whose {type table}
// (conditional type assignment) differs from the local particle's. Only the type
// table is compared — a differing type DEFINITION is permitted (a wildcard
// intentionally admits differently-typed elements).
func TestVersion11WildcardStaticEDC(t *testing.T) {
	t.Parallel()

	// Local element 'a' has no type table; the strict wildcard allows it; the
	// global 'a' carries a type table (xs:alternative). Tables differ -> invalid.
	const schemaStrictBadTable = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing">
    <xs:sequence>
      <xs:element name="a"/>
      <xs:any namespace="##local" processContents="strict"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="a">
    <xs:alternative type="xs:integer"/>
  </xs:element>
</xs:schema>`

	// Same, with a lax wildcard.
	const schemaLaxBadTable = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing">
    <xs:sequence>
      <xs:element name="a"/>
      <xs:any namespace="##local" processContents="lax"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="a">
    <xs:alternative type="xs:integer"/>
  </xs:element>
</xs:schema>`

	// Local 'a' has a type table; the global 'a' has none. Tables differ -> invalid.
	const schemaLocalTableGlobalNone = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing">
    <xs:sequence>
      <xs:element name="a">
        <xs:alternative test="0 = 1" type="xs:integer"/>
        <xs:alternative type="xs:date"/>
      </xs:element>
      <xs:any namespace="##local" processContents="strict"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="a" type="xs:date"/>
</xs:schema>`

	// Neither side carries a type table: consistent, so the schema compiles and
	// the wildcard match resolves to the (typeless) global 'a'.
	const schemaConsistent = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:any namespace="##local" processContents="lax"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="a" type="xs:string"/>
</xs:schema>`

	// A skip wildcard never resolves to the global declaration, so the type-table
	// difference imposes no EDC constraint: the schema is valid.
	const schemaSkipNoConstraint = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing">
    <xs:sequence>
      <xs:element name="a"/>
      <xs:any namespace="##local" processContents="skip"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="a">
    <xs:alternative type="xs:integer"/>
  </xs:element>
</xs:schema>`

	for name, schema := range map[string]string{
		"strict wildcard + global type table": schemaStrictBadTable,
		"lax wildcard + global type table":    schemaLaxBadTable,
		"local type table + global no table":  schemaLocalTableGlobalNone,
	} {
		t.Run("1.1 rejects "+name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, compileV(t, xsd.NewCompiler().Version(xsd.Version11), schema), xsd.ErrCompilationFailed)
		})
		t.Run("1.0 ignores "+name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, compileV(t, xsd.NewCompiler().Version(xsd.Version10), schema))
		})
	}

	t.Run("1.1 accepts consistent type tables", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileV(t, xsd.NewCompiler().Version(xsd.Version11), schemaConsistent))
	})

	t.Run("1.1 accepts skip wildcard regardless of type table", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileV(t, xsd.NewCompiler().Version(xsd.Version11), schemaSkipNoConstraint))
	})
}

// TestVersion11WildcardDynamicEDCBaseChain covers the dynamic EDC check across a
// derivation chain: a restriction drops a base type's local element declaration
// but admits the same name through a wildcard. The base declaration's type still
// constrains the element, so the wildcard's governing type (from a same-named
// global declaration, resolved laxly) must be validly substitutable for the base
// local type — otherwise the instance is invalid.
func TestVersion11WildcardDynamicEDCBaseChain(t *testing.T) {
	t.Parallel()

	// Base 'zing' declares local 'e' as union(date,time); restriction 'zang' drops
	// it but keeps a lax ##local wildcard. Global 'e' is xs:duration. An instance
	// <e>PT12H</e> resolves through the wildcard to xs:duration, which is NOT
	// substitutable for the base local union(date,time) -> invalid.
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="zing">
    <xs:sequence>
      <xs:element name="e" minOccurs="0">
        <xs:simpleType><xs:union memberTypes="xs:date xs:time"/></xs:simpleType>
      </xs:element>
      <xs:element name="f" type="xs:integer"/>
      <xs:any namespace="##local" processContents="lax"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="zang">
    <xs:complexContent>
      <xs:restriction base="zing">
        <xs:sequence>
          <xs:element name="f" type="xs:integer"/>
          <xs:any namespace="##local" processContents="lax"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="zang"/>
  <xs:element name="e" type="xs:duration"/>
</xs:schema>`

	t.Run("1.1 rejects base-local-inconsistent wildcard match", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<doc><f>42</f><e>PT12H</e></doc>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	// XSD 1.0 has no dynamic EDC: the wildcard-resolved xs:duration governs and
	// PT12H is a valid duration, so the instance is accepted.
	t.Run("1.0 accepts (no dynamic EDC)", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schema,
			`<doc><f>42</f><e>PT12H</e></doc>`)
		require.NoError(t, err)
	})
}

// TestVersion11SkipWildcardIDCScoping covers identity-constraint selector scoping
// over processContents="skip" wildcard content: an xs:key/xs:unique selector must
// NOT pick elements inside a skip-matched (un-assessed) subtree, so a missing or
// duplicate field value there does not invalidate the instance.
func TestVersion11SkipWildcardIDCScoping(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:s="urn:s" targetNamespace="urn:s" elementFormDefault="qualified">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:choice maxOccurs="unbounded">
          <xs:element ref="s:note"/>
          <xs:element ref="s:wrapper"/>
        </xs:choice>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="id-keys">
      <xs:selector xpath=".//s:note"/>
      <xs:field xpath="@id"/>
    </xs:key>
  </xs:element>
  <xs:element name="note">
    <xs:complexType>
      <xs:choice><xs:element maxOccurs="unbounded" ref="s:p"/></xs:choice>
      <xs:attribute name="id" type="xs:string" use="optional"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="p">
    <xs:complexType mixed="true">
      <xs:attribute name="id" type="xs:string"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="wrapper">
    <xs:complexType mixed="true">
      <xs:sequence>
        <xs:any maxOccurs="unbounded" minOccurs="0" namespace="##any" processContents="skip"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	// A skipped <note> with no id: the key would fail if it were selected, but it
	// is un-assessed skip content and excluded from the selector node-set.
	const instanceMissingID = `<doc xmlns="urn:s">
  <note id="note1"><p>x</p></note>
  <wrapper>text <note><p>y</p></note></wrapper>
</doc>`

	// A skipped <note> with a duplicate id: likewise not selected, so uniqueness is
	// not violated.
	const instanceDupID = `<doc xmlns="urn:s">
  <note id="note1"><p>x</p></note>
  <wrapper>text <note id="note1"><p>y</p></note></wrapper>
</doc>`

	for name, instance := range map[string]string{
		"missing id in skipped content":   instanceMissingID,
		"duplicate id in skipped content": instanceDupID,
	} {
		t.Run("1.1 accepts "+name, func(t *testing.T) {
			t.Parallel()
			err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema, instance)
			require.NoError(t, err)
		})
	}

	// A top-level (assessed) <note> with a duplicate id is still a key violation.
	t.Run("1.1 still rejects duplicate id in assessed content", func(t *testing.T) {
		t.Parallel()
		const dupAssessed = `<doc xmlns="urn:s">
  <note id="note1"><p>x</p></note>
  <note id="note1"><p>y</p></note>
</doc>`
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema, dupAssessed)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})
}
