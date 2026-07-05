package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestContentBacktrackOccurrencePartition verifies the bounded occurrence
// backtracking recovers occurrence-partition-ambiguous but UPA-clean content
// models the greedy matcher would otherwise false-reject (W3C ctZ006/ctZ008/
// ctZ009). The valid instance must validate; a genuinely short one must fail.
func TestContentBacktrackOccurrencePartition(t *testing.T) {
	t.Parallel()

	// sequence minOccurs=2 over a repeatable inner element: the correct partition
	// is one inner element per outer repetition, which greedy over-consumes.
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence minOccurs="2" maxOccurs="unbounded">
      <xs:element name="a" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`

	t.Run("valid partition accepted (xsd10)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateBtInstance(t, xsd.Version10, schema, `<root><a/><a/></root>`))
	})
	t.Run("valid partition accepted (xsd11)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root><a/><a/></root>`))
	})
	t.Run("single occurrence rejected", func(t *testing.T) {
		t.Parallel()
		// One <a> cannot satisfy the outer minOccurs=2.
		require.Error(t, validateBtInstance(t, xsd.Version10, schema, `<root><a/></root>`))
	})
}

// TestContentBacktrackChoiceElementCommit verifies the backtracker honors the
// XSD 1.1 element-over-wildcard COMMIT-NO-FALLBACK rule: a choice whose
// element-first branch matches the current child but then fails structurally
// must NOT fall back to a wildcard branch. Accepting the partial instance via
// the wildcard would be an over-acceptance the greedy matcher (correctly)
// rejects.
func TestContentBacktrackChoiceElementCommit(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r"><xs:complexType>
    <xs:choice>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="b" type="xs:int"/>
      </xs:sequence>
      <xs:any processContents="skip"/>
    </xs:choice>
  </xs:complexType></xs:element>
</xs:schema>`

	t.Run("partial element-first branch rejected (no wildcard fallback)", func(t *testing.T) {
		t.Parallel()
		// <a> alone commits to the sequence branch (a is element-first) which then
		// needs <b>; the commit rule forbids falling back to the wildcard branch.
		err := validateBtInstance(t, xsd.Version11, schema, `<r><a>1</a></r>`)
		require.Error(t, err)
	})
	t.Run("full sequence branch accepted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<r><a>1</a><b>2</b></r>`))
	})
	t.Run("wildcard branch accepted when not element-first", func(t *testing.T) {
		t.Parallel()
		// <c> is not element-first for any branch, so the skip wildcard branch matches.
		require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<r><c/></r>`))
	})
}

// TestContentBacktrackSequenceWildcardCommit verifies the wildcard-free gate: a
// content model containing an xs:any is NOT handled by the backtracker, so the
// greedy matcher's (precedence-aware) verdict stands. Here a repeating sequence
// with a leading optional element and a trailing skip wildcard must reject an
// instance where the wildcard would have to consume a child the element is
// element-first-consumer for (element-over-wildcard reservation).
func TestContentBacktrackSequenceWildcardCommit(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence minOccurs="2" maxOccurs="2">
      <xs:element name="a" type="xs:int" minOccurs="0"/>
      <xs:any processContents="skip"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`

	t.Run("wildcard cannot steal element-first child", func(t *testing.T) {
		t.Parallel()
		// <a>1</a><x/> is only two children for a sequence that must repeat twice
		// with a required wildcard each rep. Greedy (element-over-wildcard
		// reservation) rejects; the backtracker must not run (model has a wildcard).
		require.Error(t, validateBtInstance(t, xsd.Version11, schema, `<root><a>1</a><x/></root>`))
	})
}

// TestContentBacktrackSameNameDistinctDecl verifies the name→declaration
// unambiguity envelope: a UPA-clean model with two DISTINCT same-name local
// element declarations differing in nillable must NOT engage the backtracker,
// because its first-name-match attribution would validate BOTH children against
// the FIRST declaration and mask the second's constraint violation. A leading
// occurrence-ambiguous group forces the greedy matcher to fail, so without the
// envelope gate the fallback would run and over-accept.
func TestContentBacktrackSameNameDistinctDecl(t *testing.T) {
	t.Parallel()

	// The leading sequence{2,2}(x+) is occurrence-partition-ambiguous (greedy
	// over-consumes both <x> in the first rep and fails), forcing the fallback.
	// The two <a> declarations share a name but differ in nillable.
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence>
      <xs:sequence minOccurs="2" maxOccurs="2">
        <xs:element name="x" type="xs:string" maxOccurs="unbounded"/>
      </xs:sequence>
      <xs:element name="a" type="xs:string" nillable="true"/>
      <xs:element name="a" type="xs:string" nillable="false"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`

	t.Run("second same-name decl constraint enforced (no first-match masking)", func(t *testing.T) {
		t.Parallel()
		// The second <a> is nillable=false, so xsi:nil="true" on it is invalid.
		// First-name-match would validate it against the first (nillable=true) decl.
		const inst = `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><x/><x/><a>ok</a><a xsi:nil="true"/></root>`
		require.Error(t, validateBtInstance(t, xsd.Version11, schema, inst))
	})
}

// TestContentBacktrackOptionalAll exercises btReachAll's optional-all skip: an
// OPTIONAL xs:all group (minOccurs=0) with a required member admits the empty
// sequence (the group is skipped, consuming no children) but still rejects a
// partial match where the group is entered but a required member is absent. The
// empty-instance case routes through the backtracker (tryMatchAll reports the
// required member missing without honoring minOccurs=0), so btReachAll's
// zero-consumption skip endpoint must make it valid — while never admitting a
// partial match.
func TestContentBacktrackOptionalAll(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all minOccurs="0">
      <xs:element name="a" type="xs:string"/>
      <xs:element name="b" type="xs:string"/>
    </xs:all>
  </xs:complexType></xs:element>
</xs:schema>`

	t.Run("empty instance skips the optional all", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root/>`))
	})
	t.Run("full all match accepted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root><a>1</a><b>2</b></root>`))
	})
	t.Run("partial all match rejected (no skip masking a required member)", func(t *testing.T) {
		t.Parallel()
		// The group is entered (<a> present) but required <b> is absent; skipping
		// the optional group cannot mask this (there is no following particle to
		// consume the leftover <a>).
		require.Error(t, validateBtInstance(t, xsd.Version11, schema, `<root><a>1</a></root>`))
	})
}

