// Command xslt3gen parses the W3C XSLT 3.0 test catalog and generates
// table-driven Go test files for the xslt3 package.
//
// Usage:
//
//	go run ./tools/xslt3gen
//
// Prerequisites:
//
//	bash testdata/xslt30/fetch.sh   (clone the XSLT 3.0 test suite first)
//
// Output:
//
//	xslt3/w3c_<category>_gen_test.go   (table-driven tests per category)
//	testdata/xslt30/testdata/          (copied stylesheet + source assets)
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
// XSLT 3.0 test catalog XML types
// ──────────────────────────────────────────────────────────────────────

type xslCatalog struct {
	TestSets []xslTestSetRef `xml:"test-set"`
}

type xslTestSetRef struct {
	Name string `xml:"name,attr"`
	File string `xml:"file,attr"`
}

type xslTestSetFile struct {
	Name         string           `xml:"name,attr"`
	Dependencies *xslDependencies `xml:"dependencies"`
	Environments []xslEnvironment `xml:"environment"`
	TestCases    []xslTestCase    `xml:"test-case"`
}

type xslDependencies struct {
	Children []xslDependency `xml:",any"`
}

type xslDependency struct {
	XMLName   xml.Name
	Value     string `xml:"value,attr"`
	Satisfied string `xml:"satisfied,attr"`
}

type xslEnvironment struct {
	Name    string      `xml:"name,attr"`
	Ref     string      `xml:"ref,attr"`
	Sources []xslSource `xml:"source"`
}

type xslSource struct {
	Role    string      `xml:"role,attr"`
	File    string      `xml:"file,attr"`
	URI     string      `xml:"uri,attr"`
	Content *xslContent `xml:"content"`
}

type xslContent struct {
	Inner []byte `xml:",innerxml"`
}

type xslTestCase struct {
	Name         string           `xml:"name,attr"`
	Dependencies *xslDependencies `xml:"dependencies"`
	Environment  *xslEnvironment  `xml:"environment"`
	Test         xslTest          `xml:"test"`
	Result       xslResult        `xml:",any"`
}

type xslTest struct {
	Stylesheets     []xslStylesheet     `xml:"stylesheet"`
	InitialTemplate *xslInitialTemplate `xml:"initial-template"`
	Params          []xslParam          `xml:"param"`
}

type xslStylesheet struct {
	Role string `xml:"role,attr"`
	File string `xml:"file,attr"`
}

type xslInitialTemplate struct {
	Name  string     `xml:"name,attr"`
	Attrs []xml.Attr `xml:",any,attr"`
}

type xslParam struct {
	Name   string `xml:"name,attr"`
	Select string `xml:"select,attr"`
}

type xslResult struct {
	XMLName xml.Name
	Inner   []byte `xml:",innerxml"`
}

// ──────────────────────────────────────────────────────────────────────
// Assertion types
// ──────────────────────────────────────────────────────────────────────

type assertion struct {
	Type     string
	Value    string
	Children []assertion
}

type xmlResultWrapper struct {
	Children []xmlAssertion `xml:",any"`
}

type xmlAssertion struct {
	XMLName  xml.Name
	Code     string         `xml:"code,attr"`
	File     string         `xml:"file,attr"`
	Inner    []byte         `xml:",innerxml"`
	Children []xmlAssertion `xml:",any"`
}

// ──────────────────────────────────────────────────────────────────────
// Generated test structure
// ──────────────────────────────────────────────────────────────────────

type generatedTest struct {
	SetName              string
	CaseName             string
	Category             string // from catalog path: tests/<category>/...
	StylesheetPath       string
	SecondaryStylesheets []string
	SourceDocPath        string
	SourceContent        string
	InitialTemplate      string
	Params               map[string]string
	ExpectError          bool
	ErrorCode            string
	Assertions           []assertion
	Skip                 string
}

// ──────────────────────────────────────────────────────────────────────
// Main
// ──────────────────────────────────────────────────────────────────────

