package xsd

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// referenceSubstMembers is the source of truth the differential tests below
// compare substitutableMembersFor and instanceSubstMembers against: the
// uncached walk, called directly regardless of whether substitutableMembersFor
// itself is consulting a cache.
var referenceSubstMembers = substitutableMembersForUncached

// wantMemberOnly is the expected-names slice shared by every test case whose
// schema declares a single element named "member".
var wantMemberOnly = []string{"member"}

func compileSubstClosureSchema(t *testing.T, version Version, schemaXML string) *Schema {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := NewCompiler().Version(version).Label("test.xsd").Compile(t.Context(), doc)
	require.NoError(t, err)
	return schema
}

func mustGlobalElem(t *testing.T, schema *Schema, name string) *ElementDecl {
	t.Helper()
	decl, ok := schema.LookupElement(name, "")
	require.True(t, ok, "global element %q not found", name)
	return decl
}

// findElemParticle recursively searches a compiled content model for an
// element-particle term with the given local name and IsRef state.
func findElemParticle(mg *ModelGroup, name string, wantRef bool) *ElementDecl {
	if mg == nil {
		return nil
	}
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *ElementDecl:
			if term.Name.Local == name && term.IsRef == wantRef {
				return term
			}
		case *ModelGroup:
			if found := findElemParticle(term, name, wantRef); found != nil {
				return found
			}
		}
	}
	return nil
}

func mustCarrierContent(t *testing.T, schema *Schema, typeName string) *ModelGroup {
	t.Helper()
	td, ok := schema.LookupType(typeName, "")
	require.True(t, ok, "type %q not found", typeName)
	require.NotNil(t, td.ContentModel)
	return td.ContentModel
}

func substMemberNames(members []*ElementDecl) []string {
	names := make([]string, 0, len(members))
	for _, m := range members {
		names = append(names, m.Name.Local)
	}
	return names
}

// requireSameMembers asserts substitutableMembersFor and instanceSubstMembers
// agree with referenceSubstMembers on both the set of members and, for
// substitutableMembersFor, their order. instanceSubstMembers is compared
// against the abstract-filtered reference result, since that is its documented
// contract.
func requireSameMembers(t *testing.T, decl *ElementDecl, schema *Schema, wantAll, wantConcrete []string) {
	t.Helper()

	all := substitutableMembersFor(decl, schema)
	require.Equal(t, wantAll, substMemberNames(all), "substitutableMembersFor")

	ref := referenceSubstMembers(decl, schema)
	require.Equal(t, substMemberNames(ref), substMemberNames(all), "substitutableMembersFor vs reference walk")

	concrete := instanceSubstMembers(decl, schema)
	require.Equal(t, wantConcrete, substMemberNames(concrete), "instanceSubstMembers")

	wantRefConcrete := make([]string, 0, len(ref))
	for _, m := range ref {
		if !m.Abstract {
			wantRefConcrete = append(wantRefConcrete, m.Name.Local)
		}
	}
	require.Equal(t, wantRefConcrete, substMemberNames(concrete), "instanceSubstMembers vs reference walk")
}

