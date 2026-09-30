package xsd_test

import (
	"fmt"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// idcCacheSchema declares an xs:key whose selector and field are unions, so
// every selector and field evaluation sorts its result into document order.
const idcCacheSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="row" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="a" type="xs:NCName" use="required"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k">
      <xs:selector xpath="t:row|t:row"/>
      <xs:field xpath="@a|@a"/>
    </xs:key>
  </xs:element>
</xs:schema>`

// idcCacheInstance returns a root carrying rows keyed rows.
func idcCacheInstance(rows int) string {
	var inst strings.Builder
	inst.WriteString(`<root xmlns="urn:t">`)
	for i := range rows {
		fmt.Fprintf(&inst, `<row a="v%d"/>`, i)
	}
	inst.WriteString(`</root>`)
	return inst.String()
}

// TestIdentityConstraintXPathCacheLargeDocument pins that identity-constraint
// evaluation shares one document-order index across a validation run. Each
// field evaluation sorts its union into document order; indexing the whole
// document again for each of them makes validation quadratic in the row count.
//
// The bound is on allocated bytes, which depend on the work done and not on
// the machine: doubling the rows must roughly double what validation
// allocates. A per-evaluation index quadruples it, since both the number of
// evaluations and the size of each index double.
func TestIdentityConstraintXPathCacheLargeDocument(t *testing.T) {
	const (
		baseRows = 2500
		// maxGrowth sits between the linear regime (about 2x) and the
		// quadratic one (about 4x).
		maxGrowth = 3.0
	)

	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(idcCacheSchema))
	require.NoError(t, err)
	compiled, err := xsd.NewCompiler().Compile(t.Context(), sdoc)
	require.NoError(t, err)

	measure := func(rows int) uint64 {
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(idcCacheInstance(rows)))
		require.NoError(t, err)
		var verr error
		allocated := xsdCompileAllocatedBytes(t, func() {
			verr = xsd.NewValidator(compiled).Validate(t.Context(), idoc)
		})
		require.NoError(t, verr)
		return allocated
	}

	base := measure(baseRows)
	grown := measure(2 * baseRows)
	growth := float64(grown) / float64(base)
	t.Logf("validation allocated %d bytes for %d rows and %d bytes for %d rows (%.2fx)",
		base, baseRows, grown, 2*baseRows, growth)
	require.Less(t, growth, maxGrowth,
		"validation allocated %.2fx more for twice the rows (%d bytes at %d rows, %d at %d): suspect a per-evaluation document-order index",
		growth, base, baseRows, grown, 2*baseRows)
}
