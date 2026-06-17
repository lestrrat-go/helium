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
}

// TestFixedValueSpaceListOfUnion verifies that a fixed value of an xs:list whose
// itemType is a union dispatches each item through the union's member value
// spaces, so a value-equal item under any member type (e.g. integer "01" == "1")
// satisfies the constraint while a value-distinct item is rejected.
func TestFixedValueSpaceListOfUnion(t *testing.T) {
	const typeDefs = `  <xs:simpleType name="intOrBool">
    <xs:union memberTypes="xs:integer xs:boolean"/>
  </xs:simpleType>
  <xs:simpleType name="unionList">
    <xs:list itemType="intOrBool"/>
  </xs:simpleType>`

	t.Run("element/value-equal items", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="unionList" fixed="1 true"/>
</xs:schema>`
		// "01" is integer-equal to "1"; "1" is boolean-equal to "true".
		instanceXML := "<root>01 1</root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
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

	t.Run("attribute/value-equal items", func(t *testing.T) {
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
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})
}

// TestFixedValueSpaceUnion verifies that a fixed value of an xs:union dispatches
// the raw (un-normalized) instance and fixed values to each member, letting each
// member apply its *own* whiteSpace facet. A union containing xs:string must
// preserve significant trailing whitespace under the string member: fixed="abc "
// must NOT accept instance "abc" even though a sibling xs:integer member would
// collapse whitespace. A value-equal instance under any member (e.g. the integer
// member) must still be accepted.
func TestFixedValueSpaceUnion(t *testing.T) {
	const typeDefs = `  <xs:simpleType name="strOrInt">
    <xs:union memberTypes="xs:string xs:integer"/>
  </xs:simpleType>`

	t.Run("element/string member trailing space significant", func(t *testing.T) {
		t.Parallel()
		// fixed has a trailing space; the xs:string member preserves it, so the
		// instance "abc" (no trailing space) is value-distinct. The xs:integer
		// member rejects the non-integer text. The constraint must be rejected.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="strOrInt" fixed="` + abcLiteral + ` "/>
</xs:schema>`
		instanceXML := `<root>` + abcLiteral + `</root>`
		runFixedValueCase(t, schemaXML, instanceXML, true)
	})

	t.Run("element/integer member value-equal", func(t *testing.T) {
		t.Parallel()
		// The xs:integer member treats "01" as value-equal to "1".
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + typeDefs + `
  <xs:element name="root" type="strOrInt" fixed="1"/>
</xs:schema>`
		instanceXML := "<root>01</root>"
		runFixedValueCase(t, schemaXML, instanceXML, false)
	})

	t.Run("attribute/string member trailing space significant", func(t *testing.T) {
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

	t.Run("attribute/integer member value-equal", func(t *testing.T) {
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
