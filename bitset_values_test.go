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
	require.Equal(t, parseOption(1), parseRecover)
	require.Equal(t, parseOption(2), parseNoEnt)
	require.Equal(t, parseOption(4), parseDTDLoad)
	require.Equal(t, parseOption(8), parseDTDAttr)
	require.Equal(t, parseOption(16), parseDTDValid)
	require.Equal(t, parseOption(32), parseNoError)
	require.Equal(t, parseOption(64), parseNoWarning)
	require.Equal(t, parseOption(128), parsePedantic)
	require.Equal(t, parseOption(256), parseNoBlanks)
	require.Equal(t, parseOption(1024), parseXInclude)
	require.Equal(t, parseOption(2048), parseNoNet)
	require.Equal(t, parseOption(4096), parseNoDict)
	require.Equal(t, parseOption(8192), parseNsClean)
	require.Equal(t, parseOption(16384), parseNoCDATA)
	require.Equal(t, parseOption(32768), parseNoXIncNode)
	require.Equal(t, parseOption(65536), parseCompact)
	require.Equal(t, parseOption(262144), parseNoBaseFix)
	require.Equal(t, parseOption(524288), parseHuge)
	require.Equal(t, parseOption(2097152), parseIgnoreEnc)
	require.Equal(t, parseOption(4194304), parseBigLines)
	require.Equal(t, parseOption(8388608), parseNoXXE)
	require.Equal(t, parseOption(16777216), parseNoUnzip)
	require.Equal(t, parseOption(33554432), parseNoSysCatalog)
	require.Equal(t, parseOption(67108864), parseCatalogPI)
	require.Equal(t, parseOption(134217728), parseSkipIDs)
	require.Equal(t, parseOption(268435456), parseLenientXMLDecl)
}

func TestParseOptionFlagsAreUnique(t *testing.T) {
	t.Parallel()

	assertUniqueFlags(t, []parseOption{
		parseRecover,
		parseNoEnt,
		parseDTDLoad,
		parseDTDAttr,
		parseDTDValid,
		parseNoError,
		parseNoWarning,
		parsePedantic,
		parseNoBlanks,
		parseXInclude,
		parseNoNet,
		parseNoDict,
		parseNsClean,
		parseNoCDATA,
		parseNoXIncNode,
		parseCompact,
		parseNoBaseFix,
		parseHuge,
		parseIgnoreEnc,
		parseBigLines,
		parseNoXXE,
		parseNoUnzip,
		parseNoSysCatalog,
		parseCatalogPI,
		parseSkipIDs,
		parseLenientXMLDecl,
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

	t.Run("parseOption", func(t *testing.T) {
		t.Parallel()

		var p parseOption
		require.False(t, p.IsSet(parseNoEnt))

		p.Set(parseNoEnt)
		require.True(t, p.IsSet(parseNoEnt))
		require.False(t, p.IsSet(parseDTDLoad))

		p.Set(parseDTDLoad)
		require.True(t, p.IsSet(parseNoEnt))
		require.True(t, p.IsSet(parseDTDLoad))
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
