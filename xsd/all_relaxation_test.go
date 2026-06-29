package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileAll11 compiles a schema in XSD 1.1 mode and returns the compile error
// (nil when the schema is valid).
func compileAll11(t *testing.T, schemaXML string) (*xsd.Schema, error) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
}

// validateAll11 compiles the schema (must be valid) and validates the instance,
// returning the validation error (nil when the instance is valid).
func validateAll11(t *testing.T, schemaXML, instanceXML string) error {
	t.Helper()
	schema, cerr := compileAll11(t, schemaXML)
	require.NoError(t, cerr, "schema must compile")
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)
	return xsd.NewValidator(schema).Validate(t.Context(), idoc)
}

// TestAll10WildcardOnlyRegression guards the XSD 1.0 xs:all matcher against the
// regression where the flat-member rewrite dropped wildcard particles from the
// 1.0 required-member bookkeeping. A wildcard-only xs:all with minOccurs=1 (the
// parser tolerates a wildcard inside an xs:all even in 1.0) must reject empty
// content as a missing required member — exactly as the pre-rewrite matcher did,
// and unlike 1.1 (which actually matches the wildcard). The 1.0 path never
// matches a wildcard member, so any child is "not expected".
func TestAll10WildcardOnlyRegression(t *testing.T) {
	t.Parallel()

	compile10 := func(t *testing.T, schemaXML string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		return xsd.NewCompiler().Compile(t.Context(), doc)
	}
	validate10 := func(t *testing.T, schema *xsd.Schema, instanceXML string) error {
		t.Helper()
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:all minOccurs="1"><xs:any/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`

	sch, cerr := compile10(t, schema)
	require.NoError(t, cerr, "1.0 schema with a wildcard-only xs:all should compile")

	// Empty content: the required wildcard member is missing — must be rejected.
	require.Error(t, validate10(t, sch, `<doc/>`),
		"1.0 wildcard-only all minOccurs=1 must reject empty content (regression guard)")

	// A child is never matched by a wildcard in 1.0, so it is unexpected.
	require.Error(t, validate10(t, sch, `<doc><e/></doc>`))
}

// TestAll11MemberMaxOccurs verifies the XSD 1.1 relaxation that an element member
// of an xs:all may carry minOccurs/maxOccurs > 1 (cos-all-limited relaxed), and
// that order-independent per-member occurrence counting enforces those bounds.
func TestAll11MemberMaxOccurs(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType>
    <xs:all>
      <xs:element name="a" minOccurs="0" maxOccurs="5"/>
      <xs:element name="b" minOccurs="1" maxOccurs="5"/>
      <xs:element name="c" minOccurs="2" maxOccurs="unbounded"/>
      <xs:element name="d" minOccurs="1" maxOccurs="1"/>
    </xs:all>
  </xs:complexType></xs:element>
</xs:schema>`

	t.Run("schema compiles in 1.1", func(t *testing.T) {
		t.Parallel()
		_, err := compileAll11(t, schema)
		require.NoError(t, err)
	})

	// The SAME schema must be REJECTED in XSD 1.0 (maxOccurs>1 inside xs:all).
	t.Run("schema rejected in 1.0", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		_, cerr := xsd.NewCompiler().Compile(t.Context(), doc)
		require.Error(t, cerr)
	})

	t.Run("valid instance, members in range, any order", func(t *testing.T) {
		t.Parallel()
		// a*4, b*2, c*3, d*1 — all within range, interleaved/unordered.
		const inst = `<doc><a/><b/><d/><c/><a/><c/><c/><a/><a/><b/></doc>`
		require.NoError(t, validateAll11(t, schema, inst))
	})

	t.Run("invalid: too few c", func(t *testing.T) {
		t.Parallel()
		// only one c (min is 2).
		const inst = `<doc><a/><b/><d/><c/><a/></doc>`
		require.Error(t, validateAll11(t, schema, inst))
	})

	t.Run("invalid: too many b", func(t *testing.T) {
		t.Parallel()
		// six b (max is 5).
		const inst = `<doc><b/><b/><b/><b/><b/><b/><c/><c/><d/></doc>`
		require.Error(t, validateAll11(t, schema, inst))
	})
}

