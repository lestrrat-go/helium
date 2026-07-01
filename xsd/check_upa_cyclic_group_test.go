package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestCyclicGroupRefRejected verifies that a circular model-group reference — a
// named xs:group that references itself directly or transitively — is reported
// as a schema error (invalid) instead of stack-overflowing the UPA/Glushkov walk
// (an uncatchable crash). Circular group references are forbidden in both XSD 1.0
// and XSD 1.1, so the diagnostic is emitted in both versions. Mirrors W3C
// msData/group groupB012-B015.
func TestCyclicGroupRefRejected(t *testing.T) {
	t.Parallel()

	// Direct self-reference: foo's sequence contains a ref back to foo.
	directSelf := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:group ref="foo"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:group name="foo">
    <xs:sequence>
      <xs:element name="foo1" type="xs:string"/>
      <xs:group ref="foo"/>
    </xs:sequence>
  </xs:group>
</xs:schema>`

	// Indirect cycle: foo -> bar -> foo.
	indirect := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:group ref="foo"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:group name="foo">
    <xs:sequence>
      <xs:group ref="bar"/>
    </xs:sequence>
  </xs:group>
  <xs:group name="bar">
    <xs:sequence>
      <xs:group ref="foo"/>
    </xs:sequence>
  </xs:group>
</xs:schema>`

	for _, tc := range []struct {
		name   string
		schema string
	}{
		{"direct self-reference", directSelf},
		{"indirect cycle", indirect},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// DEFAULT (XSD 1.0) mode: must terminate with a circular-reference error.
			_, errs := compileWithErrors(t, tc.schema)
			require.Contains(t, errs, "Circular reference to the model group definition",
				"a circular group reference must be reported as circular; got: %q", errs)
		})
	}
}

// TestCyclicGroupRefRejectedVersion11 confirms the same circular-group rejection
// applies (and does not crash) under the XSD 1.1 opt-in.
func TestCyclicGroupRefRejectedVersion11(t *testing.T) {
	t.Parallel()

	schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:group ref="foo"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:group name="foo">
    <xs:sequence>
      <xs:element name="foo1" type="xs:string"/>
      <xs:group ref="foo"/>
    </xs:sequence>
  </xs:group>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, cerr := xsd.NewCompiler().
		Version(xsd.Version11).
		Label("test.xsd").
		ErrorHandler(collector).
		Compile(t.Context(), doc)
	requireCompileResultErr(t, cerr)
	require.Error(t, cerr, "a circular group reference must make Version11 compilation fail")
	require.NoError(t, collector.Close())
	_, errs := partitionCompileErrors(collector.Errors())
	require.Contains(t, errs, "Circular reference to the model group definition",
		"a circular group reference must be reported as circular under Version11; got: %q", errs)
}
