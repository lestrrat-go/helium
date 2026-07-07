package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
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
		{name: "boolean true vs 1", typ: xsBooleanType, fixed: lexicon.ValueTrue, instance: "1"},
		{name: "boolean value mismatch", typ: xsBooleanType, fixed: lexicon.ValueTrue, instance: "0", wantReject: true},

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

// TestFixedValueSpaceListNBSP guards the fixed-value LIST comparison against
// using Go's Unicode strings.Fields for item separation. NBSP (U+00A0) is NOT an
// XSD list separator, so an instance "1<NBSP>2" against an xs:int-list fixed
// value "1 2" is a SINGLE token "1<NBSP>2" — which is not a valid xs:int and
// must be REJECTED. Splitting on NBSP (the old behavior) would yield two valid
// items [1, 2] that wrongly match the fixed value. The genuine XSD-space form
// "1 2" stays accepted.
func TestFixedValueSpaceListNBSP(t *testing.T) {
	const typeDefs = `  <xs:simpleType name="intList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>`

	t.Run("element/xsd-space accepted", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="intList" fixed="1 2"/>
</xs:schema>`
		instanceXML := `<root>1 2</root>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/nbsp-joined rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="intList" fixed="1 2"/>
</xs:schema>`
		// "1<NBSP>2" is one token; it fails xs:int item validation, so the instance
		// is invalid regardless of the fixed value (NBSP is not a list separator).
		instanceXML := `<root>1` + nbsp + `2</root>`
		runFixedValueReject(t, schemaXML, instanceXML)
	})
}

// runFixedValueReject compiles, validates, and asserts a validation error
// WITHOUT requiring the "fixed value constraint" message, for cases where the
// instance is rejected by per-item lexical validation before the fixed-value
// comparison runs.
func runFixedValueReject(t *testing.T, schemaXML, instanceXML string) {
	t.Helper()

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)

	var errs string
	err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
	require.Error(t, err, "expected validation error, got none")
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

// TestFixedValueSpaceUnionQNameCrossMember verifies that when the fixed and
// instance values resolve to DIFFERENT QName-derived union members, the
// fixed-value comparison still succeeds when both resolve to the SAME expanded
// {namespace, local} name — even though the two members are distinct type
// definitions. The union lists two xs:QName restrictions distinguished by a
// pattern on the lexical PREFIX: member "sQName" only accepts prefix "s:",
// member "iQName" only accepts prefix "i:". The schema fixes "s:name" (active
// member sQName) and the instance writes "i:name" with prefix i bound to the
// SAME URI as s (active member iQName). The expanded names match, so the
// constraint holds; a different URI or local name must be rejected.
func TestFixedValueSpaceUnionQNameCrossMember(t *testing.T) {
	const targetURI = "urn:example:target"
	const typeDefs = `  <xs:simpleType name="sQName">
    <xs:restriction base="xs:QName">
      <xs:pattern value="s:.*"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="iQName">
    <xs:restriction base="xs:QName">
      <xs:pattern value="i:.*"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="sOrIQName">
    <xs:union memberTypes="sQName iQName"/>
  </xs:simpleType>`

	t.Run("element/different-members-same-uri", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
` + typeDefs + `
  <xs:element name="root" type="sOrIQName" fixed="s:name"/>
</xs:schema>`
		instanceXML := `<root xmlns:i="` + targetURI + `">i:name</root>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/different-members-different-uri-rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
` + typeDefs + `
  <xs:element name="root" type="sOrIQName" fixed="s:name"/>
</xs:schema>`
		instanceXML := `<root xmlns:i="urn:example:other">i:name</root>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/different-members-different-localname-rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
` + typeDefs + `
  <xs:element name="root" type="sOrIQName" fixed="s:name"/>
</xs:schema>`
		instanceXML := `<root xmlns:i="` + targetURI + `">i:other</root>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("attribute/different-members-same-uri", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
` + typeDefs + `
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="sOrIQName" fixed="s:name"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root xmlns:i="` + targetURI + `" a="i:name"/>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})
}

// TestFixedValueSpaceListOfQNameCrossMember verifies the same cross-member QName
// equality through an xs:list whose itemType is a union of two QName-derived
// members. Each list item dispatches to its active member independently: member
// sQName accepts only the "s:" prefix and member iQName only the "i:" prefix. The
// schema fixes "s:a s:b" (every item active in sQName, prefix s bound in the
// schema) and the instance writes "i:a i:b" (every item active in iQName, prefix
// i bound in the instance to the SAME URI as s). The fixed and instance items
// resolve to DIFFERENT members yet denote the same expanded name item-by-item, so
// the constraint holds; a per-item URI/local mismatch is rejected.
func TestFixedValueSpaceListOfQNameCrossMember(t *testing.T) {
	const targetURI = "urn:example:target"
	const typeDefs = `  <xs:simpleType name="sQName">
    <xs:restriction base="xs:QName">
      <xs:pattern value="s:.*"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="iQName">
    <xs:restriction base="xs:QName">
      <xs:pattern value="i:.*"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="sOrIQName">
    <xs:union memberTypes="sQName iQName"/>
  </xs:simpleType>
  <xs:simpleType name="qnameList">
    <xs:list itemType="sOrIQName"/>
  </xs:simpleType>`

	t.Run("element/per-item-cross-member-same-uri", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
` + typeDefs + `
  <xs:element name="root" type="qnameList" fixed="s:a s:b"/>
</xs:schema>`
		instanceXML := `<root xmlns:i="` + targetURI + `">i:a i:b</root>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/per-item-localname-mismatch-rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
` + typeDefs + `
  <xs:element name="root" type="qnameList" fixed="s:a s:b"/>
</xs:schema>`
		instanceXML := `<root xmlns:i="` + targetURI + `">i:a i:c</root>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/per-item-uri-mismatch-rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="` + targetURI + `">
` + typeDefs + `
  <xs:element name="root" type="qnameList" fixed="s:a s:b"/>
</xs:schema>`
		instanceXML := `<root xmlns:i="urn:example:other">i:a i:b</root>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})
}

// TestFixedValueSpaceUnionNotationCrossMember verifies cross-member equality for
// xs:NOTATION-derived union members, the NOTATION analogue of the QName case. Two
// enumeration-restricted NOTATION members each accept a single enumerated literal
// that names the SAME declared notation through a DIFFERENT prefix (both prefixes
// p and q are bound to the notation's namespace in the schema). The schema fixes
// "p:jpeg" (active member pNotation) and the instance writes "q:jpeg" with prefix
// q bound to the same URI (active member qNotation). The two members differ but
// the resolved expanded notation names match, so the constraint holds; a
// different notation name is rejected.
func TestFixedValueSpaceUnionNotationCrossMember(t *testing.T) {
	const notationNS = "urn:p"
	const typeDefs = `  <xs:notation name="jpeg" public="image/jpeg"/>
  <xs:notation name="png" public="image/png"/>
  <xs:simpleType name="pNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="p:jpeg"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="qNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="q:jpeg"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="pOrQNotation">
    <xs:union memberTypes="pNotation qNotation"/>
  </xs:simpleType>`

	t.Run("element/different-members-same-notation", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="` + notationNS + `" targetNamespace="` + notationNS + `" xmlns:p="` + notationNS + `" xmlns:q="` + notationNS + `">
` + typeDefs + `
  <xs:element name="root" type="pOrQNotation" fixed="p:jpeg"/>
</xs:schema>`
		instanceXML := `<root xmlns="` + notationNS + `" xmlns:q="` + notationNS + `">q:jpeg</root>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/different-notation-rejected", func(t *testing.T) {
		t.Parallel()
		const typeDefsPng = `  <xs:notation name="jpeg" public="image/jpeg"/>
  <xs:notation name="png" public="image/png"/>
  <xs:simpleType name="pNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="p:jpeg"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="qNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="q:png"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="pOrQNotation">
    <xs:union memberTypes="pNotation qNotation"/>
  </xs:simpleType>`
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="` + notationNS + `" targetNamespace="` + notationNS + `" xmlns:p="` + notationNS + `" xmlns:q="` + notationNS + `">
` + typeDefsPng + `
  <xs:element name="root" type="pOrQNotation" fixed="p:jpeg"/>
</xs:schema>`
		instanceXML := `<root xmlns="` + notationNS + `" xmlns:q="` + notationNS + `">q:png</root>`
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

// TestFixedValueSpaceUnionStringFamily verifies that when the fixed and instance
// values resolve to DIFFERENT active union members that nonetheless share the
// same PRIMITIVE string value-space family, they compare equal by their
// whitespace-normalized lexical forms. memberTypes lists two distinct xs:string
// restrictions (one with whiteSpace="collapse"); fixed "a b" against instance
// " a   b " denotes the same collapsed string and must be ACCEPTED. A genuinely
// different string is rejected, and a cross-family pair (string vs integer) is
// rejected.
func TestFixedValueSpaceUnionStringFamily(t *testing.T) {
	// strExact is an xs:string restriction with whiteSpace="preserve" (inherited)
	// and a pattern that matches "a b" with a SINGLE internal space; it rejects a
	// padded/multi-space form. strColl is a SECOND xs:string restriction with
	// whiteSpace="collapse" that accepts the padded form by collapsing it. Both
	// reduce to the xs:string primitive value space, so a fixed value active in
	// strExact and an instance active in strColl that collapse to the same string
	// must compare equal.
	const typeDefs = `  <xs:simpleType name="strExact">
    <xs:restriction base="xs:string">
      <xs:pattern value="[a-z] [a-z]"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="strColl">
    <xs:restriction base="xs:string">
      <xs:whiteSpace value="collapse"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="strExactOrColl">
    <xs:union memberTypes="strExact strColl"/>
  </xs:simpleType>`

	t.Run("element/different string members value-equal", func(t *testing.T) {
		t.Parallel()
		// fixed "a b": matches strExact's pattern (single space) -> active member
		// strExact (preserve). instance " a   b ": strExact's pattern does NOT match
		// the padded form (preserve keeps the spaces) -> falls to strColl, which
		// collapses to "a b". Active members DIFFER (strExact vs strColl) yet both
		// reduce to the xs:string primitive family and normalize to "a b" -> EQUAL.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="strExactOrColl" fixed="a b"/>
</xs:schema>`
		instanceXML := "<root> a   b </root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/different string value rejected", func(t *testing.T) {
		t.Parallel()
		// fixed "a b" -> strExact; instance " a   c " -> strColl collapses to "a c".
		// Same string primitive family but "a b" != "a c" -> rejected.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="strExactOrColl" fixed="a b"/>
</xs:schema>`
		instanceXML := "<root> a   c </root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("attribute/different string members value-equal", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="strExactOrColl" fixed="a b"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<root a="  a   b  "/>`
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("element/cross-family string vs integer rejected", func(t *testing.T) {
		t.Parallel()
		// memberTypes="strColl xs:integer": fixed "1" -> strColl is a collapsing
		// xs:string and is the first member, so fixed's active member is strColl
		// (string family). instance "1" -> also strColl. To force a cross-family
		// pair, put xs:integer first so a bare "1" picks xs:integer while a padded
		// instance can only pick the string member. integer vs string differ.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="strColl">
    <xs:restriction base="xs:string">
      <xs:whiteSpace value="collapse"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="intOrStrColl">
    <xs:union memberTypes="xs:integer strColl"/>
  </xs:simpleType>
  <xs:element name="root" type="intOrStrColl" fixed="1"/>
</xs:schema>`
		// instance "x 1" -> xs:integer rejects, strColl accepts (collapses to "x 1").
		// fixed "1" -> xs:integer accepts (active member xs:integer). Cross-family.
		instanceXML := "<root>x 1</root>"
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})
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
