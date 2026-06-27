package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestSelfReferentialUnionNoOverflow is a regression test for a stack overflow
// in the NOTATION-carrier / QName-carrier compile-time checks. A union whose
// memberTypes referenced itself (directly or transitively) sent the recursive
// member walk into infinite recursion and crashed the process with
// "fatal error: stack overflow". A genuinely circular union member is an invalid
// schema, so Compile must terminate and report ErrCompilationFailed rather than
// crash.
func TestSelfReferentialUnionNoOverflow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		schema string
	}{
		{
			name: "direct self-referential union member",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="selfUnion">
    <xs:union memberTypes="selfUnion xs:string"/>
  </xs:simpleType>
  <xs:element name="root" type="selfUnion"/>
</xs:schema>`,
		},
		{
			name: "mutually recursive union members",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="unionA">
    <xs:union memberTypes="unionB xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="unionB">
    <xs:union memberTypes="unionA xs:int"/>
  </xs:simpleType>
  <xs:element name="root" type="unionA"/>
</xs:schema>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
			require.NoError(t, err)

			// Must not crash with a stack overflow; a circular union member type is
			// an invalid schema, so compilation must fail with ErrCompilationFailed.
			_, err = xsd.NewCompiler().Compile(t.Context(), doc)
			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		})
	}
}
