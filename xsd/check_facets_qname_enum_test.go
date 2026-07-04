package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestEnumValueAgainstBaseQNameValueSpace verifies that the per-literal QName/
// NOTATION suppression in checkEnumValueAgainstBase is narrow: a QName base that
// is further restricted with a value-space facet (xs:length) still has its
// enumeration values validated against the base value space, so an out-of-space
// member is rejected at COMPILE time. Previously a blanket early-return for any
// QName/NOTATION base over-suppressed and let such a schema compile.
func TestEnumValueAgainstBaseQNameValueSpace(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		schema           string
		wantCompileError bool
		offending        string
	}{
		{
			// Per XSD Part 2 (W3C Schema errata, bug 4009), length/minLength/maxLength
			// do NOT apply to xs:QName — the facet is vacuously satisfied. So "abc",
			// a perfectly bound (prefix-less) QName, is a valid enumeration value even
			// though the base carries xs:length value="2": the length facet does not
			// constrain a QName, so the schema must COMPILE.
			name: "qname base length facet does not constrain enum",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="qname2">
    <xs:restriction base="xs:QName">
      <xs:length value="2"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="enumQname2">
    <xs:restriction base="qname2">
      <xs:enumeration value="abc"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="enumQname2"/>
</xs:schema>`,
			wantCompileError: false,
		},
		{
			// "ab" satisfies the length-2 facet and is a valid bound QName, so the
			// schema must still compile — the narrowed suppression must not
			// over-reject a valid QName enumeration value.
			name: "qname base length facet in-space enum compiles",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="qname2">
    <xs:restriction base="xs:QName">
      <xs:length value="2"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="enumQname2">
    <xs:restriction base="qname2">
      <xs:enumeration value="ab"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="enumQname2"/>
</xs:schema>`,
			wantCompileError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.wantCompileError {
				errs := compileSchemaErrors(t, tc.schema)
				require.NotEmpty(t, errs, "expected a compile error for out-of-space enumeration value")
				require.Contains(t, errs, "facet 'enumeration'", "expected enumeration-facet compile diagnostic")
				require.Contains(t, errs, tc.offending, "expected the offending enumeration value in the diagnostic")
				return
			}

			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
			require.NoError(t, err)
			_, err = xsd.NewCompiler().Compile(t.Context(), doc)
			require.NoError(t, err, "valid QName enumeration must compile")
		})
	}
}

// TestEnumUnboundQNameNotDoubleReported verifies that an enumeration literal with
// an unbound QName prefix is reported exactly once — by checkEnumQNameAndNotation
// (the "atomic type" diagnostic) — and is NOT additionally reported by
// checkEnumValueAgainstBase as a "base type" value-space failure. The per-literal
// suppression keeps the existing duplicate-prefix behavior intact.
func TestEnumUnboundQNameNotDoubleReported(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="enumQname">
    <xs:restriction base="xs:QName">
      <xs:enumeration value="missing:thing"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="enumQname"/>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, _ = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
	_ = collector.Close()

	var atomicHits, baseHits int
	for _, e := range collector.Errors() {
		msg := e.Error()
		if !strings.Contains(msg, "missing:thing") {
			continue
		}
		if strings.Contains(msg, "atomic type") {
			atomicHits++
		}
		if strings.Contains(msg, "base type") {
			baseHits++
		}
	}

	require.Equal(t, 1, atomicHits, "unbound-prefix QName enum must be reported once by the QName/NOTATION check")
	require.Equal(t, 0, baseHits, "unbound-prefix QName enum must NOT be double-reported as a base-type value-space failure")
}
