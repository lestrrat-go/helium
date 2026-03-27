package helium

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/internal/bitset"
	"github.com/stretchr/testify/require"
)

func TestLoadSubsetOptionFlagsAreUnique(t *testing.T) {
	t.Parallel()

	assertUniqueFlags(t, []LoadSubsetOption{
		DetectIDs,
		CompleteAttrs,
		SkipIDs,
	})
}

func TestSetAndIsSet(t *testing.T) {
	t.Parallel()

	var l LoadSubsetOption
	require.False(t, l.IsSet(DetectIDs))

	l.Set(DetectIDs)
	require.True(t, l.IsSet(DetectIDs))
	require.False(t, l.IsSet(CompleteAttrs))

	l.Set(CompleteAttrs)
	require.True(t, l.IsSet(DetectIDs))
	require.True(t, l.IsSet(CompleteAttrs))
}

func assertUniqueFlags[T bitset.Field](t *testing.T, flags []T) {
	t.Helper()

	for i, f := range flags {
		t.Run(fmt.Sprintf("flag_%d", i), func(t *testing.T) {
			t.Parallel()

			require.NotZero(t, f, "flag[%d] must be non-zero", i)
			require.Zero(t, f&(f-1), "flag[%d]=%d must be a single-bit value", i, f)
			for j := i + 1; j < len(flags); j++ {
				require.Zero(t, f&flags[j], "flags[%d]=%d and flags[%d]=%d overlap", i, f, j, flags[j])
			}
		})
	}
}
