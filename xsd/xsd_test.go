package xsd_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
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

// validateWithOutput validates a document and optionally collects error strings.
// When out is non-nil, an ErrorHandler is installed and the collected errors
// are assigned to *out.
func validateWithOutput(t *testing.T, v xsd.Validator, doc *helium.Document, out *string) error {
	t.Helper()
	if out != nil {
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		v = v.ErrorHandler(collector)
		err := v.Validate(t.Context(), doc)
		var b strings.Builder
		for _, e := range collector.Errors() {
			b.WriteString(e.Error())
		}
		*out = b.String()
		return err
	}
	return v.Validate(t.Context(), doc)
}

const testdataBase = "../testdata/libxml2-compat/schemas"

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

	resultDir := filepath.Join(testdataBase, "result")
	schemaDir := filepath.Join(testdataBase, "test")

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
	// IDC: libxml2 IDC field evaluation quirk with ref elements + attributeFormDefault="qualified".
	"idc-keyref-err1": "libxml2 IDC quirk with ref + attributeFormDefault",
}

// skipExact lists specific test cases (by full test name) that need skipping
// when their group-level skip has been removed.
var skipExact = map[string]string{
	// bug310264: greedy content model validation does not backtrack for nested
	// sequences with minOccurs/maxOccurs (inner sequence consumes 20 of 38
	// elements, leaving only 18 for second outer iteration which needs 19).
	// Pre-existing limitation exposed by resolveRefs fix for element/type
	// name collisions.
	"bug310264_0_0": "greedy nested sequence validation does not backtrack",
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
	t.Parallel()
	filterEnv := os.Getenv("HELIUM_XMLSCHEMA_TEST_FILES")

	cases := discoverTests(t)
	require.NotEmpty(t, cases, "no test cases discovered")

	var passed, skipped, failed atomic.Int64

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if filterEnv != "" && !strings.Contains(tc.name, filterEnv) {
				t.Skip("filtered out by HELIUM_XMLSCHEMA_TEST_FILES")
				skipped.Add(1)
				return
			}

			// Check skip list (exact match first, then base name prefix).
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

			// Read expected output.
			expected, err := os.ReadFile(tc.errPath)
			require.NoError(t, err)

			// Compile schema.
			xsdFilename := "./test/schemas/" + tc.xsdBase
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			schema, err := xsd.NewCompiler().SchemaFilename(xsdFilename).ErrorHandler(collector).CompileFile(t.Context(), tc.xsdPath)
			require.NoError(t, err, "schema compilation failed for %s", tc.xsdPath)
			_ = collector.Close()
			compileWarnings, compileErrors := partitionCompileErrors(collector.Errors())

			var got string
			if compileErrors != "" {
				got = compileWarnings + compileErrors
			} else {
				// Parse instance.
				xmlData, err := os.ReadFile(tc.xmlPath)
				require.NoError(t, err)
				doc, err := helium.NewParser().Parse(t.Context(), xmlData)
				require.NoError(t, err, "XML parse failed for %s", tc.xmlPath)

				// Validate. Collect validation errors via ErrorHandler.
				filename := "./test/schemas/" + tc.xmlBase
				var valErrs string
				err = validateWithOutput(t, xsd.NewValidator(schema).Filename(filename), doc, &valErrs)
				if err != nil {
					got = compileWarnings + valErrs + filename + " fails to validate\n"
				} else {
					got = compileWarnings + valErrs + filename + " validates\n"
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

func TestXsiNil(t *testing.T) {
	t.Parallel()
	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="nillable-elem" type="xs:string" nillable="true" minOccurs="0"/>
        <xs:element name="non-nillable-elem" type="xs:string" minOccurs="0"/>
        <xs:element name="nillable-complex" nillable="true" minOccurs="0">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="child" type="xs:string" minOccurs="0"/>
            </xs:sequence>
            <xs:attribute name="attr1" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaSrc))
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDoc)
	require.NoError(t, err)

	t.Run("nillable element with xsi:nil=true validates", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-elem xsi:nil="true"/>
</root>`))
		require.NoError(t, err)

		err = xsd.NewValidator(schema).Validate(t.Context(), doc)
		require.NoError(t, err)
	})

	t.Run("non-nillable element with xsi:nil=true fails", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <non-nillable-elem xsi:nil="true"/>
</root>`))
		require.NoError(t, err)

		var errs string
		err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "not nillable")
	})

	t.Run("nilled element with text content fails", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-elem xsi:nil="true">some text</nillable-elem>
</root>`))
		require.NoError(t, err)

		var errs string
		err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "nilled")
	})

	t.Run("nilled element with child element fails", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-complex xsi:nil="true"><child>x</child></nillable-complex>
</root>`))
		require.NoError(t, err)

		var errs string
		err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "nilled")
	})

	t.Run("nilled complex element with attributes validates", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-complex xsi:nil="true" attr1="val"/>
</root>`))
		require.NoError(t, err)

		err = xsd.NewValidator(schema).Validate(t.Context(), doc)
		require.NoError(t, err)
	})

	t.Run("nillable element without xsi:nil validates normally", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-elem>hello</nillable-elem>
</root>`))
		require.NoError(t, err)

		err = xsd.NewValidator(schema).Validate(t.Context(), doc)
		require.NoError(t, err)
	})

	t.Run("xsi:nil=1 is equivalent to true", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-elem xsi:nil="1"/>
</root>`))
		require.NoError(t, err)

		err = xsd.NewValidator(schema).Validate(t.Context(), doc)
		require.NoError(t, err)
	})

	t.Run("xsi:nil=false does not trigger nil handling", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-elem xsi:nil="false">hello</nillable-elem>
</root>`))
		require.NoError(t, err)

		err = xsd.NewValidator(schema).Validate(t.Context(), doc)
		require.NoError(t, err)
	})
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

