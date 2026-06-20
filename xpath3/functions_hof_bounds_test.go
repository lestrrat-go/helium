package xpath3_test

import (
	"iter"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// Shared subtest names reused across the bound tests (kept as constants to
// satisfy goconst).
const (
	tcArrayForEach     = "array-for-each"
	tcArrayForEachPair = "array-for-each-pair"
	tcArrayJoin        = "array-join"
	tcArrayFlatMap     = "array-flat-map"
	tcArrayFlatten     = "array-flatten"
	tcMapFind          = "map-find"
	tcMapForEach       = "map-for-each"
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

// varsSet returns a *xpath3.Variables binding a single name to val. Used to hand
// an oversized input to a built-in via a variable (under EvalBorrowing) so the
// input itself bypasses construction/range guards and only the function under
// test's own bound can fire.
func varsSet(name string, val xpath3.Sequence) *xpath3.Variables {
	v := xpath3.NewVariables()
	v.Set(name, val)
	return v
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

	// Cases that must hand an oversized array/map/sequence to the function under
	// test bind it via a variable so the construction does NOT trip the `1 to N`
	// range guard or a producing function's bound — only the TARGET function's
	// accumulation can overflow the limit, so each case truly exercises its named
	// bound.
	//
	// wideSeq builds a plain sequence of n single-integer items (1..n).
	wideSeq := func(n int) xpath3.Sequence {
		items := make([]xpath3.Item, n)
		for i := range items {
			items[i] = xpath3.SingleInteger(int64(i + 1)).Get(0)
		}
		return xpath3.ItemSlice(items)
	}
	// wideArray builds an array of n single-integer members (1..n).
	wideArray := func(n int) xpath3.Sequence {
		members := make([]xpath3.Sequence, n)
		for i := range members {
			members[i] = xpath3.SingleInteger(int64(i + 1))
		}
		return xpath3.ItemSlice{xpath3.NewArray(members)}
	}
	// wideMaps builds a sequence of n single-entry maps, each {"k": i}. map:find
	// collects one value per map, so n maps yield n matched values.
	wideMaps := func(n int) xpath3.Sequence {
		key := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "k"}
		items := make([]xpath3.Item, n)
		for i := range items {
			items[i] = xpath3.NewMap([]xpath3.MapEntry{{Key: key, Value: xpath3.SingleInteger(int64(i + 1))}})
		}
		return xpath3.ItemSlice(items)
	}

	overLimit := []struct {
		name string
		expr string
		vars *xpath3.Variables
	}{
		// for-each: 600 inputs, each callback yields 2 items -> 1200 > 1000.
		{name: "for-each", expr: `for-each(1 to 600, function($x) { ($x, $x) })`},
		// for-each-pair: 600 pairs, each callback yields 2 items -> 1200 > 1000.
		{name: "for-each-pair", expr: `for-each-pair(1 to 600, 1 to 600, function($a, $b) { ($a, $b) })`},
		// map:for-each over a 600-entry map, each callback yields 2 items.
		{name: "map-for-each", expr: `map:for-each(map:merge(for-each(1 to 600, function($x) { map:entry($x, $x) })), function($k, $v) { ($k, $v) })`},
		// array:join concatenating two 600-member arrays -> 1200 members > 1000.
		// Inputs stay within the limit; only array:join's accumulated member list
		// overflows it.
		{name: tcArrayJoin, expr: `array:join((array { 1 to 600 }, array { 1 to 600 }))`},
		// array:flat-map producing 2 members per input member.
		{name: tcArrayFlatMap, expr: `array:flat-map(array { 1 to 600 }, function($x) { [$x, $x] })`},
		// array:flatten over a wide (variable-bound) array whose flattened size
		// exceeds the limit. The array is bound as a variable so its construction
		// does not trip the `1 to N` range guard before array:flatten runs.
		{name: tcArrayFlatten, expr: `array:flatten($arr)`, vars: varsSet("arr", wideArray(1100))},
		// fn:filter accumulating every (true) item of a variable-bound wide
		// sequence past the limit. Binding via a variable keeps the input out of
		// the `1 to N` range guard so only filter's accumulation overflows.
		{name: "filter", expr: `filter($seq, function($x) { true() })`, vars: varsSet("seq", wideSeq(1100))},
		// map:find collecting one matching value per map past the limit. The
		// maps are bound as a variable so map:find's accumulation, not the input
		// construction, is what overflows the limit.
		{name: tcMapFind, expr: `map:find($maps, "k")`, vars: varsSet("maps", wideMaps(1100))},
		// fold-left: accumulator grows by 2 items per step -> overflows the limit.
		{name: "fold-left", expr: `fold-left(1 to 600, (), function($acc, $x) { ($acc, $x, $x) })`},
		// fold-right: same, accumulator grows past the limit.
		{name: "fold-right", expr: `fold-right(1 to 600, (), function($x, $acc) { ($x, $x, $acc) })`},
		// array:fold-left: accumulator grows by 2 items per member.
		{name: "array-fold-left", expr: `array:fold-left(array { 1 to 600 }, (), function($acc, $x) { ($acc, $x, $x) })`},
		// array:fold-right: same from the right.
		{name: "array-fold-right", expr: `array:fold-right(array { 1 to 600 }, (), function($x, $acc) { ($x, $x, $acc) })`},
		// array:filter accumulating every (true) member of a variable-bound wide
		// array past the limit. Binding via a variable keeps the input out of the
		// `1 to N` range guard so only array:filter's accumulation overflows.
		{name: "array-filter", expr: `array:filter($arr, function($x) { true() })`, vars: varsSet("arr", wideArray(1100))},
		// array:for-each producing one member per member of a variable-bound wide
		// array past the limit.
		{name: tcArrayForEach, expr: `array:for-each($arr, function($x) { $x })`, vars: varsSet("arr", wideArray(1100))},
		// array:for-each-pair producing one member per pair of two variable-bound
		// wide arrays past the limit.
		{name: tcArrayForEachPair, expr: `array:for-each-pair($arr, $arr, function($a, $b) { ($a, $b) })`, vars: varsSet("arr", wideArray(1100))},
	}
	for _, tc := range overLimit {
		t.Run("over/"+tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			// EvalBorrowing keeps a variable-bound array/map out of the
			// variable-clone path so the input itself never trips a length bound;
			// only the function under test's accumulation can overflow.
			eval := xpath3.NewEvaluator(xpath3.EvalBorrowing).
				MaxNodesForTesting(limit)
			if tc.vars != nil {
				eval = eval.Variables(tc.vars)
			}
			_, err = eval.Evaluate(t.Context(), compiled, nil)
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
		{tcArrayFlatten, `array:flatten(array { 1, [2, [3, 4]], 5 })`, 5},
		{"filter", `filter(1 to 10, function($x) { $x mod 2 = 0 })`, 5},
		{"fold-left", `fold-left(1 to 10, 0, function($acc, $x) { $acc + $x })`, 1},
		{"fold-right", `fold-right(1 to 10, 0, function($x, $acc) { $x + $acc })`, 1},
		{"array-fold-left", `array:fold-left(array { 1 to 10 }, 0, function($acc, $x) { $acc + $x })`, 1},
		{"array-fold-right", `array:fold-right(array { 1 to 10 }, 0, function($x, $acc) { $x + $acc })`, 1},
		{"array-filter", `array:flatten(array:filter(array { 1 to 10 }, function($x) { $x mod 2 = 0 }))`, 5},
		{tcArrayForEach, `array:flatten(array:for-each(array { 1 to 10 }, function($x) { $x * 2 }))`, 10},
		{tcArrayForEachPair, `array:flatten(array:for-each-pair(array { 1 to 5 }, array { 1 to 5 }, function($a, $b) { $a + $b }))`, 5},
		{tcMapFind, `array:flatten(map:find(map { "k": 1, "m": map { "k": 2 } }, "k"))`, 2},
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
		{tcArrayForEach, `array:for-each(array { 1, 2, 3 }, function($x) { $lazy })`},
		{tcArrayForEachPair, `array:for-each-pair(array { 1, 2, 3 }, array { 1, 2, 3 }, function($a, $b) { $lazy })`},
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

// TestBulkCloneSitesHonorOpLimit proves that the built-ins which clone or
// materialize a whole sub-sequence in one shot — array:for-each,
// array:for-each-pair, array:join, array:flat-map, and map:find — charge the
// sub-sequence length against the op-counter BEFORE the bulk clone/append. A
// callback result / array member list / matched value that is below maxNodes but
// whose length exceeds OpLimit must be rejected with ErrOpLimit rather than
// silently cloned. With NO node-set limit (so only the op-counter can fire) and
// a low op-limit, each case's single oversized sub-sequence must trip ErrOpLimit.
func TestBulkCloneSitesHonorOpLimit(t *testing.T) {
	t.Parallel()

	const opLimit = 1000
	// A sub-sequence longer than opLimit but small enough to materialize cheaply.
	const wide = 5000

	wideSeq := func(n int) xpath3.Sequence {
		items := make([]xpath3.Item, n)
		for i := range items {
			items[i] = xpath3.SingleInteger(int64(i + 1)).Get(0)
		}
		return xpath3.ItemSlice(items)
	}
	wideArrayVal := func(n int) xpath3.Sequence {
		members := make([]xpath3.Sequence, n)
		for i := range members {
			members[i] = xpath3.SingleInteger(int64(i + 1))
		}
		return xpath3.ItemSlice{xpath3.NewArray(members)}
	}

	cases := []struct {
		name string
		expr string
		vars *xpath3.Variables
	}{
		// array:for-each callback returns one oversized sequence; NewArray would
		// clone it. The op-charge on the result length must fire first.
		{
			name: tcArrayForEach,
			expr: `array:for-each(array { 1 }, function($x) { $wide })`,
			vars: varsSet("wide", wideSeq(wide)),
		},
		// array:for-each-pair, same: a single oversized callback result.
		{
			name: tcArrayForEachPair,
			expr: `array:for-each-pair(array { 1 }, array { 1 }, function($a, $b) { $wide })`,
			vars: varsSet("wide", wideSeq(wide)),
		},
		// array:join bulk-appends one array's wide member list.
		{
			name: tcArrayJoin,
			expr: `array:join($arr)`,
			vars: varsSet("arr", wideArrayVal(wide)),
		},
		// array:flat-map's callback returns a wide array whose members are
		// bulk-appended.
		{
			name: tcArrayFlatMap,
			expr: `array:flat-map(array { 1 }, function($x) { $arr })`,
			vars: varsSet("arr", wideArrayVal(wide)),
		},
		// map:find clones a matched value; the op-charge on the value length must
		// fire before cloneSequence. The value is a panicOnMaterializeSeq stored
		// via map:entry (which borrows, not clones), so a regression that clones
		// before charging ops calls Materialize and panics — proving the precharge
		// runs first. (A plain wide sequence cannot isolate this site because the
		// surrounding result construction also charges per-item ops.)
		{
			name: tcMapFind,
			expr: `map:find(map:entry("k", $panic), "k")`,
			vars: varsSet("panic", panicOnMaterializeSeq{n: wide}),
		},
		// map:for-each clones each entry value before invoking the callback; the
		// op-charge on the value length must fire before that clone. The value is
		// a panicOnMaterializeSeq stored via map:entry (borrowed, not cloned), and
		// the callback returns () so it never iterates $v itself — so ONLY
		// map:for-each's own per-value precharge can stop the run. A regression
		// that clones the borrowed value before charging ops calls Materialize and
		// panics instead of returning ErrOpLimit.
		{
			name: tcMapForEach,
			expr: `map:for-each(map:entry("k", $panic), function($k, $v) { () })`,
			vars: varsSet("panic", panicOnMaterializeSeq{n: wide}),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			var evalErr error
			// NotPanics guards the map-find / map-for-each cases against a
			// regression that clones the borrowed value before charging ops (which
			// calls Materialize and panics).
			require.NotPanics(t, func() {
				// EvalBorrowing keeps the value out of the variable-clone path, and
				// no MaxNodesForTesting is set, so only the op-counter can fire.
				_, evalErr = xpath3.NewEvaluator(xpath3.EvalBorrowing).
					Variables(tc.vars).
					OpLimit(opLimit).
					Evaluate(t.Context(), compiled, nil)
			})
			require.ErrorIs(t, evalErr, xpath3.ErrOpLimit)
		})
	}
}

// TestArrayMemberCountBound proves that the built-ins which build a result array
// with one MEMBER per callback result / matched value (array:for-each,
// array:for-each-pair, map:find) bound the MEMBER count independently of the
// item count. A callback that yields an empty sequence (or a map value that is
// empty) adds zero items but still adds a member, so without a member-count
// bound many empty results could build an array with more than maxNodes members
// while the item total stays at zero. Each case keeps every result empty and
// drives the member count past the limit; the result must be rejected with
// ErrNodeSetLimit.
func TestArrayMemberCountBound(t *testing.T) {
	t.Parallel()

	const limit = 1000

	// wideMapsEmpty builds a sequence of n single-entry maps {"k": ()}; map:find
	// matches "k" in each, collecting one EMPTY value per map -> n members, 0 items.
	wideMapsEmpty := func(n int) xpath3.Sequence {
		key := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "k"}
		items := make([]xpath3.Item, n)
		for i := range items {
			items[i] = xpath3.NewMap([]xpath3.MapEntry{{Key: key, Value: xpath3.ItemSlice{}}})
		}
		return xpath3.ItemSlice(items)
	}
	wideArray := func(n int) xpath3.Sequence {
		members := make([]xpath3.Sequence, n)
		for i := range members {
			members[i] = xpath3.SingleInteger(int64(i + 1))
		}
		return xpath3.ItemSlice{xpath3.NewArray(members)}
	}

	cases := []struct {
		name string
		expr string
		vars *xpath3.Variables
	}{
		// 1100 members, each callback returns () -> 1100 members, 0 items > 1000.
		{name: tcArrayForEach, expr: `array:for-each($arr, function($x) { () })`, vars: varsSet("arr", wideArray(1100))},
		// 1100 pairs, each callback returns () -> 1100 members, 0 items.
		{name: tcArrayForEachPair, expr: `array:for-each-pair($arr, $arr, function($a, $b) { () })`, vars: varsSet("arr", wideArray(1100))},
		// 1100 maps each with an empty value for "k" -> 1100 empty members.
		{name: tcMapFind, expr: `map:find($maps, "k")`, vars: varsSet("maps", wideMapsEmpty(1100))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
				Variables(tc.vars).
				MaxNodesForTesting(limit).
				Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
		})
	}
}

// TestFlatMapEmptyArrayOpLimit proves array:flat-map charges an op for every
// callback result ITEM — including items that are EMPTY arrays whose member
// expansion appends nothing. A regression that charged ops only per appended
// member (rather than once per result item) would let a callback returning many
// empty arrays run unbounded. The callback returns a variable-bound sequence of
// `wide` empty arrays (bound via EvalBorrowing so producing it costs no ops), so
// ONLY flat-map's per-result-item op-charge can stop the run; with no node-set
// limit and a low op-limit it must trip ErrOpLimit.
func TestFlatMapEmptyArrayOpLimit(t *testing.T) {
	t.Parallel()

	const opLimit = 1000
	const wide = 5000

	// A sequence of `wide` empty arrays. Each adds zero members on expansion, so a
	// regression that charged ops per appended member would never fire.
	emptyArrays := make([]xpath3.Item, wide)
	for i := range emptyArrays {
		emptyArrays[i] = xpath3.NewArray(nil)
	}
	vars := varsSet("empties", xpath3.ItemSlice(emptyArrays))

	compiled, err := xpath3.NewCompiler().Compile(`array:flat-map(array { 1 }, function($x) { $empties })`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(vars).
		OpLimit(opLimit).
		Evaluate(t.Context(), compiled, nil)
	require.ErrorIs(t, err, xpath3.ErrOpLimit)
}

// TestArrayMemberSeqLenOpLimit proves array:join and array:flat-map charge the
// ITEM length of each member sequence (seqLen), not just one op per member,
// before appending it. NewArray clones each member sequence, so a SINGLE member
// holding many items — below maxNodes but above opLimit — must be rejected with
// ErrOpLimit before being cloned. Each input array has exactly one member (a wide
// sequence of `wide` items), so a regression that charged only len(members) (= 1)
// would pass opLimit(1000); only the per-member seqLen charge (= wide) trips it.
func TestArrayMemberSeqLenOpLimit(t *testing.T) {
	t.Parallel()

	const opLimit = 1000

	cases := []struct {
		name string
		expr string
	}{
		// array:join over a single 1-member array whose member holds many items.
		// The op-charge on that member's seqLen must fire before NewArray clones it.
		{name: tcArrayJoin, expr: `array:join([ 1 to 5000 ])`},
		// array:flat-map's callback returns a 1-member array whose member holds
		// `wide` items; the per-member seqLen op-charge must fire first.
		{name: tcArrayFlatMap, expr: `array:flat-map(array { 1 }, function($x) { [ 1 to 5000 ] })`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				OpLimit(opLimit).
				Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrOpLimit)
		})
	}
}

