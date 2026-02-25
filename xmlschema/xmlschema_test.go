package xmlschema_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlschema"
	"github.com/stretchr/testify/require"
)

const testdataBase = "../testdata/libxml2/source"

// testCase represents a single golden-file test: one (schema, instance) pair.
type testCase struct {
	name     string // result file basename without .err
	xsdPath  string
	xmlPath  string
	errPath  string
	xmlBase  string // e.g. "all_0.xml" — used in output path prefix
}

// discoverTests discovers test cases from the result/schemas/ directory.
// Result files are named {name}_{variant}_{N}.err where:
//   - Schema: test/schemas/{name}_{variant}.xsd
//   - Instance: test/schemas/{name}_{N}.xml
//
// Some schemas use a shared .xsd without a variant suffix. We handle both.
func discoverTests(t *testing.T) []testCase {
	t.Helper()

	resultDir := filepath.Join(testdataBase, "result", "schemas")
	schemaDir := filepath.Join(testdataBase, "test", "schemas")

	entries, err := os.ReadDir(resultDir)
	require.NoError(t, err)

	// errRegex matches result filenames like "all_0_3.err" or "list0_0_2.err"
	errRegex := regexp.MustCompile(`^(.+)_(\d+)\.err$`)

	var cases []testCase
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".err") {
			continue
		}

		m := errRegex.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}

		schemaKey := m[1] // e.g. "all_0" or "list0_0"
		xmlIdx := m[2]    // e.g. "3"

		xsdPath := filepath.Join(schemaDir, schemaKey+".xsd")
		if _, err := os.Stat(xsdPath); err != nil {
			continue
		}

		// Figure out the XML instance name.
		// The XML base name is {baseName}_{xmlIdx}.xml where baseName
		// is derived by stripping the last _variant from schemaKey.
		xmlBase := findXMLInstance(schemaDir, schemaKey, xmlIdx)
		if xmlBase == "" {
			continue
		}

		xmlPath := filepath.Join(schemaDir, xmlBase)
		if _, err := os.Stat(xmlPath); err != nil {
			continue
		}

		cases = append(cases, testCase{
			name:    strings.TrimSuffix(e.Name(), ".err"),
			xsdPath: xsdPath,
			xmlPath: xmlPath,
			errPath: filepath.Join(resultDir, e.Name()),
			xmlBase: xmlBase,
		})
	}

	sort.Slice(cases, func(i, j int) bool { return cases[i].name < cases[j].name })
	return cases
}

// findXMLInstance resolves the XML instance filename for a given schemaKey and index.
// For schemaKey="all_0" and idx="3", it tries "all_3.xml".
// For schemaKey="list0_0" and idx="2", it tries "list0_2.xml".
func findXMLInstance(dir, schemaKey, idx string) string {
	// Split schemaKey into baseName + variant by finding the last underscore
	// that separates the base from the variant number.
	lastUnderscore := strings.LastIndex(schemaKey, "_")
	if lastUnderscore < 0 {
		return ""
	}

	baseName := schemaKey[:lastUnderscore]
	xmlName := baseName + "_" + idx + ".xml"

	if _, err := os.Stat(filepath.Join(dir, xmlName)); err == nil {
		return xmlName
	}

	return ""
}

