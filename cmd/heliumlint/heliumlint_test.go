package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// writeFile creates a file in dir with given name and content, returns full path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0644))
	return p
}

// runWithFile processes a single XML file through heliumlint with the given args.
// Returns stdout content and exit code.
func runWithFile(t *testing.T, xmlPath string, args ...string) (string, int) {
	t.Helper()

	allArgs := append(args, xmlPath)
	cfg, files := parseArgs(allArgs)
	require.NotNil(t, cfg, "parseArgs returned nil config for args: %v", allArgs)
	require.NotEmpty(t, files, "no files collected from args: %v", allArgs)

	if cfg.pretty >= 1 {
		cfg.format = true
	}

	var out strings.Builder
	input := namedInput{name: files[0]}
	code := processInput(cfg, input, nil, nil, &out)
	return out.String(), code
}

// runWithStdin processes XML from a string (simulating stdin) with the given args.
// Returns stdout content and exit code.
func runWithStdin(t *testing.T, xmlContent string, args ...string) (string, int) {
	t.Helper()

	dir := t.TempDir()
	f := writeFile(t, dir, "stdin.xml", xmlContent)
	return runWithFile(t, f, args...)
}

// =====================================================================
// parseArgs unit tests
// =====================================================================

func TestParseArgsDefaults(t *testing.T) {
	cfg, files := parseArgs([]string{})
	require.NotNil(t, cfg)
	require.Empty(t, files)
	require.Equal(t, -1, cfg.pretty)
	require.Equal(t, 1, cfg.repeat)
	require.False(t, cfg.version)
	require.False(t, cfg.noout)
	require.False(t, cfg.format)
	require.Equal(t, 0, cfg.c14nMode)
	require.Equal(t, helium.ParseOption(0), cfg.parseOptions)
}

func TestParseArgsVersion(t *testing.T) {
	cfg, files := parseArgs([]string{"--version"})
	require.NotNil(t, cfg)
	require.True(t, cfg.version)
	require.Empty(t, files)
}

func TestParseArgsSingleDash(t *testing.T) {
	cfg, _ := parseArgs([]string{"-noblanks"})
	require.NotNil(t, cfg)
	require.True(t, cfg.parseOptions.IsSet(helium.ParseNoBlanks))
}

func TestParseArgsAllParserFlags(t *testing.T) {
	args := []string{
		"--recover", "--noent", "--loaddtd", "--pedantic",
		"--noblanks", "--nsclean", "--nocdata", "--nonet",
		"--huge", "--noenc", "--noxincludenode", "--nofixup-base-uris",
	}
	cfg, _ := parseArgs(args)
	require.NotNil(t, cfg)
	require.True(t, cfg.parseOptions.IsSet(helium.ParseRecover))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseNoEnt))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseDTDLoad))
	require.True(t, cfg.parseOptions.IsSet(helium.ParsePedantic))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseNoBlanks))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseNsClean))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseNoCDATA))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseNoNet))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseHuge))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseIgnoreEnc))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseNoXIncNode))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseNoBaseFix))
}

func TestParseArgsDtdattr(t *testing.T) {
	cfg, _ := parseArgs([]string{"--dtdattr"})
	require.NotNil(t, cfg)
	require.True(t, cfg.parseOptions.IsSet(helium.ParseDTDAttr))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseDTDLoad))
}

func TestParseArgsValid(t *testing.T) {
	cfg, _ := parseArgs([]string{"--valid"})
	require.NotNil(t, cfg)
	require.True(t, cfg.parseOptions.IsSet(helium.ParseDTDValid))
	require.True(t, cfg.parseOptions.IsSet(helium.ParseDTDLoad))
}

func TestParseArgsXInclude(t *testing.T) {
	cfg, _ := parseArgs([]string{"--xinclude"})
	require.NotNil(t, cfg)
	require.True(t, cfg.doXInclude)
	require.True(t, cfg.parseOptions.IsSet(helium.ParseXInclude))
}

func TestParseArgsXPathImpliesNoout(t *testing.T) {
	cfg, _ := parseArgs([]string{"--xpath", "//a"})
	require.NotNil(t, cfg)
	require.Equal(t, "//a", cfg.xpathExpr)
	require.True(t, cfg.noout)
}

func TestParseArgsC14nModes(t *testing.T) {
	tests := []struct {
		flag string
		mode int
	}{
		{"--c14n", 1},
		{"--c14n11", 2},
		{"--exc-c14n", 3},
	}
	for _, tc := range tests {
		cfg, _ := parseArgs([]string{tc.flag})
		require.NotNil(t, cfg, "flag=%s", tc.flag)
		require.Equal(t, tc.mode, cfg.c14nMode, "flag=%s", tc.flag)
	}
}