func main() {
	repoRoot := findRepoRoot()
	sourceDir := filepath.Join(repoRoot, "testdata", "xslt30", "source")
	outputDir := filepath.Join(repoRoot, "xslt3")
	assetsDir := filepath.Join(repoRoot, "testdata", "xslt30", "testdata")

	if _, err := os.Stat(filepath.Join(sourceDir, "catalog.xml")); os.IsNotExist(err) {
		log.Fatal("XSLT 3.0 source not found. Run: bash testdata/xslt30/fetch.sh")
	}

	cat := parseCatalog(filepath.Join(sourceDir, "catalog.xml"))
	schemaKnownFailures := loadSchemaKnownFailures(repoRoot)

	var allTests []generatedTest
	assetFiles := make(map[string]struct{})

	for _, tsRef := range cat.TestSets {
		tsFile := filepath.Join(sourceDir, tsRef.File)
		ts := parseTestSet(tsFile)
		tsDir := filepath.Dir(tsRef.File)              // e.g. "tests/insn/apply-templates"
		tsDirAbs := filepath.Join(sourceDir, tsDir)     // absolute path for reading files

		localEnvs := make(map[string]*xslEnvironment)
		for i := range ts.Environments {
			localEnvs[ts.Environments[i].Name] = &ts.Environments[i]
		}

		// Test-set-level skip reason
		setSkip := getSetSkipReason(tsRef.Name, ts.Dependencies)

		// Determine category from catalog path directory
		cat := categoryFromCatalogPath(tsRef.File)

		for _, tc := range ts.TestCases {
			gt := generatedTest{
				SetName:  tsRef.Name,
				CaseName: tc.Name,
				Category: cat,
			}

			// Merge dependencies: test-set level + test-case level
			skipReason := setSkip
			if skipReason == "" {
				skipReason = getCaseSkipReason(ts.Dependencies, tc.Dependencies)
			}
			gt.Skip = skipReason

			// Find primary and secondary stylesheets
			for _, ss := range tc.Test.Stylesheets {
				relPath := filepath.Join(tsDir, ss.File)
				if ss.Role == "secondary" {
					gt.SecondaryStylesheets = append(gt.SecondaryStylesheets, relPath)
				} else {
					gt.StylesheetPath = relPath
				}
			}
			// If only one stylesheet and no explicit role, it's the primary
			if gt.StylesheetPath == "" && len(tc.Test.Stylesheets) == 1 {
				gt.StylesheetPath = filepath.Join(tsDir, tc.Test.Stylesheets[0].File)
			}

			if gt.StylesheetPath == "" {
				gt.Skip = "no stylesheet"
			}

			// Initial template
			if tc.Test.InitialTemplate != nil {
				gt.InitialTemplate = resolveInitialTemplateName(tc.Test.InitialTemplate)
			}

			// Params
			if len(tc.Test.Params) > 0 {
				gt.Params = make(map[string]string)
				for _, p := range tc.Test.Params {
					gt.Params[p.Name] = p.Select
				}
			}

			// Environment: resolve source document
			env := resolveEnvironment(tc.Environment, localEnvs)
			if env != nil {
				for _, src := range env.Sources {
					if src.File != "" {
						relPath := filepath.Join(tsDir, src.File)
						// Always copy source files as assets (for document()/fn:doc() etc.)
						assetFiles[relPath] = struct{}{}
					}
					if src.Role != "." {
						continue
					}
					if src.File != "" {
						gt.SourceDocPath = filepath.Join(tsDir, src.File)
					} else if src.Content != nil {
						gt.SourceContent = decodeXMLText(string(src.Content.Inner))
					}
				}
			}

			// Parse assertions
			gt.Assertions = parseResultAssertions(tc, tsDirAbs)
			classifyError(&gt)
			classifyUnsupportedAssertions(&gt)
			isSchemaAware := hasFeatureDep(ts.Dependencies, "schema_aware") ||
				hasFeatureDep(tc.Dependencies, "schema_aware") ||
				hasFeatureDep(ts.Dependencies, "schema-aware") ||
				hasFeatureDep(tc.Dependencies, "schema-aware")
			classifySchemaTypeChecking(&gt, isSchemaAware)
			if isSchemaAware && gt.Skip == "" && gt.StylesheetPath != "" {
				classifyAdvancedSchemaFeatures(&gt, filepath.Join(sourceDir, gt.StylesheetPath))
			}
			if gt.Skip == "" {
				if _, known := schemaKnownFailures[gt.CaseName]; known {
					gt.Skip = "requires full schema validation support"
				}
			}

			// Track stylesheet assets
			if gt.StylesheetPath != "" {
				assetFiles[gt.StylesheetPath] = struct{}{}
				// Scan transitive xsl:import/xsl:include deps
				absPath := filepath.Join(sourceDir, gt.StylesheetPath)
				for _, dep := range collectTransitiveDeps(absPath) {
					relDep, err := filepath.Rel(sourceDir, dep)
					if err == nil {
						assetFiles[relDep] = struct{}{}
					}
				}
			}
			for _, ss := range gt.SecondaryStylesheets {
				assetFiles[ss] = struct{}{}
				absPath := filepath.Join(sourceDir, ss)
				for _, dep := range collectTransitiveDeps(absPath) {
					relDep, err := filepath.Rel(sourceDir, dep)
					if err == nil {
						assetFiles[relDep] = struct{}{}
					}
				}
			}

			allTests = append(allTests, gt)
		}
	}

	// Copy asset files
	copied := 0
	for relPath := range assetFiles {
		srcFull := filepath.Join(sourceDir, relPath)
		dstFull := filepath.Join(assetsDir, relPath)
		if err := copyFile(srcFull, dstFull); err != nil {
			log.Printf("warning: copying %s: %v", relPath, err)
			continue
		}
		copied++
	}

	// Remove old generated files
	removeOldGeneratedFiles(outputDir)

	// Group tests by category and generate per-category files
	categories := groupByCategory(allTests)
	catNames := sortedMapKeys(categories)

	totalTests := 0
	totalSkipped := 0
	for _, cat := range catNames {
		tests := categories[cat]
		filename := fmt.Sprintf("w3c_%s_gen_test.go", cat)
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
			if t.Skip != "" {
				skipped++
			}
		}
		totalTests += len(tests)
		totalSkipped += skipped
		fmt.Printf("  %s: %d tests (%d run, %d skip)\n", filename, len(tests), len(tests)-skipped, skipped)
	}

	fmt.Printf("Generated %d XSLT tests across %d files in %s\n", totalTests, len(catNames), outputDir)
	fmt.Printf("Copied %d asset files to %s\n", copied, assetsDir)
	fmt.Printf("  %d will run, %d will skip\n", totalTests-totalSkipped, totalSkipped)
}

