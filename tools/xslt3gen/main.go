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
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
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
	Name        string          `xml:"name,attr"`
	Ref         string          `xml:"ref,attr"`
	Sources     []xslSource     `xml:"source"`
	Collections []xslCollection `xml:"collection"`
	Stylesheets []xslStylesheet `xml:"stylesheet"`
	Packages    []xslPackage    `xml:"package"`
	Schemas     []xslSchema     `xml:"schema"`
}

type xslSchema struct {
	Role string `xml:"role,attr"`
	File string `xml:"file,attr"`
}

type xslCollection struct {
	URI     string      `xml:"uri,attr"`
	Sources []xslSource `xml:"source"`
}

type xslSource struct {
	Role              string      `xml:"role,attr"`
	File              string      `xml:"file,attr"`
	URI               string      `xml:"uri,attr"`
	Select            string      `xml:"select,attr"`
	DefinesStylesheet string      `xml:"defines-stylesheet,attr"`
	Content           *xslContent `xml:"content"`
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
	Packages        []xslPackage        `xml:"package"`
	InitialTemplate *xslInitialTemplate `xml:"initial-template"`
	InitialMode     *xslInitialMode     `xml:"initial-mode"`
	InitialFunction *xslInitialFunction `xml:"initial-function"`
	Params          []xslParam          `xml:"param"`
}

type xslInitialFunction struct {
	Name   string     `xml:"name,attr"`
	Attrs  []xml.Attr `xml:",any,attr"`
	Params []xslParam `xml:"param"`
}

type xslInitialMode struct {
	Name   string     `xml:"name,attr"`
	Select string     `xml:"select,attr"`
	Attrs  []xml.Attr `xml:",any,attr"`
	Params []xslParam `xml:"param"`
}

type xslStylesheet struct {
	Role string `xml:"role,attr"`
	File string `xml:"file,attr"`
}

type xslPackage struct {
	Role           string `xml:"role,attr"`
	File           string `xml:"file,attr"`
	URI            string `xml:"uri,attr"`
	PackageVersion string `xml:"package-version,attr"`
}

type xslInitialTemplate struct {
	Name   string     `xml:"name,attr"`
	Attrs  []xml.Attr `xml:",any,attr"`
	Params []xslParam `xml:"param"`
}

type xslParam struct {
	Name   string     `xml:"name,attr"`
	Select string     `xml:"select,attr"`
	As     string     `xml:"as,attr"`
	Tunnel string     `xml:"tunnel,attr"`
	Attrs  []xml.Attr `xml:",any,attr"`
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
	URI      string // for assert-result-document
	Method   string // for assert-serialization
	Flags    string // for serialization-matches (regex flags, e.g. "ix")
	Children []assertion
}

type xmlResultWrapper struct {
	Children []xmlAssertion `xml:",any"`
}

type xmlAssertion struct {
	XMLName  xml.Name
	Code     string         `xml:"code,attr"`
	File     string         `xml:"file,attr"`
	URI      string         `xml:"uri,attr"`
	Method   string         `xml:"method,attr"`
	Flags    string         `xml:"flags,attr"`
	Inner    []byte         `xml:",innerxml"`
	Children []xmlAssertion `xml:",any"`
}

// ──────────────────────────────────────────────────────────────────────
// Generated test structure
// ──────────────────────────────────────────────────────────────────────

type generatedTest struct {
	SetName                     string
	CaseName                    string
	Category                    string // from catalog path: tests/<category>/...
	StylesheetPath              string
	SecondaryStylesheets        []string
	PackageDeps                 []packageDep // secondary packages (uri → file)
	SourceDocPath               string
	SourceContent               string
	InitialTemplate             string
	InitialTemplateParams       map[string]string
	InitialTemplateTunnelParams map[string]string
	InitialMode                 string
	InitialModeSelect           string
	InitialModeParams           map[string]string
	InitialModeTunnelParams     map[string]string
	InitialFunction             string
	InitialFunctionParams       []string // positional params (select expressions)
	Params                      map[string]string
	ParamTypes                  map[string]string // as types for params (from catalog <param as="...">)
	ExpectError                 bool
	AcceptErrors                []string // error codes accepted as alternative outcomes (from any-of)
	ErrorCode                   string
	OnMultipleMatch             string // "fail" when test requires on-multiple-match=error
	Assertions                  []assertion
	Skip                        string
	Collections                 []collectionDef
	SourceSchemaPath            string
	ImportSchemaPaths           []string // schemas for xsl:import-schema resolution
	EmbeddedStylesheet          bool     // source document contains <?xml-stylesheet?> PI
	VersionResolution           string   // "lowest" for lowest matching package version (default: highest)
}

// packageDep maps a package URI+version to its file path.
type packageDep struct {
	URI            string
	PackageVersion string
	FilePath       string // relative to testdata
}

type collectionDef struct {
	URI      string
	DocPaths []string
}

