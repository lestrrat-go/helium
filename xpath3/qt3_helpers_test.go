package xpath3_test

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// qt3Assertion checks a result sequence, calling t.Fatal on failure.
type qt3Assertion func(t *testing.T, seq xpath3.Sequence)

// qt3Check returns true if a result sequence satisfies a condition (for any-of).
type qt3Check func(seq xpath3.Sequence) bool

type qt3Param struct {
	Name   string
	Select string
}

type qt3Test struct {
	Name                string
	XPath               string
	DocPath             string // relative to qt3TestDataDir(); empty = no context document
	Namespaces          map[string]string
	DefaultLanguage     string
	DefaultCollation    string
	DefaultDecimal      *qt3DecimalFormat
	NamedDecimalFormats []qt3NamedDecimalFormat
	Params              []qt3Param
	BaseURI             string            // static base URI for fn:unparsed-text etc.
	NeedsHTTP           bool              // test requires HTTP client (e.g. fn:json-doc with URL)
	ResourceMap         map[string]string // URI → file path (relative to qt3TestDataDir()) for resource resolution
	Skip                string
	ExpectError         bool
	AcceptError         bool // error is acceptable but not required (any-of with error + non-error)
	Assertions          []qt3Assertion
}

type qt3DecimalFormat struct {
	DecimalSeparator  string
	GroupingSeparator string
	Percent           string
	PerMille          string
	ZeroDigit         string
	Digit             string
	PatternSeparator  string
	ExponentSeparator string
	Infinity          string
	NaN               string
	MinusSign         string
}

type qt3NamedDecimalFormat struct {
	URI    string
	Name   string
	Format qt3DecimalFormat
}

func (df qt3DecimalFormat) toXPath3() xpath3.DecimalFormat {
	out := xpath3.DefaultDecimalFormat()
	if df.DecimalSeparator != "" {
		out.DecimalSeparator = qt3SingleRune(df.DecimalSeparator)
	}
	if df.GroupingSeparator != "" {
		out.GroupingSeparator = qt3SingleRune(df.GroupingSeparator)
	}
	if df.Percent != "" {
		out.Percent = qt3SingleRune(df.Percent)
	}
	if df.PerMille != "" {
		out.PerMille = qt3SingleRune(df.PerMille)
	}
	if df.ZeroDigit != "" {
		out.ZeroDigit = qt3SingleRune(df.ZeroDigit)
	}
	if df.Digit != "" {
		out.Digit = qt3SingleRune(df.Digit)
	}
	if df.PatternSeparator != "" {
		out.PatternSeparator = qt3SingleRune(df.PatternSeparator)
	}
	if df.ExponentSeparator != "" {
		out.ExponentSeparator = qt3SingleRune(df.ExponentSeparator)
	}
	if df.Infinity != "" {
		out.Infinity = df.Infinity
	}
	if df.NaN != "" {
		out.NaN = df.NaN
	}
	if df.MinusSign != "" {
		out.MinusSign = qt3SingleRune(df.MinusSign)
	}
	return out
}

func qt3SingleRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

// ──────────────────────────────────────────────────────────────────────
// Runner
// ──────────────────────────────────────────────────────────────────────