// ──────────────────────────────────────────────────────────────────────
// Catalog parsing
// ──────────────────────────────────────────────────────────────────────

func parseCatalog(path string) *xslCatalog {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("opening catalog: %v", err)
	}
	defer func() { _ = f.Close() }()
	var c xslCatalog
	dec := xml.NewDecoder(f)
	dec.CharsetReader = charset.NewReaderLabel
	if err := dec.Decode(&c); err != nil {
		log.Fatalf("parsing catalog: %v", err)
	}
	return &c
}

func parseTestSet(path string) *xslTestSetFile {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("opening test set %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	var ts xslTestSetFile
	dec := xml.NewDecoder(f)
	dec.CharsetReader = charset.NewReaderLabel
	if err := dec.Decode(&ts); err != nil {
		log.Fatalf("parsing test set %s: %v", path, err)
	}
	return &ts
}

// ──────────────────────────────────────────────────────────────────────
// Spec & feature filtering
// ──────────────────────────────────────────────────────────────────────

func getSetSkipReason(name string, deps *xslDependencies) string {
	if deps != nil {
		if reason := getDepsSkipReason(deps); reason != "" {
			return reason
		}
	}
	return ""
}

func getCaseSkipReason(setDeps *xslDependencies, caseDeps *xslDependencies) string {
	// Check set-level deps first
	if setDeps != nil {
		if reason := getDepsSkipReason(setDeps); reason != "" {
			return reason
		}
	}
	// Check case-level deps (override set-level if present)
	if caseDeps != nil {
		if reason := getDepsSkipReason(caseDeps); reason != "" {
			return reason
		}
	}
	return ""
}

func getDepsSkipReason(deps *xslDependencies) string {
	for _, d := range deps.Children {
		typeName := d.XMLName.Local
		switch typeName {
		case "spec":
			if !specSupported(d.Value) {
				return fmt.Sprintf("unsupported spec: %s", d.Value)
			}
		case "feature":
			if d.Satisfied == "false" {
				// Test requires the feature to be absent. Skip if we support it.
				if featureSupported(d.Value) {
					return fmt.Sprintf("feature present but test requires absent: %s", d.Value)
				}
				continue
			}
			if !featureSupported(d.Value) {
				return fmt.Sprintf("unsupported feature: %s", d.Value)
			}
		case "on-multiple-match":
			// Skip tests requiring specific multiple-match behavior
			// that we don't test for
		}
	}
	return ""
}

func specSupported(spec string) bool {
	for _, s := range strings.Fields(spec) {
		switch s {
		case "XSLT10", "XSLT10+", "XSLT20", "XSLT20+", "XSLT30", "XSLT30+":
			return true
		}
	}
	return false
}

func featureSupported(feature string) bool {
	switch feature {
	case "schema_aware", "schema-aware",
		"backwards_compatibility",
		"Saxon-PE", "Saxon-EE":
		return false
	}
	return true
}

// isHeavyTestSet returns true for test sets that are computationally expensive
// and should run serially behind the HELIUM_HEAVY_TESTS env var.
func isHeavyTestSet(name string) bool {
	switch name {
	case "unicode-90":
		return true
	}
	return false
}

// ──────────────────────────────────────────────────────────────────────
// Environment resolution
// ──────────────────────────────────────────────────────────────────────

func resolveEnvironment(tcEnv *xslEnvironment, localEnvs map[string]*xslEnvironment) *xslEnvironment {
	if tcEnv == nil {
		return nil
	}
	if tcEnv.Ref != "" {
		if e, ok := localEnvs[tcEnv.Ref]; ok {
			return e
		}
		return nil
	}
	// Inline environment
	return tcEnv
}

func resolveInitialTemplateName(it *xslInitialTemplate) string {
	name := it.Name
	if name == "" {
		return ""
	}
	// Resolve QName prefix using xmlns attributes
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		for _, attr := range it.Attrs {
			if attr.Name.Space == "xmlns" && attr.Name.Local == prefix {
				return "{" + attr.Value + "}" + local
			}
			if attr.Name.Space == "" && attr.Name.Local == "xmlns" && prefix == "" {
				return "{" + attr.Value + "}" + local
			}
		}
		// Couldn't resolve — use as-is with braces for namespace lookup
	}
	return name
}

