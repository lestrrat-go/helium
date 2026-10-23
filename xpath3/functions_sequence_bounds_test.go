package xpath3_test

import (
	"context"
	"iter"
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestSequenceBuiltinsRespectOpLimit(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{
		"reverse(1 to 20000)",
		"sort(1 to 20000)",
		"insert-before(1 to 6000, 1, 1 to 6000)",
		"tail(1 to 20000)",
		"sum(1 to 20000)",
		"deep-equal(1 to 20000, 1 to 20000)",
	} {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).OpLimit(10).Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrOpLimit)
		})
	}
}

func TestSequenceBuiltinsRespectNodeLimit(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{
		"reverse(1 to 20000)",
		"sort(1 to 20000)",
		"insert-before(1 to 6000, 1, 1 to 6000)",
	} {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				MaxNodesForTesting(1000).
				Evaluate(t.Context(), compiled, nil)
			require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
		})
	}
}

func TestSequenceBuiltinsPollCancellationDuringWork(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{
		"reverse(1 to 20000)",
		"sort(1 to 20000)",
		"insert-before(1 to 6000, 1, 1 to 6000)",
		"tail(1 to 20000)",
		"sum(1 to 20000)",
		"deep-equal(1 to 20000, 1 to 20000)",
	} {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			ctx := heliumtest.NewPollContext(t.Context(), 10, context.Canceled)
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, nil)
			require.ErrorIs(t, err, context.Canceled)
			require.Less(t, ctx.Polls(), 100, "the operation must stop polling soon after cancellation")
		})
	}
}

type cancelOnReadSequence struct {
	n      int
	reads  int
	cancel context.CancelFunc
}

func (s *cancelOnReadSequence) Len() int { return s.n }

func (s *cancelOnReadSequence) Get(i int) xpath3.Item {
	s.reads++
	if s.reads == 1 {
		s.cancel()
	}
	return xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: int64(i)}
}

func (s *cancelOnReadSequence) Items() iter.Seq[xpath3.Item] {
	return func(yield func(xpath3.Item) bool) {
		for i := range s.n {
			if !yield(s.Get(i)) {
				return
			}
		}
	}
}

func (s *cancelOnReadSequence) Materialize() []xpath3.Item {
	items := make([]xpath3.Item, s.n)
	for i := range items {
		items[i] = s.Get(i)
	}
	return items
}

func TestSequenceBuiltinsStopReadingAfterCancellation(t *testing.T) {
	t.Parallel()
	for _, expr := range []string{"sum($items)", "tail($items)", "deep-equal($items, $items)"} {
		t.Run(expr, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			seq := &cancelOnReadSequence{n: 4096, cancel: cancel}
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
				Variables(varsSet("items", seq)).Evaluate(ctx, compiled, nil)
			require.ErrorIs(t, err, context.Canceled)
			require.LessOrEqual(t, seq.reads, 4, "stop consuming the input after cancellation")
		})
	}
}

func TestNestedSequenceBuiltinsPollCancellation(t *testing.T) {
	t.Parallel()
	const count = 256
	array := xpath3.NewArray(make([]xpath3.Sequence, count))
	entries := make([]xpath3.MapEntry, count)
	for i := range entries {
		entries[i].Key = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: strconv.Itoa(i)}
	}
	m := xpath3.NewMap(entries)
	doc, err := helium.NewParser().Parse(t.Context(), []byte("<r>"+strings.Repeat("<a/>", count)+"</r>"))
	require.NoError(t, err)
	for _, tc := range []struct {
		name string
		expr string
		item xpath3.Item
	}{
		{"sum empty array members", "sum($items)", array},
		{"deep-equal array", "deep-equal($items, $items)", array},
		{"deep-equal map", "deep-equal($items, $items)", m},
		{"deep-equal nodes", "deep-equal($items, $items)", xpath3.NodeItem{Node: doc}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			ctx := heliumtest.NewPollContext(t.Context(), 10, context.DeadlineExceeded)
			_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
				Variables(varsSet("items", xpath3.ItemSlice{tc.item})).Evaluate(ctx, compiled, nil)
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.Less(t, ctx.Polls(), 20)
		})
	}
}

func TestSequenceBuiltinsHonorCancellation(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{
		"reverse(1 to 20000)",
		"sort(1 to 20000)",
		"insert-before(1 to 6000, 1, 1 to 6000)",
	} {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()
			compiled, err := xpath3.NewCompiler().Compile(expr)
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, nil)
			require.ErrorIs(t, err, context.Canceled)
		})
	}
}

func TestEvaluateRejectsCancelledContext(t *testing.T) {
	t.Parallel()

	compiled, err := xpath3.NewCompiler().Compile("1")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, nil)
	require.ErrorIs(t, err, context.Canceled)
	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).EvaluateEBV(ctx, compiled, nil)
	require.ErrorIs(t, err, context.Canceled)
	state := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).NewEvalState(nil)
	_, err = compiled.EvaluateReuse(ctx, state, nil)
	require.ErrorIs(t, err, context.Canceled)
}