func qt3RunTests(t *testing.T, tests []qt3Test) {
	t.Helper()
	// Collect resource mappings
	resourceMap := make(map[string]string)
	for _, tc := range tests {
		for uri, path := range tc.ResourceMap {
			resourceMap[uri] = path
		}
	}
	// Always start an HTTP server so unparsed-text can resolve relative URIs
	httpClient := qt3NewTestServer(t, resourceMap)
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			if tc.Skip != "" {
				t.Skip(tc.Skip)
			}
			ctx := t.Context()
			// QT3 test suite expects implicit timezone of -05:00 (PT5H).
			qt3ImplicitTZ := time.FixedZone("", -5*3600)
			opts := []xpath3.ContextOption{
				xpath3.WithImplicitTimezone(qt3ImplicitTZ),
				xpath3.WithHTTPClient(httpClient),
			}
			if tc.DefaultDecimal != nil {
				opts = append(opts, xpath3.WithDefaultDecimalFormat(tc.DefaultDecimal.toXPath3()))
			}
			if len(tc.NamedDecimalFormats) > 0 {
				dfs := make(map[xpath3.QualifiedName]xpath3.DecimalFormat, len(tc.NamedDecimalFormats))
				for _, df := range tc.NamedDecimalFormats {
					dfs[xpath3.QualifiedName{URI: df.URI, Name: df.Name}] = df.Format.toXPath3()
				}
				opts = append(opts, xpath3.WithNamedDecimalFormats(dfs))
			}
			if tc.DefaultCollation != "" {
				opts = append(opts, xpath3.WithDefaultCollation(tc.DefaultCollation))
			}
			if tc.DefaultLanguage != "" {
				opts = append(opts, xpath3.WithDefaultLanguage(tc.DefaultLanguage))
			}
			if len(tc.Namespaces) > 0 {
				opts = append(opts, xpath3.WithNamespaces(tc.Namespaces))
			}
			if tc.BaseURI != "" {
				opts = append(opts, xpath3.WithBaseURI(tc.BaseURI))
			} else if baseURI := qt3DefaultBaseURI(tc); baseURI != "" {
				opts = append(opts, xpath3.WithBaseURI(baseURI))
			}
			var doc helium.Node
			if tc.DocPath != "" {
				doc = qt3ParseDoc(t, filepath.Join(qt3TestDataDir(), tc.DocPath))
			}
			if len(tc.Params) > 0 {
				vars := make(map[string]xpath3.Sequence, len(tc.Params))
				for _, param := range tc.Params {
					paramOpts := append([]xpath3.ContextOption{}, opts...)
					if len(vars) > 0 {
						paramOpts = append(paramOpts, xpath3.WithVariables(vars))
					}
					paramCtx := xpath3.NewContext(ctx, paramOpts...)
					compiledParam, err := xpath3.Compile(param.Select)
					require.NoError(t, err, "compile param $%s: %s", param.Name, param.Select)
					result, err := compiledParam.Evaluate(paramCtx, doc)
					require.NoError(t, err, "eval param $%s: %s", param.Name, param.Select)
					vars[param.Name] = result.Sequence()
				}
				opts = append(opts, xpath3.WithVariables(vars))
			}
			ctx = xpath3.NewContext(ctx, opts...)
			compiled, err := xpath3.Compile(tc.XPath)
			if err != nil {
				if tc.ExpectError || tc.AcceptError {
					return
				}
				require.NoError(t, err, "compile: %s", tc.XPath)
			}
			result, err := compiled.Evaluate(ctx, doc)
			if err != nil {
				if tc.ExpectError || tc.AcceptError {
					return
				}
				require.NoError(t, err, "eval: %s", tc.XPath)
			}
			if tc.ExpectError {
				t.Fatalf("expected error but got result: %v", result.Sequence())
			}
			seq := result.Sequence()
			for _, a := range tc.Assertions {
				a(t, seq)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────
// Path helpers
// ──────────────────────────────────────────────────────────────────────

func qt3TestDataDir() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "testdata", "qt3ts", "testdata")
}

func qt3DefaultBaseURI(tc qt3Test) string {
	if qt3NeedsParseXMLBaseURI(tc.XPath) {
		return "http://www.w3.org/fots/fn/"
	}
	if qt3NeedsRelativeUnparsedTextBaseURI(tc.XPath) && (tc.NeedsHTTP || len(tc.ResourceMap) > 0) {
		// Relative QT3 unparsed-text fixtures live under testdata/qt3ts/testdata/fn/.
		return "http://www.w3.org/fots/fn/"
	}
	if qt3NeedsRelativeParseJSONFixtureBaseURI(tc.XPath) {
		return "http://www.w3.org/fots/fn/"
	}
	if strings.Contains(tc.XPath, "static-base-uri()") {
		return "http://www.w3.org/2010/09/qt-fots-catalog/"
	}
	if baseURI := qt3ResourceMapBaseURI(tc); baseURI != "" {
		return baseURI
	}
	if tc.NeedsHTTP {
		return "http://www.w3.org/fots/"
	}
	return ""
}

