package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// TestHOFMaterializationLimit verifies that higher-order / map / array built-ins
// that accumulate per-item callback results enforce the configured
// sequence/node-set size limit instead of materializing unbounded output. The
// domain (input range / map) stays within maxNodes; only the accumulated output
// overflows it, proving the accumulation sites are bounded independently of the
// range guard.
func TestHOFMaterializationLimit(t *testing.T) {
	t.Parallel()

	const limit = 1000

	overLimit := []struct {
		name string
		expr string
	}{
		// for-each: 600 inputs, each callback yields 2 items -> 1200 > 1000.
		{"for-each", `for-each(1 to 600, function($x) { ($x, $x) })`},
		// for-each-pair: 600 pairs, each callback yields 2 items -> 1200 > 1000.
		{"for-each-pair", `for-each-pair(1 to 600, 1 to 600, function($a, $b) { ($a, $b) })`},
		// map:for-each over a 600-entry map, each callback yields 2 items.
		{"map-for-each", `map:for-each(map:merge(for-each(1 to 600, function($x) { map:entry($x, $x) })), function($k, $v) { ($k, $v) })`},
		// array:join concatenating array members past the limit.
		{"array-join", `array:join(for-each(1 to 1100, function($x) { [$x] }))`},
		// array:flat-map producing 2 members per input member.
		{"array-flat-map", `array:flat-map(array { 1 to 600 }, function($x) { [$x, $x] })`},
		// array:flatten over a wide array whose flattened size exceeds the limit.
		{"array-flatten", `array:flatten(array { 1 to 1100 })`},
		// fn:filter accumulating every (true) item past the limit.
		{"filter", `filter(1 to 1100, function($x) { true() })`},
		// map:find collecting matching values past the limit.
		{"map-find", `map:find(for-each(1 to 1100, function($x) { map:entry("k", $x) }), "k")`},
		// fold-left: accumulator grows by 2 items per step -> overflows the limit.
		{"fold-left", `fold-left(1 to 600, (), function($acc, $x) { ($acc, $x, $x) })`},
		// fold-right: same, accumulator grows past the limit.
		{"fold-right", `fold-right(1 to 600, (), function($x, $acc) { ($x, $x, $acc) })`},
		// array:fold-left: accumulator grows by 2 items per member.
		{"array-fold-left", `array:fold-left(array { 1 to 600 }, (), function($acc, $x) { ($acc, $x, $x) })`},
		// array:fold-right: same from the right.
		{"array-fold-right", `array:fold-right(array { 1 to 600 }, (), function($x, $acc) { ($x, $x, $acc) })`},
		// array:filter accumulating every (true) member past the limit.
		{"array-filter", `array:filter(array { 1 to 1100 }, function($x) { true() })`},
		// array:for-each producing one member per input member past the limit.
		{"array-for-each", `array:for-each(array { 1 to 1100 }, function($x) { $x })`},
		// array:for-each-pair producing one member per pair past the limit.
		{"array-for-each-pair", `array:for-each-pair(array { 1 to 1100 }, array { 1 to 1100 }, function($a, $b) { ($a, $b) })`},
	}
	for _, tc := range overLimit {
		t.Run("over/"+tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				MaxNodesForTesting(limit).
				Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
		})
	}

	withinLimit := []struct {
		name string
		expr string
		want int
	}{
		{"for-each", `for-each(1 to 10, function($x) { ($x, $x) })`, 20},
		{"for-each-pair", `for-each-pair(1 to 10, 1 to 10, function($a, $b) { ($a, $b) })`, 20},
		{"map-for-each", `count(map:for-each(map { 1: 1, 2: 2, 3: 3 }, function($k, $v) { ($k, $v) }))`, 1},
		{"array-flatten", `array:flatten(array { 1, [2, [3, 4]], 5 })`, 5},
		{"filter", `filter(1 to 10, function($x) { $x mod 2 = 0 })`, 5},
		{"fold-left", `fold-left(1 to 10, 0, function($acc, $x) { $acc + $x })`, 1},
		{"fold-right", `fold-right(1 to 10, 0, function($x, $acc) { $x + $acc })`, 1},
		{"array-fold-left", `array:fold-left(array { 1 to 10 }, 0, function($acc, $x) { $acc + $x })`, 1},
		{"array-fold-right", `array:fold-right(array { 1 to 10 }, 0, function($x, $acc) { $x + $acc })`, 1},
		{"array-filter", `array:flatten(array:filter(array { 1 to 10 }, function($x) { $x mod 2 = 0 }))`, 5},
		{"array-for-each", `array:flatten(array:for-each(array { 1 to 10 }, function($x) { $x * 2 }))`, 10},
		{"array-for-each-pair", `array:flatten(array:for-each-pair(array { 1 to 5 }, array { 1 to 5 }, function($a, $b) { $a + $b }))`, 5},
		{"map-find", `array:flatten(map:find(map { "k": 1, "m": map { "k": 2 } }, "k"))`, 2},
	}
	for _, tc := range withinLimit {
		t.Run("within/"+tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			res, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				MaxNodesForTesting(limit).
				Evaluate(t.Context(), compiled, nil)
			require.NoError(t, err)
			require.Equal(t, tc.want, res.Sequence().Len())
		})
	}
}