// TestSubstitutableMembersForDifferential is a table of substitution-group
// shapes, each asserting substitutableMembersFor and instanceSubstMembers
// return the same member names, in the same order, as the reference walk
// (referenceSubstMembers). It passes against today's uncached walk and stays
// green once substitutableMembersFor starts consulting a precomputed cache.
func TestSubstitutableMembersForDifferential(t *testing.T) {
	t.Parallel()

	t.Run("no substitution group", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
</xs:schema>`)
		requireSameMembers(t, mustGlobalElem(t, schema, "head"), schema, []string{}, []string{})
	})

	t.Run("flat group", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="head"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="head"/>
</xs:schema>`)
		requireSameMembers(t, mustGlobalElem(t, schema, "head"), schema,
			[]string{"m1", "m2"}, []string{"m1", "m2"})
	})

	t.Run("chain h<-m1<-m2", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="h"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="m1"/>
</xs:schema>`)
		requireSameMembers(t, mustGlobalElem(t, schema, "h"), schema,
			[]string{"m1", "m2"}, []string{"m1", "m2"})
	})

	t.Run("abstract intermediate with concrete descendant", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="h" abstract="true"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="m1"/>
</xs:schema>`)
		requireSameMembers(t, mustGlobalElem(t, schema, "h"), schema,
			[]string{"m1", "m2"}, []string{"m2"})
	})

	t.Run("block=substitution on the head", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string" block="substitution"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="h"/>
</xs:schema>`)
		requireSameMembers(t, mustGlobalElem(t, schema, "h"), schema, []string{}, []string{})
	})

	t.Run("block=substitution on an intermediate", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="h" block="substitution"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="m1"/>
</xs:schema>`)
		requireSameMembers(t, mustGlobalElem(t, schema, "h"), schema,
			[]string{"m1"}, []string{"m1"})
	})

	t.Run("type-derivation-blocked member", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:integer" block="restriction"/>
  <xs:element name="member" type="xs:int" substitutionGroup="head"/>
</xs:schema>`)
		requireSameMembers(t, mustGlobalElem(t, schema, "head"), schema, []string{}, []string{})
	})

	t.Run("multi-head substitutionGroup", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" type="xs:string"/>
  <xs:element name="b" type="xs:string"/>
  <xs:element name="member" type="xs:string" substitutionGroup="a b"/>
</xs:schema>`)
		requireSameMembers(t, mustGlobalElem(t, schema, "a"), schema, wantMemberOnly, wantMemberOnly)
		requireSameMembers(t, mustGlobalElem(t, schema, "b"), schema, wantMemberOnly, wantMemberOnly)
	})

	t.Run("ref particle to a head", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="head"/>
  <xs:complexType name="carrierType">
    <xs:sequence>
      <xs:element ref="head"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="carrier" type="carrierType"/>
</xs:schema>`)
		mg := mustCarrierContent(t, schema, "carrierType")
		refDecl := findElemParticle(mg, "head", true)
		require.NotNil(t, refDecl, "ref particle for head not found")
		require.True(t, refDecl.IsRef)
		requireSameMembers(t, refDecl, schema, []string{"m1"}, []string{"m1"})
	})

	t.Run("local element particle sharing a head's QName", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="head"/>
  <xs:complexType name="carrierType">
    <xs:sequence>
      <xs:element name="head" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="carrier" type="carrierType"/>
</xs:schema>`)
		mg := mustCarrierContent(t, schema, "carrierType")
		localDecl := findElemParticle(mg, "head", false)
		require.NotNil(t, localDecl, "local particle for head not found")
		require.NotSame(t, mustGlobalElem(t, schema, "head"), localDecl)
		requireSameMembers(t, localDecl, schema, []string{}, []string{})
	})

	t.Run("untyped member inherits head's type", func(t *testing.T) {
		t.Parallel()
		schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="member" substitutionGroup="head"/>
</xs:schema>`)
		requireSameMembers(t, mustGlobalElem(t, schema, "head"), schema, wantMemberOnly, wantMemberOnly)
	})
}

// TestInstanceSubstMembersDoesNotAliasSubstitutableMembersFor is the aliasing
// tripwire: instanceSubstMembers must never mutate the slice
// substitutableMembersFor returns. Calling substitutableMembersFor,
// instanceSubstMembers, then substitutableMembersFor again must yield the same
// full closure both times. Green today (each call currently gets a fresh
// slice); it catches a future cache that hands out a shared backing array to
// instanceSubstMembers' abstract-filter in place.
func TestInstanceSubstMembersDoesNotAliasSubstitutableMembersFor(t *testing.T) {
	t.Parallel()
	schema := compileSubstClosureSchema(t, Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="head" abstract="true"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="head"/>
</xs:schema>`)
	head := mustGlobalElem(t, schema, "head")

	before := substMemberNames(substitutableMembersFor(head, schema))
	require.Equal(t, []string{"m1", "m2"}, before)

	_ = instanceSubstMembers(head, schema)

	after := substMemberNames(substitutableMembersFor(head, schema))
	require.Equal(t, before, after, "instanceSubstMembers must not mutate substitutableMembersFor's closure")
}

