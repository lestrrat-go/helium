package xsd_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// The nested-schema load taxonomy (xs:include / xs:import / xs:redefine targets)
// is three-way:
//
//   - a benign FETCH/RESOLUTION miss (the schemaLocation hint cannot be resolved
//     or read) is demoted to a WARNING and the composition element skipped;
//   - a fetched-but-invalid CONTENT failure (malformed XML, a non-xs:schema root,
//     or a well-formed schema that fails a schema-construction rule such as a
//     top-level complexType missing its @name) is FATAL — the located document
//     exists but is invalid, so it must not be silently downgraded to a warning;
//   - a POLICY/SECURITY denial (the default deny-all FS, a byte-cap breach, a
//     path escape, an over-deep chain) is FATAL.
//
// These tests pin all three so the classification cannot regress to
// fetch-vs-content-only (which would demote a post-fetch content failure) or to
// a fail-OPEN default (which would demote a policy denial).

const (
	taxMainXSD   = "main.xsd"
	taxNestedXSD = "nested.xsd"
	taxImpXSD    = "imp.xsd"
	taxIncXSD    = "inc.xsd"

	taxInclude  = "include"
	taxImport   = "import"
	taxRedefine = "redefine"

	// A well-formed schema whose ONLY defect is a schema-construction rule
	// violation: a top-level complexType with no @name. Fetching succeeds and the
	// XML parses; the failure is post-fetch CONTENT, so it must be fatal.
	nestedContentInvalidSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType/>
</xs:schema>`
	// A no-namespace variant for the import path (the imported schema must carry
	// the imported namespace so it is loaded, then fails on the unnamed type).
	nestedContentInvalidSchemaNSB = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:b">
  <xs:complexType/>
</xs:schema>`
	// Not well-formed XML.
	nestedMalformedSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><unclosed></xs:schema>`
	// Well-formed XML whose document element is not <xs:schema>.
	nestedNonSchemaRoot = `<notASchema/>`
)

func compileSchemaFromFS(t *testing.T, fsys fs.FS) error {
	t.Helper()
	data, err := fs.ReadFile(fsys, taxMainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	_, cerr := xsd.NewCompiler().FS(fsys).Compile(t.Context(), doc)
	return cerr
}

// compileSchemaFromFSVersion11 is compileSchemaFromFS at XSD 1.1, needed for the
// xs:override taxonomy (xs:override is a 1.1-only construct).
func compileSchemaFromFSVersion11(t *testing.T, fsys fs.FS) error {
	t.Helper()
	data, err := fs.ReadFile(fsys, taxMainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	_, cerr := xsd.NewCompiler().FS(fsys).Version(xsd.Version11).Compile(t.Context(), doc)
	return cerr
}

// TestNestedFetchMissIsWarning verifies that a genuinely-missing nested target
// (the schemaLocation hint resolves to nothing) is demoted to a warning: the
// main schema — which does not depend on the missing target's declarations —
// compiles cleanly.
func TestNestedFetchMissIsWarning(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		taxInclude: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="absent.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`,
		taxImport: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:a">
  <xs:import namespace="urn:b" schemaLocation="absent.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`,
		taxRedefine: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="absent.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`,
	}
	for name, main := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// MapFS (not DenyAll) returns fs.ErrNotExist for an absent name — a
			// genuine fetch miss, not a policy denial.
			fsys := fstest.MapFS{taxMainXSD: &fstest.MapFile{Data: []byte(main)}}
			require.NoError(t, compileSchemaFromFS(t, fsys),
				"a missing %s target is a hint miss and must be demoted to a warning", name)
		})
	}
}

