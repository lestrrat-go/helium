package xslt3_test

import (
	"bytes"
	"context"
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

				t.Run(tcName, func(t *testing.T) {
					// TODO(xslt3): add a channel-based semaphore to limit concurrency.
					// The W3C XSLT test suite has thousands of test cases; unbounded
					// t.Parallel() can exhaust file descriptors and memory on CI.
					// Plan: create a package-level `var sem = make(chan struct{}, N)`
					// (e.g., N = runtime.GOMAXPROCS(0)*2), acquire before runTestCase,
					// release with defer. This preserves parallelism while bounding
					// resource usage.
					t.Parallel()

					runTestCase(t, tcElem, tsDir, environments, catalogNS)
				})
			}
		})
	}

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
				env.sourceContent = serializeChildren(cElem)
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
	// Also check for inline environment with source content directly
	if envRef == "" {
		sourceData = findInlineSourceContent(tc, tsDir)
	}

	// Find <test> element
	var stylesheetFile string
	var initialTemplate string
	var testParams map[string]string
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
				role, _ := sElem.GetAttribute("role")
				if role == "secondary" {
					continue
				}
				f, _ := sElem.GetAttribute("file")
				stylesheetFile = f
			case "initial-template":
				n, _ := sElem.GetAttribute("name")
				// Resolve QName prefix using namespace declarations on element
				if idx := strings.IndexByte(n, ':'); idx >= 0 {
					prefix := n[:idx]
					local := n[idx+1:]
					for _, ns := range sElem.Namespaces() {
						if ns.Prefix() == prefix {
							n = "{" + ns.URI() + "}" + local
							break
						}
					}
				}
				initialTemplate = n
			case "param":
				pName, _ := sElem.GetAttribute("name")
				pSelect, _ := sElem.GetAttribute("select")
				if pName != "" && pSelect != "" {
					// Only pass string literal params (surrounded by quotes)
					// Non-string params (numbers, etc.) are handled by stylesheet defaults
					if len(pSelect) >= 2 && pSelect[0] == '\'' && pSelect[len(pSelect)-1] == '\'' {
						if testParams == nil {
							testParams = make(map[string]string)
						}
						testParams[pName] = pSelect[1 : len(pSelect)-1]
					}
				}
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
	ss, err := xslt3.CompileFile(t.Context(), ssPath)

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
	if initialTemplate != "" {
		ctx = xslt3.WithInitialTemplate(ctx, initialTemplate)
	}
	for pName, pVal := range testParams {
		ctx = xslt3.WithParameter(ctx, pName, pVal)
	}
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
				result.assertXML = string(serializeChildren(rElem))
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

// findInlineSourceContent extracts source data from an inline <environment>
// element in the test-case that has no ref/name attributes.
func findInlineSourceContent(tc *helium.Element, tsDir string) []byte {
	for child := tc.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.LocalName() != "environment" {
			continue
		}
		// Only handle environments without ref/name (truly inline)
		if _, hasRef := elem.GetAttribute("ref"); hasRef {
			continue
		}
		for sc := elem.FirstChild(); sc != nil; sc = sc.NextSibling() {
			src, ok := sc.(*helium.Element)
			if !ok || src.LocalName() != "source" {
				continue
			}
			role, _ := src.GetAttribute("role")
			if role != "." {
				continue
			}
			// Check for file reference
			if f, hasFile := src.GetAttribute("file"); hasFile {
				data, err := os.ReadFile(filepath.Join(tsDir, f))
				if err == nil {
					return data
				}
			}
			// Check for inline <content>
			for cc := src.FirstChild(); cc != nil; cc = cc.NextSibling() {
				cElem, ok := cc.(*helium.Element)
				if !ok || cElem.LocalName() != "content" {
					continue
				}
				return serializeChildren(cElem)
			}
		}
	}
	return nil
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

// serializeChildren serializes all children of an element, preserving markup
// for element children (Content() only returns descendant text).
func serializeChildren(parent *helium.Element) []byte {
	var buf bytes.Buffer
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if elem, ok := child.(*helium.Element); ok {
			_ = elem.XML(&buf)
		} else {
			buf.Write(child.Content())
		}
	}
	return buf.Bytes()
}

// xmlEqual compares two XML strings for semantic equality.
// Handles whitespace normalization and attribute/namespace declaration order.
func xmlEqual(actual, expected string) bool {
	// Try DOM-based comparison first (preserves whitespace semantics)
	if domEqual(actual, expected) {
		return true
	}
	// Fall back to normalized string comparison for cases where DOM
	// parsing fails (e.g., text-only output, fragments)
	a := normalizeXMLString(actual)
	e := normalizeXMLString(expected)
	return a == e
}

// domEqual parses both strings as XML and compares the DOM trees.
// Wraps content in a synthetic root element so that multi-node fragments
// (e.g., "<a/><b/>") can be parsed.
func domEqual(a, b string) bool {
	wrapA := wrapXMLFragment(a)
	wrapB := wrapXMLFragment(b)

	docA, errA := helium.Parse(context.TODO(), []byte(wrapA))
	docB, errB := helium.Parse(context.TODO(), []byte(wrapB))
	if errA != nil || errB != nil {
		return false
	}
	return nodesEqual(docA.DocumentElement(), docB.DocumentElement())
}

// wrapXMLFragment wraps an XML string in a synthetic root element so that
// fragments with multiple top-level nodes can be parsed.
func wrapXMLFragment(s string) string {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "<?xml") {
		// Strip the XML declaration, then wrap
		if idx := strings.Index(trimmed, "?>"); idx >= 0 {
			s = trimmed[idx+2:]
		}
	}
	return "<_domEqual_root_>" + s + "</_domEqual_root_>"
}

func nodesEqual(a, b helium.Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type() != b.Type() {
		return false
	}
	switch a.Type() {
	case helium.ElementNode:
		ea := a.(*helium.Element)
		eb := b.(*helium.Element)
		if ea.LocalName() != eb.LocalName() || ea.URI() != eb.URI() {
			return false
		}
		// Compare attributes (order-independent)
		attrsA := collectAttrs(ea)
		attrsB := collectAttrs(eb)
		if len(attrsA) != len(attrsB) {
			return false
		}
		for k, v := range attrsA {
			if attrsB[k] != v {
				return false
			}
		}
		// Compare children
		childA := ea.FirstChild()
		childB := eb.FirstChild()
		for childA != nil && childB != nil {
			if !nodesEqual(childA, childB) {
				return false
			}
			childA = childA.NextSibling()
			childB = childB.NextSibling()
		}
		return childA == nil && childB == nil
	case helium.TextNode:
		return string(a.Content()) == string(b.Content())
	case helium.CommentNode:
		return string(a.Content()) == string(b.Content())
	case helium.ProcessingInstructionNode:
		return a.Name() == b.Name() && string(a.Content()) == string(b.Content())
	default:
		return string(a.Content()) == string(b.Content())
	}
}

// collectAttrs returns a map of {ns}local → value for all attributes on an element.
func collectAttrs(e *helium.Element) map[string]string {
	attrs := make(map[string]string)
	for _, attr := range e.Attributes() {
		key := "{" + attr.URI() + "}" + attr.LocalName()
		attrs[key] = attr.Value()
	}
	return attrs
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
	s = strings.ReplaceAll(s, "> <", "><") // space between tags
	s = strings.ReplaceAll(s, " >", ">")   // space before >
	s = strings.ReplaceAll(s, " />", "/>") // space before />
	return s
}
