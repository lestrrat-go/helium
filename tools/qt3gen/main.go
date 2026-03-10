// Command qt3gen parses the W3C QT3 (XPath/XQuery Test Suite 3) catalog
// and generates a table-driven Go test file for the xpath3 package.
//
// Usage:
//
//	go run ./tools/qt3gen
//
// Prerequisites:
//
//	bash testdata/qt3ts/fetch.sh   (clone the QT3 test suite first)
//
// Output:
//
//	xpath3/qt3_generated_test.go   (table-driven tests)
//	testdata/qt3ts/docs/           (context documents needed by tests)
package main

import (
	"encoding/xml"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html/charset"
)

// ──────────────────────────────────────────────────────────────────────
// QT3 catalog XML types
// ──────────────────────────────────────────────────────────────────────

const qt3NS = "http://www.w3.org/2010/09/qt-fots-catalog"

type catalog struct {
	Environments []environment `xml:"environment"`
	TestSets     []testSetRef  `xml:"test-set"`
}

type testSetRef struct {
	Name string `xml:"name,attr"`
	File string `xml:"file,attr"`
}

type testSetFile struct {
	Name         string        `xml:"name,attr"`
	Environments []environment `xml:"environment"`
	TestCases    []testCase    `xml:"test-case"`
}

type environment struct {
	Name       string      `xml:"name,attr"`
	Ref        string      `xml:"ref,attr"`
	Sources    []source    `xml:"source"`
	Namespaces []namespace `xml:"namespace"`
	Params     []param     `xml:"param"`
}

type source struct {
	Role       string `xml:"role,attr"`
	File       string `xml:"file,attr"`
	Validation string `xml:"validation,attr"`
}

type namespace struct {
	Prefix string `xml:"prefix,attr"`
	URI    string `xml:"uri,attr"`
}

type param struct {
	Name     string `xml:"name,attr"`
	Select   string `xml:"select,attr"`
	As       string `xml:"as,attr"`
	Declared string `xml:"declared,attr"`
}

type testCase struct {
	Name         string       `xml:"name,attr"`
	Test         string       `xml:"test"`
	Environment  *environment `xml:"environment"`
	Dependencies []dependency `xml:"dependency"`
	Result       resultSpec   `xml:",any"`
}

type dependency struct {
	Type      string `xml:"type,attr"`
	Value     string `xml:"value,attr"`
	Satisfied string `xml:"satisfied,attr"`
}

type resultSpec struct {
	XMLName xml.Name
	Inner   []byte `xml:",innerxml"`
}

type assertion struct {
	Type     string
	Value    string
	Children []assertion
}

// ──────────────────────────────────────────────────────────────────────
// Main
// ──────────────────────────────────────────────────────────────────────

