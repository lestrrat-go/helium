package relaxng_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

const testdataBase = "../testdata/libxml2-compat/relaxng"

// testCase represents a single golden-file test: one (schema, instance) pair.
type testCase struct {
	name    string // result file basename without .err
	rngPath string // path to .rng schema
	xmlPath string // path to .xml instance
	errPath string // path to .err expected result
	rngBase string // e.g. "choice0.rng"
	xmlBase string // e.g. "choice0_3.xml"
}

// discoverTests discovers test cases from the result/ directory.
// Result files are named {base}_{N}.err where:
//   - Schema: test/{base}.rng
//   - Instance: test/{base}_{N}.xml
func discoverTests(t *testing.T) []testCase {
	t.Helper()

	resultDir := filepath.Join(testdataBase, "result")
	testDir := filepath.Join(testdataBase, "test")

	entries, err := os.ReadDir(resultDir)
	require.NoError(t, err)

	// errRegex matches result filenames like "302836_0.err" or "choice0_3.err"
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

		schemaKey := m[1] // e.g. "302836" or "choice0"
		xmlIdx := m[2]    // e.g. "0" or "3"

		rngPath := filepath.Join(testDir, schemaKey+".rng")
		if _, err := os.Stat(rngPath); err != nil {
			continue
		}

		xmlBase := schemaKey + "_" + xmlIdx + ".xml"
		xmlPath := filepath.Join(testDir, xmlBase)
		if _, err := os.Stat(xmlPath); err != nil {
			continue
		}

		cases = append(cases, testCase{
			name:    strings.TrimSuffix(e.Name(), ".err"),
			rngPath: rngPath,
			xmlPath: xmlPath,
			errPath: filepath.Join(resultDir, e.Name()),
			rngBase: schemaKey + ".rng",
			xmlBase: xmlBase,
		})
	}

	sort.Slice(cases, func(i, j int) bool { return cases[i].name < cases[j].name })
	return cases
}

// skip lists test groups that need unimplemented features.
// Keys are base names (matched by prefix).
var skip = map[string]string{
	// Broken XML
	"broken-xml": "broken XML schema parsing",
}

// skipExact lists specific test cases (by full test name) that need skipping
// when their group-level skip has been removed.
var skipExact = map[string]string{
	// Requires compile-time check: attribute name class overlap detection
	"tutor11_3_1": "compile-time: Attributes conflicts in group check",
	// Parser can't handle closing tag split across lines
	"spec_0": "XML parser: closing tag split across lines",
}

func shouldSkip(name string) string {
	// Check exact match first.
	if reason, ok := skipExact[name]; ok {
		return reason
	}
	// Check against all skip keys using prefix matching.
	for prefix, reason := range skip {
		if strings.HasPrefix(name, prefix+"_") || name == prefix {
			return reason
		}
	}
	return ""
}

func TestGoldenFiles(t *testing.T) {
	filterEnv := os.Getenv("HELIUM_RELAXNG_TEST_FILES")

	cases := discoverTests(t)
	require.NotEmpty(t, cases, "no test cases discovered")

	passed := 0
	skipped := 0
	failed := 0

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if filterEnv != "" && !strings.Contains(tc.name, filterEnv) {
				t.Skip("filtered out by HELIUM_RELAXNG_TEST_FILES")
				skipped++
				return
			}

			if reason := shouldSkip(tc.name); reason != "" {
				t.Skipf("skipping: %s", reason)
				skipped++
				return
			}

			// Read expected output.
			expected, err := os.ReadFile(tc.errPath)
			require.NoError(t, err)

			// Compile schema.
			rngFilename := "./test/relaxng/" + tc.rngBase
			grammar, err := relaxng.CompileFile(tc.rngPath, relaxng.WithSchemaFilename(rngFilename))
			require.NoError(t, err, "schema compilation returned error for %s", tc.rngPath)

			var got string
			if grammar.CompileErrors() != "" {
				got = grammar.CompileWarnings() + grammar.CompileErrors()
				got += "Relax-NG schema " + rngFilename + " failed to compile\n"
			} else {
				// Parse instance.
				xmlData, err := os.ReadFile(tc.xmlPath)
				require.NoError(t, err)
				doc, err := helium.Parse(xmlData)
				require.NoError(t, err, "XML parse failed for %s", tc.xmlPath)

				// Validate.
				filename := "./test/relaxng/" + tc.xmlBase
				got = grammar.CompileWarnings() + relaxng.Validate(doc, grammar, relaxng.WithFilename(filename))
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
