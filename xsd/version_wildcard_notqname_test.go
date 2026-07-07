package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// mustCompile11Fail asserts schemaXML fails to compile under XSD 1.1.
func mustCompile11Fail(t *testing.T, schemaXML string) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	require.ErrorIs(t, err, xsd.ErrCompilationFailed)
}

// mustCompile11OK asserts schemaXML compiles cleanly under XSD 1.1.
func mustCompile11OK(t *testing.T, schemaXML string) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	require.NoError(t, err)
}

// TestVersion10DefaultWildcardNegatedConstraints covers the default compiler
// path used by the XSD 1.0 conformance suite: negated wildcard constraints must
// be parsed and enforced when they appear in the schema.
func TestVersion10DefaultWildcardNegatedConstraints(t *testing.T) {
	compileDefault := func(t *testing.T, schemaXML string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Compile(t.Context(), doc)
		return err
	}

	t.Run("namespace and notNamespace are mutually exclusive on element wildcard", func(t *testing.T) {
		t.Parallel()
		err := compileDefault(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:any namespace="##any" notNamespace="urn:x" processContents="skip"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("namespace and notNamespace are mutually exclusive on attribute wildcard", func(t *testing.T) {
		t.Parallel()
		err := compileDefault(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute namespace="##any" notNamespace="urn:x" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("notQName name must be in an admitted namespace", func(t *testing.T) {
		t.Parallel()
		err := compileDefault(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:b="urn:b">
  <xs:complexType name="c">
    <xs:sequence>
      <xs:any notNamespace="urn:b" notQName="b:blocked" processContents="skip"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="c"/>
</xs:schema>`)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("anyAttribute notNamespace rejects excluded absent namespace", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute notNamespace="##local" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		err := compileAndValidateV(t, xsd.NewCompiler(), schema, `<e local="x"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), schema,
			`<e n:a="x" xmlns:n="urn:n"/>`))
	})

	t.Run("definedSibling excludes repeated sibling and substitution member", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="root" type="t:ct"/>
  <xs:element name="b" type="xs:string"/>
  <xs:element name="c" type="xs:string"/>
  <xs:element name="d" substitutionGroup="t:b" type="xs:string"/>
  <xs:complexType name="ct">
    <xs:sequence>
      <xs:element ref="t:b"/>
      <xs:any notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      <xs:element ref="t:c"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), schema,
			`<t:root xmlns:t="urn:t"><t:b>one</t:b><t:c>two</t:c></t:root>`))

		errRepeat := compileAndValidateV(t, xsd.NewCompiler(), schema,
			`<t:root xmlns:t="urn:t"><t:b>one</t:b><t:b>two</t:b><t:c>three</t:c></t:root>`)
		require.ErrorIs(t, errRepeat, xsd.ErrValidationFailed)

		errSubst := compileAndValidateV(t, xsd.NewCompiler(), schema,
			`<t:root xmlns:t="urn:t"><t:b>one</t:b><t:d>two</t:d><t:c>three</t:c></t:root>`)
		require.ErrorIs(t, errSubst, xsd.ErrValidationFailed)
	})
}

// TestVersion11WildcardNotNamespace covers the XSD 1.1 @notNamespace constraint
// on xs:anyAttribute and xs:any: it admits any namespace EXCEPT the listed ones
// (with ##local = absent, ##targetNamespace = the schema TNS).
func TestVersion11WildcardNotNamespace(t *testing.T) {
	const attrSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute notNamespace="http://x.com/ http://y.com/" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("1.1 admits an attribute outside the excluded namespaces", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), attrSchema,
			`<e a:z="1" xmlns:a="http://other.com/"/>`)
		require.NoError(t, err)
	})

	t.Run("1.1 rejects an attribute in an excluded namespace", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), attrSchema,
			`<e a:z="1" xmlns:a="http://x.com/"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	const elemSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:any notNamespace="##local" processContents="skip" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("1.1 element wildcard notNamespace=##local rejects an absent-namespace child", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), elemSchema,
			`<root><c/></root>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("1.1 element wildcard notNamespace=##local admits a namespaced child", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), elemSchema,
			`<root><c xmlns="http://ns.com/"/></root>`)
		require.NoError(t, err)
	})
}

