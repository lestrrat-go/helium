package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_InterleaveOpenChildEDC covers the gauntlet finding that an
// interleave-refined child moved into the open sub-sequence must still get the
// SAME dynamic wildcard Element-Declarations-Consistent (EDC) check as the ordinary
// wildcard-particle path: a child whose name collides with a same-named local
// element whose type is inconsistent with the wildcard's governing (global) type
// must be rejected.
func TestOpenContent_InterleaveOpenChildEDC(t *testing.T) {
	t.Parallel()

	t.Run("inconsistent local int vs global duration is rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:duration"/>
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		// First <e> is the local int (valid); the second is open content matched by
		// the lax ##local wildcard, lax-resolving to global e=duration. The EDC check
		// must reject it (local int vs governing duration inconsistent).
		require.Error(t, validateOC(t, schema, `<doc><e>1</e><e>PT1H</e></doc>`),
			"open-content child colliding with an inconsistent local declaration must be rejected")
	})

	t.Run("consistent local int and global int is accepted", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:int"/>
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc><e>1</e><e>2</e></doc>`),
			"a consistent local/global type must not be rejected by the EDC guard")
	})
}

// TestOpenContent_RestrictionDropsBaseLocal covers the gauntlet finding that a
// restriction must not DROP a base local element declaration and re-admit the same
// name through a lenient open-content wildcard (skip, or lax with no global
// declaration) that does not enforce the base's declared type.
func TestOpenContent_RestrictionDropsBaseLocal(t *testing.T) {
	t.Parallel()

	t.Run("dropped base local re-admitted by a skip wildcard is rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "dropping a base local element re-admitted by a skip wildcard must be rejected")
	})

	t.Run("dropped base local re-admitted by lax-with-no-global is rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="lax"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "dropping a base local element re-admitted by lax-with-no-global must be rejected")
	})

	t.Run("keeping the base local element is accepted", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a restriction that KEEPS the base local element is valid")
	})

	t.Run("dropped base global ref re-admitted by a skip wildcard is rejected", func(t *testing.T) {
		t.Parallel()
		// The base declares the element via ref="e" (global e=int). A skip wildcard
		// returns BEFORE any global-declaration lookup, so it never enforces the
		// global's type — dropping the ref and re-admitting via skip is unsound.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:int"/>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element ref="e" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "dropping a base global ref re-admitted by a skip wildcard must be rejected")
	})

	t.Run("dropped base global ref re-admitted by lax is accepted (global enforced via dynamic EDC)", func(t *testing.T) {
		t.Parallel()
		// A ref implies a global declaration, so a lax wildcard always resolves and
		// validates the global; the dynamic EDC enforces consistency at validation.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:int"/>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element ref="e" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="lax"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a lax wildcard re-admitting a global ref (dynamic EDC enforces) must not be rejected at compile")
	})

	t.Run("dropped base local re-admitted by strict wildcard with consistent global is accepted", func(t *testing.T) {
		t.Parallel()
		// A strict wildcard resolves the name to the global declaration and the
		// dynamic EDC enforces type consistency at validation, so dropping the local
		// is a valid restriction (not a compile-time error).
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:int"/>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a strict wildcard re-admitting the name (dynamic EDC enforces) must not be rejected at compile")
	})

	t.Run("dropped base head re-admitting a substitution member via skip is rejected", func(t *testing.T) {
		t.Parallel()
		// The base declares ref="h"; m (xs:int) is substitutable for h. Runtime
		// matching admits <m> for the ref via substitution. The derived drops the ref
		// and a skip wildcard that EXCLUDES the head name h via notQName="h" still
		// re-admits the member m without enforcing its type — unsound. (Excluding the
		// head does NOT exclude the member.)
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:integer"/>
  <xs:element name="m" type="xs:int" substitutionGroup="h"/>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element ref="h" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip" notQName="h"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "dropping a base head whose substitution member is re-admitted by skip must be rejected")
	})

	t.Run("dropped base head with notQName excluding head and members is accepted", func(t *testing.T) {
		t.Parallel()
		// The derived skip wildcard EXCLUDES both h and its member m via notQName, so
		// it re-admits neither — the derived accepts fewer, still a valid subset.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:integer"/>
  <xs:element name="m" type="xs:int" substitutionGroup="h"/>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element ref="h" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip" notQName="h m"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a derived wildcard excluding the head and members via notQName must not be rejected")
	})

	t.Run("dropped base head re-admitting a substitution member via strict is accepted", func(t *testing.T) {
		t.Parallel()
		// A strict wildcard resolves m to its global declaration; the dynamic EDC
		// enforces consistency at validation, so dropping the ref is valid.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:integer"/>
  <xs:element name="m" type="xs:int" substitutionGroup="h"/>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element ref="h" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a strict wildcard re-admitting a substitution member (dynamic EDC enforces) must not be rejected")
	})
}

// TestOpenContent_RestrictionNarrowsKeptName covers the gauntlet finding that the
// kept-name exemption must require OCCURRENCE COVERAGE in interleave mode: a
// restriction that KEEPS a base element but NARROWS its maxOccurs while an
// unenforcing interleave open-content wildcard re-admits the name lets the EXCESS
// children spill into open content (where the base type is not enforced).
func TestOpenContent_RestrictionNarrowsKeptName(t *testing.T) {
	t.Parallel()

	t.Run("interleave narrows kept element maxOccurs with skip OC is rejected", func(t *testing.T) {
		t.Parallel()
		// Base e:int maxOccurs="unbounded"; derived narrows e to maxOccurs="1". A 2nd
		// <e> spills into the interleave skip open content (unenforced), but the base
		// validates both <e> as the int element → not a subset.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" maxOccurs="1"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "narrowing a kept element's maxOccurs while skip OC re-admits it must be rejected")
	})

	t.Run("interleave keeps element maxOccurs covering the base is accepted", func(t *testing.T) {
		t.Parallel()
		// Derived keeps e:int maxOccurs="unbounded" (covers the base) — no excess
		// spills, every <e> is enforced as the int element.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" maxOccurs="unbounded"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "keeping the base element's maxOccurs coverage must not be rejected")
	})

	t.Run("interleave narrows kept element but OC is strict-with-global is accepted", func(t *testing.T) {
		t.Parallel()
		// A strict wildcard resolves the spilled <e> to the global e and the dynamic
		// EDC enforces consistency at validation, so the narrowing is sound.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:int"/>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" maxOccurs="1"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "narrowing with a strict wildcard (dynamic EDC enforces) must not be rejected")
	})

	t.Run("interleave narrows kept element but OC notQName-excludes it is accepted", func(t *testing.T) {
		t.Parallel()
		// The derived wildcard EXCLUDES e via notQName, so the excess <e> is NOT
		// re-admitted as open content — the derived rejects it as a misplaced element,
		// preserving the subset relation.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip" notQName="e"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" maxOccurs="1"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "narrowing with a wildcard that excludes the name via notQName must not be rejected")
	})

	t.Run("suffix narrows kept element with skip OC is accepted", func(t *testing.T) {
		t.Parallel()
		// In SUFFIX mode a trailing child whose name is declared is rejected as
		// misplaced (never spilled into open content), so narrowing a kept element is
		// safe even with an unenforcing wildcard.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" maxOccurs="1"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "suffix mode rejects misplaced kept-name children, so narrowing is safe")
	})
}

// TestOpenContent_RestrictionDropsNonEmittingBaseElement covers the gauntlet
// finding that the drop guard must IGNORE a base element whose effective maxOccurs
// is 0 (a prohibited particle/group emits nothing): the base admits that name only
// through its open content, not the element, so dropping it in the derived while
// keeping the same open content is a VALID restriction (false-reject otherwise).
func TestOpenContent_RestrictionDropsNonEmittingBaseElement(t *testing.T) {
	t.Parallel()

	t.Run("base element under a maxOccurs=0 group is not protected (dropped + same OC accepts)", func(t *testing.T) {
		t.Parallel()
		// e is inside a maxOccurs="0" group → non-emitting; B admits e only via the
		// skip open content. R drops it (empty model) and keeps the same open content,
		// so both admit e only via open content — a valid restriction.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:sequence minOccurs="0" maxOccurs="0">
        <xs:element name="e" type="xs:int"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a base element with effective maxOccurs=0 must not be protected by the drop guard")
	})

	t.Run("genuine dropped emitting element is still rejected", func(t *testing.T) {
		t.Parallel()
		// Control: e is emitting (maxOccurs=1) in the base, so dropping it while a skip
		// wildcard re-admits it remains unsound and must be rejected.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "dropping a genuine emitting base element re-admitted by skip must still be rejected")
	})
}

// TestOpenContent_RestrictionDropsSuffixOrdered covers the gauntlet finding that a
// SUFFIX-mode base imposes an ORDERING constraint (a declared element must appear in
// the prefix region, not after open content) that a restriction loses by dropping
// the element — so a dropped base-declared name re-admitted by the derived suffix
// open content must be rejected for ANY processContents, even strict /
// lax-with-global where the dynamic EDC enforces the TYPE but not the ORDER.
func TestOpenContent_RestrictionDropsSuffixOrdered(t *testing.T) {
	t.Parallel()

	t.Run("suffix base drops global e re-admitted by strict OC is rejected (order lost)", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:int"/>
  <xs:complexType name="B">
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element ref="e" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "a suffix base dropping a declared element re-admitted by its open content must be rejected even under strict")
	})

	t.Run("suffix base drops local e re-admitted by skip OC is rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "a suffix base dropping a declared element re-admitted by skip must be rejected")
	})

	t.Run("interleave base drops global e re-admitted by strict OC is accepted", func(t *testing.T) {
		t.Parallel()
		// Interleave imposes no ordering, so strict type enforcement (dynamic EDC) is
		// sufficient — dropping e is a valid restriction.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:int"/>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element ref="e" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "an interleave base dropping a declared element with strict OC (dynamic EDC enforces) must be accepted")
	})

	t.Run("suffix base keeps e is accepted", func(t *testing.T) {
		t.Parallel()
		// The derived KEEPS e, so the derived suffix model still order-constrains it —
		// the ordering is preserved.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:int"/>
  <xs:complexType name="B">
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element ref="e" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element ref="e" minOccurs="0"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a suffix base that KEEPS the declared element preserves the ordering")
	})
}

// TestOpenContent_NonEmittingElementConsistency covers the gauntlet findings that a
// NON-EMITTING element (effective maxOccurs 0 — its particle or an ancestor group is
// maxOccurs="0") emits nothing, so it must be ignored consistently: in
// baseModelAdmitsOpenContent (it does not disqualify the wildcard-only-base shape)
// and in suffix validation (a trailing child of that name is open content, not a
// misplaced declared element).
func TestOpenContent_NonEmittingElementConsistency(t *testing.T) {
	t.Parallel()

	t.Run("baseModelAdmitsOpenContent accepts W* plus a prohibited element", func(t *testing.T) {
		t.Parallel()
		// Base = unbounded wildcard W* PLUS a maxOccurs=0 (prohibited) element e. e
		// emits nothing, so the base is effectively wildcard-only; the empty-model
		// restriction re-expressing W as open content is valid.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B"><xs:sequence>
    <xs:any namespace="http://open.com/" processContents="lax" minOccurs="0" maxOccurs="unbounded"/>
    <xs:sequence minOccurs="0" maxOccurs="0">
      <xs:element name="e" type="xs:int"/>
    </xs:sequence>
  </xs:sequence></xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a prohibited (maxOccurs=0) element must not disqualify the wildcard-only-base shape")
	})

	t.Run("suffix validation treats a non-emitting declared name as open content", func(t *testing.T) {
		t.Parallel()
		// e is non-emitting (inside a maxOccurs=0 group); a trailing <e> matching the
		// skip wildcard is open content, not a misplaced declared element.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:sequence minOccurs="0" maxOccurs="0">
        <xs:element name="e" type="xs:int"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc><a>x</a><e>anything</e></doc>`),
			"a trailing child whose only declaration is non-emitting must validate as open content")
	})

	t.Run("suffix validation still rejects an EMITTING declared name in the suffix region", func(t *testing.T) {
		t.Parallel()
		// Control: an emitting declared name appearing after open content is still a
		// misplaced declared element.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix"><xs:any namespace="http://open.com/" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		require.Error(t, validateOC(t, schema, `<doc><a>x</a><extra xmlns="http://open.com/"/><a>y</a></doc>`),
			"an emitting declared name after open content must still be rejected as misplaced")
	})
}