// TestNestedContentInvalidIsFatal verifies that a fetched-but-invalid nested
// target is fatal for every composition element and phase — malformed XML, a
// non-xs:schema root, and (the phase-aware regression guard) a well-formed schema
// that fails a post-fetch schema-construction rule.
func TestNestedContentInvalidIsFatal(t *testing.T) {
	t.Parallel()

	mainInclude := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="nested.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`
	mainImport := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:a">
  <xs:import namespace="urn:b" schemaLocation="nested.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`
	// A redefine target must have the redefining schema's targetNamespace (or
	// none); use no targetNamespace so the redefine reaches content parsing.
	mainRedefine := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="nested.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	comps := map[string]string{taxInclude: mainInclude, taxImport: mainImport, taxRedefine: mainRedefine}
	contents := map[string]string{
		"malformed-xml":      nestedMalformedSchema,
		"non-xs:schema-root": nestedNonSchemaRoot,
		"missing-name":       nestedContentInvalidSchema,
	}
	// The import nested schema must carry the imported namespace so it is loaded.
	importNested := map[string]string{
		"malformed-xml":      nestedMalformedSchema,
		"non-xs:schema-root": nestedNonSchemaRoot,
		"missing-name":       nestedContentInvalidSchemaNSB,
	}

	for comp, main := range comps {
		for cname, nested := range contents {
			if comp == taxImport {
				nested = importNested[cname]
			}
			t.Run(comp+"/"+cname, func(t *testing.T) {
				t.Parallel()
				fsys := fstest.MapFS{
					taxMainXSD:   &fstest.MapFile{Data: []byte(main)},
					taxNestedXSD: &fstest.MapFile{Data: []byte(nested)},
				}
				require.Error(t, compileSchemaFromFS(t, fsys),
					"a fetched-but-invalid %s target (%s) must abort compilation, not warn", comp, cname)
			})
		}
	}
}

// TestNestedIncludeInImportedSchemaContentInvalidIsFatal pins the imported-schema
// nested path: an include INSIDE an imported schema whose target is
// well-formed-but-invalid content (a complexType missing @name) must be fatal,
// not swallowed by the import loader's warn-on-miss fallback.
func TestNestedIncludeInImportedSchemaContentInvalidIsFatal(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		taxMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:a">
  <xs:import namespace="urn:b" schemaLocation="imp.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
		taxImpXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:b">
  <xs:include schemaLocation="inc.xsd"/>
</xs:schema>`)},
		taxIncXSD: &fstest.MapFile{Data: []byte(nestedContentInvalidSchemaNSB)},
	}
	require.Error(t, compileSchemaFromFS(t, fsys),
		"a content-invalid include nested inside an imported schema must abort compilation")
}

// TestNestedIncludeInImportedSchemaFetchMissIsWarning is the complement: an
// include inside an imported schema whose target is merely ABSENT is a hint miss
// and must be demoted to a warning, so the import (and the whole compile)
// succeeds.
func TestNestedIncludeInImportedSchemaFetchMissIsWarning(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		taxMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:a">
  <xs:import namespace="urn:b" schemaLocation="imp.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
		taxImpXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:b">
  <xs:include schemaLocation="absent.xsd"/>
  <xs:element name="fromImport" type="xs:string"/>
</xs:schema>`)},
	}
	require.NoError(t, compileSchemaFromFS(t, fsys),
		"a missing include nested inside an imported schema is a hint miss and must warn")
}

// TestNestedPolicyDenialIsFatal verifies that the default deny-all FS makes a
// nested load fatal for every composition element: DenyAll returns
// fs.ErrNotExist for every Open (indistinguishable by errno from a genuine
// miss), so the FS type — not the errno — must decide, and a denial must never be
// demoted to a fetch-miss warning.
func TestNestedPolicyDenialIsFatal(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		taxInclude: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`,
		taxImport: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:a">
  <xs:import namespace="urn:b" schemaLocation="inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`,
		taxRedefine: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`,
	}
	for name, main := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
			require.NoError(t, err)
			// No .FS() call: the compiler's default fsys is the deny-all FS.
			_, cerr := xsd.NewCompiler().BaseDir(".").Compile(t.Context(), doc)
			require.Error(t, cerr,
				"a %s denied by the default deny-all FS must abort compilation, not warn", name)
			require.ErrorIs(t, cerr, fs.ErrNotExist)
		})
	}
}

// The xs:override subsystem (XSD 1.1) applies the SAME three-way load taxonomy as
// xs:include/xs:redefine/xs:import: a fetch miss of the override TARGET — or of an
// xs:include/xs:override/xs:redefine nested INSIDE the overridden document — is a
// schemaLocation-hint miss demoted to a warning; a content or policy failure is
// fatal.

// overrideMainWithFooChild is a top-level overriding schema (no targetNamespace)
// whose xs:override points at loc and carries one replacement child. The root
// element does not depend on the override, so a demoted-to-warning miss still
// compiles.
func overrideMainWithFooChild(loc string) string {
	return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:override schemaLocation="` + loc + `">
    <xs:element name="foo" type="xs:string"/>
  </xs:override>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`
}

