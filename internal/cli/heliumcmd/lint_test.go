package heliumcmd_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func executeLint(t *testing.T, stdin io.Reader, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), stdin, &outBuf, &errBuf)
	ctx = heliumcmd.WithStdinTTY(ctx, true)
	allArgs := append([]string{"lint"}, args...)
	exit := heliumcmd.Execute(ctx, allArgs)
	return outBuf.String(), errBuf.String(), exit
}

func executeLintFile(t *testing.T, xmlPath string, flags ...string) (stdout, stderr string, code int) {
	t.Helper()
	args := append(flags, xmlPath)
	return executeLint(t, strings.NewReader(""), args...)
}

func executeLintStdin(t *testing.T, xmlContent string, flags ...string) (stdout, stderr string, code int) {
	t.Helper()
	dir := t.TempDir()
	f := writeFile(t, dir, "stdin.xml", xmlContent)
	return executeLintFile(t, f, flags...)
}

func TestParseArgsVersion(t *testing.T) {
	_, errOut, code := executeLint(t, strings.NewReader(""), "--version")
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Contains(t, errOut, "helium version")
}

func TestParseArgsUnrecognized(t *testing.T) {
	_, _, code := executeLint(t, strings.NewReader(""), "--nonexistent-flag")
	require.NotEqual(t, heliumcmd.ExitOK, code)
}

func TestParseArgsPrettyInvalid(t *testing.T) {
	_, _, code := executeLint(t, strings.NewReader(""), "--pretty", "xyz")
	require.NotEqual(t, heliumcmd.ExitOK, code)
}

func TestParseArgsRepeatInvalid(t *testing.T) {
	_, _, code := executeLint(t, strings.NewReader(""), "--repeat", "abc")
	require.NotEqual(t, heliumcmd.ExitOK, code)
}

func TestParseArgsRepeatZero(t *testing.T) {
	_, _, code := executeLint(t, strings.NewReader(""), "--repeat", "0")
	require.NotEqual(t, heliumcmd.ExitOK, code)
}

func TestParseArgsMissingValues(t *testing.T) {
	flags := []string{"--schema", "--xpath", "--output", "--encode", "--pretty", "--path", "--repeat"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			_, _, code := executeLint(t, strings.NewReader(""), flag)
			require.NotEqual(t, heliumcmd.ExitOK, code, "flag %s without value should fail", flag)
		})
	}
}

func TestBasicParse(t *testing.T) {
	out, _, code := executeLintStdin(t, `<root><child/></root>`)
	require.Equal(t, 0, code)
	require.Contains(t, out, `<?xml version="1.0"?>`)
	require.Contains(t, out, `<root><child/></root>`)
}

func TestBasicParseFile(t *testing.T) {
	dir := t.TempDir()
	f := writeFile(t, dir, "test.xml", `<?xml version="1.0"?><root/>`)
	out, _, code := executeLintFile(t, f)
	require.Equal(t, 0, code)
	require.Contains(t, out, `<root/>`)
}

func TestMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	f1 := writeFile(t, dir, "a.xml", `<?xml version="1.0"?><a/>`)
	f2 := writeFile(t, dir, "b.xml", `<?xml version="1.0"?><b/>`)

	out, _, code := executeLint(t, strings.NewReader(""), f1, f2)
	require.Equal(t, 0, code)
	require.Contains(t, out, `<a/>`)
	require.Contains(t, out, `<b/>`)
}

func TestFileNotFound(t *testing.T) {
	_, _, code := executeLintFile(t, "/nonexistent/file.xml")
	require.Equal(t, heliumcmd.ExitReadFile, code)
}

func TestMalformedXML(t *testing.T) {
	_, _, code := executeLintStdin(t, `<root><unclosed>`)
	require.NotEqual(t, 0, code)
}