// TestVersion11WildcardNotQName covers the XSD 1.1 @notQName disallowed-name set,
// including an explicit QName and ##defined (a name with a global declaration).
func TestVersion11WildcardNotQName(t *testing.T) {
	const attrSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute notQName="xml:space" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("1.1 admits an attribute not named in notQName", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), attrSchema,
			`<e a="1"/>`)
		require.NoError(t, err)
	})

	t.Run("1.1 rejects the excluded xml:space attribute", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), attrSchema,
			`<e xml:space="preserve"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("1.0 still admits xml:space through special-attribute handling", func(t *testing.T) {
		t.Parallel()
		// In 1.0 xml: attributes are leniently allowed before wildcard matching,
		// so the same instance validates even though @notQName is parsed.
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), attrSchema,
			`<e xml:space="preserve"/>`)
		require.NoError(t, err)
	})

	const definedSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute namespace="##any" notQName="##defined" processContents="skip"/>
    </xs:complexType>
  </xs:element>
  <xs:attribute name="g" type="xs:string"/>
</xs:schema>`

	t.Run("1.1 ##defined rejects an attribute with a global declaration", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), definedSchema,
			`<e g="x"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("1.1 ##defined admits an undeclared attribute", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), definedSchema,
			`<e h="x"/>`)
		require.NoError(t, err)
	})
}

// TestVersion11WildcardDefinedSibling covers @notQName="##definedSibling": an
// element wildcard does not claim children whose names match sibling element
// declarations in the same content model.
func TestVersion11WildcardDefinedSibling(t *testing.T) {
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="z"/>
  <xs:complexType name="z">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:any notQName="##definedSibling" processContents="skip" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	t.Run("wildcard admits a non-sibling name", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<root><a>x</a><b/></root>`)
		require.NoError(t, err)
	})

	t.Run("wildcard does not claim a second sibling-named child", func(t *testing.T) {
		t.Parallel()
		// The sibling element "a" has maxOccurs=1; a second <a> is excluded by the
		// ##definedSibling wildcard, so it is not accepted as open content.
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<root><a>x</a><a>y</a></root>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})
}

