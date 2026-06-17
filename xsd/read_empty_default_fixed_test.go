package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEmptyFixedConstraint checks that a fixed="" constraint is retained and
// enforced. An empty-string fixed value is a valid constraint: it requires the
// element to be empty, so non-empty content must be rejected. Previously the
// reader dropped fixed="" (treating present-but-empty as absent), so the
// constraint was silently ignored and any content was accepted.
func TestEmptyFixedConstraint(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" fixed=""/>
</xs:schema>`

	t.Run("accepts empty content", func(t *testing.T) {
		require.NoError(t, compileAndValidate(t, schemaXML, `<root/>`, nil))
	})

	t.Run("rejects non-empty content", func(t *testing.T) {
		var out string
		err := compileAndValidate(t, schemaXML, `<root>x</root>`, &out)
		require.Error(t, err)
		require.Contains(t, out, "does not match the fixed value constraint")
	})
}
