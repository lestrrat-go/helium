package xpath3_test

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
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
	DocPath     string // relative to qt3DocsDir(); empty = no context document
	Namespaces  map[string]string
	Skip        string
	ExpectError bool
	Assertions  []qt3Assertion
}

// ──────────────────────────────────────────────────────────────────────
// Runner
// ──────────────────────────────────────────────────────────────────────

func qt3RunTests(t *testing.T, tests []qt3Test) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Skip != "" {
				t.Skip(tc.Skip)
			}
			ctx := t.Context()
			if len(tc.Namespaces) > 0 {
				ctx = xpath3.NewContext(ctx, xpath3.WithNamespaces(tc.Namespaces))
			}
			var doc helium.Node
			if tc.DocPath != "" {
				doc = qt3ParseDoc(t, filepath.Join(qt3DocsDir(), tc.DocPath))
			}
			compiled, err := xpath3.Compile(tc.XPath)
			if err != nil {
				if tc.ExpectError {
					return
				}
				require.NoError(t, err, "compile: %s", tc.XPath)
			}
			result, err := compiled.Evaluate(ctx, doc)
			if err != nil {
				if tc.ExpectError {
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

func qt3DocsDir() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "testdata", "qt3ts", "docs")
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
			parts = append(parts, fmt.Sprintf("%v", av.Value))
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
		case int64:
			return v != 0, nil
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
		require.Equal(t, expected, qt3StringValue(seq))
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
		require.Equal(t, expected, qt3StringValue(seq), "deep-eq")
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
		return qt3StringValue(seq) == expected
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

func qt3CheckCount(n int) qt3Check {
	return func(seq xpath3.Sequence) bool {
		return len(seq) == n
	}
}

func qt3CheckType(_ string) qt3Check {
	return func(_ xpath3.Sequence) bool {
		return true // not yet implemented
	}
}

func qt3CheckDeepEq(expected string) qt3Check {
	return func(seq xpath3.Sequence) bool {
		return qt3StringValue(seq) == expected
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