// TestVersion11WildcardSchemaChecks covers the schema-validity rules added with
// the 1.1 wildcard attributes.
func TestVersion11WildcardSchemaChecks(t *testing.T) {
	t.Run("namespace and notNamespace are mutually exclusive", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute namespace="##other" notNamespace="##targetNamespace" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
	})

	t.Run("notQName name must be in a namespace the wildcard admits", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="c">
    <xs:all>
      <xs:any processContents="lax" namespace="##other" notQName="memory" minOccurs="0" maxOccurs="unbounded"/>
    </xs:all>
  </xs:complexType>
  <xs:element name="c" type="c"/>
</xs:schema>`)
	})

	t.Run("invalid QName in notQName is rejected", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="c">
    <xs:sequence/>
    <xs:anyAttribute processContents="lax" notQName="xml:xml:lang"/>
  </xs:complexType>
  <xs:element name="c" type="c"/>
</xs:schema>`)
	})

	t.Run("valid notNamespace + notQName compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute notNamespace="http://x.com/" notQName="xml:space" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
	})

	t.Run("attribute declaration in the XSI namespace is rejected", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="http://www.w3.org/2001/XMLSchema-instance">
  <xs:attribute name="bogus" type="xs:string"/>
</xs:schema>`)
	})
}

// TestVersion11AttrGroupWildcardIntersection covers the XSD 1.1 attribute
// wildcard INTERSECTION: a type referencing two attribute groups, each with an
// xs:anyAttribute, admits only attributes BOTH groups admit.
func TestVersion11AttrGroupWildcardIntersection(t *testing.T) {
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="t"/>
  <xs:complexType name="t">
    <xs:sequence/>
    <xs:attributeGroup ref="a"/>
    <xs:attributeGroup ref="b"/>
  </xs:complexType>
  <xs:attributeGroup name="a">
    <xs:anyAttribute notNamespace="##local" processContents="skip"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="b">
    <xs:anyAttribute notNamespace="http://eve.com/" processContents="skip"/>
  </xs:attributeGroup>
</xs:schema>`

	t.Run("admits a namespaced attribute neither group excludes", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<e m:adam="m" xmlns:m="http://adam.com/"/>`)
		require.NoError(t, err)
	})

	t.Run("rejects an attribute one group excludes (##local)", func(t *testing.T) {
		t.Parallel()
		// Group "a" excludes ##local (absent namespace); an unqualified attribute is
		// admitted by "b" but not "a", so the intersection rejects it.
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<e local="x"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("rejects an attribute the other group excludes (eve)", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<e f:x="1" xmlns:f="http://eve.com/"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("1.0 applies attribute-group wildcards with complete-wildcard aggregation", func(t *testing.T) {
		t.Parallel()
		// XSD 1.0 complete-wildcard aggregation still admits this attribute. Direct
		// wildcard notNamespace enforcement in the default compiler is covered by
		// TestVersion10DefaultWildcardNegatedConstraints.
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schema,
			`<e m:adam="m" xmlns:m="http://adam.com/"/>`)
		require.NoError(t, err)
	})
}

// TestVersion10MatchAllWildcardRegression guards the version split for a wildcard
// inside xs:all: it is an XSD 1.1-only feature, so in 1.0 the schema is a
// schema-representation error and must be REJECTED at compile (Structures §3.8.2:
// the 1.0 xs:all content model is (annotation?, element*); W3C sunData
// particles00104m1), while 1.1 compiles it and accepts a child the wildcard
// admits.
func TestVersion10MatchAllWildcardRegression(t *testing.T) {
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:all>
      <xs:any processContents="skip" minOccurs="0" maxOccurs="1"/>
    </xs:all>
  </xs:complexType>
  <xs:element name="e" type="t"/>
</xs:schema>`

	t.Run("1.0 rejects wildcard-in-all schema at compile", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		_, cerr := xsd.NewCompiler().Version(xsd.Version10).Compile(t.Context(), doc)
		require.ErrorIs(t, cerr, xsd.ErrCompilationFailed)
	})

	t.Run("1.1 accepts wildcard-in-all content", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema, `<e><b/></e>`)
		require.NoError(t, err)
	})
}

// TestVersion11RestrictionNotQName covers gauntlet finding 2: the
// restriction-derivation check (element-restricts-wildcard NSCompat) must honor
// the base wildcard's @notQName — a derived element the base wildcard EXCLUDES
// is not a valid restriction, so the schema is rejected.
func TestVersion11RestrictionNotQName(t *testing.T) {
	schema := func(notQName string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:sequence>
      <xs:any namespace="##any" notQName="` + notQName + `" processContents="skip"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:sequence>
          <xs:element name="foo" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`
	}

	t.Run("base wildcard excluding the derived element name rejects the restriction", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, schema("foo"))
	})

	t.Run("base wildcard not excluding the derived element name compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, schema("bar"))
	})
}

// TestVersion11RestrictionDefinedSiblingSubset covers gauntlet finding 3: a
// derived wildcard may not DROP ##definedSibling that the base wildcard carries.
func TestVersion11RestrictionDefinedSiblingSubset(t *testing.T) {
	schema := func(derivedNotQName string) string {
		dn := ""
		if derivedNotQName != "" {
			dn = ` notQName="` + derivedNotQName + `"`
		}
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:any namespace="##any" notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:sequence>
          <xs:element name="a" type="xs:string"/>
          <xs:any namespace="##any"` + dn + ` processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`
	}

	t.Run("dropping ##definedSibling in the derived wildcard rejects the restriction", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, schema(""))
	})

	t.Run("retaining ##definedSibling compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, schema("##definedSibling"))
	})
}

