package schematron_test

import (
	"context"
	"errors"
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
			schema, err := schematron.NewCompiler().ErrorHandler(collector).CompileFile(t.Context(), tc.sctPath)
			require.NoError(t, err, "schema compilation failed for %s", tc.sctPath)
			_ = collector.Close()
			compileWarnings, compileErrors := partitionCompileErrors(collector.Errors())

			var got string
			if compileErrors != "" {
				got = compileWarnings + compileErrors
			} else {
				xmlData, err := os.ReadFile(tc.xmlPath)
				require.NoError(t, err)
				doc, err := helium.NewParser().Parse(t.Context(), xmlData)
				require.NoError(t, err, "XML parse failed for %s", tc.xmlPath)

				filename := "./test/schematron/" + tc.xmlBase
				valCollector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
				err = schematron.NewValidator(schema).Filename(filename).ErrorHandler(valCollector).Validate(t.Context(), doc)
				_ = valCollector.Close()
				var valErrs strings.Builder
				for _, e := range valCollector.Errors() {
					valErrs.WriteString(e.Error())
				}
				if err != nil {
					got = compileWarnings + valErrs.String() + filename + " fails to validate\n"
				} else {
					got = compileWarnings + valErrs.String() + filename + " validates\n"
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

func TestWithQuiet(t *testing.T) {
	const sct = `<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern name="test">
    <rule context="AAA">
      <assert test="BBB">BBB element is missing.</assert>
      <report test="BBB">BBB element is present.</report>
    </rule>
  </pattern>
</schema>`

	// Compile the schema once for all sub-tests.
	p := helium.NewParser()
	sDoc, err := p.Parse(t.Context(), []byte(sct))
	require.NoError(t, err)
	schema, err := schematron.NewCompiler().Compile(t.Context(), sDoc)
	require.NoError(t, err)

	t.Run("failing document", func(t *testing.T) {
		doc, err := p.Parse(t.Context(), []byte(`<AAA><CCC/></AAA>`))
		require.NoError(t, err)

		// Without quiet: sentinel error returned, handler receives errors
		var collected []*schematron.ValidationError
		handler := validationErrorCollector{errors: &collected}
		err = schematron.NewValidator(schema).Filename("test.xml").ErrorHandler(handler).Validate(t.Context(), doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.NotEmpty(t, collected)

		// With quiet + handler: handler still receives errors
		var quietCollected []*schematron.ValidationError
		quietHandler := validationErrorCollector{errors: &quietCollected}
		quietErr := schematron.NewValidator(schema).Filename("test.xml").Quiet().ErrorHandler(quietHandler).Validate(t.Context(), doc)
		require.ErrorIs(t, quietErr, schematron.ErrValidationFailed)
		require.Empty(t, quietCollected)
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
		rDoc, err := p.Parse(t.Context(), []byte(reportOnly))
		require.NoError(t, err)
		rSchema, err := schematron.NewCompiler().Compile(t.Context(), rDoc)
		require.NoError(t, err)

		doc, err := p.Parse(t.Context(), []byte(`<AAA><CCC/></AAA>`))
		require.NoError(t, err)

		// Without quiet: report fires
		var collected []*schematron.ValidationError
		handler := validationErrorCollector{errors: &collected}
		err = schematron.NewValidator(rSchema).Filename("test.xml").ErrorHandler(handler).Validate(t.Context(), doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.NotEmpty(t, collected)
		require.Contains(t, collected[0].Message, "CCC is present")

		// With quiet: report suppressed (no errors delivered to handler)
		var quietCollected []*schematron.ValidationError
		quietHandler := validationErrorCollector{errors: &quietCollected}
		quietErr := schematron.NewValidator(rSchema).Filename("test.xml").Quiet().ErrorHandler(quietHandler).Validate(t.Context(), doc)
		require.ErrorIs(t, quietErr, schematron.ErrValidationFailed)
		require.Empty(t, quietCollected)
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

	p := helium.NewParser()
	sDoc, err := p.Parse(t.Context(), []byte(sct))
	require.NoError(t, err)
	schema, err := schematron.NewCompiler().Compile(t.Context(), sDoc)
	require.NoError(t, err)

	t.Run("errors delivered to handler", func(t *testing.T) {
		var collected []*schematron.ValidationError
		handler := validationErrorCollector{errors: &collected}

		doc, err := p.Parse(t.Context(), []byte(`<AAA><CCC/></AAA>`))
		require.NoError(t, err)

		err = schematron.NewValidator(schema).
			Filename("test.xml").
			ErrorHandler(handler).
			Validate(t.Context(), doc)

		require.ErrorIs(t, err, schematron.ErrValidationFailed)

		// Handler should have received both errors
		require.Len(t, collected, 2)
		require.Equal(t, "BBB element is missing.", collected[0].Message)
		require.Equal(t, "AAA", collected[0].Element)
		require.Equal(t, "/AAA", collected[0].Path)
		require.Equal(t, "test.xml", collected[0].Filename)
		require.Equal(t, "AAA needs name attribute.", collected[1].Message)
	})

	t.Run("passing document no handler calls", func(t *testing.T) {
		var collected []*schematron.ValidationError
		handler := validationErrorCollector{errors: &collected}

		doc, err := p.Parse(t.Context(), []byte(`<AAA name="x"><BBB/></AAA>`))
		require.NoError(t, err)

		err = schematron.NewValidator(schema).
			Filename("test.xml").
			ErrorHandler(handler).
			Validate(t.Context(), doc)

		require.NoError(t, err)
		require.Empty(t, collected)
	})

	t.Run("quiet suppresses handler delivery", func(t *testing.T) {
		var collected []*schematron.ValidationError
		handler := validationErrorCollector{errors: &collected}

		doc, err := p.Parse(t.Context(), []byte(`<AAA><CCC/></AAA>`))
		require.NoError(t, err)

		err = schematron.NewValidator(schema).
			Filename("test.xml").
			Quiet().
			ErrorHandler(handler).
			Validate(t.Context(), doc)

		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Empty(t, collected)
	})
}

// validationErrorCollector implements helium.ErrorHandler and extracts
// *schematron.ValidationError values via errors.As.
type validationErrorCollector struct {
	errors *[]*schematron.ValidationError
}

func (c validationErrorCollector) Handle(_ context.Context, err error) {
	var ve *schematron.ValidationError
	if errors.As(err, &ve) {
		*c.errors = append(*c.errors, ve)
	}
}

// compileTestSchema compiles a schema and returns the compiled schema and
// any compile error strings concatenated.
func compileTestSchema(t *testing.T, xml string) (*schematron.Schema, string) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, err := schematron.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	var b strings.Builder
	for _, e := range collector.Errors() {
		b.WriteString(e.Error())
	}
	return schema, b.String()
}

// validateAndCollect validates a document and collects *ValidationError values.
func validateAndCollect(t *testing.T, schema *schematron.Schema, doc *helium.Document) ([]*schematron.ValidationError, error) {
	t.Helper()
	var collected []*schematron.ValidationError
	handler := validationErrorCollector{errors: &collected}
	err := schematron.NewValidator(schema).ErrorHandler(handler).Validate(t.Context(), doc)
	return collected, err
}

func collectedString(collected []*schematron.ValidationError) string {
	var sb strings.Builder
	for _, ve := range collected {
		sb.WriteString(ve.Error())
	}
	return sb.String()
}

func TestCompileEmptyContext(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context=""><assert test="true()">ok</assert></rule></pattern>
	</schema>`)
	require.Contains(t, errs, "rule has an empty context attribute")
}

func TestCompileRuleNoAssert(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context="*"></rule></pattern>
	</schema>`)
	require.Contains(t, errs, "rule has no assert nor report element")
}

func TestCompilePatternNoRules(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern></pattern>
	</schema>`)
	require.Contains(t, errs, "Pattern has no rule element")
}

func TestCompileSchemaNoPatterns(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
	</schema>`)
	require.Contains(t, errs, "schema has no pattern element")
}

func TestCompileNonRuleInPattern(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<bogus/>
			<rule context="*"><assert test="true()">ok</assert></rule>
		</pattern>
	</schema>`)
	require.Contains(t, errs, "Expecting a rule element instead of bogus")
}

func TestCompileValidSchema(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="*"><assert test="true()">ok</assert></rule>
		</pattern>
	</schema>`)
	require.Equal(t, "", errs)
}

func TestCompileRuleWithLetOnly(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="*">
				<let name="x" value="1"/>
			</rule>
		</pattern>
	</schema>`)
	require.Contains(t, errs, "rule has no assert nor report element")
}

func TestCompileMultipleErrors(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="">
			</rule>
		</pattern>
	</schema>`)
	require.Contains(t, errs, "rule has an empty context attribute")
	require.Contains(t, errs, "Pattern has no rule element")
}

func TestCompileValueOfNoSelect(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="*"><assert test="true()">val: <value-of/></assert></rule>
		</pattern>
	</schema>`)
	require.Contains(t, errs, "value-of has no select attribute")
}

func TestCompileTitleAfterPattern(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context="*"><assert test="true()">ok</assert></rule></pattern>
		<title>late title</title>
	</schema>`)
	require.Contains(t, errs, "Expecting a pattern element instead of title")
}

func TestCompileNsAfterPattern(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context="*"><assert test="true()">ok</assert></rule></pattern>
		<ns prefix="p" uri="urn:test"/>
	</schema>`)
	require.Contains(t, errs, "Expecting a pattern element instead of ns")
}

func TestCompileTitleBeforeNsBeforePattern(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<title>my schema</title>
		<ns prefix="p" uri="urn:test"/>
		<pattern><rule context="*"><assert test="true()">ok</assert></rule></pattern>
	</schema>`)
	require.Equal(t, "", errs)
}

