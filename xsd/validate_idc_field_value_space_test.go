package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDCFieldXSITypeActualType covers identity-constraint key comparison when an
// IDC field's type is contributed by an xsi:type ACTUAL type rather than the
// element's declared type. `item` is declared as a baseType with no attributes;
// the instance supplies xsi:type="itemType" which adds an inline xs:integer
// attribute `n`. The IDC field canonicalizer must consult the actual type
// determined during content validation, so `5` and `+5` denote the same value
// and collide for uniqueness.
func TestIDCFieldXSITypeActualType(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType"/>
  <xs:complexType name="itemType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:attribute name="n">
          <xs:simpleType>
            <xs:restriction base="xs:integer"/>
          </xs:simpleType>
        </xs:attribute>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="baseType" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@n"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name: "xsi:type integer 5 and +5 collide",
			instance: `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
				`<item xsi:type="itemType" n="5"/><item xsi:type="itemType" n="+5"/></root>`,
			valid: false,
		},
		{
			name: "xsi:type integer 5 and 6 distinct",
			instance: `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
				`<item xsi:type="itemType" n="5"/><item xsi:type="itemType" n="6"/></root>`,
			valid: true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldQNameValueSpace covers identity-constraint key comparison for an
// inline xs:QName field. QName equality is in value space ({uri, local}), so two
// lexical forms p:a and q:a with both prefixes bound to the SAME namespace URI
// must collide; bound to DIFFERENT URIs they must remain distinct.
func TestIDCFieldQNameValueSpace(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:QName" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name: "same uri different prefix collide",
			instance: `<root xmlns:p="urn:x" xmlns:q="urn:x">` +
				`<item>p:a</item><item>q:a</item></root>`,
			valid: false,
		},
		{
			name: "different uri distinct",
			instance: `<root xmlns:p="urn:x" xmlns:q="urn:y">` +
				`<item>p:a</item><item>q:a</item></root>`,
			valid: true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldListValueSpace covers identity-constraint key comparison for an
// inline xs:list with itemType="xs:integer". List equality compares item-by-item
// in the item type's value space, so `5 6` and `+5 06` must collide.
func TestIDCFieldListValueSpace(t *testing.T) {
	t.Parallel()

	// itemType is a complex type with simple content whose base is an inline
	// xs:list itemType="xs:integer", so the field text is a list of integers.
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intList">
    <xs:list itemType="xs:integer"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="intList" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name:     "list integers 5 6 and +5 06 collide",
			instance: `<root><item>5 6</item><item>+5 06</item></root>`,
			valid:    false,
		},
		{
			name:     "list integers 5 6 and 5 7 distinct",
			instance: `<root><item>5 6</item><item>5 7</item></root>`,
			valid:    true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldUnionActiveMember covers identity-constraint key comparison for a
// union field. The active member of a union value is the first member (in
// declaration order) the value validates against; the value is then canonicalized
// in THAT member's value space. So memberTypes="xs:string xs:integer" makes both
// `5` and `+5` active member xs:string (lexical-only), keeping them DISTINCT,
// whereas memberTypes="xs:integer xs:string" makes both active member xs:integer,
// collapsing `5` and `+5` to the same value (duplicate).
func TestIDCFieldUnionActiveMember(t *testing.T) {
	t.Parallel()

	schemaFor := func(members string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="u">
    <xs:union memberTypes="` + members + `"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="u" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`
	}

	cases := []struct {
		name     string
		members  string
		instance string
		valid    bool
	}{
		{
			// string precedes integer: both 5 and +5 are valid xs:string, so the
			// active member is xs:string and they remain lexically distinct.
			name:     "string before integer keeps 5 and +5 distinct",
			members:  "xs:string xs:integer",
			instance: `<root><item>5</item><item>+5</item></root>`,
			valid:    true,
		},
		{
			// integer precedes string: both are valid xs:integer, so the active
			// member is xs:integer and 5 == +5 in value space (duplicate).
			name:     "integer before string collapses 5 and +5",
			members:  "xs:integer xs:string",
			instance: `<root><item>5</item><item>+5</item></root>`,
			valid:    false,
		},
		{
			// active member xs:string, lexically distinct strings stay distinct.
			name:     "string before integer distinct values",
			members:  "xs:string xs:integer",
			instance: `<root><item>5</item><item>6</item></root>`,
			valid:    true,
		},
		{
			// non-integer text falls through to the xs:string member; distinct.
			name:     "integer before string non-numeric distinct",
			members:  "xs:integer xs:string",
			instance: `<root><item>a</item><item>b</item></root>`,
			valid:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := compileValidator(t, schemaFor(tc.members))
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldUnionActiveMemberFacets covers active-member selection for a union
// whose first member's LEXICAL space accepts the value but whose FACETS reject
// it. smallInt restricts xs:integer with maxInclusive="0", so `5`/`+5` are
// lexically integers but fail the facet; the value must fall through to the
// xs:string member (active member xs:string, lexical-only) and stay DISTINCT.
// The previous lexical-only active-member selection wrongly chose smallInt and
// collapsed `5`/`+5` to one value, reporting a spurious duplicate.
func TestIDCFieldUnionActiveMemberFacets(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="smallInt">
    <xs:restriction base="xs:integer">
      <xs:maxInclusive value="0"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="smallInt xs:string"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="u" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:key>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			// smallInt rejects 5 (maxInclusive=0) so both fall through to xs:string;
			// lexically distinct, no duplicate.
			name:     "facet-rejected ints fall through to string and stay distinct",
			instance: `<root><item>5</item><item>+5</item></root>`,
			valid:    true,
		},
		{
			// both accepted by smallInt (-1, -01 == -1 in integer value space): collide.
			name:     "facet-accepted ints collapse in integer value space",
			instance: `<root><item>-1</item><item>-01</item></root>`,
			valid:    false,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldRestrictedListVariety covers a field whose type is a RESTRICTION
// over an inline xs:list itemType="xs:integer". The derived restriction keeps
// Variety==Atomic, so dispatching on td.Variety alone would mis-route it to the
// atomic path and compare list text lexically. Dispatching on resolveVariety(td)
// routes it to the list path, so `5 6` and `+5 06` collide item-by-item.
func TestIDCFieldRestrictedListVariety(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="restrictedIntList">
    <xs:restriction>
      <xs:simpleType>
        <xs:list itemType="xs:integer"/>
      </xs:simpleType>
      <xs:maxLength value="5"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="restrictedIntList" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name:     "restricted-list integers 5 6 and +5 06 collide",
			instance: `<root><item>5 6</item><item>+5 06</item></root>`,
			valid:    false,
		},
		{
			name:     "restricted-list integers 5 6 and 5 7 distinct",
			instance: `<root><item>5 6</item><item>5 7</item></root>`,
			valid:    true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldWhiteSpaceCollapse covers a field whose type is a RESTRICTION of
// xs:string with whiteSpace="collapse". The canonical key must apply the
// resolved whiteSpace facet, so `a b` and `a  b` collapse to the same value and
// collide. Without facet-aware normalization the two stay lexically distinct.
func TestIDCFieldWhiteSpaceCollapse(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="collapsed">
    <xs:restriction base="xs:string">
      <xs:whiteSpace value="collapse"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="collapsed" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name:     "collapse makes a b and a  b collide",
			instance: "<root><item>a b</item><item>a  b</item></root>",
			valid:    false,
		},
		{
			name:     "collapse keeps a b and a c distinct",
			instance: "<root><item>a b</item><item>a  c</item></root>",
			valid:    true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldUnionListMember covers active-member selection and canonicalization
// for a union one of whose members is an xs:list. memberTypes="intList xs:string"
// (intList = xs:list itemType="xs:integer"): a whitespace-separated integer list
// validates against intList, so its active member is the LIST and the key must be
// canonicalized item-by-item in xs:integer value space — `5 6` and `+5 06` collide.
// The previous code skipped list members (no builtin atomic base) and, even when a
// member was selected, always canonicalized atomically, so it compared the list
// text lexically and missed the collision.
func TestIDCFieldUnionListMember(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intList">
    <xs:list itemType="xs:integer"/>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="intList xs:string"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="u" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			// both validate as intList; canonicalized item-by-item, 5 6 == +5 06.
			name:     "list-member integers 5 6 and +5 06 collide",
			instance: `<root><item>5 6</item><item>+5 06</item></root>`,
			valid:    false,
		},
		{
			name:     "list-member integers 5 6 and 5 7 distinct",
			instance: `<root><item>5 6</item><item>5 7</item></root>`,
			valid:    true,
		},
		{
			// non-numeric text fails the list member, falls through to xs:string;
			// active member xs:string, lexically distinct.
			name:     "non-list text falls through to string and stays distinct",
			instance: `<root><item>a b</item><item>c d</item></root>`,
			valid:    true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldNestedUnionFacetFallthrough covers active-member selection for a
// union whose first member is itself a NESTED UNION whose wrapper restriction
// rejects the value by FACET. inner = restriction of a union over xs:integer with
// a wrapper xs:pattern that admits only negative-signed integer lexical forms
// (`-[0-9]+`); outer u = union memberTypes="inner xs:string". `5`/`+5` are lexical
// integers but DON'T match the inner wrapper pattern, so the inner member rejects
// them and they fall through to xs:string (active member xs:string, lexical-only)
// and stay DISTINCT. Pre-flattening the nested union to its xs:integer leaf would
// drop the wrapper pattern, wrongly accept `5`/`+5` as the integer leaf, and
// collapse them in integer value space, reporting a spurious duplicate.
//
// xs:pattern (and xs:enumeration) are the only constraining facets applicable to
// a union variety; range facets like xs:maxInclusive are rejected at compile time
// (see check_facets.go checkListUnionFacetApplicability), so the wrapper facet
// here must be a pattern to keep the schema valid while still exercising the
// nested-union value-space fallthrough path.
func TestIDCFieldNestedUnionFacetFallthrough(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="innerLeaf">
    <xs:union memberTypes="xs:integer"/>
  </xs:simpleType>
  <xs:simpleType name="inner">
    <xs:restriction base="innerLeaf">
      <xs:pattern value="-[0-9]+"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="inner xs:string"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="u" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:key>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			// inner wrapper pattern rejects 5/+5 (no leading '-'), so they fall
			// through to xs:string; lexically distinct.
			name:     "nested-union facet-rejected ints fall through to string",
			instance: `<root><item>5</item><item>+5</item></root>`,
			valid:    true,
		},
		{
			// inner accepts -1/-01 (both match the pattern); the active member is the
			// nested union, canonicalized in xs:integer value space where -1 == -01,
			// so they collide.
			name:     "nested-union facet-accepted ints collapse",
			instance: `<root><item>-1</item><item>-01</item></root>`,
			valid:    false,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldUnionQNameMember covers a union member that is an xs:QName-derived
// enumeration whose facet values are PREFIXED names. Active-member validation must
// thread the field node's in-scope namespaces so the member's QName-valued
// enumeration facet resolves prefixes against the same bindings as the instance
// value. memberTypes="kind xs:string" where kind enumerates p:a/p:b (p bound in
// the schema). Once kind is the active member, the QName value is canonicalized to
// its {uri,local} key, so two prefixes bound to the same URI collide.
func TestIDCFieldUnionQNameMember(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:x">
  <xs:simpleType name="kind">
    <xs:restriction base="xs:QName">
      <xs:enumeration value="p:a"/>
      <xs:enumeration value="p:b"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="kind xs:string"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="u" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			// p:a and q:a both bound to urn:x: QName member active, same {uri,local}.
			name: "qname union member same uri different prefix collide",
			instance: `<root xmlns:p="urn:x" xmlns:q="urn:x">` +
				`<item>p:a</item><item>q:a</item></root>`,
			valid: false,
		},
		{
			// p:a and p:b are distinct enumerated QNames.
			name: "qname union member distinct enumerated names",
			instance: `<root xmlns:p="urn:x">` +
				`<item>p:a</item><item>p:b</item></root>`,
			valid: true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldAnyTypeDescendant covers IDC field canonicalization for a field
// reached through an xs:anyType (mixed, no content model) ancestor. The root
// `doc` is xs:anyType, so pass-1 content validation has no content model to walk
// and historically returned immediately — never annotating the descendant `root`
// or its `item` children with their ACTUAL types. Pass-2 IDC evaluation then fell
// back to declared types and missed the xsi:type="itemType" inline xs:integer @n,
// comparing `5` and `+5` lexically and reporting them UNIQUE. With lax annotation
// of anyType/mixed descendants the actual type is recorded, so `5` and `+5`
// canonicalize equal in xs:integer value space and COLLIDE.
func TestIDCFieldAnyTypeDescendant(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType"/>
  <xs:complexType name="itemType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:attribute name="n">
          <xs:simpleType>
            <xs:restriction base="xs:integer"/>
          </xs:simpleType>
        </xs:attribute>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="baseType" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@n"/>
    </xs:unique>
  </xs:element>
  <xs:element name="doc" type="xs:anyType"/>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name: "anyType-nested xsi:type integer 5 and +5 collide",
			instance: `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
				`<root><item xsi:type="itemType" n="5"/>` +
				`<item xsi:type="itemType" n="+5"/></root></doc>`,
			valid: false,
		},
		{
			name: "anyType-nested xsi:type integer 5 and 6 distinct",
			instance: `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
				`<root><item xsi:type="itemType" n="5"/>` +
				`<item xsi:type="itemType" n="6"/></root></doc>`,
			valid: true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldLaxWildcardDescendant covers IDC field canonicalization for a field
// reached through an `xs:any processContents="lax"` wildcard-matched wrapper that
// has NO global element declaration. The wrapper `unknown` matches the lax
// wildcard but is not schema-assessed, so historically matchWildcardParticle
// stopped at it (continue) and never recursed into its subtree — leaving the
// nested global IDC host `root` and its `item` children with no ACTUAL type
// recorded. Pass-2 IDC evaluation then fell back to declared/raw types and missed
// the xsi:type="itemType" inline xs:integer @n, comparing `5` and `+5` lexically
// and reporting them UNIQUE. With lax recursion into the wildcard-matched
// subtree the actual type is recorded, so `5` and `+5` canonicalize equal in
// xs:integer value space and COLLIDE.
func TestIDCFieldLaxWildcardDescendant(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType"/>
  <xs:complexType name="itemType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:attribute name="n">
          <xs:simpleType>
            <xs:restriction base="xs:integer"/>
          </xs:simpleType>
        </xs:attribute>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="baseType" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@n"/>
    </xs:unique>
  </xs:element>
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:any processContents="lax" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name: "lax-wildcard-nested xsi:type integer 5 and +5 collide",
			instance: `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
				`<unknown><root><item xsi:type="itemType" n="5"/>` +
				`<item xsi:type="itemType" n="+5"/></root></unknown></doc>`,
			valid: false,
		},
		{
			name: "lax-wildcard-nested xsi:type integer 5 and 6 distinct",
			instance: `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
				`<unknown><root><item xsi:type="itemType" n="5"/>` +
				`<item xsi:type="itemType" n="6"/></root></unknown></doc>`,
			valid: true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldSkipWildcardDescendant covers IDC field canonicalization for a
// field reached through an `xs:any processContents="skip"` wildcard-matched
// wrapper that has NO global element declaration. Skipped content is NOT
// schema-assessed, so matchWildcardParticle historically returned at the matched
// wrapper without walking its subtree — leaving the nested global IDC host `root`
// and its `item` children with no ACTUAL type recorded. Pass-2 IDC evaluation
// then fell back to declared/raw types and missed the xsi:type="itemType" inline
// xs:integer @n, comparing `5` and `+5` lexically and reporting them UNIQUE. With
// an annotation-only traversal of the skipped subtree (no validation, no errors)
// the actual type is recorded, so `5` and `+5` canonicalize equal in xs:integer
// value space and COLLIDE.
func TestIDCFieldSkipWildcardDescendant(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType"/>
  <xs:complexType name="itemType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:attribute name="n">
          <xs:simpleType>
            <xs:restriction base="xs:integer"/>
          </xs:simpleType>
        </xs:attribute>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="baseType" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@n"/>
    </xs:unique>
  </xs:element>
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:any processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name: "skip-wildcard-nested xsi:type integer 5 and +5 collide",
			instance: `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
				`<unknown><root><item xsi:type="itemType" n="5"/>` +
				`<item xsi:type="itemType" n="+5"/></root></unknown></doc>`,
			valid: false,
		},
		{
			name: "skip-wildcard-nested xsi:type integer 5 and 6 distinct",
			instance: `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
				`<unknown><root><item xsi:type="itemType" n="5"/>` +
				`<item xsi:type="itemType" n="6"/></root></unknown></doc>`,
			valid: true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldSkipWildcardSelectedSelf covers IDC field canonicalization when a
// PARENT IDC SELECTS the `xs:any processContents="skip"` wildcard-matched element
// ITSELF (not a descendant of it). Skipped content is NOT schema-assessed, so the
// matched element carries no ACTUAL type from content validation. The annotation
// walk over the skipped subtree historically annotated only the matched element's
// CHILDREN, never the matched element itself — so the parent IDC's field over the
// matched element's xsi:type-introduced inline xs:integer @n fell back to the raw
// lexical value and compared `5` and `+5` as DISTINCT. With the matched element
// itself annotated with its xsi:type ACTUAL type, `5` and `+5` canonicalize equal
// in xs:integer value space and COLLIDE.
func TestIDCFieldSkipWildcardSelectedSelf(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType"/>
  <xs:complexType name="itemType">
    <xs:complexContent>
      <xs:extension base="baseType">
        <xs:attribute name="n">
          <xs:simpleType>
            <xs:restriction base="xs:integer"/>
          </xs:simpleType>
        </xs:attribute>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:any processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@n"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name: "parent-IDC-selected skip element xsi:type integer 5 and +5 collide",
			instance: `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
				`<item xsi:type="itemType" n="5"/>` +
				`<item xsi:type="itemType" n="+5"/></doc>`,
			valid: false,
		},
		{
			name: "parent-IDC-selected skip element xsi:type integer 5 and 6 distinct",
			instance: `<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
				`<item xsi:type="itemType" n="5"/>` +
				`<item xsi:type="itemType" n="6"/></doc>`,
			valid: true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldAnyTypeSubstitutionGroupMember covers IDC field canonicalization for
// a NO-TYPE SUBSTITUTION-GROUP member element that is a DIRECT child of an
// xs:anyType (open lax) element. `item` is a substitution-group member of head
// `itemHead`, declares NO type of its own, and so inherits `itemHead`'s type (which
// adds an inline xs:integer @n). The IDC on the xs:anyType `doc` selects those
// `item` children. Because `doc` is xs:anyType there is no content model, so its
// children are annotated by the lax anyType walk, which historically consulted
// `edecl.Type` DIRECTLY — nil for a no-type member — so the member was not
// annotated with its inherited head type and pass-2 IDC fell back to the raw
// lexical @n, comparing `5` and `+5` as DISTINCT. Using the EFFECTIVE declared
// type (the inherited head type) records the actual type, so `5` and `+5`
// canonicalize equal in xs:integer value space and COLLIDE.
func TestIDCFieldAnyTypeSubstitutionGroupMember(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="itemType">
    <xs:attribute name="n">
      <xs:simpleType>
        <xs:restriction base="xs:integer"/>
      </xs:simpleType>
    </xs:attribute>
  </xs:complexType>
  <xs:element name="itemHead" type="itemType"/>
  <xs:element name="item" substitutionGroup="itemHead"/>
  <xs:element name="doc" type="xs:anyType">
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@n"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name: "anyType substitution-group member integer 5 and +5 collide",
			instance: `<doc>` +
				`<item n="5"/><item n="+5"/></doc>`,
			valid: false,
		},
		{
			name: "anyType substitution-group member integer 5 and 6 distinct",
			instance: `<doc>` +
				`<item n="5"/><item n="6"/></doc>`,
			valid: true,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCFieldNotationDefaultNamespace covers identity-constraint key
// canonicalization for an xs:NOTATION field. An UNPREFIXED xs:NOTATION value picks
// up the in-scope DEFAULT namespace (like the schema's declared-notation lookup and
// the facet comparison paths), so a prefixed `p:jpeg` and an unprefixed `jpeg`
// under a default namespace urn:p both denote {urn:p}jpeg and COLLIDE for
// xs:unique/xs:key. A distinct notation (png) stays distinct. Runs for both the
// xs:unique and xs:key constraint kinds so both route through canonicalAtomicKey.
func TestIDCFieldNotationDefaultNamespace(t *testing.T) {
	t.Parallel()

	// targetNamespace and default namespace are both urn:p, so the unprefixed
	// enumeration literals jpeg/png name {urn:p}jpeg/{urn:p}png (the declared
	// notations), and elementFormDefault="qualified" puts the local `item` elements
	// in urn:p (the selector p:item resolves via the schema's p binding).
	schemaFor := func(kind string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p" xmlns="urn:p" targetNamespace="urn:p" elementFormDefault="qualified">
  <xs:notation name="jpeg" public="image/jpeg"/>
  <xs:notation name="png" public="image/png"/>
  <xs:simpleType name="imageKind">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="jpeg"/>
      <xs:enumeration value="png"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="imageKind" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:` + kind + ` name="itemKey">
      <xs:selector xpath="p:item"/>
      <xs:field xpath="."/>
    </xs:` + kind + `>
  </xs:element>
</xs:schema>`
	}

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			// p:jpeg = {urn:p}jpeg; unprefixed jpeg picks up default ns urn:p =
			// {urn:p}jpeg. Same canonical value → duplicate.
			name:     "prefixed and unprefixed notation same default ns collide",
			instance: `<root xmlns="urn:p" xmlns:p="urn:p"><item>p:jpeg</item><item>jpeg</item></root>`,
			valid:    false,
		},
		{
			// {urn:p}jpeg vs {urn:p}png — distinct notations.
			name:     "distinct notations stay distinct",
			instance: `<root xmlns="urn:p" xmlns:p="urn:p"><item>p:jpeg</item><item>png</item></root>`,
			valid:    true,
		},
	}

	for _, kind := range []string{"unique", "key"} {
		v := compileValidator(t, schemaFor(kind))
		for _, tc := range cases {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
				require.NoError(t, err)

				var errs string
				err = validateWithOutput(t, v, doc, &errs)
				if tc.valid {
					require.NoError(t, err, "expected valid, got errors: %s", errs)
					return
				}
				require.Error(t, err, "expected validation error")
			})
		}
	}
}

