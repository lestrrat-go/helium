package xsd_test

import (
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestOccursHugeMinNullableBodyNoSpin verifies that validating an instance
// against a compositor (xs:sequence/xs:choice) with a very large minOccurs and an
// EMPTIABLE body does not spin the group-repetition loop ~minReps times. Such a
// minOccurs is satisfiable by padding with zero-width repetitions, so an empty
// instance is VALID and must be accepted in O(1), not by looping (a validation-
// time CPU DoS). A NON-nullable body with the same huge minOccurs against an
// instance that cannot satisfy it must still be REJECTED, and fail fast.
func TestOccursHugeMinNullableBodyNoSpin(t *testing.T) {
	t.Parallel()

	mustCompile := func(t *testing.T, s string) *xsd.Schema {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)
		return schema
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}
	validateQuickly := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		type result struct{ err error }
		done := make(chan result, 1)
		go func() { done <- result{validate(t, schema, instance)} }()
		select {
		case r := <-done:
			return r.err
		case <-time.After(3 * time.Second):
			t.Fatal("validation did not complete within 3s (group-repetition loop is spinning)")
			return nil
		}
	}

	// Emptiable body: the sequence's huge minOccurs is satisfied by empty reps, so
	// an empty instance is valid and must validate quickly (the overflowing
	// minOccurs clamps to a large int; without the zero-progress guard this spins).
	t.Run("emptiable body huge minOccurs overflow accepts empty fast", func(t *testing.T) {
		t.Parallel()
		schema := mustCompile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence minOccurs="99999999999999999999999999999" maxOccurs="unbounded">
      <xs:element name="a" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`)
		require.NoError(t, validateQuickly(t, schema, `<root/>`))
		require.NoError(t, validateQuickly(t, schema, `<root><a>x</a></root>`))
	})

	// A large but IN-RANGE minOccurs on an emptiable-body sequence is the
	// pre-existing latent spin (independent of the overflow clamp); it must also
	// validate an empty instance quickly.
	t.Run("emptiable body large in-range minOccurs accepts empty fast", func(t *testing.T) {
		t.Parallel()
		schema := mustCompile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence minOccurs="2000000000" maxOccurs="unbounded">
      <xs:element name="a" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`)
		require.NoError(t, validateQuickly(t, schema, `<root/>`))
	})

	// A NON-nullable body (required child) with a huge minOccurs cannot be
	// satisfied by an empty instance: still rejected, and fails fast on the finite
	// child count (the body fails on the first repetition, no spin).
	t.Run("non-nullable body huge minOccurs rejects empty fast", func(t *testing.T) {
		t.Parallel()
		schema := mustCompile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence minOccurs="99999999999999999999999999999" maxOccurs="unbounded">
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`)
		require.ErrorIs(t, validateQuickly(t, schema, `<root/>`), xsd.ErrValidationFailed)
	})
}