// TestContentBacktrackProhibitedParticle verifies that a maxOccurs=0 (PROHIBITED)
// particle never consumes a child in the backtracker: the reachability automaton
// operates on a non-emitting-pruned model (pruneNonEmittingParticles), so a
// prohibited element leaf cannot route a child through and mask an invalid
// instance. The leading occurrence-ambiguous sequence forces the greedy matcher to
// fail, engaging the backtracker.
func TestContentBacktrackProhibitedParticle(t *testing.T) {
	t.Parallel()

	t.Run("prohibited element leaf cannot consume a child", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence>
      <xs:sequence minOccurs="2" maxOccurs="2">
        <xs:element name="x" type="xs:string" maxOccurs="unbounded"/>
      </xs:sequence>
      <xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="0"/>
      <xs:element name="b" type="xs:string"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		// <a/> is prohibited (maxOccurs=0), so routing it through would be an
		// over-acceptance. The instance is invalid.
		require.Error(t, validateBtInstance(t, xsd.Version11, schema, `<root><x/><x/><a/><b/></root>`))
		// The same model with the prohibited element absent is valid (x|x partition,
		// then b) — confirms pruning does not over-reject.
		require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root><x/><x/><b/></root>`))
	})

	t.Run("prohibited xs:all member cannot consume a child", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all>
      <xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="0"/>
      <xs:element name="b" type="xs:string"/>
    </xs:all>
  </xs:complexType></xs:element>
</xs:schema>`
		// <a/> is prohibited; the instance is invalid.
		require.Error(t, validateBtInstance(t, xsd.Version11, schema, `<root><a>1</a><b>2</b></root>`))
		// b alone is valid.
		require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root><b>2</b></root>`))
	})
}

// TestContentBacktrackProhibitedChoiceBranch verifies a maxOccurs=0 particle
// inside an xs:choice is an EMPTY (ε) branch — it keeps the choice NULLABLE — so
// the reachability automaton gives it skip-only reach WITHOUT dropping the empty
// branch (which pruning the particle would). This is the false-reject the
// skip-only-in-automaton approach fixes (versus pruning, which would turn a
// nullable choice non-nullable).
func TestContentBacktrackProhibitedChoiceBranch(t *testing.T) {
	t.Parallel()

	t.Run("prohibited element as choice branch keeps choice nullable", func(t *testing.T) {
		t.Parallel()
		// The leading sequence{2,2}(x+) is occurrence-ambiguous (greedy fails),
		// forcing the fallback; the trailing choice must match ε via its prohibited
		// a{0,0} branch so <x/><x/> validates.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence>
      <xs:sequence minOccurs="2" maxOccurs="2">
        <xs:element name="x" type="xs:string" maxOccurs="unbounded"/>
      </xs:sequence>
      <xs:choice>
        <xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="0"/>
        <xs:element name="b" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		// x|x then the EMPTY choice branch (a{0,0} matches ε) — valid.
		require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root><x/><x/></root>`))
		// x|x then the b branch — valid.
		require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root><x/><x/><b>1</b></root>`))
		// <a> is prohibited, so it cannot satisfy the choice — invalid.
		require.Error(t, validateBtInstance(t, xsd.Version11, schema, `<root><x/><x/><a>1</a></root>`))
	})

	t.Run("prohibited group as choice branch keeps choice nullable", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence>
      <xs:sequence minOccurs="2" maxOccurs="2">
        <xs:element name="x" type="xs:string" maxOccurs="unbounded"/>
      </xs:sequence>
      <xs:choice>
        <xs:sequence minOccurs="0" maxOccurs="0">
          <xs:element name="a" type="xs:string"/>
        </xs:sequence>
        <xs:element name="b" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		// The prohibited (maxOccurs=0) sequence branch is an empty branch, so the
		// choice is nullable and <x/><x/> validates.
		require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root><x/><x/></root>`))
	})
}

