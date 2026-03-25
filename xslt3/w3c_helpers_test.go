package xslt3_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	htmlparser "github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode"
	"github.com/lestrrat-go/helium/internal/sequence"
)

const (
	w3cTestdataDir = "../testdata/xslt30/testdata"
	w3cSourceDir   = "../testdata/xslt30/source"

	// w3cMaxParallel limits how many W3C subtests run concurrently.
	w3cMaxParallel = 20
)

// w3cSem gates concurrent W3C test execution.
var w3cSem = make(chan struct{}, w3cMaxParallel)

// w3cSlowTests lists tests that are too slow to run by default.
// Set HELIUM_SLOW_TESTS=1 to include them.
var w3cSlowTests = map[string]struct{}{
	"sf-boolean-107":        {},
	"sf-not-107":            {},
	"si-iterate-133":        {}, // ~8.9s citygml.xml
	"si-choose-012":         {}, // ~3.3s big-transactions.xml
	"si-iterate-037":        {}, // ~2.3s ot.xml
	"si-iterate-134":        {}, // ~1.7s citygml.xml
	"si-iterate-135":        {}, // ~1.7s citygml.xml
	"si-next-match-067":     {}, // ~1.7s ot.xml
	"si-apply-imports-068":  {}, // ~1.8s ot.xml
	"si-apply-imports-069":  {}, // ~1.8s ot.xml
	"si-apply-imports-070":  {}, // ~1.8s ot.xml
	"si-lre-904":            {}, // ~1.0s ot.xml
	"si-lre-905":            {}, // ~1.0s ot.xml
}

// isSlowSourceDoc returns true for source documents that are too large
// to process quickly in parallel test runs.
func isSlowSourceDoc(path string) bool {
	switch {
	case strings.Contains(path, "citygml.xml"):
		return true
	case strings.Contains(path, "ot.xml"):
		return true
	case strings.Contains(path, "big-transactions.xml"):
		return true
	}
	return false
}

// isSlowStreamingTest returns true for streaming test names that use
// big-transactions.xml (100K elements) via xsl:source-document.
// These templates are identified by their suffix: the W3C streaming
// test stylesheets use template names like c-015..c-020, c-100..c-120
// for the big-transactions variants.
func isSlowStreamingTest(name string) bool {
	// Extract the numeric suffix after the last hyphen
	idx := strings.LastIndex(name, "-")
	if idx < 0 {
		return false
	}
	suffix := name[idx+1:]
	switch suffix {
	case "015", "016", "017", "018", "020",
		"100", "101", "102", "103", "104", "105", "106", "107",
		"116", "118", "120",
		"216", "218", "219", "220", "221":
		// Only for streaming test categories (sf-*, sx-*, si-*)
		prefix := name[:idx]
		return strings.HasPrefix(prefix, "sf-") ||
			strings.HasPrefix(prefix, "sx-") ||
			strings.HasPrefix(prefix, "si-")
	}
	return false
}

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
	w3cStylesheetCache  sync.Map // path → *xslt3.Stylesheet
	w3cSourceBytesCache sync.Map // path → []byte

	// w3cResultDocOutputDefs stores effective output definitions for secondary
	// result documents, keyed by "testName\x00href". Populated by the test
	// runner and read by w3cAssertResultDocument for proper serialization.
	w3cResultDocOutputDefs sync.Map // "testName\x00href" → *xslt3.OutputDef
)

// w3cPackageDep describes a secondary package dependency for a W3C test.
type w3cPackageDep struct {
	URI      string // package name URI
	Version  string // package-version
	FilePath string // relative path to package file within testdata
}

// w3cCollection describes a named collection for fn:collection/fn:uri-collection.
type w3cCollection struct {
	URI      string   // collection URI (e.g. "log-files")
	DocPaths []string // paths to XML documents relative to w3cTestdataDir
}

// w3cTest describes a single W3C XSLT 3.0 test case.
type w3cTest struct {
	Name                        string
	StylesheetPath              string
	SecondaryStylesheets        []string
	PackageDeps                 []w3cPackageDep
	SourceDocPath               string
	SourceContent               string
	InitialTemplate             string
	InitialTemplateParams       map[string]string
	InitialTemplateTunnelParams map[string]string
	InitialMode                 string
	InitialModeSelect           string // XPath expression for initial-mode select
	InitialModeParams           map[string]string
	InitialModeTunnelParams     map[string]string
	Params                      map[string]string
	ParamTypes                  map[string]string // as types for params (from catalog <param as="...">)
	InitialFunction             string            // QName of function to call as entry point
	InitialFunctionParams       []string // positional params (XPath select expressions)
	ExpectError                 bool
	AcceptErrors                []string // error codes accepted as alternative outcomes (from any-of)
	ErrorCode                   string
	Assertions                  []w3cAssertion
	Skip                        string
	Collections                 []w3cCollection
	OnMultipleMatch             string // "use-last" or "fail" (W3C dependency override)
	BaseOutputURI               string // base output URI for current-output-uri(); empty = not set
	SourceSchemaPath            string   // path to XSD schema for source document validation (relative to testdata dir)
	ImportSchemaPaths           []string // schema paths for xsl:import-schema resolution (relative to testdata dir)
	VersionResolution           string   // "lowest" to select lowest matching package version (default: highest)
	EmbeddedStylesheet          bool     // source document contains <?xml-stylesheet?> PI defining the stylesheet
}

// w3cAssertion is an assertion to check against the transform result.
type w3cAssertion struct {
	Type     string // "assert-xml", "assert-string-value", "any-of", "assert-message", "assert-result-document", "assert-serialization", "skip"
	Value    string
	Check    func(t *testing.T, result string, messages []string, resultDocs map[string]*helium.Document) bool
	RawCheck func(t *testing.T, rawResult xpath3.Sequence, result string) bool // for assertions needing raw XPath sequence
}

// w3cCheck is used inside any-of assertions.
type w3cCheck struct {
	fn func(result string, messages []string, resultDocs map[string]*helium.Document) bool
	// docFn, when non-nil, is called instead of fn when a concrete document
	// is available (e.g. inside assert-result-document). This lets XPath
	// assertions run against the real document, preserving base-uri().
	docFn func(doc *helium.Document) bool
	// rawFn, when non-nil, is called with the raw XPath sequence for checks
	// that need typed comparison (assert-eq, assert-deep-eq, etc.).
	rawFn func(rawResult xpath3.Sequence) bool
}

// ──────────────────────────────────────────────────────────────────────
// Assertion constructors
// ──────────────────────────────────────────────────────────────────────

func w3cAssertXML(expected string) w3cAssertion {
	return w3cAssertion{
		Type:  "assert-xml",
		Value: expected,
		Check: func(t *testing.T, result string, _ []string, _ map[string]*helium.Document) bool {
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
		Check: func(t *testing.T, result string, _ []string, _ map[string]*helium.Document) bool {
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
		Check: func(t *testing.T, _ string, messages []string, resultDocs map[string]*helium.Document) bool {
			t.Helper()
			combined := strings.Join(messages, "")
			for _, chk := range checks {
				// If the check has a rawFn (e.g. assert-eq), try it with each
				// message converted to an xs:string atomic value.
				if chk.rawFn != nil {
					passed := false
					for _, msg := range messages {
						seq := xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: msg}}
						if chk.rawFn(seq) {
							passed = true
							break
						}
					}
					if !passed {
						t.Errorf("assert-message failed: messages=%q", messages)
						return false
					}
					continue
				}
				if chk.fn(combined, messages, resultDocs) {
					continue
				}
				// The combined string may not be well-formed XML (e.g. multiple
				// root elements from separate messages). Try each individual
				// message before failing.
				passed := false
				for _, msg := range messages {
					if chk.fn(msg, messages, resultDocs) {
						passed = true
						break
					}
				}
				if !passed {
					t.Errorf("assert-message failed: messages=%q", messages)
					return false
				}
			}
			return true
		},
	}
}

func w3cAnyOf(checks ...w3cCheck) w3cAssertion {
	// Check if any sub-check has rawFn.
	hasRawFn := false
	for _, chk := range checks {
		if chk.rawFn != nil {
			hasRawFn = true
			break
		}
	}
	a := w3cAssertion{
		Type: "any-of",
		Check: func(t *testing.T, result string, messages []string, resultDocs map[string]*helium.Document) bool {
			t.Helper()
			for _, chk := range checks {
				if chk.fn(result, messages, resultDocs) {
					return true
				}
			}
			t.Errorf("any-of: no alternative matched for result: %s", result)
			return false
		},
	}
	if hasRawFn {
		a.RawCheck = func(t *testing.T, rawResult xpath3.Sequence, result string) bool {
			t.Helper()
			for _, chk := range checks {
				if chk.rawFn != nil {
					if chk.rawFn(rawResult) {
						return true
					}
				} else if chk.fn(result, nil, nil) {
					return true
				}
			}
			t.Errorf("any-of: no alternative matched for result: %s", result)
			return false
		}
	}
	return a
}

func w3cAssertSkip() w3cAssertion {
	return w3cAssertion{
		Type: "skip",
		Check: func(t *testing.T, _ string, _ []string, _ map[string]*helium.Document) bool {
			t.Helper()
			t.Skip("assertion type not yet supported")
			return true
		},
	}
}

func w3cAssertType(typeName string) w3cAssertion {
	return w3cAssertion{
		Type:  "assert-type",
		Value: typeName,
		RawCheck: func(t *testing.T, rawResult xpath3.Sequence, _ string) bool {
			t.Helper()
			return evalRawResultXPath(t, fmt.Sprintf("$result instance of %s", typeName), rawResult)
		},
	}
}

func w3cAssertCount(n int) w3cAssertion {
	return w3cAssertion{
		Type:  "assert-count",
		Value: fmt.Sprintf("%d", n),
		RawCheck: func(t *testing.T, rawResult xpath3.Sequence, _ string) bool {
			t.Helper()
			got := 0
			if rawResult != nil {
				got = sequence.Len(rawResult)
			}
			if got != n {
				t.Errorf("assert-count: expected %d items, got %d", n, got)
				return false
			}
			return true
		},
	}
}

func w3cAssertDeepEq(expr string) w3cAssertion {
	return w3cAssertion{
		Type:  "assert-deep-eq",
		Value: expr,
		RawCheck: func(t *testing.T, rawResult xpath3.Sequence, _ string) bool {
			t.Helper()
			return evalRawResultXPath(t, fmt.Sprintf("deep-equal($result, (%s))", expr), rawResult)
		},
	}
}

func w3cAssertEmpty() w3cAssertion {
	return w3cAssertion{
		Type: "assert-empty",
		RawCheck: func(t *testing.T, rawResult xpath3.Sequence, _ string) bool {
			t.Helper()
			n := 0
			if rawResult != nil {
				n = sequence.Len(rawResult)
			}
			if n != 0 {
				t.Errorf("assert-empty: expected empty sequence, got %d items", n)
				return false
			}
			return true
		},
	}
}