func main() {
	repoRoot := findRepoRoot()
	sourceDir := filepath.Join(repoRoot, "testdata", "qt3ts", "source")
	outputFile := filepath.Join(repoRoot, "xpath3", "qt3_generated_test.go")
	docsDir := filepath.Join(repoRoot, "testdata", "qt3ts", "testdata")

	if _, err := os.Stat(filepath.Join(sourceDir, "catalog.xml")); os.IsNotExist(err) {
		log.Fatal("QT3 source not found. Run: bash testdata/qt3ts/fetch.sh")
	}

	cat := parseCatalog(filepath.Join(sourceDir, "catalog.xml"))

	globalEnvs := make(map[string]*environment)
	for i := range cat.Environments {
		globalEnvs[cat.Environments[i].Name] = &cat.Environments[i]
	}

	var allTests []generatedTest
	docFiles := make(map[string]bool)

	for _, tsRef := range cat.TestSets {
		tsFile := filepath.Join(sourceDir, tsRef.File)
		ts := parseTestSet(tsFile)
		tsDir := filepath.Dir(tsRef.File)

		localEnvs := make(map[string]*environment)
		for i := range ts.Environments {
			localEnvs[ts.Environments[i].Name] = &ts.Environments[i]
		}

		for _, tc := range ts.TestCases {
			if !isXPathApplicable(tc.Dependencies) {
				continue
			}

			skipReason := getSkipReason(tc.Dependencies)
			env, envIsGlobal := resolveEnvironment(tc.Environment, localEnvs, globalEnvs)

			if env != nil {
				if envSkip := checkEnvironmentSupport(env); envSkip != "" {
					if skipReason == "" {
						skipReason = envSkip
					}
				}
			}

			var contextDocPath string
			if env != nil {
				for _, src := range env.Sources {
					if src.Role == "." && src.File != "" {
						if envIsGlobal {
							contextDocPath = src.File
						} else {
							contextDocPath = filepath.Join(tsDir, src.File)
						}
						docFiles[contextDocPath] = true
					}
				}
			}

			assertions := parseResultAssertions(tc)

			allTests = append(allTests, generatedTest{
				SetName:        tsRef.Name,
				CaseName:       tc.Name,
				XPath:          strings.TrimSpace(tc.Test),
				ContextDocPath: contextDocPath,
				Namespaces:     collectNamespaces(env),
				Assertions:     assertions,
				SkipReason:     skipReason,
			})
		}
	}

	// Copy context documents (preserving directory structure)
	copied := 0
	for docPath := range docFiles {
		srcFull := filepath.Join(sourceDir, docPath)
		dstFull := filepath.Join(docsDir, docPath)
		if err := copyFile(srcFull, dstFull); err != nil {
			log.Printf("warning: copying %s: %v", docPath, err)
			continue
		}
		copied++
	}

	// Generate Go test file
	code := generateTestFile(allTests)
	formatted, err := format.Source([]byte(code))
	if err != nil {
		if writeErr := os.WriteFile(outputFile, []byte(code), 0o644); writeErr != nil {
			log.Fatalf("writing unformatted output: %v", writeErr)
		}
		log.Fatalf("gofmt failed (raw file written): %v", err)
	}
	if err := os.WriteFile(outputFile, formatted, 0o644); err != nil {
		log.Fatalf("writing %s: %v", outputFile, err)
	}

	fmt.Printf("Generated %d XPath tests in %s\n", len(allTests), outputFile)
	fmt.Printf("Copied %d context documents to %s\n", copied, docsDir)

	skipped := 0
	for _, t := range allTests {
		if t.SkipReason != "" {
			skipped++
		}
	}
	fmt.Printf("  %d will run, %d will skip\n", len(allTests)-skipped, skipped)
}

// ──────────────────────────────────────────────────────────────────────
// Parsing
// ──────────────────────────────────────────────────────────────────────

func parseCatalog(path string) *catalog {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("opening catalog: %v", err)
	}
	defer f.Close()
	var c catalog
	dec := xml.NewDecoder(f)
	dec.CharsetReader = charset.NewReaderLabel
	if err := dec.Decode(&c); err != nil {
		log.Fatalf("parsing catalog: %v", err)
	}
	return &c
}

func parseTestSet(path string) *testSetFile {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("opening test set %s: %v", path, err)
	}
	defer f.Close()
	var ts testSetFile
	dec := xml.NewDecoder(f)
	dec.CharsetReader = charset.NewReaderLabel
	if err := dec.Decode(&ts); err != nil {
		log.Fatalf("parsing test set %s: %v", path, err)
	}
	return &ts
}

// ──────────────────────────────────────────────────────────────────────
// Spec filtering
// ──────────────────────────────────────────────────────────────────────

func isXPathApplicable(deps []dependency) bool {
	hasSpecDep := false
	for _, d := range deps {
		if d.Type != "spec" {
			continue
		}
		hasSpecDep = true
		for _, v := range strings.Fields(d.Value) {
			if strings.HasPrefix(v, "XP") {
				if d.Satisfied == "false" {
					continue
				}
				return true
			}
		}
	}
	return !hasSpecDep
}

func getSkipReason(deps []dependency) string {
	for _, d := range deps {
		if d.Type == "feature" && d.Satisfied != "false" {
			switch d.Value {
			case "schemaImport", "schemaValidation", "schemaAware":
				return "requires XML Schema support"
			case "serialization":
				return "requires serialization"
			case "namespace-axis":
				return "requires namespace axis"
			case "moduleImport":
				return "requires XQuery module import"
			case "collection-stability":
				return "requires collection stability"
			case "directory-as-collection-uri":
				return "requires directory as collection URI"
			case "fn-transform-XSLT", "fn-transform-XSLT30":
				return "requires XSLT transform"
			case "fn-format-integer-CLDR":
				return "requires CLDR format-integer"
			case "non_unicode_codepoint_collation":
				return "requires non-Unicode codepoint collation"
			case "non_empty_sequence_collection":
				return "requires non-empty sequence collection"
			}
		}
		if d.Type == "spec" {
			for _, v := range strings.Fields(d.Value) {
				if v == "XP20" && d.Satisfied != "false" {
					return "requires XPath 2.0 only behavior"
				}
			}
		}
		if d.Type == "xml-version" && d.Value == "1.1" {
			return "requires XML 1.1"
		}
	}
	return ""
}

