package schematron_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
	"github.com/stretchr/testify/require"
)

const testdataBase = "../testdata/libxml2-compat/schematron"

type testCase struct {
	name    string // result file basename without .err
	sctPath string // schema file
	xmlPath string // instance file
	errPath string // expected output
	xmlBase string // e.g. "zvon1_0.xml"
}

// discoverTests discovers test cases from the result/ directory.
// Result files are named {base}_{N}.err where:
//   - Schema: test/{base}.sct
//   - Instance: test/{base}_{N}.xml
func discoverTests(t *testing.T) []testCase {
	t.Helper()

	resultDir := filepath.Join(testdataBase, "result")
	testDir := filepath.Join(testdataBase, "test")

	entries, err := os.ReadDir(resultDir)
	require.NoError(t, err)

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

		schemaBase := m[1] // e.g. "zvon1" or "cve-2025-49794"
		xmlIdx := m[2]     // e.g. "0"

		sctPath := filepath.Join(testDir, schemaBase+".sct")
		if _, err := os.Stat(sctPath); err != nil {
			continue
		}

		xmlBase := schemaBase + "_" + xmlIdx + ".xml"
		xmlPath := filepath.Join(testDir, xmlBase)
		if _, err := os.Stat(xmlPath); err != nil {
			continue
		}

		cases = append(cases, testCase{
			name:    strings.TrimSuffix(e.Name(), ".err"),
			sctPath: sctPath,
			xmlPath: xmlPath,
			errPath: filepath.Join(resultDir, e.Name()),
			xmlBase: xmlBase,
		})
	}

	sort.Slice(cases, func(i, j int) bool { return cases[i].name < cases[j].name })
	return cases
}

// skip lists test groups (matched by prefix).
var skip = map[string]string{}

// skipExact lists specific test cases (matched by exact name).
var skipExact = map[string]string{}

func shouldSkip(name string) string {
	for prefix, reason := range skip {
		if strings.HasPrefix(name, prefix+"_") || name == prefix {
			return reason
		}
	}
	return ""
}

func TestGoldenFiles(t *testing.T) {
	filterEnv := os.Getenv("HELIUM_SCHEMATRON_TEST_FILES")

	cases := discoverTests(t)
	require.NotEmpty(t, cases, "no test cases discovered")

	passed := 0
	skipped := 0
	failed := 0

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if filterEnv != "" && !strings.Contains(tc.name, filterEnv) {
				t.Skip("filtered out by HELIUM_SCHEMATRON_TEST_FILES")
				skipped++
				return
			}

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

			expected, err := os.ReadFile(tc.errPath)
			require.NoError(t, err)

			schema, err := schematron.CompileFile(tc.sctPath)
			require.NoError(t, err, "schema compilation failed for %s", tc.sctPath)

			var got string
			if schema.CompileErrors() != "" {
				got = schema.CompileWarnings() + schema.CompileErrors()
			} else {
				xmlData, err := os.ReadFile(tc.xmlPath)
				require.NoError(t, err)
				doc, err := helium.Parse(xmlData)
				require.NoError(t, err, "XML parse failed for %s", tc.xmlPath)

				filename := "./test/schematron/" + tc.xmlBase
				got = schema.CompileWarnings() + schematron.Validate(doc, schema, schematron.WithFilename(filename))
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
// e.g. "zvon1_0" -> "zvon1", "cve-2025-49794_0" -> "cve-2025-49794".
func extractBaseName(name string) string {
	idx := strings.LastIndex(name, "_")
	if idx < 0 {
		return name
	}
	return name[:idx]
}