func TestNoBlanks(t *testing.T) {
	out, _, code := executeLintStdin(t, `<a>   <b/>   </a>`, "--noblanks")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<a><b/></a>`)
}

func TestNoCDATA(t *testing.T) {
	out, _, code := executeLintStdin(t, `<a><![CDATA[hello]]></a>`, "--nocdata")
	require.Equal(t, 0, code)
	require.NotContains(t, out, `<![CDATA[`)
	require.Contains(t, out, `hello`)
}

func TestNoEnt(t *testing.T) {
	dir := t.TempDir()
	xml := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY greet "hello world">
]>
<doc>&greet;</doc>`
	f := writeFile(t, dir, "ent.xml", xml)

	out, _, code := executeLintFile(t, f, "--noent")
	require.Equal(t, 0, code)
	require.Contains(t, out, "hello world")
}

func TestRecover(t *testing.T) {
	out, _, code := executeLintStdin(t, `<root><a>text</a><b>`, "--recover")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<root>`)
}

func TestOutputControlModes(t *testing.T) {
	tests := []struct {
		name            string
		xml             string
		args            []string
		wantContains    []string
		wantNotContains []string
		wantEmpty       bool
	}{
		{
			name:      "noout",
			xml:       `<root/>`,
			args:      []string{"--noout"},
			wantEmpty: true,
		},
		{
			name:         "format",
			xml:          `<a><b><c>text</c></b></a>`,
			args:         []string{"--format"},
			wantContains: []string{"  <b>", "    <c>text</c>"},
		},
		{
			name:         "format preserves text",
			xml:          `<a><b>hello</b></a>`,
			args:         []string{"--format"},
			wantContains: []string{"<b>hello</b>"},
		},
		{
			name:         "pretty 0",
			xml:          `<a><b><c/></b></a>`,
			args:         []string{"--pretty", "0"},
			wantContains: []string{`<a><b><c/></b></a>`},
		},
		{
			name:         "pretty 1",
			xml:          `<a><b><c/></b></a>`,
			args:         []string{"--pretty", "1"},
			wantContains: []string{"  <b>"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, _, code := executeLintStdin(t, tc.xml, tc.args...)
			require.Equal(t, 0, code)
			if tc.wantEmpty {
				require.Empty(t, out)
			}
			for _, s := range tc.wantContains {
				require.Contains(t, out, s)
			}
			for _, s := range tc.wantNotContains {
				require.NotContains(t, out, s)
			}
		})
	}
}

func TestOutputFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.xml")
	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><root/>`)

	_, _, code := executeLint(t, strings.NewReader(""), "--output", outFile, xmlFile)
	require.Equal(t, 0, code)

	data, err := os.ReadFile(outFile) //nolint:gosec // reading test output file
	require.NoError(t, err)
	require.Contains(t, string(data), `<root/>`)
}

func TestC14NModes(t *testing.T) {
	tests := []struct {
		name         string
		xml          string
		args         []string
		wantContains []string
	}{
		{
			name:         "c14n",
			xml:          `<b a="1" c="2"/>`,
			args:         []string{"--c14n"},
			wantContains: []string{`<b a="1" c="2"></b>`},
		},
		{
			name:         "c14n attribute order",
			xml:          `<e z="1" a="2"/>`,
			args:         []string{"--c14n"},
			wantContains: []string{`<e a="2" z="1"></e>`},
		},
		{
			name:         "c14n comment",
			xml:          `<a><!-- comment --></a>`,
			args:         []string{"--c14n"},
			wantContains: []string{`<!-- comment -->`},
		},
		{
			name:         "c14n11",
			xml:          `<a/>`,
			args:         []string{"--c14n11"},
			wantContains: []string{`<a></a>`},
		},
		{
			name:         "exc-c14n",
			xml:          `<a xmlns:n="http://example.com"><b/></a>`,
			args:         []string{"--exc-c14n"},
			wantContains: []string{`<b></b>`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, _, code := executeLintStdin(t, tc.xml, tc.args...)
			require.Equal(t, 0, code)
			for _, s := range tc.wantContains {
				require.Contains(t, out, s)
			}
		})
	}
}

func TestC14NFile(t *testing.T) {
	dir := t.TempDir()
	f := writeFile(t, dir, "test.xml", `<?xml version="1.0"?><root z="2" a="1"/>`)

	out, _, code := executeLintFile(t, f, "--c14n")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<root a="1" z="2"></root>`)
}

func TestXPathExpressions(t *testing.T) {
	tests := []struct {
		name            string
		xml             string
		expr            string
		wantCode        int
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:         "node set",
			xml:          `<a><b>1</b><b>2</b></a>`,
			expr:         "//b",
			wantCode:     0,
			wantContains: []string{`<b>1</b>`, `<b>2</b>`},
		},
		{
			name:         "count",
			xml:          `<a><b/><b/><b/></a>`,
			expr:         "count(//b)",
			wantCode:     0,
			wantContains: []string{"3"},
		},
		{
			name:         "string",
			xml:          `<a><b>hello</b></a>`,
			expr:         "string(//b)",
			wantCode:     0,
			wantContains: []string{"hello"},
		},
		{
			name:         "boolean true",
			xml:          `<a><b/></a>`,
			expr:         "boolean(//b)",
			wantCode:     0,
			wantContains: []string{"true"},
		},
		{
			name:         "boolean false",
			xml:          `<a/>`,
			expr:         "boolean(//b)",
			wantCode:     0,
			wantContains: []string{"false"},
		},
		{
			name:            "xpath implies noout",
			xml:             `<a><b/></a>`,
			expr:            "count(//b)",
			wantCode:        0,
			wantNotContains: []string{`<?xml`},
		},
		{
			name:         "attribute",
			xml:          `<a foo="bar"/>`,
			expr:         "/a/@foo",
			wantCode:     0,
			wantContains: []string{"bar"},
		},
		{
			name:     "invalid expression",
			xml:      `<a/>`,
			expr:     "///invalid[[[",
			wantCode: heliumcmd.ExitXPath,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, _, code := executeLintStdin(t, tc.xml, "--xpath", tc.expr)
			require.Equal(t, tc.wantCode, code)
			for _, s := range tc.wantContains {
				require.Contains(t, out, s)
			}
			for _, s := range tc.wantNotContains {
				require.NotContains(t, out, s)
			}
		})
	}
}

func TestXPathWithFile(t *testing.T) {
	dir := t.TempDir()
	f := writeFile(t, dir, "test.xml", `<?xml version="1.0"?><a><b>1</b><b>2</b></a>`)

	out, _, code := executeLintFile(t, f, "--xpath", "//b")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<b>1</b>`)
	require.Contains(t, out, `<b>2</b>`)
}

func TestXInclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "included.xml", `<chapter>Hello</chapter>`)
	mainXML := `<?xml version="1.0"?>
<book xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="included.xml"/>
</book>`
	f := writeFile(t, dir, "main.xml", mainXML)

	out, _, code := executeLintFile(t, f, "--xinclude")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<chapter`)
	require.Contains(t, out, `Hello</chapter>`)
}

func TestXIncludeNoXIncludeMarker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "inc.xml", `<p>text</p>`)
	mainXML := `<?xml version="1.0"?>
<doc xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="inc.xml"/>
</doc>`
	f := writeFile(t, dir, "main.xml", mainXML)

	out, _, code := executeLintFile(t, f, "--xinclude", "--noxincludenode")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<p`)
	require.Contains(t, out, `text</p>`)
}

func TestXIncludeTextInclusion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hello.txt", "Hello, World!")
	mainXML := `<?xml version="1.0"?>
<doc xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="hello.txt" parse="text"/>
</doc>`
	f := writeFile(t, dir, "main.xml", mainXML)

	out, _, code := executeLintFile(t, f, "--xinclude", "--noxincludenode")
	require.Equal(t, 0, code)
	require.Contains(t, out, "Hello, World!")
}

func TestSchemaValidation(t *testing.T) {
	tests := []struct {
		name     string
		xsdType  string
		xmlValue string
		wantCode int
	}{
		{
			name:     "valid",
			xsdType:  "xs:string",
			xmlValue: "hello",
			wantCode: heliumcmd.ExitOK,
		},
		{
			name:     "invalid",
			xsdType:  "xs:integer",
			xmlValue: "not-an-integer",
			wantCode: heliumcmd.ExitValidation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			xsdContent := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="` + tc.xsdType + `"/>
</xs:schema>`
			xsdFile := writeFile(t, dir, "test.xsd", xsdContent)
			xmlFile := writeFile(t, dir, "test.xml", `<?xml version="1.0"?><root>`+tc.xmlValue+`</root>`)

			_, _, code := executeLintFile(t, xmlFile, "--schema", xsdFile, "--noout")
			require.Equal(t, tc.wantCode, code)
		})
	}
}

func TestDropDTD(t *testing.T) {
	dir := t.TempDir()
	xml := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (#PCDATA)>
]>
<doc>hello</doc>`
	f := writeFile(t, dir, "test.xml", xml)

	out, _, code := executeLintFile(t, f, "--dropdtd")
	require.Equal(t, 0, code)
	require.NotContains(t, out, `<!DOCTYPE`)
	require.Contains(t, out, `<doc>hello</doc>`)
}

func TestNsClean(t *testing.T) {
	xml := `<?xml version="1.0" encoding="US-ASCII"?>
<a xmlns:unused="http://example.com/unused">
  <b xmlns:unused="http://example.com/unused"/>
</a>`
	out, _, code := executeLintStdin(t, xml, "--nsclean")
	require.Equal(t, 0, code)
	count := strings.Count(out, `xmlns:unused`)
	require.Equal(t, 1, count, "redundant ns should be cleaned")
}

func TestFormatWithNoBlanks(t *testing.T) {
	out, _, code := executeLintStdin(t, `<a>   <b>   <c/> </b>   </a>`, "--noblanks", "--format")
	require.Equal(t, 0, code)
	require.Contains(t, out, "  <b>")
	require.Contains(t, out, "    <c/>")
}

func TestC14NWithOutput(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.xml")
	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><b a="1"/>`)

	_, _, code := executeLint(t, strings.NewReader(""), "--c14n", "--output", outFile, xmlFile)
	require.Equal(t, 0, code)

	data, err := os.ReadFile(outFile) //nolint:gosec // reading test output file
	require.NoError(t, err)
	require.Contains(t, string(data), `<b a="1"></b>`)
}

func TestXPathWithNoOut(t *testing.T) {
	out, _, code := executeLintStdin(t, `<a><b>42</b></a>`, "--xpath", "string(//b)")
	require.Equal(t, 0, code)
	require.Contains(t, out, "42")
	require.NotContains(t, out, `<?xml`)
}

func TestSchemaValidQuiet(t *testing.T) {
	dir := t.TempDir()
	xsdContent := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`
	xsdFile := writeFile(t, dir, "test.xsd", xsdContent)
	xmlFile := writeFile(t, dir, "test.xml", `<?xml version="1.0"?><root>hello</root>`)

	_, _, code := executeLintFile(t, xmlFile, "--schema", xsdFile, "--noout", "--quiet")
	require.Equal(t, heliumcmd.ExitOK, code)
}

func TestCatalogLoading(t *testing.T) {
	dir := t.TempDir()

	dtdContent := `<!ATTLIST doc status CDATA "active">`
	writeFile(t, dir, "test.dtd", dtdContent)

	catContent := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="http://example.com/test.dtd" uri="` + filepath.Join(dir, "test.dtd") + `"/>
</catalog>`
	catFile := writeFile(t, dir, "catalog.xml", catContent)

	xmlContent := `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "http://example.com/test.dtd">
<doc>hello</doc>`
	xmlFile := writeFile(t, dir, "test.xml", xmlContent)

	t.Setenv("XML_CATALOG_FILES", catFile)

	_, _, code := executeLintFile(t, xmlFile, "--catalogs", "--loaddtd", "--dtdattr", "--noout")
	require.Equal(t, heliumcmd.ExitOK, code)
}

func TestFormatNestedElements(t *testing.T) {
	out, _, code := executeLintStdin(t, `<a><b><c><d/></c></b></a>`, "--format")
	require.Equal(t, 0, code)
	lines := strings.Split(out, "\n")
	var indented []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<") && trimmed != `<?xml version="1.0"?>` {
			indented = append(indented, line)
		}
	}
	require.NotEmpty(t, indented)
}

func TestFormatAdditionalCases(t *testing.T) {
	tests := []struct {
		name         string
		xml          string
		wantContains []string
	}{
		{
			name:         "mixed content",
			xml:          `<a>text<b/></a>`,
			wantContains: []string{`<a>`, `<b/>`},
		},
		{
			name:         "empty element",
			xml:          `<a><b/></a>`,
			wantContains: []string{`  <b/>`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, _, code := executeLintStdin(t, tc.xml, "--format")
			require.Equal(t, 0, code)
			for _, s := range tc.wantContains {
				require.Contains(t, out, s)
			}
		})
	}
}
