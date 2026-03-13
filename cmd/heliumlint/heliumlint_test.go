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
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600)) //nolint:gosec // test helper writing temp files
	return p
}

// runWithFile processes a single XML file through heliumlint with the given args.
// Returns stdout content and exit code.
func runWithFile(t *testing.T, xmlPath string, args ...string) (string, int) {
	t.Helper()

	allArgs := make([]string, len(args)+1)
	copy(allArgs, args)
	allArgs[len(args)] = xmlPath
	cfg, files := parseArgs(allArgs)
	require.NotNil(t, cfg, "parseArgs returned nil config for args: %v", allArgs)
	require.NotEmpty(t, files, "no files collected from args: %v", allArgs)

	if cfg.pretty >= 1 {
		cfg.format = true
	}

	var out strings.Builder
	input := namedInput{name: files[0]}
	code := processInput(t.Context(), cfg, input, nil, nil, &out)
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

func TestParseArgsSimpleCases(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantFiles  []string
		check      func(*testing.T, *config)
		filesUnset bool
	}{
		{
			name: "version",
			args: []string{"--version"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.True(t, cfg.version)
			},
		},
		{
			name: "single dash",
			args: []string{"-noblanks"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.True(t, cfg.parseOptions.IsSet(helium.ParseNoBlanks))
			},
		},
		{
			name: "dtdattr",
			args: []string{"--dtdattr"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.True(t, cfg.parseOptions.IsSet(helium.ParseDTDAttr))
				require.True(t, cfg.parseOptions.IsSet(helium.ParseDTDLoad))
			},
		},
		{
			name: "valid",
			args: []string{"--valid"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.True(t, cfg.parseOptions.IsSet(helium.ParseDTDValid))
				require.True(t, cfg.parseOptions.IsSet(helium.ParseDTDLoad))
			},
		},
		{
			name: "xinclude",
			args: []string{"--xinclude"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.True(t, cfg.doXInclude)
				require.True(t, cfg.parseOptions.IsSet(helium.ParseXInclude))
			},
		},
		{
			name: "xpath implies noout",
			args: []string{"--xpath", "//a"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.Equal(t, "//a", cfg.xpathExpr)
				require.True(t, cfg.noout)
			},
		},
		{
			name: "pretty",
			args: []string{"--pretty", "2"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.Equal(t, 2, cfg.pretty)
			},
		},
		{
			name: "repeat",
			args: []string{"--repeat", "5"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.Equal(t, 5, cfg.repeat)
			},
		},
		{
			name:      "schema",
			args:      []string{"--schema", "test.xsd", "test.xml"},
			wantFiles: []string{"test.xml"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.Equal(t, "test.xsd", cfg.schemaFile)
			},
		},
		{
			name:      "output",
			args:      []string{"--output", "out.xml", "in.xml"},
			wantFiles: []string{"in.xml"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.Equal(t, "out.xml", cfg.outputFile)
			},
		},
		{
			name: "encode",
			args: []string{"--encode", "UTF-8"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.Equal(t, "UTF-8", cfg.encode)
			},
		},
		{
			name: "path",
			args: []string{"--path", "/usr/share/dtd"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.Equal(t, "/usr/share/dtd", cfg.pathDirs)
			},
		},
		{
			name: "catalogs",
			args: []string{"--catalogs"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.True(t, cfg.catalogs)
			},
		},
		{
			name: "nocatalogs",
			args: []string{"--nocatalogs"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.True(t, cfg.noCatalogs)
			},
		},
		{
			name: "boolean flags",
			args: []string{"--noout", "--format", "--quiet", "--timing", "--dropdtd", "--nowarning"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.True(t, cfg.noout)
				require.True(t, cfg.format)
				require.True(t, cfg.quiet)
				require.True(t, cfg.timing)
				require.True(t, cfg.dropdtd)
				require.True(t, cfg.parseOptions.IsSet(helium.ParseNoWarning))
			},
		},
		{
			name:      "files collected",
			args:      []string{"--noblanks", "a.xml", "b.xml"},
			wantFiles: []string{"a.xml", "b.xml"},
			check: func(t *testing.T, cfg *config) {
				t.Helper()
				require.NotNil(t, cfg)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, files := parseArgs(tc.args)
			require.NotNil(t, cfg)
			if tc.wantFiles != nil {
				require.Equal(t, tc.wantFiles, files)
			} else {
				require.Empty(t, files)
			}
			tc.check(t, cfg)
		})
	}
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

func TestParseArgsInvalidCases(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "unrecognized", args: []string{"--nonexistent-flag"}},
		{name: "pretty invalid", args: []string{"--pretty", "xyz"}},
		{name: "repeat invalid", args: []string{"--repeat", "abc"}},
		{name: "repeat zero", args: []string{"--repeat", "0"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, _ := parseArgs(tc.args)
			require.Nil(t, cfg)
		})
	}
}