// skip lists test groups that need unimplemented features.
var skip = map[string]string{
	// Substitution groups.
	"allsg":       "substitution groups not implemented",
	"subst-group":   "substitution groups not implemented",
	"subst-group-1": "substitution groups not implemented",
	// Import/include.
	"582887":   "schema import not implemented",
	"582906":   "schema import not implemented",
	"582906-1": "schema import not implemented",
	"582906-2": "schema import not implemented",
	"570702":   "schema import not implemented",
	"579746":   "schema import not implemented",
	"import":   "import not implemented",
	"import1":  "import not implemented",
	"import2":  "import not implemented",
	"include":  "include not implemented",
	"include1": "include not implemented",
	"include2": "include not implemented",
	"include3": "include not implemented",
	// Annotation errors.
	"annot-err": "annotation validation not implemented",
	// Any wildcard.
	"any":  "xs:any not implemented",
	"any1": "xs:any not implemented",
	"any2": "xs:any not implemented",
	"any3": "xs:any not implemented",
	"any4": "xs:any not implemented",
	"any5": "xs:any not implemented",
	"any6": "xs:any not implemented",
	"any7": "xs:any not implemented",
	"any8": "xs:any not implemented",
	// AnyAttribute tests.
	"anyAttr":                       "anyAttribute not implemented",
	"anyAttr1":                      "anyAttribute not implemented",
	"anyAttr-derive":                "anyAttribute not implemented",
	"anyAttr-derive1":               "anyAttribute not implemented",
	"anyAttr-derive2":               "anyAttribute not implemented",
	"anyAttr-derive-errors1":        "anyAttribute not implemented",
	"anyAttr-processContents-err1":  "anyAttribute not implemented",
	// Attribute tests.
	"attr":           "attribute validation not implemented",
	"attrgrp":        "attribute group not implemented",
	"src-attribute1": "attribute validation not implemented",
	"src-attribute2": "attribute validation not implemented",
	"src-attribute3-1":      "attribute validation not implemented",
	"src-attribute3-2-form": "attribute validation not implemented",
	"src-attribute3-2-st":   "attribute validation not implemented",
	"src-attribute3-2-type": "attribute validation not implemented",
	"src-attribute4":        "attribute validation not implemented",
	// Bug-related tests.
	"bug":    "bug-specific tests deferred",
	"bug143951": "bug-specific tests deferred",
	"bug145246": "bug-specific tests deferred",
	"bug167754": "bug-specific tests deferred",
	"bug303566": "bug-specific tests deferred",
	"bug306806": "bug-specific tests deferred",
	"bug310264": "bug-specific tests deferred",
	"bug312957": "bug-specific tests deferred",
	"bug321475": "bug-specific tests deferred",
	"bug322411": "bug-specific tests deferred",
	"bug455953": "bug-specific tests deferred",
	// Complex types / derivation.
	"complexdefault":       "complex default not implemented",
	"complex-type-extension": "complex type extension not implemented",
	"derivation-ok-extension":      "derivation not implemented",
	"derivation-ok-extension-err":  "derivation not implemented",
	"derivation-ok-restriction-2-1-1":     "derivation not implemented",
	"derivation-ok-restriction-4-1-err":   "derivation not implemented",
	// Changelog / misc.
	"changelog093": "deferred",
	// cos-st-restricts.
	"cos-st-restricts-1-2-err": "simple type restriction not implemented",
	// Decimal.
	"decimal-1": "simple type value checking not implemented",
	"decimal-2": "simple type value checking not implemented",
	"decimal-3": "simple type value checking not implemented",
	// Determinism.
	"deter0": "determinism checking not implemented",
	// Element error tests (schema parsing errors).
	"element-err":        "schema error reporting not implemented",
	"element-minmax-err": "schema error reporting not implemented",
	"src-element1":   "schema error reporting not implemented",
	"src-element2-1": "schema error reporting not implemented",
	"src-element2-2": "schema error reporting not implemented",
	"src-element3":   "schema error reporting not implemented",
	// Empty value tests (needs simple type checking).
	"empty-value": "simple type value checking not implemented",
	// Extension tests.
	"ext":        "extension not fully implemented",
	"ext1":       "extension not fully implemented",
	"extension0": "extension not fully implemented",
	"extension1": "extension not fully implemented",
	// Facets.
	"facet":                "facets not implemented",
	"facet-unionST-err1":  "facets not implemented",
	"hexbinary":            "simple type not implemented",
	"length3":              "facets not implemented",
	// Group.
	"group":  "xs:group not implemented",
	"group0": "xs:group not implemented",
	// IDC.
	"idc-keyref-err1": "identity constraints not implemented",
	// Issues.
	"issue40":  "deferred",
	"issue303": "deferred",
	"issue491": "deferred",
	// Item.
	"item": "deferred",
	// List (XSD list type, not sequence).
	"list":  "XSD list type not implemented",
	"list1": "XSD list type not implemented",
	// Mixed content with model group.
	"mixed1": "mixed content model groups deferred",
	// Namespace tests.
	"ns":  "namespace handling not fully implemented",
	"ns0": "namespace handling not fully implemented",
	// NotationType.
	"notation": "notation not implemented",
	// NVDCVE.
	"nvdcve": "deferred: complex real-world schema",
	// OSS-Fuzz.
	"oss-fuzz-51295": "deferred",
	// PO (purchase order).
	"po0": "deferred: complex schema",
	"po1": "deferred: complex schema",
	// Restriction.
	"restriction":       "restriction not implemented",
	"restriction-attr1": "restriction not implemented",
	"restriction-enum-1": "restriction not implemented",
	// SCC.
	"scc-no-xmlns": "schema component constraint not implemented",
	"scc-no-xsi":   "schema component constraint not implemented",
	// Seq with duplicate elements.
	"seq-dubl-elem1": "deferred: duplicate element in sequence",
	// Simple type tests.
	"stype": "simple type not implemented",
	// Union.
	"union":  "union not implemented",
	"union2": "union not implemented",
	// Unique / key / keyref.
	"unique": "identity constraints not implemented",
	"key":    "identity constraints not implemented",
	"keyref": "identity constraints not implemented",
	// XSD regexp (pattern facet).
	"regexp": "pattern facet not implemented",
	// Redefine.
	"redefine": "redefine not implemented",
	// VDV.
	"vdv-first0": "value-driven validation not implemented",
	"vdv-first2": "value-driven validation not implemented",
	"vdv-first3": "value-driven validation not implemented",
	"vdv-first4": "value-driven validation not implemented",
	"vdv-first5": "value-driven validation not implemented",
}

