package xpath3_test

import (
	"iter"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// panicOnMaterializeSeq is a Sequence of n items where realizing the WHOLE
// sequence at once via Materialize panics. Len and Get are O(1), and Items
// yields lazily one item at a time, so any streaming consumer — whether it
// walks by index (fold-right) or via the lazy iterator (fold-left) — applies
// its per-item op-count / size-bound checks and stops early without ever
// triggering the panic. A "materialize up front, then check" consumer panics.
// It lets a test prove a built-in consumes a lazy/borrowed input WITHOUT fully
// realizing it.
type panicOnMaterializeSeq struct {
	n int
}

func (s panicOnMaterializeSeq) Len() int { return s.n }

func (s panicOnMaterializeSeq) Get(i int) xpath3.Item {
	return xpath3.SingleInteger(int64(i + 1)).Get(0)
}

func (s panicOnMaterializeSeq) Items() iter.Seq[xpath3.Item] {
	return func(yield func(xpath3.Item) bool) {
		for i := range s.n {
			if !yield(s.Get(i)) {
				return
			}
		}
	}
}

func (s panicOnMaterializeSeq) Materialize() []xpath3.Item {
	panic("Materialize called: sequence was fully materialized")
}

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

	// These cases drive a lazy Sequence directly into the per-item callback-result
	// / accumulator accumulation sites. (Lazy MAP values are covered separately by
	// TestMapFindNeverMaterializesValue: map:entry / map:merge borrow values
	// without cloning, so a lazy value reaches map:find intact.)
	cases := []struct {
		name string
		expr string
	}{
		// Callback returns the huge lazy sequence on the very first invocation;
		// the accumulator (appendBoundedSeq) must stop before materializing it.
		{"for-each", `for-each(1 to 3, function($x) { $lazy })`},
		{"for-each-pair", `for-each-pair(1 to 3, 1 to 3, function($a, $b) { $lazy })`},
		{"map-for-each", `map:for-each(map { 1: 1 }, function($k, $v) { $lazy })`},
		// array:for-each / array:for-each-pair cap the NUMBER of result members,
		// but NewArray clones each callback result; a single oversized lazy result
		// must be rejected by the per-result seqLen bound before it is cloned.
		{"array-for-each", `array:for-each(array { 1, 2, 3 }, function($x) { $lazy })`},
		{"array-for-each-pair", `array:for-each-pair(array { 1, 2, 3 }, array { 1, 2, 3 }, function($a, $b) { $lazy })`},
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

// TestAppendBoundedSeqHonorsOpLimit proves that appendBoundedSeq — the
// accumulation helper used by fn:for-each, fn:for-each-pair, and map:for-each —
// charges an op per appended item, so draining a large lazy callback result
// respects OpLimit (and, by extension, context cancellation) instead of running
// to completion unbounded. Each callback returns the huge lazy range on its
// first invocation; with NO node-set limit but a low op-limit the drain must
// stop with ErrOpLimit rather than materialize the trillion-item range.
func TestAppendBoundedSeqHonorsOpLimit(t *testing.T) {
	t.Parallel()

	const huge = int64(1) << 40

	vars := xpath3.NewVariables()
	vars.Set("lazy", xpath3.NewRangeSequence(1, huge))

	cases := []struct {
		name string
		expr string
	}{
		{"for-each", `for-each(1 to 3, function($x) { $lazy })`},
		{"for-each-pair", `for-each-pair(1 to 3, 1 to 3, function($a, $b) { $lazy })`},
		{"map-for-each", `map:for-each(map { 1: 1 }, function($k, $v) { $lazy })`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)

			var evalErr error
			// NotPanics guards against a regression that drains the lazy range
			// without an op check (which would hang / OOM rather than abort).
			require.NotPanics(t, func() {
				_, evalErr = xpath3.NewEvaluator(xpath3.EvalBorrowing).
					Variables(vars).
					OpLimit(1000).
					Evaluate(t.Context(), compiled, nil)
			})
			require.ErrorIs(t, evalErr, xpath3.ErrOpLimit)
		})
	}
}