func TestParseArgsMissingValues(t *testing.T) {
	flags := []string{"--schema", "--xpath", "--output", "--encode", "--pretty", "--path", "--repeat"}
	for _, flag := range flags {
		cfg, _ := parseArgs([]string{flag})
		require.Nil(t, cfg, "flag %s without value should fail", flag)
	}
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
		processInput(t.Context(), cfg, namedInput{name: fn}, nil, nil, &out)
	}
	result := out.String()
	require.Contains(t, result, `<a/>`)
	require.Contains(t, result, `<b/>`)
}

func TestFileNotFound(t *testing.T) {
	cfg, _ := parseArgs([]string{})
	require.NotNil(t, cfg)

	var out strings.Builder
	code := processInput(t.Context(), cfg, namedInput{name: "/nonexistent/file.xml"}, nil, nil, &out)
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
			out, code := runWithStdin(t, tc.xml, tc.args...)
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

	cfg, files := parseArgs([]string{"--output", outFile, xmlFile})
	require.NotNil(t, cfg)

	f, err := os.Create(cfg.outputFile)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	code := processInput(t.Context(), cfg, namedInput{name: files[0]}, nil, nil, f)
	require.Equal(t, 0, code)
	require.NoError(t, f.Close())

	data, err := os.ReadFile(outFile) //nolint:gosec // reading test output file
	require.NoError(t, err)
	require.Contains(t, string(data), `<root/>`)
}

// --- C14N ---

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
			out, code := runWithStdin(t, tc.xml, tc.args...)
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

	out, code := runWithFile(t, f, "--c14n")
	require.Equal(t, 0, code)
	require.Contains(t, out, `<root a="1" z="2"></root>`)
}

// --- XPath ---

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
			wantCode: exitXPath,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, code := runWithStdin(t, tc.xml, "--xpath", tc.expr)
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

func TestXIncludeNoXIncludeMarker(t *testing.T) {
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

func TestSchemaValidation(t *testing.T) {
	tests := []struct {
		name      string
		xsdType   string
		xmlValue  string
		wantCode  int
		wantEmpty bool
	}{
		{
			name:      "valid",
			xsdType:   "xs:string",
			xmlValue:  "hello",
			wantCode:  exitOK,
			wantEmpty: true,
		},
		{
			name:     "invalid",
			xsdType:  "xs:integer",
			xmlValue: "not-an-integer",
			wantCode: exitValidation,
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

			cfg, files := parseArgs([]string{"--schema", xsdFile, "--noout", xmlFile})
			require.NotNil(t, cfg)

			schema, err := compileSchema(cfg)
			require.NoError(t, err)

			var out strings.Builder
			code := processInput(t.Context(), cfg, namedInput{name: files[0]}, nil, schema, &out)
			require.Equal(t, tc.wantCode, code)
			if tc.wantEmpty {
				require.Empty(t, out.String())
			}
		})
	}
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

	code := processInput(t.Context(), cfg, namedInput{name: files[0]}, nil, nil, f)
	require.Equal(t, 0, code)
	require.NoError(t, f.Close())

	data, err := os.ReadFile(outFile) //nolint:gosec // reading test output file
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
	code := processInput(t.Context(), cfg, namedInput{name: files[0]}, nil, schema, &out)
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
	cat, err := loadCatalogFromEnv(t.Context())
	require.NoError(t, err)
	require.NotNil(t, cat)

	var out strings.Builder
	code := processInput(t.Context(), cfg, namedInput{name: files[0]}, cat, nil, &out)
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
			out, code := runWithStdin(t, tc.xml, "--format")
			require.Equal(t, 0, code)
			for _, s := range tc.wantContains {
				require.Contains(t, out, s)
			}
		})
	}
}
