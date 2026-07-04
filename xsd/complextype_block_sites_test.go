package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestComplexTypeBlock_SubstitutionGroup covers gauntlet finding complextype-block
// site 1: a substitution-group head whose declared complex TYPE carries
// block="extension" ({prohibited substitutions}) must NOT admit a member reached by
// an extension derivation, even when the head ELEMENT declaration itself carries no
// block. The effective {disallowed substitutions} unions the element block with the
// declared type's {prohibited substitutions} (Substitution Group OK / cvc-elt.4.3),
// so an extension-derived member is excluded from the head's substitution members.
// Version-independent (the closure feeds both the 1.0 matcher and the 1.1 instance
// path via instanceSubstMembers).
func TestComplexTypeBlock_SubstitutionGroup(t *testing.T) {
	schemaFor := func(block string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
                  xmlns:t="urn:t" targetNamespace="urn:t"
                  elementFormDefault="qualified">
  <xs:complexType name="HBase" ` + block + `>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="HExt">
    <xs:complexContent>
      <xs:extension base="t:HBase">
        <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="h" type="t:HBase"/>
  <xs:element name="m" type="t:HExt" substitutionGroup="t:h"/>
  <xs:element name="container">
    <xs:complexType><xs:sequence><xs:element ref="t:h"/></xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
	}
	// The instance substitutes <m> (extension-derived) for the head <h>.
	instance := `<t:container xmlns:t="urn:t"><t:m><t:a>x</t:a><t:b>y</t:b></t:m></t:container>`

	compileValidate := func(t *testing.T, v xsd.Version, block string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaFor(block)))
		require.NoError(t, err)
		sc, err := xsd.NewCompiler().Version(v).Compile(t.Context(), doc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(sc).Validate(t.Context(), idoc)
	}

	for _, tc := range []struct {
		name string
		ver  xsd.Version
	}{{"xsd10", xsd.Version10}, {"xsd11", xsd.Version11}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("head TYPE block=extension rejects extension-derived member", func(t *testing.T) {
				require.ErrorIs(t, compileValidate(t, tc.ver, `block="extension"`), xsd.ErrValidationFailed)
			})
			t.Run("no head TYPE block accepts extension-derived member", func(t *testing.T) {
				require.NoError(t, compileValidate(t, tc.ver, ``))
			})
		})
	}
}