// TestOpenContent_DirectProhibitedParticleRuntime covers the gauntlet finding that
// a DIRECT minOccurs=0 maxOccurs=0 (prohibited) element particle must not consume a
// child in the XSD 1.1 open-content declared-content matcher: the runtime matcher
// otherwise grabs a matching child once before the maxOccurs check, so the child is
// validated against the prohibited element's type instead of falling through to open
// content. The ordinary (no open content) matcher is unchanged.
func TestOpenContent_DirectProhibitedParticleRuntime(t *testing.T) {
	t.Parallel()

	t.Run("suffix direct prohibited element leaves child for open content", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:element name="e" type="xs:int" minOccurs="0" maxOccurs="0"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc><a>x</a><e>anything</e></doc>`),
			"a trailing <e> whose only declaration is a prohibited particle must validate as open content")
	})

	t.Run("interleave direct prohibited element leaves child for open content", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:element name="e" type="xs:int" minOccurs="0" maxOccurs="0"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc><a>x</a><e>anything</e></doc>`),
			"a <e> whose only declaration is a prohibited particle must validate as open content (interleave)")
	})

	t.Run("without open content a present prohibited element is still rejected", func(t *testing.T) {
		t.Parallel()
		// Control: the ordinary (no open content) matcher is UNCHANGED — it still
		// matches a present maxOccurs=0 element against its declared type (here the
		// prohibited e:int validates the child as an int, so invalid int content is
		// rejected). This pins the ordinary-path behavior so the open-content prune
		// does not leak into it.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:element name="e" type="xs:int" minOccurs="0" maxOccurs="0"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		require.Error(t, validateOC(t, schema, `<doc><a>x</a><e>notanint</e></doc>`),
			"the ordinary matcher still validates a present prohibited element against its type (unchanged)")
	})
}

