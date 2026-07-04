package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A derived xs:any wildcard restricting a base model group of element
// declarations has NO derivation rule under §3.9.6 (Particle Valid
// (Restriction)) — the wildcard admits expanded names the element group forbids,
// so it is not a language subset — so it is invalid in BOTH XSD 1.0 and XSD 1.1
// (W3C msData particlesHa163 and siblings). The wildcard-restricts-model-group
// decision is version-independent and fail-closed: the particleLanguageSubset
// fallback cannot model a derived wildcard, so accepting this would be unsound.
func TestWildcardRestrictsModelGroup(t *testing.T) {
	t.Parallel()

	// base: sequence(choice(a,b){2}); derived: sequence(any) restricting it.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice maxOccurs="2">
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="##any"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	for _, ver := range []xsd.Version{xsd.Version10, xsd.Version11} {
		t.Run(versionName(ver)+" rejects", func(t *testing.T) {
			t.Parallel()
			schema, _, cerr := compileWith(t, ver, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
				"a derived wildcard restricting a base element group is not a language subset — invalid in both versions")
			require.Nil(t, schema)
		})
	}
}

// versionName renders an xsd.Version as a stable subtest name.
func versionName(v xsd.Version) string {
	if v == xsd.Version11 {
		return "xsd11"
	}
	return "xsd10"
}

// A base "model group" that is a §3.9.6-POINTLESS 1/1 wrapper around a single
// wildcard is not a group of element declarations: it reduces to that wildcard,
// so a derived wildcard restricting it is decided by the ordinary
// wildcard-vs-wildcard NSSubset rule and is a VALID restriction in XSD 1.0.
func TestWildcardRestrictsPointlessWildcardWrapper(t *testing.T) {
	t.Parallel()

	// base: sequence(sequence(xs:any)); derived: sequence(xs:any).
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:sequence>
        <xs:any namespace="##any"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="##any"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	schema, _, cerr := compileWith(t, xsd.Version10, schemaXML)
	require.NoError(t, cerr,
		"a wildcard restricting a pointless base wildcard wrapper reduces to NSSubset and is valid in XSD 1.0")
	require.NotNil(t, schema)
}

// A derived wildcard with an EMPTY positive namespace constraint (namespace="")
// matches no expanded name — its language is empty, a trivial subset of any base
// — so it is a VALID restriction of a base element group in XSD 1.0, by the same
// rationale as maxOccurs="0".
func TestWildcardEmptyNamespaceRestrictsGroup(t *testing.T) {
	t.Parallel()

	// base: sequence(choice(a,b){2}); derived: sequence(xs:any namespace="").
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice maxOccurs="2">
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace=""/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	schema, _, cerr := compileWith(t, xsd.Version10, schemaXML)
	require.NoError(t, cerr,
		"an empty-namespace derived wildcard has the empty language and validly restricts a base element group in XSD 1.0")
	require.NotNil(t, schema)
}

// §3.9.6 gives no rule for a wildcard restricting a base group of element
// DECLARATIONS, but a base group composed SOLELY of wildcards and/or empty
// nested groups is NOT an element group: an emitting derived wildcard restricting
// it is a wildcard-vs-wildcard language question, which this rule must not decide.
// So the emitting reject does not fire and the restriction COMPILES in XSD 1.0.
// Here the base's nested sequence{2,2}(xs:any{3,3}) has language equivalent to
// xs:any{6,6}, which the derived wildcard supplies; no element declaration appears
// in that base group.
func TestWildcardRestrictsPureWildcardGroup(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:sequence minOccurs="2" maxOccurs="2">
        <xs:any namespace="##any" minOccurs="3" maxOccurs="3" processContents="skip"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="##any" minOccurs="6" maxOccurs="6" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	schema, _, cerr := compileWith(t, xsd.Version10, schemaXML)
	require.NoError(t, cerr,
		"a wildcard restricting a base group with no element declarations is a wildcard-vs-wildcard question, not decided by the wildcard-restricts-element-group rejection")
	require.NotNil(t, schema)
}