// TestArrayMemberSeqLenNodeLimit proves array:join and array:flat-map bound the
// total ITEM count of the members they accumulate against maxNodes (not merely
// the member COUNT) before NewArray clones/materializes them. Each input array
// has a single member whose item count alone exceeds maxNodes, so a regression
// that checked only the member count (= 1 <= maxNodes) would clone the oversized
// member; only the per-member item-count bound trips ErrNodeSetLimit. The wide
// member is bound via a variable under EvalBorrowing so the input construction
// does not trip the `1 to N` range guard first.
func TestArrayMemberSeqLenNodeLimit(t *testing.T) {
	t.Parallel()

	const limit = 1000

	wideSeq := func(n int) xpath3.Sequence {
		items := make([]xpath3.Item, n)
		for i := range items {
			items[i] = xpath3.SingleInteger(int64(i + 1)).Get(0)
		}
		return xpath3.ItemSlice(items)
	}
	// wideMemberArray builds an array with ONE member: a wide sequence of n items.
	wideMemberArray := func(n int) xpath3.Sequence {
		return xpath3.ItemSlice{xpath3.NewArray([]xpath3.Sequence{wideSeq(n)})}
	}

	cases := []struct {
		name string
		expr string
		vars *xpath3.Variables
	}{
		// array:join over a single 1-member array whose member holds 1100 items.
		// The member-count check (1 <= 1000) passes; only the item-count bound fires.
		{name: tcArrayJoin, expr: `array:join($arr)`, vars: varsSet("arr", wideMemberArray(1100))},
		// array:flat-map's callback returns a 1-member array whose member holds
		// 1100 items; same — only the per-member item-count bound trips the limit.
		{name: tcArrayFlatMap, expr: `array:flat-map(array { 1 }, function($x) { $arr })`, vars: varsSet("arr", wideMemberArray(1100))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
				Variables(tc.vars).
				MaxNodesForTesting(limit).
				Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
		})
	}
}

