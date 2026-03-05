package helium_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium"

	"github.com/lestrrat-go/helium/internal/bitset"
	"github.com/stretchr/testify/require"
)

func TestParseOptionBitPositions(t *testing.T) {
	t.Parallel()

	// Verify bit positions match libxml2's XML_PARSE_* constants.
	require.Equal(t, helium.ParseOption(1), helium.ParseRecover)
	require.Equal(t, helium.ParseOption(2), helium.ParseNoEnt)
	require.Equal(t, helium.ParseOption(4), helium.ParseDTDLoad)
	require.Equal(t, helium.ParseOption(8), helium.ParseDTDAttr)
	require.Equal(t, helium.ParseOption(16), helium.ParseDTDValid)
	require.Equal(t, helium.ParseOption(32), helium.ParseNoError)
	require.Equal(t, helium.ParseOption(64), helium.ParseNoWarning)
	require.Equal(t, helium.ParseOption(128), helium.ParsePedantic)
	require.Equal(t, helium.ParseOption(256), helium.ParseNoBlanks)
	require.Equal(t, helium.ParseOption(1024), helium.ParseXInclude)
	require.Equal(t, helium.ParseOption(2048), helium.ParseNoNet)
	require.Equal(t, helium.ParseOption(4096), helium.ParseNoDict)
	require.Equal(t, helium.ParseOption(8192), helium.ParseNsClean)
	require.Equal(t, helium.ParseOption(16384), helium.ParseNoCDATA)
	require.Equal(t, helium.ParseOption(32768), helium.ParseNoXIncNode)
	require.Equal(t, helium.ParseOption(65536), helium.ParseCompact)
	require.Equal(t, helium.ParseOption(262144), helium.ParseNoBaseFix)
	require.Equal(t, helium.ParseOption(524288), helium.ParseHuge)
	require.Equal(t, helium.ParseOption(2097152), helium.ParseIgnoreEnc)
	require.Equal(t, helium.ParseOption(4194304), helium.ParseBigLines)
	require.Equal(t, helium.ParseOption(8388608), helium.ParseNoXXE)
	require.Equal(t, helium.ParseOption(16777216), helium.ParseNoUnzip)
	require.Equal(t, helium.ParseOption(33554432), helium.ParseNoSysCatalog)
	require.Equal(t, helium.ParseOption(67108864), helium.ParseCatalogPI)
	require.Equal(t, helium.ParseOption(134217728), helium.ParseSkipIDs)
}

func TestParseOptionFlagsAreUnique(t *testing.T) {
	t.Parallel()

	assertUniqueFlags(t, []helium.ParseOption{
		helium.ParseRecover,
		helium.ParseNoEnt,
		helium.ParseDTDLoad,
		helium.ParseDTDAttr,
		helium.ParseDTDValid,
		helium.ParseNoError,
		helium.ParseNoWarning,
		helium.ParsePedantic,
		helium.ParseNoBlanks,
		helium.ParseXInclude,
		helium.ParseNoNet,
		helium.ParseNoDict,
		helium.ParseNsClean,
		helium.ParseNoCDATA,
		helium.ParseNoXIncNode,
		helium.ParseCompact,
		helium.ParseNoBaseFix,
		helium.ParseHuge,
		helium.ParseIgnoreEnc,
		helium.ParseBigLines,
		helium.ParseNoXXE,
		helium.ParseNoUnzip,
		helium.ParseNoSysCatalog,
		helium.ParseCatalogPI,
		helium.ParseSkipIDs,
	})
}

func TestLoadSubsetOptionFlagsAreUnique(t *testing.T) {
	t.Parallel()

	assertUniqueFlags(t, []helium.LoadSubsetOption{
		helium.DetectIDs,
		helium.CompleteAttrs,
		helium.SkipIDs,
	})
}

func TestSetAndIsSet(t *testing.T) {
	t.Parallel()

	t.Run("helium.ParseOption", func(t *testing.T) {
		t.Parallel()

		var p helium.ParseOption
		require.False(t, p.IsSet(helium.ParseNoEnt))

		p.Set(helium.ParseNoEnt)
		require.True(t, p.IsSet(helium.ParseNoEnt))
		require.False(t, p.IsSet(helium.ParseDTDLoad))

		p.Set(helium.ParseDTDLoad)
		require.True(t, p.IsSet(helium.ParseNoEnt))
		require.True(t, p.IsSet(helium.ParseDTDLoad))
	})

	t.Run("helium.LoadSubsetOption", func(t *testing.T) {
		t.Parallel()

		var l helium.LoadSubsetOption
		require.False(t, l.IsSet(helium.DetectIDs))

		l.Set(helium.DetectIDs)
		require.True(t, l.IsSet(helium.DetectIDs))
		require.False(t, l.IsSet(helium.CompleteAttrs))

		l.Set(helium.CompleteAttrs)
		require.True(t, l.IsSet(helium.DetectIDs))
		require.True(t, l.IsSet(helium.CompleteAttrs))
	})
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
