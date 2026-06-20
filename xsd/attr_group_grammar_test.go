package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestAttrGroupSiblingRefDuplicate verifies that two SIBLING xs:attributeGroup
// ref children that reference the SAME group are each expanded, so a name the
// referenced group contributes through both siblings is reported as a duplicate
// attribute use (ag-props-correct.2). Treating the cycle guard as a global
// "seen ever" set wrongly skipped the second sibling and let the duplicate
// compile; the guard must be a recursion stack so siblings each expand while
// true cycles are still cut.
func TestAttrGroupSiblingRefDuplicate(t *testing.T) {
	t.Parallel()

	// g references h TWICE; h contributes attribute 'x'. The two refs to h each
	// bring in 'x', so 'x' is a duplicate use in g.
	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="h">
    <xs:attribute name="x" type="xs:string"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g">
    <xs:attributeGroup ref="h"/>
    <xs:attributeGroup ref="h"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`

	_, errs := compileWithErrors(t, schemaXML)
	require.Contains(t, errs, "Duplicate attribute use 'x'",
		"two sibling refs to the same group must surface the duplicated attribute; got: %q", errs)
}

// TestAttrGroupCycleStillCut verifies that the recursion-stack guard added for
// sibling expansion still cuts a genuine reference CYCLE (h -> i -> h) without
// looping or false-rejecting. The schema is structurally valid (no duplicate
// names) and must compile cleanly.
func TestAttrGroupCycleStillCut(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="h">
    <xs:attribute name="a" type="xs:string"/>
    <xs:attributeGroup ref="i"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="i">
    <xs:attribute name="b" type="xs:string"/>
    <xs:attributeGroup ref="h"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="h"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`

	// An indirect cycle is itself a circular reference, but the point of this test
	// is that the duplicate-detection walk terminates: it must not loop, and it
	// must not falsely report 'a'/'b' as duplicates.
	_, errs := compileWithErrors(t, schemaXML)
	require.NotContains(t, errs, "Duplicate attribute use 'a'",
		"a cycle must not be misreported as a duplicate of 'a'; got: %q", errs)
	require.NotContains(t, errs, "Duplicate attribute use 'b'",
		"a cycle must not be misreported as a duplicate of 'b'; got: %q", errs)
}

// TestAttrGroupSelfReferenceCircular verifies that a SELF-referential attribute
// group (<xs:attributeGroup name="g"><xs:attributeGroup ref="g"/></...>) outside
// <xs:redefine> is reported as a circular reference (src-attribute_group.3),
// rather than being silently dropped and compiling.
func TestAttrGroupSelfReferenceCircular(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="a" type="xs:string"/>
    <xs:attributeGroup ref="g"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`

	_, errs := compileWithErrors(t, schemaXML)
	require.Contains(t, errs, "Circular reference to the attribute group 'g' defined.",
		"a self-referential attribute group must be reported as circular; got: %q", errs)
}

// TestAttrGroupSelfReferenceSourceInclude verifies the circular-reference
// diagnostic for a self-referential attribute group declared in an INCLUDED
// schema is attributed to the declaring (included) file, not the top-level label.
func TestAttrGroupSelfReferenceSourceInclude(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "selfref_main.xsd"
		incXSD  = "selfref_inc.xsd"
	)

	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="selfref_inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
		incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attributeGroup ref="g"/>
  </xs:attributeGroup>
</xs:schema>`)},
	}

	errStr := compileFSErrors(t, fsys, mainXSD)
	require.Contains(t, errStr, "Circular reference to the attribute group 'g' defined.",
		"expected the circular-reference diagnostic; got: %q", errStr)
	require.Contains(t, errStr, incXSD+":",
		"circular-reference diagnostic must cite the declaring (included) file; got: %q", errStr)
	require.False(t, strings.Contains(errStr, mainXSD+":"),
		"circular-reference diagnostic must not cite the top-level label; got: %q", errStr)
}