// A base pure-wildcard group whose positions carry DIFFERENT namespace
// constraints does not reduce language-exactly to a single uniform wildcard, so a
// derived ##any wildcard is NOT a proven language subset — it would admit an
// empty, single-child, wrong-namespace, or overlong sequence the base rejects.
// The sound, fail-closed decision REJECTS it in both versions.
func TestWildcardRestrictsMixedNamespaceWildcardGroup(t *testing.T) {
	t.Parallel()

	// base: sequence(sequence(any urn:a strict, any urn:b strict)) — language is
	// exactly one urn:a child then one urn:b child.
	// derived: sequence(any ##any skip min0 unbounded) — accepts far more.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:sequence>
        <xs:any namespace="urn:a" processContents="strict"/>
        <xs:any namespace="urn:b" processContents="strict"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="##any" minOccurs="0" maxOccurs="unbounded" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	for _, ver := range []xsd.Version{xsd.Version10, xsd.Version11} {
		t.Run(versionName(ver), func(t *testing.T) {
			t.Parallel()
			schema, _, cerr := compileWith(t, ver, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
				"a mixed-namespace base wildcard group has no single-wildcard reduction, so a broadening derived wildcard is not a language subset")
			require.Nil(t, schema)
		})
	}
}

// A base group whose only element declaration is NON-EMITTING (a prohibited
// element, minOccurs="0" maxOccurs="0") is effectively pure-wildcard: the
// prohibited element emits nothing, so the base reduces to its wildcards alone
// and the derived wildcard is a valid restriction ONLY when its LANGUAGE is a
// subset. Here the base's inner sequence{1,unbounded}(any{2,2} skip, e{0,0})
// emits {2,4,6,…} children (an occurrence HOLE — never exactly 1), so it does
// NOT reduce to a bare wildcard with a contiguous range. A derived any{1,1}
// therefore admits a single child the base never accepts, so the restriction is
// INVALID — the sound, fail-closed decision REJECTS it in both versions.
func TestWildcardRestrictsGroupWithProhibitedElementHole(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:sequence maxOccurs="unbounded">
        <xs:any namespace="##any" minOccurs="2" maxOccurs="2" processContents="skip"/>
        <xs:element name="e" type="xs:string" minOccurs="0" maxOccurs="0"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="##any" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	for _, ver := range []xsd.Version{xsd.Version10, xsd.Version11} {
		t.Run(versionName(ver), func(t *testing.T) {
			t.Parallel()
			schema, _, cerr := compileWith(t, ver, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
				"a base group emitting {2,4,6,…} children has an occurrence hole; a derived any{1,1} admits a single child the base rejects, so the restriction is not a language subset")
			require.Nil(t, schema)
		})
	}
}

// wildcardGroupSchema builds a restriction of a base model group of element
// declarations by a `<xs:sequence>` wrapping a single `<xs:any>` carrying the
// given attribute string (namespace/minOccurs/maxOccurs). baseChoice controls
// whether the base group is NON-emptiable (choice(a,b){2}) or EMPTIABLE
// (choice(a,b) minOccurs="0").
func wildcardGroupSchema(anyAttrs, baseChoiceAttrs string) string {
	return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice ` + baseChoiceAttrs + `>
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any ` + anyAttrs + `/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
}

// The derived wildcard's LANGUAGE relative to the base model group decides
// validity in XSD 1.0 (§3.9.6 has no wildcard-restricts-element-group rule):
//   - an EMPTY language {} (a matchesNothing wildcard that must still occur ≥1)
//     is a subset of every base → ACCEPT;
//   - the {ε} language (maxOccurs="0", or a matchesNothing wildcard with
//     minOccurs="0") is a subset iff the base is emptiable;
//   - an EMITTING wildcard has no rule → REJECT.
func TestWildcardRestrictsGroupLanguageCells(t *testing.T) {
	t.Parallel()

	const nonEmptiableBase = `maxOccurs="2"` // choice(a,b){2} — never empty
	const emptiableBase = `minOccurs="0"`    // choice(a,b){0,1} — emptiable

	tests := []struct {
		name        string
		anyAttrs    string
		baseAttrs   string
		wantErr     bool
		description string
	}{
		{
			name:        "empty-ns min0 over non-emptiable base rejects",
			anyAttrs:    `namespace="" minOccurs="0"`,
			baseAttrs:   nonEmptiableBase,
			wantErr:     true,
			description: "language {ε}; a non-emptiable base does not contain ε",
		},
		{
			name:        "maxOccurs0 over non-emptiable base rejects",
			anyAttrs:    `namespace="##any" minOccurs="0" maxOccurs="0"`,
			baseAttrs:   nonEmptiableBase,
			wantErr:     true,
			description: "language {ε}; a non-emptiable base does not contain ε",
		},
		{
			name:        "empty-ns min1 over non-emptiable base accepts",
			anyAttrs:    `namespace="" minOccurs="1"`,
			baseAttrs:   nonEmptiableBase,
			wantErr:     false,
			description: "empty language {}: must match ≥1 of an impossible namespace → subset of any base",
		},
		{
			name:        "empty-ns min0 over emptiable base accepts",
			anyAttrs:    `namespace="" minOccurs="0"`,
			baseAttrs:   emptiableBase,
			wantErr:     false,
			description: "language {ε}; an emptiable base contains ε",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := wildcardGroupSchema(tc.anyAttrs, tc.baseAttrs)
			schema, _, cerr := compileWith(t, xsd.Version10, schemaXML)
			if tc.wantErr {
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, tc.description)
				require.Nil(t, schema)
				return
			}
			require.NoError(t, cerr, tc.description)
			require.NotNil(t, schema)
		})
	}
}

