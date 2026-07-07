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
  xmlns="urn:main" xmlns:imp="urn:imp" targetNamespace="urn:main">
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

// TestVersion11ExtensionUnionProcessContents covers gauntlet finding
// PR858-R9-001: the attribute-wildcard UNION for TYPE EXTENSION must take the
// DERIVED (second operand's) processContents, not the base's. A base wildcard
// with skip + a 1.1 field (forcing the 1.1 union path) extended by a strict
// wildcard must yield a strict effective wildcard, so an undeclared attribute is
// rejected (strict demands a global declaration).
func TestVersion11ExtensionUnionProcessContents(t *testing.T) {
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:sequence/>
    <xs:anyAttribute namespace="##any" notQName="bad" processContents="skip"/>
  </xs:complexType>
  <xs:complexType name="e">
    <xs:complexContent>
      <xs:extension base="b">
        <xs:anyAttribute namespace="##any" processContents="strict"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="e"/>
</xs:schema>`

	t.Run("extension strict wildcard rejects an undeclared attribute", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<root foo="1"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("extension strict wildcard admits a globally-declared attribute", func(t *testing.T) {
		t.Parallel()
		// A strict wildcard accepts an attribute that DOES have a global
		// declaration, confirming the result is strict (not skip) rather than
		// rejecting everything.
		const declSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="g" type="xs:string"/>
  <xs:complexType name="b">
    <xs:sequence/>
    <xs:anyAttribute namespace="##any" notQName="bad" processContents="skip"/>
  </xs:complexType>
  <xs:complexType name="e">
    <xs:complexContent>
      <xs:extension base="b">
        <xs:anyAttribute namespace="##any" processContents="strict"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="e"/>
</xs:schema>`
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), declSchema,
			`<root g="x"/>`)
		require.NoError(t, err)
	})
}

// TestVersion11ExtensionUnionPlainProcessContents covers gauntlet finding
// PR858-R10-001: the 1.1 wildcard union must be selected by VERSION, not by the
// presence of 1.1-only fields. With PLAIN namespace wildcards (no notQName), a
// base skip extended by a derived strict must still yield a strict effective
// wildcard, so an undeclared attribute is rejected. Previously this fell through
// to the 1.0 path (pc = base skip) and was wrongly accepted.
func TestVersion11ExtensionUnionPlainProcessContents(t *testing.T) {
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:sequence/>
    <xs:anyAttribute namespace="##any" processContents="skip"/>
  </xs:complexType>
  <xs:complexType name="e">
    <xs:complexContent>
      <xs:extension base="b">
        <xs:anyAttribute namespace="##any" processContents="strict"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="e"/>
</xs:schema>`

	t.Run("plain skip base + strict derived rejects an undeclared attribute", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema, `<root foo="1"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("XSD 1.0 keeps the legacy union path (attribute accepted)", func(t *testing.T) {
		t.Parallel()
		// In 1.0 the legacy union path (pc = base skip) is preserved byte-identical;
		// skip admits the undeclared attribute. This guards against regressing the
		// 1.0 behavior while fixing 1.1.
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schema, `<root foo="1"/>`)
		require.NoError(t, err)
	})
}