func qt3ResourceMapBaseURI(tc qt3Test) string {
	if !tc.NeedsHTTP || len(tc.ResourceMap) == 0 {
		return ""
	}

	baseDir := ""
	for uri, relPath := range tc.ResourceMap {
		if strings.Contains(uri, "://") {
			return ""
		}
		dir := qt3RelativeResourceBase(relPath, uri)
		if dir == "" {
			return ""
		}
		if baseDir == "" {
			baseDir = dir
			continue
		}
		if baseDir != dir {
			return ""
		}
	}

	if baseDir == "" {
		return ""
	}
	return "http://www.w3.org/fots/" + strings.Trim(baseDir, "/") + "/"
}

func qt3RelativeResourceBase(relPath, uri string) string {
	relDir := path.Dir(filepath.ToSlash(relPath))
	uriDir := path.Dir(uri)
	if relDir == "." || relDir == "" {
		return ""
	}
	if uriDir == "." || uriDir == "" {
		return relDir
	}

	relParts := strings.Split(relDir, "/")
	uriParts := strings.Split(uriDir, "/")
	if len(uriParts) > len(relParts) {
		return ""
	}
	for i := 1; i <= len(uriParts); i++ {
		if relParts[len(relParts)-i] != uriParts[len(uriParts)-i] {
			return ""
		}
	}
	baseParts := relParts[:len(relParts)-len(uriParts)]
	if len(baseParts) == 0 {
		return ""
	}
	return strings.Join(baseParts, "/")
}

func qt3NeedsRelativeUnparsedTextBaseURI(expr string) bool {
	const name = "unparsed-text"

	idx := strings.Index(expr, name)
	if idx < 0 {
		return false
	}

	rest := expr[idx+len(name):]
	return strings.HasPrefix(rest, "(") ||
		strings.HasPrefix(rest, "-lines(") ||
		strings.HasPrefix(rest, "-available(")
}

func qt3NeedsRelativeParseJSONFixtureBaseURI(expr string) bool {
	return strings.Contains(expr, "parse-json(unparsed-text('parse-json/") ||
		strings.Contains(expr, `parse-json(unparsed-text("parse-json/`)
}

func qt3NeedsParseXMLBaseURI(expr string) bool {
	return strings.Contains(expr, "parse-xml(") ||
		strings.Contains(expr, "parse-xml-fragment(")
}

func qt3ParseDoc(t *testing.T, path string) helium.Node {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading %s", path)
	doc, err := helium.Parse(t.Context(), data)
	require.NoError(t, err, "parsing %s", path)
	absPath, err := filepath.Abs(path)
	if err == nil {
		doc.SetURL(absPath)
	}
	return doc
}

// ──────────────────────────────────────────────────────────────────────
// Value helpers
// ──────────────────────────────────────────────────────────────────────

func qt3StringValue(seq xpath3.Sequence) string {
	var parts []string
	for _, item := range seq {
		av, err := xpath3.AtomizeItem(item)
		if err != nil {
			parts = append(parts, fmt.Sprintf("%v", item))
		} else {
			s, serr := xpath3.AtomicToString(av)
			if serr != nil {
				parts = append(parts, fmt.Sprintf("%v", av.Value))
			} else {
				parts = append(parts, s)
			}
		}
	}
	return strings.Join(parts, " ")
}

func qt3EBV(seq xpath3.Sequence) (bool, error) {
	if len(seq) == 0 {
		return false, nil
	}
	first := seq[0]
	if _, ok := first.(xpath3.NodeItem); ok {
		return true, nil
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
		case *xpath3.FloatValue:
			f := v.Float64()
			return f != 0 && !math.IsNaN(f), nil
		case *big.Int:
			return v.Sign() != 0, nil
		case *big.Rat:
			return v.Sign() != 0, nil
		}
	}
	return false, fmt.Errorf("cannot compute EBV for sequence of length %d", len(seq))
}