// substClosureBuildTestSchemas mirrors the shapes covered by
// TestSubstitutableMembersForDifferential (plus one wide fan-out group), used
// by TestBuildSubstClosuresMatchesUncachedWalk to check buildSubstClosures
// against every global element in each schema, not just the ones the
// differential table names explicitly.
var substClosureBuildTestSchemas = []struct {
	name    string
	version Version
	xml     string
}{
	{"no substitution group", Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
</xs:schema>`},
	{"flat group", Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="head"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="head"/>
</xs:schema>`},
	{"chain h<-m1<-m2", Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="h"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="m1"/>
</xs:schema>`},
	{"abstract intermediate with concrete descendant", Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="h" abstract="true"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="m1"/>
</xs:schema>`},
	{"block=substitution on the head", Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string" block="substitution"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="h"/>
</xs:schema>`},
	{"block=substitution on an intermediate", Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="h" block="substitution"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="m1"/>
</xs:schema>`},
	{"type-derivation-blocked member", Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:integer" block="restriction"/>
  <xs:element name="member" type="xs:int" substitutionGroup="head"/>
</xs:schema>`},
	{"multi-head substitutionGroup", Version11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" type="xs:string"/>
  <xs:element name="b" type="xs:string"/>
  <xs:element name="member" type="xs:string" substitutionGroup="a b"/>
</xs:schema>`},
	{"ref particle to a head", Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="head"/>
  <xs:complexType name="carrierType">
    <xs:sequence>
      <xs:element ref="head"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="carrier" type="carrierType"/>
</xs:schema>`},
	{"untyped member inherits head's type", Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="member" substitutionGroup="head"/>
</xs:schema>`},
	{"wide fan-out group (exercises byName threshold)", Version10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m0" type="xs:string" substitutionGroup="head"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="head"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="head"/>
  <xs:element name="m3" type="xs:string" substitutionGroup="head"/>
  <xs:element name="m4" type="xs:string" substitutionGroup="head"/>
  <xs:element name="m5" type="xs:string" substitutionGroup="head"/>
  <xs:element name="m6" type="xs:string" substitutionGroup="head"/>
  <xs:element name="m7" type="xs:string" substitutionGroup="head"/>
</xs:schema>`},
}

// TestBuildSubstClosuresMatchesUncachedWalk asserts that for every global
// element declaration in each of substClosureBuildTestSchemas, compiled under
// both XSD 1.0 and XSD 1.1, buildSubstClosures' cached entry (or absence of
// one) matches substitutableMembersForUncached exactly: same head, same "all"
// member names in the same order, same "concrete" (abstract-filtered) member
// names in the same order, and a byName index present if and only if the
// closure is at least substClosureByNameThreshold members.
func TestBuildSubstClosuresMatchesUncachedWalk(t *testing.T) {
	t.Parallel()
	for _, tc := range substClosureBuildTestSchemas {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schema := compileSubstClosureSchema(t, tc.version, tc.xml)
			schema.buildSubstClosures()

			for name, decl := range schema.elements {
				want := substitutableMembersForUncached(decl, schema)
				closure, ok := schema.substClosures[name]
				if len(want) == 0 {
					require.False(t, ok, "element %q: expected no closure entry, got one", name.Local)
					continue
				}
				require.True(t, ok, "element %q: expected a closure entry, got none", name.Local)
				require.Same(t, decl, closure.head, "element %q: head mismatch", name.Local)
				require.Equal(t, substMemberNames(want), substMemberNames(closure.all), "element %q: all", name.Local)

				wantConcrete := make([]string, 0, len(want))
				for _, m := range want {
					if !m.Abstract {
						wantConcrete = append(wantConcrete, m.Name.Local)
					}
				}
				require.Equal(t, wantConcrete, substMemberNames(closure.concrete), "element %q: concrete", name.Local)

				if len(want) >= substClosureByNameThreshold {
					require.NotNil(t, closure.byName, "element %q: expected byName index", name.Local)
					require.Len(t, closure.byName, len(want), "element %q: byName size", name.Local)
				} else {
					require.Nil(t, closure.byName, "element %q: expected no byName index", name.Local)
				}
			}
		})
	}
}

// TestSchemaWithInstanceHintsRebuildsSubstClosures is the hint-merge
// invalidation tripwire: a document whose xsi:noNamespaceSchemaLocation
// contributes a NEW substitution-group member must validate an instance of
// that member successfully. The base schema already has a NON-EMPTY closure
// for "head" (member "m1") before the merge, so cloneSchemaSymbolTables'
// struct copy would otherwise hand the merged schema a stale closureHit
// entry (missing "m2") instead of falling back to the uncached walk — the
// aliasing bug fixed by nilling substClosures in cloneSchemaSymbolTables and
// rebuilding it in schemaWithInstanceHints.
func TestSchemaWithInstanceHintsRebuildsSubstClosures(t *testing.T) {
	t.Parallel()

	const hintedSchemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="head"/>
</xs:schema>`
	fsys := fstest.MapFS{
		"member.xsd": &fstest.MapFile{Data: []byte(hintedSchemaXML)},
	}

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`))
	require.NoError(t, err)
	base, err := NewCompiler().FS(fsys).Label("test.xsd").Compile(t.Context(), doc)
	require.NoError(t, err)

	// Confirm the base schema's own closure for "head" is non-empty (m1 only)
	// before any hint is merged in, so the test actually exercises a
	// closureHit on the stale entry rather than the empty-closure fallback.
	baseHead := mustGlobalElem(t, base, "head")
	require.Equal(t, []string{"m1"}, substMemberNames(substitutableMembersFor(baseHead, base)))

	instDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:noNamespaceSchemaLocation="member.xsd">
  <m1>a</m1>
  <m2>b</m2>
</root>`))
	require.NoError(t, err)

	err = NewValidator(base).Label("test.xml").Validate(t.Context(), instDoc)
	require.NoError(t, err, "m2, contributed only by the xsi:schemaLocation hint, must validate as a head member")
}