func TestDefaultFixedValidation(t *testing.T) {
	t.Parallel()
	compileAndValidate := func(t *testing.T, xsdStr, xmlStr string, out *string) error {
		t.Helper()
		xsdDoc, err := helium.NewParser().Parse(t.Context(), []byte(xsdStr))
		require.NoError(t, err, "XSD parse failed")
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		schema, err := xsd.NewCompiler().ErrorHandler(collector).Compile(t.Context(), xsdDoc)
		require.NoError(t, err, "schema compilation failed")
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		require.Empty(t, compileErrors, "unexpected compile errors")

		xmlDoc, err := helium.NewParser().Parse(t.Context(), []byte(xmlStr))
		require.NoError(t, err, "XML parse failed")
		v := xsd.NewValidator(schema).Filename("test.xml")
		if out != nil {
			valCollector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			v = v.ErrorHandler(valCollector)
			err = v.Validate(t.Context(), xmlDoc)
			var b strings.Builder
			for _, e := range valCollector.Errors() {
				b.WriteString(e.Error())
			}
			*out = b.String()
			return err
		}
		return v.Validate(t.Context(), xmlDoc)
	}

	t.Run("element_fixed_correct_value", func(t *testing.T) {
		t.Parallel()
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" fixed="hello"/>
</xs:schema>`
		xmlStr := `<root>hello</root>`
		err := compileAndValidate(t, xsdStr, xmlStr, nil)
		require.NoError(t, err)
	})

	t.Run("element_fixed_wrong_value", func(t *testing.T) {
		t.Parallel()
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" fixed="hello"/>
</xs:schema>`
		xmlStr := `<root>wrong</root>`
		var errs string
		err := compileAndValidate(t, xsdStr, xmlStr, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "fixed value constraint")
	})

	t.Run("element_fixed_empty", func(t *testing.T) {
		t.Parallel()
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" fixed="hello"/>
</xs:schema>`
		xmlStr := `<root/>`
		err := compileAndValidate(t, xsdStr, xmlStr, nil)
		require.NoError(t, err)
	})

	t.Run("element_default_empty_integer", func(t *testing.T) {
		t.Parallel()
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:integer" default="42"/>
</xs:schema>`
		xmlStr := `<root/>`
		err := compileAndValidate(t, xsdStr, xmlStr, nil)
		require.NoError(t, err)
	})

	t.Run("attribute_fixed_correct", func(t *testing.T) {
		t.Parallel()
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="color" type="xs:string" fixed="red"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		xmlStr := `<root color="red"/>`
		err := compileAndValidate(t, xsdStr, xmlStr, nil)
		require.NoError(t, err)
	})

	t.Run("attribute_fixed_wrong", func(t *testing.T) {
		t.Parallel()
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="color" type="xs:string" fixed="red"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		xmlStr := `<root color="blue"/>`
		var errs string
		err := compileAndValidate(t, xsdStr, xmlStr, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "fixed value constraint")
	})
}

