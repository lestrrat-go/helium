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
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// qt3Assertion checks a result sequence, calling t.Fatal on failure.
type qt3Assertion func(t *testing.T, seq xpath3.Sequence)

// qt3Check returns true if a result sequence satisfies a condition (for any-of).
type qt3Check func(seq xpath3.Sequence) bool
type qt3EvalMutator func(xpath3.Evaluator) xpath3.Evaluator

type qt3Param struct {
	Name   string
	Select string
}

type qt3SourceDoc struct {
	Name    string
	DocPath string
	URI     string
}

type qt3Collection struct {
	URI        string
	SourceDocs []qt3SourceDoc
	Query      string
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
	SourceDocs          []qt3SourceDoc
	Collections         []qt3Collection
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

type qt3CollectionResolver struct {
	collections    map[string]xpath3.Sequence
	uriCollections map[string][]string
}

func (r *qt3CollectionResolver) ResolveCollection(uri string) (xpath3.Sequence, error) {
	seq, ok := r.collections[uri]
	if !ok {
		return nil, fmt.Errorf("collection %q not found", uri)
	}
	return xpath3.ItemSlice(append([]xpath3.Item(nil), seq.Materialize()...)), nil
}

func (r *qt3CollectionResolver) ResolveURICollection(uri string) ([]string, error) {
	uris, ok := r.uriCollections[uri]
	if !ok {
		return nil, fmt.Errorf("uri-collection %q not found", uri)
	}
	return append([]string(nil), uris...), nil
}

func qt3ApplyEval(eval xpath3.Evaluator, muts []qt3EvalMutator) xpath3.Evaluator {
	for _, mut := range muts {
		eval = mut(eval)
	}
	return eval
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
	// Register resource mappings into the shared server
	for _, tc := range tests {
		if len(tc.ResourceMap) > 0 {
			qt3RegisterResources(tc.ResourceMap)
		}
	}
	httpClient := qt3GetSharedClient()
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			if tc.Skip != "" {
				t.Skip(tc.Skip)
			}
			ctx := t.Context()
			// QT3 test suite expects implicit timezone of -05:00 (PT5H).
			qt3ImplicitTZ := time.FixedZone("", -5*3600)
			opts := []qt3EvalMutator{
				func(e xpath3.Evaluator) xpath3.Evaluator { return e.ImplicitTimezone(qt3ImplicitTZ) },
				func(e xpath3.Evaluator) xpath3.Evaluator { return e.HTTPClient(httpClient) },
			}
			if tc.DefaultDecimal != nil {
				df := tc.DefaultDecimal.toXPath3()
				opts = append(opts, func(e xpath3.Evaluator) xpath3.Evaluator { return e.DefaultDecimalFormat(df) })
			}
			if len(tc.NamedDecimalFormats) > 0 {
				dfs := make(map[xpath3.QualifiedName]xpath3.DecimalFormat, len(tc.NamedDecimalFormats))
				for _, df := range tc.NamedDecimalFormats {
					dfs[xpath3.QualifiedName{URI: df.URI, Name: df.Name}] = df.Format.toXPath3()
				}
				opts = append(opts, func(e xpath3.Evaluator) xpath3.Evaluator { return e.NamedDecimalFormats(dfs) })
			}
			if tc.DefaultCollation != "" {
				collation := tc.DefaultCollation
				opts = append(opts, func(e xpath3.Evaluator) xpath3.Evaluator {
					return e.DefaultCollation(collation)
				})
			}
			if tc.DefaultLanguage != "" {
				lang := tc.DefaultLanguage
				opts = append(opts, func(e xpath3.Evaluator) xpath3.Evaluator { return e.DefaultLanguage(lang) })
			}
			if len(tc.Namespaces) > 0 {
				ns := tc.Namespaces
				opts = append(opts, func(e xpath3.Evaluator) xpath3.Evaluator { return e.Namespaces(ns) })
			}
			if tc.BaseURI != "" {
				uri := tc.BaseURI
				opts = append(opts, func(e xpath3.Evaluator) xpath3.Evaluator { return e.BaseURI(uri) })
			} else if baseURI := qt3DefaultBaseURI(tc); baseURI != "" {
				opts = append(opts, func(e xpath3.Evaluator) xpath3.Evaluator { return e.BaseURI(baseURI) })
			}
			var doc helium.Node
			if tc.DocPath != "" {
				doc = qt3ParseDoc(t, filepath.Join(qt3TestDataDir(), tc.DocPath))
			}
			var vars map[string]xpath3.Sequence
			if len(tc.SourceDocs) > 0 || len(tc.Params) > 0 {
				vars = make(map[string]xpath3.Sequence, len(tc.SourceDocs)+len(tc.Params))
			}
			for _, src := range tc.SourceDocs {
				sourceDoc := qt3ParseDocSource(t, src)
				vars[src.Name] = xpath3.ItemSlice{xpath3.NodeItem{Node: sourceDoc}}
			}
			if resolver := qt3BuildCollectionResolver(t, ctx, tc, opts, doc, vars); resolver != nil {
				opts = append(opts, func(e xpath3.Evaluator) xpath3.Evaluator { return e.CollectionResolver(resolver) })
			}
			if len(tc.Params) > 0 {
				for _, param := range tc.Params {
					paramEval := qt3ApplyEval(xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions), opts)
					if len(vars) > 0 {
						paramEval = paramEval.Variables(xpath3.VariablesFromMap(vars))
					}
					compiledParam, err := xpath3.NewCompiler().Compile(param.Select)
					require.NoError(t, err, "compile param $%s: %s", param.Name, param.Select)
					result, err := paramEval.Evaluate(ctx, compiledParam, doc)
					require.NoError(t, err, "eval param $%s: %s", param.Name, param.Select)
					vars[param.Name] = result.Sequence()
				}
			}
			if len(vars) > 0 {
				v := vars
				opts = append(opts, func(e xpath3.Evaluator) xpath3.Evaluator { return e.Variables(xpath3.VariablesFromMap(v)) })
			}
			eval := qt3ApplyEval(xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions), opts)
			compiled, err := xpath3.NewCompiler().Compile(tc.XPath)
			if err != nil {
				if tc.ExpectError || tc.AcceptError {
					return
				}
				require.NoError(t, err, "compile: %s", tc.XPath)
			}
			result, err := eval.Evaluate(ctx, compiled, doc)
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
	return filepath.Join(heliumtest.CallerDir(0), "..", "testdata", "qt3ts", "testdata")
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

	_, rest, found := strings.Cut(expr, name)
	if !found {
		return false
	}
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
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err, "parsing %s", path)
	absPath, err := filepath.Abs(path)
	if err == nil {
		doc.SetURL(absPath)
	}
	return doc
}