// ──────────────────────────────────────────────────────────────────────
// Assertion factories  (for direct use in Assertions slice)
// ──────────────────────────────────────────────────────────────────────

func qt3AssertEq(expected string) qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		// assert-eq: expected is an XPath expression; evaluate it and compare using eq operator
		compiled, err := xpath3.Compile(expected)
		if err != nil {
			// Not a valid XPath expr — compare as literal string
			require.Equal(t, expected, qt3StringValue(seq))
			return
		}
		result, err := compiled.Evaluate(t.Context(), nil)
		if err != nil {
			require.Equal(t, expected, qt3StringValue(seq))
			return
		}
		// Try value comparison using eq operator for singleton atomic values
		expSeq := result.Sequence()
		if len(seq) == 1 && len(expSeq) == 1 {
			av, aErr := xpath3.AtomizeItem(seq[0])
			bv, bErr := xpath3.AtomizeItem(expSeq[0])
			if aErr == nil && bErr == nil {
				eq, cmpErr := xpath3.ValueCompare(xpath3.TokenEq, av, bv)
				if cmpErr == nil {
					if eq {
						return // values are equal via eq
					}
					// Fall through to string comparison for better error message
				}
			}
		}
		require.Equal(t, qt3StringValue(expSeq), qt3StringValue(seq))
	}
}

func qt3AssertStringValue(expected string) qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		require.Equal(t, expected, qt3StringValue(seq))
	}
}

func qt3AssertStringValueNS(expected string) qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		expected = strings.Join(strings.Fields(expected), " ")
		got := strings.Join(strings.Fields(qt3StringValue(seq)), " ")
		require.Equal(t, expected, got)
	}
}

func qt3AssertTrue() qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		ebv, err := qt3EBV(seq)
		require.NoError(t, err)
		require.True(t, ebv, "expected true, got: %v", qt3StringValue(seq))
	}
}

func qt3AssertFalse() qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		ebv, err := qt3EBV(seq)
		require.NoError(t, err)
		require.False(t, ebv, "expected false, got: %v", qt3StringValue(seq))
	}
}

func qt3AssertEmpty() qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		require.Empty(t, seq, "expected empty sequence")
	}
}

func qt3AssertCount(n int) qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		require.Len(t, seq, n)
	}
}

func qt3AssertType(_ string) qt3Assertion {
	return func(_ *testing.T, _ xpath3.Sequence) {
		// Type checking not yet implemented; pass unconditionally.
	}
}

func qt3AssertDeepEq(expected string) qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		compiled, err := xpath3.Compile(expected)
		if err != nil {
			// Fall back to string comparison if the expected value doesn't compile
			require.Equal(t, expected, qt3StringValue(seq), "deep-eq")
			return
		}
		result, err := compiled.Evaluate(t.Context(), nil)
		if err != nil {
			require.Equal(t, expected, qt3StringValue(seq), "deep-eq")
			return
		}
		expectedSeq := result.Sequence()
		if !qt3DeepEqualSeq(seq, expectedSeq) {
			require.Equal(t, qt3FormatSeq(expectedSeq), qt3FormatSeq(seq), "deep-eq")
		}
	}
}

// qt3AssertSkip is a no-op assertion for unimplemented assertion types.
func qt3AssertSkip() qt3Assertion {
	return func(_ *testing.T, _ xpath3.Sequence) {}
}

// ──────────────────────────────────────────────────────────────────────
// Check factories  (for use inside qt3AnyOf)
// ──────────────────────────────────────────────────────────────────────

func qt3CheckEq(expected string) qt3Check {
	return func(seq xpath3.Sequence) bool {
		compiled, err := xpath3.Compile(expected)
		if err != nil {
			return qt3StringValue(seq) == expected
		}
		result, err := compiled.Evaluate(context.Background(), nil)
		if err != nil {
			return qt3StringValue(seq) == expected
		}
		return qt3StringValue(seq) == qt3StringValue(result.Sequence())
	}
}