func TestMultipleAttributeErrors(t *testing.T) {
	t.Parallel()
	compileAndValidate := func(t *testing.T, xsdStr, xmlStr string, out *string) error {
		t.Helper()
		xsdDoc, err := helium.NewParser().Parse(t.Context(), []byte(xsdStr))
		require.NoError(t, err, "XSD parse failed")
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		schema, err := xsd.NewCompiler().ErrorHandler(collector).Compile(t.Context(), xsdDoc)
		require.NoError(t, err, "schema compilation failed")
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		require.Empty(t, compileErrors, "unexpected compile errors")

		xmlDoc, err := helium.NewParser().Parse(t.Context(), []byte(xmlStr))
		require.NoError(t, err, "XML parse failed")
		v := xsd.NewValidator(schema).Filename("test.xml")
		if out != nil {
			valCollector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			v = v.ErrorHandler(valCollector)
			err = v.Validate(t.Context(), xmlDoc)
			var b strings.Builder
			for _, e := range valCollector.Errors() {
				b.WriteString(e.Error())
			}
			*out = b.String()
			return err
		}
		return v.Validate(t.Context(), xmlDoc)
	}

	t.Run("multiple unknown attributes", func(t *testing.T) {
		t.Parallel()
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="id" type="xs:string"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		xmlStr := `<root id="1" foo="x" bar="y"/>`
		var errs string
		err := compileAndValidate(t, xsdStr, xmlStr, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "'foo' is not allowed")
		require.Contains(t, errs, "'bar' is not allowed")
	})

	t.Run("multiple missing required attributes", func(t *testing.T) {
		t.Parallel()
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="xs:string" use="required"/>
      <xs:attribute name="b" type="xs:string" use="required"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		xmlStr := `<root/>`
		var errs string
		err := compileAndValidate(t, xsdStr, xmlStr, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "'a' is required but missing")
		require.Contains(t, errs, "'b' is required but missing")
	})

	t.Run("no declarations multiple attrs", func(t *testing.T) {
		t.Parallel()
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`
		xmlStr := `<root x="1" y="2">text</root>`
		var errs string
		err := compileAndValidate(t, xsdStr, xmlStr, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "'x' is not allowed")
		require.Contains(t, errs, "'y' is not allowed")
	})
}

func TestRedefine(t *testing.T) {
	t.Parallel()
	tmpDir := filepath.Join("..", ".tmp", "redefine-test")
	require.NoError(t, os.MkdirAll(tmpDir, 0o750)) //nolint:gosec // test temp directory
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	writeFile := func(t *testing.T, name, content string) string {
		t.Helper()
		p := filepath.Join(tmpDir, name)
		require.NoError(t, os.WriteFile(p, []byte(content), 0o600)) //nolint:gosec // test helper writing temp files
		return p
	}

	compileAndValidate := func(t *testing.T, xsdPath, xmlStr string) error {
		t.Helper()
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		schema, err := xsd.NewCompiler().ErrorHandler(collector).CompileFile(t.Context(), xsdPath)
		require.NoError(t, err, "schema compilation failed")
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		require.Empty(t, compileErrors, "unexpected compile errors: %s", compileErrors)

		xmlDoc, err := helium.NewParser().Parse(t.Context(), []byte(xmlStr))
		require.NoError(t, err, "XML parse failed")
		return xsd.NewValidator(schema).Validate(t.Context(), xmlDoc)
	}

	t.Run("complexType_extension", func(t *testing.T) {
		t.Parallel()
		writeFile(t, "base-ct-ext.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="personType">
    <xs:sequence>
      <xs:element name="name" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

		mainPath := writeFile(t, "main-ct-ext.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base-ct-ext.xsd">
    <xs:complexType name="personType">
      <xs:complexContent>
        <xs:extension base="personType">
          <xs:sequence>
            <xs:element name="age" type="xs:integer"/>
          </xs:sequence>
        </xs:extension>
      </xs:complexContent>
    </xs:complexType>
  </xs:redefine>
  <xs:element name="person" type="personType"/>
</xs:schema>`)

		err := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<person><name>Alice</name><age>30</age></person>`)
		require.NoError(t, err)

		err = compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<person><name>Alice</name></person>`)
		require.Error(t, err)
	})

	t.Run("complexType_restriction", func(t *testing.T) {
		t.Parallel()
		writeFile(t, "base-ct-restr.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="itemType">
    <xs:sequence>
      <xs:element name="name" type="xs:string"/>
      <xs:element name="desc" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

		mainPath := writeFile(t, "main-ct-restr.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base-ct-restr.xsd">
    <xs:complexType name="itemType">
      <xs:complexContent>
        <xs:restriction base="itemType">
          <xs:sequence>
            <xs:element name="name" type="xs:string"/>
          </xs:sequence>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:redefine>
  <xs:element name="item" type="itemType"/>
</xs:schema>`)

		err := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<item><name>Widget</name></item>`)
		require.NoError(t, err)
	})

	t.Run("simpleType_restriction", func(t *testing.T) {
		t.Parallel()
		writeFile(t, "base-st.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="nameType">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)

		mainPath := writeFile(t, "main-st.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base-st.xsd">
    <xs:simpleType name="nameType">
      <xs:restriction base="nameType">
        <xs:maxLength value="5"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:redefine>
  <xs:element name="val" type="nameType"/>
</xs:schema>`)

		err := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<val>Hi</val>`)
		require.NoError(t, err)

		err = compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<val>TooLongValue</val>`)
		require.Error(t, err)
	})

	t.Run("group_redefine", func(t *testing.T) {
		t.Parallel()
		writeFile(t, "base-grp.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="fieldsGroup">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:group>
</xs:schema>`)

		mainPath := writeFile(t, "main-grp.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base-grp.xsd">
    <xs:group name="fieldsGroup">
      <xs:sequence>
        <xs:group ref="fieldsGroup"/>
        <xs:element name="b" type="xs:string"/>
      </xs:sequence>
    </xs:group>
  </xs:redefine>
  <xs:element name="root">
    <xs:complexType>
      <xs:group ref="fieldsGroup"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)

		err := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<root><a>1</a><b>2</b></root>`)
		require.NoError(t, err)

		err = compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<root><a>1</a></root>`)
		require.Error(t, err)
	})

	t.Run("attributeGroup_redefine", func(t *testing.T) {
		t.Parallel()
		writeFile(t, "base-ag.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="myAttrs">
    <xs:attribute name="attr1" type="xs:string"/>
  </xs:attributeGroup>
</xs:schema>`)

		mainPath := writeFile(t, "main-ag.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base-ag.xsd">
    <xs:attributeGroup name="myAttrs">
      <xs:attributeGroup ref="myAttrs"/>
      <xs:attribute name="attr2" type="xs:string"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:element name="root">
    <xs:complexType>
      <xs:attributeGroup ref="myAttrs"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)

		err := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<root attr1="a" attr2="b"/>`)
		require.NoError(t, err)
	})

	t.Run("chameleon_redefine", func(t *testing.T) {
		t.Parallel()
		// Redefined schema has no targetNamespace — adopts the including schema's NS.
		writeFile(t, "base-chameleon.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="codeType">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)

		mainPath := writeFile(t, "main-chameleon.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/ns"
           xmlns:tns="http://example.com/ns">
  <xs:redefine schemaLocation="base-chameleon.xsd">
    <xs:simpleType name="codeType">
      <xs:restriction base="tns:codeType">
        <xs:maxLength value="10"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:redefine>
  <xs:element name="code" type="tns:codeType"/>
</xs:schema>`)

		err := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<code xmlns="http://example.com/ns">ABC</code>`)
		require.NoError(t, err)

		err = compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<code xmlns="http://example.com/ns">VeryLongCodeValue</code>`)
		require.Error(t, err)
	})
}

func TestFacetConsistency(t *testing.T) {
	t.Parallel()
	compileWithErrors := func(t *testing.T, xsdStr string) string {
		t.Helper()
		xsdDoc, err := helium.NewParser().Parse(t.Context(), []byte(xsdStr))
		require.NoError(t, err, "XSD parse failed")
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().SchemaFilename("test.xsd").ErrorHandler(collector).Compile(t.Context(), xsdDoc)
		require.NoError(t, err, "schema compilation failed")
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		return compileErrors
	}

	t.Run("minLength_greater_than_maxLength", func(t *testing.T) {
		t.Parallel()
		errs := compileWithErrors(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="badType">
    <xs:restriction base="xs:string">
      <xs:minLength value="5"/>
      <xs:maxLength value="3"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)
		require.Contains(t, errs, "'minLength' to be greater than the value of 'maxLength'")
	})

	t.Run("length_with_minLength", func(t *testing.T) {
		t.Parallel()
		errs := compileWithErrors(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="badType">
    <xs:restriction base="xs:string">
      <xs:length value="5"/>
      <xs:minLength value="3"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)
		require.Contains(t, errs, "both 'length' and either of 'minLength' or 'maxLength'")
	})

	t.Run("maxInclusive_with_maxExclusive", func(t *testing.T) {
		t.Parallel()
		errs := compileWithErrors(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="badType">
    <xs:restriction base="xs:decimal">
      <xs:maxInclusive value="10"/>
      <xs:maxExclusive value="10"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)
		require.Contains(t, errs, "both 'maxInclusive' and 'maxExclusive'")
	})

	t.Run("minInclusive_with_minExclusive", func(t *testing.T) {
		t.Parallel()
		errs := compileWithErrors(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="badType">
    <xs:restriction base="xs:decimal">
      <xs:minInclusive value="5"/>
      <xs:minExclusive value="5"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)
		require.Contains(t, errs, "both 'minInclusive' and 'minExclusive'")
	})

	t.Run("minInclusive_greater_than_maxInclusive", func(t *testing.T) {
		t.Parallel()
		errs := compileWithErrors(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="badType">
    <xs:restriction base="xs:decimal">
      <xs:minInclusive value="10"/>
      <xs:maxInclusive value="5"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)
		require.Contains(t, errs, "'minInclusive' to be greater than the value of 'maxInclusive'")
	})

	t.Run("fractionDigits_greater_than_totalDigits", func(t *testing.T) {
		t.Parallel()
		errs := compileWithErrors(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="badType">
    <xs:restriction base="xs:decimal">
      <xs:totalDigits value="3"/>
      <xs:fractionDigits value="5"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)
		require.Contains(t, errs, "'fractionDigits' to be greater than the value of 'totalDigits'")
	})

	t.Run("valid_minLength_maxLength", func(t *testing.T) {
		t.Parallel()
		errs := compileWithErrors(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="goodType">
    <xs:restriction base="xs:string">
      <xs:minLength value="2"/>
      <xs:maxLength value="5"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)
		require.Empty(t, errs)
	})

	t.Run("derived_widens_maxLength", func(t *testing.T) {
		t.Parallel()
		errs := compileWithErrors(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseType">
    <xs:restriction base="xs:string">
      <xs:maxLength value="5"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="derivedType">
    <xs:restriction base="baseType">
      <xs:maxLength value="10"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)
		require.Contains(t, errs, "'maxLength' value '10' is greater than the 'maxLength' value of the base type '5'")
	})

	t.Run("derived_widens_minLength", func(t *testing.T) {
		t.Parallel()
		errs := compileWithErrors(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseType">
    <xs:restriction base="xs:string">
      <xs:minLength value="3"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="derivedType">
    <xs:restriction base="baseType">
      <xs:minLength value="1"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)
		require.Contains(t, errs, "'minLength' value '1' is less than the 'minLength' value of the base type '3'")
	})
}

func TestWithAnnotations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="rootType"/>
  <xs:complexType name="rootType">
    <xs:sequence>
      <xs:element name="name" type="xs:string"/>
      <xs:element name="age" type="xs:integer"/>
    </xs:sequence>
    <xs:attribute name="id" type="xs:ID"/>
  </xs:complexType>
</xs:schema>`

	instanceXML := `<root id="r1"><name>Alice</name><age>30</age></root>`

	schemaDoc, err := helium.NewParser().Parse(ctx, []byte(schemaXML))
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().Compile(ctx, schemaDoc)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(ctx, []byte(instanceXML))
	require.NoError(t, err)

	var ann xsd.TypeAnnotations
	err = xsd.NewValidator(schema).Annotations(&ann).Validate(ctx, doc)
	require.NoError(t, err)
	require.NotEmpty(t, ann)

	// Collect annotations by node name for easier assertion.
	byName := make(map[string]string)
	for node, typeName := range ann {
		switch n := node.(type) {
		case *helium.Element:
			byName["elem:"+n.LocalName()] = typeName
		case *helium.Attribute:
			byName["attr:"+n.LocalName()] = typeName
		}
	}

	require.Equal(t, "Q{}rootType", byName["elem:root"])
	require.Equal(t, "xs:string", byName["elem:name"])
	require.Equal(t, "xs:integer", byName["elem:age"])
	require.Equal(t, "xs:ID", byName["attr:id"])
}

func TestZeroCompilerFluent(t *testing.T) {
	t.Parallel()
	var c xsd.Compiler
	require.NotPanics(t, func() {
		c2 := c.SchemaFilename("test.xsd").BaseDir("/tmp")
		_ = c2
	})
}

func TestZeroValidatorFluent(t *testing.T) {
	t.Parallel()
	var v xsd.Validator
	require.NotPanics(t, func() {
		v2 := v.Filename("test.xml")
		_ = v2
	})
}