func TestParseArgsPretty(t *testing.T) {
	cfg, _ := parseArgs([]string{"--pretty", "2"})
	require.NotNil(t, cfg)
	require.Equal(t, 2, cfg.pretty)
}

func TestParseArgsRepeat(t *testing.T) {
	cfg, _ := parseArgs([]string{"--repeat", "5"})
	require.NotNil(t, cfg)
	require.Equal(t, 5, cfg.repeat)
}

func TestParseArgsSchema(t *testing.T) {
	cfg, files := parseArgs([]string{"--schema", "test.xsd", "test.xml"})
	require.NotNil(t, cfg)
	require.Equal(t, "test.xsd", cfg.schemaFile)
	require.Equal(t, []string{"test.xml"}, files)
}

func TestParseArgsOutput(t *testing.T) {
	cfg, files := parseArgs([]string{"--output", "out.xml", "in.xml"})
	require.NotNil(t, cfg)
	require.Equal(t, "out.xml", cfg.outputFile)
	require.Equal(t, []string{"in.xml"}, files)
}

func TestParseArgsEncode(t *testing.T) {
	cfg, _ := parseArgs([]string{"--encode", "UTF-8"})
	require.NotNil(t, cfg)
	require.Equal(t, "UTF-8", cfg.encode)
}

func TestParseArgsPath(t *testing.T) {
	cfg, _ := parseArgs([]string{"--path", "/usr/share/dtd"})
	require.NotNil(t, cfg)
	require.Equal(t, "/usr/share/dtd", cfg.pathDirs)
}

func TestParseArgsCatalogs(t *testing.T) {
	cfg, _ := parseArgs([]string{"--catalogs"})
	require.NotNil(t, cfg)
	require.True(t, cfg.catalogs)
}

func TestParseArgsNoCatalogs(t *testing.T) {
	cfg, _ := parseArgs([]string{"--nocatalogs"})
	require.NotNil(t, cfg)
	require.True(t, cfg.noCatalogs)
}

func TestParseArgsBooleanFlags(t *testing.T) {
	cfg, _ := parseArgs([]string{"--noout", "--format", "--quiet", "--timing", "--dropdtd", "--nowarning"})
	require.NotNil(t, cfg)
	require.True(t, cfg.noout)
	require.True(t, cfg.format)
	require.True(t, cfg.quiet)
	require.True(t, cfg.timing)
	require.True(t, cfg.dropdtd)
	require.True(t, cfg.parseOptions.IsSet(helium.ParseNoWarning))
}

func TestParseArgsFilesCollected(t *testing.T) {
	cfg, files := parseArgs([]string{"--noblanks", "a.xml", "b.xml"})
	require.NotNil(t, cfg)
	require.Equal(t, []string{"a.xml", "b.xml"}, files)
}

func TestParseArgsUnrecognized(t *testing.T) {
	cfg, _ := parseArgs([]string{"--nonexistent-flag"})
	require.Nil(t, cfg)
}

func TestParseArgsMissingValues(t *testing.T) {
	flags := []string{"--schema", "--xpath", "--output", "--encode", "--pretty", "--path", "--repeat"}
	for _, flag := range flags {
		cfg, _ := parseArgs([]string{flag})
		require.Nil(t, cfg, "flag %s without value should fail", flag)
	}
}

func TestParseArgsPrettyInvalid(t *testing.T) {
	cfg, _ := parseArgs([]string{"--pretty", "xyz"})
	require.Nil(t, cfg)
}

func TestParseArgsRepeatInvalid(t *testing.T) {
	cfg, _ := parseArgs([]string{"--repeat", "abc"})
	require.Nil(t, cfg)
}

func TestParseArgsRepeatZero(t *testing.T) {
	cfg, _ := parseArgs([]string{"--repeat", "0"})
	require.Nil(t, cfg)
}

// =====================================================================
// processInput integration tests
// =====================================================================

func TestBasicParse(t *testing.T) {
	out, code := runWithStdin(t, `<root><child/></root>`)
	require.Equal(t, 0, code)
	require.Contains(t, out, `<?xml version="1.0"?>`)
	require.Contains(t, out, `<root><child/></root>`)
}