// A base xs:choice with a non-emitting (maxOccurs=0) element branch alongside a
// wildcard is ε-emptiable and carries no emitting element declaration, so an
// emitting derived wildcard restricting it stays a conservative accept in XSD
// 1.0. A syntactic fold of the base must not drop the choice branch's
// emptiability and false-reject.
func TestWildcardRestrictsChoiceWithProhibitedBranch(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice>
        <xs:any namespace="##any"/>
        <xs:element name="e" type="xs:string" minOccurs="0" maxOccurs="0"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="##any" minOccurs="0"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	schema, _, cerr := compileWith(t, xsd.Version10, schemaXML)
	require.NoError(t, cerr,
		"an emitting wildcard restricting a pure-wildcard/ε-emptiable base group is a conservative accept in XSD 1.0")
	require.NotNil(t, schema)
}

// FINDING 1 — the XSD 1.1 open-content leniency must NOT be a blanket type-level
// accept. Here the base declares a NON-emptiable model group (choice(a) requires an
// `a`) and carries interleave open content over urn:open; the derived drops the
// required element and restricts with a declared wildcard over the DISJOINT
// urn:other. The base accepts neither a urn:other child (not in its open content)
// nor an empty declared model (a is required), so the derived is not a language
// subset. Because the base group at this position is not emptiable AND the derived
// wildcard reaches outside the base open content, the wildcard-restricts-model-group
// decision falls through to the sound reject rather than deferring to quadrant B
// (which exempts a disjoint-namespace wildcard).
func TestWildcardRestrictsGroupOpenContentDisjointNamespace(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:openContent mode="interleave"><xs:any namespace="urn:open" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:choice>
        <xs:element name="a" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:openContent mode="interleave"><xs:any namespace="urn:open" processContents="skip"/></xs:openContent>
        <xs:sequence>
          <xs:any namespace="urn:other" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	schema, _, cerr := compileWith(t, xsd.Version11, schemaXML)
	require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
		"a wildcard over a namespace disjoint from the base open content, dropping a required base element, is not a language subset")
	require.Nil(t, schema)
}

// The emptiable-base companion of FINDING 1: the base model group IS emptiable
// (choice(a) minOccurs=0) and carries interleave open content over urn:open, but the
// derived declared wildcard is over the DISJOINT urn:other. A urn:other child is
// admitted by neither the emptiable declared model nor the base open content, so the
// derived still is not a subset — the namespace-subset guard (not just emptiability)
// keeps it rejected.
func TestWildcardRestrictsEmptiableGroupOpenContentDisjoint(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:openContent mode="interleave"><xs:any namespace="urn:open" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:choice minOccurs="0">
        <xs:element name="a" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:openContent mode="interleave"><xs:any namespace="urn:open" processContents="skip"/></xs:openContent>
        <xs:sequence>
          <xs:any namespace="urn:other" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	schema, _, cerr := compileWith(t, xsd.Version11, schemaXML)
	require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
		"an emitting wildcard outside the base open content namespace is not covered even when the base group is emptiable")
	require.Nil(t, schema)
}