// TestVersion11DefinedSiblingAttributeRejected covers gauntlet finding 4:
// ##definedSibling is permitted only on ELEMENT wildcards, not xs:anyAttribute.
func TestVersion11DefinedSiblingAttributeRejected(t *testing.T) {
	t.Run("##definedSibling on xs:anyAttribute is a schema error", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:anyAttribute notQName="##definedSibling" processContents="skip"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
	})

	t.Run("##definedSibling on xs:any (element wildcard) compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:any notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="e" type="t"/>
</xs:schema>`)
	})
}

// TestVersion11RestrictionPerBaseWildcardMax covers gauntlet finding 5: an
// xs:all restriction must respect PER-NAMESPACE wildcard capacity, not just the
// aggregate total. A derived wildcard confined to one base wildcard's namespace
// may not exceed THAT base wildcard's maxOccurs even when the aggregate total
// across all base wildcards would allow it.
func TestVersion11RestrictionPerBaseWildcardMax(t *testing.T) {
	schema := func(derivedMax string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:all>
      <xs:any namespace="http://a/" minOccurs="0" maxOccurs="2" processContents="skip"/>
      <xs:any namespace="http://bb/" minOccurs="0" maxOccurs="2" processContents="skip"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:all>
          <xs:any namespace="http://a/" minOccurs="0" maxOccurs="` + derivedMax + `" processContents="skip"/>
        </xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`
	}

	t.Run("derived wildcard exceeding one base wildcard's max rejects the restriction", func(t *testing.T) {
		t.Parallel()
		// aggregate base max is 4 (2+2); derived max 4 would pass an aggregate-only
		// check, but namespace http://a/ is capped at 2 by its base wildcard.
		mustCompile11Fail(t, schema("4"))
	})

	t.Run("derived wildcard within the per-namespace max compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, schema("2"))
	})
}