// TestArrayFilterSelectedItemBound proves array:filter bounds the total ITEM
// count of the members it SELECTS against maxNodes (not merely the selected
// member COUNT) before NewArray clones/materializes them. The input array holds
// a single member whose item count alone exceeds maxNodes and the callback
// selects it, so a regression that checked only the selected member count
// (= 1 <= maxNodes) would clone the oversized member; only the per-member
// item-count bound trips ErrNodeSetLimit. The wide member is bound via a
// variable under EvalBorrowing so the input construction does not trip the
// `1 to N` range guard first.
func TestArrayFilterSelectedItemBound(t *testing.T) {
	t.Parallel()

	const limit = 1000

	wideSeq := func(n int) xpath3.Sequence {
		items := make([]xpath3.Item, n)
		for i := range items {
			items[i] = xpath3.SingleInteger(int64(i + 1)).Get(0)
		}
		return xpath3.ItemSlice(items)
	}
	// wideMemberArray builds an array with ONE member: a wide sequence of n items.
	vars := varsSet("arr", xpath3.ItemSlice{xpath3.NewArray([]xpath3.Sequence{wideSeq(1100)})})

	compiled, err := xpath3.NewCompiler().Compile(`array:filter($arr, function($x) { true() })`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(vars).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
}

// TestArrayJoinFlatMapEmptyMemberCountBound proves array:join and array:flat-map
// bound the result MEMBER count independently of the item count. A member that
// is an EMPTY sequence adds zero items but still adds a member, so without a
// member-count bound many empty members could build an array with more than
// maxNodes members while the item total stays at zero. Each case drives the
// member count past the limit with empty members and must be rejected with
// ErrNodeSetLimit.
func TestArrayJoinFlatMapEmptyMemberCountBound(t *testing.T) {
	t.Parallel()

	const limit = 1000

	// emptyMemberArray builds an array with n EMPTY members.
	emptyMemberArray := func(n int) xpath3.Sequence {
		members := make([]xpath3.Sequence, n)
		for i := range members {
			members[i] = xpath3.ItemSlice{}
		}
		return xpath3.ItemSlice{xpath3.NewArray(members)}
	}

	cases := []struct {
		name string
		expr string
		vars *xpath3.Variables
	}{
		// array:join over a single array with 1100 empty members -> 1100 members,
		// 0 items > 1000. Only the member-count bound can fire.
		{name: tcArrayJoin, expr: `array:join($arr)`, vars: varsSet("arr", emptyMemberArray(1100))},
		// array:flat-map's callback returns a 1100-empty-member array; same — only
		// the member-count bound trips the limit.
		{name: tcArrayFlatMap, expr: `array:flat-map(array { 1 }, function($x) { $arr })`, vars: varsSet("arr", emptyMemberArray(1100))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
				Variables(tc.vars).
				MaxNodesForTesting(limit).
				Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
		})
	}
}