// ──────────────────────────────────────────────────────────────────────
// Stylesheet dependency scanning
// ──────────────────────────────────────────────────────────────────────

func collectTransitiveDeps(xslPath string) []string {
	visited := make(map[string]struct{})
	var result []string
	collectDepsRecursive(xslPath, visited, &result)
	return result
}

func collectDepsRecursive(xslPath string, visited map[string]struct{}, result *[]string) {
	absPath, err := filepath.Abs(xslPath)
	if err != nil {
		return
	}
	if _, ok := visited[absPath]; ok {
		return
	}
	visited[absPath] = struct{}{}

	f, err := os.Open(absPath)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	dec := xml.NewDecoder(f)
	dec.CharsetReader = charset.NewReaderLabel

	dir := filepath.Dir(absPath)
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		isXSLT := se.Name.Space == "" || se.Name.Space == "http://www.w3.org/1999/XSL/Transform"
		isXSD := se.Name.Space == "" || se.Name.Space == "http://www.w3.org/2001/XMLSchema"

		// xsl:import / xsl:include → follow href recursively
		if isXSLT && (se.Name.Local == "import" || se.Name.Local == "include") {
			for _, attr := range se.Attr {
				if attr.Name.Local == "href" {
					depPath := filepath.Join(dir, attr.Value)
					*result = append(*result, depPath)
					collectDepsRecursive(depPath, visited, result)
				}
			}
			continue
		}
		// xsl:import-schema → follow schema-location
		if isXSLT && se.Name.Local == "import-schema" {
			for _, attr := range se.Attr {
				if attr.Name.Local == "schema-location" && attr.Value != "" {
					depPath := filepath.Join(dir, attr.Value)
					*result = append(*result, depPath)
					collectSchemaDeps(depPath, visited, result)
				}
			}
			continue
		}
		// xs:include / xs:import inside .xsd files → follow schemaLocation
		if isXSD && (se.Name.Local == "include" || se.Name.Local == "import") {
			for _, attr := range se.Attr {
				if attr.Name.Local == "schemaLocation" && attr.Value != "" {
					depPath := filepath.Join(dir, attr.Value)
					*result = append(*result, depPath)
					collectSchemaDeps(depPath, visited, result)
				}
			}
			continue
		}
		if se.Name.Local != "import" && se.Name.Local != "include" && se.Name.Local != "import-schema" {
			continue
		}
	}
}

