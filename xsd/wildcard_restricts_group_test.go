package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A derived xs:any wildcard restricting a base model group of element
// declarations has NO derivation rule under §3.9.6 (Particle Valid
// (Restriction)) — the wildcard admits expanded names the element group forbids
// — so it is not a valid restriction in XSD 1.0 (W3C msData particlesHa163 and
// siblings). XSD 1.1's particleLanguageSubset fallback cannot model a derived
// wildcard, so it keeps the prior conservative accept for that path.
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

	t.Run("xsd10 rejects", func(t *testing.T) {
		t.Parallel()
		schema, _, cerr := compileWith(t, xsd.Version10, schemaXML)
		require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
			"a derived wildcard restricting a base model group is invalid in XSD 1.0")
		require.Nil(t, schema)
	})

	t.Run("xsd11 accepts", func(t *testing.T) {
		t.Parallel()
		schema, _, cerr := compileWith(t, xsd.Version11, schemaXML)
		require.NoError(t, cerr, "XSD 1.1 keeps the conservative accept for this path")
		require.NotNil(t, schema)
	})
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

// A base group whose only element declaration is NON-EMITTING (a prohibited
// element, minOccurs="0" maxOccurs="0") is effectively pure-wildcard: the
// prohibited element emits nothing and can never appear in an instance, so an
// emitting derived wildcard restricting the base is a wildcard-vs-wildcard
// question the rejection must not decide. modelGroupContainsElementDecl counts
// only EMITTING element declarations, so it does not color this base as an
// element group and the restriction COMPILES in XSD 1.0.
//
// The base's inner sequence{1,unbounded}(any{2,2} skip, e{0,0}) has an
// occurrence HOLE (2,4,6,… elements, never 1), so pointlessReduce refuses to
// fold it to a bare wildcard — the derived wildcard therefore reaches the
// wildcard-restricts-base-model-group rule (particleValidRestriction line ~211)
// where modelGroupContainsElementDecl decides. Without the emitting-only fix the
// prohibited e is miscounted and the restriction is wrongly REJECTED.
func TestWildcardRestrictsGroupWithProhibitedElement(t *testing.T) {
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

	schema, _, cerr := compileWith(t, xsd.Version10, schemaXML)
	require.NoError(t, cerr,
		"a base group whose only element declaration is prohibited (maxOccurs=0) is effectively pure-wildcard, so an emitting derived wildcard validly restricts it in XSD 1.0")
	require.NotNil(t, schema)
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