func shouldSkip(name string) string {
	// Check against all skip keys using prefix matching.
	for prefix, reason := range skip {
		if strings.HasPrefix(name, prefix+"_") || name == prefix {
			return reason
		}
	}
	return ""
}

func TestGoldenFiles(t *testing.T) {
	filterEnv := os.Getenv("HELIUM_XMLSCHEMA_TEST_FILES")

	cases := discoverTests(t)
	require.NotEmpty(t, cases, "no test cases discovered")

	passed := 0
	skipped := 0
	failed := 0

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if filterEnv != "" && !strings.Contains(tc.name, filterEnv) {
				t.Skip("filtered out by HELIUM_XMLSCHEMA_TEST_FILES")
				skipped++
				return
			}

			// Check skip list.
			// Extract base name (everything before the last two _N segments).
			baseName := extractBaseName(tc.name)
			if reason := shouldSkip(baseName); reason != "" {
				t.Skipf("skipping: %s", reason)
				skipped++
				return
			}

			// Read expected output.
			expected, err := os.ReadFile(tc.errPath)
			require.NoError(t, err)

			// Compile schema.
			schema, err := xmlschema.CompileFile(tc.xsdPath)
			require.NoError(t, err, "schema compilation failed for %s", tc.xsdPath)

			// Parse instance.
			xmlData, err := os.ReadFile(tc.xmlPath)
			require.NoError(t, err)
			doc, err := helium.Parse(xmlData)
			require.NoError(t, err, "XML parse failed for %s", tc.xmlPath)

			// Validate.
			filename := "./test/schemas/" + tc.xmlBase
			got := xmlschema.Validate(doc, schema, xmlschema.WithFilename(filename))

			if got == string(expected) {
				passed++
			} else {
				failed++
				require.Equal(t, string(expected), got)
			}
		})
	}

	t.Logf("Results: %d passed, %d failed, %d skipped (out of %d total)", passed, failed, skipped, len(cases))
}

// extractBaseName extracts the base name from a result file name.
// e.g. "all_0_3" → "all", "list0_0_2" → "list0", "elem0_0_0" → "elem0"
func extractBaseName(name string) string {
	// Result names are {base}_{variant}_{instance}.
	// We want {base} which is everything before the last two _N segments.
	parts := strings.Split(name, "_")
	if len(parts) < 3 {
		return name
	}
	return strings.Join(parts[:len(parts)-2], "_")
}
