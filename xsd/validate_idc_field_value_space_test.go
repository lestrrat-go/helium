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

func compileValidator(t *testing.T, src string) xsd.Validator {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)
	return xsd.NewValidator(schema)
}