func TestContentBacktrackProhibitedWildcard(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence>
      <xs:sequence minOccurs="2" maxOccurs="2">
        <xs:element name="x" type="xs:string" maxOccurs="unbounded"/>
      </xs:sequence>
      <xs:any processContents="skip" minOccurs="0" maxOccurs="0"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`

	require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root><x/><x/></root>`))
	require.Error(t, validateBtInstance(t, xsd.Version11, schema, `<root><x/><x/><y/></root>`))
}

func TestContentBacktrackAbstractOnlySubstitutionMember(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m" type="xs:string" abstract="true" substitutionGroup="h"/>
  <xs:element name="root"><xs:complexType>
    <xs:sequence minOccurs="2" maxOccurs="2">
      <xs:element ref="h" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`

	require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root><h/><h/></root>`))
}

func TestContentBacktrackSuffixOpenContentPrefix(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:openContent mode="suffix">
      <xs:any namespace="urn:o" processContents="skip"/>
    </xs:openContent>
    <xs:sequence minOccurs="2" maxOccurs="2">
      <xs:element name="a" type="xs:string" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`

	require.NoError(t, validateBtInstance(t, xsd.Version11, schema, `<root xmlns:o="urn:o"><a/><a/><o:x/></root>`))
	require.Error(t, validateBtInstance(t, xsd.Version11, schema, `<root xmlns:o="urn:o"><a/><o:x/><a/></root>`))
}

// TestGreedyProhibitedParticle verifies the GREEDY matcher (the common path, no
// backtracking) never lets a maxOccurs=0 (prohibited) element particle consume a
// child, in BOTH XSD 1.0 and 1.1 — matchElementParticle/tryMatchElementParticle
// enforce MaxOccurs before consuming, and matchAll10/tryMatchAll10 exclude a
// maxOccurs=0 member from the by-name map.
func TestGreedyProhibitedParticle(t *testing.T) {
	t.Parallel()

	t.Run("prohibited element in a sequence rejects a present child", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence>
      <xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="0"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			require.Error(t, validateBtInstance(t, v, schema, `<root><a/></root>`),
				"prohibited <a> must reject (version %v)", v)
			require.NoError(t, validateBtInstance(t, v, schema, `<root/>`),
				"empty content is valid (version %v)", v)
		}
	})

	t.Run("prohibited element after a sibling rejects a present child", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence>
      <xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="0"/>
      <xs:element name="b" type="xs:string"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			require.Error(t, validateBtInstance(t, v, schema, `<root><a/><b>1</b></root>`),
				"prohibited <a> must reject (version %v)", v)
			require.NoError(t, validateBtInstance(t, v, schema, `<root><b>1</b></root>`),
				"b alone is valid (version %v)", v)
		}
	})

	t.Run("prohibited wildcard in a sequence rejects a present child", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence>
      <xs:any processContents="skip" minOccurs="0" maxOccurs="0"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			require.Error(t, validateBtInstance(t, v, schema, `<root><child/></root>`),
				"prohibited wildcard must reject a present child (version %v)", v)
			require.NoError(t, validateBtInstance(t, v, schema, `<root/>`),
				"empty content is valid (version %v)", v)
		}
	})

	t.Run("prohibited xs:all member rejects a present child", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all>
      <xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="0"/>
      <xs:element name="b" type="xs:string"/>
    </xs:all>
  </xs:complexType></xs:element>
</xs:schema>`
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			require.Error(t, validateBtInstance(t, v, schema, `<root><a/><b>1</b></root>`),
				"prohibited all member <a> must reject (version %v)", v)
			require.NoError(t, validateBtInstance(t, v, schema, `<root><b>1</b></root>`),
				"b alone is valid (version %v)", v)
		}
	})
}

// validateBtInstance compiles schemaXML at the given version (which must be valid)
// and validates instanceXML, returning the validation error (nil when valid).
func validateBtInstance(t *testing.T, version xsd.Version, schemaXML, instanceXML string) error {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, cerr := xsd.NewCompiler().Version(version).Compile(t.Context(), doc)
	require.NoError(t, cerr, "schema must compile")
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)
	return xsd.NewValidator(schema).Validate(t.Context(), idoc)
}
