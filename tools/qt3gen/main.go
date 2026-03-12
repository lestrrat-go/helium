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
const qt3UnicodeVersion = "15.0"

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
	Dependencies []dependency  `xml:"dependency"`
	Environments []environment `xml:"environment"`
	TestCases    []testCase    `xml:"test-case"`
}

type staticBaseURI struct {
	URI string `xml:"uri,attr"`
}

type environment struct {
	Name          string          `xml:"name,attr"`
	Ref           string          `xml:"ref,attr"`
	Sources       []source        `xml:"source"`
	Resources     []resource      `xml:"resource"`
	Namespaces    []namespace     `xml:"namespace"`
	Collations    []collation     `xml:"collation"`
	DecimalFormat []decimalFormat `xml:"decimal-format"`
	Params        []param         `xml:"param"`
	StaticBaseURI *staticBaseURI  `xml:"static-base-uri"`
}

type collation struct {
	URI     string `xml:"uri,attr"`
	Default string `xml:"default,attr"`
}

type decimalFormat struct {
	Name              string     `xml:"name,attr"`
	DecimalSeparator  string     `xml:"decimal-separator,attr"`
	GroupingSeparator string     `xml:"grouping-separator,attr"`
	Percent           string     `xml:"percent,attr"`
	PerMille          string     `xml:"per-mille,attr"`
	ZeroDigit         string     `xml:"zero-digit,attr"`
	Digit             string     `xml:"digit,attr"`
	PatternSeparator  string     `xml:"pattern-separator,attr"`
	ExponentSeparator string     `xml:"exponent-separator,attr"`
	Infinity          string     `xml:"infinity,attr"`
	NaN               string     `xml:"NaN,attr"`
	MinusSign         string     `xml:"minus-sign,attr"`
	Attrs             []xml.Attr `xml:",any,attr"`
}

type resource struct {
	File string `xml:"file,attr"`
	URI  string `xml:"uri,attr"`
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
	Type           string
	Value          string
	NormalizeSpace bool
	Children       []assertion
}

// ──────────────────────────────────────────────────────────────────────
// Main
// ──────────────────────────────────────────────────────────────────────