func TestNameInterpolation(t *testing.T) {
	t.Run("non-namespaced element", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="item">
				<assert test="false()"><name/> is invalid</assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Contains(t, collected[0].Message, "item is invalid")
	})

	t.Run("namespaced element", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<ns prefix="ex" uri="http://example.com"/>
			<pattern><rule context="ex:item">
				<assert test="false()"><name/> is invalid</assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:ex="http://example.com"><ex:item/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Contains(t, collected[0].Message, "ex:item is invalid")
	})
}

func TestValueOfInterpolation(t *testing.T) {
	t.Run("boolean true", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="item">
				<assert test="false()">result is <value-of select="true()"/></assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Contains(t, collected[0].Message, "result is True")
	})

	t.Run("boolean false", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="item">
				<assert test="false()">result is <value-of select="false()"/></assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Contains(t, collected[0].Message, "result is False")
	})

	t.Run("nodeset names", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="root">
				<assert test="false()">children: <value-of select="*"/></assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/><b/><c/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Contains(t, collected[0].Message, "children: a b c")
	})

	t.Run("empty nodeset", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="root">
				<assert test="false()">children: [<value-of select="nonexistent"/>]</assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Contains(t, collected[0].Message, "children: []")
	})
}

func TestContextPatterns(t *testing.T) {
	t.Run("simple element", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="a">
				<assert test="false()">found a</assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/><b/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Len(t, collected, 1)
		require.Equal(t, "a", collected[0].Element)
	})

	t.Run("union context", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="a | b">
				<assert test="false()">found</assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/><b/><c/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Len(t, collected, 2)
		elements := []string{collected[0].Element, collected[1].Element}
		sort.Strings(elements)
		require.Equal(t, []string{"a", "b"}, elements)
	})

	t.Run("absolute path", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="/root/a">
				<assert test="false()">found</assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/><sub><a/></sub></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		// Only the direct child /root/a should match, not /root/sub/a
		require.Len(t, collected, 1)
	})

	t.Run("predicate", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="a[@x]">
				<assert test="false()">found</assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a x="1"/><a/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		// Only <a x="1"> should match
		require.Len(t, collected, 1)
	})

	t.Run("wildcard", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="*">
				<assert test="false()">found <name/></assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/><b/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		// root, a, b — all elements should match
		require.Len(t, collected, 3)
	})
}

