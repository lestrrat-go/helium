package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestRestrictionAttrInheritance10 covers XSD 1.0 §3.4.2 complex type
// {attribute uses}: a complex type derived by RESTRICTION inherits each base
// attribute use (and the base {attribute wildcard}) it does not redeclare or
// prohibit. An instance carrying such an inherited attribute must validate.
// These mirror the W3C msData ctF013/ctG001/ctG013, particlesZ002 and sunData
// combined/009 conformance cases (all expected VALID).
func TestRestrictionAttrInheritance10(t *testing.T) {
	t.Parallel()

	t.Run("complexContent restriction inherits base attribute", func(t *testing.T) {
		t.Parallel()
		// ctF013: fooType restricts myType (which declares myAttr); the instance
		// carries the inherited myAttr.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="myType">
    <xs:choice>
      <xs:element name="myElement" type="xs:string"/>
      <xs:element name="myElement2" type="xs:string"/>
    </xs:choice>
    <xs:attribute name="myAttr"/>
  </xs:complexType>
  <xs:complexType name="fooType">
    <xs:complexContent>
      <xs:restriction base="myType">
        <xs:sequence>
          <xs:element name="myElement" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="fooType"/>
</xs:schema>`
		instanceXML := `<root myAttr="test attribute"><myElement>test data</myElement></root>`
		require.NoError(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})

	t.Run("restriction with no explicit override inherits all base attributes", func(t *testing.T) {
		t.Parallel()
		// sunData combined/009 test.3.v: 'default' restricts 'base' with an empty
		// restriction; a,b,c are all inherited.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="urn:foo" targetNamespace="urn:foo" elementFormDefault="qualified">
  <xs:complexType name="base">
    <xs:attribute name="a" type="xs:string"/>
    <xs:attribute name="b" type="xs:string"/>
    <xs:attribute name="c" type="xs:string"/>
  </xs:complexType>
  <xs:element name="default">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="base"/>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<foo:default xmlns:foo="urn:foo" a="xxx" b="xxx" c="xxx"/>`
		require.NoError(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})

	t.Run("restriction that prohibits one attribute inherits the rest", func(t *testing.T) {
		t.Parallel()
		// sunData combined/009 test.11.v: 'prohibit' restricts 'base' prohibiting c;
		// a,b inherited, c absent from the instance.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="urn:foo" targetNamespace="urn:foo" elementFormDefault="qualified">
  <xs:complexType name="base">
    <xs:attribute name="a" type="xs:string"/>
    <xs:attribute name="b" type="xs:string"/>
    <xs:attribute name="c" type="xs:string"/>
  </xs:complexType>
  <xs:element name="prohibit">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="base">
          <xs:attribute name="c" use="prohibited"/>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<foo:prohibit xmlns:foo="urn:foo" a="xxx" b="xxx"/>`
		require.NoError(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})

	t.Run("restriction over a wildcard base inherits the named attribute use", func(t *testing.T) {
		t.Parallel()
		// particlesZ002 Derived3: restricts Base3 (foo + anyAttribute ##local),
		// prohibiting bar; foo is inherited as an attribute USE (not via the
		// wildcard, which a restriction does not inherit), so foo on the instance
		// matches the inherited use.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base3">
    <xs:attribute name="foo"/>
    <xs:anyAttribute namespace="##local"/>
  </xs:complexType>
  <xs:complexType name="Derived3">
    <xs:complexContent>
      <xs:restriction base="Base3">
        <xs:attribute name="bar" use="prohibited"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived3"/>
</xs:schema>`
		instanceXML := `<root foo="123"/>`
		require.NoError(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})

	t.Run("empty restriction does NOT inherit the base attribute wildcard", func(t *testing.T) {
		t.Parallel()
		// sunData combined/008 alias/test.10.n: 'alias' is an empty restriction of a
		// base carrying anyAttribute; in XSD 1.0 the restriction's wildcard is
		// computed solely from its own content, so it has NONE — an instance
		// attribute the base wildcard would admit is rejected.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="urn:foo" xmlns:a="urn:a" targetNamespace="urn:foo" elementFormDefault="qualified">
  <xs:complexType name="base">
    <xs:anyAttribute namespace="urn:a urn:b" processContents="skip"/>
  </xs:complexType>
  <xs:element name="alias">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="base"/>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<foo:alias xmlns:foo="urn:foo" xmlns:a="urn:a" a:xxx="xxx"/>`
		require.Error(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})

	t.Run("simpleContent restriction inherits base attribute", func(t *testing.T) {
		t.Parallel()
		// ctE019: fooType simpleContent-restricts mytype1 (attrTest1, attrTest2),
		// redeclaring only attrTest1; attrTest2 is inherited.
		schemaXML := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns="a">
  <xsd:simpleType name="myType">
    <xsd:restriction base="xsd:string"/>
  </xsd:simpleType>
  <xsd:complexType name="mytype1">
    <xsd:simpleContent>
      <xsd:extension base="myType">
        <xsd:attribute ref="attrTest1"/>
        <xsd:attribute ref="attrTest2"/>
      </xsd:extension>
    </xsd:simpleContent>
  </xsd:complexType>
  <xsd:complexType name="fooType">
    <xsd:simpleContent>
      <xsd:restriction base="mytype1">
        <xsd:attribute ref="attrTest1"/>
      </xsd:restriction>
    </xsd:simpleContent>
  </xsd:complexType>
  <xsd:attribute name="attrTest1" type="xsd:int"/>
  <xsd:attribute name="attrTest2" type="xsd:ID"/>
  <xsd:element name="doc" type="fooType"/>
</xsd:schema>`
		instanceXML := `<a:doc xmlns:a="a" a:attrTest1="1" a:attrTest2="b"/>`
		require.NoError(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})
}

// TestExtensionOfRestrictionAttrInheritance10 covers the UBL/CCTS pattern: when
// E extends R and R restricts B while only redeclaring some of B's attributes,
// E's effective {attribute uses} must still include the undeclared uses R
// inherited from B (XSD 1.0 §3.4.2.2).
func TestExtensionOfRestrictionAttrInheritance10(t *testing.T) {
	t.Parallel()

	const simpleContentSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="E"/>
  <xs:complexType name="E">
    <xs:simpleContent>
      <xs:extension base="R"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:simpleContent>
      <xs:restriction base="B">
        <xs:attribute name="mimeCode" type="xs:normalizedString" use="required"/>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:simpleContent>
      <xs:extension base="xs:base64Binary">
        <xs:attribute name="mimeCode" type="xs:normalizedString" use="optional"/>
        <xs:attribute name="filename" type="xs:string" use="optional"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`

	const complexContentSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="E"/>
  <xs:complexType name="E">
    <xs:complexContent>
      <xs:extension base="R"/>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent>
      <xs:restriction base="B">
        <xs:sequence>
          <xs:element name="c" type="xs:string"/>
        </xs:sequence>
        <xs:attribute name="a1" type="xs:string" use="required"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:sequence>
      <xs:element name="c" type="xs:string"/>
    </xs:sequence>
    <xs:attribute name="a1" type="xs:string" use="optional"/>
    <xs:attribute name="a2" type="xs:string" use="optional"/>
  </xs:complexType>
</xs:schema>`

	cases := map[string]struct {
		schema   string
		instance string
		wantErr  bool
	}{
		"simpleContent extension of restriction inherits undeclared base attribute": {
			schema:   simpleContentSchema,
			instance: `<e mimeCode="application/pdf" filename="x.pdf">QQ==</e>`,
		},
		"simpleContent extension of restriction still requires redeclared required attribute": {
			schema:   simpleContentSchema,
			instance: `<e filename="x.pdf">QQ==</e>`,
			wantErr:  true,
		},
		"complexContent extension of restriction inherits undeclared base attribute": {
			schema:   complexContentSchema,
			instance: `<e a1="x" a2="y"><c>z</c></e>`,
		},
		"complexContent extension of restriction still requires redeclared required attribute": {
			schema:   complexContentSchema,
			instance: `<e a2="y"><c>z</c></e>`,
			wantErr:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			err := compileAndValidate(t, tc.schema, tc.instance, nil)

			// Assert
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestExtensionOwnAttrsSurviveBaseGrowth10 pins the effective {attribute uses}
// of an extension against the base attribute slice growing after the extension
// merged it. The slices that must carry a spare slot are the ones a later append
// writes into: the sibling base B declares three attributes and the restriction R
// declares three of its own, and because a type's []*AttrUse grows one use at a
// time (append growth 1, 2, 4, ...) each of those is len 3 / cap 4 with exactly one
// spare slot. A merge that appends the extension's own uses directly onto such a
// slice parks them in that spare slot, where the next append to the same slice —
// E2 merging B again, or finalizeAttrUses10 folding the inherited g1 into R —
// overwrites them and the extension silently loses its own attribute. The other
// fixture types drive those appends: G declares four attributes (r1, r2, r3, g1) so
// that R inherits g1, and E1, E2 and E each declare one own attribute.
//
// Refutation of a review finding that these cases miss the "five-to-seven-attribute
// slice-capacity boundaries": three IS such a boundary. Append growth for a type's
// []*AttrUse runs 1, 2, 4, so a three-use base carries cap 4 with one spare slot,
// which is the exact state the aliasing defect needs. Verified by reverting concatAttrUses
// to `append(base, own...)` with everything else unchanged: four of the five subtests below
// fail, and all five pass with concatAttrUses in place. A five-to-seven-attribute base is
// the same mechanism on a larger array, and concatAttrUses never appends into the base's
// array at any length, so no length-specific behavior exists for a wider band to reach.
func TestExtensionOwnAttrsSurviveBaseGrowth10(t *testing.T) {
	t.Parallel()

	// Two extensions of one base: each merges the same base slice, so E2's merge
	// would overwrite the use E1 parked in the base's spare capacity.
	const siblingExtensionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e1" type="E1"/>
  <xs:element name="e2" type="E2"/>
  <xs:complexType name="B">
    <xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>
    <xs:attribute name="b1" type="xs:string"/>
    <xs:attribute name="b2" type="xs:string"/>
    <xs:attribute name="b3" type="xs:string"/>
  </xs:complexType>
  <xs:complexType name="E1">
    <xs:complexContent>
      <xs:extension base="B">
        <xs:attribute name="own1" type="xs:string"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="E2">
    <xs:complexContent>
      <xs:extension base="B">
        <xs:attribute name="own2" type="xs:string"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	// E extends R, and R then inherits g1 from its own base G: that inheriting
	// append would overwrite the use E parked in R's spare capacity.
	const extensionOfRestrictionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="E"/>
  <xs:complexType name="G">
    <xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>
    <xs:attribute name="r1" type="xs:string"/>
    <xs:attribute name="r2" type="xs:string"/>
    <xs:attribute name="r3" type="xs:string"/>
    <xs:attribute name="g1" type="xs:string"/>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent>
      <xs:restriction base="G">
        <xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>
        <xs:attribute name="r1" type="xs:string"/>
        <xs:attribute name="r2" type="xs:string"/>
        <xs:attribute name="r3" type="xs:string"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="E">
    <xs:complexContent>
      <xs:extension base="R">
        <xs:attribute name="own" type="xs:string"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	cases := map[string]struct {
		schema   string
		instance string
	}{
		"first of two sibling extensions keeps its own attribute": {
			schema:   siblingExtensionSchema,
			instance: `<e1 own1="x"><c>z</c></e1>`,
		},
		"second of two sibling extensions keeps its own attribute": {
			schema:   siblingExtensionSchema,
			instance: `<e2 own2="x"><c>z</c></e2>`,
		},
		"sibling extension still inherits the base attributes": {
			schema:   siblingExtensionSchema,
			instance: `<e1 b1="x" b2="y" b3="z" own1="w"><c>z</c></e1>`,
		},
		"extension of a restriction keeps its own attribute": {
			schema:   extensionOfRestrictionSchema,
			instance: `<e own="x"><c>z</c></e>`,
		},
		"extension of a restriction inherits the grandbase attribute": {
			schema:   extensionOfRestrictionSchema,
			instance: `<e r1="x" g1="y" own="z"><c>z</c></e>`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			err := compileAndValidate(t, tc.schema, tc.instance, nil)

			// Assert
			require.NoError(t, err)
		})
	}
}

// TestSimpleContentRestrictionKeepsBaseAttrs covers a pattern that ISO 20022
// and EPC SEPA schemas use everywhere (GitHub issue #1483):
//
//	<xs:complexType name="Amt">                      <!-- a decimal amount -->
//	  <xs:simpleContent>                             <!-- with a required  -->
//	    <xs:extension base="xs:decimal">             <!-- Ccy attribute    -->
//	      <xs:attribute name="Ccy" use="required"/>
//	...
//	<xs:restriction base="Amt">                      <!-- same type, but   -->
//	  <xs:minInclusive value="0.01"/>                <!-- amount >= 0.01   -->
//	</xs:restriction>
//
// The restriction only adds a limit on the number. It does not mention the Ccy
// attribute again, because it does not need to: XSD says a type derived by
// restriction keeps every attribute of its base type unless it says otherwise
// (XSD 1.0 Structures §3.4.2, {attribute uses}). So the derived type still has
// a required Ccy attribute, and an element that leaves out Ccy is invalid.
// libxml2 (xmllint) accepts the schema and enforces both the attribute and the
// 0.01 minimum.
//
// Two separate things have to work for this schema, and both are checked in
// XSD 1.0 and XSD 1.1 mode:
//
//  1. The schema must compile. The restriction rule "a required base attribute
//     must have a matching attribute in the derived type" is satisfied by the
//     attribute the derived type keeps from its base. The only way to break it
//     is to write use="prohibited" for that attribute.
//  2. The limits the restriction adds must be checked in instance documents.
//     They can be written directly inside <xs:restriction> (minInclusive
//     below) or as a nested <xs:simpleType>. A value below the minimum, or
//     above the maximum of the nested type, is invalid.
//
// Each case lists the instance documents to validate and whether each one must
// be accepted.
func TestSimpleContentRestrictionKeepsBaseAttrs(t *testing.T) {
	t.Parallel()

	const amtType = `<xs:complexType name="Amt">
    <xs:simpleContent>
      <xs:extension base="xs:decimal">
        <xs:attribute name="Ccy" type="xs:string" use="required"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>`

	cases := map[string]struct {
		schema    string
		instances map[string]bool // instance XML -> must be valid
	}{
		// The schema from issue #1483: the restriction adds a minInclusive
		// facet and does not repeat the Ccy attribute.
		"facet added directly, attribute not repeated": {
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  ` + amtType + `
  <xs:element name="A">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="Amt">
          <xs:minInclusive value="0.01"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			instances: map[string]bool{
				`<A Ccy="EUR">1</A>`:    true,  // has Ccy, amount within the limit
				`<A>1</A>`:              false, // Ccy is still required
				`<A Ccy="EUR">0</A>`:    false, // below the minInclusive of 0.01
				`<A Ccy="EUR">abc</A>`:  false, // not a decimal at all
				`<A Ccy="EUR">0.5</A>`:  true,  // above the minimum
				`<A Ccy="EUR">0.01</A>`: true,  // the minimum itself is allowed
			},
		},
		// The same idea with the limit written as a nested simple type. The
		// derived type repeats Ccy here, which is also allowed.
		"limit in a nested simpleType": {
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  ` + amtType + `
  <xs:element name="A">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="Amt">
          <xs:simpleType>
            <xs:restriction base="xs:decimal">
              <xs:maxInclusive value="5"/>
            </xs:restriction>
          </xs:simpleType>
          <xs:attribute name="Ccy" type="xs:string" use="required"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			instances: map[string]bool{
				`<A Ccy="EUR">3</A>`: true,  // within the nested type's maximum
				`<A Ccy="EUR">9</A>`: false, // above the nested type's maximum of 5
				`<A>3</A>`:           false, // Ccy is required
			},
		},
		// Writing use="prohibited" removes the base's required attribute, which
		// a restriction may not do. This must still fail to compile.
		"required attribute prohibited": {
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  ` + amtType + `
  <xs:element name="A">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="Amt">
          <xs:attribute name="Ccy" use="prohibited"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
	}

	for _, version := range []xsd.Version{xsd.Version10, xsd.Version11} {
		for name, tc := range cases {
			t.Run(version.String()+" "+name, func(t *testing.T) {
				t.Parallel()
				schema, errs, err := compileWith(t, version, tc.schema)
				if tc.instances == nil {
					require.Error(t, err)
					require.Contains(t, errs, "A matching attribute use for the 'required' attribute use 'Ccy'")
					return
				}
				require.NoError(t, err, errs)
				for instance, wantValid := range tc.instances {
					err := validateXML(t, schema, instance)
					if wantValid {
						require.NoError(t, err, instance)
						continue
					}
					require.Error(t, err, instance)
				}
			})
		}
	}
}

// TestRestrictionOfRestrictionAttrs checks a restriction whose base is itself
// a restriction. The middle type B does not mention its base's attribute, so B
// keeps it (see TestSimpleContentRestrictionKeepsBaseAttrs). A type C that
// restricts B must be compared with every attribute B has, including the one B
// only kept from A. xmllint gives the same result for each case.
func TestRestrictionOfRestrictionAttrs(t *testing.T) {
	t.Parallel()

	// schemaFor builds A -> B -> C, where A declares aDecl, B restricts A
	// without mentioning any attribute, and C restricts B with cDecl.
	schemaFor := func(aDecl, cDecl string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="A">
    ` + aDecl + `
  </xs:complexType>
  <xs:complexType name="B">
    <xs:complexContent><xs:restriction base="A"/></xs:complexContent>
  </xs:complexType>
  <xs:complexType name="C">
    <xs:complexContent>
      <xs:restriction base="B">
        ` + cDecl + `
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="C"/>
</xs:schema>`
	}

	cases := map[string]struct {
		schema  string
		wantErr string // empty: the schema must compile
	}{
		// C may repeat an optional attribute that B kept from A.
		"repeat kept optional attribute": {
			schema: schemaFor(`<xs:attribute name="o" type="xs:string"/>`,
				`<xs:attribute name="o" type="xs:string"/>`),
		},
		// C may not remove a required attribute that B kept from A.
		"prohibit kept required attribute": {
			schema: schemaFor(`<xs:attribute name="r" type="xs:string" use="required"/>`,
				`<xs:attribute name="r" use="prohibited"/>`),
			wantErr: "A matching attribute use for the 'required' attribute use 'r'",
		},
		// C may not make a required attribute that B kept from A optional.
		"make kept required attribute optional": {
			schema: schemaFor(`<xs:attribute name="r" type="xs:string" use="required"/>`,
				`<xs:attribute name="r" type="xs:string" use="optional"/>`),
			wantErr: "The 'optional' attribute use is inconsistent with the corresponding 'required' attribute use",
		},
	}

	for _, version := range []xsd.Version{xsd.Version10, xsd.Version11} {
		for name, tc := range cases {
			t.Run(version.String()+" "+name, func(t *testing.T) {
				t.Parallel()
				_, errs, err := compileWith(t, version, tc.schema)
				if tc.wantErr == "" {
					require.NoError(t, err, errs)
					return
				}
				require.Error(t, err)
				require.Contains(t, errs, tc.wantErr)
			})
		}
	}
}
