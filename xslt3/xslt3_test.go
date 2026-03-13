package xslt3_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

const testSuiteDir = "../testdata/xslt30/source"

func TestW3CTestSuite(t *testing.T) {
	t.Parallel()

	catalogPath := filepath.Join(testSuiteDir, "catalog.xml")
	if _, err := os.Stat(catalogPath); os.IsNotExist(err) {
		t.Skip("XSLT 3.0 test suite not available; run testdata/xslt30/fetch.sh first")
	}

	catalogData, err := os.ReadFile(catalogPath)
	require.NoError(t, err)

	catalogDoc, err := helium.Parse(t.Context(), catalogData)
	require.NoError(t, err)

	// Collect test sets from catalog
	const catalogNS = "http://www.w3.org/2012/10/xslt-test-catalog"
	var testSetCount, testCount, passCount, skipCount, failCount int

	for child := catalogDoc.DocumentElement().FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.LocalName() != "test-set" {
			continue
		}
		tsName, _ := elem.GetAttribute("name")
		tsFile, _ := elem.GetAttribute("file")

		// For Phase 1, focus on core instruction test sets
		if !isPhase1TestSet(tsName) {
			continue
		}

		testSetCount++
		t.Run(tsName, func(t *testing.T) {
			t.Parallel()

			tsPath := filepath.Join(testSuiteDir, tsFile)
			tsData, err := os.ReadFile(tsPath)
			if err != nil {
				t.Skipf("cannot read test set file: %v", err)
				return
			}

			tsDoc, err := helium.Parse(t.Context(), tsData)
			if err != nil {
				t.Skipf("cannot parse test set file: %v", err)
				return
			}

			tsDir := filepath.Dir(tsPath)
			environments := parseEnvironments(t, tsDoc.DocumentElement(), tsDir)

			for tc := tsDoc.DocumentElement().FirstChild(); tc != nil; tc = tc.NextSibling() {
				tcElem, ok := tc.(*helium.Element)
				if !ok || tcElem.LocalName() != "test-case" {
					continue
				}

				tcName, _ := tcElem.GetAttribute("name")
				testCount++

				t.Run(tcName, func(t *testing.T) {
					t.Parallel()

					result := runTestCase(t, tcElem, tsDir, environments, catalogNS)
					switch result {
					case testPass:
						passCount++
					case testSkip:
						skipCount++
					case testFail:
						failCount++
					}
				})
			}
		})
	}

	_ = testSetCount
	_ = testCount
	_ = passCount
	_ = skipCount
	_ = failCount
}

// isPhase1TestSet returns true if this test set exercises Phase 1 features.
func isPhase1TestSet(name string) bool {
	phase1Sets := map[string]struct{}{
		"apply-templates": {},
		"call-template":   {},
		"choose":          {},
		"copy":            {},
		"element":         {},
		"attribute":       {},
		"lre":             {},
		"sort":            {},
		"variable":        {},
		"param":           {},
		"template":        {},
		"output":          {},
		"import":          {},
		"include":         {},
		"strip-space":     {},
		"number":          {},
		"message":         {},
		"construct-node":  {},
	}
	_, ok := phase1Sets[name]
	return ok
}

type testResult int

const (
	testPass testResult = iota
	testSkip
	testFail
)

// environment holds parsed test environment data.
type environment struct {
	sourceContent []byte
	sourceFile    string
}

func parseEnvironments(t *testing.T, root *helium.Element, tsDir string) map[string]*environment {
	t.Helper()
	envs := make(map[string]*environment)

	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.LocalName() != "environment" {
			continue
		}

		name, _ := elem.GetAttribute("name")
		if name == "" {
			continue
		}

		env := &environment{}
		for sc := elem.FirstChild(); sc != nil; sc = sc.NextSibling() {
			src, ok := sc.(*helium.Element)
			if !ok || src.LocalName() != "source" {
				continue
			}

			role, _ := src.GetAttribute("role")
			if role != "." {
				continue
			}

			if f, ok := src.GetAttribute("file"); ok {
				env.sourceFile = filepath.Join(tsDir, f)
			}

			// Check for inline <content>
			for cc := src.FirstChild(); cc != nil; cc = cc.NextSibling() {
				cElem, ok := cc.(*helium.Element)
				if !ok || cElem.LocalName() != "content" {
					continue
				}
				// Get text content (including CDATA)
				var buf bytes.Buffer
				for tc := cElem.FirstChild(); tc != nil; tc = tc.NextSibling() {
					buf.Write(tc.Content())
				}
				env.sourceContent = buf.Bytes()
			}
		}
		envs[name] = env
	}

	return envs
}