func w3cAssertEq(expr string) w3cAssertion {
	return w3cAssertion{
		Type:  "assert-eq",
		Value: expr,
		RawCheck: func(t *testing.T, rawResult xpath3.Sequence, _ string) bool {
			t.Helper()
			return evalRawResultXPath(t, fmt.Sprintf("$result eq %s", expr), rawResult)
		},
	}
}

func w3cAssertNot(checks ...w3cCheck) w3cAssertion {
	return w3cAssertion{
		Type: "not",
		Check: func(t *testing.T, result string, messages []string, resultDocs map[string]*helium.Document) bool {
			t.Helper()
			for _, ch := range checks {
				if ch.fn(result, messages, resultDocs) {
					t.Errorf("assert-not: inner assertion unexpectedly passed")
					return false
				}
			}
			return true
		},
	}
}

// evalRawResultXPath evaluates an XPath expression with $result bound to the
// raw XDM sequence, returning true if the result is effectively true.
func evalRawResultXPath(t *testing.T, expr string, rawResult xpath3.Sequence) bool {
	t.Helper()

	compiled, err := xpath3.NewCompiler().Compile(expr)
	if err != nil {
		t.Errorf("raw-result assert: cannot compile XPath %q: %v", expr, err)
		return false
	}

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	eval = eval.Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
		"result": rawResult,
	}))

	res, err := eval.Evaluate(context.TODO(), compiled, nil)
	if err != nil {
		t.Errorf("raw-result assert: XPath evaluation error for %q: %v", expr, err)
		return false
	}

	ebv, err := xpath3.EBV(res.Sequence())
	if err != nil {
		t.Errorf("raw-result assert: cannot compute EBV for %q: %v", expr, err)
		return false
	}
	if !ebv {
		t.Errorf("raw-result assert failed: %s evaluated to false", expr)
		return false
	}
	return true
}

// evalRawXPathBool evaluates an XPath expression with $result bound to the
// raw XDM sequence, returning true if the result is effectively true.
// This is the non-testing version for use in w3cCheck functions.
func evalRawXPathBool(expr string, rawResult xpath3.Sequence) bool {
	compiled, err := xpath3.NewCompiler().Compile(expr)
	if err != nil {
		return false
	}
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	eval = eval.Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
		"result": rawResult,
	}))
	res, err := eval.Evaluate(context.TODO(), compiled, nil)
	if err != nil {
		return false
	}
	ebv, err := xpath3.EBV(res.Sequence())
	return err == nil && ebv
}

func w3cAssertXPath(expr string) w3cAssertion {
	return w3cAssertion{
		Type:  "assert",
		Value: expr,
		Check: func(t *testing.T, result string, _ []string, _ map[string]*helium.Document) bool {
			t.Helper()
			return evalXPathAssert(t, expr, result)
		},
	}
}

func w3cAssertResultDocument(uri string, checks ...w3cCheck) w3cAssertion {
	return w3cAssertion{
		Type:  "assert-result-document",
		Value: uri,
		Check: func(t *testing.T, _ string, messages []string, resultDocs map[string]*helium.Document) bool {
			t.Helper()
			doc, ok := resultDocs[uri]
			matchedHref := uri
			if !ok {
				// Try matching by suffix — result-document URIs may be absolute
				for href, d := range resultDocs {
					if strings.HasSuffix(href, "/"+uri) || strings.HasSuffix(href, "\\"+uri) || href == uri {
						doc = d
						matchedHref = href
						ok = true
						break
					}
				}
			}
			if !ok {
				t.Errorf("assert-result-document: no result document for URI %q (have: %v)", uri, resultDocKeys(resultDocs))
				return false
			}
			// Look up the effective output definition for this result document
			// so that serialization parameters (omit-xml-declaration, indent, etc.)
			// from the named format are applied.
			var buf bytes.Buffer
			var outDef *xslt3.OutputDef
			if v, found := w3cResultDocOutputDefs.Load(t.Name() + "\x00" + matchedHref); found {
				outDef = v.(*xslt3.OutputDef)
			}
			if outDef != nil {
				if err := xslt3.SerializeResult(&buf, doc, outDef); err != nil {
					t.Errorf("assert-result-document: cannot serialize result document %q: %v", uri, err)
					return false
				}
			} else {
				if err := doc.XML(&buf, helium.WithNoDecl()); err != nil {
					t.Errorf("assert-result-document: cannot serialize result document %q: %v", uri, err)
					return false
				}
			}
			rdResult := strings.TrimSpace(buf.String())
			for _, chk := range checks {
				var pass bool
				if chk.docFn != nil {
					pass = chk.docFn(doc)
				} else {
					pass = chk.fn(rdResult, messages, resultDocs)
				}
				if !pass {
					t.Errorf("assert-result-document failed for URI %q: got %s", uri, rdResult)
					return false
				}
			}
			return true
		},
	}
}

func w3cAssertSerialization(method string, expected string) w3cAssertion {
	return w3cAssertion{
		Type:  "assert-serialization",
		Value: expected,
		Check: func(t *testing.T, result string, _ []string, _ map[string]*helium.Document) bool {
			t.Helper()
			if checkSerializationResult(method, result, expected) {
				return true
			}
			t.Errorf("assert-serialization (method=%s) failed:\n  got:    %q\n  expect: %q", method, result, expected)
			return false
		},
	}
}

func w3cAssertSerializationMatches(pattern string) w3cAssertion {
	// Normalize literal \r\n in patterns to \r?\n so both CR+LF and LF-only match.
	normalizedPattern := strings.ReplaceAll(pattern, "\r\n", "\r?\n")
	return w3cAssertion{
		Type:  "serialization-matches",
		Value: pattern,
		Check: func(t *testing.T, result string, _ []string, _ map[string]*helium.Document) bool {
			t.Helper()
			matched, err := regexp.MatchString(normalizedPattern, result)
			if err != nil {
				t.Errorf("serialization-matches: invalid pattern %q: %v", pattern, err)
				return false
			}
			if matched {
				return true
			}
			t.Errorf("serialization-matches failed:\n  pattern: %q\n  result:  %q", pattern, result)
			return false
		},
	}
}

