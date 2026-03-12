package xpath3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVariableScopeLookupPrefersNearestBinding(t *testing.T) {
	root := scopeWithBindings(nil, map[string]Sequence{
		"x": SingleInteger(1),
	})
	child := scopeWithBindings(root, map[string]Sequence{
		"y": SingleInteger(2),
	})
	shadow := scopeWithBindings(child, map[string]Sequence{
		"x": SingleInteger(3),
	})

	seq, ok := shadow.Lookup("x")
	require.True(t, ok)
	require.Equal(t, int64(3), seq[0].(AtomicValue).IntegerVal())

	seq, ok = shadow.Lookup("y")
	require.True(t, ok)
	require.Equal(t, int64(2), seq[0].(AtomicValue).IntegerVal())

	seq, ok = root.Lookup("x")
	require.True(t, ok)
	require.Equal(t, int64(1), seq[0].(AtomicValue).IntegerVal())
}