// collectSchemaDeps scans an XSD file for xs:include/xs:import with
// schemaLocation attributes and collects them as transitive dependencies.
func collectSchemaDeps(xsdPath string, visited map[string]struct{}, result *[]string) {
	absPath, err := filepath.Abs(xsdPath)
	if err != nil {
		return
	}
	if _, ok := visited[absPath]; ok {
		return
	}
	visited[absPath] = struct{}{}

	f, err := os.Open(absPath)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	dec := xml.NewDecoder(f)
	dec.CharsetReader = charset.NewReaderLabel

	dir := filepath.Dir(absPath)
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local != "include" && se.Name.Local != "import" {
			continue
		}
		for _, attr := range se.Attr {
			if attr.Name.Local == "schemaLocation" && attr.Value != "" {
				depPath := filepath.Join(dir, attr.Value)
				*result = append(*result, depPath)
				collectSchemaDeps(depPath, visited, result)
			}
		}
	}
}

// ──────────────────────────────────────────────────────────────────────
// Assertion parsing
// ──────────────────────────────────────────────────────────────────────

const xslTestNS = "http://www.w3.org/2012/10/xslt-test-catalog"

func parseResultAssertions(tc xslTestCase, tsDir string) []assertion {
	resultXML := "<result xmlns=\"" + xslTestNS + "\">" + string(tc.Result.Inner) + "</result>"
	return parseAssertionXML(resultXML, tsDir)
}

func parseAssertionXML(s string, tsDir string) []assertion {
	var result xmlResultWrapper
	if err := xml.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	var out []assertion
	for _, child := range result.Children {
		out = append(out, convertAssertion(child, tsDir))
	}
	return out
}

func convertAssertion(xa xmlAssertion, tsDir string) assertion {
	a := assertion{
		Type: xa.XMLName.Local,
	}

	switch xa.XMLName.Local {
	case "assert-xml":
		if xa.File != "" {
			// Read the .out file content at gen time and embed it
			outPath := filepath.Join(tsDir, xa.File)
			data, err := os.ReadFile(outPath)
			if err != nil {
				a.Value = fmt.Sprintf("ERROR: cannot read %s: %v", xa.File, err)
			} else {
				a.Value = string(data)
			}
		} else {
			a.Value = decodeXMLText(string(xa.Inner))
		}
	case "error":
		a.Value = xa.Code
	case "assert-string-value":
		a.Value = decodeXMLText(string(xa.Inner))
	case "all-of", "any-of":
		for _, child := range xa.Children {
			a.Children = append(a.Children, convertAssertion(child, tsDir))
		}
	case "assert-message":
		for _, child := range xa.Children {
			a.Children = append(a.Children, convertAssertion(child, tsDir))
		}
	case "assert-result-document":
		a.Value = "skip: assert-result-document not supported"
	case "assert-serialization":
		a.Value = "skip: assert-serialization not supported"
	case "assert-posture-and-sweep":
		a.Value = "skip: streaming not supported"
	default:
		a.Value = decodeXMLText(string(xa.Inner))
	}

	return a
}

