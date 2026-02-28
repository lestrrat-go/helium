package helium

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/internal/bitset"
	"github.com/stretchr/testify/require"
)

func TestParseOptionBitPositions(t *testing.T) {
	t.Parallel()

	// Verify bit positions match libxml2's XML_PARSE_* constants.
	require.Equal(t, ParseOption(1), ParseRecover)
	require.Equal(t, ParseOption(2), ParseNoEnt)
	require.Equal(t, ParseOption(4), ParseDTDLoad)
	require.Equal(t, ParseOption(8), ParseDTDAttr)
	require.Equal(t, ParseOption(16), ParseDTDValid)
	require.Equal(t, ParseOption(32), ParseNoError)
	require.Equal(t, ParseOption(64), ParseNoWarning)
	require.Equal(t, ParseOption(128), ParsePedantic)
	require.Equal(t, ParseOption(256), ParseNoBlanks)
	require.Equal(t, ParseOption(1024), ParseXInclude)
	require.Equal(t, ParseOption(2048), ParseNoNet)
	require.Equal(t, ParseOption(4096), ParseNoDict)
	require.Equal(t, ParseOption(8192), ParseNsClean)
	require.Equal(t, ParseOption(16384), ParseNoCDATA)
	require.Equal(t, ParseOption(32768), ParseNoXIncNode)
	require.Equal(t, ParseOption(65536), ParseCompact)
	require.Equal(t, ParseOption(262144), ParseNoBaseFix)
	require.Equal(t, ParseOption(524288), ParseHuge)
	require.Equal(t, ParseOption(2097152), ParseIgnoreEnc)
	require.Equal(t, ParseOption(4194304), ParseBigLines)
	require.Equal(t, ParseOption(8388608), ParseNoXXE)
	require.Equal(t, ParseOption(16777216), ParseNoUnzip)
	require.Equal(t, ParseOption(33554432), ParseNoSysCatalog)
	require.Equal(t, ParseOption(67108864), ParseCatalogPI)
	require.Equal(t, ParseOption(134217728), ParseSkipIDs)
}

func TestParseOptionFlagsAreUnique(t *testing.T) {
	t.Parallel()

	assertUniqueFlags(t, []ParseOption{
		ParseRecover,
		ParseNoEnt,
		ParseDTDLoad,
		ParseDTDAttr,
		ParseDTDValid,
		ParseNoError,
		ParseNoWarning,
		ParsePedantic,
		ParseNoBlanks,
		ParseXInclude,
		ParseNoNet,
		ParseNoDict,
		ParseNsClean,
		ParseNoCDATA,
		ParseNoXIncNode,
		ParseCompact,
		ParseNoBaseFix,
		ParseHuge,
		ParseIgnoreEnc,
		ParseBigLines,
		ParseNoXXE,
		ParseNoUnzip,
		ParseNoSysCatalog,
		ParseCatalogPI,
		ParseSkipIDs,
	})
}

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

	t.Run("ParseOption", func(t *testing.T) {
		t.Parallel()

		var p ParseOption
		require.False(t, p.IsSet(ParseNoEnt))

		p.Set(ParseNoEnt)
		require.True(t, p.IsSet(ParseNoEnt))
		require.False(t, p.IsSet(ParseDTDLoad))

		p.Set(ParseDTDLoad)
		require.True(t, p.IsSet(ParseNoEnt))
		require.True(t, p.IsSet(ParseDTDLoad))
	})

	t.Run("LoadSubsetOption", func(t *testing.T) {
		t.Parallel()

		var l LoadSubsetOption
		require.False(t, l.IsSet(DetectIDs))

		l.Set(DetectIDs)
		require.True(t, l.IsSet(DetectIDs))
		require.False(t, l.IsSet(CompleteAttrs))

		l.Set(CompleteAttrs)
		require.True(t, l.IsSet(DetectIDs))
		require.True(t, l.IsSet(CompleteAttrs))
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
