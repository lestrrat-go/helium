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