// TestArrayJoinFlatMapEmptyMemberOpLimit proves array:join and array:flat-map
// charge at least one op per appended member even when the member is an EMPTY
// sequence (item length 0). A regression that charged only seqLen(member) would
// let many empty members run unbounded because fnCountOps(...,0) never trips
// OpLimit. With NO node-set limit (so only the op-counter can fire) and a low
// op-limit, the wide list of empty members must trip ErrOpLimit. The wide array
// is bound via a variable under EvalBorrowing so producing it costs no ops.
func TestArrayJoinFlatMapEmptyMemberOpLimit(t *testing.T) {
	t.Parallel()

	const opLimit = 1000
	const wide = 5000

	emptyMemberArray := func(n int) xpath3.Sequence {
		members := make([]xpath3.Sequence, n)
		for i := range members {
			members[i] = xpath3.ItemSlice{}
		}
		return xpath3.ItemSlice{xpath3.NewArray(members)}
	}

	cases := []struct {
		name string
		expr string
		vars *xpath3.Variables
	}{
		{name: tcArrayJoin, expr: `array:join($arr)`, vars: varsSet("arr", emptyMemberArray(wide))},
		{name: tcArrayFlatMap, expr: `array:flat-map(array { 1 }, function($x) { $arr })`, vars: varsSet("arr", emptyMemberArray(wide))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
				Variables(tc.vars).
				OpLimit(opLimit).
				Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrOpLimit)
		})
	}
}

