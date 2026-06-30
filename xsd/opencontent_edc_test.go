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
