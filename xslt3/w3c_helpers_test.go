package xslt3_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

const w3cTestdataDir = "../testdata/xslt30/testdata"

// TODO: slow streaming tests — investigate performance:
//   si-iterate-133   ~8.9s  (citygml.xml, 2849 polygons)
//   si-choose-012    ~3.3s  (big-transactions.xml, large DOM)
//   si-iterate-037   ~2.3s  (ot.xml, tokenize + iterate)
//   si-iterate-134   ~1.7s  (citygml.xml, failing)
//   si-iterate-135   ~1.7s  (citygml.xml, failing)
//   si-next-match-067      ~1.7s  (ot.xml, deep template chain)
//   si-apply-imports-068/069/070  ~1.8s  (ot.xml, import chain)
//   si-lre-904/905   ~1.0s  (ot.xml, XTSE3430 expected)

// Caches for compiled stylesheets and source file bytes, keyed by absolute path.
// These are safe for concurrent use because sync.Map handles its own locking.
var (
	w3cStylesheetCache sync.Map // path → *xslt3.Stylesheet
	w3cSourceBytesCache sync.Map // path → []byte
)

// w3cTest describes a single W3C XSLT 3.0 test case.
type w3cTest struct {
	Name                 string
	StylesheetPath       string
	SecondaryStylesheets []string
	SourceDocPath        string
	SourceContent        string
	InitialTemplate      string
	Params               map[string]string
	ExpectError          bool
	ErrorCode            string
	Assertions           []w3cAssertion
	Skip                 string
}

// w3cAssertion is an assertion to check against the transform result.
type w3cAssertion struct {
	Type  string // "assert-xml", "assert-string-value", "any-of", "assert-message", "skip"
	Value string
	Check func(t *testing.T, result string, messages []string) bool
}

// w3cCheck is used inside any-of assertions.
type w3cCheck struct {
	fn func(result string, messages []string) bool
}

// ──────────────────────────────────────────────────────────────────────
// Assertion constructors
// ──────────────────────────────────────────────────────────────────────

func w3cAssertXML(expected string) w3cAssertion {
	return w3cAssertion{
		Type:  "assert-xml",
		Value: expected,
		Check: func(t *testing.T, result string, _ []string) bool {
			t.Helper()
			if xmlEqual(result, expected) {
				return true
			}
			t.Errorf("assert-xml failed:\n  got:    %s\n  expect: %s", result, expected)
			return false
		},
	}
}

func w3cAssertStringValue(expected string) w3cAssertion {
	return w3cAssertion{
		Type:  "assert-string-value",
		Value: expected,
		Check: func(t *testing.T, result string, _ []string) bool {
			t.Helper()
			actual := extractTextContent(result)
			if actual == expected {
				return true
			}
			// W3C test catalog assert-string-value defaults normalize-space="true":
			// collapse whitespace sequences and trim leading/trailing whitespace.
			if normalizeSpace(actual) == normalizeSpace(expected) {
				return true
			}
			t.Errorf("assert-string-value failed:\n  got:    %q\n  expect: %q", actual, expected)
			return false
		},
	}
}

func w3cAssertMessage(checks ...w3cCheck) w3cAssertion {
	return w3cAssertion{
		Type: "assert-message",
		Check: func(t *testing.T, _ string, messages []string) bool {
			t.Helper()
			combined := strings.Join(messages, "")
			for _, chk := range checks {
				if !chk.fn(combined, messages) {
					t.Errorf("assert-message failed: messages=%q", messages)
					return false
				}
			}
			return true
		},
	}
}

func w3cAnyOf(checks ...w3cCheck) w3cAssertion {
	return w3cAssertion{
		Type: "any-of",
		Check: func(t *testing.T, result string, messages []string) bool {
			t.Helper()
			for _, chk := range checks {
				if chk.fn(result, messages) {
					return true
				}
			}
			t.Errorf("any-of: no alternative matched for result: %s", result)
			return false
		},
	}
}

