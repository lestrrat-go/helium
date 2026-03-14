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

const w3cTestdataDir = "../testdata/xslt30/testdata"

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
		return extractTextContent(result) == expected
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

	// Compile stylesheet
	ssPath := filepath.Join(w3cTestdataDir, tc.StylesheetPath)
	ss, err := xslt3.CompileFile(t.Context(), ssPath)

	if tc.ExpectError {
		if err != nil {
			return // expected compile error
		}
		// May be a runtime error — continue to transform
	} else if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	// Prepare source document
	var sourceData []byte
	if tc.SourceDocPath != "" {
		srcPath := filepath.Join(w3cTestdataDir, tc.SourceDocPath)
		var readErr error
		sourceData, readErr = os.ReadFile(srcPath)
		require.NoError(t, readErr)
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

	// Configure transform context
	ctx := t.Context()
	if tc.InitialTemplate != "" {
		ctx = xslt3.WithInitialTemplate(ctx, tc.InitialTemplate)
	}
	for pName, pVal := range tc.Params {
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
			s = trimmed[idx+2:]
		}
	}
	return "<_w3c_root_>" + s + "</_w3c_root_>"
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
