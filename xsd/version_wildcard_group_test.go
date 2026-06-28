package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileFSV11 compiles mainXSD (a key in fsys) under XSD 1.1, returning the
// compile error (or nil).
func compileFSV11(t *testing.T, fsys fstest.MapFS, mainXSD string) error {
	t.Helper()
	data, err := fsys.ReadFile(mainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	_, err = xsd.NewCompiler().Version(xsd.Version11).Label(mainXSD).FS(fsys).Compile(t.Context(), doc)
	return err
}

// TestVersion11ImportedGroupWildcardNoPanic covers gauntlet finding PR858-R6-001:
// an imported schema whose attribute group declares an xs:anyAttribute must not
// panic (the import sub-compiler previously never initialized attrGroupWildcards),
// and the importing type must see the imported group's wildcard.
func TestVersion11ImportedGroupWildcardNoPanic(t *testing.T) {
	fsys := fstest.MapFS{
		importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:imp="urn:imp" targetNamespace="urn:main">
  <xs:import namespace="urn:imp" schemaLocation="imp.xsd"/>
  <xs:complexType name="t">
    <xs:sequence/>
    <xs:attributeGroup ref="imp:ag"/>
  </xs:complexType>
  <xs:element name="e" type="t"/>
</xs:schema>`)},
		"imp.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:imp">
  <xs:attributeGroup name="ag">
    <xs:anyAttribute namespace="##any" processContents="skip"/>
  </xs:attributeGroup>
</xs:schema>`)},
	}

	t.Run("imported group wildcard compiles without panic", func(t *testing.T) {
		require.NoError(t, compileFSV11(t, fsys, importMainXSD))
	})

	t.Run("imported group wildcard admits an undeclared attribute", func(t *testing.T) {
		data, err := fsys.ReadFile(importMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Label(importMainXSD).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, err)
		inst, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<e xmlns="urn:main" xmlns:foo="urn:foo" foo:x="1"/>`))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), inst))
	})
}

// TestVersion11RedefineAddsGroupWildcard covers gauntlet finding PR858-R6-002: an
// xs:redefine that adds an xs:anyAttribute to an attribute group must make the
// referencing type admit attributes the new wildcard allows (the override path
// previously ignored xs:anyAttribute).
func TestVersion11RedefineAddsGroupWildcard(t *testing.T) {
	fsys := fstest.MapFS{
		importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:redefine schemaLocation="base.xsd">
    <xs:attributeGroup name="ag">
      <xs:attributeGroup ref="t:ag"/>
      <xs:anyAttribute namespace="##any" processContents="skip"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:complexType name="ct">
    <xs:sequence/>
    <xs:attributeGroup ref="t:ag"/>
  </xs:complexType>
  <xs:element name="e" type="t:ct"/>
</xs:schema>`)},
		"base.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t">
  <xs:attributeGroup name="ag">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
</xs:schema>`)},
	}

	require.NoError(t, compileFSV11(t, fsys, importMainXSD))

	data, err := fsys.ReadFile(importMainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Label(importMainXSD).FS(fsys).Compile(t.Context(), doc)
	require.NoError(t, err)

	t.Run("redefine-added wildcard admits an undeclared attribute", func(t *testing.T) {
		inst, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<e xmlns="urn:t" xmlns:foo="urn:foo" foo:x="1"/>`))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), inst))
	})
}