func w3cAssertSkip() w3cAssertion {
	return w3cAssertion{
		Type: "skip",
		Check: func(t *testing.T, _ string, _ []string) bool {
			t.Helper()
			t.Skip("assertion type not yet supported")
			return true
		},
	}
}

func w3cAssertXPath(expr string) w3cAssertion {
	return w3cAssertion{
		Type:  "assert",
		Value: expr,
		Check: func(t *testing.T, result string, _ []string) bool {
			t.Helper()
			return evalXPathAssert(t, expr, result)
		},
	}
}

// ──────────────────────────────────────────────────────────────────────
// Check constructors (for any-of / assert-message)
// ──────────────────────────────────────────────────────────────────────

func w3cCheckXML(expected string) w3cCheck {
	return w3cCheck{fn: func(result string, _ []string) bool {
		return xmlEqual(result, expected)
	}}
}

func w3cCheckStringValue(expected string) w3cCheck {
	return w3cCheck{fn: func(result string, _ []string) bool {
		actual := extractTextContent(result)
		if actual == expected {
			return true
		}
		// W3C test catalog assert-string-value defaults normalize-space="true"
		return normalizeSpace(actual) == normalizeSpace(expected)
	}}
}

func w3cCheckXPath(expr string) w3cCheck {
	return w3cCheck{fn: func(result string, _ []string) bool {
		doc, err := helium.Parse(context.TODO(), []byte(result))
		if err != nil {
			return false
		}
		compiled, err := xpath3.Compile(expr)
		if err != nil {
			return false
		}
		ctx := context.TODO()
		if root := doc.DocumentElement(); root != nil {
			ns := make(map[string]string)
			for _, n := range root.Namespaces() {
				if n.Prefix() != "" {
					ns[n.Prefix()] = n.URI()
				}
			}
			if len(ns) > 0 {
				ctx = xpath3.WithNamespaces(ctx, ns)
			}
		}
		res, err := compiled.Evaluate(ctx, doc)
		if err != nil {
			return false
		}
		ebv, err := xpath3.EBV(res.Sequence())
		return err == nil && ebv
	}}
}

func w3cCheckSkip() w3cCheck {
	return w3cCheck{fn: func(_ string, _ []string) bool {
		return true // skip = pass
	}}
}

// ──────────────────────────────────────────────────────────────────────
// Test runner
// ──────────────────────────────────────────────────────────────────────

func w3cRunTests(t *testing.T, tests []w3cTest) {
	t.Helper()

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			w3cRunOne(t, tc)
		})
	}
}

// w3cRunHeavyTests runs tests serially without t.Parallel().
// Use for test sets that are computationally expensive (e.g. unicode-90 which
// materializes ~1.1M code points per test and runs regex across all of them).
func w3cRunHeavyTests(t *testing.T, tests []w3cTest) {
	t.Helper()

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			w3cRunOne(t, tc)
		})
	}
}