// TestHOFLazySequenceLimit proves that the higher-order / map / array built-ins
// reject an oversized LAZY callback result, map value, or array member with
// ErrNodeSetLimit WITHOUT first materializing it. The lazy source is a 1<<40
// (≈1.1 trillion) integer range supplied via a variable: materializing it would
// attempt a ~9 TB allocation (each Item is multiple words), so any path that
// still does "materialize, then check" would OOM or hang here rather than return
// promptly. A correct, streaming/precheck implementation stops after at most
// `limit` items.
//
// The range is handed in as a borrowed variable (not written as `1 to N`) so it
// bypasses the range-expression construction guard AND the variable-clone path
// (which would otherwise materialize it), reaching the accumulation sites as a
// genuine unbounded lazy Sequence.
func TestHOFLazySequenceLimit(t *testing.T) {
	t.Parallel()

	const limit = 1000
	// A lazy range far larger than anything that could be materialized in memory.
	const huge = int64(1) << 40

	vars := xpath3.NewVariables()
	vars.Set("lazy", xpath3.NewRangeSequence(1, huge))

	// Only the callback-result / accumulator accumulation sites can receive a
	// genuinely lazy Sequence: maps and arrays are eager value types whose
	// members/values are cloned (materialized) at construction, so a lazy member
	// or map value cannot be constructed through the expression path. These cases
	// drive a lazy Sequence directly into the per-item accumulators.
	cases := []struct {
		name string
		expr string
	}{
		// Callback returns the huge lazy sequence on the very first invocation;
		// the accumulator (appendBoundedSeq) must stop before materializing it.
		{"for-each", `for-each(1 to 3, function($x) { $lazy })`},
		{"for-each-pair", `for-each-pair(1 to 3, 1 to 3, function($a, $b) { $lazy })`},
		{"map-for-each", `map:for-each(map { 1: 1 }, function($k, $v) { $lazy })`},
		// fold accumulator becomes the huge lazy sequence; the seqLen check (O(1)
		// on a lazy range) must reject it without materializing.
		{"fold-left", `fold-left(1 to 3, (), function($acc, $x) { $lazy })`},
		{"fold-right", `fold-right(1 to 3, (), function($x, $acc) { $lazy })`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			var evalErr error
			// NotPanics guards against a regression that materializes the range
			// (which would either OOM-panic or take effectively forever).
			require.NotPanics(t, func() {
				// EvalBorrowing keeps the lazy range out of the variable-clone
				// path so it stays unmaterialized until an accumulation site
				// consumes it.
				_, evalErr = xpath3.NewEvaluator(xpath3.EvalBorrowing).
					Variables(vars).
					MaxNodesForTesting(limit).
					Evaluate(t.Context(), compiled, nil)
			})
			require.ErrorIs(t, evalErr, xpath3.ErrNodeSetLimit)
		})
	}
}