func TestBasicParseFile(t *testing.T) {
	dir := t.TempDir()
	f := writeFile(t, dir, "test.xml", `<?xml version="1.0"?><root/>`)

	out, code := runWithFile(t, f)
	require.Equal(t, 0, code)
	require.Contains(t, out, `<root/>`)
}

func TestMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	f1 := writeFile(t, dir, "a.xml", `<?xml version="1.0"?><a/>`)
	f2 := writeFile(t, dir, "b.xml", `<?xml version="1.0"?><b/>`)

	cfg, files := parseArgs([]string{f1, f2})
	require.NotNil(t, cfg)

	var out strings.Builder
	for _, fn := range files {
		processInput(cfg, namedInput{name: fn}, nil, nil, &out)
	}
	result := out.String()
	require.Contains(t, result, `<a/>`)
	require.Contains(t, result, `<b/>`)
}

func TestFileNotFound(t *testing.T) {
	cfg, _ := parseArgs([]string{})
	require.NotNil(t, cfg)

	var out strings.Builder
	code := processInput(cfg, namedInput{name: "/nonexistent/file.xml"}, nil, nil, &out)
	require.Equal(t, exitReadFile, code)
}

func TestMalformedXML(t *testing.T) {
	_, code := runWithStdin(t, `<root><unclosed>`)
	require.NotEqual(t, 0, code)
}

// --- Parser flags ---

func TestNoBlanks(t *testing.T) {
	out, code := runWithStdin(t, `<a>   <b/>   </a>`, "--noblanks")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<a><b/></a>`)
}

func TestNoCDATA(t *testing.T) {
	out, code := runWithStdin(t, `<a><![CDATA[hello]]></a>`, "--nocdata")
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

	out, code := runWithFile(t, f, "--noent")
	require.Equal(t, 0, code)
	require.Contains(t, out, "hello world")
}

func TestRecover(t *testing.T) {
	out, code := runWithStdin(t, `<root><a>text</a><b>`, "--recover")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<root>`)
}

// --- Output control ---

func TestNoOut(t *testing.T) {
	out, code := runWithStdin(t, `<root/>`, "--noout")
	require.Equal(t, 0, code)
	require.Empty(t, out)
}

func TestFormat(t *testing.T) {
	out, code := runWithStdin(t, `<a><b><c>text</c></b></a>`, "--format")
	require.Equal(t, 0, code)
	require.Contains(t, out, "  <b>")
	require.Contains(t, out, "    <c>text</c>")
}

func TestFormatPreservesTextContent(t *testing.T) {
	out, code := runWithStdin(t, `<a><b>hello</b></a>`, "--format")
	require.Equal(t, 0, code)
	require.Contains(t, out, "<b>hello</b>")
}

func TestPretty0(t *testing.T) {
	out, code := runWithStdin(t, `<a><b><c/></b></a>`, "--pretty", "0")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<a><b><c/></b></a>`)
}

func TestPretty1(t *testing.T) {
	out, code := runWithStdin(t, `<a><b><c/></b></a>`, "--pretty", "1")
	require.Equal(t, 0, code)
	require.Contains(t, out, "  <b>")
}

func TestOutputFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.xml")
	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><root/>`)

	cfg, files := parseArgs([]string{"--output", outFile, xmlFile})
	require.NotNil(t, cfg)

	f, err := os.Create(cfg.outputFile)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	code := processInput(cfg, namedInput{name: files[0]}, nil, nil, f)
	require.Equal(t, 0, code)
	require.NoError(t, f.Close())

	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	require.Contains(t, string(data), `<root/>`)
}

// --- C14N ---

func TestC14N(t *testing.T) {
	out, code := runWithStdin(t, `<b a="1" c="2"/>`, "--c14n")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<b a="1" c="2"></b>`)
}

func TestC14NAttributeOrder(t *testing.T) {
	out, code := runWithStdin(t, `<e z="1" a="2"/>`, "--c14n")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<e a="2" z="1"></e>`)
}

func TestC14NComment(t *testing.T) {
	out, code := runWithStdin(t, `<a><!-- comment --></a>`, "--c14n")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<!-- comment -->`)
}

func TestC14N11(t *testing.T) {
	out, code := runWithStdin(t, `<a/>`, "--c14n11")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<a></a>`)
}

func TestExcC14N(t *testing.T) {
	out, code := runWithStdin(t, `<a xmlns:n="http://example.com"><b/></a>`, "--exc-c14n")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<b></b>`)
}

func TestC14NFile(t *testing.T) {
	dir := t.TempDir()
	f := writeFile(t, dir, "test.xml", `<?xml version="1.0"?><root z="2" a="1"/>`)

	out, code := runWithFile(t, f, "--c14n")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<root a="1" z="2"></root>`)
}