// TestComplexTypeBlock_SubstitutionIntermediate covers the intermediate-type case of
// Substitution Group OK (Transitive) §3.3.6.3: the {prohibited substitutions} of an
// INTERMEDIATE type definition in a member's derivation chain — not only the head/base
// type — must block the substitution. Chain Base <- Mid(block="extension") <- Leaf,
// head <h> type Base, member <m> type Leaf: Leaf derives from Mid by extension and Mid
// blocks extension, so <m> is not substitutable for <h> even though neither Base nor
// the head element blocks extension. Removing Mid's block makes it valid.
// Version-independent (the closure feeds the 1.0 matcher and the 1.1 instance path).
func TestComplexTypeBlock_SubstitutionIntermediate(t *testing.T) {
	schemaFor := func(midBlock string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
                  xmlns:t="urn:t" targetNamespace="urn:t"
                  elementFormDefault="qualified">
  <xs:complexType name="Base">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="Mid" ` + midBlock + `>
    <xs:complexContent>
      <xs:extension base="t:Base">
        <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="Leaf">
    <xs:complexContent>
      <xs:extension base="t:Mid">
        <xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="h" type="t:Base"/>
  <xs:element name="m" type="t:Leaf" substitutionGroup="t:h"/>
  <xs:element name="container">
    <xs:complexType><xs:sequence><xs:element ref="t:h"/></xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
	}
	instance := `<t:container xmlns:t="urn:t"><t:m><t:a>x</t:a><t:b>y</t:b><t:c>z</t:c></t:m></t:container>`

	// compileValidateSchema compiles the given schema and validates the given instance.
	// The schema MUST compile (a compile error fails the test) — substitution-group
	// blocking is a VALIDATION error (ErrValidationFailed), not a compile error.
	compileValidateSchema := func(t *testing.T, v xsd.Version, schema, inst string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		sc, err := xsd.NewCompiler().Version(v).Compile(t.Context(), doc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(inst))
		require.NoError(t, err)
		return xsd.NewValidator(sc).Validate(t.Context(), idoc)
	}
	compileValidate := func(t *testing.T, v xsd.Version, midBlock string) error {
		t.Helper()
		return compileValidateSchema(t, v, schemaFor(midBlock), instance)
	}

	// Deep-suffix schemas: the blocked method occurs LOWER in the derivation suffix
	// than the direct child step into the blocking intermediate. Checking only the
	// immediate child step into `mid` misses these — the whole derived..mid suffix
	// must be examined. All restrictions repeat the base model verbatim (a valid
	// pointless restriction) so the schema COMPILES and the only source of rejection is
	// the intermediate block. The 3-child instance below matches the Leaf model in every
	// case, so structural validity never masks the block outcome.
	deepInstance := `<t:container xmlns:t="urn:t"><t:m><t:a>x</t:a><t:b>y</t:b><t:c>z</t:c></t:m></t:container>`
	// Base <- Mid(block=extension) <- R(restriction) <- Leaf(extension): Leaf..Mid
	// includes Leaf's EXTENSION step, so Mid (blocks extension) blocks the member even
	// though the direct step into Mid (R's) is a restriction.
	extBelowRestriction := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
                  xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:complexType name="Base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="Mid" block="extension">
    <xs:complexContent><xs:extension base="t:Base">
      <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent><xs:restriction base="t:Mid">
      <xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence>
    </xs:restriction></xs:complexContent>
  </xs:complexType>
  <xs:complexType name="Leaf">
    <xs:complexContent><xs:extension base="t:R">
      <xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:element name="h" type="t:Base"/>
  <xs:element name="m" type="t:Leaf" substitutionGroup="t:h"/>
  <xs:element name="container"><xs:complexType><xs:sequence><xs:element ref="t:h"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`
	// Symmetric: Base <- Mid(block=restriction) <- R(extension) <- Leaf(restriction):
	// Leaf..Mid includes Leaf's RESTRICTION step, so Mid (blocks restriction) blocks the
	// member even though the direct step into Mid (R's) is an extension. R extends Mid
	// by adding only an OPTIONAL ATTRIBUTE, so R's content model equals Mid's and Leaf's
	// flat restriction is a one-level-nested shape the XSD 1.0 syntactic checker accepts.
	restrictionBelowExt := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
                  xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:complexType name="Base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="Mid" block="restriction">
    <xs:complexContent><xs:extension base="t:Base">
      <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent><xs:extension base="t:Mid">
      <xs:attribute name="k" type="xs:string" use="optional"/>
    </xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:complexType name="Leaf">
    <xs:complexContent><xs:restriction base="t:R">
      <xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence>
    </xs:restriction></xs:complexContent>
  </xs:complexType>
  <xs:element name="h" type="t:Base"/>
  <xs:element name="m" type="t:Leaf" substitutionGroup="t:h"/>
  <xs:element name="container"><xs:complexType><xs:sequence><xs:element ref="t:h"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`
	// Non-blocked multi-level control: Base <- Mid(block=extension) <- R(restriction)
	// <- Leaf(restriction). Leaf..Mid uses ONLY restriction steps, so Mid's
	// extension-block never fires and the member is admitted. Guards against
	// over-rejection by the whole-suffix walk.
	controlNoBlock := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
                  xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:complexType name="Base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="Mid" block="extension">
    <xs:complexContent><xs:extension base="t:Base">
      <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent><xs:restriction base="t:Mid">
      <xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence>
    </xs:restriction></xs:complexContent>
  </xs:complexType>
  <xs:complexType name="Leaf">
    <xs:complexContent><xs:restriction base="t:R">
      <xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence>
    </xs:restriction></xs:complexContent>
  </xs:complexType>
  <xs:element name="h" type="t:Base"/>
  <xs:element name="m" type="t:Leaf" substitutionGroup="t:h"/>
  <xs:element name="container"><xs:complexType><xs:sequence><xs:element ref="t:h"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`
	controlInstance := `<t:container xmlns:t="urn:t"><t:m><t:a>x</t:a><t:b>y</t:b></t:m></t:container>`

	for _, tc := range []struct {
		name string
		ver  xsd.Version
	}{{"xsd10", xsd.Version10}, {"xsd11", xsd.Version11}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("intermediate TYPE block=extension rejects member", func(t *testing.T) {
				require.ErrorIs(t, compileValidate(t, tc.ver, `block="extension"`), xsd.ErrValidationFailed)
			})
			t.Run("no intermediate TYPE block accepts member", func(t *testing.T) {
				require.NoError(t, compileValidate(t, tc.ver, ``))
			})
			t.Run("extension hidden below restriction into block=extension mid rejects member", func(t *testing.T) {
				require.ErrorIs(t, compileValidateSchema(t, tc.ver, extBelowRestriction, deepInstance), xsd.ErrValidationFailed)
			})
			t.Run("restriction hidden below extension into block=restriction mid rejects member", func(t *testing.T) {
				require.ErrorIs(t, compileValidateSchema(t, tc.ver, restrictionBelowExt, controlInstance), xsd.ErrValidationFailed)
			})
			t.Run("multi-level restriction-only suffix past block=extension mid accepts member", func(t *testing.T) {
				require.NoError(t, compileValidateSchema(t, tc.ver, controlNoBlock, controlInstance))
			})
		})
	}
}

// TestComplexTypeBlock_CTAAlternative covers site 2: an <xs:alternative> whose
// {type definition} is derived by extension from the element's declared type must be
// rejected at compile time when the declared type carries block="extension" (the
// effective {disallowed substitutions} unions the element block and the declared
// type's {prohibited substitutions}). xs:error alternatives are always permitted.
// XSD 1.1 only (conditional type assignment).
func TestComplexTypeBlock_CTAAlternative(t *testing.T) {
	t.Parallel()
	schemaFor := func(block string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
                  xmlns:t="urn:t" targetNamespace="urn:t"
                  elementFormDefault="qualified">
  <xs:complexType name="Base" ` + block + `>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:attribute name="k" type="xs:string"/>
  </xs:complexType>
  <xs:complexType name="Ext">
    <xs:complexContent>
      <xs:extension base="t:Base">
        <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="t:Base">
    <xs:alternative test="@k='x'" type="t:Ext"/>
  </xs:element>
</xs:schema>`
	}

	t.Run("declared TYPE block=extension rejects extension alternative", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, schemaFor(`block="extension"`))
		require.Error(t, cerr)
	})
	t.Run("no TYPE block accepts extension alternative", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, schemaFor(``))
		require.NoError(t, cerr)
	})
}