func w3cRunOne(t *testing.T, tc w3cTest) {
	t.Helper()

	if tc.Skip != "" {
		t.Skip(tc.Skip)
		return
	}

	if tc.StylesheetPath == "" {
		t.Skip("no stylesheet")
		return
	}

	// Compile stylesheet (cached across tests sharing the same path)
	ssPath := filepath.Join(w3cTestdataDir, tc.StylesheetPath)
	ss, err := w3cCompileCached(t.Context(), ssPath)

	if tc.ExpectError {
		if err != nil {
			return // expected compile error
		}
		// May be a runtime error — continue to transform
	} else if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	// Prepare source document (file bytes cached across tests sharing the same path)
	var sourceData []byte
	if tc.SourceDocPath != "" {
		srcPath := filepath.Join(w3cTestdataDir, tc.SourceDocPath)
		sourceData = w3cReadSourceCached(t, srcPath)
	} else if tc.SourceContent != "" {
		sourceData = []byte(tc.SourceContent)
	} else {
		sourceData = []byte(`<?xml version="1.0"?><empty/>`)
	}

	sourceDoc, err := helium.Parse(t.Context(), sourceData)
	if err != nil {
		if tc.ExpectError {
			return // expected error during source parse
		}
		t.Fatalf("cannot parse source: %v", err)
	}
	// Set document URL for entity URI resolution (unparsed-entity-uri).
	if tc.SourceDocPath != "" {
		srcAbsPath, _ := filepath.Abs(filepath.Join(w3cTestdataDir, tc.SourceDocPath))
		sourceDoc.SetURL(srcAbsPath)
	}

	// Configure transform context
	ctx := t.Context()
	if tc.InitialTemplate != "" {
		ctx = xslt3.WithInitialTemplate(ctx, tc.InitialTemplate)
	}
	for pName, pVal := range tc.Params {
		// The W3C test catalog specifies param values as XPath expressions.
		// Strip enclosing quotes from string literals like "'Svalue'".
		if len(pVal) >= 2 && pVal[0] == '\'' && pVal[len(pVal)-1] == '\'' {
			pVal = pVal[1 : len(pVal)-1]
		}
		ctx = xslt3.WithParameter(ctx, pName, pVal)
	}

	// Capture messages
	var messages []string
	ctx = xslt3.WithMessageHandler(ctx, func(msg string, terminate bool) {
		messages = append(messages, msg)
	})

	// Transform
	resultDoc, err := xslt3.Transform(ctx, sourceDoc, ss)
	if err != nil {
		if tc.ExpectError {
			return // expected runtime error
		}
		t.Fatalf("transform error: %v", err)
	}

	if tc.ExpectError {
		t.Fatalf("expected error %s but transformation succeeded", tc.ErrorCode)
	}

	// Serialize result
	var buf bytes.Buffer
	err = resultDoc.XML(&buf, helium.WithNoDecl())
	require.NoError(t, err)
	result := strings.TrimSpace(buf.String())

	// Check assertions
	for _, a := range tc.Assertions {
		a.Check(t, result, messages)
	}
}

// ──────────────────────────────────────────────────────────────────────
// XML comparison helpers (adapted from xslt3_test.go)
// ──────────────────────────────────────────────────────────────────────

func xmlEqual(actual, expected string) bool {
	if domEqual(actual, expected) {
		return true
	}
	a := normalizeXMLString(actual)
	e := normalizeXMLString(expected)
	return a == e
}

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

func wrapXMLFragment(s string) string {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "<?xml") {
		if idx := strings.Index(trimmed, "?>"); idx >= 0 {
			trimmed = strings.TrimSpace(trimmed[idx+2:])
		}
	}
	return "<_w3c_root_>" + trimmed + "</_w3c_root_>"
}

