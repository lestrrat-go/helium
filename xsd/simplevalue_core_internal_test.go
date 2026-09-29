package xsd

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// simpleTypeInfoSchema declares list, union, QName/NOTATION, ID-family, and
// whitespace variants, plus anonymous element types, so the memo is compared
// against the walkers on every shape the value validators handle.
const simpleTypeInfoSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:notation name="png" public="image/png"/>
  <xs:simpleType name="intList"><xs:list itemType="xs:int"/></xs:simpleType>
  <xs:simpleType name="shortIntList">
    <xs:restriction base="t:intList"><xs:maxLength value="3"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="smallInt">
    <xs:restriction base="xs:int"><xs:minInclusive value="0"/><xs:maxInclusive value="9"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="tinyInt">
    <xs:restriction base="t:smallInt"><xs:maxInclusive value="3"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="intOrQName"><xs:union memberTypes="xs:int xs:QName"/></xs:simpleType>
  <xs:simpleType name="intOrString"><xs:union memberTypes="xs:int xs:string"/></xs:simpleType>
  <xs:simpleType name="enumUnion">
    <xs:restriction base="t:intOrString"><xs:enumeration value="1"/><xs:enumeration value="a"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="prefixedName">
    <xs:restriction base="xs:QName"><xs:pattern value="t:.*"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="qnameList"><xs:list itemType="xs:QName"/></xs:simpleType>
  <xs:simpleType name="pic">
    <xs:restriction base="xs:NOTATION"><xs:enumeration value="t:png"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="code">
    <xs:restriction base="xs:ID"><xs:length value="4"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="refs"><xs:list itemType="xs:IDREF"/></xs:simpleType>
  <xs:simpleType name="idOrInt"><xs:union memberTypes="xs:int t:code"/></xs:simpleType>
  <xs:simpleType name="norm">
    <xs:restriction base="xs:normalizedString"><xs:maxLength value="8"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="preserved">
    <xs:restriction base="xs:string"><xs:whiteSpace value="replace"/></xs:restriction>
  </xs:simpleType>
  <xs:complexType name="codeContent">
    <xs:simpleContent><xs:extension base="t:code"><xs:attribute name="a" type="xs:QName"/></xs:extension></xs:simpleContent>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="anonList">
          <xs:simpleType><xs:list><xs:simpleType><xs:restriction base="xs:QName"/></xs:simpleType></xs:list></xs:simpleType>
        </xs:element>
        <xs:element name="anonInt">
          <xs:simpleType><xs:restriction base="t:tinyInt"><xs:minInclusive value="1"/></xs:restriction></xs:simpleType>
        </xs:element>
        <xs:element name="content" type="t:codeContent"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// simpleTypeInfoSchema11 adds XSD 1.1 xs:assertion facets, directly, through a
// list item type, and through a union member.
const simpleTypeInfoSchema11 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:simpleType name="even">
    <xs:restriction base="xs:int"><xs:assertion test="$value mod 2 = 0"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="evenList"><xs:list itemType="t:even"/></xs:simpleType>
  <xs:simpleType name="evenOrString"><xs:union memberTypes="t:even xs:string"/></xs:simpleType>
  <xs:simpleType name="derivedEven">
    <xs:restriction base="t:even"><xs:maxInclusive value="10"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="plainInt">
    <xs:restriction base="xs:int"><xs:maxInclusive value="10"/></xs:restriction>
  </xs:simpleType>
