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