// knownSkips maps test case names to skip reasons for tests that are
// valid per spec but test an alternative behavior our implementation
// does not follow.
var knownSkips = map[string]string{
	// XSLT 3.0 allows zero-length regex matches (XTDE1150 is optional).
	// Our implementation handles them; these variants expect the error.
	"analyze-string-090a": "implementation handles zero-length matches (XSLT 3.0)",
	"analyze-string-091a": "implementation handles zero-length matches (XSLT 3.0)",

	// These tests require package-scoped strip-space isolation not yet implemented.
	"document-2401": "requires package-scoped strip-space isolation",
	"document-2402": "requires package-scoped strip-space isolation",
	"collection-006": "requires package-scoped strip-space isolation",

	// XSLT 2.0 test expects XTSE0870 for empty xsl:value-of, but XSLT 3.0 allows it.
	"select-7502a": "XSLT 2.0 test; XSLT 3.0 correctly accepts empty xsl:value-of",

	// XSLT 2.0 test expecting XTSE0090 for exponent-separator attribute.
	// Our XSLT 3.0 processor correctly recognizes exponent-separator.
	"format-number-069c": "XSLT 2.0 test; 3.0 processor recognizes exponent-separator",

	// Stylesheet defines f:format-number with 3 params in a custom namespace,
	// but the XPath evaluator resolves to built-in fn:format-number (arity mismatch).
	"format-number-070": "user-defined function shadows built-in format-number; arity mismatch",

	// Requires external Unicode Consortium NormalizationTest.txt file not included in test suite.
	"normalize-unicode-008": "missing external fixture NormalizationTest.txt",

	// XSD 1.0 variant expects xs:dateTimeStamp unavailable; our processor
	// targets XSD 1.1 where it is available. The 0151a variant tests XSD 1.1
	// behavior but is gated behind the XSD_1.1 feature flag.
	"type-available-0151": "XSD 1.0 test; our processor targets XSD 1.1 (xs:dateTimeStamp is available)",

}

