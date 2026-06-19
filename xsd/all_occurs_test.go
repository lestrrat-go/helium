package xsd_test

import (
	"testing"
	"testing/fstest"

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

	// A lexically invalid occurs value on a direct xs:all child must surface only
	// the generic xs:nonNegativeInteger/allNNI lexical error from checkLocalElement
	// — never the all-specific "(must be 0 or 1)" diagnostic, matching xmllint.
	t.Run("child lexical error not all-specific", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			schema     string
			wantMsg    string
			notWantMsg string
		}{
			{
				name:       "child minOccurs -1",
				wantMsg:    "Expected is 'xs:nonNegativeInteger'.",
				notWantMsg: wantChildMin,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all><xs:element name="child" type="xs:string" minOccurs="-1"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:       "child maxOccurs -2",
				wantMsg:    "Expected is '(xs:nonNegativeInteger | unbounded)'.",
				notWantMsg: wantChildMax,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all><xs:element name="child" type="xs:string" maxOccurs="-2"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				got := compileErrors(t, tc.schema)
				require.Contains(t, got, tc.wantMsg)
				require.NotContains(t, got, tc.notWantMsg)
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
			// xs:nonNegativeInteger lexical space allows leading zeros: "01"
			// parses to 1 and is accepted on both the all compositor and a child,
			// matching xmllint (these were wrongly rejected by raw "0"/"1" string
			// comparison before the fix).
			{
				name: "all minOccurs 01",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all minOccurs="01"><xs:element name="child" type="xs:string"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "all maxOccurs 01",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all maxOccurs="01"><xs:element name="child" type="xs:string"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "child minOccurs 01",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all><xs:element name="child" type="xs:string" minOccurs="01"/></xs:all>
  </xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "child maxOccurs 01",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root"><xs:complexType>
    <xs:all><xs:element name="child" type="xs:string" maxOccurs="01"/></xs:all>
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

// TestAllGroupRefConstraints verifies the constraints on an xs:group reference
// that resolves to an 'all' model group (XSD Part 1 §3.8.6 cos-all-limited /
// §3.8.2). Before the fix both invalid forms compiled with zero errors;
// /usr/bin/xmllint rejects them with the wording mirrored below:
//
//	The particle's {max occurs} must be 1, since the reference resolves to an 'all' model group.
//	A model group definition is referenced, but it contains an 'all' model group, which cannot be contained by model groups.
func TestAllGroupRefConstraints(t *testing.T) {
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
		wantDirectMax = "The particle's {max occurs} must be 1, since the reference resolves to an 'all' model group."
		wantNested    = "A model group definition is referenced, but it contains an 'all' model group, which cannot be contained by model groups."
	)

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name    string
			schema  string
			wantMsg string
		}{
			{
				name:    "direct ref maxOccurs 2",
				wantMsg: wantDirectMax,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:group ref="g" maxOccurs="2"/></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "direct ref maxOccurs unbounded",
				wantMsg: wantDirectMax,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:group ref="g" maxOccurs="unbounded"/></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "nested in sequence",
				wantMsg: wantNested,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:    "nested in choice",
				wantMsg: wantNested,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:choice><xs:group ref="g"/></xs:choice></xs:complexType></xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.Contains(t, compileErrors(t, tc.schema), tc.wantMsg)
			})
		}
	})

	// A direct (non-nested) reference to an 'all' model group with the default or
	// explicit maxOccurs="1" is valid and must compile cleanly.
	t.Run("accepts direct ref", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{
				name: "default maxOccurs",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "explicit maxOccurs 1",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:group ref="g" maxOccurs="1"/></xs:complexType></xs:element>
</xs:schema>`,
			},
			// A 0/0 occurrence is a prohibited particle that maps to no particle
			// at all, so the all-group constraints do not apply. xmllint accepts
			// these both as a direct content model and nested in another group.
			{
				name: "direct ref minOccurs 0 maxOccurs 0",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:group ref="g" minOccurs="0" maxOccurs="0"/></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "nested ref minOccurs 0 maxOccurs 0",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:sequence>
    <xs:group ref="g" minOccurs="0" maxOccurs="0"/>
    <xs:element name="b" type="xs:string"/>
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

	// When the maxOccurs lexical value fails to parse, or it is "0" with the
	// default minOccurs=1, the generic occurrence validator already reports the
	// lexical / ">= 1" diagnostic. The all-specific "must be 1" message must NOT
	// also fire — xmllint reports only the generic error in these cases.
	t.Run("ref occurs error not all-specific", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			schema     string
			wantMsg    string
			notWantMsg string
		}{
			{
				name:       "maxOccurs lexical invalid",
				wantMsg:    "Expected is '(xs:nonNegativeInteger | unbounded)'.",
				notWantMsg: wantDirectMax,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:group ref="g" maxOccurs="abc"/></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name:       "maxOccurs 0 default minOccurs",
				wantMsg:    "The value must be greater than or equal to 1.",
				notWantMsg: wantDirectMax,
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
  <xs:element name="root"><xs:complexType><xs:group ref="g" maxOccurs="0"/></xs:complexType></xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				got := compileErrors(t, tc.schema)
				require.Contains(t, got, tc.wantMsg)
				require.NotContains(t, got, tc.notWantMsg)
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

// TestRedefineAllGroupNesting verifies that an xs:redefine of an xs:all model
// group enforces the all-group placement rule (cos-all-limited) on its
// self-reference. Redefining an 'all' group as a sequence/choice that nests the
// self-reference is illegal: the resolved 'all' group cannot be contained by
// another model group. Before the fix the self-reference was deleted from the
// group-ref table before checkAllGroupRef could run, so these compiled with
// zero diagnostics. A legitimate redefine (self-reference as the entire content
// model) must still compile cleanly.
func TestRedefineAllGroupNesting(t *testing.T) {
	t.Parallel()

	// base.xsd declares group "g" as an xs:all model group; main.xsd redefines it.
	const baseSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
</xs:schema>`

	compileErrors := func(t *testing.T, mainSchema string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(mainSchema))
		require.NoError(t, err)
		fsys := fstest.MapFS{"base.xsd": &fstest.MapFile{Data: []byte(baseSchema)}}
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("main.xsd").FS(fsys).ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	const wantNested = "A model group definition is referenced, but it contains an 'all' model group, which cannot be contained by model groups."

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{
				name: "self-reference nested in sequence",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:sequence><xs:group ref="g"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>
  </xs:redefine>
  <xs:element name="root"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`,
			},
			{
				name: "self-reference nested in choice",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:choice><xs:group ref="g"/><xs:element name="b" type="xs:string"/></xs:choice></xs:group>
  </xs:redefine>
  <xs:element name="root"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.Contains(t, compileErrors(t, tc.schema), wantNested)
			})
		}
	})

	// A legitimate redefine must still compile cleanly. Here group "g" is
	// redefined to a new xs:all model group (no self-reference), which is a valid
	// override and exercises the same redefine-group path without triggering the
	// nested-all placement rule.
	t.Run("accepts legitimate redefine", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:all></xs:group>
  </xs:redefine>
  <xs:element name="root"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`
		require.Empty(t, compileErrors(t, schema))
	})
}