// --- XPath ---

func TestXPathNodeSet(t *testing.T) {
	out, code := runWithStdin(t, `<a><b>1</b><b>2</b></a>`, "--xpath", "//b")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<b>1</b>`)
	require.Contains(t, out, `<b>2</b>`)
}

func TestXPathCount(t *testing.T) {
	out, code := runWithStdin(t, `<a><b/><b/><b/></a>`, "--xpath", "count(//b)")
	require.Equal(t, 0, code)
	require.Contains(t, out, "3")
}

func TestXPathString(t *testing.T) {
	out, code := runWithStdin(t, `<a><b>hello</b></a>`, "--xpath", "string(//b)")
	require.Equal(t, 0, code)
	require.Contains(t, out, "hello")
}

func TestXPathBoolean(t *testing.T) {
	out, code := runWithStdin(t, `<a><b/></a>`, "--xpath", "boolean(//b)")
	require.Equal(t, 0, code)
	require.Contains(t, out, "true")
}

func TestXPathBooleanFalse(t *testing.T) {
	out, code := runWithStdin(t, `<a/>`, "--xpath", "boolean(//b)")
	require.Equal(t, 0, code)
	require.Contains(t, out, "false")
}

func TestXPathImpliesNoout(t *testing.T) {
	out, code := runWithStdin(t, `<a><b/></a>`, "--xpath", "count(//b)")
	require.Equal(t, 0, code)
	require.NotContains(t, out, `<?xml`)
}

func TestXPathInvalidExpression(t *testing.T) {
	_, code := runWithStdin(t, `<a/>`, "--xpath", "///invalid[[[")
	require.Equal(t, exitXPath, code)
}

func TestXPathAttribute(t *testing.T) {
	out, code := runWithStdin(t, `<a foo="bar"/>`, "--xpath", "/a/@foo")
	require.Equal(t, 0, code)
	require.Contains(t, out, "bar")
}

func TestXPathWithFile(t *testing.T) {
	dir := t.TempDir()
	f := writeFile(t, dir, "test.xml", `<?xml version="1.0"?><a><b>1</b><b>2</b></a>`)

	out, code := runWithFile(t, f, "--xpath", "//b")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<b>1</b>`)
	require.Contains(t, out, `<b>2</b>`)
}

// --- XInclude ---

func TestXInclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "included.xml", `<chapter>Hello</chapter>`)
	mainXML := `<?xml version="1.0"?>
<book xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="included.xml"/>
</book>`
	f := writeFile(t, dir, "main.xml", mainXML)

	out, code := runWithFile(t, f, "--xinclude")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<chapter`)
	require.Contains(t, out, `Hello</chapter>`)
}

func TestXIncludeNoXIncludeNode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "inc.xml", `<p>text</p>`)
	mainXML := `<?xml version="1.0"?>
<doc xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="inc.xml"/>
</doc>`
	f := writeFile(t, dir, "main.xml", mainXML)

	out, code := runWithFile(t, f, "--xinclude", "--noxincludenode")
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

	out, code := runWithFile(t, f, "--xinclude", "--noxincludenode")
	require.Equal(t, 0, code)
	require.Contains(t, out, "Hello, World!")
}

// --- Schema validation ---

func TestSchemaValid(t *testing.T) {
	dir := t.TempDir()
	xsdContent := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`
	xsdFile := writeFile(t, dir, "test.xsd", xsdContent)
	xmlFile := writeFile(t, dir, "test.xml", `<?xml version="1.0"?><root>hello</root>`)

	cfg, files := parseArgs([]string{"--schema", xsdFile, "--noout", xmlFile})
	require.NotNil(t, cfg)

	schema, err := compileSchema(cfg)
	require.NoError(t, err)

	var out strings.Builder
	code := processInput(cfg, namedInput{name: files[0]}, nil, schema, &out)
	require.Equal(t, exitOK, code)
	require.Empty(t, out.String())
}