// classifyError sets ExpectError/ErrorCode on the test when assertions indicate
// an expected error.
func classifyError(gt *generatedTest) {
	if len(gt.Assertions) == 1 && gt.Assertions[0].Type == "error" {
		gt.ExpectError = true
		gt.ErrorCode = gt.Assertions[0].Value
		return
	}
	// Check if top-level any-of with all error children
	if len(gt.Assertions) == 1 && gt.Assertions[0].Type == "any-of" {
		allError := true
		for _, child := range gt.Assertions[0].Children {
			if child.Type != "error" {
				allError = false
				break
			}
		}
		if allError {
			gt.ExpectError = true
			if len(gt.Assertions[0].Children) > 0 {
				gt.ErrorCode = gt.Assertions[0].Children[0].Value
			}
		}
	}
}

// classifyUnsupportedAssertions sets Skip when all emitted assertions would
// be w3cAssertSkip(). This avoids compiling/transforming stylesheets for
// tests whose results cannot be verified.
func classifyUnsupportedAssertions(gt *generatedTest) {
	if gt.Skip != "" || gt.ExpectError {
		return
	}
	exprs := emitAssertions(gt.Assertions)
	if len(exprs) == 0 {
		return
	}
	for _, e := range exprs {
		if e != "w3cAssertSkip()" {
			return
		}
	}
	gt.Skip = unsupportedAssertionReason(gt.Assertions)
}

func unsupportedAssertionReason(assertions []assertion) string {
	for _, a := range assertions {
		switch a.Type {
		case "assert-result-document":
			return "unsupported assertion: assert-result-document"
		case "assert-serialization":
			return "unsupported assertion: assert-serialization"
		case "assert-posture-and-sweep":
			return "unsupported assertion: assert-posture-and-sweep"
		case "all-of":
			if reason := unsupportedAssertionReason(a.Children); reason != "" {
				return reason
			}
		}
	}
	return "unsupported assertion type"
}

// classifySchemaTypeChecking marks tests that expect schema-aware runtime
// type-checking errors (XTTE*) as skipped. These tests require the processor
// to validate result types against schema declarations at runtime, which is
// not yet implemented. Only applies to tests from schema_aware test sets.
// hasFeatureDep returns true if deps contains a feature dependency with the given value.
func hasFeatureDep(deps *xslDependencies, feature string) bool {
	if deps == nil {
		return false
	}
	for _, d := range deps.Children {
		if d.XMLName.Local == "feature" && d.Value == feature && d.Satisfied != "false" {
			return true
		}
	}
	return false
}

func classifySchemaTypeChecking(gt *generatedTest, isSchemaAware bool) {
	if gt.Skip != "" {
		return
	}
	if !gt.ExpectError || !isSchemaAware {
		return
	}
	// Skip schema-aware tests requiring type checking or static checking
	// that we don't enforce.
	switch {
	case strings.HasPrefix(gt.ErrorCode, "XTTE"):
		gt.Skip = "requires runtime type checking: " + gt.ErrorCode
	case gt.ErrorCode == "XPTY0004":
		gt.Skip = "requires schema-aware type checking: " + gt.ErrorCode
	case gt.ErrorCode == "XTSE0220" || gt.ErrorCode == "XTSE0215":
		gt.Skip = "requires schema-aware static checking: " + gt.ErrorCode
	case gt.ErrorCode == "XTSE0010":
		gt.Skip = "requires schema-aware static checking: " + gt.ErrorCode
	case gt.ErrorCode == "XXXX9999":
		gt.Skip = "placeholder error code: " + gt.ErrorCode
	}
}

