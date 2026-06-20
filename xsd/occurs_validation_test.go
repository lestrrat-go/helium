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
		requireCompileResultErr(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	const (
		wantNonNegInt = "is not valid. Expected is 'xs:nonNegativeInteger'."
		wantAllNNI    = "is not valid. Expected is '(xs:nonNegativeInteger | unbounded)'."
		wantMinGtMax  = "The value must not be greater than the value of 'maxOccurs'"
		wantMaxGteOne = "The value must be greater than or equal to 1"
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
			// xs:nonNegativeInteger / xs:allNNI have no leading sign: "+0", "+1"
			// and "-0" are invalid lexical forms even though strconv.Atoi would
			// parse them. libxml2 rejects all three.
			{name: "plus-zero minOccurs", occurs: `minOccurs="+0"`, wantMsg: wantNonNegInt},
			{name: "plus-one minOccurs", occurs: `minOccurs="+1"`, wantMsg: wantNonNegInt},
			{name: "minus-zero minOccurs", occurs: `minOccurs="-0"`, wantMsg: wantNonNegInt},
			{name: "plus-zero maxOccurs", occurs: `maxOccurs="+0"`, wantMsg: wantAllNNI},
			{name: "plus-one maxOccurs", occurs: `maxOccurs="+1"`, wantMsg: wantAllNNI},
			{name: "minus-zero maxOccurs", occurs: `maxOccurs="-0"`, wantMsg: wantAllNNI},
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

	// A group reference can also appear directly under xs:complexType (without an
	// enclosing compositor). That branch lives in read_types.go rather than
	// read_particles.go and formerly used the non-validating occurs parser, so a
	// negative minOccurs was silently accepted while xmllint rejects it.
	t.Run("rejects on direct group reference under complexType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g">
    <xs:sequence>
      <xs:element name="child" type="xs:string"/>
    </xs:sequence>
  </xs:group>
  <xs:element name="root">
    <xs:complexType>
      <xs:group ref="g" minOccurs="-1"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantNonNegInt)
	})

	// An explicitly empty occurs attribute (minOccurs="" / maxOccurs="") is an
	// invalid lexical, not an absent attribute. Before the fix presence was
	// detected with value!="" so empty strings were silently treated as absent
	// and defaulted; xmllint rejects them. These run across every particle kind.
	t.Run("rejects empty occurs", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name    string
			schema  string
			wantMsg string
		}{
			{
				name:    "element empty minOccurs",
				wantMsg: wantNonNegInt,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:element name="child" type="xs:string" minOccurs=""/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "element empty maxOccurs",
				wantMsg: wantAllNNI,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:element name="child" type="xs:string" maxOccurs=""/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "sequence empty minOccurs",
				wantMsg: wantNonNegInt,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence minOccurs=""><xs:element name="child" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "sequence empty maxOccurs",
				wantMsg: wantAllNNI,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence maxOccurs=""><xs:element name="child" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "any empty maxOccurs",
				wantMsg: wantAllNNI,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:any maxOccurs=""/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "group ref empty minOccurs",
				wantMsg: wantNonNegInt,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:sequence><xs:element name="child" type="xs:string"/></xs:sequence></xs:group>
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:group ref="g" minOccurs=""/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.Contains(t, compileErrors(t, tc.schema), tc.wantMsg)
			})
		}
	})

	// maxOccurs=0 with an absent minOccurs (effective default minOccurs=1) is a
	// content model that requires the particle yet forbids it: xmllint reports the
	// "maxOccurs >= 1" diagnostic. Before the fix non-element particles only
	// enforced min<=max when minOccurs was explicitly present, so these compiled.
	t.Run("rejects default-min maxOccurs zero", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{
				name: "element absent min",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:element name="child" type="xs:string" maxOccurs="0"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "element explicit min one",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:element name="child" type="xs:string" minOccurs="1" maxOccurs="0"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "sequence absent min",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence maxOccurs="0"><xs:element name="child" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "any absent min",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:any maxOccurs="0"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "group ref absent min",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:sequence><xs:element name="child" type="xs:string"/></xs:sequence></xs:group>
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:group ref="g" maxOccurs="0"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				errs := compileErrors(t, tc.schema)
				require.Contains(t, errs, wantMaxGteOne)
				// xmllint reports only the maxOccurs error here, not a duplicate
				// min>max diagnostic.
				require.NotContains(t, errs, wantMinGtMax)
			})
		}
	})

	// minOccurs=0 maxOccurs=0 is a legal prohibited particle on every particle
	// kind (not just xs:element); it must compile cleanly.
	t.Run("accepts prohibited particle on all kinds", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{
				name: "sequence",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence minOccurs="0" maxOccurs="0"><xs:element name="child" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "any",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:any minOccurs="0" maxOccurs="0"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "group ref",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:sequence><xs:element name="child" type="xs:string"/></xs:sequence></xs:group>
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:group ref="g" minOccurs="0" maxOccurs="0"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.Empty(t, compileErrors(t, tc.schema))
			})
		}
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
			// maxOccurs=0 is a legal prohibited particle when minOccurs is also 0;
			// libxml2 compiles this without error (only rejects maxOccurs<1 when
			// the effective minOccurs is >= 1).
			{name: "prohibited particle", occurs: `minOccurs="0" maxOccurs="0"`},
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