// TestVersion11AllRestrictionPlainWildcardNamespace covers gauntlet finding
// PR858-R10-001 (xs:all restriction half): a base xs:all union of disjoint PLAIN
// wildcards (##other | ##local) must compute the EXACT admitted namespace set,
// not the 1.0 ##any approximation. A derived restriction adding a
// target-namespace concrete element — which NO base wildcard admits (##other
// excludes the target namespace, ##local admits only absent) — must be rejected.
func TestVersion11AllRestrictionPlainWildcardNamespace(t *testing.T) {
	// elementFormDefault selects the namespace of the derived local element "a":
	// "qualified" → urn:t (the target namespace, admitted by NEITHER ##other nor
	// ##local → invalid); "unqualified" → absent namespace (admitted by ##local →
	// valid).
	schema := func(formDefault string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="` + formDefault + `">
  <xs:complexType name="b">
    <xs:all>
      <xs:any namespace="##other" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      <xs:any namespace="##local" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="t:b">
        <xs:all>
          <xs:element name="a" type="xs:string" minOccurs="0"/>
        </xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="t:r"/>
</xs:schema>`
	}

	t.Run("target-namespace element no base wildcard admits is rejected", func(t *testing.T) {
		t.Parallel()
		// qualified → element "a" is in urn:t: ##other excludes the target namespace
		// and ##local admits only absent, so the base union admits neither. Under
		// the 1.0 approximation (##other|##local ≈ ##any) this wrongly compiled.
		mustCompile11Fail(t, schema("qualified"))
	})

	t.Run("absent-namespace element admitted by ##local compiles", func(t *testing.T) {
		t.Parallel()
		// unqualified → element "a" is in the absent namespace, which ##local admits.
		mustCompile11OK(t, schema("unqualified"))
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

// TestVersion11DefinedSiblingNoCrossTypeAlias covers gauntlet finding
// PR858-REVIEW-001: resolveDefinedSiblings must not mutate a wildcard SHARED
// across types (extension embeds the base model-group pointer; group refs reuse
// the group's particle slice). The base type's ##definedSibling wildcard admits
// `c` (not a base sibling), but the derived extension adds `c` so its wildcard
// excludes it; sharing the wildcard let map-iteration order overwrite the base's
// sibling set. Compiling fresh many times must ALWAYS accept the base instance
// (per-type clone makes resolution deterministic).
func TestVersion11DefinedSiblingNoCrossTypeAlias(t *testing.T) {
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:any notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:extension base="base">
        <xs:sequence>
          <xs:element name="c" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="base"/>
  <xs:element name="droot" type="derived"/>
</xs:schema>`

	// Go randomizes map iteration order, so a single fresh compile may differ from
	// the next. Loop enough to make a cross-type aliasing regression overwhelmingly
	// likely to surface; combine with `go test -count=N` for extra coverage.
	for i := range 50 {
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<root><a>x</a><c>y</c></root>`)
		require.NoErrorf(t, err, "iteration %d: base instance must be valid (c is not a base sibling)", i)
	}
}

// TestVersion10GlobalAttrXSINamespaceAllowed verifies the GLOBAL-attribute
// no-xsi rejection is version-INDEPENDENT: a global attribute's {target
// namespace} is the schema targetNamespace, so a schema whose targetNamespace IS
// the XSI namespace declaring a global attribute is illegal in BOTH XSD 1.0 and
// 1.1 (W3C msMeta/Attribute_w3c attKa015). The XSI namespace is reserved for the
// four processor attributes; a schema may not add to it.
func TestVersion10GlobalAttrXSINamespaceAllowed(t *testing.T) {
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="http://www.w3.org/2001/XMLSchema-instance">
  <xs:attribute name="foo" type="xs:string"/>
</xs:schema>`

	t.Run("1.0 rejects a global attribute in the XSI namespace", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version10).Compile(t.Context(), doc)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("1.1 rejects a global attribute in the XSI namespace", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}

// TestVersion11NotQNameResolveQNameSemantics covers gauntlet finding
// PR858-NQ-001: @notQName tokens use resolve-QName ACTUAL VALUE semantics — an
// unprefixed token resolves through the in-scope DEFAULT namespace (or ABSENT if
// none), NEVER the schema's targetNamespace. Previously the targetNamespace
// fallback both rejected valid `namespace="##local" notQName="a"` schemas and
// excluded the wrong target-namespace name.
func TestVersion11NotQNameResolveQNameSemantics(t *testing.T) {
	t.Run("unprefixed token in a targetNamespace schema with no default resolves to absent", func(t *testing.T) {
		t.Parallel()
		// namespace="##local" admits only the absent namespace. notQName="a" must
		// resolve to {absent}a (NOT {urn:t}a), so it is in an allowed namespace and
		// the schema compiles.
		mustCompile11OK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute namespace="##local" notQName="a" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
	})

	t.Run("unprefixed notQName excludes the absent-namespace name, not the target-namespace one", func(t *testing.T) {
		t.Parallel()
		// The element's wildcard (namespace ##local, notQName="a") must EXCLUDE the
		// absent-namespace attribute "a" (rejected) while still ADMITTING a
		// different absent-namespace attribute "b" (accepted). attributeFormDefault
		// is unqualified, so instance attributes a/b are in the absent namespace.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:element name="e" type="t:ct"/>
  <xs:complexType name="ct">
    <xs:sequence/>
    <xs:anyAttribute namespace="##local" notQName="a" processContents="skip"/>
  </xs:complexType>
</xs:schema>`
		errExcluded := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<t:e xmlns:t="urn:t" a="1"/>`)
		require.ErrorIs(t, errExcluded, xsd.ErrValidationFailed)

		errAdmitted := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<t:e xmlns:t="urn:t" b="1"/>`)
		require.NoError(t, errAdmitted)
	})

	t.Run("unprefixed token resolves through the in-scope default namespace", func(t *testing.T) {
		t.Parallel()
		// With a default namespace (xmlns="urn:t") in scope on the schema, an
		// unprefixed notQName token resolves to {urn:t}. The wildcard namespace
		// "##targetNamespace" admits urn:t, so the schema compiles, and the named
		// element {urn:t}a is excluded while {urn:t}b is admitted.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:any namespace="##targetNamespace" notQName="a" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		// {urn:t}a is excluded by notQName → not admitted by the wildcard → invalid.
		errExcluded := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<root xmlns="urn:t"><a/></root>`)
		require.ErrorIs(t, errExcluded, xsd.ErrValidationFailed)
		// {urn:t}b is admitted.
		errAdmitted := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<root xmlns="urn:t"><b/></root>`)
		require.NoError(t, errAdmitted)
	})

	t.Run("prefixed token with an unbound prefix is a schema error", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute namespace="##any" notQName="nope:a" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
	})
}

// TestVersion11NotNamespaceEmpty covers gauntlet finding PR858-WC-001: an EMPTY
// @notNamespace is valid in XSD 1.1 (xs:basicNamespaceList may be empty) and
// means a `not` constraint with an empty excluded set — it admits ALL
// namespaces. It must compile (not be rejected) and behave as a wildcard that
// admits any name, while staying distinct from an absent @notNamespace.
func TestVersion11NotNamespaceEmpty(t *testing.T) {
	t.Run("xs:anyAttribute notNamespace='' compiles and admits any attribute", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute notNamespace="" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		// An attribute in any namespace (and an unqualified one) is admitted.
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<e a:z="1" xmlns:a="http://x.com/"/>`))
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<e plain="1"/>`))
	})

	t.Run("xs:any notNamespace='' compiles and admits any child", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:any notNamespace="" processContents="skip" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		// Children in any namespace and the absent namespace are admitted.
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<root><c xmlns="http://ns.com/"/><d/></root>`))
	})
}