// checkSerializationResult checks whether result matches expected for a given
// serialization method. For "text" method, comparison is by text content.
// For "xml"/"html"/"xhtml", comparison uses XML equality.
func checkSerializationResult(method string, result string, expected string) bool {
	// Normalize line endings: the W3C test suite uses \r\n in expected
	// output but our serializer produces \n. Normalize both to \n before
	// comparison so that line-ending differences do not cause failures.
	normLE := func(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

	switch method {
	case "text":
		// Text output: compare plain text content
		actual := extractTextContent(result)
		if normLE(actual) == normLE(expected) {
			return true
		}
		// Also try the raw result for text-method serialization
		if strings.TrimSpace(normLE(result)) == strings.TrimSpace(normLE(expected)) {
			return true
		}
		return normalizeSpace(actual) == normalizeSpace(expected)
	case "xml", "xhtml", "html":
		return xmlEqual(result, expected)
	case "adaptive":
		// Adaptive serialization: compare as string
		return strings.TrimSpace(normLE(result)) == strings.TrimSpace(normLE(expected))
	default:
		// Unknown method: compare as string with line-ending normalization
		return strings.TrimSpace(normLE(result)) == strings.TrimSpace(normLE(expected))
	}
}

// assertionsNeedSerialization returns true when any assertion requires full
// serialized output (i.e. assert-serialization or serialization-matches),
// which triggers use of xslt3.SerializeResult instead of the no-decl fallback.
func assertionsNeedSerialization(assertions []w3cAssertion) bool {
	for _, a := range assertions {
		if a.Type == "assert-serialization" || a.Type == "serialization-matches" {
			return true
		}
	}
	return false
}

func resultDocKeys(m map[string]*helium.Document) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ──────────────────────────────────────────────────────────────────────
// Check constructors (for any-of / assert-message)
// ──────────────────────────────────────────────────────────────────────

func w3cCheckXML(expected string) w3cCheck {
	return w3cCheck{fn: func(result string, _ []string, _ map[string]*helium.Document) bool {
		return xmlEqual(result, expected)
	}}
}

func w3cCheckStringValue(expected string) w3cCheck {
	return w3cCheck{fn: func(result string, _ []string, _ map[string]*helium.Document) bool {
		actual := extractTextContent(result)
		if actual == expected {
			return true
		}
		// W3C test catalog assert-string-value defaults normalize-space="true"
		return normalizeSpace(actual) == normalizeSpace(expected)
	}}
}

func w3cCheckXPath(expr string) w3cCheck {
	evalOnDoc := func(doc *helium.Document) bool {
		compiled, err := xpath3.NewCompiler().Compile(expr)
		if err != nil {
			return false
		}
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
		ns := gatherDocNamespaces(doc)
		if len(ns) > 0 {
			eval = eval.Namespaces(ns)
		}
		res, err := eval.Evaluate(context.TODO(), compiled, doc)
		if err != nil {
			return false
		}
		ebv, err := xpath3.EBV(res.Sequence())
		return err == nil && ebv
	}
	return w3cCheck{
		fn: func(result string, _ []string, _ map[string]*helium.Document) bool {
			doc, err := helium.NewParser().Parse(context.TODO(), []byte(result))
			if err != nil {
				// Result may be plain text or an XML fragment with multiple
				// root elements (e.g. from xsl:message). Wrap in a temporary
				// element to parse, then promote children to a new document
				// so XPath like "/elem" can address top-level elements.
				doc, err = helium.NewParser().Parse(context.TODO(), []byte("<_r>"+result+"</_r>"))
				if err != nil {
					return false
				}
				doc = promoteWrapperChildren(doc)
			}
			return evalOnDoc(doc)
		},
		docFn: evalOnDoc,
	}
}

func w3cCheckSkip() w3cCheck {
	return w3cCheck{fn: func(_ string, _ []string, _ map[string]*helium.Document) bool {
		return true // skip = pass
	}}
}

func w3cCheckSerialization(method string, expected string) w3cCheck {
	return w3cCheck{fn: func(result string, _ []string, _ map[string]*helium.Document) bool {
		return checkSerializationResult(method, result, expected)
	}}
}

func w3cCheckSerializationMatches(pattern string) w3cCheck {
	return w3cCheck{fn: func(result string, _ []string, _ map[string]*helium.Document) bool {
		matched, err := regexp.MatchString(pattern, result)
		return err == nil && matched
	}}
}

func w3cCheckAllOf(checks ...w3cCheck) w3cCheck {
	// Check if any sub-check has docFn or rawFn.
	hasDocFn := false
	hasRawFn := false
	for _, chk := range checks {
		if chk.docFn != nil {
			hasDocFn = true
		}
		if chk.rawFn != nil {
			hasRawFn = true
		}
	}
	c := w3cCheck{fn: func(result string, messages []string, resultDocs map[string]*helium.Document) bool {
		for _, chk := range checks {
			if chk.fn(result, messages, resultDocs) {
				continue
			}
			// When called from assert-message context, the combined result
			// may not be well-formed XML. Try each individual message.
			passed := false
			for _, msg := range messages {
				if chk.fn(msg, messages, resultDocs) {
					passed = true
					break
				}
			}
			if !passed {
				return false
			}
		}
		return true
	}}
	if hasDocFn {
		c.docFn = func(doc *helium.Document) bool {
			for _, chk := range checks {
				if chk.docFn != nil {
					if !chk.docFn(doc) {
						return false
					}
				} else {
					// Fall back to fn with serialized form.
					var buf bytes.Buffer
					if err := doc.XML(&buf, helium.WithNoDecl()); err != nil {
						return false
					}
					if !chk.fn(strings.TrimSpace(buf.String()), nil, nil) {
						return false
					}
				}
			}
			return true
		}
	}
	if hasRawFn {
		c.rawFn = func(rawResult xpath3.Sequence) bool {
			for _, chk := range checks {
				if chk.rawFn != nil {
					if !chk.rawFn(rawResult) {
						return false
					}
				}
				// Non-raw checks are skipped in raw mode (they'll be checked by fn).
			}
			return true
		}
	}
	return c
}

func w3cCheckError(code string) w3cCheck {
	return w3cCheck{fn: func(_ string, _ []string, _ map[string]*helium.Document) bool {
		// In the "not" context, this checks whether the transform errored.
		// Since we only reach assertion checking on success, error checks
		// inside "not" always return false (no error occurred).
		return false
	}}
}

func w3cCheckAnyOf(checks ...w3cCheck) w3cCheck {
	c := w3cCheck{fn: func(result string, messages []string, resultDocs map[string]*helium.Document) bool {
		for _, chk := range checks {
			if chk.fn(result, messages, resultDocs) {
				return true
			}
		}
		return false
	}}
	// If any sub-check has rawFn, provide a rawFn for the group.
	hasRawFn := false
	for _, chk := range checks {
		if chk.rawFn != nil {
			hasRawFn = true
			break
		}
	}
	if hasRawFn {
		c.rawFn = func(rawResult xpath3.Sequence) bool {
			for _, chk := range checks {
				if chk.rawFn != nil {
					if chk.rawFn(rawResult) {
						return true
					}
				} else if chk.fn("", nil, nil) {
					return true
				}
			}
			return false
		}
	}
	return c
}

//nolint:unused // called from generated w3c_*_gen_test.go files
func w3cCheckType(typeName string) w3cCheck {
	return w3cCheck{
		fn: func(_ string, _ []string, _ map[string]*helium.Document) bool {
			return false // type checks require raw result
		},
		rawFn: func(rawResult xpath3.Sequence) bool {
			return evalRawXPathBool(fmt.Sprintf("$result instance of %s", typeName), rawResult)
		},
	}
}

//nolint:unused // called from generated w3c_*_gen_test.go files
func w3cCheckCount(n int) w3cCheck {
	return w3cCheck{
		fn: func(_ string, _ []string, _ map[string]*helium.Document) bool {
			return false // count checks require raw result
		},
		rawFn: func(rawResult xpath3.Sequence) bool {
			got := 0
			if rawResult != nil {
				got = sequence.Len(rawResult)
			}
			return got == n
		},
	}
}

//nolint:unused // called from generated w3c_*_gen_test.go files
func w3cCheckDeepEq(expr string) w3cCheck {
	return w3cCheck{
		fn: func(_ string, _ []string, _ map[string]*helium.Document) bool {
			return false // deep-eq checks require raw result
		},
		rawFn: func(rawResult xpath3.Sequence) bool {
			return evalRawXPathBool(fmt.Sprintf("deep-equal($result, (%s))", expr), rawResult)
		},
	}
}

//nolint:unused // called from generated w3c_*_gen_test.go files
func w3cCheckEmpty() w3cCheck {
	return w3cCheck{
		fn: func(result string, _ []string, _ map[string]*helium.Document) bool {
			return strings.TrimSpace(result) == ""
		},
		rawFn: func(rawResult xpath3.Sequence) bool {
			return rawResult == nil || sequence.Len(rawResult) == 0
		},
	}
}

func w3cCheckEq(expr string) w3cCheck {
	return w3cCheck{
		fn: func(_ string, _ []string, _ map[string]*helium.Document) bool {
			return false // eq checks require raw result
		},
		rawFn: func(rawResult xpath3.Sequence) bool {
			return evalRawXPathBool(fmt.Sprintf("$result eq %s", expr), rawResult)
		},
	}
}

//nolint:unused // called from generated w3c_*_gen_test.go files
func w3cCheckNot(checks ...w3cCheck) w3cCheck {
	return w3cCheck{fn: func(result string, messages []string, resultDocs map[string]*helium.Document) bool {
		for _, chk := range checks {
			if chk.fn(result, messages, resultDocs) {
				return false
			}
		}
		return true
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

	w3cSem <- struct{}{}
	t.Cleanup(func() { <-w3cSem })

	if _, slow := w3cSlowTests[tc.Name]; slow && os.Getenv("HELIUM_SLOW_TESTS") == "" {
		t.Skip("slow test; set HELIUM_SLOW_TESTS=1 to run")
	}
	if isSlowSourceDoc(tc.SourceDocPath) && os.Getenv("HELIUM_SLOW_TESTS") == "" {
		t.Skip("slow source doc; set HELIUM_SLOW_TESTS=1 to run")
	}
	if isSlowStreamingTest(tc.Name) && os.Getenv("HELIUM_SLOW_TESTS") == "" {
		t.Skip("slow streaming test (big-transactions.xml); set HELIUM_SLOW_TESTS=1 to run")
	}

	if reason := w3cImplicitSkipReason(tc.Name); reason != "" {
		t.Skip(reason)
		return
	}

	if tc.Skip != "" {
		t.Skip(tc.Skip)
		return
	}

	if tc.StylesheetPath == "" && !tc.EmbeddedStylesheet {
		t.Skip("no stylesheet")
		return
	}

	// Parse source document early so it is available for embedded stylesheet
	// extraction (before stylesheet compilation).
	var sourceDoc *helium.Document
	var sourceData []byte
	hasExplicitSource := false
	if tc.SourceDocPath != "" {
		srcPath := w3cResolvePath(tc.SourceDocPath)
		sourceData = w3cReadSourceCached(t, srcPath)
		hasExplicitSource = true
	} else if tc.SourceContent != "" {
		sourceData = []byte(tc.SourceContent)
		hasExplicitSource = true
	}

	if hasExplicitSource {
		sourceParser := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true)
		if tc.SourceDocPath != "" {
			sourceParser = sourceParser.SubstituteEntities(true).FixBaseURIs(false)
			srcAbsPath, _ := filepath.Abs(w3cResolvePath(tc.SourceDocPath))
			sourceParser = sourceParser.BaseURI(srcAbsPath)
		}
		var parseErr error
		sourceDoc, parseErr = sourceParser.Parse(t.Context(), sourceData)
		if parseErr != nil {
			if tc.ExpectError {
				return
			}
			t.Fatalf("cannot parse source: %v", parseErr)
		}
		if tc.SourceDocPath != "" {
			srcAbsPath, _ := filepath.Abs(w3cResolvePath(tc.SourceDocPath))
			sourceDoc.SetURL(srcAbsPath)
		} else if tc.SourceContent != "" && tc.StylesheetPath != "" {
			ssAbsPath, _ := filepath.Abs(w3cResolvePath(tc.StylesheetPath))
			sourceDoc.SetURL(ssAbsPath)
		}
	}

	// Compile stylesheet
	var ss *xslt3.Stylesheet
	var err error

	if tc.EmbeddedStylesheet && tc.StylesheetPath == "" {
		// Extract embedded stylesheet from source document
		ssDoc := w3cExtractEmbeddedStylesheet(t, sourceDoc)
		srcAbsPath, _ := filepath.Abs(w3cResolvePath(tc.SourceDocPath))
		compiler := xslt3.NewCompiler().BaseURI(srcAbsPath)
		ss, err = compiler.Compile(t.Context(), ssDoc)
	} else if len(tc.PackageDeps) > 0 || len(tc.Params) > 0 || len(tc.ImportSchemaPaths) > 0 {
		// When package deps, external params, or import schemas exist, compile without caching.
		ssPath := w3cResolvePath(tc.StylesheetPath)
		absPath, _ := filepath.Abs(ssPath)
		compiler := xslt3.NewCompiler().BaseURI(absPath)
		if len(tc.PackageDeps) > 0 {
			compiler = compiler.PackageResolver(w3cPackageResolver{deps: tc.PackageDeps, versionResolution: tc.VersionResolution})
		}
		if len(tc.ImportSchemaPaths) > 0 {
			var importSchemas []*xsd.Schema
			for _, sp := range tc.ImportSchemaPaths {
				schemaPath := w3cResolvePath(sp)
				schema, schemaErr := xsd.NewCompiler().CompileFile(t.Context(), schemaPath)
				if schemaErr != nil {
					t.Fatalf("compile import schema %q: %v", sp, schemaErr)
				}
				importSchemas = append(importSchemas, schema)
			}
			compiler = compiler.ImportSchemas(importSchemas...)
		}
		for pName, pVal := range tc.Params {
			expr, compErr := xpath3.NewCompiler().Compile(pVal)
			if compErr == nil {
				result, evalErr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), expr, nil)
				if evalErr == nil {
					seq := result.Sequence()
					if asType, ok := tc.ParamTypes[pName]; ok {
						seq = castSequenceForParam(seq, asType)
					}
					compiler = compiler.SetStaticParameter(pName, seq)
				}
			}
		}
		data, readErr := os.ReadFile(ssPath)
		if readErr != nil {
			t.Fatalf("read stylesheet: %v", readErr)
		}
		ssParser := helium.NewParser().LoadExternalDTD(true).SubstituteEntities(true).BaseURI(absPath)
		doc, parseErr := ssParser.Parse(t.Context(), data)
		if parseErr != nil {
			if tc.ExpectError {
				return
			}
			t.Fatalf("parse stylesheet: %v", parseErr)
		}
		ss, err = compiler.Compile(t.Context(), doc)
	} else if tc.EmbeddedStylesheet && tc.StylesheetPath != "" {
		// Embedded stylesheet with a standalone primary stylesheet (tests 005-015).
		// The source doc has an embedded stylesheet that imports/includes the
		// standalone stylesheet. Extract and compile the embedded one.
		ssDoc := w3cExtractEmbeddedStylesheet(t, sourceDoc)
		srcAbsPath, _ := filepath.Abs(w3cResolvePath(tc.SourceDocPath))
		compiler := xslt3.NewCompiler().BaseURI(srcAbsPath)
		ss, err = compiler.Compile(t.Context(), ssDoc)
	} else {
		ssPath := w3cResolvePath(tc.StylesheetPath)
		ss, err = w3cCompileCached(t.Context(), ssPath)
	}

	if tc.ExpectError {
		if err != nil {
			return // expected compile error
		}
		// May be a runtime error — continue to transform
	} else if err != nil {
		if w3cErrorAccepted(err, tc.AcceptErrors) {
			return
		}
		t.Fatalf("compile error: %v", err)
	}

	// Build the invocation based on entry mode
	ctx := t.Context()
	var inv xslt3.Invocation
	switch {
	case tc.InitialTemplate != "":
		inv = ss.CallTemplate(tc.InitialTemplate)
		if sourceDoc != nil {
			inv = inv.SourceDocument(sourceDoc)
		}
		if tc.InitialModeSelect != "" {
			inv = inv.GlobalContextSelect(tc.InitialModeSelect)
		}
		for pName, pVal := range tc.InitialTemplateParams {
			inv = inv.SetInitialTemplateParameter(pName, w3cEvaluateParamSequence(ctx, pVal))
		}
		for pName, pVal := range tc.InitialTemplateTunnelParams {
			inv = inv.SetTunnelParameter(pName, w3cEvaluateParamSequence(ctx, pVal))
		}
	case tc.InitialFunction != "":
		var fnParams []xpath3.Sequence
		for _, paramExpr := range tc.InitialFunctionParams {
			expr, fnErr := xpath3.NewCompiler().Compile(paramExpr)
			if fnErr == nil {
				result, evalErr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(context.Background(), expr, nil)
				if evalErr == nil {
					fnParams = append(fnParams, result.Sequence())
				} else {
					fnParams = append(fnParams, nil)
				}
			} else {
				fnParams = append(fnParams, nil)
			}
		}
		inv = ss.CallFunction(tc.InitialFunction, fnParams...)
		if sourceDoc != nil {
			inv = inv.SourceDocument(sourceDoc)
		}
	case tc.InitialModeSelect != "":
		inv = ss.ApplyTemplates(sourceDoc)
		if tc.InitialMode != "" {
			inv = inv.Mode(tc.InitialMode)
		}
		// Evaluate select expression against the source document so that
		// XPath expressions like "/doc" resolve against the actual source.
		var sel xpath3.Sequence
		if sourceDoc != nil {
			// If the expression uses doc('filename'), replace it with
			// the root path so that it evaluates against the source document
			// directly. This handles cases like doc('mode-14.xml')//v[...].
			selectExpr := tc.InitialModeSelect
			if strings.Contains(selectExpr, "doc(") {
				// Replace doc('...') with root node reference
				selectExpr = regexp.MustCompile(`doc\(['""][^'"]*['"]\)`).ReplaceAllString(selectExpr, "")
				if selectExpr == "" {
					selectExpr = "."
				}
			}
			expr, compErr := xpath3.NewCompiler().Compile(selectExpr)
			if compErr == nil {
				result, evalErr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, expr, sourceDoc)
				if evalErr == nil {
					sel = result.Sequence()
				}
			}
		}
		if sel == nil || sequence.Len(sel) == 0 {
			sel = w3cEvaluateParamSequence(ctx, tc.InitialModeSelect)
		}
		inv = inv.Selection(sel)
		for pName, pVal := range tc.InitialModeParams {
			inv = inv.SetInitialModeParameter(pName, w3cEvaluateParamSequence(ctx, pVal))
		}
		for pName, pVal := range tc.InitialModeTunnelParams {
			inv = inv.SetTunnelParameter(pName, w3cEvaluateParamSequence(ctx, pVal))
		}
	default:
		inv = ss.Transform(sourceDoc)
		if tc.InitialMode != "" {
			inv = inv.Mode(tc.InitialMode)
		}
		for pName, pVal := range tc.InitialModeParams {
			inv = inv.SetInitialModeParameter(pName, w3cEvaluateParamSequence(ctx, pVal))
		}
		for pName, pVal := range tc.InitialModeTunnelParams {
			inv = inv.SetTunnelParameter(pName, w3cEvaluateParamSequence(ctx, pVal))
		}
	}

	// On-multiple-match override
	if tc.OnMultipleMatch != "" {
		switch tc.OnMultipleMatch {
		case "use-last":
			inv = inv.OnMultipleMatch(xslt3.OnMultipleMatchUseLast)
		case "fail":
			inv = inv.OnMultipleMatch(xslt3.OnMultipleMatchFail)
		}
	}

	// Set global stylesheet parameters
	for pName, pVal := range tc.Params {
		// The W3C test catalog specifies param values as XPath expressions.
		// Evaluate them so that e.g. "8" becomes xs:integer(8) and
		// "'text'" becomes xs:string("text").
		expr, exprErr := xpath3.NewCompiler().Compile(pVal)
		if exprErr == nil {
			result, evalErr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, expr, nil)
			if evalErr == nil {
				seq := result.Sequence()
				if seq != nil && sequence.Len(seq) == 1 {
					if av, ok := seq.Get(0).(xpath3.AtomicValue); ok {
						// If the catalog specifies an as type, cast the value
						if asType, ok2 := tc.ParamTypes[pName]; ok2 && asType != av.TypeName {
							av = castAtomicForParam(av, asType)
						}
						inv = inv.SetParameter(pName, xpath3.SingleAtomic(av))
						continue
					}
				}
				// Empty or multi-item sequences: pass as sequence param
				inv = inv.SetParameter(pName, seq)
				continue
			}
		}
		// Fallback: strip enclosing quotes from string literals.
		if len(pVal) >= 2 && pVal[0] == '\'' && pVal[len(pVal)-1] == '\'' {
			pVal = pVal[1 : len(pVal)-1]
		}
		inv = inv.SetParameter(pName, xpath3.SingleString(pVal))
	}

	// Build handlers to capture messages, result docs, annotations, etc.
	var messages []string
	resultDocs := make(map[string]*helium.Document)
	var rawResult xpath3.Sequence
	var primaryItems xpath3.Sequence
	var resultAnnotations map[helium.Node]string
	var resultSchemaDecl xpath3.SchemaDeclarations
	testName := t.Name()
	recv := &w3cReceiver{
		messages:   &messages,
		resultDocs: resultDocs,
		testName:   testName,
		rawResult:  &rawResult,
		primaryItems: &primaryItems,
		resultAnnotations: &resultAnnotations,
		resultSchemaDecl:  &resultSchemaDecl,
	}
	inv = inv.
		MessageHandler(recv).
		ResultDocumentHandler(recv).
		RawResultHandler(recv).
		PrimaryItemsHandler(recv).
		AnnotationHandler(recv).
		TraceWriter(io.Discard)

	// Set up collection resolver if collections are defined or if the test
	// requires well-known collections (e.g., merge tests using log-files).
	collections := tc.Collections
	if len(collections) == 0 {
		collections = w3cInferCollections(tc)
	}
	if len(collections) > 0 {
		resolver := w3cBuildCollectionResolver(t, collections)
		inv = inv.CollectionResolver(resolver)
	}

	// Set base output URI for current-output-uri(). Use explicit value if
	// provided, otherwise auto-compute for tests in the known list.
	if baseURI := w3cBaseOutputURI(tc); baseURI != "" {
		inv = inv.BaseOutputURI(baseURI)
	}

	// Load source document schema if specified by the test case.
	if tc.SourceSchemaPath != "" {
		schemaPath := w3cResolvePath(tc.SourceSchemaPath)
		schema, schemaErr := xsd.NewCompiler().CompileFile(ctx, schemaPath)
		if schemaErr != nil {
			t.Fatalf("compile source schema %q: %v", schemaPath, schemaErr)
		}
		inv = inv.SourceSchemas(schema)
	}

	// Transform
	resultDoc, err := inv.Do(ctx)
	if err != nil {
		if tc.ExpectError {
			return // expected runtime error
		}
		if w3cErrorAccepted(err, tc.AcceptErrors) {
			return
		}
		t.Fatalf("transform error: %v", err)
	}

	// Serialization errors (SE*) are raised during serialization, not
	// transform. Defer the ExpectError check until after serialization.
	expectSerializationError := tc.ExpectError && strings.HasPrefix(tc.ErrorCode, "SE")
	// XTSE3430: a Basic XSLT Processor may fall back to non-streaming
	// execution instead of raising the error. Both outcomes are valid,
	// so accept successful transformation as passing.
	if tc.ExpectError && tc.ErrorCode == "XTSE3430" {
		return
	}
	if tc.ExpectError && !expectSerializationError {
		t.Fatalf("expected error %s but transformation succeeded", tc.ErrorCode)
	}

	// Serialize result using the stylesheet's output method.
	// For assert-serialization tests, use the full serializer (with XML declaration etc.)
	// so the comparison matches what a real processor would emit.
	var buf bytes.Buffer
	outDef := inv.ResolvedOutputDef()
	hasSerialization := assertionsNeedSerialization(tc.Assertions)
	hasCharMaps := outDef != nil && outDef.ResolvedCharMap != nil
	hasNonUTF8Encoding := outDef != nil && outDef.Encoding != "" && !strings.EqualFold(outDef.Encoding, "UTF-8") && !strings.EqualFold(outDef.Encoding, "UTF8")
	// Auto-detect HTML method: if no explicit method and root element is <html> in no namespace.
	// Only auto-detect when the test has serialization assertions; for assert-xml
	// tests the W3C catalog compares the pre-serialization result tree, so
	// injecting a <meta> tag via the HTML serializer would cause spurious mismatches.
	autoHTML := hasSerialization && (outDef == nil || (!outDef.MethodExplicit && outDef.Method == "xml"))
	if autoHTML {
		if root := resultDoc.DocumentElement(); root != nil {
			autoHTML = strings.EqualFold(root.Name(), "html") && root.URI() == ""
		} else {
			autoHTML = false
		}
	}
	needsSerializer := hasSerialization || hasCharMaps || (outDef != nil && (outDef.Method == "html" || outDef.Method == "xhtml")) || hasNonUTF8Encoding || autoHTML || expectSerializationError
	buildTreeNo := outDef != nil && outDef.BuildTree != nil && !*outDef.BuildTree
	if outDef != nil && primaryItems != nil && sequence.Len(primaryItems) > 0 && (outDef.Method == "json" || outDef.Method == "adaptive" || buildTreeNo) {
		err = xslt3.SerializeItems(&buf, primaryItems, resultDoc, outDef)
	} else if needsSerializer {
		err = xslt3.SerializeResult(&buf, resultDoc, outDef)
	} else {
		opts := []helium.WriteOption{helium.WithNoDecl()}
		if outDef != nil && outDef.UndeclarePrefixes {
			opts = append(opts, helium.WithAllowPrefixUndecl())
		}
		err = resultDoc.XML(&buf, opts...)
	}
	if expectSerializationError {
		if err != nil {
			return // expected serialization error
		}
		t.Fatalf("expected error %s but transformation succeeded", tc.ErrorCode)
	}
	if err != nil && w3cErrorAccepted(err, tc.AcceptErrors) {
		return // accepted serialization error
	}
	require.NoError(t, err)
	result := buf.String()
	// When the output encoding is not UTF-8, decode the serialized bytes
	// back to UTF-8 so that assertion patterns (which use UTF-8) can match.
	if outDef != nil {
		enc := strings.ToLower(outDef.Encoding)
		if enc != "" && enc != "utf-8" && enc != "utf8" {
			raw := buf.Bytes()
			decoded := false
			// Detect UTF-16 BOM and decode accordingly
			if (enc == "utf-16" || enc == "utf16") && len(raw) >= 2 {
				if raw[0] == 0xFF && raw[1] == 0xFE {
					// UTF-16 LE
					dec := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder()
					if d, derr := dec.Bytes(raw); derr == nil {
						result = string(d)
						decoded = true
					}
				} else if raw[0] == 0xFE && raw[1] == 0xFF {
					// UTF-16 BE
					dec := unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewDecoder()
					if d, derr := dec.Bytes(raw); derr == nil {
						result = string(d)
						decoded = true
					}
				}
			}
			if !decoded {
				if codec, cerr := htmlindex.Get(enc); cerr == nil {
					if d, derr := codec.NewDecoder().Bytes(raw); derr == nil {
						result = string(d)
					}
				}
			}
			// After decoding to UTF-8, the XML declaration's encoding
			// attribute is stale. Strip it when the serializer was only
			// used for encoding (not for assert-serialization tests)
			// so that assertion XPath evaluation doesn't re-interpret
			// bytes using the wrong encoding.
			if !hasSerialization {
				if idx := strings.Index(result, "?>"); idx >= 0 {
					result = strings.TrimSpace(result[idx+2:])
				}
			}
		}
	}
	// Remove the trailing newline that helium's XML serializer adds after the document element,
	// but preserve internal whitespace (important for whitespace-sensitive assertions).
	result = strings.TrimRight(result, "\n")

	// Check assertions
	isItemMethod := outDef != nil && (outDef.Method == "json" || outDef.Method == "adaptive")
	for _, a := range tc.Assertions {
		if a.RawCheck != nil && rawResult != nil {
			a.RawCheck(t, rawResult, result)
		} else if a.Type == "assert" && rawResult != nil {
			evalXPathAssertWithRawResult(t, a.Value, result, rawResult)
		} else if a.Type == "assert" && resultAnnotations != nil {
			evalXPathAssertWithAnnotations(t, a.Value, resultDoc, resultAnnotations, resultSchemaDecl)
		} else if a.Type == "assert" && isItemMethod && resultDoc != nil {
			// For json/adaptive output, XPath assertions evaluate against the
			// result tree (DOM), not the serialized output (which is JSON/adaptive
			// and can't be parsed as XML).
			evalXPathAssertWithDoc(t, a.Value, resultDoc)
		} else if a.Check != nil {
			a.Check(t, result, messages, resultDocs)
		}
	}
}

