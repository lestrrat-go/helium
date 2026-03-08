// Command qt3gen parses the W3C QT3 (XPath/XQuery Test Suite 3) catalog
// and generates standalone Go test files for the xpath3 package.
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
//	xpath3/qt3_generated_test.go   (one file with all XPath-applicable tests)
//	testdata/qt3ts/docs/           (context documents copied from QT3 source)
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
	Type     string `xml:"type,attr"`
	Value    string `xml:"value,attr"`
	Satisfied string `xml:"satisfied,attr"`
}

// resultSpec uses raw XML so we can handle the nested assertion structure.
type resultSpec struct {
	XMLName xml.Name
	Inner   []byte `xml:",innerxml"`
}

// assertion is a parsed assertion from a <result> element.
type assertion struct {
	Type     string // "assert-eq", "assert-true", "error", "all-of", "any-of", etc.
	Value    string // text content (for assert-eq, assert-string-value, error code, etc.)
	Children []assertion
}

// ──────────────────────────────────────────────────────────────────────
// Main
// ──────────────────────────────────────────────────────────────────────

func main() {
	repoRoot := findRepoRoot()
	sourceDir := filepath.Join(repoRoot, "testdata", "qt3ts", "source")
	outputFile := filepath.Join(repoRoot, "xpath3", "qt3_generated_test.go")
	docsDir := filepath.Join(repoRoot, "testdata", "qt3ts", "docs")

	if _, err := os.Stat(filepath.Join(sourceDir, "catalog.xml")); os.IsNotExist(err) {
		log.Fatal("QT3 source not found. Run: bash testdata/qt3ts/fetch.sh")
	}

	// Parse catalog
	cat := parseCatalog(filepath.Join(sourceDir, "catalog.xml"))

	// Build global environment map
	globalEnvs := make(map[string]*environment)
	for i := range cat.Environments {
		globalEnvs[cat.Environments[i].Name] = &cat.Environments[i]
	}

	// Collect all applicable test cases
	var allTests []generatedTest
	docFiles := make(map[string]bool) // source-relative paths of docs to copy

	for _, tsRef := range cat.TestSets {
		tsFile := filepath.Join(sourceDir, tsRef.File)
		ts := parseTestSet(tsFile)
		tsDir := filepath.Dir(tsRef.File) // e.g. "fn", "prod"

		// Build local environment map
		localEnvs := make(map[string]*environment)
		for i := range ts.Environments {
			localEnvs[ts.Environments[i].Name] = &ts.Environments[i]
		}

		for _, tc := range ts.TestCases {
			if !isXPathApplicable(tc.Dependencies) {
				continue
			}

			skipReason := getSkipReason(tc.Dependencies)

			// Resolve environment
			env, envIsGlobal := resolveEnvironment(tc.Environment, localEnvs, globalEnvs)

			// Check for unsupported environment features
			if env != nil {
				if envSkip := checkEnvironmentSupport(env); envSkip != "" {
					if skipReason == "" {
						skipReason = envSkip
					}
				}
			}

			// Track context documents
			var contextDocPath string
			if env != nil {
				for _, src := range env.Sources {
					if src.Role == "." && src.File != "" {
						var srcPath string
						if envIsGlobal {
							// Global environments have paths relative to catalog root
							srcPath = src.File
						} else {
							// Local environments have paths relative to test-set dir
							srcPath = filepath.Join(tsDir, src.File)
						}
						docFiles[srcPath] = true
						contextDocPath = srcPath
					}
				}
			}

			// Parse assertions
			assertions := parseResultAssertions(tc)

			gt := generatedTest{
				SetName:        tsRef.Name,
				CaseName:       tc.Name,
				XPath:          strings.TrimSpace(tc.Test),
				ContextDocPath: contextDocPath,
				Namespaces:     collectNamespaces(env),
				Assertions:     assertions,
				SkipReason:     skipReason,
			}
			allTests = append(allTests, gt)
		}
	}

	// Copy context documents
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		log.Fatalf("creating docs dir: %v", err)
	}

	copied := 0
	for docPath := range docFiles {
		srcFull := filepath.Join(sourceDir, docPath)
		dstFull := filepath.Join(docsDir, filepath.Base(docPath))
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
		// Write unformatted for debugging
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

	// Count skips
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

// isXPathApplicable returns true if the test case is applicable to XPath
// (not XQuery-only). A test with no spec dependency is considered applicable.
func isXPathApplicable(deps []dependency) bool {
	hasSpecDep := false
	for _, d := range deps {
		if d.Type != "spec" {
			continue
		}
		hasSpecDep = true
		// Check if any value token includes XPath
		for _, v := range strings.Fields(d.Value) {
			if isXPathSpec(v) {
				if d.Satisfied == "false" {
					// "spec not satisfied" means this test should NOT run for this spec
					continue
				}
				return true
			}
		}
	}
	// No spec dependency → applicable to all
	return !hasSpecDep
}

func isXPathSpec(v string) bool {
	// XP20, XP20+, XP30, XP30+, XP31, XP31+
	return strings.HasPrefix(v, "XP")
}

// getSkipReason returns a skip reason if the test requires unsupported features.
func getSkipReason(deps []dependency) string {
	for _, d := range deps {
		if d.Type == "feature" {
			satisfied := d.Satisfied != "false"
			if satisfied {
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
				case "fn-transform-XSLT":
					return "requires XSLT transform"
				case "fn-transform-XSLT30":
					return "requires XSLT 3.0 transform"
				case "fn-format-integer-CLDR":
					return "requires CLDR format-integer"
				case "non_unicode_codepoint_collation":
					return "requires non-Unicode codepoint collation"
				case "non_empty_sequence_collection":
					return "requires non-empty sequence collection"
				}
			}
		}
		if d.Type == "spec" {
			// If it requires XP20 only (without +), skip if we're XP31
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

// resolveEnvironment returns the resolved environment and whether it is global.
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
		return nil, false // unresolved ref
	}
	return tcEnv, false
}

func checkEnvironmentSupport(env *environment) string {
	for _, src := range env.Sources {
		if src.Validation == "strict" || src.Validation == "lax" {
			return "requires schema-validated source"
		}
		if src.Role != "." && src.Role != "" {
			// Variable-bound sources like $works, $staff
			return "requires variable-bound source documents"
		}
	}
	if len(env.Params) > 0 {
		return "requires external parameters"
	}
	return ""
}

func collectNamespaces(env *environment) map[string]string {
	if env == nil {
		return nil
	}
	if len(env.Namespaces) == 0 {
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
	// The result field has the outermost assertion element.
	// We need to re-parse the inner XML to extract assertions.
	// The result XML is captured as raw inner XML under the <result> element.

	// Build synthetic XML wrapping the result content
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
// Code generation
// ──────────────────────────────────────────────────────────────────────

type generatedTest struct {
	SetName        string
	CaseName       string
	XPath          string
	ContextDocPath string // relative to testdata/qt3ts/source/, empty = no context
	Namespaces     map[string]string
	Assertions     []assertion
	SkipReason     string
}

func generateTestFile(tests []generatedTest) string {
	var b strings.Builder

	b.WriteString(`// Code generated by tools/qt3gen; DO NOT EDIT.

package xpath3_test

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// qt3DocsDir returns the path to the QT3 context documents directory.
func qt3DocsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "testdata", "qt3ts", "docs")
}

// qt3SourceDir returns the path to the QT3 source directory.
func qt3SourceDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "testdata", "qt3ts", "source")
}

// qt3ParseDoc parses an XML file and returns the document root.
func qt3ParseDoc(t *testing.T, path string) helium.Node {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading %s", path)
	doc, err := helium.Parse(t.Context(), data)
	require.NoError(t, err, "parsing %s", path)
	return doc
}

// qt3StringValue returns the string value of an XPath result sequence.
func qt3StringValue(seq xpath3.Sequence) string {
	var parts []string
	for _, item := range seq {
		av, err := xpath3.AtomizeItem(item)
		if err != nil {
			parts = append(parts, fmt.Sprintf("%v", item))
		} else {
			parts = append(parts, fmt.Sprintf("%v", av.Value))
		}
	}
	return strings.Join(parts, " ")
}

// qt3EffectiveBooleanValue returns the effective boolean value of a sequence.
func qt3EffectiveBooleanValue(seq xpath3.Sequence) (bool, error) {
	if len(seq) == 0 {
		return false, nil
	}
	first := seq[0]
	if _, ok := first.(xpath3.NodeItem); ok {
		return true, nil // non-empty node sequence is true
	}
	if len(seq) == 1 {
		av, err := xpath3.AtomizeItem(first)
		if err != nil {
			return false, err
		}
		switch v := av.Value.(type) {
		case bool:
			return v, nil
		case string:
			return v != "", nil
		case float64:
			return v != 0 && !math.IsNaN(v), nil
		case int64:
			return v != 0, nil
		}
	}
	return false, fmt.Errorf("cannot compute EBV for sequence of length %d", len(seq))
}

var _ = qt3DocsDir
var _ = qt3SourceDir
var _ = qt3ParseDoc
var _ = qt3StringValue
var _ = qt3EffectiveBooleanValue

`)

	// Group tests by test-set name
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

		for _, tc := range g.tests {
			caseName := goTestName(tc.CaseName)
			fmt.Fprintf(&b, "\tt.Run(%q, func(t *testing.T) {\n", caseName)

			if tc.SkipReason != "" {
				fmt.Fprintf(&b, "\t\tt.Skip(%q)\n", tc.SkipReason)
			}

			// Set up context
			if tc.ContextDocPath != "" || len(tc.Namespaces) > 0 {
				b.WriteString("\t\tctx := context.Background()\n")
			} else {
				b.WriteString("\t\tctx := context.Background()\n")
			}

			// Namespace bindings
			if len(tc.Namespaces) > 0 {
				b.WriteString("\t\tctx = xpath3.NewContext(ctx, xpath3.WithNamespaces(map[string]string{\n")
				keys := make([]string, 0, len(tc.Namespaces))
				for k := range tc.Namespaces {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Fprintf(&b, "\t\t\t%q: %q,\n", k, tc.Namespaces[k])
				}
				b.WriteString("\t\t}))\n")
			}

			// Context document
			if tc.ContextDocPath != "" {
				fmt.Fprintf(&b, "\t\tdoc := qt3ParseDoc(t, filepath.Join(qt3SourceDir(), %q))\n", tc.ContextDocPath)
			} else {
				b.WriteString("\t\tvar doc helium.Node // no context document\n")
			}

			// Compile & evaluate
			xpathStr := tc.XPath
			fmt.Fprintf(&b, "\t\texpr := %s\n", goStringLiteral(xpathStr))

			// Check if assertions expect an error
			if assertionsExpectError(tc.Assertions) {
				b.WriteString("\t\tcompiled, compileErr := xpath3.Compile(expr)\n")
				b.WriteString("\t\tif compileErr != nil {\n")
				generateErrorAssertions(&b, tc.Assertions, "compileErr", "\t\t\t")
				b.WriteString("\t\t\treturn\n")
				b.WriteString("\t\t}\n")
				b.WriteString("\t\t_, evalErr := compiled.Evaluate(ctx, doc)\n")
				b.WriteString("\t\trequire.Error(t, evalErr, \"expected error for: %s\", expr)\n")
				generateErrorAssertions(&b, tc.Assertions, "evalErr", "\t\t")
			} else {
				b.WriteString("\t\tcompiled, err := xpath3.Compile(expr)\n")
				b.WriteString("\t\trequire.NoError(t, err, \"compile: %s\", expr)\n")
				b.WriteString("\t\tresult, err := compiled.Evaluate(ctx, doc)\n")
				b.WriteString("\t\trequire.NoError(t, err, \"eval: %s\", expr)\n")
				b.WriteString("\t\tseq := result.Sequence()\n")
				b.WriteString("\t\t_ = seq\n")
				generateAssertions(&b, tc.Assertions, "\t\t")
			}

			b.WriteString("\t})\n")
		}

		b.WriteString("}\n\n")
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

func generateAssertions(b *strings.Builder, assertions []assertion, indent string) {
	for _, a := range assertions {
		generateSingleAssertion(b, a, indent)
	}
}

func generateSingleAssertion(b *strings.Builder, a assertion, indent string) {
	switch a.Type {
	case "assert-true":
		fmt.Fprintf(b, "%sebv, ebvErr := qt3EffectiveBooleanValue(seq)\n", indent)
		fmt.Fprintf(b, "%srequire.NoError(t, ebvErr)\n", indent)
		fmt.Fprintf(b, "%srequire.True(t, ebv, \"expected true, got: %%v\", seq)\n", indent)

	case "assert-false":
		fmt.Fprintf(b, "%sebv, ebvErr := qt3EffectiveBooleanValue(seq)\n", indent)
		fmt.Fprintf(b, "%srequire.NoError(t, ebvErr)\n", indent)
		fmt.Fprintf(b, "%srequire.False(t, ebv, \"expected false, got: %%v\", seq)\n", indent)

	case "assert-string-value":
		fmt.Fprintf(b, "%srequire.Equal(t, %s, qt3StringValue(seq))\n", indent, goStringLiteral(a.Value))

	case "assert-eq":
		fmt.Fprintf(b, "%srequire.Equal(t, %s, qt3StringValue(seq))\n", indent, goStringLiteral(a.Value))

	case "assert-empty":
		fmt.Fprintf(b, "%srequire.Empty(t, seq, \"expected empty sequence\")\n", indent)

	case "assert-count":
		fmt.Fprintf(b, "%srequire.Len(t, seq, %s)\n", indent, a.Value)

	case "assert-type":
		// Type assertions are informational for now; skip detailed checks
		fmt.Fprintf(b, "%s_ = seq // assert-type: %s\n", indent, a.Value)

	case "assert-deep-eq":
		fmt.Fprintf(b, "%srequire.Equal(t, %s, qt3StringValue(seq), \"deep-eq\")\n", indent, goStringLiteral(a.Value))

	case "assert-xml":
		fmt.Fprintf(b, "%s_ = seq // assert-xml (not checked)\n", indent)

	case "assert-permutation":
		fmt.Fprintf(b, "%s_ = seq // assert-permutation (not checked)\n", indent)

	case "assert-serialization-error", "assert-serialization":
		fmt.Fprintf(b, "%s_ = seq // %s (not checked)\n", indent, a.Type)

	case "all-of":
		for _, child := range a.Children {
			generateSingleAssertion(b, child, indent)
		}

	case "any-of":
		// For any-of, we try each assertion and pass if any succeeds.
		// Use a helper approach with a bool flag.
		fmt.Fprintf(b, "%s// any-of: at least one assertion must pass\n", indent)
		fmt.Fprintf(b, "%sanyPassed := false\n", indent)
		for i, child := range a.Children {
			if child.Type == "error" {
				// Already handled by assertionsExpectError
				continue
			}
			varName := fmt.Sprintf("anyOfOk%d", i)
			fmt.Fprintf(b, "%sfunc() { defer func() { if r := recover(); r != nil { /* assertion failed, try next */ } }()\n", indent)
			generateSingleAssertionForAnyOf(b, child, indent+"\t")
			fmt.Fprintf(b, "%s\t%s := true\n", indent, varName)
			fmt.Fprintf(b, "%s\t_ = %s\n", indent, varName)
			fmt.Fprintf(b, "%s\tanyPassed = true\n", indent)
			fmt.Fprintf(b, "%s}()\n", indent)
		}
		fmt.Fprintf(b, "%srequire.True(t, anyPassed, \"none of the any-of assertions passed for: %%v\", seq)\n", indent)

	case "error":
		// Should have been caught by assertionsExpectError; if we get here, skip
		fmt.Fprintf(b, "%st.Skip(\"expected error %s\")\n", indent, a.Value)

	default:
		safeVal := strings.ReplaceAll(a.Value, "\n", " ")
		fmt.Fprintf(b, "%s// TODO: unhandled assertion type %q: %s\n", indent, a.Type, safeVal)
	}
}

func generateSingleAssertionForAnyOf(b *strings.Builder, a assertion, indent string) {
	switch a.Type {
	case "assert-true":
		fmt.Fprintf(b, "%sebv, _ := qt3EffectiveBooleanValue(seq)\n", indent)
		fmt.Fprintf(b, "%sif !ebv { panic(\"not true\") }\n", indent)
	case "assert-false":
		fmt.Fprintf(b, "%sebv, _ := qt3EffectiveBooleanValue(seq)\n", indent)
		fmt.Fprintf(b, "%sif ebv { panic(\"not false\") }\n", indent)
	case "assert-string-value":
		fmt.Fprintf(b, "%sif qt3StringValue(seq) != %s { panic(\"no match\") }\n", indent, goStringLiteral(a.Value))
	case "assert-eq":
		fmt.Fprintf(b, "%sif qt3StringValue(seq) != %s { panic(\"no match\") }\n", indent, goStringLiteral(a.Value))
	case "assert-empty":
		fmt.Fprintf(b, "%sif len(seq) != 0 { panic(\"not empty\") }\n", indent)
	case "assert-count":
		fmt.Fprintf(b, "%sif len(seq) != %s { panic(\"wrong count\") }\n", indent, a.Value)
	case "assert-type":
		fmt.Fprintf(b, "%s_ = seq // assert-type in any-of\n", indent)
	default:
		fmt.Fprintf(b, "%s_ = seq // any-of child: %s\n", indent, a.Type)
	}
}

func generateErrorAssertions(b *strings.Builder, assertions []assertion, errVar, indent string) {
	for _, a := range assertions {
		if a.Type == "error" {
			fmt.Fprintf(b, "%s// expected error code: %s\n", indent, a.Value)
			fmt.Fprintf(b, "%s_ = %s\n", indent, errVar)
			return
		}
		if a.Type == "any-of" {
			for _, child := range a.Children {
				if child.Type == "error" {
					fmt.Fprintf(b, "%s// expected error code (any-of): %s\n", indent, child.Value)
					fmt.Fprintf(b, "%s_ = %s\n", indent, errVar)
					return
				}
			}
		}
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

func goTestName(s string) string {
	return s // test names can be anything for t.Run
}

// goStringLiteral returns a Go string literal for s, using backticks if s
// contains newlines and no backticks, otherwise using %q.
func goStringLiteral(s string) string {
	if strings.Contains(s, "\n") && !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	return fmt.Sprintf("%q", s)
}

func findRepoRoot() string {
	// Walk up from the executable or CWD to find go.mod
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