// TestNestedOverrideTargetFetchMissIsWarning: a top-level xs:override whose TARGET
// document is absent is a hint miss demoted to a warning (the overriding schema
// still compiles).
func TestNestedOverrideTargetFetchMissIsWarning(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{taxMainXSD: &fstest.MapFile{Data: []byte(overrideMainWithFooChild("absent.xsd"))}}
	require.NoError(t, compileSchemaFromFSVersion11(t, fsys),
		"a missing xs:override target is a hint miss and must be demoted to a warning")
}

// TestNestedRedefineInOverrideFetchMissIsWarning pins the specific corner: an
// xs:redefine nested inside an overridden document whose redefine target is absent
// must be demoted to a warning, not made fatal by the override transform.
func TestNestedRedefineInOverrideFetchMissIsWarning(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		taxMainXSD: &fstest.MapFile{Data: []byte(overrideMainWithFooChild("nested.xsd"))},
		// The override target is present but carries an xs:redefine to an absent
		// document. Its foo declaration lets the override child match a component.
		taxNestedXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="absent.xsd"/>
  <xs:element name="foo" type="xs:string"/>
</xs:schema>`)},
	}
	require.NoError(t, compileSchemaFromFSVersion11(t, fsys),
		"a missing xs:redefine target nested inside an overridden document is a hint miss and must warn")
}

// TestNestedOverrideTargetContentInvalidIsFatal: a fetched-but-invalid override
// target (malformed XML, a non-xs:schema root, or a well-formed schema that fails
// a construction rule) is fatal, never demoted to a warning.
func TestNestedOverrideTargetContentInvalidIsFatal(t *testing.T) {
	t.Parallel()
	contents := map[string]string{
		"malformed-xml":      nestedMalformedSchema,
		"non-xs:schema-root": nestedNonSchemaRoot,
		"missing-name":       nestedContentInvalidSchema,
	}
	for cname, nested := range contents {
		t.Run(cname, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{
				taxMainXSD:   &fstest.MapFile{Data: []byte(overrideMainWithFooChild("nested.xsd"))},
				taxNestedXSD: &fstest.MapFile{Data: []byte(nested)},
			}
			require.Error(t, compileSchemaFromFSVersion11(t, fsys),
				"a fetched-but-invalid xs:override target (%s) must abort compilation, not warn", cname)
		})
	}
}

// TestNestedRedefineInOverrideContentInvalidIsFatal: an xs:redefine nested inside
// an overridden document whose target is well-formed-but-invalid content is fatal.
func TestNestedRedefineInOverrideContentInvalidIsFatal(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		taxMainXSD: &fstest.MapFile{Data: []byte(overrideMainWithFooChild("nested.xsd"))},
		taxNestedXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="inc.xsd"/>
  <xs:element name="foo" type="xs:string"/>
</xs:schema>`)},
		taxIncXSD: &fstest.MapFile{Data: []byte(nestedContentInvalidSchema)},
	}
	require.Error(t, compileSchemaFromFSVersion11(t, fsys),
		"a content-invalid xs:redefine target nested inside an overridden document must abort compilation")
}

// TestNestedOverrideTargetPolicyDenialIsFatal: the default deny-all FS makes an
// xs:override target load fatal — a policy denial must never be demoted to a
// fetch-miss warning.
func TestNestedOverrideTargetPolicyDenialIsFatal(t *testing.T) {
	t.Parallel()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(overrideMainWithFooChild("nested.xsd")))
	require.NoError(t, err)
	// No .FS() call: the compiler's default fsys is the deny-all FS.
	_, cerr := xsd.NewCompiler().BaseDir(".").Version(xsd.Version11).Compile(t.Context(), doc)
	require.Error(t, cerr, "an xs:override target denied by the default deny-all FS must abort compilation, not warn")
	require.ErrorIs(t, cerr, fs.ErrNotExist)
}
