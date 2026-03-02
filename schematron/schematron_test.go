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

// partitionCompileErrors splits collected errors by severity.
// Errors with ErrorLevelFatal go to "errors"; all others to "warnings".
func partitionCompileErrors(errs []error) (warnings, errors string) {
	var w, e strings.Builder
	for _, err := range errs {
		if l, ok := err.(helium.ErrorLeveler); ok && l.ErrorLevel() == helium.ErrorLevelFatal {
			e.WriteString(err.Error())
		} else {
			w.WriteString(err.Error())
		}
	}
	return w.String(), e.String()
}

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

			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			schema, err := schematron.CompileFile(tc.sctPath, schematron.WithCompileErrorHandler(collector))
			require.NoError(t, err, "schema compilation failed for %s", tc.sctPath)
			collector.Close()
			compileWarnings, compileErrors := partitionCompileErrors(collector.Errors())

			var got string
			if compileErrors != "" {
				got = compileWarnings + compileErrors
			} else {
				xmlData, err := os.ReadFile(tc.xmlPath)
				require.NoError(t, err)
				doc, err := helium.Parse(t.Context(), xmlData)
				require.NoError(t, err, "XML parse failed for %s", tc.xmlPath)

				filename := "./test/schematron/" + tc.xmlBase
				err = schematron.Validate(doc, schema, schematron.WithFilename(filename))
				if err != nil {
					got = compileWarnings + err.Error()
				} else {
					got = compileWarnings + filename + " validates\n"
				}
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

// compileAndValidate is a helper that compiles a schema from a string and
// validates an XML document string against it, returning the validation error.
func compileAndValidate(t *testing.T, schemaXML, instanceXML string, opts ...schematron.ValidateOption) error {
	t.Helper()
	sDoc, err := helium.Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := schematron.Compile(sDoc)
	require.NoError(t, err)
	doc, err := helium.Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)
	return schematron.Validate(doc, schema, opts...)
}

func TestWithQuiet(t *testing.T) {
	const sct = `<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern name="test">
    <rule context="AAA">
      <assert test="BBB">BBB element is missing.</assert>
      <report test="BBB">BBB element is present.</report>
    </rule>
  </pattern>
</schema>`

	t.Run("failing document", func(t *testing.T) {
		// Without quiet: per-error lines + "fails to validate"
		err := compileAndValidate(t, sct, `<AAA><CCC/></AAA>`, schematron.WithFilename("test.xml"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "schematron error")
		require.Contains(t, err.Error(), "fails to validate")

		// With quiet: only "fails to validate" line, no per-error lines
		quietErr := compileAndValidate(t, sct, `<AAA><CCC/></AAA>`, schematron.WithFilename("test.xml"), schematron.WithQuiet())
		require.Error(t, quietErr)
		require.NotContains(t, quietErr.Error(), "schematron error")
		require.Equal(t, "test.xml fails to validate\n", quietErr.Error())
	})

	t.Run("passing document", func(t *testing.T) {
		// This schema only has a report (no assert), so a doc without BBB validates.
		const reportOnly = `<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern name="test">
    <rule context="AAA">
      <report test="CCC">CCC is present.</report>
    </rule>
  </pattern>
</schema>`
		// Without quiet: report fires, "fails to validate"
		err := compileAndValidate(t, reportOnly, `<AAA><CCC/></AAA>`, schematron.WithFilename("test.xml"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "CCC is present")

		// With quiet: report suppressed, but since the report fired, it still "fails to validate"
		quietErr := compileAndValidate(t, reportOnly, `<AAA><CCC/></AAA>`, schematron.WithFilename("test.xml"), schematron.WithQuiet())
		require.Error(t, quietErr)
		require.NotContains(t, quietErr.Error(), "CCC is present")
		require.Equal(t, "test.xml fails to validate\n", quietErr.Error())
	})
}

func TestWithErrorHandler(t *testing.T) {
	const sct = `<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern name="test">
    <rule context="AAA">
      <assert test="BBB">BBB element is missing.</assert>
      <assert test="@name">AAA needs name attribute.</assert>
    </rule>
  </pattern>
</schema>`

	t.Run("errors delivered to handler", func(t *testing.T) {
		var errors []schematron.ValidationError
		handler := schematron.ErrorHandlerFunc(func(e schematron.ValidationError) {
			errors = append(errors, e)
		})

		err := compileAndValidate(t, sct, `<AAA><CCC/></AAA>`,
			schematron.WithFilename("test.xml"),
			schematron.WithErrorHandler(handler),
		)

		// Error output should not contain per-error lines
		require.Error(t, err)
		require.NotContains(t, err.Error(), "schematron error")
		require.Contains(t, err.Error(), "fails to validate")

		// Handler should have received both errors
		require.Len(t, errors, 2)
		require.Equal(t, "BBB element is missing.", errors[0].Message)
		require.Equal(t, "AAA", errors[0].Element)
		require.Equal(t, "/AAA", errors[0].Path)
		require.Equal(t, "test.xml", errors[0].Filename)
		require.Equal(t, "AAA needs name attribute.", errors[1].Message)
	})

	t.Run("passing document no handler calls", func(t *testing.T) {
		var errors []schematron.ValidationError
		handler := schematron.ErrorHandlerFunc(func(e schematron.ValidationError) {
			errors = append(errors, e)
		})

		err := compileAndValidate(t, sct, `<AAA name="x"><BBB/></AAA>`,
			schematron.WithFilename("test.xml"),
			schematron.WithErrorHandler(handler),
		)

		require.NoError(t, err)
		require.Empty(t, errors)
	})

	t.Run("quiet with error handler delivers errors", func(t *testing.T) {
		// When both quiet and error handler are set, errors go to the handler
		var errors []schematron.ValidationError
		handler := schematron.ErrorHandlerFunc(func(e schematron.ValidationError) {
			errors = append(errors, e)
		})

		err := compileAndValidate(t, sct, `<AAA><CCC/></AAA>`,
			schematron.WithFilename("test.xml"),
			schematron.WithQuiet(),
			schematron.WithErrorHandler(handler),
		)

		require.Error(t, err)
		require.NotContains(t, err.Error(), "schematron error")
		require.Contains(t, err.Error(), "fails to validate")
		require.Len(t, errors, 2)
	})
}