// ──────────────────────────────────────────────────────────────────────
// Environment resolution
// ──────────────────────────────────────────────────────────────────────

func resolveEnvironment(tcEnv *environment, local, global map[string]*environment) (*environment, bool) {
	if tcEnv == nil {
		return nil, false
	}
	if tcEnv.Ref != "" {
		if e, ok := local[tcEnv.Ref]; ok {
			return e, false
		}
		if e, ok := global[tcEnv.Ref]; ok {
			return e, true
		}
		return nil, false
	}
	return tcEnv, false
}

func checkEnvironmentSupport(env *environment) string {
	for _, src := range env.Sources {
		if src.Validation == "strict" || src.Validation == "lax" {
			return "requires schema-validated source"
		}
		if src.Role != "." && src.Role != "" {
			return "requires variable-bound source documents"
		}
	}
	if len(env.Params) > 0 {
		return "requires external parameters"
	}
	return ""
}

func collectNamespaces(env *environment) map[string]string {
	if env == nil || len(env.Namespaces) == 0 {
		return nil
	}
	ns := make(map[string]string)
	for _, n := range env.Namespaces {
		ns[n.Prefix] = n.URI
	}
	return ns
}

// ──────────────────────────────────────────────────────────────────────
// Assertion parsing
// ──────────────────────────────────────────────────────────────────────

func parseResultAssertions(tc testCase) []assertion {
	resultXML := "<result xmlns=\"" + qt3NS + "\">" + string(tc.Result.Inner) + "</result>"
	return parseAssertionXML(resultXML)
}

type xmlResult struct {
	Children []xmlAssertion `xml:",any"`
}

type xmlAssertion struct {
	XMLName  xml.Name
	Code     string         `xml:"code,attr"`
	Value    string         `xml:",chardata"`
	Children []xmlAssertion `xml:",any"`
}

func parseAssertionXML(s string) []assertion {
	var result xmlResult
	if err := xml.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	var out []assertion
	for _, child := range result.Children {
		out = append(out, convertAssertion(child))
	}
	return out
}

func convertAssertion(xa xmlAssertion) assertion {
	a := assertion{
		Type:  xa.XMLName.Local,
		Value: strings.TrimSpace(xa.Value),
	}
	if xa.Code != "" {
		a.Value = xa.Code
	}
	for _, child := range xa.Children {
		a.Children = append(a.Children, convertAssertion(child))
	}
	return a
}

// ──────────────────────────────────────────────────────────────────────
// Code generation (table-driven)
// ──────────────────────────────────────────────────────────────────────

type generatedTest struct {
	SetName        string
	CaseName       string
	XPath          string
	ContextDocPath string
	Namespaces     map[string]string
	Assertions     []assertion
	SkipReason     string
}

func generateTestFile(tests []generatedTest) string {
	var b strings.Builder

	b.WriteString("// Code generated by tools/qt3gen; DO NOT EDIT.\n\n")
	b.WriteString("package xpath3_test\n\n")
	b.WriteString("import \"testing\"\n\n")

	// Group by test-set
	type setGroup struct {
		name  string
		tests []generatedTest
	}
	groups := make(map[string]*setGroup)
	var groupOrder []string

	for _, t := range tests {
		g, ok := groups[t.SetName]
		if !ok {
			g = &setGroup{name: t.SetName}
			groups[t.SetName] = g
			groupOrder = append(groupOrder, t.SetName)
		}
		g.tests = append(g.tests, t)
	}

	for _, setName := range groupOrder {
		g := groups[setName]
		funcName := "TestQT3_" + goIdentifier(setName)

		fmt.Fprintf(&b, "func %s(t *testing.T) {\n", funcName)
		fmt.Fprintf(&b, "\tt.Parallel()\n")
		fmt.Fprintf(&b, "\tqt3RunTests(t, []qt3Test{\n")

		for _, tc := range g.tests {
			b.WriteString("\t\t{")
			fmt.Fprintf(&b, "Name: %q, ", tc.CaseName)
			fmt.Fprintf(&b, "XPath: %s", goStringLiteral(tc.XPath))

			if tc.ContextDocPath != "" {
				fmt.Fprintf(&b, ", DocPath: %q", tc.ContextDocPath)
			}
			if len(tc.Namespaces) > 0 {
				b.WriteString(", Namespaces: map[string]string{")
				keys := sortedKeys(tc.Namespaces)
				for i, k := range keys {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q: %q", k, tc.Namespaces[k])
				}
				b.WriteString("}")
			}
			if tc.SkipReason != "" {
				fmt.Fprintf(&b, ", Skip: %q", tc.SkipReason)
			}
			if assertionsExpectError(tc.Assertions) {
				b.WriteString(", ExpectError: true")
			} else {
				assertExprs := emitAssertions(tc.Assertions)
				if len(assertExprs) > 0 {
					b.WriteString(", Assertions: []qt3Assertion{")
					b.WriteString(strings.Join(assertExprs, ", "))
					b.WriteString("}")
				}
			}

			b.WriteString("},\n")
		}

		fmt.Fprintf(&b, "\t})\n")
		fmt.Fprintf(&b, "}\n\n")
	}

	return b.String()
}

