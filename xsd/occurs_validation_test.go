package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestOccursValidation verifies that minOccurs/maxOccurs are parsed as
// non-negative integers (maxOccurs additionally allowing "unbounded") and that
// negative values, non-integer values, and minOccurs > maxOccurs are reported
// as fatal schema parser errors at compile time. Before the fix a value such as
// minOccurs="-1" was silently accepted, producing a too-permissive content
// model that let a missing required child validate.
func TestOccursValidation(t *testing.T) {
	t.Parallel()

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	const (
		wantNonNegInt = "is not a valid value of the atomic type 'xs:nonNegativeInteger'"
		wantAllNNI    = "is not a valid value of the union type 'xs:allNNI'"
		wantMinGtMax  = "The value must not be greater than the value of 'maxOccurs'"
	)

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name    string
			occurs  string
			wantMsg string
		}{
			{name: "negative minOccurs", occurs: `minOccurs="-1"`, wantMsg: wantNonNegInt},
			{name: "negative maxOccurs", occurs: `maxOccurs="-2"`, wantMsg: wantAllNNI},
			{name: "non-integer minOccurs", occurs: `minOccurs="abc"`, wantMsg: wantNonNegInt},
			{name: "non-integer maxOccurs", occurs: `maxOccurs="abc"`, wantMsg: wantAllNNI},
			{name: "min greater than max", occurs: `minOccurs="3" maxOccurs="2"`, wantMsg: wantMinGtMax},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="child" type="xs:string" ` + tc.occurs + `/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
				require.Contains(t, compileErrors(t, schemaXML), tc.wantMsg)
			})
		}
	})

	// The bug also surfaces on the compositor (sequence/choice/all) particle
	// itself and on any/group references, which checkLocalElement never covered.
	t.Run("rejects on compositor particle", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence minOccurs="-1">
        <xs:element name="child" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantNonNegInt)
	})

	t.Run("rejects on choice min greater than max", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:choice minOccurs="5" maxOccurs="2">
        <xs:element name="child" type="xs:string"/>
      </xs:choice>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMinGtMax)
	})

	t.Run("rejects on any wildcard", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:any maxOccurs="-3"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantAllNNI)
	})

	t.Run("rejects on group reference", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g">
    <xs:sequence>
      <xs:element name="child" type="xs:string"/>
    </xs:sequence>
  </xs:group>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:group ref="g" minOccurs="-1"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantNonNegInt)
	})

	// Valid occurs forms must still compile cleanly, including unbounded and
	// minOccurs=0 (optional) and maxOccurs=0 (prohibited particle).
	t.Run("accepts valid occurs", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			occurs string
		}{
			{name: "default", occurs: ``},
			{name: "minOccurs zero", occurs: `minOccurs="0"`},
			{name: "unbounded", occurs: `maxOccurs="unbounded"`},
			{name: "range", occurs: `minOccurs="0" maxOccurs="5"`},
			{name: "zero to unbounded", occurs: `minOccurs="0" maxOccurs="unbounded"`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="child" type="xs:string" ` + tc.occurs + `/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
				require.Empty(t, compileErrors(t, schemaXML))
			})
		}
	})
}