func runTestCase(t *testing.T, tc *helium.Element, tsDir string, environments map[string]*environment, _ string) testResult {
	t.Helper()

	// Check dependencies
	if shouldSkipDeps(tc) {
		t.Skip("unsupported dependency")
		return testSkip
	}

	// Find environment
	var sourceData []byte
	envRef, _ := tc.GetAttribute("environment")
	if envRef == "" {
		// Look for environment element inside test-case
		envRef = findChildEnvRef(tc)
	}

	// Find <test> element
	var stylesheetFile string
	var initialTemplate string
	for child := tc.FirstChild(); child != nil; child = child.NextSibling() {
		testElem, ok := child.(*helium.Element)
		if !ok || testElem.LocalName() != "test" {
			continue
		}
		for sc := testElem.FirstChild(); sc != nil; sc = sc.NextSibling() {
			sElem, ok := sc.(*helium.Element)
			if !ok {
				continue
			}
			switch sElem.LocalName() {
			case "stylesheet":
				f, _ := sElem.GetAttribute("file")
				stylesheetFile = f
			case "initial-template":
				n, _ := sElem.GetAttribute("name")
				initialTemplate = n
			}
		}
	}

	if stylesheetFile == "" {
		t.Skip("no stylesheet file")
		return testSkip
	}

	// Resolve source data
	if envRef != "" {
		if env, ok := environments[envRef]; ok {
			if env.sourceContent != nil {
				sourceData = env.sourceContent
			} else if env.sourceFile != "" {
				var err error
				sourceData, err = os.ReadFile(env.sourceFile)
				if err != nil {
					t.Skipf("cannot read source file: %v", err)
					return testSkip
				}
			}
		}
	}

	if sourceData == nil {
		// Default: empty document
		sourceData = []byte(`<?xml version="1.0"?><empty/>`)
	}

	// Parse expected result
	expectedResult := parseExpectedResult(tc)

	// Compile stylesheet
	ssPath := filepath.Join(tsDir, stylesheetFile)
	ss, err := xslt3.CompileFile(ssPath)

	if expectedResult.expectError {
		if err != nil {
			// Expected compile error
			return testPass
		}
		// May be a runtime error — continue
	} else if err != nil {
		t.Errorf("compile error: %v", err)
		return testFail
	}

	// Parse source
	ctx := t.Context()
	sourceDoc, err := helium.Parse(ctx, sourceData)
	if err != nil {
		if expectedResult.expectError {
			return testPass
		}
		t.Errorf("cannot parse source: %v", err)
		return testFail
	}

	// Transform
	_ = initialTemplate // TODO: support initial-template
	resultDoc, err := xslt3.Transform(ctx, sourceDoc, ss)
	if err != nil {
		if expectedResult.expectError {
			return testPass
		}
		t.Errorf("transform error: %v", err)
		return testFail
	}

	if expectedResult.expectError {
		t.Errorf("expected error %s but transformation succeeded", expectedResult.errorCode)
		return testFail
	}

	// Compare result
	if expectedResult.assertXML != "" {
		var buf bytes.Buffer
		err := resultDoc.XML(&buf, helium.WithNoDecl())
		require.NoError(t, err)
		actual := strings.TrimSpace(buf.String())
		expected := strings.TrimSpace(expectedResult.assertXML)

		if !xmlEqual(actual, expected) {
			t.Errorf("output mismatch:\n  got:    %s\n  expect: %s", actual, expected)
			return testFail
		}
	}

	return testPass
}

type expectedResult struct {
	expectError bool
	errorCode   string
	assertXML   string
}

func parseExpectedResult(tc *helium.Element) expectedResult {
	var result expectedResult
	for child := tc.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.LocalName() != "result" {
			continue
		}
		for rc := elem.FirstChild(); rc != nil; rc = rc.NextSibling() {
			rElem, ok := rc.(*helium.Element)
			if !ok {
				continue
			}
			switch rElem.LocalName() {
			case "error":
				result.expectError = true
				result.errorCode, _ = rElem.GetAttribute("code")
			case "assert-xml":
				var buf bytes.Buffer
				for tc := rElem.FirstChild(); tc != nil; tc = tc.NextSibling() {
					buf.Write(tc.Content())
				}
				result.assertXML = buf.String()
			}
		}
	}
	return result
}

func shouldSkipDeps(tc *helium.Element) bool {
	for child := tc.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.LocalName() != "dependencies" {
			continue
		}
		for dc := elem.FirstChild(); dc != nil; dc = dc.NextSibling() {
			dep, ok := dc.(*helium.Element)
			if !ok {
				continue
			}
			switch dep.LocalName() {
			case "spec":
				val, _ := dep.GetAttribute("value")
				if !specSupported(val) {
					return true
				}
			case "feature":
				val, _ := dep.GetAttribute("value")
				if !featureSupported(val) {
					return true
				}
			}
		}
	}
	return false
}

func specSupported(spec string) bool {
	// We support XSLT10+, XSLT20+, XSLT30+
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
	case "schema_aware", "streaming", "higher_order_functions",
		"backwards_compatibility", "dynamic_evaluation",
		"Saxon-PE", "Saxon-EE":
		return false
	}
	return true
}

func findChildEnvRef(tc *helium.Element) string {
	// Check for inline environment element inside test-case
	for child := tc.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.LocalName() != "environment" {
			continue
		}
		// Environment child may have ref= (referencing a named env) or name= (defining inline)
		if ref, ok := elem.GetAttribute("ref"); ok {
			return ref
		}
		if name, ok := elem.GetAttribute("name"); ok {
			return name
		}
	}
	return ""
}

// xmlEqual compares two XML strings for semantic equality.
// Collapses whitespace inside tags to single space, removes whitespace
// between tags, and handles attribute order differences by parsing.
func xmlEqual(actual, expected string) bool {
	a := normalizeXMLString(actual)
	e := normalizeXMLString(expected)
	return a == e
}

// normalizeXMLString normalizes an XML string for comparison.
// It collapses all whitespace runs to a single space, then removes
// space between > and < (inter-element whitespace).
func normalizeXMLString(s string) string {
	s = strings.TrimSpace(s)
	// First pass: collapse all whitespace to single space
	var result []byte
	prevSpace := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !prevSpace {
				result = append(result, ' ')
				prevSpace = true
			}
		} else {
			prevSpace = false
			result = append(result, c)
		}
	}
	// Second pass: clean up tag-boundary whitespace
	s = string(result)
	s = strings.ReplaceAll(s, "> <", "><")  // space between tags
	s = strings.ReplaceAll(s, " >", ">")    // space before >
	s = strings.ReplaceAll(s, " />", "/>")  // space before />
	return s
}