// TestAll11Wildcard verifies that an xs:all may contain element wildcards in XSD
// 1.1, and that wildcard members are matched with their own occurrence bounds and
// namespace constraints (weak-wildcard precedence: declared elements win).
func TestAll11Wildcard(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType mixed="true">
    <xs:all>
      <xs:any namespace="http://a.ns/" minOccurs="2" maxOccurs="5" processContents="skip"/>
      <xs:any namespace="http://b.ns/" minOccurs="0" maxOccurs="unbounded" processContents="skip"/>
    </xs:all>
  </xs:complexType></xs:element>
</xs:schema>`

	t.Run("schema compiles in 1.1", func(t *testing.T) {
		t.Parallel()
		_, err := compileAll11(t, schema)
		require.NoError(t, err)
	})

	t.Run("valid instance", func(t *testing.T) {
		t.Parallel()
		const inst = `<doc xmlns:a="http://a.ns/" xmlns:b="http://b.ns/"><a:x/><b:y/><a:z/></doc>`
		require.NoError(t, validateAll11(t, schema, inst))
	})

	t.Run("invalid: too few a.ns wildcard matches", func(t *testing.T) {
		t.Parallel()
		// only one element in a.ns (min is 2).
		const inst = `<doc xmlns:a="http://a.ns/" xmlns:b="http://b.ns/"><a:x/><b:y/></doc>`
		require.Error(t, validateAll11(t, schema, inst))
	})

	t.Run("invalid: element in disallowed namespace", func(t *testing.T) {
		t.Parallel()
		const inst = `<doc xmlns:a="http://a.ns/" xmlns:c="http://c.ns/"><a:x/><a:y/><c:z/></doc>`
		require.Error(t, validateAll11(t, schema, inst))
	})
}

// TestAll11NestedGroupRef verifies that an xs:group reference resolving to an
// all group may be nested directly inside another xs:all in XSD 1.1 (flattened
// into the parent), while a referenced sequence/choice group, or an occurrence
// other than 1/1, is rejected.
func TestAll11NestedGroupRef(t *testing.T) {
	t.Parallel()

	t.Run("nested all-group ref valid and flattened", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType>
    <xs:all>
      <xs:element name="a" minOccurs="0" maxOccurs="5"/>
      <xs:group ref="g"/>
    </xs:all>
  </xs:complexType></xs:element>
  <xs:group name="g"><xs:all>
    <xs:element name="b" minOccurs="1" maxOccurs="5"/>
    <xs:element name="c" minOccurs="2" maxOccurs="unbounded"/>
    <xs:element name="d" minOccurs="1" maxOccurs="1"/>
  </xs:all></xs:group>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.NoError(t, cerr)
		// flattened members a,b,c,d matched order-independently.
		require.NoError(t, validateAll11(t, schema, `<doc><b/><c/><d/><c/><a/></doc>`))
		require.Error(t, validateAll11(t, schema, `<doc><b/><c/><d/></doc>`)) // only one c
	})

	t.Run("nested sequence-group ref rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType>
    <xs:all>
      <xs:element name="a"/>
      <xs:group ref="g"/>
    </xs:all>
  </xs:complexType></xs:element>
  <xs:group name="g"><xs:sequence><xs:element name="b"/></xs:sequence></xs:group>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})

	t.Run("nested all-group ref with occurrence != 1 rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType>
    <xs:all>
      <xs:element name="a"/>
      <xs:group ref="g" minOccurs="0" maxOccurs="1"/>
    </xs:all>
  </xs:complexType></xs:element>
  <xs:group name="g"><xs:all><xs:element name="b"/></xs:all></xs:group>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})
}

// TestAll11RestrictionSubsumption verifies XSD 1.1 occurrence-counting
// subsumption when restricting a base xs:all: derived element occurrence ranges
// must be subsets of the base member ranges, and several derived particles (e.g.
// substitution-group members) may collectively restrict one base member with
// their summed occurrence within the base range.
func TestAll11RestrictionSubsumption(t *testing.T) {
	t.Parallel()

	t.Run("range subset valid (all to all)", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element name="a" minOccurs="0" maxOccurs="5"/>
    <xs:element name="b" minOccurs="1" maxOccurs="5"/>
    <xs:element name="c" minOccurs="2" maxOccurs="unbounded"/>
    <xs:element name="d" minOccurs="1" maxOccurs="1"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element name="d" minOccurs="1" maxOccurs="1"/>
    <xs:element name="b" minOccurs="3" maxOccurs="4"/>
    <xs:element name="c" minOccurs="2" maxOccurs="4"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.NoError(t, cerr)
	})

	t.Run("derived range exceeds base max invalid", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element name="b" minOccurs="1" maxOccurs="5"/>
    <xs:element name="d" minOccurs="1" maxOccurs="1"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element name="d" minOccurs="1" maxOccurs="5"/>
    <xs:element name="b" minOccurs="2" maxOccurs="4"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr) // derived d (1..5) exceeds base d (1..1)
	})

	t.Run("substitution-group members sum into base member valid", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element ref="a" minOccurs="10" maxOccurs="20"/>
    <xs:element name="b" minOccurs="0" maxOccurs="5"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element ref="A1" minOccurs="6" maxOccurs="8"/>
    <xs:element ref="A2" minOccurs="6" maxOccurs="8"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="a"/>
  <xs:element name="A1" substitutionGroup="a"/>
  <xs:element name="A2" substitutionGroup="a"/>
</xs:schema>`
		// combined A1+A2 = 12..16, within base a 10..20.
		_, cerr := compileAll11(t, schema)
		require.NoError(t, cerr)
	})

	t.Run("substitution-group members sum below base min invalid", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element ref="a" minOccurs="10" maxOccurs="20"/>
    <xs:element name="b" minOccurs="0" maxOccurs="5"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element ref="A1" minOccurs="3" maxOccurs="8"/>
    <xs:element ref="A2" minOccurs="3" maxOccurs="8"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="a"/>
  <xs:element name="A1" substitutionGroup="a"/>
  <xs:element name="A2" substitutionGroup="a"/>
</xs:schema>`
		// combined A1+A2 = 6..16; min 6 < base a min 10.
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})

	// A derived CHOICE restricting a base all: alternative branches that each map
	// to the SAME base member must correlate on the member (min/max over branches),
	// not be summed as independent names. choice(A1{2,2}, A2{2,2}) emits exactly 2
	// of base member a per selection, valid against base a{2,2}.
	t.Run("choice branches with subst-group members correlate on base member", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element ref="a" minOccurs="2" maxOccurs="2"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:choice>
    <xs:element ref="A1" minOccurs="2" maxOccurs="2"/>
    <xs:element ref="A2" minOccurs="2" maxOccurs="2"/>
  </xs:choice></xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="a"/>
  <xs:element name="A1" substitutionGroup="a"/>
  <xs:element name="A2" substitutionGroup="a"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.NoError(t, cerr) // each branch emits exactly 2 of a, within a{2,2}
	})

	// A choice branch that exceeds the base member range is still rejected (max
	// over branches): branch a{1,8} over base a{0,5}.
	t.Run("choice branch exceeding base member max invalid", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element name="a" minOccurs="0" maxOccurs="5"/>
    <xs:element name="b" minOccurs="1" maxOccurs="5"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:choice>
    <xs:sequence><xs:element name="b" minOccurs="3" maxOccurs="4"/></xs:sequence>
    <xs:sequence><xs:element name="a" minOccurs="1" maxOccurs="8"/><xs:element name="b" minOccurs="3" maxOccurs="4"/></xs:sequence>
  </xs:choice></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr) // second branch emits a up to 8 > base a max 5
	})

	// A prohibited (maxOccurs=0) wildcard inside the derived all emits nothing and
	// must NOT abort counting subsumption of a no-wildcard base all.
	t.Run("prohibited wildcard in derived all emits nothing", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element name="a" minOccurs="0" maxOccurs="5"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element name="a" minOccurs="1" maxOccurs="1"/>
    <xs:any minOccurs="0" maxOccurs="0"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.NoError(t, cerr)
	})
}

// TestAll11Extension verifies that an xs:all may be extended by another xs:all in
// XSD 1.1 (the members merge into a single all group), while a cross-compositor
// extension (sequence extending all, or all extending sequence) and a minOccurs
// mismatch remain invalid.
func TestAll11Extension(t *testing.T) {
	t.Parallel()

	t.Run("all extends all valid and merges members", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">
  <xs:complexType name="b"><xs:all>
    <xs:element name="a" minOccurs="0" maxOccurs="5"/>
    <xs:element name="b" minOccurs="1" maxOccurs="5"/>
  </xs:all></xs:complexType>
  <xs:complexType name="e"><xs:complexContent><xs:extension base="b"><xs:all>
    <xs:element name="e" minOccurs="1" maxOccurs="1"/>
    <xs:element name="f" minOccurs="3" maxOccurs="4"/>
  </xs:all></xs:extension></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="e"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.NoError(t, cerr)
		// merged members a,b,e,f matched order-independently.
		require.NoError(t, validateAll11(t, schema, `<doc><b/><e/><f/><f/><f/><a/></doc>`))
		require.Error(t, validateAll11(t, schema, `<doc><b/><e/><f/></doc>`)) // f needs >=3
	})

	t.Run("sequence extends all invalid", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b" mixed="true"><xs:all><xs:element name="a"/></xs:all></xs:complexType>
  <xs:complexType name="e"><xs:complexContent mixed="true"><xs:extension base="b"><xs:sequence>
    <xs:element name="d" minOccurs="0" maxOccurs="2"/>
  </xs:sequence></xs:extension></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="e"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})

	t.Run("all extends all minOccurs mismatch invalid", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all><xs:element name="child1"/></xs:all></xs:complexType>
  <xs:complexType name="e"><xs:complexContent><xs:extension base="b"><xs:all minOccurs="0">
    <xs:element name="child2"/>
  </xs:all></xs:extension></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="e"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})
}

// TestAll11UPA verifies the Unique Particle Attribution check over xs:all members
// in XSD 1.1: an xs:all with two members where one element is in the substitution
// group of the other is ambiguous (a member-named child could attribute to either
// particle) and is rejected. Two same-named members are likewise rejected.
func TestAll11UPA(t *testing.T) {
	t.Parallel()

	t.Run("member in substitution group of another member", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType><xs:all>
    <xs:element ref="o"/>
    <xs:element name="x" type="xs:boolean"/>
    <xs:element ref="p"/>
  </xs:all></xs:complexType></xs:element>
  <xs:element name="o" type="xs:integer"/>
  <xs:element name="p" substitutionGroup="o"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr) // a <p> attributes to both the p member and (via subst) the o member
	})

	t.Run("two same-name members", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType><xs:all>
    <xs:element ref="o"/>
    <xs:element name="x" type="xs:boolean"/>
    <xs:element ref="o"/>
  </xs:all></xs:complexType></xs:element>
  <xs:element name="o" type="xs:integer"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})

	// XSD 1.1 multiple-head substitution: an element in the substitution group of
	// TWO distinct all members is substitutable for both → UPA violation.
	t.Run("multi-head substitution member ambiguous over two all members", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType><xs:all>
    <xs:element ref="o"/>
    <xs:element ref="p"/>
  </xs:all></xs:complexType></xs:element>
  <xs:element name="o" type="xs:integer"/>
  <xs:element name="p" type="xs:integer"/>
  <xs:element name="q" substitutionGroup="o p"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr) // q is substitutable for both o and p
	})
}

// TestAll11MultiHeadSubstParsing verifies that a multiple-head substitutionGroup
// list is split on XSD whitespace only and registers the element under every head
// (so it is accepted where any head is expected); under XSD 1.0 the (spaced)
// value is a single — non-matching — QName.
func TestAll11MultiHeadSubstParsing(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType><xs:sequence>
    <xs:element ref="h1"/>
    <xs:element ref="h2"/>
  </xs:sequence></xs:complexType></xs:element>
  <xs:element name="h1" abstract="true" type="xs:string"/>
  <xs:element name="h2" abstract="true" type="xs:string"/>
  <xs:element name="m" substitutionGroup="h1  h2" type="xs:string"/>
</xs:schema>`

	// m is substitutable for both h1 and h2.
	require.NoError(t, validateAll11(t, schema, `<doc><m>x</m><m>y</m></doc>`))
}