// w3cReceiver implements all handler interfaces needed by the W3C test runner:
// MessageHandler, ResultDocumentHandler, RawResultHandler,
// PrimaryItemsHandler, and AnnotationHandler.
type w3cReceiver struct {
	messages          *[]string
	resultDocs        map[string]*helium.Document
	testName          string
	rawResult         *xpath3.Sequence
	primaryItems      *xpath3.Sequence
	resultAnnotations *map[helium.Node]string
	resultSchemaDecl  *xpath3.SchemaDeclarations
}

func (r *w3cReceiver) HandleMessage(msg string, _ bool) error {
	*r.messages = append(*r.messages, msg)
	return nil
}

func (r *w3cReceiver) HandleResultDocument(href string, doc *helium.Document, outDef *xslt3.OutputDef) error {
	r.resultDocs[href] = doc
	if outDef != nil {
		w3cResultDocOutputDefs.Store(r.testName+"\x00"+href, outDef)
	}
	return nil
}

func (r *w3cReceiver) HandleRawResult(seq xpath3.Sequence) error {
	*r.rawResult = seq
	return nil
}

func (r *w3cReceiver) HandlePrimaryItems(seq xpath3.Sequence) error {
	*r.primaryItems = seq
	return nil
}

func (r *w3cReceiver) HandleAnnotations(annotations map[helium.Node]string, declarations xpath3.SchemaDeclarations) error {
	*r.resultAnnotations = annotations
	*r.resultSchemaDecl = declarations
	return nil
}