// TestOpenContent_AllNonEmittingGroup covers the gauntlet finding that pruning a
// group down to ZERO members (e.g. an xs:choice all of whose branches are
// maxOccurs=0) must drop the group rather than leave an empty group whose matcher
// reports "missing" — the all-prohibited declared model emits nothing, so every
// child routes to open content.
func TestOpenContent_AllNonEmittingGroup(t *testing.T) {
	t.Parallel()

	t.Run("suffix choice with all branches prohibited routes child to open content", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:choice>
      <xs:element name="e" type="xs:int" minOccurs="0" maxOccurs="0"/>
      <xs:element name="f" type="xs:int" minOccurs="0" maxOccurs="0"/>
    </xs:choice>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc><e>anything</e></doc>`),
			"a choice with all branches prohibited emits nothing; a branch-named child is open content (suffix)")
	})

	t.Run("interleave choice with all branches prohibited routes child to open content", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:choice>
      <xs:element name="e" type="xs:int" minOccurs="0" maxOccurs="0"/>
      <xs:element name="f" type="xs:int" minOccurs="0" maxOccurs="0"/>
    </xs:choice>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc><e>anything</e></doc>`),
			"a choice with all branches prohibited emits nothing; a branch-named child is open content (interleave)")
	})

	t.Run("sequence containing only a maxOccurs=0 choice routes child to open content", func(t *testing.T) {
		t.Parallel()
		// Nested empties propagate: the whole choice is maxOccurs=0 (dropped), which
		// empties the sequence (dropped), leaving a fully-empty declared model.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:choice minOccurs="0" maxOccurs="0">
        <xs:element name="e" type="xs:int"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc><e>anything</e></doc>`),
			"a fully non-emitting declared model means only open content applies")
	})

	t.Run("choice with at least one emitting branch still matches normally", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:choice>
      <xs:element name="e" type="xs:int" minOccurs="0" maxOccurs="0"/>
      <xs:element name="g" type="xs:string"/>
    </xs:choice>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc><g>x</g></doc>`),
			"an emitting branch still matches normally")
		require.NoError(t, validateOC(t, schema, `<doc><g>x</g><e>anything</e></doc>`),
			"the prohibited branch's name routes to open content alongside the satisfied emitting branch")
		require.Error(t, validateOC(t, schema, `<doc><e>anything</e></doc>`),
			"the choice still requires its emitting branch g (the prohibited branch does not make it optional)")
	})
}

