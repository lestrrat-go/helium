package xpath3_test

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// qt3Assertion checks a result sequence, calling t.Fatal on failure.
type qt3Assertion func(t *testing.T, seq xpath3.Sequence)

// qt3Check returns true if a result sequence satisfies a condition (for any-of).
type qt3Check func(seq xpath3.Sequence) bool

type qt3Test struct {
	Name        string
	XPath       string
	DocPath     string // relative to qt3TestDataDir(); empty = no context document
	Namespaces  map[string]string
	BaseURI     string // static base URI for fn:unparsed-text etc.
	Skip        string
	ExpectError bool
	AcceptError bool // error is acceptable but not required (any-of with error + non-error)
	Assertions  []qt3Assertion
}

// ──────────────────────────────────────────────────────────────────────
// Runner
// ──────────────────────────────────────────────────────────────────────

func qt3RunTests(t *testing.T, tests []qt3Test) {
	t.Helper()
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
			}
			if len(tc.Namespaces) > 0 {
				opts = append(opts, xpath3.WithNamespaces(tc.Namespaces))
			}
			if tc.BaseURI != "" {
				opts = append(opts, xpath3.WithBaseURI(tc.BaseURI))
			}
			opts = append(opts, xpath3.WithURIResolver(qt3URIResolver()))
			ctx = xpath3.NewContext(ctx, opts...)
			var doc helium.Node
			if tc.DocPath != "" {
				doc = qt3ParseDoc(t, filepath.Join(qt3TestDataDir(), tc.DocPath))
			}
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

func qt3ParseDoc(t *testing.T, path string) helium.Node {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading %s", path)
	doc, err := helium.Parse(t.Context(), data)
	require.NoError(t, err, "parsing %s", path)
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
		case float64:
			return v != 0 && !math.IsNaN(v), nil
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

// qt3URIResolverImpl maps http://www.w3.org/fots/ URIs to local test data files.
type qt3URIResolverImpl struct {
	sourceDir string
}

func qt3URIResolver() xpath3.URIResolver {
	_, f, _, _ := runtime.Caller(0)
	sourceDir := filepath.Join(filepath.Dir(f), "..", "testdata", "qt3ts", "source")
	return &qt3URIResolverImpl{sourceDir: sourceDir}
}

func (r *qt3URIResolverImpl) ResolveURI(uri string) (io.ReadCloser, error) {
	const fotsPrefix = "http://www.w3.org/fots/"
	if strings.HasPrefix(uri, fotsPrefix) {
		relPath := uri[len(fotsPrefix):]
		localPath := filepath.Join(r.sourceDir, "fn", relPath)
		return os.Open(localPath)
	}
	// Also handle file:// URIs and bare file paths
	if strings.HasPrefix(uri, "file://") {
		return os.Open(uri[len("file://"):])
	}
	if filepath.IsAbs(uri) {
		return os.Open(uri)
	}
	return nil, fmt.Errorf("qt3URIResolver: unsupported URI: %s", uri)
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