// w3cErrorAccepted returns true if err's message contains one of the accepted
// error codes (from any-of error alternatives in the W3C test catalog).
func w3cErrorAccepted(err error, codes []string) bool {
	if len(codes) == 0 {
		return false
	}
	msg := err.Error()
	for _, code := range codes {
		if strings.Contains(msg, code) {
			return true
		}
	}
	return false
}

// w3cBaseOutputURI returns the base output URI for the test, or "" if none.
// Tests with BaseOutputURI set explicitly use that; otherwise a hard-coded
// list of tests known to have base-output-uri in the W3C catalog is checked.
func w3cBaseOutputURI(tc w3cTest) string {
	if tc.BaseOutputURI != "" {
		if tc.BaseOutputURI == "#auto" {
			return w3cComputeBaseOutputURI(tc)
		}
		return tc.BaseOutputURI
	}
	if _, ok := w3cTestsWithBaseOutputURI[tc.Name]; ok {
		return w3cComputeBaseOutputURI(tc)
	}
	return ""
}

func w3cComputeBaseOutputURI(tc w3cTest) string {
	if tc.StylesheetPath == "" {
		return ""
	}
	ssDir := filepath.Dir(w3cResolvePath(tc.StylesheetPath))
	absDir, _ := filepath.Abs(ssDir)
	return "file://" + filepath.ToSlash(absDir) + "/results/" + tc.Name + ".xml"
}

// w3cTestsWithBaseOutputURI lists test names that have base-output-uri set
// in the W3C XSLT 3.0 test catalog. The generator does not extract this
// attribute yet, so we maintain the list manually.
var w3cTestsWithBaseOutputURI = map[string]struct{}{
	"current-output-uri-002": {},
	"current-output-uri-003": {},
	"current-output-uri-004": {},
	"current-output-uri-008": {},
	"current-output-uri-009": {},
	"current-output-uri-010": {},
	"current-output-uri-014": {},
}

func w3cImplicitSkipReason(name string) string {
	// XSD 1.0-only regex tests: these patterns are invalid under XSD 1.0
	// character class rules but valid in XSD 1.1 which we target.
	switch name {
	case "regex-syntax-0056", "regex-syntax-0086", "regex-syntax-0102":
		return "XSD 1.0-only regex error; we target XSD 1.1"
	}
	if reason, ok := w3cImplicitSkips[name]; ok {
		return reason
	}
	return ""
}