// TestOpenContent_DefinedSiblingWildcard covers the gauntlet finding that an
// open-content wildcard carrying notQName="##definedSibling" must have its
// SiblingNames resolved (the element names declared in the same content model), so
// the runtime exclusion applies: a declared sibling name cannot be moved into open
// content even when the declared model cannot place an extra occurrence.
func TestOpenContent_DefinedSiblingWildcard(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any processContents="skip" notQName="##definedSibling"/></xs:openContent>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`

	t.Run("a defined sibling is excluded from open content", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validateOC(t, schema, `<doc><a>x</a><a>y</a></doc>`),
			"a second <a> (a defined sibling) must not be accepted as open content")
	})

	t.Run("a non-sibling child is still admitted as open content", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateOC(t, schema, `<doc><a>x</a><other/></doc>`),
			"a non-sibling name is still open content")
	})
}

// TestOpenContent_DefinedSiblingUnion covers the gauntlet finding that the
// ##definedSibling SiblingNames must be MATERIALIZED on the open-content wildcard
// BEFORE the extension union (cos-aw-union): in a union where one operand excludes
// the sibling via notQName and the other via ##definedSibling, the union must
// retain that sibling as a finite exclusion even though the live marker is folded
// away — otherwise the sibling exclusion vanishes and an extra declared-sibling
// child is wrongly accepted as open content.
func TestOpenContent_DefinedSiblingUnion(t *testing.T) {
	t.Parallel()
	// Base B excludes `a` from its open content via explicit notQName="a"; derived R
	// (an extension) excludes `a` via notQName="##definedSibling". Their union must
	// keep `a` excluded.
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip" notQName="a"/></xs:openContent>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent><xs:extension base="B">
      <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip" notQName="##definedSibling"/></xs:openContent>
      <xs:sequence/>
    </xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`

	t.Run("a sibling excluded by both union operands stays excluded", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validateOC(t, schema, `<doc><a>x</a><a>y</a></doc>`),
			"a second <a> (excluded by both union operands) must not be accepted as open content")
	})

	t.Run("a non-sibling child is still admitted by the union", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateOC(t, schema, `<doc><a>x</a><other/></doc>`),
			"a non-sibling name is still open content under the union")
	})
}