// TestAll11OptionalAllIdenticalRestriction guards against occurrence
// double-folding: an identical restriction of an OPTIONAL xs:all (minOccurs="0")
// whose member is required must be valid — the group-level occurrence is checked
// once by groupRestrictsGroup, so the per-member contribution must NOT also be
// scaled by the root group's minOccurs (which would drop the member min to 0).
func TestAll11OptionalAllIdenticalRestriction(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:complexContent><xs:restriction base="xs:anyType">
    <xs:all minOccurs="0">
      <xs:element name="a" minOccurs="1" maxOccurs="1"/>
    </xs:all>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b">
    <xs:all minOccurs="0">
      <xs:element name="a" minOccurs="1" maxOccurs="1"/>
    </xs:all>
  </xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
	_, cerr := compileAll11(t, schema)
	require.NoError(t, cerr)
}

// TestAll11NestedWildcardRestrictionRouting guards flatten-aware wildcard
// detection: a base/derived all whose wildcard is reached ONLY through a nested
// 1/1 all-group ref must route to the wildcard-aware subsumption, not the
// counting fast-path (which rejects any wildcard). An identical restriction is
// valid.
func TestAll11NestedWildcardRestrictionRouting(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all>
    <xs:element name="a" minOccurs="0" maxOccurs="3"/>
    <xs:any namespace="##other" minOccurs="0" maxOccurs="5" processContents="skip"/>
  </xs:all></xs:group>
  <xs:complexType name="b"><xs:all>
    <xs:group ref="g"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:group ref="g"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
	_, cerr := compileAll11(t, schema)
	require.NoError(t, cerr)
}