</xs:schema>`

func compileSimpleTypeInfoSchema(t *testing.T, src string, version Version) *Schema {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	schema, err := NewCompiler().Version(version).Compile(t.Context(), doc)
	require.NoError(t, err)
	return schema
}

// simpleTypeInfoTargets returns every named type plus the anonymous types of
// the global elements' local element declarations.
func simpleTypeInfoTargets(schema *Schema) []*TypeDef {
	var out []*TypeDef
	for _, td := range schema.types {
		out = append(out, td)
	}
	for _, edecl := range schema.elements {
		if edecl.Type == nil || edecl.Type.ContentModel == nil {
			continue
		}
		for _, p := range edecl.Type.ContentModel.Particles {
			if child, ok := p.Term.(*ElementDecl); ok && child.Type != nil {
				out = append(out, child.Type)
			}
		}
	}
	return out
}

// requireInfoMatchesWalkers checks every memo field against the walker it
// caches, for every target type.
func requireInfoMatchesWalkers(t *testing.T, schema *Schema) {
	t.Helper()
	vc := newValidationContext(schema, &validateConfig{}, "", helium.NilErrorHandler{})
	targets := simpleTypeInfoTargets(schema)
	require.NotEmpty(t, targets)
	for _, td := range targets {
		info := vc.simpleTypeInfo(td)
		require.NotNil(t, info)
		require.Same(t, info, vc.simpleTypeInfo(td), "%s: second lookup returns the memoized info", td.Name)
		require.Equal(t, resolveWhiteSpace(td), info.whiteSpace, "%s: whiteSpace", td.Name)
		require.Equal(t, builtinBaseLocal(td), info.builtinLocal, "%s: builtinLocal", td.Name)
		require.Equal(t, resolveVariety(td), info.variety, "%s: variety", td.Name)
		require.Equal(t, idFamilyType(td), info.idFamily, "%s: idFamily", td.Name)
		require.Equal(t, typeConsultsNS(td, schema.version), info.consultsNS, "%s: consultsNS", td.Name)
		var facets []*FacetSet
		for cur := range baseChain(td) {
			if cur.Facets != nil {
				facets = append(facets, cur.Facets)
			}
		}
		require.Equal(t, facets, info.facets, "%s: facets in most-derived-first order", td.Name)
	}
}

func TestSimpleTypeInfo(t *testing.T) {
	t.Run("matches walkers in 1.0", func(t *testing.T) {
		requireInfoMatchesWalkers(t, compileSimpleTypeInfoSchema(t, simpleTypeInfoSchema, Version10))
	})
	t.Run("matches walkers in 1.1", func(t *testing.T) {
		requireInfoMatchesWalkers(t, compileSimpleTypeInfoSchema(t, simpleTypeInfoSchema, Version11))
		requireInfoMatchesWalkers(t, compileSimpleTypeInfoSchema(t, simpleTypeInfoSchema11, Version11))
	})
	t.Run("derived fields", func(t *testing.T) {
		schema := compileSimpleTypeInfoSchema(t, simpleTypeInfoSchema, Version10)
		vc := newValidationContext(schema, &validateConfig{}, "", helium.NilErrorHandler{})
		lookup := func(local string) *simpleTypeInfo {
			td, ok := schema.LookupType(local, "urn:t")
			require.True(t, ok, local)
			return vc.simpleTypeInfo(td)
		}

		shortList := lookup("shortIntList")
		require.Equal(t, TypeVarietyList, shortList.variety)
		require.Len(t, shortList.facets, 1)
		require.False(t, shortList.consultsNS)

		tiny := lookup("tinyInt")
		require.Equal(t, "int", tiny.builtinLocal)
		require.Equal(t, "collapse", tiny.whiteSpace)
		require.Len(t, tiny.facets, 2)
		require.Equal(t, "3", *tiny.facets[0].MaxInclusive, "the most derived facet set comes first")

		require.Equal(t, "replace", lookup("norm").whiteSpace)
		require.Equal(t, "replace", lookup("preserved").whiteSpace)

		for _, local := range []string{"intOrQName", "prefixedName", "qnameList", "pic"} {
			require.True(t, lookup(local).consultsNS, "%s resolves prefixes", local)
		}
		for _, local := range []string{"intList", "smallInt", "intOrString", "enumUnion", "code", "refs", "norm"} {
			require.False(t, lookup(local).consultsNS, "%s never reads the namespace map", local)
		}

		for _, local := range []string{"code", "refs", "idOrInt"} {
			require.True(t, lookup(local).idFamily, "%s involves xs:ID/xs:IDREF", local)
		}
		require.False(t, lookup("intOrString").idFamily)
	})
	t.Run("assertion facets consult namespaces only in 1.1", func(t *testing.T) {
		schema := compileSimpleTypeInfoSchema(t, simpleTypeInfoSchema11, Version11)
		for _, local := range []string{"even", "evenList", "evenOrString", "derivedEven"} {
			td, ok := schema.LookupType(local, "urn:t")
			require.True(t, ok, local)
			require.True(t, typeConsultsNS(td, Version11), "%s reaches an xs:assertion facet", local)
			require.False(t, typeConsultsNS(td, Version10), "%s: assertions are ignored under 1.0", local)
		}
		plain, ok := schema.LookupType("plainInt", "urn:t")
		require.True(t, ok)
		require.False(t, typeConsultsNS(plain, Version11))
	})
	t.Run("throwaway context has no memo", func(t *testing.T) {
		schema := compileSimpleTypeInfoSchema(t, simpleTypeInfoSchema, Version10)
		td, ok := schema.LookupType("tinyInt", "urn:t")
		require.True(t, ok)
		vc := &validationContext{schema: schema, errorHandler: helium.NilErrorHandler{}}
		require.Nil(t, vc.simpleTypeInfo(td))
		require.Empty(t, vc.typeInfo, "a context without a memo never creates one")
	})
	t.Run("cyclic base chain terminates", func(t *testing.T) {
		a := &TypeDef{Name: QName{Local: "a"}}
		b := &TypeDef{Name: QName{Local: "b"}, BaseType: a}
		a.BaseType = b
		a.MemberTypes = []*TypeDef{b}
		require.False(t, typeConsultsNS(a, Version11))
		info := newSimpleTypeInfo(a, Version11)
		require.Equal(t, resolveWhiteSpace(a), info.whiteSpace)
		require.Empty(t, info.facets)
	})
}