// TestOccursDiagnosticSourceInclude verifies that the occurrence/all schema
// diagnostics introduced for cos-all-limited / xs:nonNegativeInteger occurrence
// validation are attributed to the DECLARING file when the offending particle
// lives in an INCLUDED schema, rather than to the top-level compiler label.
func TestOccursDiagnosticSourceInclude(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "occurs_main.xsd"
		incXSD  = "occurs_inc.xsd"
	)

	// Each case puts the offending particle entirely in the included file, so the
	// reported line number is meaningful only when attributed to that file.
	cases := []struct {
		name    string
		incBody string
		msg     string
	}{
		{
			name: "invalid maxOccurs on sequence",
			incBody: `  <xs:complexType name="t">
    <xs:sequence maxOccurs="abc">
      <xs:element name="e" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>`,
			msg: "Expected is '(xs:nonNegativeInteger | unbounded)'.",
		},
		{
			name: "all maxOccurs not 1",
			incBody: `  <xs:complexType name="t">
    <xs:all maxOccurs="2">
      <xs:element name="e" type="xs:string"/>
    </xs:all>
  </xs:complexType>`,
			msg: "Expected is '1'.",
		},
		{
			name: "all element particle maxOccurs not 0 or 1",
			incBody: `  <xs:complexType name="t">
    <xs:all>
      <xs:element name="e" type="xs:string" maxOccurs="2"/>
    </xs:all>
  </xs:complexType>`,
			msg: "Invalid value for maxOccurs (must be 0 or 1).",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{
				mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="occurs_inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
				incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + tc.incBody + `
</xs:schema>`)},
			}

			errStr := compileFSErrors(t, fsys, mainXSD)
			require.Contains(t, errStr, tc.msg, "expected the occurrence diagnostic; got: %q", errStr)
			require.Contains(t, errStr, incXSD+":",
				"occurrence diagnostic must cite the declaring (included) file; got: %q", errStr)
			require.False(t, strings.Contains(errStr, mainXSD+":"),
				"occurrence diagnostic must not cite the top-level label; got: %q", errStr)
		})
	}
}

// TestAllGroupRefDiagnosticSourceInclude verifies that the cos-all-limited
// group-reference diagnostics (checkAllGroupRef) are attributed to the declaring
// (included) file when the referencing xs:group particle lives in an included
// schema, using the source captured on groupRefSource at parse time.
func TestAllGroupRefDiagnosticSourceInclude(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "allgrp_main.xsd"
		incXSD  = "allgrp_inc.xsd"
	)

	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="allgrp_inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
		// An xs:group whose body is an 'all' model group, referenced while NESTED
		// inside an xs:sequence — forbidden by cos-all-limited.
		incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="ag">
    <xs:all>
      <xs:element name="e" type="xs:string"/>
    </xs:all>
  </xs:group>
  <xs:complexType name="t">
    <xs:sequence>
      <xs:group ref="ag"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)},
	}

	errStr := compileFSErrors(t, fsys, mainXSD)
	require.Contains(t, errStr, "cannot be contained by model groups",
		"expected the nested-all group-ref diagnostic; got: %q", errStr)
	require.Contains(t, errStr, incXSD+":",
		"all-group-ref diagnostic must cite the declaring (included) file; got: %q", errStr)
	require.False(t, strings.Contains(errStr, mainXSD+":"),
		"all-group-ref diagnostic must not cite the top-level label; got: %q", errStr)
}

// compileFSErrors compiles the schema rooted at mainName from fsys and returns
// the concatenated FATAL compile diagnostics (warnings excluded).
func compileFSErrors(t *testing.T, fsys fstest.MapFS, mainName string) string {
	t.Helper()
	data, err := fsys.ReadFile(mainName)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = xsd.NewCompiler().Label(mainName).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
	requireCompileResultErr(t, err)
	require.NoError(t, collector.Close())
	_, errStr := partitionCompileErrors(collector.Errors())
	return errStr
}