// TestAll11AbstractBaseHeadNoDirectRestriction: a derived local element sharing
// the QName of an ABSTRACT base all member head must NOT be accepted as a direct
// restriction (runtime can't accept a direct instance of the abstract head).
func TestAll11AbstractBaseHeadNoDirectRestriction(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element ref="h"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element name="h" type="xs:string"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="h" abstract="true" type="xs:string"/>
</xs:schema>`
	_, cerr := compileAll11(t, schema)
	require.Error(t, cerr)
}

// TestAll11SubstMemberActualType: a derived local element sharing a QName with a
// global substitution member must be checked against the MEMBER's type, not the
// (anyType) head. A local element typed xs:string cannot restrict a base whose
// matching member is typed xs:int.
func TestAll11SubstMemberActualType(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element ref="a"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element name="m" type="xs:string"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="a" abstract="true"/>
  <xs:element name="m" substitutionGroup="a" type="xs:int"/>
</xs:schema>`
	_, cerr := compileAll11(t, schema)
	require.Error(t, cerr) // local m (xs:string) is not derived from global member m (xs:int)
}

// TestAll11AbstractSubstMemberInstance: an ABSTRACT substitution member can never
// appear in an instance, so it does not satisfy an xs:all member referencing its
// head. (Guards the shared elemMatchesDeclOrSubst fix.)
func TestAll11AbstractSubstMemberInstance(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType><xs:all>
    <xs:element ref="h"/>
  </xs:all></xs:complexType></xs:element>
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m" abstract="true" substitutionGroup="h" type="xs:string"/>
</xs:schema>`
	require.NoError(t, validateAll11(t, schema, `<doc><h>x</h></doc>`)) // concrete head ok
	require.Error(t, validateAll11(t, schema, `<doc><m>x</m></doc>`))   // abstract member cannot appear
}

// TestAll11MultiHeadParsing covers the multi-head substitutionGroup parsing edge
// cases: a whitespace-only value is a schema error, duplicate heads are deduped
// (no false UPA), and an invalid QName token is rejected.
func TestAll11MultiHeadParsing(t *testing.T) {
	t.Parallel()

	t.Run("whitespace-only substitutionGroup is an error", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m" substitutionGroup="   " type="xs:string"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})

	t.Run("duplicate heads deduped, no false UPA", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType><xs:all>
    <xs:element ref="h"/>
    <xs:element name="x" type="xs:string"/>
  </xs:all></xs:complexType></xs:element>
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m" substitutionGroup="h h" type="xs:string"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.NoError(t, cerr) // "h h" dedupes to one head; m registers once, no UPA
	})

	t.Run("invalid QName head token is an error", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m" substitutionGroup="h :bad" type="xs:string"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})
}

// TestAll11WildcardRestrictionNestedGroupFailClosed guards against the
// wildcard-aware all-restriction path SILENTLY accepting a nested non-all derived
// group (sequence/choice). Such a group's elements/wildcards are not accounted
// against the base wildcard, so a nested group emitting an element outside the
// base wildcard's namespace would otherwise false-accept. The path fails closed.
func TestAll11WildcardRestrictionNestedGroupFailClosed(t *testing.T) {
	t.Parallel()

	// Reviewer repro: base all with a namespace-limited wildcard; derived nested
	// sequences emitting <c/> (absent namespace, not admitted by the base wildcard).
	t.Run("nested sequence under wildcard base rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:any namespace="http://allowed.example/"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:sequence>
    <xs:sequence><xs:element name="c"/></xs:sequence>
  </xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})

	// The true false-accept: base wildcard minOccurs="0" (so the missing-required
	// min check does NOT mask the bug); derived nested choice emits <bad/> (absent
	// namespace) which the allowed.example wildcard rejects.
	t.Run("nested choice under optional wildcard base rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:any namespace="http://allowed.example/" minOccurs="0" maxOccurs="1"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:sequence>
    <xs:choice><xs:element name="bad"/></xs:choice>
  </xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})

	// Guard: a direct (non-nested) derived wildcard restricting a wildcard base all
	// still compiles — fail-closed only rejects nested non-all groups, not the
	// supported flat wildcard/element members.
	t.Run("direct wildcard restriction still valid", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:any namespace="http://allowed.example/" minOccurs="0" maxOccurs="3"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:any namespace="http://allowed.example/" minOccurs="0" maxOccurs="2"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.NoError(t, cerr)
	})
}

// TestAll11SubstMemberNoTypeRestriction guards XSDALL-880-001: when the
// constraining substitution-group member has NO explicit @type (it inherits its
// head's type), the restriction type-derivation check must use the EFFECTIVE type
// — not the nil raw Type pointer (which would skip the check and false-accept). A
// derived local element sharing the member's QName but typed xs:string cannot
// restrict a base whose member effectively types xs:int (from the head).
func TestAll11SubstMemberNoTypeRestriction(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element ref="h"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element name="m" type="xs:string"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="h" abstract="true" type="xs:int"/>
  <xs:element name="m" substitutionGroup="h"/>
</xs:schema>`
	_, cerr := compileAll11(t, schema)
	require.Error(t, cerr) // local m (xs:string) not derived from member m's effective type (xs:int)
}