// TestMapFindNeverMaterializesValue proves map:find applies its size bound to a
// matched map VALUE before cloning/materializing it. A map value stored via
// map:entry / map:merge is NOT cloned at construction (the single-entry and
// builder paths borrow the value), so a borrowed lazy value reaches map:find
// intact. map:find must reject an oversized matched value with ErrNodeSetLimit
// using an O(1) length check (seqLen) on the borrowed value, never the clone
// path that would materialize it.
//
// Two sources prove this: a panicOnMaterializeSeq (Materialize panics, so any
// "clone first, check later" regression panics) and a 1<<40 lazy integer range
// (materializing it would attempt a multi-TB allocation).
func TestMapFindNeverMaterializesValue(t *testing.T) {
	t.Parallel()

	const limit = 1000
	const inputLen = 1 << 20
	const huge = int64(1) << 40

	cases := []struct {
		name  string
		value xpath3.Sequence
		expr  string
	}{
		// Materialize panics; a clone-before-check regression panics instead of
		// returning a limit error.
		{"panic-on-materialize", panicOnMaterializeSeq{n: inputLen}, `map:find(map:entry("k", $lazy), "k")`},
		// A huge lazy range; materializing it would OOM/hang, so a correct O(1)
		// seqLen precheck must reject it promptly.
		{"huge-lazy-range", xpath3.NewRangeSequence(1, huge), `map:find(map:entry("k", $lazy), "k")`},
		// Same, reached through map:merge (also a non-cloning builder path).
		{"huge-via-merge", xpath3.NewRangeSequence(1, huge), `map:find(map:merge(map:entry("k", $lazy)), "k")`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			vars := xpath3.NewVariables()
			vars.Set("lazy", tc.value)

			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)

			var evalErr error
			require.NotPanics(t, func() {
				_, evalErr = xpath3.NewEvaluator(xpath3.EvalBorrowing).
					Variables(vars).
					MaxNodesForTesting(limit).
					Evaluate(t.Context(), compiled, nil)
			})
			require.ErrorIs(t, evalErr, xpath3.ErrNodeSetLimit)
		})
	}
}

// TestFoldNeverMaterializesInput proves fold-left and fold-right consume their
// (lazy/borrowed) input sequence by streaming — index-by-index — and apply the
// per-item op-count and accumulator size-bound BEFORE the input is ever fully
// materialized. The input is a panicOnMaterializeSeq: Get/Len are safe but
// Materialize/Items panic. A regression that does `seqMaterialize(seq)` up
// front (the original fold-right) would panic here instead of returning a limit
// error.
func TestFoldNeverMaterializesInput(t *testing.T) {
	t.Parallel()

	const limit = 1000
	// Far more items than the limit, so the accumulator bound trips well before
	// the input is exhausted (and long before any whole-sequence realization).
	const inputLen = 1 << 20

	cases := []struct {
		name string
		expr string
	}{
		// Accumulator grows by 1 per step; after `limit` steps it exceeds the
		// node-set limit and must be rejected with ErrNodeSetLimit.
		{"fold-left", `fold-left($huge, (), function($acc, $x) { ($acc, $x) })`},
		{"fold-right", `fold-right($huge, (), function($x, $acc) { ($x, $acc) })`},
	}
	for _, tc := range cases {
		t.Run("node-limit/"+tc.name, func(t *testing.T) {
			t.Parallel()
			vars := xpath3.NewVariables()
			vars.Set("huge", panicOnMaterializeSeq{n: inputLen})

			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)

			var evalErr error
			// NotPanics guards against a regression that materializes the input
			// up front (which panicOnMaterializeSeq turns into a panic).
			require.NotPanics(t, func() {
				_, evalErr = xpath3.NewEvaluator(xpath3.EvalBorrowing).
					Variables(vars).
					MaxNodesForTesting(limit).
					Evaluate(t.Context(), compiled, nil)
			})
			require.ErrorIs(t, evalErr, xpath3.ErrNodeSetLimit)
		})

		t.Run("op-limit/"+tc.name, func(t *testing.T) {
			t.Parallel()
			vars := xpath3.NewVariables()
			vars.Set("huge", panicOnMaterializeSeq{n: inputLen})

			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)

			var evalErr error
			require.NotPanics(t, func() {
				_, evalErr = xpath3.NewEvaluator(xpath3.EvalBorrowing).
					Variables(vars).
					OpLimit(1000).
					Evaluate(t.Context(), compiled, nil)
			})
			require.ErrorIs(t, evalErr, xpath3.ErrOpLimit)
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
