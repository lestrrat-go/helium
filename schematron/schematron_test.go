package schematron_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
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
	t.Parallel()
	filterEnv := os.Getenv("HELIUM_SCHEMATRON_TEST_FILES")

	cases := discoverTests(t)
	require.NotEmpty(t, cases, "no test cases discovered")

	var passed, skipped, failed atomic.Int64

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if filterEnv != "" && !strings.Contains(tc.name, filterEnv) {
				t.Skip("filtered out by HELIUM_SCHEMATRON_TEST_FILES")
				skipped.Add(1)
				return
			}

			if reason, ok := skipExact[tc.name]; ok {
				t.Skipf("skipping: %s", reason)
				skipped.Add(1)
				return
			}
			baseName := extractBaseName(tc.name)
			if reason := shouldSkip(baseName); reason != "" {
				t.Skipf("skipping: %s", reason)
				skipped.Add(1)
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
				err = schematron.NewValidator(schema).Label(filename).ErrorHandler(valCollector).Validate(t.Context(), doc)
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
				passed.Add(1)
			} else {
				failed.Add(1)
				require.Equal(t, string(expected), got)
			}
		})
	}

	t.Logf("Results: %d passed, %d failed, %d skipped (out of %d total)", passed.Load(), failed.Load(), skipped.Load(), len(cases))
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
	t.Parallel()
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
		t.Parallel()
		doc, err := p.Parse(t.Context(), []byte(`<AAA><CCC/></AAA>`))
		require.NoError(t, err)

		// Without quiet: sentinel error returned, handler receives errors
		var collected []*schematron.ValidationError
		handler := validationErrorCollector{errors: &collected}
		err = schematron.NewValidator(schema).Label("test.xml").ErrorHandler(handler).Validate(t.Context(), doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.NotEmpty(t, collected)

		// With quiet + handler: handler still receives errors
		var quietCollected []*schematron.ValidationError
		quietHandler := validationErrorCollector{errors: &quietCollected}
		quietErr := schematron.NewValidator(schema).Label("test.xml").Quiet().ErrorHandler(quietHandler).Validate(t.Context(), doc)
		require.ErrorIs(t, quietErr, schematron.ErrValidationFailed)
		require.Empty(t, quietCollected)
	})

	t.Run("passing document", func(t *testing.T) {
		t.Parallel()
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
		err = schematron.NewValidator(rSchema).Label("test.xml").ErrorHandler(handler).Validate(t.Context(), doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.NotEmpty(t, collected)
		require.Contains(t, collected[0].Message, "CCC is present")

		// With quiet: report suppressed (no errors delivered to handler)
		var quietCollected []*schematron.ValidationError
		quietHandler := validationErrorCollector{errors: &quietCollected}
		quietErr := schematron.NewValidator(rSchema).Label("test.xml").Quiet().ErrorHandler(quietHandler).Validate(t.Context(), doc)
		require.ErrorIs(t, quietErr, schematron.ErrValidationFailed)
		require.Empty(t, quietCollected)
	})
}

func TestWithErrorHandler(t *testing.T) {
	t.Parallel()
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
		t.Parallel()
		var collected []*schematron.ValidationError
		handler := validationErrorCollector{errors: &collected}

		doc, err := p.Parse(t.Context(), []byte(`<AAA><CCC/></AAA>`))
		require.NoError(t, err)

		err = schematron.NewValidator(schema).
			Label("test.xml").
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
		t.Parallel()
		var collected []*schematron.ValidationError
		handler := validationErrorCollector{errors: &collected}

		doc, err := p.Parse(t.Context(), []byte(`<AAA name="x"><BBB/></AAA>`))
		require.NoError(t, err)

		err = schematron.NewValidator(schema).
			Label("test.xml").
			ErrorHandler(handler).
			Validate(t.Context(), doc)

		require.NoError(t, err)
		require.Empty(t, collected)
	})

	t.Run("quiet suppresses handler delivery", func(t *testing.T) {
		t.Parallel()
		var collected []*schematron.ValidationError
		handler := validationErrorCollector{errors: &collected}

		doc, err := p.Parse(t.Context(), []byte(`<AAA><CCC/></AAA>`))
		require.NoError(t, err)

		err = schematron.NewValidator(schema).
			Label("test.xml").
			Quiet().
			ErrorHandler(handler).
			Validate(t.Context(), doc)

		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Empty(t, collected)
	})
}