// The delegation-accept companion of FINDING 1: the base model group is emptiable
// AND the derived declared wildcard is a SUBSET of the base open content (urn:open),
// with processContents at least as strong (skip>=skip), so its children land in the
// open-content region and the restriction is a valid subset. The wildcard-restricts-
// model-group decision defers to quadrant B, which accepts.
func TestWildcardRestrictsEmptiableGroupOpenContentCovered(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:openContent mode="interleave"><xs:any namespace="urn:open" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:choice minOccurs="0">
        <xs:element name="a" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:openContent mode="interleave"><xs:any namespace="urn:open" processContents="skip"/></xs:openContent>
        <xs:sequence>
          <xs:any namespace="urn:open" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	schema, _, cerr := compileWith(t, xsd.Version11, schemaXML)
	require.NoError(t, cerr,
		"a derived wildcard within the base open content namespace over an emptiable base group is a valid subset restriction")
	require.NotNil(t, schema)
}

// FINDING 2 — the occurrence-interval collapse must model language EXACTLY, holes
// included. The base wraps an optional group (sequence{0,1}) around an
// any{2,unbounded}, so its emission count-set is {0} ∪ [2,∞) — a HOLE at 1. A
// derived any{1,1} emits exactly one child, which the base never accepts, so the
// restriction is invalid. applyOccReduction must NOT collapse {0} ∪ [2,∞) to [0,∞).
// Version-INDEPENDENT: the reduction runs in both XSD 1.0 and 1.1.
func TestWildcardRestrictsGroupOccurrenceHoleAtOne(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:sequence minOccurs="0" maxOccurs="1">
        <xs:any namespace="##any" minOccurs="2" maxOccurs="unbounded" processContents="skip"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="##any" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	for _, ver := range []xsd.Version{xsd.Version10, xsd.Version11} {
		t.Run(versionName(ver), func(t *testing.T) {
			t.Parallel()
			schema, _, cerr := compileWith(t, ver, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
				"the base emits {0} union [2, infinity) — a hole at 1 — so a derived any{1,1} is not a language subset")
			require.Nil(t, schema)
		})
	}
}

// FINDING 3 — a base xs:choice of two skip wildcards over DIFFERENT namespaces
// (urn:a, urn:b), each emitting at most one child, reduces LANGUAGE-EXACTLY to a
// single wildcard over {urn:a, urn:b}: the choice picks one branch, i.e. one child in
// urn:a OR one in urn:b — exactly what the union wildcard admits (no room to mix, as
// each branch is {1,1}). So the derived any urn:a skip IS a subset. The choice
// reduction must UNION the namespaces (with identical processContents), not require
// identical constraints. Version-INDEPENDENT.
func TestWildcardRestrictsChoiceUnionNamespaces(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice>
        <xs:any namespace="urn:a" processContents="skip"/>
        <xs:any namespace="urn:b" processContents="skip"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="urn:a" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	for _, ver := range []xsd.Version{xsd.Version10, xsd.Version11} {
		t.Run(versionName(ver), func(t *testing.T) {
			t.Parallel()
			schema, _, cerr := compileWith(t, ver, schemaXML)
			require.NoError(t, cerr,
				"a base choice of single-child wildcards over urn:a|urn:b reduces to one wildcard over {urn:a,urn:b}; any urn:a is a subset")
			require.NotNil(t, schema)
		})
	}
}

// The soundness boundary of FINDING 3: when a choice branch emits MORE than one
// child (any urn:b {2,2}), unioning the namespaces would let the single reduced
// wildcard admit a namespace MIX (e.g. one urn:a and one urn:b) that the choice —
// which picks one branch — forbids. So the different-namespace union is NOT
// language-exact and the reduction fails closed, keeping a broadening derived
// wildcard rejected. Version-INDEPENDENT.
func TestWildcardRestrictsChoiceUnionRejectsMultiChildBranch(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice>
        <xs:any namespace="urn:a" processContents="skip"/>
        <xs:any namespace="urn:b" minOccurs="2" maxOccurs="2" processContents="skip"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="##any" minOccurs="0" maxOccurs="unbounded" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	for _, ver := range []xsd.Version{xsd.Version10, xsd.Version11} {
		t.Run(versionName(ver), func(t *testing.T) {
			t.Parallel()
			schema, _, cerr := compileWith(t, ver, schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
				"a choice branch emitting two urn:b children makes the namespace union inexact, so the broadening derived wildcard is not a proven subset")
			require.Nil(t, schema)
		})
	}
}

