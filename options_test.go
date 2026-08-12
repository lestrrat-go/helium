package helium_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/bitset"
	"github.com/stretchr/testify/require"
)

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

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("set and is set", func(t *testing.T) {
		var l helium.LoadSubsetOption
		require.False(t, l.IsSet(helium.DetectIDs))

		l.Set(helium.DetectIDs)
		require.True(t, l.IsSet(helium.DetectIDs))
		require.False(t, l.IsSet(helium.CompleteAttrs))

		l.Set(helium.CompleteAttrs)
		require.True(t, l.IsSet(helium.DetectIDs))
		require.True(t, l.IsSet(helium.CompleteAttrs))
	})

	t.Run("flags are unique", func(t *testing.T) {
		assertUniqueFlags(t, []helium.LoadSubsetOption{
			helium.DetectIDs,
			helium.CompleteAttrs,
			helium.SkipIDs,
		})
	})
}
