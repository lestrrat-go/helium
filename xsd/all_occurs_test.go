package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestAllOccursValidation verifies the xs:all-specific occurrence constraints
// (XSD Part 1 §3.8.6): the xs:all compositor particle's minOccurs must be 0 or 1
// and its maxOccurs must be 1, and every element particle directly inside an
// xs:all must have minOccurs/maxOccurs of 0 or 1. Before the fix these compiled
// with zero errors; /usr/bin/xmllint rejects them. The wording mirrors xmllint:
//
//	attribute 'minOccurs': The value 'N' is not valid. Expected is '(0 | 1)'.
//	attribute 'maxOccurs': The value 'N' is not valid. Expected is '1'.
//	Element '{...}element': Invalid value for maxOccurs (must be 0 or 1).
func TestAllOccursValidation(t *testing.T) {
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
		wantAllMin   = "is not valid. Expected is '(0 | 1)'."
		wantAllMax   = "is not valid. Expected is '1'."
		wantChildMax = "Invalid value for maxOccurs (must be 0 or 1)."
		wantChildMin = "Invalid value for minOccurs (must be 0 or 1)."
	)

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name    string
			schema  string
			wantMsg string
		}{
			{
				name:    "all minOccurs 2",
				wantMsg: wantAllMin,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all minOccurs="2"><xs:element name="child" type="xs:string"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "all maxOccurs 2",
				wantMsg: wantAllMax,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all maxOccurs="2"><xs:element name="child" type="xs:string"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "all maxOccurs 0",
				wantMsg: wantAllMax,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all maxOccurs="0"><xs:element name="child" type="xs:string"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "all maxOccurs unbounded",
				wantMsg: wantAllMax,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all maxOccurs="unbounded"><xs:element name="child" type="xs:string"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "all child maxOccurs 2",
				wantMsg: wantChildMax,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all><xs:element name="child" type="xs:string" maxOccurs="2"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "all child minOccurs 2",
				wantMsg: wantChildMin,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all><xs:element name="child" type="xs:string" minOccurs="2"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "all in named group minOccurs 2",
				wantMsg: wantAllMin,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all minOccurs="2"><xs:element name="child" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.Contains(t, compileErrors(t, tc.schema), tc.wantMsg)
			})
		}
	})

	// Valid xs:all forms must still compile cleanly: default occurs, minOccurs=0,
	// minOccurs=1, maxOccurs=1, and child element occurs in {0,1}.
	t.Run("accepts valid all", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{
				name: "default",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all><xs:element name="child" type="xs:string"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "minOccurs 0",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all minOccurs="0"><xs:element name="child" type="xs:string"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "minOccurs 1 maxOccurs 1",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all minOccurs="1" maxOccurs="1"><xs:element name="child" type="xs:string"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "child minOccurs 0",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all><xs:element name="child" type="xs:string" minOccurs="0" maxOccurs="1"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.Empty(t, compileErrors(t, tc.schema))
			})
		}
	})
}

// TestOccursLexicalMessageParity verifies that the lexical-error wording for an
// invalid minOccurs/maxOccurs matches /usr/bin/xmllint exactly:
//
//	minOccurs: The value '-1' is not valid. Expected is 'xs:nonNegativeInteger'.
//	maxOccurs: The value '-2' is not valid. Expected is '(xs:nonNegativeInteger | unbounded)'.
//
// These run on both an xs:element particle (checkLocalElement path) and a
// compositor/wildcard particle (validateOccursAttrs path).
func TestOccursLexicalMessageParity(t *testing.T) {
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
		wantMin = "The value '-1' is not valid. Expected is 'xs:nonNegativeInteger'."
		wantMax = "The value '-2' is not valid. Expected is '(xs:nonNegativeInteger | unbounded)'."
	)

	for _, tc := range []struct {
		name    string
		schema  string
		wantMsg string
	}{
		{
			name:    "element minOccurs",
			wantMsg: wantMin,
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:element name="child" type="xs:string" minOccurs="-1"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
		},
		{
			name:    "element maxOccurs",
			wantMsg: wantMax,
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:element name="child" type="xs:string" maxOccurs="-2"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
		},
		{
			name:    "sequence minOccurs",
			wantMsg: wantMin,
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:sequence minOccurs="-1"><xs:element name="child" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`,
		},
		{
			name:    "any maxOccurs",
			wantMsg: wantMax,
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:any maxOccurs="-2"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Contains(t, compileErrors(t, tc.schema), tc.wantMsg)
		})
	}
}