// TestVersion11RestrictionConcreteElemWildcardCardinality covers gauntlet
// finding PR858-R2-001: a CONCRETE derived element admitted by a base WILDCARD
// (not mapped to a base element) must participate in wildcard cardinality
// accounting — both its MAX (so extra concrete elements cannot overload a base
// wildcard's maxOccurs) and its MIN (so a required concrete element satisfies a
// base wildcard's minOccurs instead of being ignored).
func TestVersion11RestrictionConcreteElemWildcardCardinality(t *testing.T) {
	t.Run("two concrete derived elements exceed a max-1 base wildcard (rejected)", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:all>
      <xs:any namespace="##any" minOccurs="0" maxOccurs="1" processContents="skip"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:all>
          <xs:element name="a" type="xs:string" minOccurs="0"/>
          <xs:element name="c" type="xs:string" minOccurs="0"/>
        </xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`)
	})

	t.Run("one concrete derived element within a max-1 base wildcard (compiles)", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:all>
      <xs:any namespace="##any" minOccurs="0" maxOccurs="1" processContents="skip"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:all>
          <xs:element name="a" type="xs:string" minOccurs="0"/>
        </xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`)
	})

	t.Run("a required concrete element satisfies a min-1 base wildcard (compiles)", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:all>
      <xs:any namespace="##any" minOccurs="1" maxOccurs="1" processContents="skip"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:all>
          <xs:element name="a" type="xs:string" minOccurs="1"/>
        </xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`)
	})
}

// TestVersion11RestrictionDefinedSiblingNames covers gauntlet finding
// PR858-R2-002: when BOTH base and derived wildcards carry ##definedSibling but
// resolve to DIFFERENT sibling-name sets, the derived may not re-admit a name
// the base excludes. Comparing the marker bit alone is insufficient.
func TestVersion11RestrictionDefinedSiblingNames(t *testing.T) {
	schema := func(includeSibling bool) string {
		derivedAll := `        <xs:all>
          <xs:any notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
        </xs:all>`
		if includeSibling {
			derivedAll = `        <xs:all>
          <xs:element name="a" type="xs:int" minOccurs="0"/>
          <xs:any notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
        </xs:all>`
		}
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:complexType name="b">
    <xs:all>
      <xs:element name="a" type="xs:int" minOccurs="0"/>
      <xs:any notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
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

	t.Run("derived dropping a sibling narrows ##definedSibling exclusions (rejected)", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, schema(false))
	})

	t.Run("derived keeping the same sibling set compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, schema(true))
	})
}

// TestVersion11ExtensionUnionDefined covers the cos-aw-union {disallowed names}
// rule for the attribute-wildcard UNION on extension (XSD 1.1 §3.10.6.3, the
// area gauntlet finding PR858-R2-003 touched). ##defined is folded ONLY as a
// whole (kept iff BOTH operands carry it); it does NOT make an individual QName
// disallowed for the union. So a global attribute one operand excludes via
// ##defined but admits by namespace, while the OTHER operand excludes it
// explicitly, is still ADMITTED by the union (mirrors W3C wild083's `surprise`).
// A QName excluded EXPLICITLY by BOTH operands stays disallowed.
func TestVersion11ExtensionUnionDefined(t *testing.T) {
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:attribute name="foo" type="xs:string"/>
  <xs:complexType name="B">
    <xs:sequence/>
    <xs:anyAttribute namespace="##any" notQName="##defined t:dup" processContents="skip"/>
  </xs:complexType>
  <xs:complexType name="E">
    <xs:complexContent>
      <xs:extension base="t:B">
        <xs:anyAttribute namespace="##any" notQName="t:foo t:dup" processContents="skip"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="t:E"/>
</xs:schema>`

	t.Run("union ADMITS a global attr excluded by ##defined in one operand only (accepted)", func(t *testing.T) {
		t.Parallel()
		// t:foo is excluded by B via ##defined (B admits it by namespace) and by E
		// explicitly. Per cos-aw-union it is still admitted by the union.
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<t:e xmlns:t="urn:t" t:foo="x"/>`)
		require.NoError(t, err)
	})

	t.Run("union excludes a name BOTH operands disallow explicitly (rejected)", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<t:e xmlns:t="urn:t" t:dup="x"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("union admits an ordinary attribute (accepted)", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<t:e xmlns:t="urn:t" xmlns:o="urn:other" o:bar="x"/>`)
		require.NoError(t, err)
	})
}

// TestVersion11DefinedSiblingInlineType covers gauntlet finding PR858-R3-001:
// an INLINE ANONYMOUS complexType carrying an xs:any with
// @notQName="##definedSibling" must have its SiblingNames resolved too (not just
// named schema types), so the wildcard does not falsely admit a second
// sibling-named child.
func TestVersion11DefinedSiblingInlineType(t *testing.T) {
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:string"/>
        <xs:any notQName="##definedSibling" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("inline type wildcard rejects a duplicate sibling-named child", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<e><a>x</a><a>y</a></e>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("inline type wildcard admits a non-sibling child", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema,
			`<e><a>x</a><b/></e>`)
		require.NoError(t, err)
	})
}