func assertionsExpectError(assertions []assertion) bool {
	for _, a := range assertions {
		if a.Type == "error" {
			return true
		}
		if a.Type == "any-of" {
			for _, child := range a.Children {
				if child.Type == "error" {
					return true
				}
			}
		}
	}
	return false
}

// emitAssertions returns Go expressions for assertion values.
func emitAssertions(assertions []assertion) []string {
	var out []string
	for _, a := range assertions {
		out = append(out, emitAssertion(a)...)
	}
	return out
}

// emitAssertion returns one or more Go expressions (all-of expands to multiple).
func emitAssertion(a assertion) []string {
	switch a.Type {
	case "assert-eq":
		return []string{fmt.Sprintf("qt3AssertEq(%s)", goStringLiteral(a.Value))}
	case "assert-string-value":
		return []string{fmt.Sprintf("qt3AssertStringValue(%s)", goStringLiteral(a.Value))}
	case "assert-true":
		return []string{"qt3AssertTrue()"}
	case "assert-false":
		return []string{"qt3AssertFalse()"}
	case "assert-empty":
		return []string{"qt3AssertEmpty()"}
	case "assert-count":
		n, _ := strconv.Atoi(a.Value)
		return []string{fmt.Sprintf("qt3AssertCount(%d)", n)}
	case "assert-type":
		return []string{fmt.Sprintf("qt3AssertType(%q)", a.Value)}
	case "assert-deep-eq":
		return []string{fmt.Sprintf("qt3AssertDeepEq(%s)", goStringLiteral(a.Value))}
	case "assert-xml", "assert-permutation", "assert-serialization-error", "assert-serialization":
		return []string{"qt3AssertSkip()"}
	case "all-of":
		return emitAssertions(a.Children)
	case "any-of":
		var checks []string
		for _, child := range a.Children {
			if child.Type == "error" {
				continue // handled by assertionsExpectError
			}
			checks = append(checks, emitCheck(child))
		}
		if len(checks) == 0 {
			return nil
		}
		return []string{fmt.Sprintf("qt3AnyOf(%s)", strings.Join(checks, ", "))}
	case "error":
		return nil // handled separately
	default:
		return []string{"qt3AssertSkip()"}
	}
}

// emitCheck returns a Go expression for a qt3Check value (used in any-of).
func emitCheck(a assertion) string {
	switch a.Type {
	case "assert-eq":
		return fmt.Sprintf("qt3CheckEq(%s)", goStringLiteral(a.Value))
	case "assert-string-value":
		return fmt.Sprintf("qt3CheckStringValue(%s)", goStringLiteral(a.Value))
	case "assert-true":
		return "qt3CheckTrue()"
	case "assert-false":
		return "qt3CheckFalse()"
	case "assert-empty":
		return "qt3CheckEmpty()"
	case "assert-count":
		n, _ := strconv.Atoi(a.Value)
		return fmt.Sprintf("qt3CheckCount(%d)", n)
	case "assert-type":
		return fmt.Sprintf("qt3CheckType(%q)", a.Value)
	case "assert-deep-eq":
		return fmt.Sprintf("qt3CheckDeepEq(%s)", goStringLiteral(a.Value))
	default:
		return "qt3CheckSkip()"
	}
}

// ──────────────────────────────────────────────────────────────────────
// Utilities
// ──────────────────────────────────────────────────────────────────────

var nonIdentRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func goIdentifier(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return nonIdentRE.ReplaceAllString(s, "_")
}

func goStringLiteral(s string) string {
	if strings.Contains(s, "\n") && !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	return fmt.Sprintf("%q", s)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			log.Fatal("could not find repo root (go.mod)")
		}
		dir = parent
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