// loadSchemaKnownFailures reads the schema_known_failures.txt file containing
// test case names (one per line) that require full schema validation and cannot
// pass with partial schema support.
func loadSchemaKnownFailures(repoRoot string) map[string]struct{} {
	path := filepath.Join(repoRoot, "tools", "xslt3gen", "schema_known_failures.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	result := make(map[string]struct{})
	for _, line := range strings.Split(string(data), "\n") {
		name := strings.TrimSpace(line)
		if name != "" && !strings.HasPrefix(name, "#") {
			result[name] = struct{}{}
		}
	}
	return result
}

// classifyAdvancedSchemaFeatures scans a schema-aware test stylesheet for
// features that require deep schema support beyond basic type annotations.
// If found, the test is skipped with a specific reason.
func classifyAdvancedSchemaFeatures(gt *generatedTest, xslPath string) {
	data, err := os.ReadFile(xslPath)
	if err != nil {
		return
	}
	content := string(data)

	// schema-element() / schema-attribute() in match/as/select patterns
	if strings.Contains(content, "schema-element(") || strings.Contains(content, "schema-attribute(") {
		gt.Skip = "requires schema-element/schema-attribute node tests"
		return
	}
	// validation="strict" or validation="lax" on instructions (requires full schema validation)
	if strings.Contains(content, `validation="strict"`) || strings.Contains(content, `validation="lax"`) {
		gt.Skip = "requires schema validation (strict/lax)"
		return
	}
	// document-node(schema-element(...))
	if strings.Contains(content, "document-node(schema-element(") {
		gt.Skip = "requires document-node(schema-element()) support"
		return
	}
	// strip-type-annotations requires source document schema validation
	if strings.Contains(content, "strip-type-annotations") {
		gt.Skip = "requires source document schema validation"
		return
	}
	// xsl:import-schema combined with instance-of schema type tests on source nodes
	// requires source document validation
	if strings.Contains(content, "import-schema") &&
		(strings.Contains(content, "instance of element(") ||
			strings.Contains(content, "instance of attribute(")) {
		gt.Skip = "requires source document schema validation"
		return
	}
}

// ──────────────────────────────────────────────────────────────────────
// Code generation
// ──────────────────────────────────────────────────────────────────────

func generateTestFile(tests []generatedTest) string {
	var b strings.Builder

	b.WriteString("// Code generated by tools/xslt3gen; DO NOT EDIT.\n\n")
	b.WriteString("package xslt3_test\n\n")
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
		funcName := "TestW3C_" + goIdentifier(setName)
		heavy := isHeavyTestSet(setName)

		fmt.Fprintf(&b, "func %s(t *testing.T) {\n", funcName)
		if heavy {
			fmt.Fprintf(&b, "\tw3cRunHeavyTests(t, []w3cTest{\n")
		} else {
			fmt.Fprintf(&b, "\tt.Parallel()\n")
			fmt.Fprintf(&b, "\tw3cRunTests(t, []w3cTest{\n")
		}

		for _, tc := range g.tests {
			b.WriteString("\t\t{")
			fmt.Fprintf(&b, "Name: %q, ", tc.CaseName)
			fmt.Fprintf(&b, "StylesheetPath: %q", tc.StylesheetPath)

			if len(tc.SecondaryStylesheets) > 0 {
				b.WriteString(", SecondaryStylesheets: []string{")
				for i, ss := range tc.SecondaryStylesheets {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q", ss)
				}
				b.WriteString("}")
			}

			if tc.SourceDocPath != "" {
				fmt.Fprintf(&b, ", SourceDocPath: %q", tc.SourceDocPath)
			}
			if tc.SourceContent != "" {
				fmt.Fprintf(&b, ", SourceContent: %s", goStringLiteral(tc.SourceContent))
			}
			if tc.InitialTemplate != "" {
				fmt.Fprintf(&b, ", InitialTemplate: %q", tc.InitialTemplate)
			}
			if len(tc.Params) > 0 {
				b.WriteString(", Params: map[string]string{")
				keys := sortedStringKeys(tc.Params)
				for i, k := range keys {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q: %s", k, goStringLiteral(tc.Params[k]))
				}
				b.WriteString("}")
			}
			if tc.ExpectError {
				b.WriteString(", ExpectError: true")
				if tc.ErrorCode != "" {
					fmt.Fprintf(&b, ", ErrorCode: %q", tc.ErrorCode)
				}
			} else {
				assertExprs := emitAssertions(tc.Assertions)
				if len(assertExprs) > 0 {
					b.WriteString(", Assertions: []w3cAssertion{")
					b.WriteString(strings.Join(assertExprs, ", "))
					b.WriteString("}")
				}
			}
			if tc.Skip != "" {
				fmt.Fprintf(&b, ", Skip: %q", tc.Skip)
			}

			b.WriteString("},\n")
		}

		fmt.Fprintf(&b, "\t})\n")
		fmt.Fprintf(&b, "}\n\n")
	}

	return b.String()
}