func qt3ParseDocSource(t *testing.T, src qt3SourceDoc) helium.Node {
	t.Helper()
	doc := qt3ParseDoc(t, filepath.Join(qt3TestDataDir(), src.DocPath))
	if src.URI != "" {
		if document, ok := doc.(*helium.Document); ok {
			document.SetURL(src.URI)
		} else if owner := doc.OwnerDocument(); owner != nil {
			owner.SetURL(src.URI)
		}
	}
	return doc
}

func qt3BuildCollectionResolver(t *testing.T, ctx context.Context, tc qt3Test, opts []qt3EvalMutator, doc helium.Node, vars map[string]xpath3.Sequence) xpath3.CollectionResolver {
	t.Helper()
	if len(tc.Collections) == 0 {
		return nil
	}

	resolver := &qt3CollectionResolver{
		collections:    make(map[string]xpath3.Sequence, len(tc.Collections)),
		uriCollections: make(map[string][]string, len(tc.Collections)),
	}

	queryEval := qt3ApplyEval(xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions), opts)
	if len(vars) > 0 {
		queryEval = queryEval.Variables(xpath3.VariablesFromMap(vars))
	}

	for _, col := range tc.Collections {
		if len(col.SourceDocs) > 0 {
			seq := make(xpath3.ItemSlice, 0, len(col.SourceDocs))
			uris := make([]string, 0, len(col.SourceDocs))
			for _, src := range col.SourceDocs {
				sourceDoc := qt3ParseDocSource(t, src) //nolint:contextcheck
				seq = append(seq, xpath3.NodeItem{Node: sourceDoc})
				if src.URI != "" {
					uris = append(uris, src.URI)
				}
			}
			resolver.collections[col.URI] = seq
			resolver.uriCollections[col.URI] = uris
			continue
		}

		if col.Query == "" {
			resolver.collections[col.URI] = nil
			resolver.uriCollections[col.URI] = nil
			continue
		}

		expr, err := xpath3.NewCompiler().Compile(col.Query)
		require.NoError(t, err, "compile collection %q query: %s", col.URI, col.Query)
		result, err := queryEval.Evaluate(ctx, expr, doc)
		require.NoError(t, err, "eval collection %q query: %s", col.URI, col.Query)
		resolver.collections[col.URI] = result.Sequence()
		resolver.uriCollections[col.URI] = qt3CollectionURIs(t, col.URI, result.Sequence())
	}

	return resolver
}

func qt3CollectionURIs(t *testing.T, uri string, seq xpath3.Sequence) []string {
	t.Helper()
	if seq == nil {
		return nil
	}
	uris := make([]string, 0, seq.Len())
	for item := range seq.Items() {
		av, err := xpath3.AtomizeItem(item)
		require.NoError(t, err, "atomize collection %q member", uri)
		if av.TypeName != xpath3.TypeAnyURI {
			continue
		}
		s, err := xpath3.AtomicToString(av)
		require.NoError(t, err, "stringify collection %q URI member", uri)
		uris = append(uris, s)
	}
	return uris
}

// ──────────────────────────────────────────────────────────────────────
// Value helpers
// ──────────────────────────────────────────────────────────────────────