// TestVersion11UnionRetainsSiblingNames covers gauntlet finding PR858-R6-003: a
// materialized wildcard UNION (base xs:all with two disjoint ##definedSibling
// wildcards) must carry the resolved SiblingNames, not just the marker bit. A
// derived restriction whose ##definedSibling wildcard resolves to a NARROWER
// sibling set re-admits a base-excluded sibling and must be rejected.
func TestVersion11UnionRetainsSiblingNames(t *testing.T) {
	schema := func(derivedDropsSibling bool) string {
		derivedAll := `        <xs:all>
          <xs:element name="a" type="xs:string" minOccurs="0"/>
          <xs:element name="bb" type="xs:string" minOccurs="0"/>
          <xs:any namespace="##targetNamespace ##local" notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
        </xs:all>`
		if derivedDropsSibling {
			derivedAll = `        <xs:all>
          <xs:element name="a" type="xs:string" minOccurs="0"/>
          <xs:any namespace="##targetNamespace ##local" notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
        </xs:all>`
		}
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:complexType name="b">
    <xs:all>
      <xs:element name="a" type="xs:string" minOccurs="0"/>
      <xs:element name="bb" type="xs:string" minOccurs="0"/>
      <xs:any namespace="##targetNamespace" notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      <xs:any namespace="##local" notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="t:b">
` + derivedAll + `
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="t:r"/>
</xs:schema>`
	}

	t.Run("derived narrowing the sibling set re-admits a base-excluded sibling (rejected)", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, schema(true))
	})

	t.Run("derived preserving the sibling set compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, schema(false))
	})
}

// TestVersion11UnionSiblingNamesWithoutMarker covers gauntlet finding
// PR858-R8-001: when a base xs:all union mixes a ##definedSibling wildcard with a
// DISJOINT plain wildcard, the materialized union retains the resolved sibling
// names but DROPS the NotQNameDefinedSibling marker (kept only when BOTH operands
// carry it). Those retained names must still be honored: classification
// (wildcardHas11Fields) and the subset check must treat them as exclusions, so a
// derived wildcard that re-admits a base-excluded sibling is rejected.
func TestVersion11UnionSiblingNamesWithoutMarker(t *testing.T) {
	schema := func(derivedKeepsSibling bool) string {
		derivedWildcard := `          <xs:any namespace="##targetNamespace urn:o" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>`
		if derivedKeepsSibling {
			derivedWildcard = `          <xs:any namespace="##targetNamespace urn:o" notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>`
		}
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:complexType name="b">
    <xs:all>
      <xs:element name="a" type="xs:string" minOccurs="0"/>
      <xs:any namespace="##targetNamespace" notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      <xs:any namespace="urn:o" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="t:b">
        <xs:all>
          <xs:element name="a" type="xs:string" minOccurs="0"/>
` + derivedWildcard + `
        </xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="t:r"/>
</xs:schema>`
	}

	t.Run("derived dropping ##definedSibling re-admits the base-excluded sibling (rejected)", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, schema(false))
	})

	t.Run("derived keeping ##definedSibling compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, schema(true))
	})
}

// TestVersion11NotQNameAcceptsXMLNameChars covers gauntlet finding PR858-R6-004:
// notQName must accept any valid XML NCName, including non-ASCII NameChars like
// the middle dot (U+00B7), via the shared xmlchar.IsValidQName validator.
func TestVersion11NotQNameAcceptsXMLNameChars(t *testing.T) {
	mustCompile11OK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute notQName="a`+"·"+`b" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
}

// TestVersion11AttrGroupGrammar covers gauntlet finding PR858-R6-005: an
// attribute group's xs:anyAttribute must be the optional FINAL child and unique.
func TestVersion11AttrGroupGrammar(t *testing.T) {
	t.Run("attribute after the wildcard is rejected", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="ag">
    <xs:anyAttribute namespace="##any" processContents="skip"/>
    <xs:attribute name="x" type="xs:string"/>
  </xs:attributeGroup>
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)
	})

	t.Run("two attribute wildcards are rejected", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="ag">
    <xs:anyAttribute namespace="##any" processContents="skip"/>
    <xs:anyAttribute namespace="##other" processContents="skip"/>
  </xs:attributeGroup>
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)
	})

	t.Run("wildcard as the final child compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="ag">
    <xs:attribute name="x" type="xs:string"/>
    <xs:anyAttribute namespace="##any" processContents="skip"/>
  </xs:attributeGroup>
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)
	})
}