func emitAssertions(assertions []assertion) []string {
	var out []string
	for _, a := range assertions {
		out = append(out, emitAssertion(a)...)
	}
	return out
}

func emitAssertion(a assertion) []string {
	switch a.Type {
	case "assert-xml":
		return []string{fmt.Sprintf("w3cAssertXML(%s)", goStringLiteral(strings.TrimSpace(a.Value)))}
	case "assert-string-value":
		return []string{fmt.Sprintf("w3cAssertStringValue(%s)", goStringLiteral(a.Value))}
	case "error":
		return nil // handled by classifyError
	case "all-of":
		return emitAssertions(a.Children)
	case "any-of":
		var checks []string
		for _, child := range a.Children {
			if child.Type == "error" {
				continue
			}
			checks = append(checks, emitCheck(child))
		}
		if len(checks) == 0 {
			return nil
		}
		return []string{fmt.Sprintf("w3cAnyOf(%s)", strings.Join(checks, ", "))}
	case "assert-message":
		var msgChecks []string
		for _, child := range a.Children {
			msgChecks = append(msgChecks, emitCheck(child))
		}
		if len(msgChecks) == 0 {
			return nil
		}
		return []string{fmt.Sprintf("w3cAssertMessage(%s)", strings.Join(msgChecks, ", "))}
	case "assert":
		return []string{fmt.Sprintf("w3cAssertXPath(%s)", goStringLiteral(strings.TrimSpace(a.Value)))}
	case "assert-result-document", "assert-serialization", "assert-posture-and-sweep":
		return []string{"w3cAssertSkip()"}
	default:
		return []string{"w3cAssertSkip()"}
	}
}

func emitCheck(a assertion) string {
	switch a.Type {
	case "assert-xml":
		return fmt.Sprintf("w3cCheckXML(%s)", goStringLiteral(strings.TrimSpace(a.Value)))
	case "assert-string-value":
		return fmt.Sprintf("w3cCheckStringValue(%s)", goStringLiteral(a.Value))
	case "assert":
		return fmt.Sprintf("w3cCheckXPath(%s)", goStringLiteral(strings.TrimSpace(a.Value)))
	case "error":
		return fmt.Sprintf("w3cCheckError(%q)", a.Value)
	default:
		return "w3cCheckSkip()"
	}
}

// ──────────────────────────────────────────────────────────────────────
// Categorization
// ──────────────────────────────────────────────────────────────────────

func groupByCategory(tests []generatedTest) map[string][]generatedTest {
	cats := make(map[string][]generatedTest)
	for _, t := range tests {
		cats[t.Category] = append(cats[t.Category], t)
	}
	return cats
}

// categoryFromCatalogPath extracts the category from the catalog file path.
// e.g. "tests/insn/apply-templates/_apply-templates-test-set.xml" → "insn"
// e.g. "tests/attr/avt/_avt-test-set.xml" → "attr"
func categoryFromCatalogPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) >= 2 && parts[0] == "tests" {
		return parts[1]
	}
	return "misc"
}

// ──────────────────────────────────────────────────────────────────────
// Utilities
// ──────────────────────────────────────────────────────────────────────

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
	if strings.Contains(s, "\n") && !strings.Contains(s, "`") && !strings.Contains(s, "\r") {
		return "`" + s + "`"
	}
	return fmt.Sprintf("%q", s)
}

func sortedStringKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeys[V any](m map[string]V) []string {
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

func removeOldGeneratedFiles(dir string) {
	matches, _ := filepath.Glob(filepath.Join(dir, "w3c_*_gen_test.go"))
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
