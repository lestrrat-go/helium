package xsd_test

import (
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestCyclicSimpleTypeTerminates is a regression test for infinite recursion /
// hangs triggered by cyclic simple-type definitions. Two distinct defects are
// covered:
//
//   - A union whose memberTypes referenced itself (directly or transitively)
//     sent the NOTATION/QName carrier walk into infinite recursion and crashed
//     with "fatal error: stack overflow".
//   - A restriction-base cycle (A restricts B, B restricts A) or a list-item
//     cycle spun forever inside resolveRefs → resolveVariety/resolveItemType,
//     which walk the BaseType chain.
//
// All of these are invalid schemas, so Compile must terminate and report
// ErrCompilationFailed rather than crash or hang. Each case runs under a
// watchdog so the test FAILS (instead of hanging the suite) if the loop ever
// regresses.
func TestCyclicSimpleTypeTerminates(t *testing.T) {
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
		{
			name: "restriction base cycle",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="rA">
    <xs:restriction base="rB"/>
  </xs:simpleType>
  <xs:simpleType name="rB">
    <xs:restriction base="rA"/>
  </xs:simpleType>
  <xs:element name="root" type="rA"/>
</xs:schema>`,
		},
		{
			name: "list item cycle",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="listA">
    <xs:list itemType="listB"/>
  </xs:simpleType>
  <xs:simpleType name="listB">
    <xs:list itemType="listA"/>
  </xs:simpleType>
  <xs:element name="root" type="listA"/>
</xs:schema>`,
		},
		{
			name: "list of self-referential union",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="cyclicUnion">
    <xs:union memberTypes="cyclicUnion xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="listOfCyclic">
    <xs:list itemType="cyclicUnion"/>
  </xs:simpleType>
  <xs:element name="root" type="listOfCyclic"/>
</xs:schema>`,
		},
		{
			// Reaches resolveWhiteSpace/validateValue via checkAttrUseConstraints
			// during resolveRefs, i.e. BEFORE checkCircularSimpleTypes runs — the
			// attribute-default path that previously hung on the cyclic type.
			name: "cyclic type on attribute default",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="cA">
    <xs:restriction base="cB"/>
  </xs:simpleType>
  <xs:simpleType name="cB">
    <xs:restriction base="cA"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="cA" default="x"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "cyclic type on element fixed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="fA">
    <xs:restriction base="fB"/>
  </xs:simpleType>
  <xs:simpleType name="fB">
    <xs:restriction base="fA"/>
  </xs:simpleType>
  <xs:element name="root" type="fA" fixed="x"/>
</xs:schema>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
			require.NoError(t, err)

			// Run Compile under a watchdog: if the cyclic-type guards regress and
			// the walk loops forever, the watchdog fires and the test FAILS quickly
			// instead of hanging the whole suite.
			done := make(chan error, 1)
			go func() {
				_, cerr := xsd.NewCompiler().Compile(t.Context(), doc)
				done <- cerr
			}()

			select {
			case cerr := <-done:
				// A circular simple type is an invalid schema.
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed)
			case <-time.After(10 * time.Second):
				t.Fatal("Compile did not terminate: cyclic simple-type guard regressed")
			}
		})
	}
}