// TestErroringTestXPath ensures that an assertion/report test whose XPath
// cannot be evaluated is not silently treated as satisfied. A broken test
// must surface the XPath error and, for an assert, fail validation.
func TestErroringTestXPath(t *testing.T) {
	t.Parallel()
	p := helium.NewParser()

	t.Run("assert with erroring test fails and reports error", func(t *testing.T) {
		t.Parallel()
		const sct = `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
  <pattern>
    <rule context="/root">
      <assert test="not-a-function()">bad assert</assert>
    </rule>
  </pattern>
</schema>`
		sDoc, err := p.Parse(t.Context(), []byte(sct))
		require.NoError(t, err)
		schema, err := schematron.NewCompiler().Compile(t.Context(), sDoc)
		require.NoError(t, err)

		doc, err := p.Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		var msgs []string
		handler := messageCollector{msgs: &msgs}
		verr := schematron.NewValidator(schema).Label("test.xml").ErrorHandler(handler).Validate(t.Context(), doc)

		require.ErrorIs(t, verr, schematron.ErrValidationFailed)
		require.True(t, hasXPathError(msgs), "expected an XPath error to be reported, got %v", msgs)
	})

	t.Run("report with erroring test reports error without spurious pass", func(t *testing.T) {
		t.Parallel()
		const sct = `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
  <pattern>
    <rule context="/root">
      <report test="not-a-function()">bad report</report>
    </rule>
  </pattern>
</schema>`
		sDoc, err := p.Parse(t.Context(), []byte(sct))
		require.NoError(t, err)
		schema, err := schematron.NewCompiler().Compile(t.Context(), sDoc)
		require.NoError(t, err)

		doc, err := p.Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		var msgs []string
		handler := messageCollector{msgs: &msgs}
		// A report whose test errors is treated as false (does not fire),
		// so validation is not marked failed, but the XPath error must
		// still be surfaced rather than swallowed.
		verr := schematron.NewValidator(schema).Label("test.xml").ErrorHandler(handler).Validate(t.Context(), doc)

		require.NoError(t, verr)
		require.True(t, hasXPathError(msgs), "expected an XPath error to be reported, got %v", msgs)
	})

	t.Run("erroring rule context fails and reports error", func(t *testing.T) {
		t.Parallel()
		// The context expression compiles but errors at evaluation
		// (unknown function inside a predicate). libxml2 wraps rule
		// contexts as //<expr>, so a bare not-a-function() would be
		// rejected at compile time; using a predicate reaches the
		// runtime evaluation-error path under test.
		const sct = `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
  <pattern>
    <rule context="*[unknownfn()]">
      <assert test="true()">ok</assert>
    </rule>
  </pattern>
</schema>`
		sDoc, err := p.Parse(t.Context(), []byte(sct))
		require.NoError(t, err)
		schema, err := schematron.NewCompiler().Compile(t.Context(), sDoc)
		require.NoError(t, err)

		doc, err := p.Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		var msgs []string
		handler := messageCollector{msgs: &msgs}
		verr := schematron.NewValidator(schema).Label("test.xml").ErrorHandler(handler).Validate(t.Context(), doc)

		require.ErrorIs(t, verr, schematron.ErrValidationFailed)
		require.True(t, hasXPathError(msgs), "expected an XPath error to be reported, got %v", msgs)
	})

	t.Run("valid schema still passes and fires genuine assertion", func(t *testing.T) {
		t.Parallel()
		const sct = `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
  <pattern>
    <rule context="/root">
      <assert test="@id">missing id</assert>
    </rule>
  </pattern>
</schema>`
		sDoc, err := p.Parse(t.Context(), []byte(sct))
		require.NoError(t, err)
		schema, err := schematron.NewCompiler().Compile(t.Context(), sDoc)
		require.NoError(t, err)

		okDoc, err := p.Parse(t.Context(), []byte(`<root id="x"/>`))
		require.NoError(t, err)
		var okMsgs []string
		require.NoError(t, schematron.NewValidator(schema).ErrorHandler(messageCollector{msgs: &okMsgs}).Validate(t.Context(), okDoc))
		require.False(t, hasXPathError(okMsgs), "valid run should report no XPath error, got %v", okMsgs)

		badDoc, err := p.Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)
		var badMsgs []string
		verr := schematron.NewValidator(schema).ErrorHandler(messageCollector{msgs: &badMsgs}).Validate(t.Context(), badDoc)
		require.ErrorIs(t, verr, schematron.ErrValidationFailed)
		require.False(t, hasXPathError(badMsgs), "genuine assertion failure should not be an XPath error, got %v", badMsgs)
	})
}