// TestVersion11AttrGroupIntersectionProcessContents covers gauntlet finding
// PR858-R4-001: the attribute-wildcard INTERSECTION across attribute groups must
// carry the STRONGER processContents, so the result is order-independent. A skip
// group intersected with a strict group yields a strict wildcard either way: an
// undeclared wildcard attribute is rejected regardless of the group ref order.
func TestVersion11AttrGroupIntersectionProcessContents(t *testing.T) {
	schema := func(firstRef, secondRef string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="t"/>
  <xs:complexType name="t">
    <xs:sequence/>
    <xs:attributeGroup ref="` + firstRef + `"/>
    <xs:attributeGroup ref="` + secondRef + `"/>
  </xs:complexType>
  <xs:attributeGroup name="askip">
    <xs:anyAttribute namespace="##any" processContents="skip"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="bstrict">
    <xs:anyAttribute namespace="##any" processContents="strict"/>
  </xs:attributeGroup>
</xs:schema>`
	}

	t.Run("skip-then-strict rejects an undeclared wildcard attribute", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema("askip", "bstrict"),
			`<e foo:x="1" xmlns:foo="urn:foo"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("strict-then-skip rejects the same attribute (order-independent)", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schema("bstrict", "askip"),
			`<e foo:x="1" xmlns:foo="urn:foo"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})
}

// TestVersion11RestrictionDisjointWildcardProcessContents covers gauntlet finding
// PR858-R4-002: a derived wildcard's processContents must be at least as strong
// as every INTERSECTING base wildcard, not merely the weakest base wildcard in
// the whole union. A skip derived wildcard may not restrict a strict base
// wildcard in the same namespace even though a DISJOINT base wildcard is skip.
func TestVersion11RestrictionDisjointWildcardProcessContents(t *testing.T) {
	schema := func(derivedPC string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:all>
      <xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="1"/>
      <xs:any namespace="urn:bb" processContents="skip" minOccurs="0" maxOccurs="1"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:all>
          <xs:any namespace="urn:a" processContents="` + derivedPC + `" minOccurs="0" maxOccurs="1"/>
        </xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`
	}

	t.Run("skip derived wildcard cannot restrict a strict same-namespace base wildcard", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, schema("skip"))
	})

	t.Run("strict derived wildcard restricting the strict base wildcard compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, schema("strict"))
	})
}

// TestVersion11RestrictionAttrAgainstBaseWildcard covers gauntlet finding
// PR858-R5-001: a derived CONCRETE attribute checked against a base
// xs:anyAttribute must use the full notQName/##defined-aware expanded-name test,
// not namespace-only matching. A base wildcard excluding the attribute (by
// explicit notQName or ##defined) makes the restriction invalid.
func TestVersion11RestrictionAttrAgainstBaseWildcard(t *testing.T) {
	t.Run("base wildcard excluding the attribute via explicit notQName rejects it", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:sequence/>
    <xs:anyAttribute namespace="##any" notQName="bad" processContents="skip"/>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:sequence/>
        <xs:attribute name="bad" type="xs:string"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`)
	})

	t.Run("base wildcard excluding the attribute via ##defined rejects it", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="g" type="xs:string"/>
  <xs:complexType name="b">
    <xs:sequence/>
    <xs:anyAttribute namespace="##any" notQName="##defined" processContents="skip"/>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:sequence/>
        <xs:attribute ref="g"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`)
	})

	t.Run("base wildcard admitting the attribute compiles", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b">
    <xs:sequence/>
    <xs:anyAttribute namespace="##any" notQName="bad" processContents="skip"/>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:sequence/>
        <xs:attribute name="ok" type="xs:string"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`)
	})
}

// TestVersion11SubsetDefinedDischargesNamedExclusion covers gauntlet finding
// PR858-R5-002: the per-name subset test in wildcardConstraintSubset11 is the
// full ##defined-aware "allows expanded name" test. A derived wildcard with
// notQName="##defined" validly restricts a base wildcard with notQName="g" when
// g has a global declaration (the derived ##defined excludes g), and is invalid
// when g is NOT globally declared (the derived re-admits a name the base excludes).
func TestVersion11SubsetDefinedDischargesNamedExclusion(t *testing.T) {
	schema := func(gGlobal bool) string {
		decl := ""
		if gGlobal {
			decl = `  <xs:element name="g" type="xs:string"/>` + "\n"
		}
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + decl + `  <xs:complexType name="b">
    <xs:sequence>
      <xs:any namespace="##any" notQName="g" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="r">
    <xs:complexContent>
      <xs:restriction base="b">
        <xs:sequence>
          <xs:any namespace="##any" notQName="##defined" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="r"/>
</xs:schema>`
	}

	t.Run("derived ##defined discharges base notQName for a globally-declared name", func(t *testing.T) {
		t.Parallel()
		mustCompile11OK(t, schema(true))
	})

	t.Run("derived ##defined does not discharge a non-global name (rejected)", func(t *testing.T) {
		t.Parallel()
		mustCompile11Fail(t, schema(false))
	})
}
