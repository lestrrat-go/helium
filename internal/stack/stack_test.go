package stack_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/stack"
	"github.com/stretchr/testify/require"
)

func TestStack(t *testing.T) {
	t.Parallel()

	t.Run("push, peek, pop", func(t *testing.T) {
		t.Parallel()

		var s stack.Stack[string]
		require.Equal(t, 0, s.Len())
		require.Empty(t, s.Peek(1))

		s.Push("a")
		s.Push("b")
		s.Push("c")
		require.Equal(t, 3, s.Len())
		require.Equal(t, []string{"c"}, s.Peek(1))
		require.Equal(t, []string{"b", "c"}, s.Peek(2))
		require.Equal(t, []string{"a", "b", "c"}, s.Peek(5), "Peek past the bottom returns the whole stack")

		s.Pop()
		require.Equal(t, []string{"a", "b"}, []string(s))
		s.Pop(2)
		require.Equal(t, 0, s.Len())
	})

	t.Run("pop more than the stack holds", func(t *testing.T) {
		t.Parallel()

		var s stack.Stack[int]
		s.Push(1)
		s.Push(2)
		s.Pop(5)
		require.Equal(t, 0, s.Len())
		s.Pop()
		require.Equal(t, 0, s.Len(), "Pop on an empty stack is a no-op")
	})

	t.Run("pop of zero or fewer entries is a no-op", func(t *testing.T) {
		t.Parallel()

		var s stack.Stack[int]
		s.Push(1)
		s.Pop(0)
		s.Pop(-1)
		require.Equal(t, []int{1}, []int(s))
	})

	t.Run("popped slots are cleared", func(t *testing.T) {
		t.Parallel()

		var s stack.Stack[*int]
		v1, v2, v3 := 1, 2, 3
		s.Push(&v1)
		s.Push(&v2)
		s.Push(&v3)
		s.Pop(2)
		require.Equal(t, 1, s.Len())

		// Re-slice up to the old length to look at the vacated slots.
		backing := s[:3]
		require.Same(t, &v1, backing[0])
		require.Nil(t, backing[1])
		require.Nil(t, backing[2])
	})

	t.Run("reuse after deep descent", func(t *testing.T) {
		t.Parallel()

		const depth = 64
		var s stack.Stack[int]
		for i := range depth {
			s.Push(i)
		}
		grown := s.Cap()

		// Walk back up to the bottom one level at a time, the way end tags
		// pop the parser's element stack, then descend again.
		for range depth - 1 {
			s.Pop()
			require.Equal(t, grown, s.Cap(), "Pop keeps the backing array")
		}
		require.Equal(t, []int{0}, []int(s))

		for i := 1; i < depth; i++ {
			s.Push(i * 10)
		}
		require.Equal(t, grown, s.Cap(), "a second descent to the same depth reuses the backing array")
		require.Equal(t, depth, s.Len())
		require.Equal(t, []int{(depth - 2) * 10, (depth - 1) * 10}, s.Peek(2))
	})
}

type keyedItem struct {
	key   string
	value string
}

func (i keyedItem) Key() string {
	return i.key
}

func TestKeyedStack(t *testing.T) {
	t.Parallel()

	t.Run("push, lookup, peek, pop", func(t *testing.T) {
		t.Parallel()

		var s stack.KeyedStack[keyedItem]
		require.NoError(t, s.Push(keyedItem{key: "a", value: "1"}))
		require.NoError(t, s.Push(keyedItem{key: "b", value: "2"}))
		require.ErrorIs(t, s.Push(keyedItem{key: "a", value: "3"}), stack.ErrDuplicateItem)
		require.Equal(t, 2, s.Len())

		item, ok := s.Lookup("a")
		require.True(t, ok)
		require.Equal(t, "1", item.value)
		_, ok = s.Lookup("missing")
		require.False(t, ok)

		require.Equal(t, []keyedItem{{key: "b", value: "2"}}, s.Peek(1))

		s.Pop()
		_, ok = s.Lookup("b")
		require.False(t, ok, "a popped entry is no longer found")
		s.Pop(3)
		require.Equal(t, 0, s.Len())
	})

	t.Run("lookup finds the topmost entry", func(t *testing.T) {
		t.Parallel()

		// The parser's namespace stack appends directly so a child element can
		// shadow a binding; Lookup must return the innermost one.
		s := stack.KeyedStack[keyedItem]{{key: "p", value: "outer"}, {key: "p", value: "inner"}}
		item, ok := s.Lookup("p")
		require.True(t, ok)
		require.Equal(t, "inner", item.value)

		s.Pop()
		item, ok = s.Lookup("p")
		require.True(t, ok)
		require.Equal(t, "outer", item.value)
	})

	t.Run("reuse after deep descent", func(t *testing.T) {
		t.Parallel()

		const depth = 64
		var s stack.KeyedStack[keyedItem]
		for i := range depth {
			require.NoError(t, s.Push(keyedItem{key: string(rune('A' + i))}))
		}
		grown := s.Cap()

		s.Pop(depth - 1)
		require.Equal(t, 1, s.Len())
		require.Equal(t, grown, s.Cap(), "Pop keeps the backing array")
		backing := s[:depth]
		require.Equal(t, keyedItem{}, backing[1], "popped slots are cleared")

		for i := 1; i < depth; i++ {
			require.NoError(t, s.Push(keyedItem{key: string(rune('A' + i))}))
		}
		require.Equal(t, grown, s.Cap(), "a second descent to the same depth reuses the backing array")
	})
}