// messageCollector implements helium.ErrorHandler and records every error
// message string delivered to it (including non-ValidationError errors such
// as surfaced XPath errors).
type messageCollector struct {
	msgs *[]string
}

func (c messageCollector) Handle(_ context.Context, err error) {
	*c.msgs = append(*c.msgs, err.Error())
}

// hasXPathError reports whether any collected message is a surfaced XPath
// evaluation error.
func hasXPathError(msgs []string) bool {
	for _, m := range msgs {
		if strings.Contains(m, "XPath error :") {
			return true
		}
	}
	return false
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
// any compile error strings concatenated. A fatal compilation failure
// (ErrCompileFailed) is tolerated so callers can assert on the collected
// diagnostic strings; on such a failure the returned schema is nil.
func compileTestSchema(t *testing.T, xml string) (*schematron.Schema, string) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, err := schematron.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
	if err != nil {
		require.ErrorIs(t, err, schematron.ErrCompileFailed)
	}
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
	t.Parallel()
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context=""><assert test="true()">ok</assert></rule></pattern>
	</schema>`)
	require.Contains(t, errs, "rule has an empty context attribute")
}

func TestCompileRuleNoAssert(t *testing.T) {
	t.Parallel()
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context="*"></rule></pattern>
	</schema>`)
	require.Contains(t, errs, "rule has no assert nor report element")
}

func TestCompilePatternNoRules(t *testing.T) {
	t.Parallel()
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern></pattern>
	</schema>`)
	require.Contains(t, errs, "Pattern has no rule element")
}

func TestCompileSchemaNoPatterns(t *testing.T) {
	t.Parallel()
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
	</schema>`)
	require.Contains(t, errs, "schema has no pattern element")
}

func TestCompileNonRuleInPattern(t *testing.T) {
	t.Parallel()
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<bogus/>
			<rule context="*"><assert test="true()">ok</assert></rule>
		</pattern>
	</schema>`)
	require.Contains(t, errs, "Expecting a rule element instead of bogus")
}

func TestCompileInvalidValueOfSelect(t *testing.T) {
	t.Parallel()
	// An invalid <value-of select="..."> XPath must be reported through the
	// error handler, mirroring the <name path="..."> case, instead of being
	// silently swallowed (which leaves a truncated assertion message).
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="/root">
				<assert test="false()">val is <value-of select="("/></assert>
			</rule>
		</pattern>
	</schema>`)
	require.Contains(t, errs, "Failed to compile select expression '('")
}

func TestCompileValidSchema(t *testing.T) {
	t.Parallel()
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="*"><assert test="true()">ok</assert></rule>
		</pattern>
	</schema>`)
	require.Equal(t, "", errs)
}

func TestCompileRuleWithLetOnly(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="*"><assert test="true()">val: <value-of/></assert></rule>
		</pattern>
	</schema>`)
	require.Contains(t, errs, "value-of has no select attribute")
}

func TestCompileTitleAfterPattern(t *testing.T) {
	t.Parallel()
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context="*"><assert test="true()">ok</assert></rule></pattern>
		<title>late title</title>
	</schema>`)
	require.Contains(t, errs, "Expecting a pattern element instead of title")
}

func TestCompileNsAfterPattern(t *testing.T) {
	t.Parallel()
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context="*"><assert test="true()">ok</assert></rule></pattern>
		<ns prefix="p" uri="urn:test"/>
	</schema>`)
	require.Contains(t, errs, "Expecting a pattern element instead of ns")
}