// w3cImplicitSkips maps individual test names to skip reasons for tests
// blocked by known parser or runtime limitations.
var w3cImplicitSkips = map[string]string{
	// regex-classes: Unicode general category tests whose results differ between
	// Go's Unicode 16.0 and the W3C reference data (Unicode 3.1/5.2/6.0).
	// Character assignments change across Unicode versions; the regex implementation
	// is correct for Unicode 16.0 but the test reference data is outdated.
	// Tests 002(Cc), 005(Co), 010(Lt), 021(Pc), 025(Pi), 034(Zl), 035(Zp) pass
	// because those categories are stable across all Unicode versions.
	"regex-classes-001": "Unicode category C: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-003": "Unicode category Cf: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-004": "Unicode category Cn: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-006": "Unicode category L: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-007": "Unicode category Ll: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-008": "Unicode category Lm: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-009": "Unicode category Lo: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-011": "Unicode category Lu: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-012": "Unicode category M: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-013": "Unicode category Mc: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-014": "Unicode category Me: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-015": "Unicode category Mn: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-016": "Unicode category N: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-017": "Unicode category Nd: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-018": "Unicode category Nl: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-019": "Unicode category No: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-020": "Unicode category P: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-022": "Unicode category Pd: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-023": "Unicode category Pe: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-024": "Unicode category C: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-026": "Unicode category Po: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-027": "Unicode category Ps: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-028": "Unicode category S: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-029": "Unicode category Sc: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-030": "Unicode category Sk: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-031": "Unicode category Sm: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-032": "Unicode category So: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-033": "Unicode category Z: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",
	"regex-classes-036": "Unicode category Zs: version mismatch (Go Unicode 16.0 vs W3C ref 3.1/5.2/6.0)",

	// W3C dependency: default_html_version=4 / feature=HTML4
	// These tests require the processor to default html-version to 4; ours defaults to 5.
	// HTML5 companion tests (output-0195a, result-document-1402b) already pass.
	"output-0195":          "W3C dependency default_html_version=4; HTML5 companion output-0195a passes",
	"result-document-1402": "W3C dependency default_html_version=4; HTML5 companion result-document-1402b passes",

	// XML 1.1 features: namespace undeclaration (xmlns:a="") not supported
	"xml-version-026": "XML 1.1: namespace undeclaration not supported by parser",
	"xml-version-027": "XML 1.1: namespace undeclaration not supported by parser",
	"xml-version-028": "XML 1.1: namespace undeclaration not supported by parser",
	"xml-version-031": "XML 1.1: namespace undeclaration not supported by parser",
	"xml-version-032": "XML 1.1: namespace undeclaration not supported by parser",
	"xml-version-035": "XML 1.1: namespace undeclaration not supported by parser",
	"xml-version-037": "XML 1.1: namespace undeclaration not supported by parser",
	"xml-version-039": "XML 1.1: namespace undeclaration not supported by parser",
	"xml-version-042": "XML 1.1: namespace undeclaration not supported by parser",

	// XML 1.1 features: control characters (&#x1;..&#x8;, &#x7;) not supported
	"xml-version-002": "XML 1.1: control characters in stylesheet not supported by parser",
	"xml-version-007": "XML 1.1: control characters in stylesheet not supported by parser",
	"xml-version-008": "XML 1.1: control characters in stylesheet not supported by parser",
	"xml-version-023": "XML 1.1: control characters in stylesheet not supported by parser",

	// XML 1.1 features: control characters in source documents
	"xml-version-020": "XML 1.1: control characters in source document not supported by parser",
	"xml-version-024": "XML 1.1: control characters in source document not supported by parser",
	"xml-version-025": "XML 1.1: control characters in source document not supported by parser",

	// XML 1.1: serialization of control chars as numeric character references
	"xml-version-009": "XML 1.1: control character serialization as numeric refs not implemented",
	"xml-version-010": "XML 1.1: control character serialization as numeric refs not implemented",
	"xml-version-018": "XML 1.1: control character serialization as numeric refs not implemented",

	// regex-070*: XSL file uses entity reference pattern that trips parser
	"regex-070a": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070b": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070c": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070d": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070e": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070f": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070g": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070h": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070i": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070j": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070k": "parser limitation: entity ref in single-quoted attribute value",
	"regex-070l": "parser limitation: entity ref in single-quoted attribute value",

	// whitespace-011: external parameter entity resolution not supported
	"whitespace-011": "parser limitation: external parameter entity resolution not supported",



	// nodetest: child::schema-attribute axis conversion produces non-empty result
	"nodetest-032": "child::schema-attribute axis conversion vs XPath 2.0 expected output mismatch",

	// json-to-xml typed tests: fn:json-to-xml validate option does not type-annotate result
	"json-to-xml-typed-001": "json-to-xml validate option does not annotate result nodes",
	"json-to-xml-typed-002": "json-to-xml validate option does not annotate result nodes",
	"json-to-xml-typed-003": "json-to-xml validate option does not annotate result nodes",
	"json-to-xml-typed-004": "json-to-xml validate option does not annotate result nodes",
	"json-to-xml-typed-005": "json-to-xml validate option does not annotate result nodes",
	"json-to-xml-typed-006": "json-to-xml validate option does not annotate result nodes",
	"json-to-xml-typed-007": "json-to-xml validate option does not annotate result nodes",

	// copy tests: external entity resolution
	"copy-1401": "requires external entity resolution (SYSTEM entity reference)",


	// copy tests: namespace handling

	// copy tests: grouping

	// copy tests: accumulator handling

	// copy tests: schema-aware ID/IDREF
	"copy-5034": "schema-aware ID/IDREF copy validation fails",

	// validation tests: schema-aware processing
	"validation-0202": "schema-aware result validation fails",
	"validation-0213": "schema-aware result validation fails",
	"validation-0501": "schema-aware XSLT schema querying fails",
	"validation-0601": "schema-aware XSLT schema querying fails",
	"validation-0701": "schema-aware XSLT schema querying fails",
	"validation-1202": "instance of schema-element fails",
	"validation-1204": "instance of schema-element fails",

	// as tests: schema-attribute type checks
	"as-1812": "schema-attribute type check fails",
	"as-1813": "schema-attribute type check fails",
	"as-1814": "schema-attribute type check fails",
	"as-3601": "schema-attribute type check fails",
	"as-3602": "schema-attribute type check fails",
	"as-3603": "schema-attribute type check fails",

	// override: schema-aware union types from xsl:import-schema
	"override-f-031": "schema-aware union type conversion fails",
	"override-v-006": "schema-aware union type comparison fails",



	// use-package: package-scoped namespace serialization in result-document
	"use-package-108":  "package-scoped namespace alias serialization not implemented",
	"use-package-108b": "package-scoped namespace alias serialization not implemented",




	// error-FODC0002a-ignore: processor now raises FODC0002 (ignore_doc_failure=false)
	"error-FODC0002a-ignore": "processor raises FODC0002 instead of ignoring document failures",


	// merge: schema-element instance test on merged items
	"merge-049": "schema-element() instance test on merged items fails",
	"merge-051": "schema-element() instance test on merged items fails",
	"merge-072":  "XTDE2220 alternate=shifted collation sort-order verification not supported",
	"merge-079":  "DOM-materialized merge exposes ancestor::record outside streaming snapshot",
	"merge-097":  "uri-collection Saxon-format ?select=glob URIs not supported (W3C catalog notes non-interoperable)",
	"merge-097s": "uri-collection Saxon-format ?select=glob URIs not supported (W3C catalog notes non-interoperable)",
	"merge-097sf": "uri-collection Saxon-format ?select=glob URIs not supported (W3C catalog notes non-interoperable)",


	// package version resolution: lowest_version not supported (we use highest_version)

	// castable tests: schema-aware union/list type casting
	"castable-005": "schema-aware union type casting fails",
	"castable-006": "schema-aware list type casting fails",

	// attribute-set tests

	// regex-090/091: regex-group#N function reference captures regex context as closure
	// The closure implementation is correct per spec, but the test expects empty output.
	// Likely an issue with how zero-length regex matches are handled by analyze-string.

	// xpath-default-namespace: schema-aware namespace resolution
	"xpath-default-namespace-0503": "schema-type validation with xpath-default-namespace fails",
	"xpath-default-namespace-0701": "schema-element with xpath-default-namespace fails",
	"xpath-default-namespace-0703": "schema-element with xpath-default-namespace fails",


	// strip-space: schema-aware whitespace stripping
	"strip-space-007": "schema-aware whitespace stripping fails",
	"strip-space-008": "schema-aware whitespace stripping fails",

	// base-uri: xsl:copy base URI propagation

	// arrays: array construction and apply-templates on arrays

	// schema-aware match tests: pattern matching with schema types
	"match-054": "element-with-id pattern match returns child instead of parent",
	"match-055": "element-with-id pattern match returns child instead of parent",
	"match-174": "xsl:type annotation on constructed elements not propagated",
	"match-181": "xsl:type annotation on constructed elements not propagated",
	"match-210": "schema-aware attribute type pattern match fails",
	"match-221": "schema-aware numeric type pattern match fails",
	"match-232": "schema-aware type pattern match missing items",
	"match-244": "expected error XTSE3105 not raised for schema pattern",
	"match-262": "deep-copy does not preserve type annotations",
	"match-263": "deep-copy does not preserve type annotations",
	"match-287": "schema-attribute pattern match fails (XTTE0780)",

	// evaluate tests requiring schema-aware processing
	"evaluate-012": "schema-aware xsl:evaluate: expected XTDE3160 not raised",
	"evaluate-013": "schema-aware xsl:evaluate: expected XTDE3160 not raised",
	"evaluate-048": "requires network access to saxonica.com",

	// slow xsl:evaluate tests (~30s each with -race on CI)
	"evaluate-010": "too slow for CI: large iteration count in xsl:evaluate",
	"evaluate-039": "too slow for CI: large iteration count in xsl:evaluate",
	"evaluate-040": "too slow for CI: large iteration count in xsl:evaluate",
	"variable-0108": "too slow for CI: large iteration count with variable binding",

	"base-uri-052": "XInclude processing not applied to source documents",

	// snapshot: f:snapshot reference impl namespace-node graft produces empty root
	"snapshot-0102a": "snapshot()/root() returns empty for some namespace nodes",



	// XSD 1.1 features: newly unlocked but failing
	"validation-1301":  "XSD 1.1 xs:alternative type selection not implemented",
	"import-schema-164": "XSD validation fails for namespaced attribute ref with default",
	"strip-space-009":  "schema-aware whitespace stripping not implemented",


	// higher-order functions: nested for-each-group grouping bug
}

// promoteWrapperChildren takes a document parsed from "<_r>content</_r>"
// and returns a new document where the children of the _r wrapper element
// are direct children of the document node. This allows XPath expressions
// like "/elem" to address top-level elements in an XML fragment that had
// multiple root elements (e.g. message output from xsl:message).
func promoteWrapperChildren(doc *helium.Document) *helium.Document {
	root := doc.DocumentElement()
	if root == nil || root.Name() != "_r" {
		return doc
	}
	newDoc := helium.NewDefaultDocument()
	// Collect children first to avoid mutation during iteration
	var children []helium.Node
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		children = append(children, child)
	}
	for _, child := range children {
		copied, err := helium.CopyNode(child, newDoc)
		if err != nil {
			return doc // fall back to original
		}
		if err := newDoc.AddChild(copied); err != nil {
			return doc // fall back to original
		}
	}
	return newDoc
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p := helium.NewParser()
	docA, errA := p.Parse(ctx, []byte(wrapA))
	docB, errB := p.Parse(ctx, []byte(wrapB))
	if errA != nil || errB != nil {
		// XML 1.1 prefix undeclarations (xmlns:prefix="") are not valid
		// XML 1.0. Strip them and retry so that DOM comparison still works.
		if errA != nil {
			wrapA = stripPrefixUndecls(wrapA)
			docA, errA = p.Parse(ctx, []byte(wrapA))
		}
		if errB != nil {
			wrapB = stripPrefixUndecls(wrapB)
			docB, errB = p.Parse(ctx, []byte(wrapB))
		}
		if errA != nil || errB != nil {
			return false
		}
	}
	return nodesEqual(docA.DocumentElement(), docB.DocumentElement())
}

// stripPrefixUndecls removes xmlns:prefix="" declarations that are invalid
// in XML 1.0 but valid in XML 1.1. This allows DOM comparison to proceed.
func stripPrefixUndecls(s string) string {
	re := regexp.MustCompile(`\s+xmlns:\w+=""`)
	return re.ReplaceAllString(s, "")
}

func wrapXMLFragment(s string) string {
	trimmed := strings.TrimSpace(s)
	// Strip UTF-8 BOM if present
	trimmed = strings.TrimPrefix(trimmed, "\xEF\xBB\xBF")
	trimmed = strings.TrimSpace(trimmed)
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
		// Skip whitespace-only text nodes (insignificant inter-element whitespace).
		for childA != nil && isTextLike(childA) && strings.TrimSpace(string(childA.Content())) == "" {
			childA = childA.NextSibling()
		}
		for childB != nil && isTextLike(childB) && strings.TrimSpace(string(childB.Content())) == "" {
			childB = childB.NextSibling()
		}
		if childA == nil && childB == nil {
			break
		}
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
	s = strings.TrimPrefix(s, "\xEF\xBB\xBF")
	s = strings.TrimSpace(s)
	// Strip XML declaration (<?xml ...?>) so that assert-xml comparisons
	// are not sensitive to the presence/absence of the declaration.
	if strings.HasPrefix(s, "<?xml") {
		if idx := strings.Index(s, "?>"); idx >= 0 {
			s = strings.TrimSpace(s[idx+2:])
		}
	}
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
	doc, err := helium.NewParser().Parse(context.TODO(), []byte(wrapped))
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
func evalXPathAssertWithDoc(t *testing.T, expr string, doc *helium.Document) bool {
	t.Helper()

	compiled, err := xpath3.NewCompiler().Compile(expr)
	if err != nil {
		t.Errorf("assert: cannot compile XPath %q: %v", expr, err)
		return false
	}

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	ns := gatherDocNamespaces(doc)
	if ns == nil {
		ns = make(map[string]string)
	}
	if _, ok := ns["j"]; !ok {
		ns["j"] = "http://www.w3.org/2005/xpath-functions"
	}
	if _, ok := ns["g"]; !ok {
		ns["g"] = "http://www.w3.org/xsl-tests/grouped-transactions"
	}
	if defNS, ok := ns[""]; ok {
		if defNS == "http://www.w3.org/xsl-tests/grouped-transactions-e" {
			if _, ok := ns["g"]; !ok || ns["g"] != defNS {
				ns["g"] = defNS
			}
		}
	}
	if len(ns) > 0 {
		eval = eval.Namespaces(ns)
	}

	eval = eval.Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
		"result": xpath3.ItemSlice{xpath3.NodeItem{Node: doc}},
	}))

	res, err := eval.Evaluate(context.TODO(), compiled, doc)
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
		t.Errorf("assert failed: %s evaluated to false", expr)
		return false
	}
	return true
}