// TestAll11UnderMinDiagnostic guards PR880-001 (diagnostic): a member PRESENT but
// under its minOccurs must still appear in the missing-element "Expected is"
// hint, so the list is never empty.
func TestAll11UnderMinDiagnostic(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType><xs:all>
    <xs:element name="c" minOccurs="2" maxOccurs="5"/>
  </xs:all></xs:complexType></xs:element>
</xs:schema>`
	schema11, cerr := compileAll11(t, schema)
	require.NoError(t, cerr)
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc><c/></doc>`))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	verr := xsd.NewValidator(schema11).ErrorHandler(collector).Validate(t.Context(), idoc)
	require.Error(t, verr) // one <c/> is below minOccurs=2

	var b strings.Builder
	for _, e := range collector.Errors() {
		b.WriteString(e.Error())
	}
	msg := b.String()
	require.Contains(t, msg, "Missing child element")
	require.Contains(t, msg, "( c )")   // expected list names the under-min member c
	require.NotContains(t, msg, "(  )") // not the empty-list artifact
}

// TestAll11SubstMemberTypeSubstitutability guards XSDALL-880-001 (round 5):
// subsumption must verify the substitution member's EFFECTIVE type is validly
// substitutable for the base head's type — not merely that the name is a member.
// A derived element mapping to a member whose type is NOT derived from the head
// type is rejected; one whose type IS derived is accepted.
func TestAll11SubstMemberTypeSubstitutability(t *testing.T) {
	t.Parallel()

	t.Run("member type not substitutable for head rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element ref="h"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element name="m" type="xs:string"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="h" abstract="true" type="xs:int"/>
  <xs:element name="m" substitutionGroup="h" type="xs:string"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr) // member m (xs:string) is not substitutable for head h (xs:int)
	})

	t.Run("member type derived from head accepted", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element ref="h"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element name="m" type="xs:int"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="h" abstract="true" type="xs:integer"/>
  <xs:element name="m" substitutionGroup="h" type="xs:int"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.NoError(t, cerr) // member m (xs:int) is derived from head h (xs:integer)
	})
}

