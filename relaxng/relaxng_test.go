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
				err = relaxng.Validate(doc, grammar, relaxng.WithFilename(filename))
				if err != nil {
					got = grammar.CompileWarnings() + err.Error()
				} else {
					got = grammar.CompileWarnings() + filename + " validates\n"
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
	// Verify that whitespace-padded name attributes are trimmed so that
	// define/ref matching works correctly.
	input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><ref name="  foo  "/></start>
  <define name="  foo  ">
    <element name="a"><empty/></element>
  </define>
</grammar>`
	doc, err := helium.Parse([]byte(input))
	require.NoError(t, err)

	grammar, err := relaxng.Compile(doc)
	require.NoError(t, err)
	require.Empty(t, grammar.CompileErrors(), "schema with whitespace-padded names should compile without errors")

	xmlData := []byte(`<a/>`)
	xmlDoc, err := helium.Parse(xmlData)
	require.NoError(t, err)

	err = relaxng.Validate(xmlDoc, grammar)
	require.NoError(t, err, "valid document should validate")
}

func TestXmlBaseInclude(t *testing.T) {
	// Schema uses <div xml:base="xmlbase/"> wrapping <include href="included.rng"/>.
	// The included file lives in testdata/xmlbase/included.rng.
	grammar, err := relaxng.CompileFile("testdata/xmlbase_include.rng")
	require.NoError(t, err)
	require.Empty(t, grammar.CompileErrors(), "include via xml:base should compile without errors")

	xmlData := []byte(`<sub/>`)
	xmlDoc, err := helium.Parse(xmlData)
	require.NoError(t, err)

	err = relaxng.Validate(xmlDoc, grammar)
	require.NoError(t, err, "valid document should validate")
}

func TestXmlBaseExternalRef(t *testing.T) {
	// Schema uses xml:base on the root grammar element to redirect externalRef
	// resolution to the xmlbase/ subdirectory.
	grammar, err := relaxng.CompileFile("testdata/xmlbase_extref.rng")
	require.NoError(t, err)
	require.Empty(t, grammar.CompileErrors(), "externalRef via xml:base should compile without errors")

	xmlData := []byte(`<sub/>`)
	xmlDoc, err := helium.Parse(xmlData)
	require.NoError(t, err)

	err = relaxng.Validate(xmlDoc, grammar)
	require.NoError(t, err, "valid document should validate")
}

func TestCheckCombine(t *testing.T) {
	t.Run("conflicting define combine modes", func(t *testing.T) {
		input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><ref name="foo"/></start>
  <define name="foo" combine="choice">
    <element name="a"><empty/></element>
  </define>
  <define name="foo" combine="interleave">
    <element name="b"><empty/></element>
  </define>
</grammar>`
		doc, err := helium.Parse([]byte(input))
		require.NoError(t, err)
		grammar, err := relaxng.Compile(doc)
		require.NoError(t, err)
		require.Contains(t, grammar.CompileErrors(), "Defines for foo use both 'interleave' and 'choice'")
	})

	t.Run("multiple defines without combine", func(t *testing.T) {
		input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><ref name="foo"/></start>
  <define name="foo">
    <element name="a"><empty/></element>
  </define>
  <define name="foo">
    <element name="b"><empty/></element>
  </define>
</grammar>`
		doc, err := helium.Parse([]byte(input))
		require.NoError(t, err)
		grammar, err := relaxng.Compile(doc)
		require.NoError(t, err)
		require.Contains(t, grammar.CompileErrors(), "Some defines for foo needs the combine attribute")
	})

	t.Run("conflicting start combine modes", func(t *testing.T) {
		input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start combine="choice">
    <element name="a"><empty/></element>
  </start>
  <start combine="interleave">
    <element name="b"><empty/></element>
  </start>
</grammar>`
		doc, err := helium.Parse([]byte(input))
		require.NoError(t, err)
		grammar, err := relaxng.Compile(doc)
		require.NoError(t, err)
		require.Contains(t, grammar.CompileErrors(), "<start> use both 'interleave' and 'choice'")
	})

	t.Run("multiple starts without combine", func(t *testing.T) {
		input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="a"><empty/></element>
  </start>
  <start>
    <element name="b"><empty/></element>
  </start>
</grammar>`
		doc, err := helium.Parse([]byte(input))
		require.NoError(t, err)
		grammar, err := relaxng.Compile(doc)
		require.NoError(t, err)
		require.Contains(t, grammar.CompileErrors(), "Some <start> element miss the combine attribute")
	})

	t.Run("valid combine modes agree", func(t *testing.T) {
		input := `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><ref name="foo"/></start>
  <define name="foo" combine="choice">
    <element name="a"><empty/></element>
  </define>
  <define name="foo" combine="choice">
    <element name="b"><empty/></element>
  </define>
</grammar>`
		doc, err := helium.Parse([]byte(input))
		require.NoError(t, err)
		grammar, err := relaxng.Compile(doc)
		require.NoError(t, err)
		require.Empty(t, grammar.CompileErrors())
	})
}

func TestCheckRules(t *testing.T) {
	const rngNS = "http://relaxng.org/ns/structure/1.0"

	compile := func(t *testing.T, input string) *relaxng.Grammar {
		t.Helper()
		doc, err := helium.Parse([]byte(input))
		require.NoError(t, err)
		grammar, err := relaxng.Compile(doc, relaxng.WithSchemaFilename("test.rng"))
		require.NoError(t, err)
		return grammar
	}

	t.Run("list//element", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <list><element name="a"><empty/></element></list>
</element>`)
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern list//element")
	})

	t.Run("attribute//element", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <attribute name="a"><element name="b"><empty/></element></attribute>
</element>`)
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern attribute//element")
	})

	t.Run("list//list", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <list><list><data type="string" datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"/></list></list>
</element>`)
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern list//list")
	})

	t.Run("attribute//attribute", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <attribute name="a"><attribute name="b"/></attribute>
</element>`)
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern attribute//attribute")
	})

	t.Run("start//attribute", func(t *testing.T) {
		g := compile(t, `<grammar xmlns="`+rngNS+`">
  <start><attribute name="a"/></start>
</grammar>`)
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern start//attribute")
	})

	t.Run("oneOrMore//group//attribute", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <oneOrMore><group><attribute name="a"/></group></oneOrMore>
</element>`)
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern oneOrMore//group//attribute")
	})

	t.Run("oneOrMore//interleave//attribute", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <oneOrMore><interleave><attribute name="a"/><text/></interleave></oneOrMore>
</element>`)
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern oneOrMore//interleave//attribute")
	})

	t.Run("anyName attribute without oneOrMore warning", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <attribute><anyName/></attribute>
</element>`)
		require.Empty(t, g.CompileErrors())
		require.Contains(t, g.CompileWarnings(), "Found anyName attribute without oneOrMore ancestor")
	})

	t.Run("nsName attribute without oneOrMore warning", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <attribute><nsName ns="http://example.com"/></attribute>
</element>`)
		require.Empty(t, g.CompileErrors())
		require.Contains(t, g.CompileWarnings(), "Found nsName attribute without oneOrMore ancestor")
	})

	t.Run("anyName attribute with oneOrMore no warning", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <oneOrMore><attribute><anyName/></attribute></oneOrMore>
</element>`)
		require.Empty(t, g.CompileErrors())
		require.Empty(t, g.CompileWarnings())
	})

	t.Run("element resets context", func(t *testing.T) {
		// attribute inside element inside list: element resets context,
		// so the attribute should not trigger list//attribute.
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <list>
    <element name="inner">
      <attribute name="a"/>
    </element>
  </list>
</element>`)
		// list//element error expected (element inside list is still forbidden),
		// but no list//attribute error.
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern list//element")
		require.NotContains(t, g.CompileErrors(), "list//attribute")
	})

	t.Run("valid schema", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <oneOrMore>
    <element name="item">
      <attribute name="id"/>
      <text/>
    </element>
  </oneOrMore>
</element>`)
		require.Empty(t, g.CompileErrors())
		require.Empty(t, g.CompileWarnings())
	})

	t.Run("data/except//element", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`"
  datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="string">
    <except><element name="a"><empty/></element></except>
  </data>
</element>`)
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern data/except//element")
	})

	t.Run("list//text", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <list><text/></list>
</element>`)
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern list//text")
	})

	t.Run("list//interleave", func(t *testing.T) {
		g := compile(t, `<element name="root" xmlns="`+rngNS+`">
  <list>
    <interleave>
      <data type="string" datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"/>
      <data type="integer" datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"/>
    </interleave>
  </list>
</element>`)
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern list//interleave")
	})

	t.Run("ref in data/except", func(t *testing.T) {
		g := compile(t, `<grammar xmlns="`+rngNS+`"
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
		require.Contains(t, g.CompileErrors(), "Found forbidden pattern data/except//ref")
	})
}