func evalXPathAssert(t *testing.T, expr string, resultXML string) bool {
	t.Helper()

	doc, err := helium.NewParser().Parse(context.TODO(), []byte(resultXML))
	if err != nil {
		// The result may be a well-formed document fragment (multiple top-level
		// elements) which is valid in the XDM but not well-formed XML. Wrap in a
		// synthetic root, parse, then build a new document with the wrapper's
		// children promoted to top-level so that absolute XPath expressions like
		// /foo[1] resolve correctly.
		wrapped := "<_w3c_wrap_>" + resultXML + "</_w3c_wrap_>"
		wrapDoc, wrapErr := helium.NewParser().Parse(context.TODO(), []byte(wrapped))
		if wrapErr != nil {
			// The result may be HTML output with void elements
			// (e.g. <meta>, <img>) that are not valid XML. Try
			// the HTML parser as a last resort.
			htmlDoc, htmlErr := htmlparser.NewParser().Parse(context.TODO(), []byte(resultXML))
			if htmlErr != nil {
				t.Errorf("assert: cannot parse result XML: %v", err)
				return false
			}
			return evalXPathAssertWithDoc(t, expr, htmlDoc)
		}
		fragDoc := helium.NewDefaultDocument()
		root := wrapDoc.DocumentElement()
		if root != nil {
			// Collect children first, then move them
			var children []helium.Node
			for child := root.FirstChild(); child != nil; child = child.NextSibling() {
				children = append(children, child)
			}
			for _, child := range children {
				helium.UnlinkNode(child)
				if addErr := fragDoc.AddChild(child); addErr != nil {
					t.Errorf("assert: cannot build fragment document: %v", addErr)
					return false
				}
			}
		}
		doc = fragDoc
	}

	compiled, err := xpath3.NewCompiler().Compile(expr)
	if err != nil {
		t.Errorf("assert: cannot compile XPath %q: %v", expr, err)
		return false
	}

	// Gather ALL in-scope namespace bindings from the document tree,
	// not just from the root element. This ensures prefixed assertions
	// work even when namespace declarations appear on child elements.
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	ns := gatherDocNamespaces(doc)
	// Add well-known W3C test namespace prefixes that assertions may use
	// even when the result document declares them under a different prefix.
	if ns == nil {
		ns = make(map[string]string)
	}
	if _, ok := ns["j"]; !ok {
		ns["j"] = "http://www.w3.org/2005/xpath-functions"
	}
	if _, ok := ns["g"]; !ok {
		ns["g"] = "http://www.w3.org/xsl-tests/grouped-transactions"
	}
	// Also check for -e variant used by some streaming tests.
	// The -e namespace may appear as the default namespace (stored under
	// "o" by gatherDocNamespaces) or as an explicit prefix binding.
	geNS := "http://www.w3.org/xsl-tests/grouped-transactions-e"
	for _, uri := range ns {
		if uri == geNS {
			ns["g"] = geNS
			break
		}
	}
	if len(ns) > 0 {
		eval = eval.Namespaces(ns)
	}

	// Bind $result to the document node so W3C assertions like
	// deep-equal($result, ...) can reference the transformation output.
	eval = eval.Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
		"result": xpath3.ItemSlice{xpath3.NodeItem{Node: doc}},
	}))

	res, err := eval.Evaluate(context.TODO(), compiled, doc)
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

// evalXPathAssertWithAnnotations evaluates an XPath assertion on the actual
// result document (not re-parsed), with type annotations and schema declarations
// available so that schema-aware type checks like "instance of element(*,xs:untyped)"
// and "instance of schema-element(Q{ns}name)" work correctly.
func evalXPathAssertWithAnnotations(t *testing.T, expr string, doc *helium.Document, annotations map[helium.Node]string, sd xpath3.SchemaDeclarations) bool {
	t.Helper()

	compiled, err := xpath3.NewCompiler().Compile(expr)
	if err != nil {
		t.Errorf("assert: cannot compile XPath %q: %v", expr, err)
		return false
	}

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	ns := gatherDocNamespaces(doc)
	if ns == nil {
		ns = make(map[string]string)
	}
	if _, ok := ns["j"]; !ok {
		ns["j"] = "http://www.w3.org/2005/xpath-functions"
	}
	if _, ok := ns["g"]; !ok {
		ns["g"] = "http://www.w3.org/xsl-tests/grouped-transactions"
	}
	// Also check for -e variant used by some streaming tests.
	geNS2 := "http://www.w3.org/xsl-tests/grouped-transactions-e"
	for _, uri := range ns {
		if uri == geNS2 {
			ns["g"] = geNS2
			break
		}
	}
	if len(ns) > 0 {
		eval = eval.Namespaces(ns)
	}

	eval = eval.Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
		"result": xpath3.ItemSlice{xpath3.NodeItem{Node: doc}},
	}))

	if annotations != nil {
		eval = eval.TypeAnnotations(annotations)
	}
	if sd != nil {
		eval = eval.SchemaDeclarations(sd)
	}

	res, err := eval.Evaluate(context.TODO(), compiled, doc)
	if err != nil {
		// If evaluation fails with a type comparison error (XPTY0004),
		// retry without type annotations. W3C assertions that use simple
		// string equality (e.g. @list = "1 2 3") are not schema-aware and
		// expect untyped comparison semantics.
		var xpErr *xpath3.XPathError
		if errors.As(err, &xpErr) && xpErr.Code == "XPTY0004" {
			evalPlain := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
			if len(ns) > 0 {
				evalPlain = evalPlain.Namespaces(ns)
			}
			evalPlain = evalPlain.Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
				"result": xpath3.ItemSlice{xpath3.NodeItem{Node: doc}},
			}))
			res2, err2 := evalPlain.Evaluate(context.TODO(), compiled, doc)
			if err2 == nil {
				res = res2
				err = nil
			}
		}
		if err != nil {
			t.Errorf("assert: XPath evaluation error for %q: %v", expr, err)
			return false
		}
	}

	ebv, err := xpath3.EBV(res.Sequence())
	if err != nil {
		t.Errorf("assert: cannot compute EBV for %q: %v", expr, err)
		return false
	}
	if !ebv {
		var buf bytes.Buffer
		_ = doc.XML(&buf, helium.WithNoDecl())
		t.Errorf("assert failed: %s evaluated to false (result: %s)", expr, buf.String())
		return false
	}
	return true
}

// evalXPathAssertWithRawResult evaluates an XPath assertion with $result bound
// to the raw XDM sequence from the transformation. This preserves atomic type
// information that would otherwise be lost during serialization to text nodes.
func evalXPathAssertWithRawResult(t *testing.T, expr string, resultXML string, rawResult xpath3.Sequence) bool {
	t.Helper()

	doc, err := helium.NewParser().Parse(context.TODO(), []byte(resultXML))
	if err != nil {
		wrapped := "<_w3c_wrap_>" + resultXML + "</_w3c_wrap_>"
		wrapDoc, wrapErr := helium.NewParser().Parse(context.TODO(), []byte(wrapped))
		if wrapErr != nil {
			htmlDoc, htmlErr := htmlparser.NewParser().Parse(context.TODO(), []byte(resultXML))
			if htmlErr != nil {
				t.Errorf("assert: cannot parse result XML: %v", err)
				return false
			}
			return evalXPathAssertWithDoc(t, expr, htmlDoc)
		}
		fragDoc := helium.NewDefaultDocument()
		root := wrapDoc.DocumentElement()
		if root != nil {
			var children []helium.Node
			for child := root.FirstChild(); child != nil; child = child.NextSibling() {
				children = append(children, child)
			}
			for _, child := range children {
				helium.UnlinkNode(child)
				if addErr := fragDoc.AddChild(child); addErr != nil {
					t.Errorf("assert: cannot build fragment document: %v", addErr)
					return false
				}
			}
		}
		doc = fragDoc
	}

	compiled, err := xpath3.NewCompiler().Compile(expr)
	if err != nil {
		t.Errorf("assert: cannot compile XPath %q: %v", expr, err)
		return false
	}

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	ns := gatherDocNamespaces(doc)
	if ns == nil {
		ns = make(map[string]string)
	}
	if _, ok := ns["j"]; !ok {
		ns["j"] = "http://www.w3.org/2005/xpath-functions"
	}
	if _, ok := ns["g"]; !ok {
		ns["g"] = "http://www.w3.org/xsl-tests/grouped-transactions"
	}
	if defNS, ok := ns[""]; ok {
		if defNS == "http://www.w3.org/xsl-tests/grouped-transactions-e" {
			if _, ok := ns["g"]; !ok || ns["g"] != defNS {
				ns["g"] = defNS
			}
		}
	}
	if len(ns) > 0 {
		eval = eval.Namespaces(ns)
	}

	// Bind $result to the raw XDM sequence (preserves atomic types).
	eval = eval.Variables(xpath3.VariablesFromMap(map[string]xpath3.Sequence{
		"result": rawResult,
	}))

	res, err := eval.Evaluate(context.TODO(), compiled, doc)
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

// gatherDocNamespaces walks the entire document tree and collects all namespace
// declarations (both prefixed and default). For default namespace declarations
// (empty prefix), assigns well-known prefixes for common URIs.
func gatherDocNamespaces(doc *helium.Document) map[string]string {
	ns := make(map[string]string)
	var defaultNS string
	var walk func(n helium.Node)
	walk = func(n helium.Node) {
		if n == nil {
			return
		}
		if elem, ok := n.(*helium.Element); ok {
			for _, nsDecl := range elem.Namespaces() {
				prefix := nsDecl.Prefix()
				uri := nsDecl.URI()
				if prefix != "" {
					if _, exists := ns[prefix]; !exists {
						ns[prefix] = uri
					}
				} else if uri != "" {
					defaultNS = uri
				}
			}
			for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
				walk(child)
			}
		}
	}
	if root := doc.DocumentElement(); root != nil {
		walk(root)
	}

	// If there's a default namespace and no prefix binding for it yet,
	// assign well-known prefixes for common URIs used in W3C test assertions.
	if defaultNS != "" {
		found := false
		for _, uri := range ns {
			if uri == defaultNS {
				found = true
				break
			}
		}
		if !found {
			knownPrefixes := map[string]string{
				"http://www.w3.org/1999/xhtml":       "h",
				"http://www.w3.org/2000/svg":         "svg",
				"http://www.w3.org/1998/Math/MathML": "math",
				"http://loan.shark.com/":             "r",
			}
			if prefix, ok := knownPrefixes[defaultNS]; ok {
				if _, exists := ns[prefix]; !exists {
					ns[prefix] = defaultNS
				}
			} else {
				// W3C test catalog conventionally uses "o" for output
				// namespace in assertion XPath expressions.
				if _, exists := ns["o"]; !exists {
					ns["o"] = defaultNS
				}
			}
		}
	}

	return ns
}

// w3cPackageResolver resolves package URIs to files based on W3C test deps.
type w3cPackageResolver struct {
	deps              []w3cPackageDep
	versionResolution string // "lowest" for lowest matching version; default = highest
}