// TestOpenContent_InterleaveDeclaredWildcard covers the gauntlet finding that the
// interleave partition must account for a DECLARED wildcard particle, not just
// declared element NAMES: a child that matches a declared xs:any (in the content
// model) must be assignable to the declared partition to satisfy that wildcard's
// occurrence, instead of being forced into open content.
func TestOpenContent_InterleaveDeclaredWildcard(t *testing.T) {
	t.Parallel()
	// Open content ##any; declared model is a single REQUIRED xs:any limited to
	// urn:a, maxOccurs=1.
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:any namespace="urn:a" processContents="skip" minOccurs="1" maxOccurs="1"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`

	t.Run("a child satisfying the declared wildcard is accepted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateOC(t, schema, `<doc><x xmlns="urn:a"/></doc>`),
			"a child matching the declared wildcard satisfies it (empty open partition)")
	})

	t.Run("a required declared wildcard with no matching child is rejected", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validateOC(t, schema, `<doc><y xmlns="urn:b"/></doc>`),
			"a child not matching the declared wildcard goes to open; the required declared wildcard is unsatisfied")
		require.Error(t, validateOC(t, schema, `<doc/>`),
			"no children at all leaves the required declared wildcard unsatisfied")
	})

	t.Run("a child matching only the open wildcard still goes to open", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateOC(t, schema, `<doc><x xmlns="urn:a"/><y xmlns="urn:b"/></doc>`),
			"the urn:a child satisfies the declared wildcard; the urn:b child is open content")
	})

	t.Run("extras beyond the declared wildcard maxOccurs go to open", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateOC(t, schema, `<doc><x xmlns="urn:a"/><z xmlns="urn:a"/></doc>`),
			"one urn:a satisfies the maxOccurs=1 declared wildcard; the extra urn:a spills to open")
	})

	t.Run("suffix mode already handles a declared wildcard in the prefix", func(t *testing.T) {
		t.Parallel()
		const suffixSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:any namespace="urn:a" processContents="skip" minOccurs="1" maxOccurs="1"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, suffixSchema, `<doc><x xmlns="urn:a"/></doc>`),
			"suffix matches the declared wildcard as the prefix")
	})
}

// TestOpenContent_RestrictionDropsBaseWildcard covers the gauntlet finding that the
// restriction drop-guard must protect dropped base DECLARED WILDCARD particles, not
// just element declarations: now that a declared wildcard wins attribution in the
// interleave partition, dropping it and re-admitting its namespace through a WEAKER
// open-content wildcard is unsound.
func TestOpenContent_RestrictionDropsBaseWildcard(t *testing.T) {
	t.Parallel()

	t.Run("dropped strict declared wildcard re-admitted by skip open is rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "dropping a strict declared wildcard re-admitted by a weaker (skip) open wildcard must be rejected")
	})

	t.Run("dropped skip declared wildcard re-admitted by skip open is accepted", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="skip" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "dropping a skip declared wildcard re-admitted by an equal (skip) open wildcard is sound")
	})

	t.Run("keeping the declared wildcard is accepted", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a restriction that KEEPS the declared wildcard (covering namespace + pc + occurrence) is valid")
	})

	t.Run("suffix base dropping a declared wildcard is rejected (order loss)", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="suffix"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="suffix"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "a suffix base dropping a declared wildcard re-admitted by its open content must be rejected (order loss)")
	})
}

// TestOpenContent_DropsBaseWildcardCoverage covers the three follow-up gauntlet
// findings on checkOpenContentDropsBaseWildcard: (3) the kept/narrow spill check
// must first require namespace intersection with the derived open wildcard; (1)
// suffix kept wildcards are NOT blanket-exempt (suffix runtime spills a leftover
// declared-wildcard match to open content); (2) occurrence coverage must be
// AGGREGATE across overlapping base wildcards and across multiple derived wildcards.
func TestOpenContent_DropsBaseWildcardCoverage(t *testing.T) {
	t.Parallel()

	t.Run("F3 narrowed declared wildcard whose namespace the open content EXCLUDES is accepted", func(t *testing.T) {
		t.Parallel()
		// The open content only admits urn:b, so the narrowed urn:a declared wildcard's
		// excess cannot spill into open content (the derived model rejects it).
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="urn:b" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="urn:b" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="1"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "an open content that excludes the declared wildcard's namespace cannot receive its excess; must be accepted")
	})

	t.Run("F1 suffix narrowing a strict declared wildcard with weaker suffix open is rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="suffix"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="suffix"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="1"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "suffix narrowing a strict declared wildcard while a weaker suffix open re-admits its namespace spills excess unenforced")
	})

	t.Run("F1 suffix keeping a fully-covering declared wildcard is accepted", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="suffix"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="suffix"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a suffix kept declared wildcard with full occurrence coverage is valid")
	})

	// NOTE: the multi-wildcard AGGREGATE scenario from finding 2 (two overlapping
	// base/derived wildcards) is not constructible as a valid schema — UPA
	// (cos-nonambig) rejects two overlapping declared wildcards in a content model —
	// so the operative logic is single-wildcard occurrence coverage (occursCovers).
	// The aggregate accounting is implemented defensively; these two cases exercise
	// the coverage path that the aggregate sums reduce to in the single-wildcard case.

	t.Run("F2 interleave narrowing a strict wildcard maxOccurs below base spills excess to weaker open (reject)", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="1"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "narrowing a strict wildcard's maxOccurs below the base while a weaker open re-admits its namespace spills excess unenforced")
	})

	t.Run("F2 interleave keeping a strict wildcard's full occurrence is accepted", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="2"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:any namespace="urn:a" processContents="strict" minOccurs="0" maxOccurs="2"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a kept strict wildcard with full occurrence coverage is valid")
	})
}