// FINDING 1 (round 3) — the XSD 1.1 open-content delegation must be INTERLEAVE-only.
// Suffix open content requires open-content children to TRAIL every declared element,
// so a derived declared wildcard placed BEFORE a required declared element cannot be
// governed by the base's suffix open content. Here the base is
// choice(a)?, b, any(urn:t)* with SUFFIX open content over urn:t; the derived reorders
// a urn:t wildcard to the FRONT (any(urn:t), b, any(urn:t)*). The base rejects a urn:t
// child before b (it must trail b as declared-wildcard/open content), but the derived
// accepts it — not a language subset. The delegation must NOT accept the front wildcard
// against the emptiable choice(a)? by leaning on the trailing open content.
func TestWildcardRestrictsGroupSuffixOpenContentReorder(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="base">
    <xs:openContent mode="suffix"><xs:any namespace="urn:t" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:choice minOccurs="0">
        <xs:element name="a" type="xs:string"/>
      </xs:choice>
      <xs:element name="b" type="xs:string"/>
      <xs:any namespace="urn:t" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:openContent mode="suffix"><xs:any namespace="urn:t" processContents="skip"/></xs:openContent>
        <xs:sequence>
          <xs:any namespace="urn:t" processContents="skip"/>
          <xs:element name="b" type="xs:string"/>
          <xs:any namespace="urn:t" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	schema, _, cerr := compileWith(t, xsd.Version11, schemaXML)
	require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
		"a derived declared wildcard reordered before a required element cannot be governed by the base's SUFFIX open content, which requires trailing children")
	require.Nil(t, schema)
}

// FINDING 2 (round 3) — wildcard-constraint equality must be an ORDER-INDEPENDENT SET
// comparison. A base pure-wildcard sequence of two wildcards over the SAME namespace
// set listed in DIFFERENT lexical order (urn:a urn:b, then urn:b urn:a) reduces
// language-exactly to any({urn:a,urn:b}){2,2}; the derived any(urn:a){2,2} is a
// provable subset. Comparing the namespace lists lexically would fail the reduction
// and false-reject the reversed-order base. Version-INDEPENDENT.
func TestWildcardRestrictsGroupNamespaceListOrder(t *testing.T) {
	t.Parallel()

	mk := func(secondNS string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:sequence>
        <xs:any namespace="urn:a urn:b" processContents="skip"/>
        <xs:any namespace="` + secondNS + `" processContents="skip"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="urn:a" minOccurs="2" maxOccurs="2" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
	}

	for _, secondNS := range []string{"urn:a urn:b", "urn:b urn:a"} {
		for _, ver := range []xsd.Version{xsd.Version10, xsd.Version11} {
			t.Run(secondNS+"/"+versionName(ver), func(t *testing.T) {
				t.Parallel()
				schema, _, cerr := compileWith(t, ver, mk(secondNS))
				require.NoError(t, cerr,
					"two base wildcards over the same namespace set (any order) reduce to any({urn:a,urn:b}){2,2}; any(urn:a){2,2} is a subset")
				require.NotNil(t, schema)
			})
		}
	}
}

// FINDING 3 (round 4) — the wildcard-restricts-model-group proof is BRANCH-AWARE. A
// base xs:choice with an ELEMENT branch alongside a WILDCARD branch is not a "group of
// element declarations" that blanket-rejects an emitting derived wildcard: the choice
// admits either <a> OR a urn:x child, so a derived wildcard confined to the WILDCARD
// branch's language (any urn:x skip) is a valid subset — the base wildcard branch
// governs those children. The element branch contributes nothing the derived wildcard
// must produce. A derived wildcard broader than the covering branch (any ##any) admits
// names — e.g. <c> — that neither the element branch nor the wildcard branch accepts,
// so it is NOT a subset and stays rejected. Version-INDEPENDENT.
func TestWildcardRestrictsChoiceWithElementBranch(t *testing.T) {
	t.Parallel()

	mk := func(derivedAny string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice>
        <xs:element name="a" type="xs:string"/>
        <xs:any namespace="urn:x" processContents="skip"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>` + derivedAny + `</xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
	}

	tests := []struct {
		name       string
		derivedAny string
		wantErr    bool
		reason     string
	}{
		{
			name:       "confined-to-wildcard-branch accepts",
			derivedAny: `<xs:any namespace="urn:x" processContents="skip"/>`,
			wantErr:    false,
			reason:     "the derived wildcard is confined to the base wildcard branch's language — a subset",
		},
		{
			name:       "broader-than-branch rejects",
			derivedAny: `<xs:any namespace="##any" processContents="skip"/>`,
			wantErr:    true,
			reason:     "the derived ##any admits names neither the element nor the wildcard branch accepts — not a subset",
		},
	}

	for _, tc := range tests {
		for _, ver := range []xsd.Version{xsd.Version10, xsd.Version11} {
			t.Run(tc.name+"/"+versionName(ver), func(t *testing.T) {
				t.Parallel()
				schema, _, cerr := compileWith(t, ver, mk(tc.derivedAny))
				if tc.wantErr {
					require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, tc.reason)
					require.Nil(t, schema)
					return
				}
				require.NoError(t, cerr, tc.reason)
				require.NotNil(t, schema)
			})
		}
	}
}

