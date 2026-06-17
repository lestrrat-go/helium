package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestFixedValueSpace verifies that fixed value constraints are compared in the
// declared simple type's value space (applying its whitespace facet), not by
// unconditional TrimSpace (element content) or raw string equality (attribute
// value). A lexically distinct but value-equal instance must be accepted; a
// value-distinct instance must be rejected. For string-family types whose
// whiteSpace facet is "preserve", trailing whitespace is significant.
func TestFixedValueSpace(t *testing.T) {
	type testCase struct {
		name       string
		typ        string // type for the element/attribute
		fixed      string // fixed value
		instance   string // instance value
		wantReject bool
	}

	cases := []testCase{
		// integer — lexical variants that are value-equal to the fixed value
		// must be accepted; a value-distinct instance must be rejected.
		{name: "integer plus sign", typ: "xs:integer", fixed: "1", instance: "+1"},
		{name: "integer leading zero", typ: "xs:integer", fixed: "1", instance: "01"},
		{name: "integer value mismatch", typ: "xs:integer", fixed: "1", instance: "2", wantReject: true},

		// decimal — trailing-zero forms are value-equal.
		{name: "decimal trailing zero", typ: "xs:decimal", fixed: "5", instance: "5.0"},
		{name: "decimal value mismatch", typ: "xs:decimal", fixed: "5", instance: "6", wantReject: true},

		// boolean — "true"/"1" are value-equal.
		{name: "boolean true vs 1", typ: xsBooleanType, fixed: "true", instance: "1"},
		{name: "boolean value mismatch", typ: xsBooleanType, fixed: "true", instance: "0", wantReject: true},

		// float — value space is IEEE-754 single precision, so 16777216 and
		// 16777217 round to the same float32 and must be accepted; xs:double keeps
		// them distinct as float64 and must be rejected.
		{name: "float single-precision equal", typ: xsFloatType, fixed: "16777216", instance: "16777217"},
		{name: "float value mismatch", typ: xsFloatType, fixed: "16777216", instance: "16777220", wantReject: true},
		{name: "double full-precision distinct", typ: xsDoubleType, fixed: "16777216", instance: "16777217", wantReject: true},

		// hexBinary — value space is the decoded octets, so case differences are
		// not significant ("0A" == "0a"); a different byte must be rejected.
		{name: "hexBinary case-insensitive", typ: xsHexBinaryType, fixed: "0A", instance: "0a"},
		{name: "hexBinary value mismatch", typ: xsHexBinaryType, fixed: "0A", instance: "0b", wantReject: true},

		// string (whiteSpace=preserve) — trailing whitespace is significant, so a
		// fixed value with a trailing space must NOT match an instance without it.
		{name: "string trailing space significant", typ: xsStringType, fixed: abcLiteral + " ", instance: abcLiteral, wantReject: true},
		{name: "string exact match", typ: xsStringType, fixed: abcLiteral, instance: abcLiteral},
		{name: "string value mismatch", typ: xsStringType, fixed: abcLiteral, instance: "xyz", wantReject: true},

		// token (whiteSpace=collapse) — leading/trailing/internal whitespace is
		// collapsed, so a padded instance value-matches the fixed value.
		{name: "token collapses whitespace", typ: "xs:token", fixed: abcLiteral, instance: abcLiteral},
	}

	for _, tc := range cases {
		t.Run("element/"+tc.name, func(t *testing.T) {
			t.Parallel()

			schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="` + tc.typ + `" fixed="` + tc.fixed + `"/>
</xs:schema>`
			instanceXML := `<root>` + tc.instance + `</root>`

			runFixedValueCase(t, schemaXML, instanceXML, tc.wantReject)
		})

		t.Run("attribute/"+tc.name, func(t *testing.T) {
			t.Parallel()

			schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="` + tc.typ + `" fixed="` + tc.fixed + `"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
			instanceXML := `<root a="` + tc.instance + `"/>`

			runFixedValueCase(t, schemaXML, instanceXML, tc.wantReject)
		})
	}
}

// TestFixedValueSpaceDerivedWhitespace verifies that a fixed value declared with
// a simple type whose whiteSpace facet is *derived* on a restriction (not the
// builtin's default) is compared after applying that derived facet. A
// builtin-name-only canonicalization would use xs:string's "preserve" and
// wrongly reject a value-equal-but-whitespace-padded instance.
func TestFixedValueSpaceDerivedWhitespace(t *testing.T) {
	// collapsedString restricts xs:string with whiteSpace="collapse", so leading,
	// trailing, and internal runs of whitespace are collapsed before comparison.
	const typeDefs = `  <xs:simpleType name="collapsedString">
    <xs:restriction base="xs:string">
      <xs:whiteSpace value="collapse"/>
    </xs:restriction>
  </xs:simpleType>`

	t.Run("element", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="collapsedString" fixed="a b"/>
</xs:schema>`
		// Instance has padded/extra internal whitespace that collapses to "a b".
		instanceXML := "<root> a   b </root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/mismatch", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="collapsedString" fixed="a b"/>
</xs:schema>`
		instanceXML := "<root>a c</root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("attribute", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="collapsedString" fixed="a b"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root a="  a   b  "/>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})
}

// TestFixedValueSpaceList verifies that a fixed value of an xs:list type is
// compared item-by-item in the item type's value space, so lexically distinct
// but value-equal items (e.g. "01" == "1", "+2" == "2") satisfy the constraint
// while a value-distinct item is rejected.
func TestFixedValueSpaceList(t *testing.T) {
	const typeDefs = `  <xs:simpleType name="intList">
    <xs:list itemType="xs:integer"/>
  </xs:simpleType>`

	t.Run("element/value-equal items", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="intList" fixed="1 2"/>
</xs:schema>`
		instanceXML := "<root>01 +2</root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/item mismatch", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="intList" fixed="1 2"/>
</xs:schema>`
		instanceXML := "<root>1 3</root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/length mismatch", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="intList" fixed="1 2"/>
</xs:schema>`
		instanceXML := "<root>1 2 3</root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("attribute/value-equal items", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="intList" fixed="1 2"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root a="01 +2"/>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})
}

// TestFixedValueSpaceQName verifies that a fixed value of xs:QName is compared in
// value space: each lexical QName is resolved against its own in-scope namespaces
// (the schema's namespaces for the fixed value, the instance's for the instance
// value) and the resolved {namespace URI, local name} pairs are compared. Two
// different prefixes bound to the same URI must be equal; a same-prefix different
// URI binding or a different local name must be rejected.
func TestFixedValueSpaceQName(t *testing.T) {
	const targetURI = "urn:example:target"

	t.Run("element/prefix-differs-same-uri", func(t *testing.T) {
		t.Parallel()
		// Schema binds prefix s -> targetURI and fixes the QName as s:name.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
  <xs:element name="root" type="xs:QName" fixed="s:name"/>
</xs:schema>`
		// Instance binds a different prefix i -> the same URI; i:name resolves equal.
		instanceXML := `<root xmlns:i="` + targetURI + `">i:name</root>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/different-uri", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
  <xs:element name="root" type="xs:QName" fixed="s:name"/>
</xs:schema>`
		// Same prefix text but bound to a different URI -> resolved QNames differ.
		instanceXML := `<root xmlns:s="urn:example:other">s:name</root>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/different-localname", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
  <xs:element name="root" type="xs:QName" fixed="s:name"/>
</xs:schema>`
		instanceXML := `<root xmlns:i="` + targetURI + `">i:other</root>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("attribute/prefix-differs-same-uri", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="xs:QName" fixed="s:name"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root xmlns:i="` + targetURI + `" a="i:name"/>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("attribute/different-uri", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="xs:QName" fixed="s:name"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root xmlns:s="urn:example:other" a="s:name"/>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/instance-prefix-unresolved-rejected", func(t *testing.T) {
		t.Parallel()
		// Schema binds prefix s -> targetURI and fixes s:name. The instance text is
		// the same lexical "s:name" but the instance has NO xmlns:s binding, so the
		// instance QName prefix cannot be resolved. A genuinely unresolvable QName is
		// itself invalid; the fixed comparison must NOT pass on raw lexical match.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
  <xs:element name="root" type="xs:QName" fixed="s:name"/>
</xs:schema>`
		instanceXML := `<root>s:name</root>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})
}

// TestFixedValueSpaceListOfUnion verifies that a fixed value of an xs:list whose
// itemType is a union dispatches each item through the union's *ordered* active
// member. For memberTypes="xs:integer xs:boolean", a value such as "1" validates
// against the first member (xs:integer) and so has active member xs:integer; a
// value such as "true" fails xs:integer and has active member xs:boolean. Each
// item compares only when the fixed and instance items resolve to the SAME
// active member, in that member's value space — so a value-equal item under that
// member (integer "01" == "1") satisfies the constraint, while an item whose
// instance and fixed values resolve to different members does not.
func TestFixedValueSpaceListOfUnion(t *testing.T) {
	const typeDefs = `  <xs:simpleType name="intOrBool">
    <xs:union memberTypes="xs:integer xs:boolean"/>
  </xs:simpleType>
  <xs:simpleType name="unionList">
    <xs:list itemType="intOrBool"/>
  </xs:simpleType>`

	t.Run("element/value-equal items same active member", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="unionList" fixed="1 true"/>
</xs:schema>`
		// Item 1: fixed "1" and instance "01" both have active member xs:integer
		// (integer-equal). Item 2: fixed "true" and instance "true" both have
		// active member xs:boolean (string-equal "true").
		instanceXML := "<root>01 true</root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/cross-member item rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="unionList" fixed="1 true"/>
</xs:schema>`
		// Item 2: fixed "true" has active member xs:boolean, instance "1" has
		// active member xs:integer (first member accepts "1"). Different active
		// members -> the item is not equal even though "1" is boolean-equal to
		// "true". The whole list fixed-value constraint is rejected.
		instanceXML := "<root>01 1</root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/item mismatch", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="unionList" fixed="1 true"/>
</xs:schema>`
		instanceXML := "<root>2 true</root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("attribute/value-equal items same active member", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="unionList" fixed="1 true"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root a="01 true"/>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("attribute/cross-member item rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="unionList" fixed="1 true"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root a="01 1"/>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})
}

// TestFixedValueSpaceUnion verifies that a fixed value of an xs:union is compared
// using XSD's *ordered* active-member semantics, not "any member that makes the
// lexicals compare equal". Each value's active member is the first member type
// (in declaration order) it fully validates against; the fixed and instance
// values compare only when they share the same active member, in that member's
// value space.
//
// For memberTypes="xs:string xs:integer" the first member xs:string accepts any
// text, so EVERY value's active member is xs:string and the comparison is always
// string-based: fixed="1" does NOT accept instance "01" (string-distinct) even
// though a hypothetical integer member would treat them as equal.
func TestFixedValueSpaceUnion(t *testing.T) {
	const typeDefs = `  <xs:simpleType name="strOrInt">
    <xs:union memberTypes="xs:string xs:integer"/>
  </xs:simpleType>`

	t.Run("element/string active member trailing space significant", func(t *testing.T) {
		t.Parallel()
		// fixed has a trailing space; the active member is xs:string (the first
		// member), whiteSpace="preserve", so the instance "abc" (no trailing space)
		// is value-distinct. The constraint must be rejected.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="strOrInt" fixed="` + abcLiteral + ` "/>
</xs:schema>`
		instanceXML := `<root>` + abcLiteral + `</root>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/string active member rejects integer-equal lexicals", func(t *testing.T) {
		t.Parallel()
		// Both "1" and "01" have active member xs:string (first member accepts any
		// text), so they compare as strings and are NOT equal — the earlier
		// "any member that makes them equal" behavior wrongly accepted this.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="strOrInt" fixed="1"/>
</xs:schema>`
		instanceXML := "<root>01</root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/string active member exact lexical match", func(t *testing.T) {
		t.Parallel()
		// Same active member (xs:string) and string-equal values are accepted.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="strOrInt" fixed="1"/>
</xs:schema>`
		instanceXML := "<root>1</root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("attribute/string active member trailing space significant", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="strOrInt" fixed="` + abcLiteral + ` "/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root a="` + abcLiteral + `"/>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("attribute/string active member rejects integer-equal lexicals", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="strOrInt" fixed="1"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root a="01"/>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})
}

// TestFixedValueSpaceUnionOrdered exercises the ordered active-member rules
// directly: a same-active-member value-equal pair is accepted, a cross-member
// pair is rejected, and member declaration ORDER changes the active member (so
// the same lexicals accept or reject depending on which member comes first).
func TestFixedValueSpaceUnionOrdered(t *testing.T) {
	t.Run("integer-first same active member value-equal", func(t *testing.T) {
		t.Parallel()
		// memberTypes="xs:integer xs:boolean": "1" and "01" both validate against
		// xs:integer first, so both have active member xs:integer and compare in
		// integer value space -> equal.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrBool">
    <xs:union memberTypes="xs:integer xs:boolean"/>
  </xs:simpleType>
  <xs:element name="root" type="intOrBool" fixed="1"/>
</xs:schema>`
		instanceXML := "<root>01</root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("cross-member rejected", func(t *testing.T) {
		t.Parallel()
		// memberTypes="xs:integer xs:boolean": fixed "1" has active member
		// xs:integer; instance "true" has active member xs:boolean. Different
		// active members -> not equal.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrBool">
    <xs:union memberTypes="xs:integer xs:boolean"/>
  </xs:simpleType>
  <xs:element name="root" type="intOrBool" fixed="1"/>
</xs:schema>`
		instanceXML := "<root>true</root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("member order changes active member", func(t *testing.T) {
		t.Parallel()
		// memberTypes="xs:boolean xs:integer": "1" validates against xs:boolean
		// first (boolean accepts "1"), so its active member is xs:boolean — not
		// xs:integer. fixed "1" and instance "true" both resolve to xs:boolean
		// (boolean value-equal "1" == "true"), so the constraint is satisfied.
		// With the reverse order (integer first) this same pair is rejected (see
		// the cross-member case above) — demonstrating order matters.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="boolOrInt">
    <xs:union memberTypes="xs:boolean xs:integer"/>
  </xs:simpleType>
  <xs:element name="root" type="boolOrInt" fixed="1"/>
</xs:schema>`
		instanceXML := "<root>true</root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})
}

// TestFixedValueSpaceUnionSharedValueSpace exercises the case where the fixed and
// instance values resolve to DIFFERENT active members that nonetheless live in
// the same comparable value space (xs:integer and xs:decimal both reduce to the
// decimal value space). Such values must compare equal in that shared space even
// though their active members differ. Cross-family pairs (string vs integer,
// integer vs boolean) stay unequal.
func TestFixedValueSpaceUnionSharedValueSpace(t *testing.T) {
	const intOrDec = `  <xs:simpleType name="intOrDec">
    <xs:union memberTypes="xs:integer xs:decimal"/>
  </xs:simpleType>`

	t.Run("element/different members same decimal value space", func(t *testing.T) {
		t.Parallel()
		// fixed "1.0": xs:integer rejects it, so active member is xs:decimal.
		// instance "1": xs:integer accepts it, so active member is xs:integer.
		// Active members differ, but both reduce to the decimal value space and
		// 1.0 == 1, so the constraint is satisfied.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + intOrDec + `
  <xs:element name="root" type="intOrDec" fixed="1.0"/>
</xs:schema>`
		instanceXML := "<root>1</root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("attribute/different members same decimal value space", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + intOrDec + `
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="intOrDec" fixed="1.0"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root a="1"/>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/same value space but unequal values rejected", func(t *testing.T) {
		t.Parallel()
		// fixed "2.0" (active member xs:decimal) vs instance "1" (active member
		// xs:integer): same value space, but 2.0 != 1, so rejected.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + intOrDec + `
  <xs:element name="root" type="intOrDec" fixed="2.0"/>
</xs:schema>`
		instanceXML := "<root>1</root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/cross-family string vs integer rejected", func(t *testing.T) {
		t.Parallel()
		// memberTypes="xs:integer xs:string": fixed "1" -> active member xs:integer
		// (first member accepts it); instance " 1" (leading space) -> xs:integer
		// rejects (leading space invalid for integer), so active member xs:string.
		// integer and string are different value-space families -> rejected.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrStr">
    <xs:union memberTypes="xs:integer xs:string"/>
  </xs:simpleType>
  <xs:element name="root" type="intOrStr" fixed="1"/>
</xs:schema>`
		instanceXML := "<root> x </root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/cross-family integer vs boolean rejected", func(t *testing.T) {
		t.Parallel()
		// Already covered by TestFixedValueSpaceUnionOrdered, restated here to guard
		// the shared-value-space change: integer and boolean are different families.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrBool">
    <xs:union memberTypes="xs:integer xs:boolean"/>
  </xs:simpleType>
  <xs:element name="root" type="intOrBool" fixed="1"/>
</xs:schema>`
		instanceXML := "<root>true</root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/nested union member same decimal value space", func(t *testing.T) {
		t.Parallel()
		// Outer union memberTypes="inner xs:decimal" where inner is ITSELF a union
		// "xs:integer xs:boolean". fixed "1.0": xs:integer rejects it, xs:boolean
		// rejects it, so inner rejects and the active member is the outer xs:decimal.
		// instance "1": validates against inner (xs:integer first), so its active
		// member is the inner union — and within it, the active basic member is
		// xs:integer. The active basic members differ (decimal vs integer) but both
		// reduce to the decimal value space and 1.0 == 1, so the constraint holds.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="inner">
    <xs:union memberTypes="xs:integer xs:boolean"/>
  </xs:simpleType>
  <xs:simpleType name="outer">
    <xs:union memberTypes="inner xs:decimal"/>
  </xs:simpleType>
  <xs:element name="root" type="outer" fixed="1.0"/>
</xs:schema>`
		instanceXML := "<root>1</root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("attribute/nested union member same decimal value space", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="inner">
    <xs:union memberTypes="xs:integer xs:boolean"/>
  </xs:simpleType>
  <xs:simpleType name="outer">
    <xs:union memberTypes="inner xs:decimal"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="outer" fixed="1.0"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root a="1"/>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})
}

// TestFixedValueSpaceXsiTypeDeclaredType verifies that the element fixed-value
// constraint is compared in the element DECLARATION's type, not in an xsi:type
// ACTUAL type that may derive a different whiteSpace facet. A declared
// xs:string (whiteSpace="preserve") fixed="abc " keeps its trailing space, so an
// instance whose xsi:type collapses whitespace and supplies content "abc" must
// still be REJECTED against the declared string fixed value.
func TestFixedValueSpaceXsiTypeDeclaredType(t *testing.T) {
	t.Parallel()
	// collapsedString derives from xs:string with whiteSpace="collapse"; it is a
	// valid xsi:type for the declared xs:string element, but the fixed-value
	// comparison must use the declared xs:string (preserve), not collapse.
	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="collapsedString">
    <xs:restriction base="xs:string">
      <xs:whiteSpace value="collapse"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="xs:string" fixed="` + abcLiteral + ` "/>
</xs:schema>`
	instanceXML := `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="collapsedString">` + abcLiteral + `</root>`
	runFixedValueCase(t, schemaXML, instanceXML, true)
}

// TestFixedValueSpaceUnionWhitespaceOperand verifies that for union values whose
// active members DIFFER but share a value-space family, each operand is
// whitespace-normalized with ITS active member's effective whiteSpace facet
// before the value-space comparison. union fixed="1.0" must accept the
// whitespace-padded instance " 1 " (which collapses to "1" under its xs:integer
// active member) just as it accepts "1".
func TestFixedValueSpaceUnionWhitespaceOperand(t *testing.T) {
	const intOrDec = `  <xs:simpleType name="intOrDec">
    <xs:union memberTypes="xs:integer xs:decimal"/>
  </xs:simpleType>`

	t.Run("attribute/padded instance accepted", func(t *testing.T) {
		t.Parallel()
		// fixed "1.0" -> active member xs:decimal; instance " 1 " -> xs:integer
		// (after collapse). Different members, shared decimal family, value-equal
		// after each operand's whiteSpace collapse.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + intOrDec + `
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="intOrDec" fixed="1.0"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root a=" 1 "/>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})
}

// TestFixedValueSpaceWildcardGlobalFixed verifies that a global attribute matched
// through an xs:anyAttribute wildcard (processContents="strict") still enforces
// the global attribute's fixed value. A different value supplied via the wildcard
// must be REJECTED.
func TestFixedValueSpaceWildcardGlobalFixed(t *testing.T) {
	const gattr = `  <xs:attribute name="ga" type="xs:string" fixed="locked"/>`

	t.Run("wildcard match different value rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + gattr + `
  <xs:element name="root">
    <xs:complexType>
      <xs:anyAttribute processContents="strict"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root ga="other"/>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("wildcard match fixed value accepted", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + gattr + `
  <xs:element name="root">
    <xs:complexType>
      <xs:anyAttribute processContents="strict"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root ga="locked"/>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})
}

func runFixedValueCase(t *testing.T, schemaXML, instanceXML string, wantReject bool) {
	t.Helper()

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)

	var errs string
	err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)

	if wantReject {
		require.Error(t, err)
		require.Contains(t, errs, "fixed value constraint")
		return
	}
	require.NoError(t, err, "validation errors: %s", errs)
}