// generatedAssetSourceAliases maps stale upstream asset paths to the source file
// that should be copied into the generated testdata tree.
var generatedAssetSourceAliases = map[string]string{
	"tests/decl/import-schema/variousTypesSchemaInline.xsd": "tests/decl/import-schema/schema004.xsd",
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
	assetFiles["catalog.xml"] = struct{}{}

	for _, tsRef := range cat.TestSets {
		// The "catalog" test set contains only meta-tests that validate the
		// W3C test catalog itself (schema checks, name consistency, etc.).
		// They require the full set of test-set XML files which we don't ship.
		if tsRef.Name == "catalog" {
			continue
		}

		tsFile := filepath.Join(sourceDir, tsRef.File)
		ts := parseTestSet(tsFile)
		tsDir := filepath.Dir(tsRef.File)           // e.g. "tests/insn/apply-templates"
		tsDirAbs := filepath.Join(sourceDir, tsDir) // absolute path for reading files
		addCatalogReferencedFiles(assetFiles, tsDir, ts)

		// Some test sets reference non-XML files (JSON, text, etc.) at runtime
		// via unparsed-text(). Copy the entire directory tree for these.
		switch tsRef.Name {
		case "regex-classes", "json-to-xml", "xml-to-json", "unparsed-text",
			"base-uri", "resolve-uri", "document", "accumulator":
			if err := addAssetTree(assetFiles, sourceDir, tsDir); err != nil {
				log.Fatalf("collecting %s assets: %v", tsRef.Name, err)
			}
		}

		localEnvs := make(map[string]*xslEnvironment)
		for i := range ts.Environments {
			localEnvs[ts.Environments[i].Name] = &ts.Environments[i]
			// Track environment-level stylesheet and source assets
			for _, ss := range ts.Environments[i].Stylesheets {
				if ss.File != "" {
					relPath := filepath.Join(tsDir, ss.File)
					assetFiles[relPath] = struct{}{}
					absPath := filepath.Join(sourceDir, relPath)
					for _, dep := range collectTransitiveDeps(absPath) {
						relDep, err := filepath.Rel(sourceDir, dep)
						if err == nil {
							assetFiles[relDep] = struct{}{}
						}
					}
				}
			}
			// Track environment-level package assets
			for _, pkg := range ts.Environments[i].Packages {
				if pkg.File != "" {
					relPath := filepath.Join(tsDir, pkg.File)
					assetFiles[relPath] = struct{}{}
					absPath := filepath.Join(sourceDir, relPath)
					for _, dep := range collectTransitiveDeps(absPath) {
						relDep, err := filepath.Rel(sourceDir, dep)
						if err == nil {
							assetFiles[relDep] = struct{}{}
						}
					}
				}
			}
			for _, src := range ts.Environments[i].Sources {
				if src.File != "" {
					relPath := filepath.Join(tsDir, src.File)
					assetFiles[relPath] = struct{}{}
				}
			}
			for _, sch := range ts.Environments[i].Schemas {
				if sch.File != "" {
					relPath := filepath.Join(tsDir, sch.File)
					assetFiles[relPath] = struct{}{}
				}
			}
			for _, col := range ts.Environments[i].Collections {
				for _, src := range col.Sources {
					if src.File == "" {
						continue
					}
					relPath := filepath.Join(tsDir, src.File)
					assetFiles[relPath] = struct{}{}
				}
			}
		}

		// Determine category from catalog path directory
		cat := categoryFromCatalogPath(tsRef.File)
		setSkip := getCategorySkipReason(cat)
		if setSkip == "" {
			setSkip = getSetSkipReason(tsRef.Name, ts.Dependencies)
		}

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
			// Apply strm unlock limit: only the first N strm tests run.
			if skipReason == "" && cat == "strm" {
				strmUnlocked++
				if strmUnlockLimit > 0 && strmUnlocked > strmUnlockLimit {
					skipReason = "streaming test suite disabled"
				}
			}
			gt.Skip = skipReason

			// Extract on-multiple-match dependency
			gt.OnMultipleMatch = getOnMultipleMatch(ts.Dependencies, tc.Dependencies)

			// Extract package version resolution dependency
			gt.VersionResolution = getVersionResolution(ts.Dependencies, tc.Dependencies)

			// Find primary and secondary stylesheets
			for _, ss := range tc.Test.Stylesheets {
				relPath := filepath.Join(tsDir, ss.File)
				if ss.Role == "secondary" {
					gt.SecondaryStylesheets = append(gt.SecondaryStylesheets, relPath)
				} else {
					gt.StylesheetPath = relPath
				}
			}
			// Find primary and secondary packages from test-level.
			// Process packages before the single-stylesheet fallback so
			// that a principal package takes priority.
			for _, pkg := range tc.Test.Packages {
				relPath := filepath.Join(tsDir, pkg.File)
				if pkg.Role == "principal" {
					if gt.StylesheetPath == "" {
						gt.StylesheetPath = relPath
					}
				} else {
					gt.PackageDeps = append(gt.PackageDeps, packageDep{
						URI:            pkg.URI,
						PackageVersion: pkg.PackageVersion,
						FilePath:       relPath,
					})
				}
				assetFiles[relPath] = struct{}{}
				absPath := filepath.Join(sourceDir, relPath)
				for _, dep := range collectTransitiveDeps(absPath) {
					relDep, err := filepath.Rel(sourceDir, dep)
					if err == nil {
						assetFiles[relDep] = struct{}{}
					}
				}
			}

			// If only one stylesheet and no explicit role, it's the primary
			if gt.StylesheetPath == "" && len(tc.Test.Stylesheets) == 1 {
				gt.StylesheetPath = filepath.Join(tsDir, tc.Test.Stylesheets[0].File)
			}

			// Fall back to environment-level stylesheet/packages if test has none
			if gt.StylesheetPath == "" {
				env := resolveEnvironment(tc.Environment, localEnvs)
				if env != nil {
					// Check if source document defines the stylesheet
					// (embedded via <?xml-stylesheet?> PI).
					sourceDefinesStylesheet := false
					for _, src := range env.Sources {
						if src.Role == "." && src.DefinesStylesheet == "true" {
							sourceDefinesStylesheet = true
							gt.EmbeddedStylesheet = true
							break
						}
					}
					for _, ss := range env.Stylesheets {
						relPath := filepath.Join(tsDir, ss.File)
						if ss.Role == "secondary" {
							gt.SecondaryStylesheets = append(gt.SecondaryStylesheets, relPath)
						} else {
							gt.StylesheetPath = relPath
						}
					}
					// Only promote a lone environment stylesheet to primary
					// when the source document does not define the stylesheet.
					if gt.StylesheetPath == "" && len(env.Stylesheets) == 1 && !sourceDefinesStylesheet {
						gt.StylesheetPath = filepath.Join(tsDir, env.Stylesheets[0].File)
					}
					// Environment-level packages as principal fallback
					for _, pkg := range env.Packages {
						relPath := filepath.Join(tsDir, pkg.File)
						if pkg.Role == "principal" && gt.StylesheetPath == "" {
							gt.StylesheetPath = relPath
						}
					}
				}
			}

			// Collect environment-level secondary packages (always, even if stylesheet is found)
			{
				env := resolveEnvironment(tc.Environment, localEnvs)
				if env != nil {
					for _, pkg := range env.Packages {
						if pkg.Role == "secondary" || pkg.Role == "" {
							relPath := filepath.Join(tsDir, pkg.File)
							gt.PackageDeps = append(gt.PackageDeps, packageDep{
								URI:            pkg.URI,
								PackageVersion: pkg.PackageVersion,
								FilePath:       relPath,
							})
							assetFiles[relPath] = struct{}{}
							absPath := filepath.Join(sourceDir, relPath)
							for _, dep := range collectTransitiveDeps(absPath) {
								relDep, err := filepath.Rel(sourceDir, dep)
								if err == nil {
									assetFiles[relDep] = struct{}{}
								}
							}
						}
					}
				}
			}

			if gt.StylesheetPath == "" && !gt.EmbeddedStylesheet {
				gt.Skip = "no stylesheet"
			}
			// Initial template
			if tc.Test.InitialTemplate != nil {
				gt.InitialTemplate = resolveInitialTemplateName(tc.Test.InitialTemplate)
				for _, p := range tc.Test.InitialTemplate.Params {
					// Resolve QName using namespace declarations from the param element itself
					resolvedName := resolveQNameWithAttrs(p.Name, p.Attrs)
					if p.Tunnel == "yes" {
						if gt.InitialTemplateTunnelParams == nil {
							gt.InitialTemplateTunnelParams = make(map[string]string)
						}
						gt.InitialTemplateTunnelParams[resolvedName] = p.Select
					} else {
						if gt.InitialTemplateParams == nil {
							gt.InitialTemplateParams = make(map[string]string)
						}
						gt.InitialTemplateParams[p.Name] = p.Select
					}
				}
			}

			// Initial mode
			if tc.Test.InitialMode != nil {
				if tc.Test.InitialMode.Name != "" {
					gt.InitialMode = resolveQNameWithAttrs(tc.Test.InitialMode.Name, tc.Test.InitialMode.Attrs)
				}
				if tc.Test.InitialMode.Select != "" {
					gt.InitialModeSelect = tc.Test.InitialMode.Select
				}
				for _, p := range tc.Test.InitialMode.Params {
					name := resolveQNameWithAttrs(p.Name, p.Attrs)
					if boolAttrTrue(p.Tunnel) {
						if gt.InitialModeTunnelParams == nil {
							gt.InitialModeTunnelParams = make(map[string]string)
						}
						gt.InitialModeTunnelParams[name] = p.Select
					} else {
						if gt.InitialModeParams == nil {
							gt.InitialModeParams = make(map[string]string)
						}
						gt.InitialModeParams[name] = p.Select
					}
				}
			}

			// Initial function
			if tc.Test.InitialFunction != nil {
				gt.InitialFunction = resolveQNameWithAttrs(tc.Test.InitialFunction.Name, tc.Test.InitialFunction.Attrs)
				for _, p := range tc.Test.InitialFunction.Params {
					gt.InitialFunctionParams = append(gt.InitialFunctionParams, p.Select)
				}
			}

			// Params
			if len(tc.Test.Params) > 0 {
				gt.Params = make(map[string]string)
				for _, p := range tc.Test.Params {
					gt.Params[p.Name] = p.Select
					if p.As != "" {
						if gt.ParamTypes == nil {
							gt.ParamTypes = make(map[string]string)
						}
						gt.ParamTypes[p.Name] = p.As
					}
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
					} else if xmlContent := extractParseXMLContent(src.Select); xmlContent != "" {
						gt.SourceContent = xmlContent
					}
					if src.DefinesStylesheet == "true" {
						gt.EmbeddedStylesheet = true
					}
					// When source has a select attribute (e.g., select="/doc"),
					// use it as the initial match selection so the transformation
					// starts at the selected node rather than the root.
					if src.Select != "" && gt.InitialModeSelect == "" {
						if extractParseXMLContent(src.Select) == "" {
							gt.InitialModeSelect = src.Select
						}
					}
				}
				for _, col := range env.Collections {
					def := collectionDef{URI: col.URI}
					for _, src := range col.Sources {
						if src.File == "" {
							continue
						}
						relPath := filepath.Join(tsDir, src.File)
						def.DocPaths = append(def.DocPaths, relPath)
						assetFiles[normalizeAssetPath(relPath)] = struct{}{}
					}
					gt.Collections = append(gt.Collections, def)
				}
				// Extract schema for source document validation.
				// Use the first schema with role="source-reference" or no role
				// (not "stylesheet-import" which is for xsl:import-schema).
				for _, sch := range env.Schemas {
					if sch.File == "" {
						continue
					}
					relPath := filepath.Join(tsDir, sch.File)
					assetFiles[relPath] = struct{}{}
					if sch.Role == "stylesheet-import" {
						gt.ImportSchemaPaths = append(gt.ImportSchemaPaths, relPath)
					} else {
						// Default/empty role: source schema AND import schema
						if gt.SourceSchemaPath == "" {
							gt.SourceSchemaPath = relPath
						}
						gt.ImportSchemaPaths = append(gt.ImportSchemaPaths, relPath)
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
			if gt.Skip == "" {
				if reason, ok := knownSkips[gt.CaseName]; ok {
					gt.Skip = reason
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

			if isExcludedTestCase(gt.CaseName) {
				continue
			}

			allTests = append(allTests, gt)
		}
	}

	// Copy asset files
	copied := 0
	for relPath := range assetFiles {
		normalizedRelPath := normalizeAssetPath(relPath)
		srcFull := filepath.Join(sourceDir, resolvedAssetSourcePath(normalizedRelPath))
		dstFull := filepath.Join(assetsDir, normalizedRelPath)
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

func addCatalogReferencedFiles(assetFiles map[string]struct{}, tsDir string, ts *xslTestSetFile) {
	for _, env := range ts.Environments {
		addCatalogFileRefs(assetFiles, tsDir, env.Stylesheets, env.Packages)
	}

	for _, tc := range ts.TestCases {
		if tc.Environment != nil {
			addCatalogFileRefs(assetFiles, tsDir, tc.Environment.Stylesheets, tc.Environment.Packages)
		}
		addCatalogFileRefs(assetFiles, tsDir, tc.Test.Stylesheets, tc.Test.Packages)
	}
}

func addCatalogFileRefs(assetFiles map[string]struct{}, tsDir string, stylesheets []xslStylesheet, packages []xslPackage) {
	for _, ss := range stylesheets {
		if ss.File == "" {
			continue
		}
		assetFiles[normalizeAssetPath(filepath.Join(tsDir, ss.File))] = struct{}{}
	}

	for _, pkg := range packages {
		if pkg.File == "" {
			continue
		}
		assetFiles[normalizeAssetPath(filepath.Join(tsDir, pkg.File))] = struct{}{}
	}
}

// ──────────────────────────────────────────────────────────────────────
// Spec & feature filtering
// ──────────────────────────────────────────────────────────────────────

func getSetSkipReason(name string, deps *xslDependencies) string {
	switch name {
	case "load-xquery-module":
		return "requires XQuery load-xquery-module"
	}
	if deps != nil {
		if reason := getDepsSkipReason(deps); reason != "" {
			return reason
		}
	}
	return ""
}

// strmUnlockLimit controls how many strm tests are unlocked (0 = all).
const strmUnlockLimit = 2542

var strmUnlocked int

func getCategorySkipReason(category string) string {
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
		case "year_component_values":
			if d.Satisfied == "false" {
				// Test requires this year-value capability to be absent.
				if yearComponentValueSupported(d.Value) {
					return fmt.Sprintf("year component value present but test requires absent: %s", d.Value)
				}
				continue
			}
			if !yearComponentValueSupported(d.Value) {
				return fmt.Sprintf("unsupported year component value: %s", d.Value)
			}
		case "on-multiple-match":
			// Handled separately via getOnMultipleMatch
		case "package_version_resolution":
			// Handled separately via getVersionResolution
		case "enable_assertions":
			// We evaluate assertions; skip tests that require them disabled
			if d.Satisfied == "false" {
				return "test requires assertions disabled; we evaluate assertions"
			}
		}
	}
	return ""
}

// getOnMultipleMatch extracts the on-multiple-match dependency value from
// test-set and test-case dependencies. Returns "fail" when the test requires
// error behavior, empty string otherwise.
func getOnMultipleMatch(setDeps *xslDependencies, caseDeps *xslDependencies) string {
	check := func(deps *xslDependencies) string {
		if deps == nil {
			return ""
		}
		for _, d := range deps.Children {
			if d.XMLName.Local == "on-multiple-match" {
				if d.Value == "error" {
					return "fail"
				}
			}
		}
		return ""
	}
	if v := check(caseDeps); v != "" {
		return v
	}
	return check(setDeps)
}

// getVersionResolution extracts the package_version_resolution dependency
// from test-set and test-case dependencies. Returns "lowest" when the test
// requires lowest_version selection, empty string otherwise.
func getVersionResolution(setDeps *xslDependencies, caseDeps *xslDependencies) string {
	check := func(deps *xslDependencies) string {
		if deps == nil {
			return ""
		}
		for _, d := range deps.Children {
			if d.XMLName.Local == "package_version_resolution" {
				if d.Value == "lowest_version" {
					return "lowest"
				}
			}
		}
		return ""
	}
	if v := check(caseDeps); v != "" {
		return v
	}
	return check(setDeps)
}

func specSupported(spec string) bool {
	for _, s := range strings.Fields(spec) {
		switch s {
		// Our processor is XSLT 3.0.
		// "X+" means "version X or later" — we satisfy 1.0+, 2.0+, 3.0+.
		// "X" without "+" means exactly that version — skip 1.0 and 2.0 only tests.
		case "XSLT10+", "XSLT20+", "XSLT30", "XSLT30+":
			return true
		}
	}
	return false
}

// isExcludedTestCase returns true for test cases that should not be generated at all.
func isExcludedTestCase(name string) bool {
	// The unicode-90 suite is computationally expensive and serializes the
	// entire xslt3 package test run behind one non-parallel top-level test.
	if strings.HasPrefix(name, "unicode90-") {
		return true
	}
	return false
}

func featureSupported(feature string) bool {
	switch feature {
	case
		"backwards_compatibility",
		"Saxon-PE", "Saxon-EE":
		return false
	}
	return true
}

func yearComponentValueSupported(value string) bool {
	switch value {
	case "support negative year", "support year above 9999":
		return true
	}
	return false
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
	return resolveQNameWithAttrs(it.Name, it.Attrs)
}

func resolveQNameWithAttrs(name string, attrs []xml.Attr) string {
	if name == "" {
		return ""
	}
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		for _, attr := range attrs {
			if attr.Name.Space == "xmlns" && attr.Name.Local == prefix {
				return helium.ClarkName(attr.Value, local)
			}
			if attr.Name.Space == "" && attr.Name.Local == "xmlns" && prefix == "" {
				return helium.ClarkName(attr.Value, local)
			}
		}
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
		// xsl:source-document → collect literal href as data dependency
		if isXSLT && se.Name.Local == "source-document" {
			for _, attr := range se.Attr {
				if attr.Name.Local == "href" && attr.Value != "" && !strings.Contains(attr.Value, "{") {
					depPath := filepath.Join(dir, normalizeAssetPath(attr.Value))
					*result = append(*result, depPath)
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
		a.URI = xa.URI
		for _, child := range xa.Children {
			a.Children = append(a.Children, convertAssertion(child, tsDir))
		}
	case "assert-serialization":
		a.Method = xa.Method
		if xa.File != "" {
			outPath := filepath.Join(tsDir, xa.File)
			data, err := os.ReadFile(outPath)
			if err != nil {
				a.Value = fmt.Sprintf("ERROR: cannot read %s: %v", xa.File, err)
			} else if !utf8.Valid(data) {
				// Non-UTF-8 serialization output cannot be embedded in Go source.
				a.Type = "assert-posture-and-sweep"
				a.Value = "skip: non-UTF-8 serialization comparison"
			} else {
				a.Value = string(data)
			}
		} else {
			a.Value = decodeXMLText(string(xa.Inner))
		}
	case "assert-serialization-error":
		a.Type = "error"
		a.Value = xa.Code
	case "serialization-matches":
		a.Value = decodeXMLText(string(xa.Inner))
		a.Flags = xa.Flags
	case "assert-type", "assert-count", "assert-deep-eq", "assert-empty", "assert-eq":
		a.Value = decodeXMLText(string(xa.Inner))
	case "not":
		for _, child := range xa.Children {
			a.Children = append(a.Children, convertAssertion(child, tsDir))
		}
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
		} else {
			// Mixed any-of: collect error codes as accepted alternatives.
			for _, child := range gt.Assertions[0].Children {
				if child.Type == "error" && child.Value != "" {
					gt.AcceptErrors = append(gt.AcceptErrors, child.Value)
				}
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
	if gt.ErrorCode == "XXXX9999" {
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
	// Schema-aware features are now unlocked for testing.
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

			if len(tc.PackageDeps) > 0 {
				b.WriteString(", PackageDeps: []w3cPackageDep{")
				for i, pd := range tc.PackageDeps {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "{URI: %q, Version: %q, FilePath: %q}", pd.URI, pd.PackageVersion, pd.FilePath)
				}
				b.WriteString("}")
			}

			if tc.SourceDocPath != "" {
				fmt.Fprintf(&b, ", SourceDocPath: %q", tc.SourceDocPath)
			}
			if tc.SourceSchemaPath != "" {
				fmt.Fprintf(&b, ", SourceSchemaPath: %q", tc.SourceSchemaPath)
			}
			if len(tc.ImportSchemaPaths) > 0 {
				b.WriteString(", ImportSchemaPaths: []string{")
				for i, p := range tc.ImportSchemaPaths {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q", p)
				}
				b.WriteString("}")
			}
			if tc.SourceContent != "" {
				fmt.Fprintf(&b, ", SourceContent: %s", goStringLiteral(tc.SourceContent))
			}
			if tc.InitialTemplate != "" {
				fmt.Fprintf(&b, ", InitialTemplate: %q", tc.InitialTemplate)
			}
			if len(tc.InitialTemplateParams) > 0 {
				b.WriteString(", InitialTemplateParams: map[string]string{")
				keys := sortedStringKeys(tc.InitialTemplateParams)
				for i, k := range keys {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q: %s", k, goStringLiteral(tc.InitialTemplateParams[k]))
				}
				b.WriteString("}")
			}
			if len(tc.InitialTemplateTunnelParams) > 0 {
				b.WriteString(", InitialTemplateTunnelParams: map[string]string{")
				keys := sortedStringKeys(tc.InitialTemplateTunnelParams)
				for i, k := range keys {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q: %s", k, goStringLiteral(tc.InitialTemplateTunnelParams[k]))
				}
				b.WriteString("}")
			}
			if tc.InitialMode != "" {
				fmt.Fprintf(&b, ", InitialMode: %q", tc.InitialMode)
			}
			if tc.InitialModeSelect != "" {
				fmt.Fprintf(&b, ", InitialModeSelect: %q", tc.InitialModeSelect)
			}
			if len(tc.InitialModeParams) > 0 {
				b.WriteString(", InitialModeParams: map[string]string{")
				keys := sortedStringKeys(tc.InitialModeParams)
				for i, k := range keys {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q: %s", k, goStringLiteral(tc.InitialModeParams[k]))
				}
				b.WriteString("}")
			}
			if len(tc.InitialModeTunnelParams) > 0 {
				b.WriteString(", InitialModeTunnelParams: map[string]string{")
				keys := sortedStringKeys(tc.InitialModeTunnelParams)
				for i, k := range keys {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q: %s", k, goStringLiteral(tc.InitialModeTunnelParams[k]))
				}
				b.WriteString("}")
			}
			if tc.InitialFunction != "" {
				fmt.Fprintf(&b, ", InitialFunction: %q", tc.InitialFunction)
			}
			if len(tc.InitialFunctionParams) > 0 {
				b.WriteString(", InitialFunctionParams: []string{")
				for i, p := range tc.InitialFunctionParams {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%s", goStringLiteral(p))
				}
				b.WriteString("}")
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
			if len(tc.ParamTypes) > 0 {
				b.WriteString(", ParamTypes: map[string]string{")
				keys := sortedStringKeys(tc.ParamTypes)
				for i, k := range keys {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q: %q", k, tc.ParamTypes[k])
				}
				b.WriteString("}")
			}
			if len(tc.Collections) > 0 {
				b.WriteString(", Collections: []w3cCollection{")
				for i, col := range tc.Collections {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "{URI: %q, DocPaths: []string{", col.URI)
					for j, docPath := range col.DocPaths {
						if j > 0 {
							b.WriteString(", ")
						}
						fmt.Fprintf(&b, "%q", docPath)
					}
					b.WriteString("}}")
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
			if len(tc.AcceptErrors) > 0 {
				b.WriteString(", AcceptErrors: []string{")
				for i, code := range tc.AcceptErrors {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%q", code)
				}
				b.WriteString("}")
			}
			if tc.EmbeddedStylesheet {
				b.WriteString(", EmbeddedStylesheet: true")
			}
			if tc.Skip != "" {
				fmt.Fprintf(&b, ", Skip: %q", tc.Skip)
			}
			if tc.OnMultipleMatch != "" {
				fmt.Fprintf(&b, ", OnMultipleMatch: %q", tc.OnMultipleMatch)
			}
			if tc.VersionResolution != "" {
				fmt.Fprintf(&b, ", VersionResolution: %q", tc.VersionResolution)
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
	// Filter out w3cAssertSkip() when real assertions exist alongside.
	// This handles W3C tests with mixed result alternatives (e.g.
	// assert-xml + assert-posture-and-sweep): the unsupported assertion
	// would cause a t.Skip that hides the passing real assertion.
	hasReal := false
	for _, e := range out {
		if e != "w3cAssertSkip()" {
			hasReal = true
			break
		}
	}
	if hasReal {
		filtered := out[:0]
		for _, e := range out {
			if e != "w3cAssertSkip()" {
				filtered = append(filtered, e)
			}
		}
		out = filtered
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
	case "assert-result-document":
		var childChecks []string
		for _, child := range a.Children {
			childChecks = append(childChecks, emitCheck(child))
		}
		if len(childChecks) == 0 {
			return []string{"w3cAssertSkip()"}
		}
		return []string{fmt.Sprintf("w3cAssertResultDocument(%q, %s)", a.URI, strings.Join(childChecks, ", "))}
	case "assert-serialization":
		return []string{fmt.Sprintf("w3cAssertSerialization(%q, %s)", a.Method, goStringLiteral(a.Value))}
	case "serialization-matches":
		pattern := buildSerializationMatchesPattern(a.Value, a.Flags)
		return []string{fmt.Sprintf("w3cAssertSerializationMatches(%s)", goStringLiteral(pattern))}
	case "assert-type":
		return []string{fmt.Sprintf("w3cAssertType(%s)", goStringLiteral(strings.TrimSpace(a.Value)))}
	case "assert-count":
		return []string{fmt.Sprintf("w3cAssertCount(%s)", strings.TrimSpace(a.Value))}
	case "assert-deep-eq":
		return []string{fmt.Sprintf("w3cAssertDeepEq(%s)", goStringLiteral(strings.TrimSpace(a.Value)))}
	case "assert-empty":
		return []string{"w3cAssertEmpty()"}
	case "assert-eq":
		return []string{fmt.Sprintf("w3cAssertEq(%s)", goStringLiteral(strings.TrimSpace(a.Value)))}
	case "not":
		var checks []string
		for _, child := range a.Children {
			checks = append(checks, emitCheck(child))
		}
		if len(checks) == 0 {
			return []string{"w3cAssertSkip()"}
		}
		return []string{fmt.Sprintf("w3cAssertNot(%s)", strings.Join(checks, ", "))}
	case "assert-posture-and-sweep":
		return []string{"w3cAssertSkip()"}
	default:
		return []string{"w3cAssertSkip()"}
	}
}

// buildSerializationMatchesPattern converts a W3C serialization-matches pattern
// with optional flags into a Go regexp pattern. The W3C catalog uses flag "i"
// for case-insensitive matching and "x" for extended mode; we prepend them as
// Go inline flags ((?i), etc.). The (?s) flag (dot-matches-newline) is always
// added since serialized output is multi-line.
func buildSerializationMatchesPattern(pattern, flags string) string {
	if flags == "" {
		return pattern
	}
	// Map W3C flags to Go regexp inline flags.
	// "i" → case insensitive, "x" → extended (strip unescaped whitespace).
	var goFlags strings.Builder
	hasX := false
	for _, f := range flags {
		switch f {
		case 'i':
			goFlags.WriteRune('i')
		case 'm':
			goFlags.WriteRune('m')
		case 's':
			goFlags.WriteRune('s')
		case 'x':
			hasX = true
		}
	}
	// Extended mode ("x"): strip unescaped whitespace and #-comments from pattern.
	// Go regexp does not support (?x), so we apply the transformation manually.
	if hasX {
		pattern = stripExtendedWhitespace(pattern)
	}
	if goFlags.Len() == 0 {
		return pattern
	}
	return "(?" + goFlags.String() + ")" + pattern
}

// stripExtendedWhitespace removes unescaped whitespace and #-comments from
// a regex pattern, emulating the XPath/Perl "x" flag. Whitespace inside
// character classes [...] is preserved.
func stripExtendedWhitespace(pattern string) string {
	var b strings.Builder
	inCharClass := false
	i := 0
	for i < len(pattern) {
		ch := pattern[i]
		if ch == '\\' && i+1 < len(pattern) {
			// Escaped character — keep as-is.
			b.WriteByte(ch)
			b.WriteByte(pattern[i+1])
			i += 2
			continue
		}
		if ch == '[' && !inCharClass {
			inCharClass = true
			b.WriteByte(ch)
			i++
			continue
		}
		if ch == ']' && inCharClass {
			inCharClass = false
			b.WriteByte(ch)
			i++
			continue
		}
		if !inCharClass {
			if ch == '#' {
				// Skip #-comment to end of line.
				for i < len(pattern) && pattern[i] != '\n' {
					i++
				}
				continue
			}
			if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
				i++
				continue
			}
		}
		b.WriteByte(ch)
		i++
	}
	return b.String()
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
	case "assert-serialization":
		return fmt.Sprintf("w3cCheckSerialization(%q, %s)", a.Method, goStringLiteral(a.Value))
	case "serialization-matches":
		pattern := buildSerializationMatchesPattern(a.Value, a.Flags)
		return fmt.Sprintf("w3cCheckSerializationMatches(%s)", goStringLiteral(pattern))
	case "all-of":
		var checks []string
		for _, child := range a.Children {
			checks = append(checks, emitCheck(child))
		}
		return fmt.Sprintf("w3cCheckAllOf(%s)", strings.Join(checks, ", "))
	case "any-of":
		var checks []string
		for _, child := range a.Children {
			checks = append(checks, emitCheck(child))
		}
		return fmt.Sprintf("w3cCheckAnyOf(%s)", strings.Join(checks, ", "))
	case "assert-type":
		return fmt.Sprintf("w3cCheckType(%s)", goStringLiteral(strings.TrimSpace(a.Value)))
	case "assert-count":
		return fmt.Sprintf("w3cCheckCount(%s)", strings.TrimSpace(a.Value))
	case "assert-deep-eq":
		return fmt.Sprintf("w3cCheckDeepEq(%s)", goStringLiteral(strings.TrimSpace(a.Value)))
	case "assert-empty":
		return "w3cCheckEmpty()"
	case "assert-eq":
		return fmt.Sprintf("w3cCheckEq(%s)", goStringLiteral(strings.TrimSpace(a.Value)))
	case "not":
		var checks []string
		for _, child := range a.Children {
			checks = append(checks, emitCheck(child))
		}
		if len(checks) == 0 {
			return "w3cCheckSkip()"
		}
		return fmt.Sprintf("w3cCheckNot(%s)", strings.Join(checks, ", "))
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

// extractParseXMLContent extracts the XML string from a select expression
// of the form parse-xml('...') used in W3C test catalog environments.
func extractParseXMLContent(sel string) string {
	sel = strings.TrimSpace(sel)
	const prefix = "parse-xml('"
	if !strings.HasPrefix(sel, prefix) {
		return ""
	}
	rest := sel[len(prefix):]
	end := strings.LastIndex(rest, "')")
	if end < 0 {
		return ""
	}
	return strings.ReplaceAll(rest[:end], "''", "'")
}

var nonIdentRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func goIdentifier(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return nonIdentRE.ReplaceAllString(s, "_")
}

func goStringLiteral(s string) string {
	if strings.Contains(s, "\n") && !strings.Contains(s, "`") && !strings.Contains(s, "\r") && utf8.ValidString(s) {
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
	// Allow override via XSLT3GEN_ROOT env var for worktree scenarios
	// where os.Getwd() may resolve through symlinks.
	if root := os.Getenv("XSLT3GEN_ROOT"); root != "" {
		return root
	}
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

func resolvedAssetSourcePath(relPath string) string {
	if src, ok := generatedAssetSourceAliases[relPath]; ok {
		return src
	}
	return relPath
}

func addAssetTree(assetFiles map[string]struct{}, rootDir, relDir string) error {
	absDir := filepath.Join(rootDir, relDir)
	return filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(filepath.Base(relPath), "_") {
			return nil
		}
		assetFiles[relPath] = struct{}{}
		return nil
	})
}

func normalizeAssetPath(path string) string {
	base, _, _ := strings.Cut(path, "#")
	return base
}

func boolAttrTrue(v string) bool {
	switch strings.TrimSpace(v) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