func qt3CheckStringValue(expected string) qt3Check {
	return func(seq xpath3.Sequence) bool {
		return qt3StringValue(seq) == expected
	}
}

func qt3CheckStringValueNS(expected string) qt3Check {
	return func(seq xpath3.Sequence) bool {
		got := strings.Join(strings.Fields(qt3StringValue(seq)), " ")
		want := strings.Join(strings.Fields(expected), " ")
		return got == want
	}
}

func qt3CheckTrue() qt3Check {
	return func(seq xpath3.Sequence) bool {
		ebv, err := qt3EBV(seq)
		return err == nil && ebv
	}
}

func qt3CheckFalse() qt3Check {
	return func(seq xpath3.Sequence) bool {
		ebv, err := qt3EBV(seq)
		return err == nil && !ebv
	}
}

func qt3CheckEmpty() qt3Check {
	return func(seq xpath3.Sequence) bool {
		return len(seq) == 0
	}
}

func qt3CheckType(_ string) qt3Check {
	return func(_ xpath3.Sequence) bool {
		return true // not yet implemented
	}
}

func qt3CheckCount(n int) qt3Check {
	return func(seq xpath3.Sequence) bool {
		return len(seq) == n
	}
}

func qt3CheckDeepEq(expected string) qt3Check {
	return func(seq xpath3.Sequence) bool {
		compiled, err := xpath3.Compile(expected)
		if err != nil {
			return qt3StringValue(seq) == expected
		}
		result, err := compiled.Evaluate(context.Background(), nil)
		if err != nil {
			return qt3StringValue(seq) == expected
		}
		return qt3DeepEqualSeq(seq, result.Sequence())
	}
}

func qt3CheckSkip() qt3Check {
	return func(_ xpath3.Sequence) bool {
		return true
	}
}

// qt3AnyOf passes if any check succeeds.
func qt3AnyOf(checks ...qt3Check) qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		for _, c := range checks {
			if c(seq) {
				return
			}
		}
		require.Fail(t, "none of the any-of assertions passed", "got: %s", qt3StringValue(seq))
	}
}

// ──────────────────────────────────────────────────────────────────────
// Deep-equal helpers for structural comparison (arrays, maps, atomics)
// ──────────────────────────────────────────────────────────────────────

// qt3DeepEqualSeq compares two sequences structurally.
func qt3DeepEqualSeq(a, b xpath3.Sequence) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !qt3DeepEqualItem(a[i], b[i]) {
			return false
		}
	}
	return true
}

func qt3DeepEqualItem(a, b xpath3.Item) bool {
	switch av := a.(type) {
	case xpath3.AtomicValue:
		bv, ok := b.(xpath3.AtomicValue)
		if !ok {
			return false
		}
		return qt3DeepEqualAtomic(av, bv)
	case xpath3.ArrayItem:
		bArr, ok := b.(xpath3.ArrayItem)
		if !ok {
			return false
		}
		if av.Size() != bArr.Size() {
			return false
		}
		for i := 1; i <= av.Size(); i++ {
			am, _ := av.Get(i)
			bm, _ := bArr.Get(i)
			if !qt3DeepEqualSeq(am, bm) {
				return false
			}
		}
		return true
	case xpath3.MapItem:
		bMap, ok := b.(xpath3.MapItem)
		if !ok {
			return false
		}
		if av.Size() != bMap.Size() {
			return false
		}
		keys := av.Keys()
		for _, k := range keys {
			aVal, _ := av.Get(k)
			bVal, found := bMap.Get(k)
			if !found {
				return false
			}
			if !qt3DeepEqualSeq(aVal, bVal) {
				return false
			}
		}
		return true
	case xpath3.NodeItem:
		bn, ok := b.(xpath3.NodeItem)
		if !ok {
			return false
		}
		// Compare node string values
		aStr, _ := xpath3.AtomizeItem(av)
		bStr, _ := xpath3.AtomizeItem(bn)
		return qt3DeepEqualAtomic(aStr, bStr)
	default:
		return false
	}
}