func TestCompileTitleBeforeNsBeforePattern(t *testing.T) {
	t.Parallel()
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<title>my schema</title>
		<ns prefix="p" uri="urn:test"/>
		<pattern><rule context="*"><assert test="true()">ok</assert></rule></pattern>
	</schema>`)
	require.Equal(t, "", errs)
}

func TestNameInterpolation(t *testing.T) {
	t.Parallel()
	t.Run("non-namespaced element", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
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
	t.Parallel()
	t.Run("boolean true", func(t *testing.T) {
		t.Parallel()
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
		// XPath 1.0 string(boolean): true() converts to "true" (lowercase).
		require.Contains(t, collected[0].Message, "result is true")
	})

	t.Run("boolean false", func(t *testing.T) {
		t.Parallel()
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
		// XPath 1.0 string(boolean): false() converts to "false" (lowercase).
		require.Contains(t, collected[0].Message, "result is false")
	})

	t.Run("number formatting", func(t *testing.T) {
		t.Parallel()
		// XPath 1.0 string(number) lexical forms (xmlXPathFormatNumber):
		// integers carry no decimal point or exponent, special values use
		// the "NaN"/"Infinity"/"-Infinity" spellings, and negative zero
		// renders as "0". Go's default %g formatting diverges on all of these.
		for _, tc := range []struct {
			name   string
			expr   string
			expect string
		}{
			{"large integer", "1234567", "n=1234567"},
			{"decimal", "3 div 2", "n=1.5"},
			{"NaN", "0 div 0", "n=NaN"},
			{"positive infinity", "1 div 0", "n=Infinity"},
			{"negative infinity", "-1 div 0", "n=-Infinity"},
			{"negative zero", "-1 * 0", "n=0"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
					<pattern><rule context="item">
						<assert test="false()">n=<value-of select="`+tc.expr+`"/></assert>
					</rule></pattern>
				</schema>`)
				require.Equal(t, "", errs)

				doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
				require.NoError(t, err)

				collected, err := validateAndCollect(t, schema, doc)
				require.ErrorIs(t, err, schematron.ErrValidationFailed)
				require.Contains(t, collected[0].Message, tc.expect)
			})
		}
	})

	t.Run("nodeset string-value", func(t *testing.T) {
		t.Parallel()
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="root">
				<assert test="false()">children: <value-of select="*"/></assert>
			</rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a>X</a><b>Y</b><c>Z</c></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		// XPath 1.0: a node-set converts to the string-value of the node
		// first in document order, i.e. the text of <a>, not the names.
		require.Contains(t, collected[0].Message, "children: X")
	})

	t.Run("empty nodeset", func(t *testing.T) {
		t.Parallel()
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
	t.Parallel()
	t.Run("simple element", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
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
	t.Parallel()
	t.Run("independent lets", func(t *testing.T) {
		t.Parallel()
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

	t.Run("document order evaluation", func(t *testing.T) {
		t.Parallel()
		// Lets are evaluated in document order, so a later let may depend on
		// an earlier one: a is defined first, then b references $a. We assert
		// the dependent value b=2 (a=1 plus 1). Under the old reversed (LIFO)
		// order b would be evaluated before a was bound, making $a NaN and
		// b NaN — so this asserting b's resolved value would fail.
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern>
				<rule context="item">
					<let name="a" value="1"/>
					<let name="b" value="$a + 1"/>
					<report test="$b = 2">b=<value-of select="$b"/></report>
				</rule>
			</pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Contains(t, collectedString(collected), "b=2")
	})
}

func TestUnionContextIntegration(t *testing.T) {
	t.Parallel()
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

func TestAttributeContext(t *testing.T) {
	t.Parallel()
	// A rule whose context selects an attribute (context="@id" becomes
	// //@id) must run its asserts against each attribute node. The assert
	// fails for short id values. Before the fix, attribute context nodes
	// were silently skipped and the document validated.
	schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="@id">
				<assert test="string-length(.) &gt; 3">id too short</assert>
			</rule>
		</pattern>
	</schema>`)
	require.Equal(t, "", errs)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root id="a"><child id="b"/></root>`))
	require.NoError(t, err)

	collected, err := validateAndCollect(t, schema, doc)
	require.ErrorIs(t, err, schematron.ErrValidationFailed)
	require.NotEmpty(t, collected)

	var paths []string
	for _, ve := range collected {
		paths = append(paths, ve.Path)
	}
	require.Contains(t, paths, "/root/@id")
	require.Contains(t, paths, "/root/child/@id")
}

func TestZeroCompilerFluent(t *testing.T) {
	t.Parallel()
	var c schematron.Compiler
	require.NotPanics(t, func() {
		c2 := c.Label("test.sch")
		_ = c2
	})
}

func TestZeroValidatorFluent(t *testing.T) {
	t.Parallel()
	var v schematron.Validator
	require.NotPanics(t, func() {
		v2 := v.Label("test.xml")
		_ = v2
	})
}

// TestFirstMatchOnlyRule verifies ISO Schematron semantics: within a pattern,
// each node is processed by only the FIRST rule whose context matches it.
// A second rule in the same pattern that also matches the node must not fire.
func TestFirstMatchOnlyRule(t *testing.T) {
	t.Parallel()

	t.Run("later rule skipped within pattern", func(t *testing.T) {
		t.Parallel()
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern>
				<rule context="item"><report test="true()">first rule</report></rule>
				<rule context="item"><report test="true()">second rule</report></rule>
			</pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Len(t, collected, 1)
		require.Equal(t, "first rule", collected[0].Message)
	})

	t.Run("broader later rule still skips matched node", func(t *testing.T) {
		t.Parallel()
		// First rule claims <item>; the second rule (context="*") matches
		// every element but must skip the already-claimed <item>.
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern>
				<rule context="item"><report test="true()">specific</report></rule>
				<rule context="*"><report test="true()">wildcard <name/></report></rule>
			</pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)

		got := collectedString(collected)
		require.Contains(t, got, "specific")
		require.Contains(t, got, "wildcard root")
		// The wildcard rule must NOT fire for <item>: it was claimed first.
		require.NotContains(t, got, "wildcard item")
	})

	t.Run("separate patterns are independent", func(t *testing.T) {
		t.Parallel()
		// First-match-only is scoped per pattern: a matching rule in a second
		// pattern still fires for the same node.
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern><rule context="item"><report test="true()">p1</report></rule></pattern>
			<pattern><rule context="item"><report test="true()">p2</report></rule></pattern>
		</schema>`)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Len(t, collected, 2)
		got := collectedString(collected)
		require.Contains(t, got, "p1")
		require.Contains(t, got, "p2")
	})
}

// TestForeignNamespaceRejected verifies that foreign-namespaced markup under a
// Schematron <schema> is not executed as Schematron: neither foreign elements
// (e.g. <x:rule>) nor foreign-namespaced structural attributes (e.g. x:test).
func TestForeignNamespaceRejected(t *testing.T) {
	t.Parallel()

	t.Run("foreign rule element rejected", func(t *testing.T) {
		t.Parallel()
		// The only "rule" is foreign (x:rule); the pattern therefore has no
		// Schematron rule and compilation fails.
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron" xmlns:x="urn:foreign">
			<pattern>
				<x:rule context="item"><x:assert test="false()">foreign fired</x:assert></x:rule>
			</pattern>
		</schema>`)
		require.Nil(t, schema)
		require.Contains(t, errs, "Pattern has no rule element")
	})

	t.Run("foreign assert/report ignored inside rule", func(t *testing.T) {
		t.Parallel()
		// The Schematron rule has only a foreign assert, so it has no
		// Schematron test and compilation fails.
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron" xmlns:x="urn:foreign">
			<pattern>
				<rule context="item"><x:assert test="false()">foreign</x:assert></rule>
			</pattern>
		</schema>`)
		require.Nil(t, schema)
		require.Contains(t, errs, "rule has no assert nor report element")
	})

	t.Run("foreign-namespaced structural attribute not read", func(t *testing.T) {
		t.Parallel()
		// The assert carries only x:test (foreign); the unqualified test
		// attribute is absent, so the assert is dropped, leaving the rule
		// with no test.
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron" xmlns:x="urn:foreign">
			<pattern>
				<rule context="item"><assert x:test="false()">foreign attr</assert></rule>
			</pattern>
		</schema>`)
		require.Nil(t, schema)
		require.Contains(t, errs, "rule has no assert nor report element")
	})

	t.Run("unqualified test still honored alongside foreign attr", func(t *testing.T) {
		t.Parallel()
		// A valid unqualified test plus a stray foreign attribute must compile
		// and use only the unqualified test.
		schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron" xmlns:x="urn:foreign">
			<pattern>
				<rule context="item"><assert test="false()" x:test="true()">real</assert></rule>
			</pattern>
		</schema>`)
		require.Equal(t, "", errs)
		require.NotNil(t, schema)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
		require.NoError(t, err)

		collected, err := validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Len(t, collected, 1)
		require.Equal(t, "real", collected[0].Message)
	})
}

// TestCompileFailsOnBrokenSchema verifies that a fatal compile error makes
// Compile return ErrCompileFailed and a nil schema, even when no error handler
// is configured (so the error cannot be silently lost).
func TestCompileFailsOnBrokenSchema(t *testing.T) {
	t.Parallel()

	broken := []struct {
		name string
		sct  string
	}{
		{"no pattern", `<schema xmlns="http://purl.oclc.org/dsdl/schematron"></schema>`},
		{"pattern without rule", `<schema xmlns="http://purl.oclc.org/dsdl/schematron"><pattern/></schema>`},
		{"rule without test", `<schema xmlns="http://purl.oclc.org/dsdl/schematron"><pattern><rule context="*"/></pattern></schema>`},
	}

	for _, tc := range broken {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.sct))
			require.NoError(t, err)

			// No error handler configured: the failure must surface via the
			// returned error rather than being discarded.
			schema, err := schematron.NewCompiler().Compile(t.Context(), doc)
			require.ErrorIs(t, err, schematron.ErrCompileFailed)
			require.Nil(t, schema)
		})
	}

	t.Run("valid schema still compiles", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<schema xmlns="http://purl.oclc.org/dsdl/schematron"><pattern><rule context="*"><assert test="true()">ok</assert></rule></pattern></schema>`))
		require.NoError(t, err)
		schema, err := schematron.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)
		require.NotNil(t, schema)
	})
}