func nodesEqual(a, b helium.Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	aType, bType := a.Type(), b.Type()
	// Treat CDATA and Text as equivalent
	if aType == helium.CDATASectionNode {
		aType = helium.TextNode
	}
	if bType == helium.CDATASectionNode {
		bType = helium.TextNode
	}
	if aType != bType {
		return false
	}
	switch aType {
	case helium.ElementNode:
		ea := a.(*helium.Element)
		eb := b.(*helium.Element)
		if ea.LocalName() != eb.LocalName() || ea.URI() != eb.URI() {
			return false
		}
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
		// Compare children, merging adjacent text/CDATA nodes
		return mergedChildrenEqual(ea, eb)
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

func isTextLike(n helium.Node) bool {
	return n.Type() == helium.TextNode || n.Type() == helium.CDATASectionNode
}

// mergedChildrenEqual compares element children, merging adjacent text/CDATA nodes.
func mergedChildrenEqual(a, b *helium.Element) bool {
	childA := a.FirstChild()
	childB := b.FirstChild()
	for childA != nil || childB != nil {
		if childA == nil || childB == nil {
			return false
		}
		// If both are text-like, merge and compare
		if isTextLike(childA) && isTextLike(childB) {
			var textA, textB strings.Builder
			for childA != nil && isTextLike(childA) {
				textA.Write(childA.Content())
				childA = childA.NextSibling()
			}
			for childB != nil && isTextLike(childB) {
				textB.Write(childB.Content())
				childB = childB.NextSibling()
			}
			if textA.String() != textB.String() {
				return false
			}
			continue
		}
		if !nodesEqual(childA, childB) {
			return false
		}
		childA = childA.NextSibling()
		childB = childB.NextSibling()
	}
	return true
}

func collectAttrs(e *helium.Element) map[string]string {
	attrs := make(map[string]string)
	for _, attr := range e.Attributes() {
		key := "{" + attr.URI() + "}" + attr.LocalName()
		attrs[key] = attr.Value()
	}
	return attrs
}

func normalizeXMLString(s string) string {
	s = strings.TrimSpace(s)
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
	s = string(result)
	s = strings.ReplaceAll(s, "> <", "><")
	s = strings.ReplaceAll(s, " >", ">")
	s = strings.ReplaceAll(s, " />", "/>")
	return s
}

// normalizeSpace mimics fn:normalize-space: collapse whitespace runs to a
// single space and trim leading/trailing whitespace.
func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// extractTextContent extracts all text content from an XML string,
// similar to XPath string-value of the root node.
func extractTextContent(xmlStr string) string {
	wrapped := wrapXMLFragment(xmlStr)
	doc, err := helium.Parse(context.TODO(), []byte(wrapped))
	if err != nil {
		return strings.TrimSpace(xmlStr)
	}
	return collectText(doc.DocumentElement())
}

func collectText(n helium.Node) string {
	if n == nil {
		return ""
	}
	switch n.Type() {
	case helium.TextNode:
		return string(n.Content())
	case helium.ElementNode:
		elem := n.(*helium.Element)
		var b strings.Builder
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			b.WriteString(collectText(child))
		}
		return b.String()
	default:
		return ""
	}
}

// evalXPathAssert parses the result XML, evaluates the XPath expression
// against it, and checks that the effective boolean value is true.
func evalXPathAssert(t *testing.T, expr string, resultXML string) bool {
	t.Helper()

	doc, err := helium.Parse(context.TODO(), []byte(resultXML))
	if err != nil {
		t.Errorf("assert: cannot parse result XML: %v", err)
		return false
	}

	compiled, err := xpath3.Compile(expr)
	if err != nil {
		t.Errorf("assert: cannot compile XPath %q: %v", expr, err)
		return false
	}

	// Extract namespace bindings from the document element for XPath evaluation
	ctx := context.TODO()
	if root := doc.DocumentElement(); root != nil {
		ns := make(map[string]string)
		for _, n := range root.Namespaces() {
			if n.Prefix() != "" {
				ns[n.Prefix()] = n.URI()
			}
		}
		if len(ns) > 0 {
			ctx = xpath3.WithNamespaces(ctx, ns)
		}
	}

	res, err := compiled.Evaluate(ctx, doc)
	if err != nil {
		t.Errorf("assert: XPath evaluation error for %q: %v", expr, err)
		return false
	}

	ebv, err := xpath3.EBV(res.Sequence())
	if err != nil {
		t.Errorf("assert: cannot compute EBV for %q: %v", expr, err)
		return false
	}
	if !ebv {
		t.Errorf("assert failed: %s evaluated to false (result: %s)", expr, resultXML)
		return false
	}
	return true
}

// w3cCompileCached compiles a stylesheet, caching the result by path.
// Compile errors are not cached so that tests expecting compile errors
// still report them per test case.
func w3cCompileCached(ctx context.Context, path string) (*xslt3.Stylesheet, error) {
	if v, ok := w3cStylesheetCache.Load(path); ok {
		return v.(*xslt3.Stylesheet), nil
	}
	ss, err := xslt3.CompileFile(ctx, path)
	if err != nil {
		return nil, err
	}
	actual, loaded := w3cStylesheetCache.LoadOrStore(path, ss)
	if loaded {
		return actual.(*xslt3.Stylesheet), nil
	}
	return ss, nil
}

// w3cReadSourceCached reads source file bytes, caching by path.
func w3cReadSourceCached(t *testing.T, path string) []byte {
	t.Helper()
	if v, ok := w3cSourceBytesCache.Load(path); ok {
		return v.([]byte)
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	w3cSourceBytesCache.LoadOrStore(path, data)
	return data
}