// TestArrayFlattenDeepNesting verifies that array:flatten on a deeply nested
// array structure is iterative/bounded and does not blow the goroutine stack.
// The nested array flattens to a single item, so the node-set limit is not the
// guard here; the op-counter is, ensuring deep recursion-shaped input is
// rejected with ErrOpLimit rather than exhausting the stack.
func TestArrayFlattenDeepNesting(t *testing.T) {
	t.Parallel()

	// Build a deeply nested array: [[[...[1]...]]] of the given depth.
	const depth = 200000
	var nested xpath3.Item = xpath3.NewArray([]xpath3.Sequence{xpath3.SingleInteger(1)})
	for range depth {
		nested = xpath3.NewArray([]xpath3.Sequence{xpath3.ItemSlice{nested}})
	}

	compiled, err := xpath3.NewCompiler().Compile(`array:flatten(.)`)
	require.NoError(t, err)

	// With a low op-limit, the deep traversal must abort with ErrOpLimit and
	// must not panic (which a recursive implementation would risk via stack
	// exhaustion).
	t.Run("op-limit", func(t *testing.T) {
		t.Parallel()
		var evalErr error
		require.NotPanics(t, func() {
			_, evalErr = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				OpLimit(1000).
				ContextItem(nested).
				Evaluate(t.Context(), compiled, nil)
		})
		require.ErrorIs(t, evalErr, xpath3.ErrOpLimit)
	})

	// Without an op-limit it must still complete (iteratively) without a stack
	// blowup, flattening to the single innermost item.
	t.Run("completes", func(t *testing.T) {
		t.Parallel()
		var res *xpath3.Result
		var evalErr error
		require.NotPanics(t, func() {
			res, evalErr = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				ContextItem(nested).
				Evaluate(t.Context(), compiled, nil)
		})
		require.NoError(t, evalErr)
		require.Equal(t, 1, res.Sequence().Len())
	})
}

// TestMapFindDeepNesting verifies that map:find on a deeply nested map structure
// is iterative/bounded and does not blow the goroutine stack. The nesting yields
// a single match, so the op-counter (not the node-set limit) is the guard,
// ensuring deep recursion-shaped input is rejected with ErrOpLimit rather than
// exhausting the stack.
func TestMapFindDeepNesting(t *testing.T) {
	t.Parallel()

	// Build a deeply nested map: { "next": { "next": { ... { "k": 1 } } } }.
	const depth = 200000
	key := func(s string) xpath3.AtomicValue {
		return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}
	}
	var nested xpath3.Item = xpath3.NewMap([]xpath3.MapEntry{{Key: key("k"), Value: xpath3.SingleInteger(1)}})
	for range depth {
		nested = xpath3.NewMap([]xpath3.MapEntry{{Key: key("next"), Value: xpath3.ItemSlice{nested}}})
	}

	compiled, err := xpath3.NewCompiler().Compile(`map:find(., "k")`)
	require.NoError(t, err)

	// With a low op-limit, the deep traversal must abort with ErrOpLimit and must
	// not panic (which a recursive implementation would risk via stack overflow).
	t.Run("op-limit", func(t *testing.T) {
		t.Parallel()
		var evalErr error
		require.NotPanics(t, func() {
			_, evalErr = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				OpLimit(1000).
				ContextItem(nested).
				Evaluate(t.Context(), compiled, nil)
		})
		require.ErrorIs(t, evalErr, xpath3.ErrOpLimit)
	})

	// Without an op-limit it must still complete (iteratively) without a stack
	// blowup, finding the single innermost match.
	t.Run("completes", func(t *testing.T) {
		t.Parallel()
		var res *xpath3.Result
		var evalErr error
		require.NotPanics(t, func() {
			res, evalErr = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				ContextItem(nested).
				Evaluate(t.Context(), compiled, nil)
		})
		require.NoError(t, evalErr)
		// The result is a single array holding the one matched value.
		require.Equal(t, 1, res.Sequence().Len())
	})
}