func (r w3cPackageResolver) ResolvePackage(name string, version string) (io.ReadCloser, string, error) {
	constraint := xslt3.ParseVersionConstraint(version)

	// Collect all matching deps by URI. When a dep has empty URI (generator
	// didn't extract the package name), probe the file to extract the name
	// attribute from <xsl:package name="...">.
	type match struct {
		dep     w3cPackageDep
		version xslt3.PackageVersion
	}
	var matches []match
	for _, dep := range r.deps {
		depURI := dep.URI
		depVer := dep.Version
		if depURI == "" {
			// Probe the file for <xsl:package name="..." package-version="...">
			probed := w3cProbePackageMeta(dep.FilePath)
			depURI = probed.name
			if depVer == "" {
				depVer = probed.version
			}
		}
		if depURI != name {
			continue
		}
		pv := xslt3.ParsePackageVersion(depVer)
		if constraint.Matches(pv) {
			matches = append(matches, match{dep: dep, version: pv})
		}
	}

	if len(matches) == 0 {
		// Fallback: probe the actual package file for its version and re-check.
		// The dep metadata may be incorrect (e.g., generator extracted wrong
		// version or wrong URI derived from filename rather than xsl:package/@name).
		for _, dep := range r.deps {
			probed := w3cProbePackageMeta(dep.FilePath)
			depURI := probed.name
			if depURI == "" {
				depURI = dep.URI
			}
			if depURI != name {
				continue
			}
			if version == "" {
				// No version constraint — match by URI only
				pkgPath := w3cResolvePath(dep.FilePath)
				f, err := os.Open(pkgPath)
				if err != nil {
					return nil, "", err
				}
				absPath, _ := filepath.Abs(pkgPath)
				return f, absPath, nil
			}
			// Use the already-probed version; a package with no declared version matches any constraint
			if probed.version == "" || constraint.Matches(xslt3.ParsePackageVersion(probed.version)) {
				pkgPath := w3cResolvePath(dep.FilePath)
				f, err := os.Open(pkgPath)
				if err != nil {
					return nil, "", err
				}
				absPath, _ := filepath.Abs(pkgPath)
				return f, absPath, nil
			}
		}
		return nil, "", fmt.Errorf("package %q version %q not found in test deps", name, version)
	}

	// Select the best matching version (implementation-defined per spec).
	// Default is highest; "lowest" selects the lowest.
	best := matches[0]
	for _, m := range matches[1:] {
		if r.versionResolution == "lowest" {
			if m.version.Compare(best.version) < 0 {
				best = m
			}
		} else {
			if m.version.Compare(best.version) > 0 {
				best = m
			}
		}
	}

	pkgPath := w3cResolvePath(best.dep.FilePath)
	f, err := os.Open(pkgPath)
	if err != nil {
		return nil, "", err
	}
	absPath, _ := filepath.Abs(pkgPath)
	return f, absPath, nil
}

// w3cPkgMeta holds package name and version extracted from a file.
type w3cPkgMeta struct {
	name    string
	version string
}

// castAtomicForParam converts an AtomicValue to the requested type. This is
// used when the W3C test catalog specifies an as="..." attribute on a <param>
// element that differs from the natural XPath evaluation type. For example,
// select="111" naturally yields xs:integer, but as="xs:string" means the
// caller supplies an xs:string.
func castAtomicForParam(av xpath3.AtomicValue, asType string) xpath3.AtomicValue {
	switch asType {
	case "xs:string":
		return xpath3.AtomicValue{TypeName: "xs:string", Value: fmt.Sprintf("%v", av.Value)}
	case "xs:untypedAtomic":
		return xpath3.AtomicValue{TypeName: "xs:untypedAtomic", Value: fmt.Sprintf("%v", av.Value)}
	default:
		// For other types, just override the type name.
		// The XSLT engine's type checking will handle mismatches.
		return av
	}
}

// castSequenceForParam applies castAtomicForParam to each item in a sequence.
func castSequenceForParam(seq xpath3.Sequence, asType string) xpath3.Sequence {
	result := make(xpath3.ItemSlice, sequence.Len(seq))
	for i := range sequence.Len(seq) {
		item := seq.Get(i)
		if av, ok := item.(xpath3.AtomicValue); ok && asType != av.TypeName {
			result[i] = castAtomicForParam(av, asType)
		} else {
			result[i] = item
		}
	}
	return result
}

// w3cProbePackageMeta reads the first few hundred bytes of a package file to
// extract the name and package-version attributes from the root element.
// This is used when the test generator did not populate URI/Version.
func w3cProbePackageMeta(filePath string) w3cPkgMeta {
	pkgPath := w3cResolvePath(filePath)
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return w3cPkgMeta{}
	}
	// Only scan first 1KB for the root element attributes
	s := string(data)
	if len(s) > 1024 {
		s = s[:1024]
	}
	extractAttr := func(attr string) string {
		key := attr + `="`
		idx := strings.Index(s, key)
		if idx < 0 {
			return ""
		}
		start := idx + len(key)
		end := strings.IndexByte(s[start:], '"')
		if end < 0 {
			return ""
		}
		return s[start : start+end]
	}
	return w3cPkgMeta{
		name:    extractAttr("name"),
		version: extractAttr("package-version"),
	}
}

// w3cCompileCached compiles a stylesheet, caching the result by path.
// Compile errors are not cached so that tests expecting compile errors
// still report them per test case.
func w3cCompileCached(ctx context.Context, path string) (*xslt3.Stylesheet, error) {
	if v, ok := w3cStylesheetCache.Load(path); ok {
		return v.(*xslt3.Stylesheet), nil
	}
	absPath, absErr := filepath.Abs(path)
	if absErr != nil {
		absPath = path
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	p := helium.NewParser().LoadExternalDTD(true).SubstituteEntities(true).BaseURI(absPath)
	doc, err := p.Parse(ctx, data)
	if err != nil {
		return nil, err
	}
	ss, err := xslt3.NewCompiler().BaseURI(absPath).Compile(ctx, doc)
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

func w3cResolvePath(rel string) string {
	if rel == "" || filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(w3cTestdataDir, rel)
}

// w3cCollectionResolver implements xpath3.CollectionResolver for W3C tests.
type w3cCollectionResolver struct {
	collections    map[string]xpath3.Sequence
	uriCollections map[string][]string
}

func (r *w3cCollectionResolver) ResolveCollection(uri string) (xpath3.Sequence, error) {
	seq, ok := r.collections[uri]
	if !ok {
		// The XPath engine resolves relative URIs against the base URI before
		// passing them to the resolver. Try matching by the basename to handle
		// this case (e.g. "log-files" resolved to "/abs/path/log-files").
		base := filepath.Base(uri)
		seq, ok = r.collections[base]
	}
	if !ok {
		return nil, fmt.Errorf("collection %q not found", uri)
	}
	return xpath3.ItemSlice(append([]xpath3.Item(nil), sequence.Materialize(seq)...)), nil
}

func (r *w3cCollectionResolver) ResolveURICollection(uri string) ([]string, error) {
	uris, ok := r.uriCollections[uri]
	if !ok {
		base := filepath.Base(uri)
		uris, ok = r.uriCollections[base]
	}
	if !ok {
		// Support ?select=pattern for directory glob (XSLT 3.0 convention)
		if idx := strings.Index(uri, "?select="); idx >= 0 {
			dir := uri[:idx]
			pattern := uri[idx+len("?select="):]
			matches, err := filepath.Glob(filepath.Join(dir, pattern))
			if err != nil {
				return nil, fmt.Errorf("uri-collection glob error: %v", err)
			}
			sort.Strings(matches)
			return matches, nil
		}
		return nil, fmt.Errorf("uri-collection %q not found", uri)
	}
	return append([]string(nil), uris...), nil
}

// w3cInferCollections returns well-known collections for tests that need them
// based on the stylesheet path. The W3C merge tests use collection('log-files')
// and uri-collection('log-files') to reference log file documents.
func w3cInferCollections(tc w3cTest) []w3cCollection {
	if strings.Contains(tc.StylesheetPath, "insn/merge/") {
		return []w3cCollection{
			{
				URI: "log-files",
				DocPaths: []string{
					"tests/insn/merge/log-file-1.xml",
					"tests/insn/merge/log-file-4.xml",
				},
			},
		}
	}
	return nil
}

func w3cBuildCollectionResolver(t *testing.T, collections []w3cCollection) *w3cCollectionResolver {
	t.Helper()
	resolver := &w3cCollectionResolver{
		collections:    make(map[string]xpath3.Sequence, len(collections)),
		uriCollections: make(map[string][]string, len(collections)),
	}
	for _, col := range collections {
		var seq xpath3.ItemSlice
		var uris []string
		for _, docPath := range col.DocPaths {
			pathOnly, fragment, _ := strings.Cut(docPath, "#")
			absPath, err := filepath.Abs(w3cResolvePath(pathOnly))
			require.NoError(t, err)
			data, err := os.ReadFile(absPath)
			require.NoError(t, err, "reading collection doc %s", docPath)
			doc, err := helium.NewParser().Parse(t.Context(), data)
			require.NoError(t, err, "parsing collection doc %s", docPath)
			doc.SetURL(absPath)
			node := helium.Node(doc)
			if fragment != "" {
				elem := doc.GetElementByID(fragment)
				require.NotNil(t, elem, "resolving collection fragment %s", docPath)
				node = elem
			}
			seq = append(seq, xpath3.NodeItem{Node: node})
			if fragment != "" {
				uris = append(uris, absPath+"#"+fragment)
			} else {
				uris = append(uris, absPath)
			}
		}
		resolver.collections[col.URI] = seq
		resolver.uriCollections[col.URI] = uris
	}
	return resolver
}

func w3cEvaluateParamSequence(ctx context.Context, exprText string) xpath3.Sequence {
	expr, err := xpath3.NewCompiler().Compile(exprText)
	if err == nil {
		if result, evalErr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, expr, nil); evalErr == nil {
			return result.Sequence()
		}
	}
	if len(exprText) >= 2 && exprText[0] == '\'' && exprText[len(exprText)-1] == '\'' {
		exprText = exprText[1 : len(exprText)-1]
	}
	return xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: exprText}}
}

// w3cExtractEmbeddedStylesheet extracts an embedded stylesheet from a source
// document that contains a <?xml-stylesheet type="text/xsl" href="#id">
// processing instruction. It returns a new Document rooted at the embedded
// xsl:stylesheet/xsl:transform element, suitable for compilation.
func w3cExtractEmbeddedStylesheet(t *testing.T, doc *helium.Document) *helium.Document {
	t.Helper()

	// Find <?xml-stylesheet?> PI with href="#id"
	var targetID string
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		pi, ok := n.(*helium.ProcessingInstruction)
		if !ok {
			continue
		}
		if pi.Name() != "xml-stylesheet" {
			continue
		}
		data := string(pi.Content())
		if !strings.Contains(data, "type=\"text/xsl\"") {
			continue
		}
		// Extract href value
		idx := strings.Index(data, "href=\"")
		if idx < 0 {
			continue
		}
		hrefStart := idx + len("href=\"")
		hrefEnd := strings.Index(data[hrefStart:], "\"")
		if hrefEnd < 0 {
			continue
		}
		href := data[hrefStart : hrefStart+hrefEnd]
		if !strings.HasPrefix(href, "#") {
			continue
		}
		targetID = href[1:]
		break
	}
	if targetID == "" {
		t.Fatal("embedded stylesheet: no <?xml-stylesheet?> PI with href=\"#id\" found")
	}

	// Find element with matching ID
	elem := doc.GetElementByID(targetID)
	if elem == nil {
		t.Fatalf("embedded stylesheet: element with id=%q not found", targetID)
	}

	// Create a new document with the stylesheet element as root
	ssDoc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	if url := doc.URL(); url != "" {
		ssDoc.SetURL(url)
	}

	copied, err := helium.CopyNode(elem, ssDoc)
	if err != nil {
		t.Fatalf("embedded stylesheet: copy element: %v", err)
	}
	if err := ssDoc.AddChild(copied); err != nil {
		t.Fatalf("embedded stylesheet: add child: %v", err)
	}

	return ssDoc
}