func TestSchemaInvalid(t *testing.T) {
	dir := t.TempDir()
	xsdContent := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:integer"/>
</xs:schema>`
	xsdFile := writeFile(t, dir, "test.xsd", xsdContent)
	xmlFile := writeFile(t, dir, "test.xml", `<?xml version="1.0"?><root>not-an-integer</root>`)

	cfg, files := parseArgs([]string{"--schema", xsdFile, "--noout", xmlFile})
	require.NotNil(t, cfg)

	schema, err := compileSchema(cfg)
	require.NoError(t, err)

	var out strings.Builder
	code := processInput(cfg, namedInput{name: files[0]}, nil, schema, &out)
	require.Equal(t, exitValidation, code)
}

// --- Behavioral flags ---

func TestDropDTD(t *testing.T) {
	dir := t.TempDir()
	xml := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (#PCDATA)>
]>
<doc>hello</doc>`
	f := writeFile(t, dir, "test.xml", xml)

	out, code := runWithFile(t, f, "--dropdtd")
	require.Equal(t, 0, code)
	require.NotContains(t, out, `<!DOCTYPE`)
	require.Contains(t, out, `<doc>hello</doc>`)
}

func TestRepeat(t *testing.T) {
	cfg, _ := parseArgs([]string{"--repeat", "3"})
	require.NotNil(t, cfg)
	require.Equal(t, 3, cfg.repeat)
}

// --- NsClean ---

func TestNsClean(t *testing.T) {
	xml := `<?xml version="1.0" encoding="US-ASCII"?>
<a xmlns:unused="http://example.com/unused">
  <b xmlns:unused="http://example.com/unused"/>
</a>`
	out, code := runWithStdin(t, xml, "--nsclean")
	require.Equal(t, 0, code)
	count := strings.Count(out, `xmlns:unused`)
	require.Equal(t, 1, count, "redundant ns should be cleaned")
}

// --- Combination flags ---

func TestFormatWithNoBlanks(t *testing.T) {
	out, code := runWithStdin(t, `<a>   <b>   <c/> </b>   </a>`, "--noblanks", "--format")
	require.Equal(t, 0, code)
	require.Contains(t, out, "  <b>")
	require.Contains(t, out, "    <c/>")
}

func TestC14NWithOutput(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.xml")
	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><b a="1"/>`)

	cfg, files := parseArgs([]string{"--c14n", "--output", outFile, xmlFile})
	require.NotNil(t, cfg)

	f, err := os.Create(cfg.outputFile)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	code := processInput(cfg, namedInput{name: files[0]}, nil, nil, f)
	require.Equal(t, 0, code)
	require.NoError(t, f.Close())

	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	require.Contains(t, string(data), `<b a="1"></b>`)
}

func TestXPathWithNoOut(t *testing.T) {
	// --xpath implies --noout, verify no XML decl but xpath result present
	out, code := runWithStdin(t, `<a><b>42</b></a>`, "--xpath", "string(//b)")
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

	cfg, files := parseArgs([]string{"--schema", xsdFile, "--noout", "--quiet", xmlFile})
	require.NotNil(t, cfg)
	require.True(t, cfg.quiet)

	schema, err := compileSchema(cfg)
	require.NoError(t, err)

	var out strings.Builder
	code := processInput(cfg, namedInput{name: files[0]}, nil, schema, &out)
	require.Equal(t, exitOK, code)
}

// --- Catalog ---

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

	cfg, files := parseArgs([]string{"--catalogs", "--loaddtd", "--dtdattr", "--noout", xmlFile})
	require.NotNil(t, cfg)

	// Simulate XML_CATALOG_FILES env
	t.Setenv("XML_CATALOG_FILES", catFile)

	// Load catalog the same way run() does
	cat, err := loadCatalogFromEnv()
	require.NoError(t, err)
	require.NotNil(t, cat)

	var out strings.Builder
	code := processInput(cfg, namedInput{name: files[0]}, cat, nil, &out)
	require.Equal(t, exitOK, code)
}

// --- Format edge cases ---

func TestFormatNestedElements(t *testing.T) {
	out, code := runWithStdin(t, `<a><b><c><d/></c></b></a>`, "--format")
	require.Equal(t, 0, code)
	lines := strings.Split(out, "\n")
	// Find lines with indentation
	var indented []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<") && trimmed != `<?xml version="1.0"?>` {
			indented = append(indented, line)
		}
	}
	require.NotEmpty(t, indented)
}

func TestFormatMixedContent(t *testing.T) {
	// Element with both text and element children
	out, code := runWithStdin(t, `<a>text<b/></a>`, "--format")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<a>`)
	require.Contains(t, out, `<b/>`)
}

func TestFormatEmptyElement(t *testing.T) {
	out, code := runWithStdin(t, `<a><b/></a>`, "--format")
	require.Equal(t, 0, code)
	require.Contains(t, out, `  <b/>`)
}