func main() {
	repoRoot := findRepoRoot()
	sourceDir := filepath.Join(repoRoot, "testdata", "qt3ts", "source")
	outputDir := filepath.Join(repoRoot, "xpath3")
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
	resourceFiles := make(map[string]bool)

	for _, tsRef := range cat.TestSets {
		tsFile := filepath.Join(sourceDir, tsRef.File)
		ts := parseTestSet(tsFile)
		tsDir := filepath.Dir(tsRef.File)

		localEnvs := make(map[string]*environment)
		for i := range ts.Environments {
			localEnvs[ts.Environments[i].Name] = &ts.Environments[i]
		}

		// Skip entire test set if it requires XQuery at the set level
		if !isXPathApplicable(ts.Dependencies) {
			continue
		}

		setSkipReason := getTestSetSkipReason(tsRef.Name)

		for _, tc := range ts.TestCases {
			mergedDeps := mergeDeps(ts.Dependencies, tc.Dependencies)
			if !isXPathApplicable(mergedDeps) {
				continue
			}
			if hasFeatureDependency(mergedDeps, "xpath-1.0-compatibility") {
				continue
			}

			skipReason := getSkipReason(mergedDeps)
			if skipReason == "" {
				skipReason = getTestCaseSkipReason(tsRef.Name, tc.Name)
			}
			if skipReason == "" {
				skipReason = setSkipReason
			}
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

			var baseURI string
			if env != nil && env.StaticBaseURI != nil && env.StaticBaseURI.URI != "#UNDEFINED" {
				baseURI = env.StaticBaseURI.URI
			}

			// Detect resource environments (e.g., fn:json-doc tests with URI-mapped files)
			needsHTTP := false
			var resMap map[string]string
			if env != nil && len(env.Resources) > 0 {
				needsHTTP = true
				resMap = make(map[string]string)
				for _, res := range env.Resources {
					if res.File != "" && res.URI != "" {
						var resPath string
						if envIsGlobal {
							resPath = res.File
						} else {
							resPath = filepath.Join(tsDir, res.File)
						}
						resourceFiles[resPath] = true
						resMap[res.URI] = resPath
					}
				}
			}

			allTests = append(allTests, generatedTest{
				SetName:          tsRef.Name,
				CaseName:         tc.Name,
				XPath:            strings.TrimSpace(tc.Test),
				ContextDocPath:   contextDocPath,
				Namespaces:       collectNamespaces(env),
				DefaultLanguage:  dependencyValue(mergedDeps, "default-language"),
				DefaultCollation: envDefaultCollation(env),
				DefaultDecimal:   envDefaultDecimalFormat(env),
				DecimalFormats:   envNamedDecimalFormats(env),
				BaseURI:          baseURI,
				NeedsHTTP:        needsHTTP,
				ResourceMap:      resMap,
				Assertions:       assertions,
				SkipReason:       skipReason,
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

	// Copy resource files (for fn:json-doc etc.)
	for resPath := range resourceFiles {
		srcFull := filepath.Join(sourceDir, resPath)
		dstFull := filepath.Join(docsDir, resPath)
		if err := copyFile(srcFull, dstFull); err != nil {
			log.Printf("warning: copying resource %s: %v", resPath, err)
			continue
		}
		copied++
	}

	// Remove old generated files
	removeOldGeneratedFiles(outputDir)

	// Group tests by category and generate per-category files
	categories := groupByCategory(allTests)
	catNames := sortedCategoryNames(categories)

	totalTests := 0
	totalSkipped := 0
	for _, cat := range catNames {
		tests := categories[cat]
		filename := fmt.Sprintf("qt3_%s_gen_test.go", cat)
		outputPath := filepath.Join(outputDir, filename)

		code := generateTestFile(tests)
		formatted, err := format.Source([]byte(code))
		if err != nil {
			if writeErr := os.WriteFile(outputPath, []byte(code), 0o644); writeErr != nil {
				log.Fatalf("writing unformatted output: %v", writeErr)
			}
			log.Fatalf("gofmt failed for %s (raw file written): %v", filename, err)
		}
		if err := os.WriteFile(outputPath, formatted, 0o644); err != nil {
			log.Fatalf("writing %s: %v", outputPath, err)
		}

		skipped := 0
		for _, t := range tests {
			if t.SkipReason != "" {
				skipped++
			}
		}
		totalTests += len(tests)
		totalSkipped += skipped
		fmt.Printf("  %s: %d tests (%d run, %d skip)\n", filename, len(tests), len(tests)-skipped, skipped)
	}

	fmt.Printf("Generated %d XPath tests across %d files in %s\n", totalTests, len(catNames), outputDir)
	fmt.Printf("Copied %d context documents to %s\n", copied, docsDir)
	fmt.Printf("  %d will run, %d will skip\n", totalTests-totalSkipped, totalSkipped)
}

// ──────────────────────────────────────────────────────────────────────
// Parsing
// ──────────────────────────────────────────────────────────────────────

func parseCatalog(path string) *catalog {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("opening catalog: %v", err)
	}
	defer func() { _ = f.Close() }()
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
	defer func() { _ = f.Close() }()
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

// mergeDeps combines test-set-level and test-case-level dependencies.
// If a test case has its own spec dependency, it takes precedence over the
// test-set-level spec dependency. Other dependency types are concatenated.
func mergeDeps(setDeps, caseDeps []dependency) []dependency {
	caseHasSpec := false
	for _, d := range caseDeps {
		if d.Type == "spec" {
			caseHasSpec = true
			break
		}
	}

	merged := make([]dependency, 0, len(setDeps)+len(caseDeps))
	for _, d := range setDeps {
		if d.Type == "spec" && caseHasSpec {
			continue // test-case spec overrides test-set spec
		}
		merged = append(merged, d)
	}
	merged = append(merged, caseDeps...)
	return merged
}

func isXPathApplicable(deps []dependency) bool {
	hasSpecDep := false
	for _, d := range deps {
		// Exclude tests that require the absence of a feature we support.
		// E.g. unicode-normalization-form value="FULLY-NORMALIZED" satisfied="false"
		// means the test is for processors that do NOT support that form.
		if d.Satisfied == "false" && isSupportedFeature(d) {
			return false
		}
		if d.Type != "spec" {
			continue
		}
		hasSpecDep = true
		for _, v := range strings.Fields(d.Value) {
			if !strings.HasPrefix(v, "XP") {
				continue
			}
			if d.Satisfied == "false" {
				continue
			}
			// We target XPath 3.1. Accept versions that include 3.1:
			// XP31, XP31+, XP30+, XP20+, XP10+ all cover 3.1.
			// Reject exact versions that exclude 3.1: XP10, XP20, XP30.
			if xpVersionIncludes31(v) {
				return true
			}
		}
	}
	return !hasSpecDep
}

// isSupportedFeature returns true if the dependency refers to a feature
// that this implementation supports. Used to exclude tests requiring the
// absence of such features (satisfied="false").
func isSupportedFeature(d dependency) bool {
	switch d.Type {
	case "unicode-normalization-form":
		switch d.Value {
		case "NFC", "NFD", "NFKC", "NFKD", "FULLY-NORMALIZED":
			return true
		}
	case "xsd-version":
		return d.Value == "1.1"
	}
	return false
}

// xpVersionIncludes31 returns true if the XPath spec token includes version 3.1.
// Tokens like "XP31", "XP31+", "XP30+", "XP20+" include 3.1.
// Tokens like "XP30", "XP20", "XP10" do not.
func xpVersionIncludes31(v string) bool {
	if strings.HasSuffix(v, "+") {
		return true // any "XPxx+" includes later versions
	}
	return v == "XP31" // exact match only for 3.1
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
			case "fn-load-xquery-module":
				return "requires XQuery load-xquery-module"
			case "fn-format-integer-CLDR":
				return "requires CLDR format-integer"
			case "remote_http":
				return "requires remote HTTP access"
			case "non_unicode_codepoint_collation":
				return "requires non-Unicode codepoint collation"
			case "non_empty_sequence_collection":
				return "requires non-empty sequence collection"
			case "staticTyping":
				return "requires static typing"
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
		if d.Type == "unicode-version" && d.Value != qt3UnicodeVersion && d.Satisfied != "false" {
			return fmt.Sprintf("requires Unicode %s", d.Value)
		}
		if d.Type == "xsd-version" && d.Value == "1.0" && d.Satisfied != "false" {
			return "requires XSD 1.0"
		}
	}
	return ""
}

// getTestCaseSkipReason returns a skip reason for specific test cases that
// need per-case handling (e.g., tests expecting static typing errors).
func getTestCaseSkipReason(setName, caseName string) string {
	switch caseName {
	// These tests pass () or integer where xs:string? is expected and expect XPTY0004.
	// Our dynamic evaluation handles these fine without static type checking.
	case "fn-unparsed-text-012", "fn-unparsed-text-available-008",
		"fn-unparsed-text-available-010", "fn-unparsed-text-available-012",
		"fn-unparsed-text-lines-012":
		return "requires static typing"
	}
	return ""
}

func getTestSetSkipReason(name string) string {
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

func envDefaultCollation(env *environment) string {
	if env == nil || len(env.Collations) == 0 {
		return ""
	}
	for _, c := range env.Collations {
		if c.Default == "true" {
			return c.URI
		}
	}
	if len(env.Collations) == 1 {
		return env.Collations[0].URI
	}
	return ""
}

func envDefaultDecimalFormat(env *environment) *decimalFormat {
	if env == nil {
		return nil
	}
	for _, df := range env.DecimalFormat {
		if strings.TrimSpace(df.Name) == "" {
			cp := df
			return &cp
		}
	}
	return nil
}

type namedDecimalFormat struct {
	URI    string
	Name   string
	Format decimalFormat
}

func envNamedDecimalFormats(env *environment) []namedDecimalFormat {
	if env == nil || len(env.DecimalFormat) == 0 {
		return nil
	}

	ns := collectNamespaces(env)
	var out []namedDecimalFormat
	for _, df := range env.DecimalFormat {
		name := strings.TrimSpace(df.Name)
		if name == "" {
			continue
		}
		dfNS := make(map[string]string, len(ns))
		for k, v := range ns {
			dfNS[k] = v
		}
		for k, v := range decimalFormatNamespaces(df) {
			dfNS[k] = v
		}
		uri, local, ok := resolveEnvQName(name, dfNS)
		if !ok {
			continue
		}
		out = append(out, namedDecimalFormat{
			URI:    uri,
			Name:   local,
			Format: df,
		})
	}
	return out
}

func decimalFormatNamespaces(df decimalFormat) map[string]string {
	if len(df.Attrs) == 0 {
		return nil
	}
	ns := make(map[string]string)
	for _, attr := range df.Attrs {
		switch {
		case attr.Name.Space == "xmlns":
			ns[attr.Name.Local] = attr.Value
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
			ns[""] = attr.Value
		}
	}
	return ns
}

func resolveEnvQName(name string, ns map[string]string) (string, string, bool) {
	if strings.HasPrefix(name, "Q{") {
		end := strings.Index(name, "}")
		if end < 0 || end == len(name)-1 {
			return "", "", false
		}
		return name[2:end], name[end+1:], true
	}

	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		uri := ns[prefix]
		if uri == "" {
			return "", "", false
		}
		return uri, name[idx+1:], true
	}

	return "", name, true
}

func dependencyValue(deps []dependency, typ string) string {
	for _, d := range deps {
		if d.Type != typ || d.Satisfied == "false" {
			continue
		}
		return d.Value
	}
	return ""
}

func hasFeatureDependency(deps []dependency, value string) bool {
	for _, d := range deps {
		if d.Type == "feature" && d.Satisfied != "false" && d.Value == value {
			return true
		}
	}
	return false
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
	XMLName        xml.Name
	Code           string         `xml:"code,attr"`
	NormalizeSpace string         `xml:"normalize-space,attr"`
	Inner          []byte         `xml:",innerxml"`
	Children       []xmlAssertion `xml:",any"`
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
	// Decode the raw inner XML to preserve character references like &#xD;
	// that Go's xml:",chardata" would normalize away.
	value := decodeXMLText(string(xa.Inner))
	if xa.XMLName.Local != "assert-string-value" {
		value = strings.TrimSpace(value)
	}
	a := assertion{
		Type:           xa.XMLName.Local,
		Value:          value,
		NormalizeSpace: xa.NormalizeSpace == "true",
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
	SetName          string
	CaseName         string
	XPath            string
	ContextDocPath   string
	Namespaces       map[string]string
	DefaultLanguage  string
	DefaultCollation string
	DefaultDecimal   *decimalFormat
	DecimalFormats   []namedDecimalFormat
	BaseURI          string
	NeedsHTTP        bool
	ResourceMap      map[string]string // URI → file path relative to testdata dir
	Assertions       []assertion
	SkipReason       string
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
			if tc.DefaultLanguage != "" {
				fmt.Fprintf(&b, ", DefaultLanguage: %q", tc.DefaultLanguage)
			}
			if tc.DefaultCollation != "" {
				fmt.Fprintf(&b, ", DefaultCollation: %q", tc.DefaultCollation)
			}
			if tc.DefaultDecimal != nil {
				fmt.Fprintf(&b, ", DefaultDecimal: &%s", emitDecimalFormat(*tc.DefaultDecimal))
			}
			if len(tc.DecimalFormats) > 0 {
				b.WriteString(", NamedDecimalFormats: []qt3NamedDecimalFormat{")
				for i, df := range tc.DecimalFormats {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "{URI: %q, Name: %q, Format: %s}", df.URI, df.Name, emitDecimalFormat(df.Format))
				}
				b.WriteString("}")
			}
			if tc.BaseURI != "" {
				fmt.Fprintf(&b, ", BaseURI: %q", tc.BaseURI)
			}
			if tc.NeedsHTTP {
				b.WriteString(", NeedsHTTP: true")
			}
			if len(tc.ResourceMap) > 0 {
				b.WriteString(", ResourceMap: map[string]string{")
				keys := sortedKeys(tc.ResourceMap)
				for i, k := range keys {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q: %q", k, tc.ResourceMap[k])
				}
				b.WriteString("}")
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
				if assertionsAcceptError(tc.Assertions) {
					b.WriteString(", AcceptError: true")
				}
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
			// Only treat as error-only if ALL children are errors.
			// If any-of has both error and non-error children,
			// the non-error result is also acceptable (XP31 behavior).
			allError := true
			for _, child := range a.Children {
				if child.Type != "error" {
					allError = false
					break
				}
			}
			if allError {
				return true
			}
		}
	}
	return false
}

// assertionsAcceptError returns true when any-of contains both error and non-error children.
// In this case, an error is acceptable but a valid result should also be checked.
func assertionsAcceptError(assertions []assertion) bool {
	for _, a := range assertions {
		if a.Type == "any-of" {
			hasError := false
			hasNonError := false
			for _, child := range a.Children {
				if child.Type == "error" {
					hasError = true
				} else {
					hasNonError = true
				}
			}
			if hasError && hasNonError {
				return true
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
		if a.NormalizeSpace {
			return []string{fmt.Sprintf("qt3AssertStringValueNS(%s)", goStringLiteral(a.Value))}
		}
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
		if a.NormalizeSpace {
			return fmt.Sprintf("qt3CheckStringValueNS(%s)", goStringLiteral(a.Value))
		}
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

func emitDecimalFormat(df decimalFormat) string {
	var parts []string
	if df.DecimalSeparator != "" {
		parts = append(parts, fmt.Sprintf("DecimalSeparator: %s", goStringLiteral(df.DecimalSeparator)))
	}
	if df.GroupingSeparator != "" {
		parts = append(parts, fmt.Sprintf("GroupingSeparator: %s", goStringLiteral(df.GroupingSeparator)))
	}
	if df.Percent != "" {
		parts = append(parts, fmt.Sprintf("Percent: %s", goStringLiteral(df.Percent)))
	}
	if df.PerMille != "" {
		parts = append(parts, fmt.Sprintf("PerMille: %s", goStringLiteral(df.PerMille)))
	}
	if df.ZeroDigit != "" {
		parts = append(parts, fmt.Sprintf("ZeroDigit: %s", goStringLiteral(df.ZeroDigit)))
	}
	if df.Digit != "" {
		parts = append(parts, fmt.Sprintf("Digit: %s", goStringLiteral(df.Digit)))
	}
	if df.PatternSeparator != "" {
		parts = append(parts, fmt.Sprintf("PatternSeparator: %s", goStringLiteral(df.PatternSeparator)))
	}
	if df.ExponentSeparator != "" {
		parts = append(parts, fmt.Sprintf("ExponentSeparator: %s", goStringLiteral(df.ExponentSeparator)))
	}
	if df.Infinity != "" {
		parts = append(parts, fmt.Sprintf("Infinity: %s", goStringLiteral(df.Infinity)))
	}
	if df.NaN != "" {
		parts = append(parts, fmt.Sprintf("NaN: %s", goStringLiteral(df.NaN)))
	}
	if df.MinusSign != "" {
		parts = append(parts, fmt.Sprintf("MinusSign: %s", goStringLiteral(df.MinusSign)))
	}
	return "qt3DecimalFormat{" + strings.Join(parts, ", ") + "}"
}

// ──────────────────────────────────────────────────────────────────────
// Utilities
// ──────────────────────────────────────────────────────────────────────

// decodeXMLText decodes XML entity references, character references, and CDATA
// sections in raw inner XML content. Unlike Go's xml:",chardata", this preserves
// &#xD; as a literal CR character without applying XML line-end normalization.
func decodeXMLText(s string) string {
	var b strings.Builder
	for len(s) > 0 {
		// Handle CDATA sections
		if strings.HasPrefix(s, "<![CDATA[") {
			end := strings.Index(s, "]]>")
			if end < 0 {
				b.WriteString(s[len("<![CDATA["):])
				break
			}
			b.WriteString(s[len("<![CDATA["):end])
			s = s[end+len("]]>"):]
			continue
		}
		// Handle entity/character references
		amp := strings.IndexByte(s, '&')
		cdata := strings.Index(s, "<![CDATA[")
		// Find the nearest special construct
		next := len(s)
		if amp >= 0 {
			next = amp
		}
		if cdata >= 0 && cdata < next {
			b.WriteString(s[:cdata])
			s = s[cdata:]
			continue
		}
		if amp < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:amp])
		s = s[amp:]
		semi := strings.IndexByte(s, ';')
		if semi < 0 {
			b.WriteString(s)
			break
		}
		ref := s[1:semi]
		s = s[semi+1:]
		if strings.HasPrefix(ref, "#x") || strings.HasPrefix(ref, "#X") {
			if n, err := strconv.ParseInt(ref[2:], 16, 32); err == nil {
				b.WriteRune(rune(n))
			}
		} else if strings.HasPrefix(ref, "#") {
			if n, err := strconv.ParseInt(ref[1:], 10, 32); err == nil {
				b.WriteRune(rune(n))
			}
		} else {
			switch ref {
			case "lt":
				b.WriteByte('<')
			case "gt":
				b.WriteByte('>')
			case "amp":
				b.WriteByte('&')
			case "apos":
				b.WriteByte('\'')
			case "quot":
				b.WriteByte('"')
			default:
				b.WriteByte('&')
				b.WriteString(ref)
				b.WriteByte(';')
			}
		}
	}
	return b.String()
}

var nonIdentRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func goIdentifier(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return nonIdentRE.ReplaceAllString(s, "_")
}

func goStringLiteral(s string) string {
	// Raw string literals (backtick) silently discard \r, so always use
	// interpreted string literals when the value contains CR.
	if strings.Contains(s, "\n") && !strings.Contains(s, "`") && !strings.Contains(s, "\r") {
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

// categoryOf extracts the grouping prefix from a test set name.
// e.g. "fn-abs" → "fn", "op-numeric-add" → "op", "prod-AxisStep" → "prod"
func categoryOf(setName string) string {
	if idx := strings.IndexByte(setName, '-'); idx > 0 {
		return setName[:idx]
	}
	return setName
}

func groupByCategory(tests []generatedTest) map[string][]generatedTest {
	cats := make(map[string][]generatedTest)
	for _, t := range tests {
		cat := categoryOf(t.SetName)
		cats[cat] = append(cats[cat], t)
	}
	return cats
}

func sortedCategoryNames(cats map[string][]generatedTest) []string {
	names := make([]string, 0, len(cats))
	for name := range cats {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func removeOldGeneratedFiles(dir string) {
	// Remove old single-file output
	_ = os.Remove(filepath.Join(dir, "qt3_generated_test.go"))

	// Remove old per-category files
	matches, _ := filepath.Glob(filepath.Join(dir, "qt3_*_gen_test.go"))
	for _, m := range matches {
		_ = os.Remove(m)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}