// FINDING 4 (round 4) — a base xs:choice of HETEROGENEOUS-processContents wildcard
// branches must NOT be forced into one reduced wildcard. choice(any urn:a strict, any
// urn:b skip) is a set of two pc-tagged branches. A derived any urn:b skip is governed
// by the skip branch (skip >= skip), so it is a valid subset even though the two
// branches cannot fold to a single wildcard (different pc). A derived any urn:a skip,
// however, is governed by the base's STRICT urn:a branch — skip is weaker, so the
// derived admits urn:a content the base strict branch rejects → NOT a subset, rejected.
// The branch-set coverage keys on "base branch no stricter than the derived".
// Version-INDEPENDENT.
func TestWildcardRestrictsChoiceHeterogeneousPC(t *testing.T) {
	t.Parallel()

	mk := func(derivedAny string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice>
        <xs:any namespace="urn:a" processContents="strict"/>
        <xs:any namespace="urn:b" processContents="skip"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>` + derivedAny + `</xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
	}

	tests := []struct {
		name       string
		derivedAny string
		wantErr    bool
		reason     string
	}{
		{
			name:       "skip-branch-covered accepts",
			derivedAny: `<xs:any namespace="urn:b" processContents="skip"/>`,
			wantErr:    false,
			reason:     "the derived urn:b skip is covered by the base skip branch (skip >= skip)",
		},
		{
			name:       "weaker-than-strict-branch rejects",
			derivedAny: `<xs:any namespace="urn:a" processContents="skip"/>`,
			wantErr:    true,
			reason:     "the derived urn:a skip admits content the base's STRICT urn:a branch rejects — not a subset",
		},
	}

	for _, tc := range tests {
		for _, ver := range []xsd.Version{xsd.Version10, xsd.Version11} {
			t.Run(tc.name+"/"+versionName(ver), func(t *testing.T) {
				t.Parallel()
				schema, _, cerr := compileWith(t, ver, mk(tc.derivedAny))
				if tc.wantErr {
					require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, tc.reason)
					require.Nil(t, schema)
					return
				}
				require.NoError(t, cerr, tc.reason)
				require.NotNil(t, schema)
			})
		}
	}
}