// TestComplexTypeBlock_ParticleRestriction covers site 3: a content-model
// restriction that redeclares a base element with a type reached from the base
// element's type by a derivation the BASE TYPE's {prohibited substitutions} blocks
// must be rejected (cvc-elt.4.3 / NameAndTypeOK). Here the derived redeclaration
// changes element x's type from XBase to a RESTRICTION XRes; when XBase carries
// block="restriction" the retyping is invalid, otherwise it is accepted. A
// restriction (not extension) retyping is used because NameAndTypeOK clause
// 3.2.5.2 disallows an EXTENSION retyping unconditionally (subset
// {extension, list, union}) — that case is invalid regardless of @block, so only a
// restriction retyping can exhibit the @block distinction at this site.
// Version-independent (elementRestrictsElement runs in both 1.0 and 1.1).
func TestComplexTypeBlock_ParticleRestriction(t *testing.T) {
	schemaFor := func(block string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
                  xmlns:t="urn:t" targetNamespace="urn:t"
                  elementFormDefault="qualified">
  <xs:complexType name="XBase" ` + block + `>
    <xs:sequence><xs:element name="p" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="XRes">
    <xs:complexContent>
      <xs:restriction base="t:XBase">
        <xs:sequence><xs:element name="p" type="xs:string"/></xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="ContBase">
    <xs:sequence><xs:element name="x" type="t:XBase"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="ContDer">
    <xs:complexContent>
      <xs:restriction base="t:ContBase">
        <xs:sequence><xs:element name="x" type="t:XRes"/></xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="t:ContDer"/>
</xs:schema>`
	}

	t.Run("xsd11", func(t *testing.T) {
		t.Run("base TYPE block=restriction rejects retyping restriction", func(t *testing.T) {
			_, _, cerr := compileV11(t, schemaFor(`block="restriction"`))
			require.Error(t, cerr)
		})
		t.Run("no base TYPE block accepts retyping restriction", func(t *testing.T) {
			_, _, cerr := compileV11(t, schemaFor(``))
			require.NoError(t, cerr)
		})
	})
	t.Run("xsd10", func(t *testing.T) {
		t.Run("base TYPE block=restriction rejects retyping restriction", func(t *testing.T) {
			_, cerr := compileV10(t, schemaFor(`block="restriction"`))
			require.Error(t, cerr)
		})
		t.Run("no base TYPE block accepts retyping restriction", func(t *testing.T) {
			_, cerr := compileV10(t, schemaFor(``))
			require.NoError(t, cerr)
		})
	})
}
