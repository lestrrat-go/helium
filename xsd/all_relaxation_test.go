package xsd_test

import (
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
