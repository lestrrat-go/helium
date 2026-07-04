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

// TestAttrGroupCycleStillCut verifies an INDIRECT attribute-group reference cycle
// (h -> i -> h) is reported as a circular reference (src-attribute_group.3),
// matching the direct-self-reference behavior — an invalid (circular) schema must
// NOT compile silently. The cycle's back-edge is cut so the duplicate-detection
// walk still terminates without misreporting the structurally-valid names 'a'/'b'
// as duplicates.
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

	_, errs := compileWithErrors(t, schemaXML)
	require.Contains(t, errs, "Circular reference to the attribute group",
		"an indirect attribute-group cycle must be reported as circular; got: %q", errs)
	// The back-edge cut must not turn the structurally-valid names into duplicates.
	require.NotContains(t, errs, "Duplicate attribute use 'a'",
		"a cycle must not be misreported as a duplicate of 'a'; got: %q", errs)
	require.NotContains(t, errs, "Duplicate attribute use 'b'",
		"a cycle must not be misreported as a duplicate of 'b'; got: %q", errs)
}

// TestAttrGroupIndirectCycleThreeNode verifies a 3-node indirect cycle
// (a -> b -> c -> a) is reported as a circular attribute-group reference rather
// than silently compiling.
func TestAttrGroupIndirectCycleThreeNode(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="a">
    <xs:attribute name="pa" type="xs:string"/>
    <xs:attributeGroup ref="b"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="b">
    <xs:attribute name="pb" type="xs:string"/>
    <xs:attributeGroup ref="c"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="c">
    <xs:attribute name="pc" type="xs:string"/>
    <xs:attributeGroup ref="a"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="a"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`

	_, errs := compileWithErrors(t, schemaXML)
	require.Contains(t, errs, "Circular reference to the attribute group",
		"a 3-node indirect attribute-group cycle must be reported as circular; got: %q", errs)
	require.NotContains(t, errs, "Duplicate attribute use",
		"a cycle must not be misreported as a duplicate attribute use; got: %q", errs)
}

// TestAttrGroupDiamondNoCycle verifies a NON-cyclic diamond of attribute-group
// references (a -> b, a -> c, b -> d, c -> d) compiles WITHOUT a false circular-
// reference error: d is reachable by two paths but no path returns to a node
// already on the reference chain.
func TestAttrGroupDiamondNoCycle(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="d">
    <xs:attribute name="pd" type="xs:string"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="b">
    <xs:attribute name="pb" type="xs:string"/>
    <xs:attributeGroup ref="d"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="c">
    <xs:attribute name="pc" type="xs:string"/>
    <xs:attributeGroup ref="d"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="a">
    <xs:attribute name="pa" type="xs:string"/>
    <xs:attributeGroup ref="b"/>
    <xs:attributeGroup ref="c"/>
  </xs:attributeGroup>
  <xs:element name="root">
    <xs:complexType>
      <xs:attributeGroup ref="a"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	_, errs := compileWithErrors(t, schemaXML)
	require.NotContains(t, errs, "Circular reference to the attribute group",
		"a non-cyclic diamond must not be reported as circular; got: %q", errs)
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

// TestAttrGroupIndirectCycleBackEdgeLine verifies that an INDIRECT
// attribute-group cycle's circular-reference diagnostic is attributed to the
// BACK-EDGE <xs:attributeGroup ref="..."> element's line — the ref that actually
// closed the cycle — not to the owning group's declaration line. The owner
// declaration and the back-edge ref are on deliberately different lines.
func TestAttrGroupIndirectCycleBackEdgeLine(t *testing.T) {
	t.Parallel()

	// DFS visits roots in sorted QName order (h before i): visit(h)->visit(i),
	// where i's <xs:attributeGroup ref="h"/> is the back-edge closing the cycle.
	// Line numbers (1-based): line 1 is <xs:schema>.
	//   2: <xs:attributeGroup name="h">
	//   3:   <xs:attribute name="a"/>
	//   4:   <xs:attributeGroup ref="i"/>
	//   5: </xs:attributeGroup>
	//   6: <xs:attributeGroup name="i">   <- owner declaration line
	//   7:   <xs:attribute name="b"/>
	//   8:   <xs:attributeGroup ref="h"/> <- BACK-EDGE ref line (expected)
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

	_, errs := compileWithErrors(t, schemaXML)
	require.Contains(t, errs, "Circular reference to the attribute group",
		"an indirect attribute-group cycle must be reported as circular; got: %q", errs)
	// The diagnostic must cite the back-edge ref line (8), not the owner 'i'
	// declaration line (6).
	require.Contains(t, errs, ":8:",
		"circular-reference diagnostic must cite the back-edge ref line; got: %q", errs)
	require.NotContains(t, errs, ":6:",
		"circular-reference diagnostic must not cite the owner group declaration line; got: %q", errs)
}

// TestAttrGroupIndirectCycleBackEdgeSourceInclude verifies that an INDIRECT
// attribute-group cycle spanning an xs:include is attributed to the FILE and
// LINE of the back-edge <xs:attributeGroup ref="..."> element that closed the
// cycle, not to the owning group's declaration (which lives in a different
// file). This is the cross-file case the per-edge source map exists for.
func TestAttrGroupIndirectCycleBackEdgeSourceInclude(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "agcycle_main.xsd"
		incXSD  = "agcycle_inc.xsd"
	)

	// Group h is declared in the main file and refs i; group i is declared in the
	// included file and refs h. DFS visits h (main) then i (inc): i's ref-to-h is
	// the back-edge. The diagnostic must cite the included file's ref line, since
	// that ref element lives in the included file.
	//
	// Included file line numbers (1-based):
	//   1: <xs:schema>
	//   2: <xs:attributeGroup name="i">   <- owner declaration
	//   3:   <xs:attribute name="b"/>
	//   4:   <xs:attributeGroup ref="h"/> <- BACK-EDGE ref (expected line 4)
	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="agcycle_inc.xsd"/>
  <xs:attributeGroup name="h">
    <xs:attribute name="a" type="xs:string"/>
    <xs:attributeGroup ref="i"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="h"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
		incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="i">
    <xs:attribute name="b" type="xs:string"/>
    <xs:attributeGroup ref="h"/>
  </xs:attributeGroup>
</xs:schema>`)},
	}

	errStr := compileFSErrors(t, fsys, mainXSD)
	require.Contains(t, errStr, "Circular reference to the attribute group",
		"expected the circular-reference diagnostic; got: %q", errStr)
	// The back-edge ref lives in the included file at line 4.
	require.Contains(t, errStr, incXSD+":4:",
		"circular-reference diagnostic must cite the back-edge ref file and line; got: %q", errStr)
	require.False(t, strings.Contains(errStr, mainXSD+":"),
		"circular-reference diagnostic must not cite the file holding the owner declaration; got: %q", errStr)
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