// TestIDCFieldQNameNoDefaultNamespace verifies the xs:QName IDC path is UNCHANGED
// by the NOTATION default-namespace resolution: an unprefixed xs:QName VALUE
// resolves to NO namespace (value-space semantics), NOT the in-scope default. So
// with a default namespace urn:x in scope, `p:a` = {urn:x}a and unprefixed `a` =
// {}a are DISTINCT — the contrast with the NOTATION case above, which collides.
func TestIDCFieldQNameNoDefaultNamespace(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:x" targetNamespace="urn:x" elementFormDefault="qualified">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:QName" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="p:item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			// p:a = {urn:x}a; unprefixed a = {}a (QName value ignores the default ns).
			// Distinct, so no duplicate.
			name:     "prefixed and unprefixed qname distinct under default ns",
			instance: `<root xmlns="urn:x" xmlns:p="urn:x"><item>p:a</item><item>a</item></root>`,
			valid:    true,
		},
		{
			// Two prefixes bound to the same uri: {urn:x}a both → duplicate.
			name:     "same uri different prefix qname collide",
			instance: `<root xmlns="urn:x" xmlns:p="urn:x" xmlns:q="urn:x"><item>p:a</item><item>q:a</item></root>`,
			valid:    false,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}

func compileValidator(t *testing.T, src string) xsd.Validator {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)
	return xsd.NewValidator(schema)
}
