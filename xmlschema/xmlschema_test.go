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
	name    string // result file basename without .err
	xsdPath string
	xmlPath string
	errPath string
	xmlBase string // e.g. "all_0.xml" — used in output path prefix
	xsdBase string // e.g. "all_0.xsd" — used in schema error path prefix
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
			xsdBase: schemaKey + ".xsd",
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
// Keys are base names (matched by prefix).
var skip = map[string]string{
	// Import/include.
	"582906-2": "schema import error reporting not implemented",
	"import1":  "import error reporting not implemented",
	"include3": "include schema component constraint not implemented",
	// Annotation errors.
	"annot-err": "annotation validation not implemented",
	// Any wildcard.
	"any4": "xs:any determinism checking not implemented",
	// Complex schema error tests.
	"element-err": "complex element error reporting order mismatch",
	// Simple type/value validation.
	"changelog093":             "deferred",
	// Determinism.
	"deter0": "determinism checking not implemented",
	// IDC.
	"idc-keyref-err1": "identity constraints not implemented",
	// Issues.
}

// skipExact lists specific test cases (by full test name) that need skipping
// when their group-level skip has been removed.
var skipExact = map[string]string{
	"bug303566_1_1": "identity constraints not implemented",
	"bug312957_1_0":    "identity constraints not implemented",
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

			// Check skip list (exact match first, then base name prefix).
			if reason, ok := skipExact[tc.name]; ok {
				t.Skipf("skipping: %s", reason)
				skipped++
				return
			}
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
			xsdFilename := "./test/schemas/" + tc.xsdBase
			schema, err := xmlschema.CompileFile(tc.xsdPath, xmlschema.WithSchemaFilename(xsdFilename))
			require.NoError(t, err, "schema compilation failed for %s", tc.xsdPath)

			var got string
			if schema.CompileErrors() != "" {
				got = schema.CompileErrors()
			} else {
				// Parse instance.
				xmlData, err := os.ReadFile(tc.xmlPath)
				require.NoError(t, err)
				doc, err := helium.Parse(xmlData)
				require.NoError(t, err, "XML parse failed for %s", tc.xmlPath)

				// Validate. Prepend any compile warnings to the output.
				filename := "./test/schemas/" + tc.xmlBase
				got = schema.CompileWarnings() + xmlschema.Validate(doc, schema, xmlschema.WithFilename(filename))
			}

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