func TestLetVariableChainedDependency(t *testing.T) {
	t.Run("independent lets", func(t *testing.T) {
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern>
				<rule context="item">
					<let name="x" value="string(@val)"/>
					<let name="y" value="'hello'"/>
					<assert test="$x = 'ok'">x is <value-of select="$x"/>, y is <value-of select="$y"/></assert>
				</rule>
			</pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item val="bad"/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		got := collectedString(collected)
		require.Contains(t, got, "x is bad")
		require.Contains(t, got, "y is hello")
	})

	t.Run("LIFO evaluation order", func(t *testing.T) {
		// libxml2 stores lets in LIFO order. When a is defined first and b
		// second, the list is [b, a]. b is evaluated first (before a is
		// registered), so $a in b's expression is NaN. We verify the
		// observable effect: a=1 is reported correctly.
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern>
				<rule context="item">
					<let name="a" value="1"/>
					<let name="b" value="$a + 1"/>
					<report test="$a = 1">a=<value-of select="$a"/></report>
				</rule>
			</pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Contains(t, collectedString(collected), "a=1")
	})
}

func TestUnionContextIntegration(t *testing.T) {
	schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="invoice | credit-note">
				<assert test="@id">Missing id attribute</assert>
			</rule>
		</pattern>
	</schema>`)
	require.Equal(t, "", errs)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><invoice/><credit-note/><other/></root>`))
	require.NoError(t, err)

	collected, err := validateAndCollect(t, schema, doc)
	require.ErrorIs(t, err, schematron.ErrValidationFailed)
	got := collectedString(collected)
	require.Contains(t, got, "invoice")
	require.Contains(t, got, "credit-note")
	require.NotContains(t, got, "other")
}

func TestZeroCompilerFluent(t *testing.T) {
	var c schematron.Compiler
	require.NotPanics(t, func() {
		c2 := c.SchemaFilename("test.sch")
		_ = c2
	})
}

func TestZeroValidatorFluent(t *testing.T) {
	var v schematron.Validator
	require.NotPanics(t, func() {
		v2 := v.Filename("test.xml")
		_ = v2
	})
}