// TestAll11UPAAbstractMemberCompetes: in XSD 1.1 an ABSTRACT substitution member
// is still a member of the head's substitution group BY DECLARATION, so it
// COMPETES for unique particle attribution even though it can never appear in an
// instance. An xs:all with a head ref plus a local element sharing the QName of an
// abstract member of that head is therefore a UPA violation (W3C wgData/sg/upa.xsd,
// bug 4337 — XSD 1.0 would accept it, 1.1 rejects it). Instance MATCHING, by
// contrast, skips abstract members (a separate question, covered elsewhere).
func TestAll11UPAAbstractMemberCompetes(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t"><xs:all>
    <xs:element ref="h"/>
    <xs:element name="m" type="xs:string"/>
  </xs:all></xs:complexType>
  <xs:element name="doc" type="t"/>
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m" abstract="true" substitutionGroup="h"/>
</xs:schema>`
	_, cerr := compileAll11(t, schema)
	require.Error(t, cerr) // abstract member m competes with local m → cos-nonambig (UPA) violation
}

// TestAll11SubsumptionBuiltinDerivation guards XSDALL-880-002 (round 6): the
// all-restriction NameAndTypeOK check must be BUILT-IN-AWARE. The 1.0 built-in
// simple types are not BaseType-linked, so a derived member typed xs:int
// restricting a base member typed xs:integer must be ACCEPTED (xs:int IS derived
// from xs:integer) — a plain isDerivedFrom would false-reject it.
func TestAll11SubsumptionBuiltinDerivation(t *testing.T) {
	t.Parallel()

	t.Run("derived xs:int restricting base xs:integer accepted", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element name="c" type="xs:integer"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element name="c" type="xs:int"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.NoError(t, cerr)
	})

	// A non-derivation (xs:string vs xs:integer) is still rejected.
	t.Run("unrelated builtin types rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"><xs:all>
    <xs:element name="c" type="xs:integer"/>
  </xs:all></xs:complexType>
  <xs:complexType name="r"><xs:complexContent><xs:restriction base="b"><xs:all>
    <xs:element name="c" type="xs:string"/>
  </xs:all></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})
}

// TestAll11InlineNestedAllRejected guards XSDALL-880-003: an INLINE <xs:all>
// directly inside another <xs:all> violates cos-all-limited even in 1.1 and is a
// schema error (only a <xs:group ref> resolving to a 1/1 all-group is the relaxed
// allowed nesting).
func TestAll11InlineNestedAllRejected(t *testing.T) {
	t.Parallel()

	t.Run("inline all under all rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t"><xs:all>
    <xs:element name="a"/>
    <xs:all><xs:element name="b"/></xs:all>
  </xs:all></xs:complexType>
  <xs:element name="doc" type="t"/>
</xs:schema>`
		_, cerr := compileAll11(t, schema)
		require.Error(t, cerr)
	})

	t.Run("nested all via 1/1 group ref still allowed", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">
  <xs:complexType name="t"><xs:all>
    <xs:element name="a"/>
    <xs:group ref="g"/>
  </xs:all></xs:complexType>
  <xs:group name="g"><xs:all><xs:element name="b"/></xs:all></xs:group>
  <xs:element name="doc" type="t"/>
</xs:schema>`
		schema11, cerr := compileAll11(t, schema)
		require.NoError(t, cerr)
		// flattened members a,b matched order-independently.
		require.NoError(t, validateAll11(t, schema, `<doc><b/><a/></doc>`))
		_ = schema11
	})
}