// FINDING 1 (round 4) — the §3.4.6.4 quadrant-B exemption must be OCCURRENCE-CAPACITY
// aware. When a derived declared wildcard maps to a base declared wildcard (quadrant D)
// only UP TO that base wildcard's occurrence capacity, the EXCESS spills into the base
// OPEN content and must satisfy the open-content restriction (namespace/processContents).
// Here the base wraps a declared `any urn:x skip maxOccurs=1` in an emptiable group (so
// the derived wildcard maps to the base MODEL GROUP and the content-model check delegates
// to quadrant B) plus INTERLEAVE open content over `urn:x STRICT`; the derived declares
// `any urn:x skip maxOccurs=unbounded`. The first urn:x child is covered by the base
// declared wildcard, but every further one spills into the base's STRICT open content —
// which the derived's SKIP does not enforce, so the base rejects invalid urn:x content
// the derived accepts. A capacity-blind exemption compiles this unsoundly; the
// capacity-aware exemption rejects it. When the base declared wildcard's capacity is
// UNBOUNDED (covers the derived), no child spills and the restriction is a valid subset.
func TestWildcardRestrictsGroupOpenContentCapacity(t *testing.T) {
	t.Parallel()

	mk := func(baseMax string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:x" xmlns:t="urn:x">
  <xs:complexType name="base">
    <xs:openContent mode="interleave"><xs:any namespace="urn:x" processContents="strict"/></xs:openContent>
    <xs:sequence>
      <xs:sequence minOccurs="0">
        <xs:any namespace="urn:x" processContents="skip" minOccurs="0" maxOccurs="` + baseMax + `"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:openContent mode="interleave"><xs:any namespace="urn:x" processContents="strict"/></xs:openContent>
        <xs:sequence>
          <xs:any namespace="urn:x" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
	}

	t.Run("excess-spills-to-strict-open rejects", func(t *testing.T) {
		t.Parallel()
		schema, _, cerr := compileWith(t, xsd.Version11, mk("1"))
		require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
			"the derived skip wildcard's excess beyond the base declared max spills into the base's STRICT open content, escaping validation")
		require.Nil(t, schema)
	})

	t.Run("base-capacity-covers-derived accepts", func(t *testing.T) {
		t.Parallel()
		schema, _, cerr := compileWith(t, xsd.Version11, mk("unbounded"))
		require.NoError(t, cerr,
			"an unbounded base declared wildcard covers the derived, so no child spills into open content — a valid restriction")
		require.NotNil(t, schema)
	})
}

// FINDING (round 5) — the all-wildcard emission profile ERASES element declarations
// (an optional/prohibited/dead element reduces to ε), so its name is dropped from the
// profile. But XSD 1.1 ELEMENT-OVER-WILDCARD PRECEDENCE routes a base child whose name
// matches an element declaration to that ELEMENT (validated against its type), not to
// an overlapping wildcard. A derived wildcard that re-admits such a reserved name
// accepts content the base validates more strictly, so it is NOT a language subset. The
// reserved-names guard REJECTS a derived wildcard that admits a base element name a live
// cover also admits.
//
//   - choice(element e:int, any ##any skip) restricted by any ##any skip: the base routes
//     <e> to the int element (element precedence over the ##any branch) and rejects
//     <e>bad</e>; the derived ##any skip accepts it unvalidated → REJECT.
//   - sequence(element e:int min0, any ##any skip) restricted by any ##any skip: the base
//     routes <e> to the int element and rejects <e>bad</e>; the derived accepts it → REJECT.
//
// Both bases overlap an element with a ##any wildcard, which is UPA-invalid in XSD 1.0
// (never reaching the reduction), so the unsound accept is reachable — and the guard
// gated — only in XSD 1.1.
func TestWildcardRestrictsGroupReservedElementName(t *testing.T) {
	t.Parallel()

	choiceBase := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice>
        <xs:element name="e" type="xs:int"/>
        <xs:any namespace="##any" processContents="skip"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="##any" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	sequenceBase := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:element name="e" type="xs:int" minOccurs="0"/>
      <xs:any namespace="##any" processContents="skip"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="##any" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	for _, tc := range []struct {
		name      string
		schemaXML string
	}{
		{"choice-element-branch", choiceBase},
		{"sequence-optional-element", sequenceBase},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schema, _, cerr := compileWith(t, xsd.Version11, tc.schemaXML)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
				"element-over-wildcard precedence routes the reserved element name to the base element's type; a derived wildcard re-admitting it is not a language subset")
			require.Nil(t, schema)
		})
	}
}

// The sound companion of the reserved-element guard: the base element `a` is in the
// absent namespace, DISJOINT from the derived wildcard's namespace (urn:x), so the
// derived wildcard EXCLUDES `a` — it never re-admits the reserved element name. The
// base wildcard branch (urn:x skip) governs every urn:x child the derived produces, so
// the restriction IS a language subset and compiles.
func TestWildcardRestrictsGroupReservedElementExcluded(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice>
        <xs:element name="a" type="xs:int"/>
        <xs:any namespace="urn:x" processContents="skip"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:any namespace="urn:x" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	schema, _, cerr := compileWith(t, xsd.Version11, schemaXML)
	require.NoError(t, cerr,
		"the base element is in a namespace the derived wildcard excludes, so no reserved name is re-admitted — a valid subset restriction")
	require.NotNil(t, schema)
}
