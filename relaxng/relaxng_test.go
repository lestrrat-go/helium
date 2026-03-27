package relaxng_test

import (
	"errors"
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
var skip = map[string]string{}

// skipExact lists specific test cases (by full test name) that need skipping
// when their group-level skip has been removed.
var skipExact = map[string]string{}

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
	t.Parallel()
	filterEnv := os.Getenv("HELIUM_RELAXNG_TEST_FILES")

	cases := discoverTests(t)
	require.NotEmpty(t, cases, "no test cases discovered")

	passed := 0
	skipped := 0
	failed := 0

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			grammar, err := relaxng.NewCompiler().SchemaFilename(rngFilename).ErrorHandler(collector).CompileFile(t.Context(), tc.rngPath)
			require.NoError(t, err, "schema compilation returned error for %s", tc.rngPath)
			_ = collector.Close()
			compileWarnings, compileErrors := partitionCompileErrors(collector.Errors())

			var got string
			if compileErrors != "" {
				got = compileWarnings + compileErrors
				got += "Relax-NG schema " + rngFilename + " failed to compile\n"
			} else {
				// Parse instance.
				xmlData, err := os.ReadFile(tc.xmlPath)
				require.NoError(t, err)
				doc, err := helium.NewParser().Parse(t.Context(), xmlData)
				require.NoError(t, err, "XML parse failed for %s", tc.xmlPath)

				// Validate.
				filename := "./test/relaxng/" + tc.xmlBase
				valCollector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
				err = relaxng.NewValidator(grammar).Filename(filename).ErrorHandler(valCollector).Validate(t.Context(), doc)
				_ = valCollector.Close()
				if errors.Is(err, relaxng.ErrValidationFailed) {
					var valErrs strings.Builder
					for _, ve := range valCollector.Errors() {
						valErrs.WriteString(ve.Error())
					}
					got = compileWarnings + valErrs.String() + filename + " fails to validate\n"
				} else {
					require.NoError(t, err)
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

func TestGetAttrWhitespace(t *testing.T) {
	t.Parallel()
	// Verify that whitespace-padded name attributes are trimmed so that
	// define/ref matching works correctly.
	input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><ref name="  foo  "/></start>
  <define name="  foo  ">
    <element name="a"><empty/></element>
  </define>
</grammar>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	grammar, err := relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	require.Empty(t, compileErrors, "schema with whitespace-padded names should compile without errors")

	xmlData := []byte(`<a/>`)
	xmlDoc, err := helium.NewParser().Parse(t.Context(), xmlData)
	require.NoError(t, err)

	err = relaxng.NewValidator(grammar).Validate(t.Context(), xmlDoc)
	require.NoError(t, err, "valid document should validate")
}

func TestXmlBaseInclude(t *testing.T) {
	t.Parallel()
	// Schema uses <div xml:base="xmlbase/"> wrapping <include href="included.rng"/>.
	// The included file lives in testdata/xmlbase/included.rng.
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	grammar, err := relaxng.NewCompiler().ErrorHandler(collector).CompileFile(t.Context(), "testdata/xmlbase_include.rng")
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	require.Empty(t, compileErrors, "include via xml:base should compile without errors")

	xmlData := []byte(`<sub/>`)
	xmlDoc, err := helium.NewParser().Parse(t.Context(), xmlData)
	require.NoError(t, err)

	err = relaxng.NewValidator(grammar).Validate(t.Context(), xmlDoc)
	require.NoError(t, err, "valid document should validate")
}

func TestXmlBaseExternalRef(t *testing.T) {
	t.Parallel()
	// Schema uses xml:base on the root grammar element to redirect externalRef
	// resolution to the xmlbase/ subdirectory.
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	grammar, err := relaxng.NewCompiler().ErrorHandler(collector).CompileFile(t.Context(), "testdata/xmlbase_extref.rng")
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	require.Empty(t, compileErrors, "externalRef via xml:base should compile without errors")

	xmlData := []byte(`<sub/>`)
	xmlDoc, err := helium.NewParser().Parse(t.Context(), xmlData)
	require.NoError(t, err)

	err = relaxng.NewValidator(grammar).Validate(t.Context(), xmlDoc)
	require.NoError(t, err, "valid document should validate")
}

func TestCheckCombine(t *testing.T) {
	t.Parallel()
	t.Run("conflicting define combine modes", func(t *testing.T) {
		t.Parallel()
		input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><ref name="foo"/></start>
  <define name="foo" combine="choice">
    <element name="a"><empty/></element>
  </define>
  <define name="foo" combine="interleave">
    <element name="b"><empty/></element>
  </define>
</grammar>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		require.Contains(t, compileErrors, "Defines for foo use both 'interleave' and 'choice'")
	})

	t.Run("multiple defines without combine", func(t *testing.T) {
		t.Parallel()
		input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><ref name="foo"/></start>
  <define name="foo">
    <element name="a"><empty/></element>
  </define>
  <define name="foo">
    <element name="b"><empty/></element>
  </define>
</grammar>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		require.Contains(t, compileErrors, "Some defines for foo needs the combine attribute")
	})

	t.Run("conflicting start combine modes", func(t *testing.T) {
		t.Parallel()
		input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start combine="choice">
    <element name="a"><empty/></element>
  </start>
  <start combine="interleave">
    <element name="b"><empty/></element>
  </start>
</grammar>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		require.Contains(t, compileErrors, "<start> use both 'interleave' and 'choice'")
	})

	t.Run("multiple starts without combine", func(t *testing.T) {
		t.Parallel()
		input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="a"><empty/></element>
  </start>
  <start>
    <element name="b"><empty/></element>
  </start>
</grammar>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		require.Contains(t, compileErrors, "Some <start> element miss the combine attribute")
	})

	t.Run("valid combine modes agree", func(t *testing.T) {
		t.Parallel()
		input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><ref name="foo"/></start>
  <define name="foo" combine="choice">
    <element name="a"><empty/></element>
  </define>
  <define name="foo" combine="choice">
    <element name="b"><empty/></element>
  </define>
</grammar>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		require.Empty(t, compileErrors)
	})
}

func TestCheckRules(t *testing.T) {
	t.Parallel()
	const rngNS = "http://relaxng.org/ns/structure/1.0"

	compile := func(t *testing.T, input string) (compileErrors, compileWarnings string) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = relaxng.NewCompiler().SchemaFilename("test.rng").ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		compileWarnings, compileErrors = partitionCompileErrors(collector.Errors())
		return compileErrors, compileWarnings
	}

	t.Run("list//element", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <list><element name="a"><empty/></element></list>
</element>`)
		require.Contains(t, errs, "Found forbidden pattern list//element")
	})

	t.Run("attribute//element", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <attribute name="a"><element name="b"><empty/></element></attribute>
</element>`)
		require.Contains(t, errs, "Found forbidden pattern attribute//element")
	})

	t.Run("list//list", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <list><list><data type="string" datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"/></list></list>
</element>`)
		require.Contains(t, errs, "Found forbidden pattern list//list")
	})

	t.Run("attribute//attribute", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <attribute name="a"><attribute name="b"/></attribute>
</element>`)
		require.Contains(t, errs, "Found forbidden pattern attribute//attribute")
	})

	t.Run("start//attribute", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<grammar xmlns="`+rngNS+`">
  <start><attribute name="a"/></start>
</grammar>`)
		require.Contains(t, errs, "Found forbidden pattern start//attribute")
	})

	t.Run("oneOrMore//group//attribute", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <oneOrMore><group><attribute name="a"/></group></oneOrMore>
</element>`)
		require.Contains(t, errs, "Found forbidden pattern oneOrMore//group//attribute")
	})

	t.Run("oneOrMore//interleave//attribute", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <oneOrMore><interleave><attribute name="a"/><text/></interleave></oneOrMore>
</element>`)
		require.Contains(t, errs, "Found forbidden pattern oneOrMore//interleave//attribute")
	})

	t.Run("anyName attribute without oneOrMore warning", func(t *testing.T) {
		t.Parallel()
		errs, warnings := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <attribute><anyName/></attribute>
</element>`)
		require.Empty(t, errs)
		require.Contains(t, warnings, "Found anyName attribute without oneOrMore ancestor")
	})

	t.Run("nsName attribute without oneOrMore warning", func(t *testing.T) {
		t.Parallel()
		errs, warnings := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <attribute><nsName ns="http://example.com"/></attribute>
</element>`)
		require.Empty(t, errs)
		require.Contains(t, warnings, "Found nsName attribute without oneOrMore ancestor")
	})

	t.Run("anyName attribute with oneOrMore no warning", func(t *testing.T) {
		t.Parallel()
		errs, warnings := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <oneOrMore><attribute><anyName/></attribute></oneOrMore>
</element>`)
		require.Empty(t, errs)
		require.Empty(t, warnings)
	})

	t.Run("element resets context", func(t *testing.T) {
		t.Parallel()
		// attribute inside element inside list: element resets context,
		// so the attribute should not trigger list//attribute.
		errs, _ := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <list>
    <element name="inner">
      <attribute name="a"/>
    </element>
  </list>
</element>`)
		// list//element error expected (element inside list is still forbidden),
		// but no list//attribute error.
		require.Contains(t, errs, "Found forbidden pattern list//element")
		require.NotContains(t, errs, "list//attribute")
	})

	t.Run("valid schema", func(t *testing.T) {
		t.Parallel()
		errs, warnings := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <oneOrMore>
    <element name="item">
      <attribute name="id"/>
      <text/>
    </element>
  </oneOrMore>
</element>`)
		require.Empty(t, errs)
		require.Empty(t, warnings)
	})

	t.Run("data/except//element", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<element name="root" xmlns="`+rngNS+`"
  datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="string">
    <except><element name="a"><empty/></element></except>
  </data>
</element>`)
		require.Contains(t, errs, "Found forbidden pattern data/except//element")
	})

	t.Run("list//text", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <list><text/></list>
</element>`)
		require.Contains(t, errs, "Found forbidden pattern list//text")
	})

	t.Run("list//interleave", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <list>
    <interleave>
      <data type="string" datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"/>
      <data type="integer" datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"/>
    </interleave>
  </list>
</element>`)
		require.Contains(t, errs, "Found forbidden pattern list//interleave")
	})

	t.Run("ref in data/except", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<grammar xmlns="`+rngNS+`"
  datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <start>
    <element name="root">
      <data type="string">
        <except><ref name="foo"/></except>
      </data>
    </element>
  </start>
  <define name="foo"><value>x</value></define>
</grammar>`)
		require.Contains(t, errs, "Found forbidden pattern data/except//ref")
	})
}

func TestZeroCompilerFluent(t *testing.T) {
	t.Parallel()
	var c relaxng.Compiler
	require.NotPanics(t, func() {
		c2 := c.SchemaFilename("test.rng")
		_ = c2
	})
}

func TestZeroValidatorFluent(t *testing.T) {
	t.Parallel()
	var v relaxng.Validator
	require.NotPanics(t, func() {
		v2 := v.Filename("test.xml")
		_ = v2
	})
}