// TestSeqCursorEmptyMemberOpLimit proves the seqCursor — the shared one-item-
// at-a-time walker behind array:flatten and map:find — charges an op for every
// EMPTY member/value sequence it steps past. An empty member yields no item, so
// it never reaches the consumers' per-item op charge; without a per-empty charge
// at the cursor level, array:flatten over thousands of empty array members and a
// non-matching map:find over thousands of empty map values would each scan for
// free and never trip OpLimit. Each case has NO node-set limit (so only the
// op-counter can fire) and a low op-limit, with all members/values empty so only
// the per-empty cursor charge can stop the run.
func TestSeqCursorEmptyMemberOpLimit(t *testing.T) {
	t.Parallel()

	const opLimit = 1000
	const wide = 5000

	// emptyMemberArray builds an array with n EMPTY members. array:flatten steps
	// the cursor past every one, yielding nothing per member.
	emptyMemberArray := func(n int) xpath3.Sequence {
		members := make([]xpath3.Sequence, n)
		for i := range members {
			members[i] = xpath3.ItemSlice{}
		}
		return xpath3.ItemSlice{xpath3.NewArray(members)}
	}
	// emptyValueMaps builds a sequence of n single-entry maps {"k": ()}. map:find
	// for a NON-matching key ("absent") collects nothing, so it only steps the
	// cursor past every empty value — each must cost an op.
	emptyValueMaps := func(n int) xpath3.Sequence {
		key := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "k"}
		items := make([]xpath3.Item, n)
		for i := range items {
			items[i] = xpath3.NewMap([]xpath3.MapEntry{{Key: key, Value: xpath3.ItemSlice{}}})
		}
		return xpath3.ItemSlice(items)
	}

	cases := []struct {
		name string
		expr string
		vars *xpath3.Variables
	}{
		// array:flatten over an array of `wide` empty members: each member adds zero
		// items, so only the per-empty cursor op-charge can trip OpLimit.
		{name: tcArrayFlatten, expr: `array:flatten($arr)`, vars: varsSet("arr", emptyMemberArray(wide))},
		// map:find for a key that matches NO entry over `wide` maps with empty
		// values: nothing is collected, so only the per-empty cursor op-charge fires.
		{name: tcMapFind, expr: `map:find($maps, "absent")`, vars: varsSet("maps", emptyValueMaps(wide))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
				Variables(tc.vars).
				OpLimit(opLimit).
				Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrOpLimit)
		})
	}
}