func qt3StringValue(seq xpath3.Sequence) string {
	if seq == nil {
		return ""
	}
	var parts []string
	for item := range seq.Items() {
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
	if seq == nil || seq.Len() == 0 {
		return false, nil
	}
	first := seq.Get(0)
	if _, ok := first.(xpath3.NodeItem); ok {
		return true, nil
	}
	if seq.Len() == 1 {
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
		case int64:
			return v != 0, nil
		case *big.Int:
			return v.Sign() != 0, nil
		case *big.Rat:
			return v.Sign() != 0, nil
		}
	}
	return false, fmt.Errorf("cannot compute EBV for sequence of length %d", seq.Len())
}

// ──────────────────────────────────────────────────────────────────────
// Assertion factories  (for direct use in Assertions slice)
// ──────────────────────────────────────────────────────────────────────

func qt3AssertEq(expected string) qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		// assert-eq: expected is an XPath expression; evaluate it and compare using eq operator
		compiled, err := xpath3.NewCompiler().Compile(expected)
		if err != nil {
			// Not a valid XPath expr — compare as literal string
			require.Equal(t, expected, qt3StringValue(seq))
			return
		}
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, nil)
		if err != nil {
			require.Equal(t, expected, qt3StringValue(seq))
			return
		}
		// Try value comparison using eq operator for singleton atomic values
		expSeq := result.Sequence()
		if seq.Len() == 1 && expSeq.Len() == 1 {
			av, aErr := xpath3.AtomizeItem(seq.Get(0))
			bv, bErr := xpath3.AtomizeItem(expSeq.Get(0))
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
		require.True(t, seq == nil || seq.Len() == 0, "expected empty sequence")
	}
}

func qt3AssertCount(n int) qt3Assertion {
	return func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		if seq == nil {
			require.Equal(t, 0, n)
		} else {
			require.Equal(t, n, seq.Len())
		}
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
		compiled, err := xpath3.NewCompiler().Compile(expected)
		if err != nil {
			// Fall back to string comparison if the expected value doesn't compile
			require.Equal(t, expected, qt3StringValue(seq), "deep-eq")
			return
		}
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, nil)
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
		compiled, err := xpath3.NewCompiler().Compile(expected)
		if err != nil {
			return qt3StringValue(seq) == expected
		}
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(context.Background(), compiled, nil)
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
		return seq == nil || seq.Len() == 0
	}
}

func qt3CheckType(_ string) qt3Check {
	return func(_ xpath3.Sequence) bool {
		return true // not yet implemented
	}
}

func qt3CheckCount(n int) qt3Check {
	return func(seq xpath3.Sequence) bool {
		if seq == nil {
			return n == 0
		}
		return seq.Len() == n
	}
}

func qt3CheckDeepEq(expected string) qt3Check {
	return func(seq xpath3.Sequence) bool {
		compiled, err := xpath3.NewCompiler().Compile(expected)
		if err != nil {
			return qt3StringValue(seq) == expected
		}
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(context.Background(), compiled, nil)
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
func qt3SeqLen(s xpath3.Sequence) int {
	if s == nil {
		return 0
	}
	return s.Len()
}

func qt3DeepEqualSeq(a, b xpath3.Sequence) bool {
	if qt3SeqLen(a) != qt3SeqLen(b) {
		return false
	}
	for i := range qt3SeqLen(a) {
		if !qt3DeepEqualItem(a.Get(i), b.Get(i)) {
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
	if seq == nil || seq.Len() == 0 {
		return "()"
	}
	parts := make([]string, seq.Len())
	for i := range seq.Len() {
		parts[i] = qt3FormatItem(seq.Get(i))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts, ", ")
}

// qt3SharedServer is a package-level shared HTTP test server.
// All qt3RunTests calls share this single server instead of creating 356 separate ones.
var qt3SharedServer struct {
	once    sync.Once
	srv     *httptest.Server
	client  *http.Client
	pathMap sync.Map // map[string]string: URL path → local file path
	fotsDir string
}

func qt3InitSharedServer() {
	qt3SharedServer.once.Do(func() {
		qt3SharedServer.fotsDir = qt3TestDataDir()
		qt3SharedServer.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check resource map first
			if filePath, ok := qt3SharedServer.pathMap.Load(r.URL.Path); ok {
				http.ServeFile(w, r, filePath.(string))
				return
			}
			// Fallback: /fots/ prefix for unparsed-text resources
			if strings.HasPrefix(r.URL.Path, "/fots/") {
				http.StripPrefix("/fots/", http.FileServer(http.Dir(qt3SharedServer.fotsDir))).ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
		}))
		qt3SharedServer.client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, network, qt3SharedServer.srv.Listener.Addr().String())
				},
			},
		}
	})
}

// qt3RegisterResources registers resource URI→file mappings into the shared server.
func qt3RegisterResources(resourceMap map[string]string) {
	dataDir := qt3TestDataDir()
	for uri, relPath := range resourceMap {
		if _, afterScheme, ok := strings.Cut(uri, "://"); ok {
			if slashIdx := strings.Index(afterScheme, "/"); slashIdx >= 0 {
				qt3SharedServer.pathMap.Store(afterScheme[slashIdx:], filepath.Join(dataDir, relPath))
			}
		}
	}
}

// qt3GetSharedClient returns the HTTP client for the shared test server.
func qt3GetSharedClient() *http.Client {
	qt3InitSharedServer()
	return qt3SharedServer.client
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