// TestAttrGroupRefResolution verifies that an <xs:attributeGroup ref="..."> whose
// ref does not resolve to a globally-declared attribute group is rejected in XSD
// 1.0 (src-resolve / Attribute Group Definition Representation OK 3): a ref naming
// a component in the wrong symbol space (a complexType or a global attribute of
// the same name), a name declared nowhere, and an empty ref value are all schema
// errors, while a ref to a genuine attribute group compiles.
func TestAttrGroupRefResolution(t *testing.T) {
	t.Parallel()

	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`
	const tail = `</xs:schema>`

	for _, tc := range []struct {
		name   string
		body   string
		reject bool
		// rejectMsg, when non-empty, is the substring the rejection diagnostic must
		// contain; it defaults to the "does not resolve" wording. A PRESENT-but-empty
		// ref="" is a lexically-invalid (empty) QName caught at the read point, so it
		// reports the shared invalid-QName diagnostic instead.
		rejectMsg string
	}{
		{
			name: "valid ref to a global attribute group",
			body: `
  <xs:attributeGroup name="g"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>
  <xs:complexType name="t"><xs:attributeGroup ref="g"/></xs:complexType>`,
			reject: false,
		},
		{
			name: "ref names a complexType (wrong symbol space)",
			body: `
  <xs:complexType name="foo"><xs:sequence><xs:element name="e"/></xs:sequence></xs:complexType>
  <xs:complexType name="t"><xs:attributeGroup ref="foo"/></xs:complexType>`,
			reject: true,
		},
		{
			name: "ref names a global attribute (wrong symbol space)",
			body: `
  <xs:attribute name="foo" type="xs:string"/>
  <xs:complexType name="t"><xs:attributeGroup ref="foo"/></xs:complexType>`,
			reject: true,
		},
		{
			name:   "ref names nothing",
			body:   `<xs:complexType name="t"><xs:attributeGroup ref="nope"/></xs:complexType>`,
			reject: true,
		},
		{
			name:      "empty ref value",
			body:      `<xs:complexType name="t"><xs:attributeGroup ref=""/></xs:complexType>`,
			reject:    true,
			rejectMsg: "The QName value '' is not a valid QName.",
		},
		{
			name: "nested ref inside a global group names a simpleType",
			body: `
  <xs:simpleType name="st"><xs:restriction base="xs:string"/></xs:simpleType>
  <xs:attributeGroup name="g"><xs:attributeGroup ref="st"/></xs:attributeGroup>
  <xs:complexType name="t"><xs:attributeGroup ref="g"/></xs:complexType>`,
			reject: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := compileWithErrors(t, head+tc.body+tail)
			if tc.reject {
				want := tc.rejectMsg
				if want == "" {
					want = "does not resolve to a(n) attribute group definition"
				}
				require.Contains(t, errs, want,
					"an unresolved attribute-group reference must be rejected; got: %q", errs)
				return
			}
			require.NotContains(t, errs, "does not resolve to a(n) attribute group definition",
				"a valid attribute-group reference must compile; got: %q", errs)
		})
	}
}
