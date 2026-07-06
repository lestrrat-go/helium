package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestOccursLargeNonNegativeInteger verifies that a minOccurs/maxOccurs whose
// lexical value is a valid xs:nonNegativeInteger too large for a machine int is
// accepted, not rejected. XSD's xs:nonNegativeInteger (minOccurs) and
// (xs:nonNegativeInteger | "unbounded") (maxOccurs) value spaces are unbounded,
// so an arbitrarily large all-digit occurrence literal is lexically valid. Before
// the fix strconv.Atoi overflowed and the value was reported as an invalid
// xs:nonNegativeInteger, false-rejecting an otherwise valid schema (W3C xsd10
// particlesZ033_a, whose element carries minOccurs/maxOccurs of ~2^96).
func TestOccursLargeNonNegativeInteger(t *testing.T) {
	t.Parallel()

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		if err != nil {
			requireCompileResultErr(t, err)
		}
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	for _, tc := range []struct {
		name   string
		schema string
	}{
		{
			// The exact construct from W3C xsd10 particlesZ033_a: an element with
			// minOccurs/maxOccurs ~2^96, inside a sequence with ordinary occurs.
			name: "particlesZ033_a construct",
			schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:complexType name="fooType">
    <xsd:sequence minOccurs="56" maxOccurs="100">
      <xsd:element name="e1" minOccurs="79228162514244337593543950335" maxOccurs="79228162514264337593543950335"/>
      <xsd:any namespace="##other"/>
    </xsd:sequence>
  </xsd:complexType>
</xsd:schema>`,
		},
		{
			// A large minOccurs alone (well past int64) on an element particle.
			name: "large element minOccurs",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:element name="child" type="xs:string" minOccurs="99999999999999999999999999999" maxOccurs="unbounded"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
		},
		{
			// A large maxOccurs alone on a compositor particle (validateOccursAttrs path).
			name: "large sequence maxOccurs",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence maxOccurs="99999999999999999999999999999">
      <xs:element name="child" type="xs:string"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`,
		},
		{
			// Both minOccurs and maxOccurs overflow int but minOccurs (2^96) <
			// maxOccurs (2^96 + 1): a valid though unsatisfiable schema. The relative
			// min <= max constraint holds, so it must still compile.
			name: "both overflow min less than max",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:element name="child" type="xs:string" minOccurs="79228162514264337593543950335" maxOccurs="79228162514264337593543950336"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Empty(t, compileErrors(t, tc.schema))
		})
	}
}

// TestOccursOverflowMinGreaterThanMax verifies that the relative min <= max
// occurrence constraint (cos-particle / schema-for-schemas) is enforced even when
// BOTH minOccurs and maxOccurs are too large for a machine int. Because an
// overflowing maxOccurs clamps to the "unbounded" sentinel, a naive int-only
// min<=max check is bypassed and min>max slips through as a FALSE-ACCEPT; the
// magnitude-safe digit-string comparison must still reject it.
func TestOccursOverflowMinGreaterThanMax(t *testing.T) {
	t.Parallel()

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	const wantMsg = "The value must not be greater than the value of 'maxOccurs'."

	for _, tc := range []struct {
		name   string
		schema string
	}{
		{
			// Element particle (checkLocalElement path): minOccurs 2^97 > maxOccurs 2^96.
			name: "element min greater than max both overflow",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:element name="child" type="xs:string" minOccurs="158456325028528675187087900672" maxOccurs="79228162514264337593543950336"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
		},
		{
			// Compositor particle (validateOccursAttrs path): sequence minOccurs > maxOccurs.
			name: "sequence min greater than max both overflow",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence minOccurs="158456325028528675187087900672" maxOccurs="79228162514264337593543950336">
      <xs:element name="child" type="xs:string"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`,
		},
		{
			// Same-length digit strings differing only in the last digit (2^96+1 vs 2^96).
			name: "element min greater than max same length",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:element name="child" type="xs:string" minOccurs="79228162514264337593543950336" maxOccurs="79228162514264337593543950335"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Contains(t, compileErrors(t, tc.schema), wantMsg)
		})
	}
}