// TestFlatMapScalarMemberCountBound proves array:flat-map bounds the result
// MEMBER count at its NON-array (scalar) append site too. A callback result item
// that is NOT an array becomes one scalar member; without the member-count bound
// at that site, maxNodes EMPTY array members (each adding a member but 0 items)
// can fill the member count to the limit, then a single scalar (item count 1,
// which passes the item-count bound 1 <= maxNodes-0) pushes the member count to
// maxNodes+1. The callback yields one array with maxNodes empty members followed
// by one scalar item, so only the scalar-branch member-count bound can trip
// ErrNodeSetLimit.
func TestFlatMapScalarMemberCountBound(t *testing.T) {
	t.Parallel()

	const limit = 1000

	// One array with `limit` EMPTY members (each becomes a member, 0 items),
	// followed by a single scalar integer item. The empty members fill the result
	// member count to exactly the limit; the trailing scalar adds member limit+1
	// via the non-array branch.
	empties := make([]xpath3.Sequence, limit)
	for i := range empties {
		empties[i] = xpath3.ItemSlice{}
	}
	items := []xpath3.Item{xpath3.NewArray(empties), xpath3.SingleInteger(42).Get(0)}
	vars := varsSet("mix", xpath3.ItemSlice(items))

	compiled, err := xpath3.NewCompiler().Compile(`array:flat-map(array { 1 }, function($x) { $mix })`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(vars).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
}

// TestArrayFilterEmptyMemberCountBound proves array:filter bounds the SELECTED
// MEMBER count independently of the item count. A selected member that is an
// EMPTY sequence adds zero items but still adds a member, so without a
// member-count bound 1100 selected empty members would build an array with 1100
// members under a 1000-node limit while the item total stays at zero. The input
// array holds 1100 empty members and the callback selects every one, so only the
// member-count bound can trip ErrNodeSetLimit.
func TestArrayFilterEmptyMemberCountBound(t *testing.T) {
	t.Parallel()

	const limit = 1000

	members := make([]xpath3.Sequence, 1100)
	for i := range members {
		members[i] = xpath3.ItemSlice{}
	}
	vars := varsSet("arr", xpath3.ItemSlice{xpath3.NewArray(members)})

	compiled, err := xpath3.NewCompiler().Compile(`array:filter($arr, function($x) { true() })`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(vars).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
}

// TestMapFindSingleClone proves map:find clones each matched value EXACTLY ONCE.
// The function collects matched values into a slice that is then handed to
// NewArray, which clones every member defensively. A regression that also cloned
// the value when appending it would clone each matched value twice — doubling the
// per-value allocation and exceeding the single-clone op precharge. The result is
// correct and the clone work stays at one clone per matched value, observed as a
// stable allocation count: appending the uncloned value keeps allocs at the
// single-clone level; a double-clone roughly doubles the per-value allocations.
func TestMapFindSingleClone(t *testing.T) {
	// No t.Parallel(): testing.AllocsPerRun panics if called from a parallel test.

	const n = 200

	key := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "k"}
	items := make([]xpath3.Item, n)
	for i := range items {
		items[i] = xpath3.NewMap([]xpath3.MapEntry{{Key: key, Value: xpath3.SingleInteger(int64(i))}})
	}
	vars := varsSet("maps", xpath3.ItemSlice(items))

	compiled, err := xpath3.NewCompiler().Compile(`map:find($maps, "k")`)
	require.NoError(t, err)

	// Sanity: every value is collected once into the result array.
	res, err := xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(vars).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)
	require.Equal(t, 1, res.Sequence().Len())
	arr, ok := res.Sequence().Get(0).(xpath3.ArrayItem)
	require.True(t, ok)
	require.Equal(t, n, arr.Size())

	// A single clone of n matched values allocates close to one clone's worth per
	// value; a double-clone regression allocates roughly twice as much. The
	// threshold sits between the two regimes (measured: ~430 single vs ~830
	// double), so it locks in single-clone behavior without being brittle.
	allocs := testing.AllocsPerRun(50, func() {
		_, evalErr := xpath3.NewEvaluator(xpath3.EvalBorrowing).
			Variables(vars).
			Evaluate(t.Context(), compiled, nil)
		require.NoError(t, evalErr)
	})
	require.Less(t, allocs, float64(600),
		"map:find should clone each matched value once; a higher alloc count indicates a double clone")
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
