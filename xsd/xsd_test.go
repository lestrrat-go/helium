package xsd_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

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
var skipExact = map[string]string{}

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
			schema, err := xsd.CompileFile(tc.xsdPath, xsd.WithSchemaFilename(xsdFilename))
			require.NoError(t, err, "schema compilation failed for %s", tc.xsdPath)

			var got string
			if schema.CompileErrors() != "" {
				got = schema.CompileWarnings() + schema.CompileErrors()
			} else {
				// Parse instance.
				xmlData, err := os.ReadFile(tc.xmlPath)
				require.NoError(t, err)
				doc, err := helium.Parse(xmlData)
				require.NoError(t, err, "XML parse failed for %s", tc.xmlPath)

				// Validate. Prepend any compile warnings to the output.
				filename := "./test/schemas/" + tc.xmlBase
				got = schema.CompileWarnings() + xsd.Validate(doc, schema, xsd.WithFilename(filename))
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

func TestXsiNil(t *testing.T) {
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

	schemaDoc, err := helium.Parse([]byte(schemaSrc))
	require.NoError(t, err)

	schema, err := xsd.Compile(schemaDoc)
	require.NoError(t, err)

	t.Run("nillable element with xsi:nil=true validates", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-elem xsi:nil="true"/>
</root>`))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("non-nillable element with xsi:nil=true fails", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <non-nillable-elem xsi:nil="true"/>
</root>`))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.Contains(t, result, "not nillable")
		require.Contains(t, result, "fails to validate")
	})

	t.Run("nilled element with text content fails", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-elem xsi:nil="true">some text</nillable-elem>
</root>`))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.Contains(t, result, "nilled")
		require.Contains(t, result, "fails to validate")
	})

	t.Run("nilled element with child element fails", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-complex xsi:nil="true"><child>x</child></nillable-complex>
</root>`))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.Contains(t, result, "nilled")
		require.Contains(t, result, "fails to validate")
	})

	t.Run("nilled complex element with attributes validates", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-complex xsi:nil="true" attr1="val"/>
</root>`))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("nillable element without xsi:nil validates normally", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-elem>hello</nillable-elem>
</root>`))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("xsi:nil=1 is equivalent to true", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-elem xsi:nil="1"/>
</root>`))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("xsi:nil=false does not trigger nil handling", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <nillable-elem xsi:nil="false">hello</nillable-elem>
</root>`))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
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
	compileAndValidate := func(t *testing.T, xsdStr, xmlStr string) string {
		t.Helper()
		xsdDoc, err := helium.Parse([]byte(xsdStr))
		require.NoError(t, err, "XSD parse failed")
		schema, err := xsd.Compile(xsdDoc)
		require.NoError(t, err, "schema compilation failed")
		require.Empty(t, schema.CompileErrors(), "unexpected compile errors")

		xmlDoc, err := helium.Parse([]byte(xmlStr))
		require.NoError(t, err, "XML parse failed")
		return xsd.Validate(xmlDoc, schema, xsd.WithFilename("test.xml"))
	}

	t.Run("element_fixed_correct_value", func(t *testing.T) {
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" fixed="hello"/>
</xs:schema>`
		xmlStr := `<root>hello</root>`
		result := compileAndValidate(t, xsdStr, xmlStr)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("element_fixed_wrong_value", func(t *testing.T) {
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" fixed="hello"/>
</xs:schema>`
		xmlStr := `<root>wrong</root>`
		result := compileAndValidate(t, xsdStr, xmlStr)
		require.Contains(t, result, "fixed value constraint")
		require.Contains(t, result, "fails to validate")
	})

	t.Run("element_fixed_empty", func(t *testing.T) {
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" fixed="hello"/>
</xs:schema>`
		xmlStr := `<root/>`
		result := compileAndValidate(t, xsdStr, xmlStr)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("element_default_empty_integer", func(t *testing.T) {
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:integer" default="42"/>
</xs:schema>`
		xmlStr := `<root/>`
		result := compileAndValidate(t, xsdStr, xmlStr)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("attribute_fixed_correct", func(t *testing.T) {
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="color" type="xs:string" fixed="red"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		xmlStr := `<root color="red"/>`
		result := compileAndValidate(t, xsdStr, xmlStr)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("attribute_fixed_wrong", func(t *testing.T) {
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="color" type="xs:string" fixed="red"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		xmlStr := `<root color="blue"/>`
		result := compileAndValidate(t, xsdStr, xmlStr)
		require.Contains(t, result, "fixed value constraint")
		require.Contains(t, result, "fails to validate")
	})
}

func TestMultipleAttributeErrors(t *testing.T) {
	compileAndValidate := func(t *testing.T, xsdStr, xmlStr string) string {
		t.Helper()
		xsdDoc, err := helium.Parse([]byte(xsdStr))
		require.NoError(t, err, "XSD parse failed")
		schema, err := xsd.Compile(xsdDoc)
		require.NoError(t, err, "schema compilation failed")
		require.Empty(t, schema.CompileErrors(), "unexpected compile errors")

		xmlDoc, err := helium.Parse([]byte(xmlStr))
		require.NoError(t, err, "XML parse failed")
		return xsd.Validate(xmlDoc, schema, xsd.WithFilename("test.xml"))
	}

	t.Run("multiple unknown attributes", func(t *testing.T) {
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="id" type="xs:string"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		xmlStr := `<root id="1" foo="x" bar="y"/>`
		result := compileAndValidate(t, xsdStr, xmlStr)
		require.Contains(t, result, "'foo' is not allowed")
		require.Contains(t, result, "'bar' is not allowed")
	})

	t.Run("multiple missing required attributes", func(t *testing.T) {
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="xs:string" use="required"/>
      <xs:attribute name="b" type="xs:string" use="required"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		xmlStr := `<root/>`
		result := compileAndValidate(t, xsdStr, xmlStr)
		require.Contains(t, result, "'a' is required but missing")
		require.Contains(t, result, "'b' is required but missing")
	})

	t.Run("no declarations multiple attrs", func(t *testing.T) {
		xsdStr := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`
		xmlStr := `<root x="1" y="2">text</root>`
		result := compileAndValidate(t, xsdStr, xmlStr)
		require.Contains(t, result, "'x' is not allowed")
		require.Contains(t, result, "'y' is not allowed")
	})
}

func TestRedefine(t *testing.T) {
	tmpDir := filepath.Join("..", ".tmp", "redefine-test")
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	writeFile := func(t *testing.T, name, content string) string {
		t.Helper()
		p := filepath.Join(tmpDir, name)
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
		return p
	}

	compileAndValidate := func(t *testing.T, xsdPath, xmlStr string) string {
		t.Helper()
		schema, err := xsd.CompileFile(xsdPath)
		require.NoError(t, err, "schema compilation failed")
		require.Empty(t, schema.CompileErrors(), "unexpected compile errors: %s", schema.CompileErrors())

		xmlDoc, err := helium.Parse([]byte(xmlStr))
		require.NoError(t, err, "XML parse failed")
		return xsd.Validate(xmlDoc, schema)
	}

	t.Run("complexType_extension", func(t *testing.T) {
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

		result := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<person><name>Alice</name><age>30</age></person>`)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")

		result = compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<person><name>Alice</name></person>`)
		require.Contains(t, result, "fails to validate")
	})

	t.Run("complexType_restriction", func(t *testing.T) {
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

		result := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<item><name>Widget</name></item>`)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("simpleType_restriction", func(t *testing.T) {
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

		result := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<val>Hi</val>`)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")

		result = compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<val>TooLongValue</val>`)
		require.Contains(t, result, "fails to validate")
	})

	t.Run("group_redefine", func(t *testing.T) {
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

		result := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<root><a>1</a><b>2</b></root>`)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")

		result = compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<root><a>1</a></root>`)
		require.Contains(t, result, "fails to validate")
	})

	t.Run("attributeGroup_redefine", func(t *testing.T) {
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

		result := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<root attr1="a" attr2="b"/>`)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("chameleon_redefine", func(t *testing.T) {
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

		result := compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<code xmlns="http://example.com/ns">ABC</code>`)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")

		result = compileAndValidate(t, mainPath, `<?xml version="1.0"?>
<code xmlns="http://example.com/ns">VeryLongCodeValue</code>`)
		require.Contains(t, result, "fails to validate")
	})
}

func TestFacetConsistency(t *testing.T) {
	compileWithErrors := func(t *testing.T, xsdStr string) string {
		t.Helper()
		xsdDoc, err := helium.Parse([]byte(xsdStr))
		require.NoError(t, err, "XSD parse failed")
		schema, err := xsd.Compile(xsdDoc, xsd.WithSchemaFilename("test.xsd"))
		require.NoError(t, err, "schema compilation failed")
		return schema.CompileErrors()
	}

	t.Run("minLength_greater_than_maxLength", func(t *testing.T) {
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