// TestValidateNilSchema verifies that validating with a nil/zero-value schema
// returns a typed error instead of panicking.
func TestValidateNilSchema(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	t.Run("NewValidator(nil)", func(t *testing.T) {
		t.Parallel()
		var verr error
		require.NotPanics(t, func() {
			verr = schematron.NewValidator(nil).Validate(t.Context(), doc)
		})
		require.ErrorIs(t, verr, schematron.ErrNoSchema)
	})

	t.Run("zero-value Validator", func(t *testing.T) {
		t.Parallel()
		var v schematron.Validator
		var verr error
		require.NotPanics(t, func() {
			verr = v.Validate(t.Context(), doc)
		})
		require.ErrorIs(t, verr, schematron.ErrNoSchema)
	})

	t.Run("empty Schema with no patterns", func(t *testing.T) {
		t.Parallel()
		var verr error
		require.NotPanics(t, func() {
			verr = schematron.NewValidator(&schematron.Schema{}).Validate(t.Context(), doc)
		})
		require.ErrorIs(t, verr, schematron.ErrNoSchema)
	})
}

// TestCompilerParserInjection verifies that a parser injected via
// Compiler.Parser governs the internal parse of the schema document: a parser
// configured with a tiny MaxDepth rejects a deeply nested schema, while the
// same schema compiles when no parser policy is injected.
func TestCompilerParserInjection(t *testing.T) {
	t.Parallel()

	// schema(1) > pattern(2) > rule(3) > assert(4)
	const schemaSrc = `<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern name="test">
    <rule context="AAA">
      <assert test="BBB">BBB element is missing.</assert>
    </rule>
  </pattern>
</schema>`

	dir := t.TempDir()
	path := filepath.Join(dir, "schema.sch")
	require.NoError(t, os.WriteFile(path, []byte(schemaSrc), 0o600))

	t.Run("injected parser policy enforced", func(t *testing.T) {
		t.Parallel()
		_, err := schematron.NewCompiler().
			Parser(helium.NewParser().MaxDepth(2)).
			CompileFile(t.Context(), path)
		require.Error(t, err, "schema nested deeper than the injected MaxDepth must fail to parse")
	})

	t.Run("control without injection", func(t *testing.T) {
		t.Parallel()
		schema, err := schematron.NewCompiler().CompileFile(t.Context(), path)
		require.NoError(t, err)
		require.NotNil(t, schema)
	})
}