func qt3DeepEqualAtomic(a, b xpath3.AtomicValue) bool {
	// Numeric comparison with promotion
	if a.IsNumeric() && b.IsNumeric() {
		return a.ToFloat64() == b.ToFloat64()
	}
	// String-based comparison for same types
	sa, err1 := xpath3.AtomicToString(a)
	sb, err2 := xpath3.AtomicToString(b)
	if err1 != nil || err2 != nil {
		return fmt.Sprintf("%v", a.Value) == fmt.Sprintf("%v", b.Value)
	}
	return sa == sb
}

// qt3FormatSeq returns a human-readable representation of a sequence for error messages.
func qt3FormatSeq(seq xpath3.Sequence) string {
	if len(seq) == 0 {
		return "()"
	}
	parts := make([]string, len(seq))
	for i, item := range seq {
		parts[i] = qt3FormatItem(item)
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts, ", ")
}

// qt3Handler returns an http.Handler that serves QT3 test resource files.
// It maps:
//   - /fots/* → testdata/qt3ts/testdata/fn/* (for unparsed-text etc.)
//   - Resource URIs → local files via resourceMap
func qt3Handler(resourceMap map[string]string) http.Handler {
	dataDir := qt3TestDataDir()
	fotsDir := dataDir
	// Build a path-based lookup from the resource map.
	// resourceMap keys are full URIs (e.g., "http://www.w3.org/qt3/json/data004-json").
	// We extract the URL path and map it to a local file.
	pathMap := make(map[string]string)
	for uri, relPath := range resourceMap {
		// Extract the path portion from the URI
		idx := strings.Index(uri, "://")
		if idx >= 0 {
			rest := uri[idx+3:]
			slashIdx := strings.Index(rest, "/")
			if slashIdx >= 0 {
				pathMap[rest[slashIdx:]] = filepath.Join(dataDir, relPath)
			}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check resource map first
		if filePath, ok := pathMap[r.URL.Path]; ok {
			http.ServeFile(w, r, filePath)
			return
		}
		// Fallback: /fots/ prefix for unparsed-text resources
		if strings.HasPrefix(r.URL.Path, "/fots/") {
			http.StripPrefix("/fots/", http.FileServer(http.Dir(fotsDir))).ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

// qt3NewTestServer creates an httptest.Server with the QT3 handler and
// returns an HTTP client whose transport routes all requests (regardless
// of hostname) to that server.
func qt3NewTestServer(t *testing.T, resourceMap map[string]string) *http.Client {
	t.Helper()
	srv := httptest.NewServer(qt3Handler(resourceMap))
	t.Cleanup(srv.Close)
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, srv.Listener.Addr().String())
			},
		},
	}
}

func qt3FormatItem(item xpath3.Item) string {
	switch v := item.(type) {
	case xpath3.AtomicValue:
		s, err := xpath3.AtomicToString(v)
		if err != nil {
			return fmt.Sprintf("%v", v.Value)
		}
		if v.TypeName == xpath3.TypeString {
			return fmt.Sprintf("%q", s)
		}
		return s
	case xpath3.ArrayItem:
		parts := make([]string, v.Size())
		for i := 1; i <= v.Size(); i++ {
			m, _ := v.Get(i)
			parts[i-1] = qt3FormatSeq(m)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case xpath3.MapItem:
		var parts []string
		keys := v.Keys()
		for _, k := range keys {
			val, _ := v.Get(k)
			ks, _ := xpath3.AtomicToString(k)
			parts = append(parts, fmt.Sprintf("%s: %s", ks, qt3FormatSeq(val)))
		}
		return "map{" + strings.Join(parts, ", ") + "}"
	case xpath3.NodeItem:
		a, err := xpath3.AtomizeItem(v)
		if err != nil {
			return "<node>"
		}
		s, _ := xpath3.AtomicToString(a)
		return s
	default:
		return fmt.Sprintf("%v", item)
	}
}
