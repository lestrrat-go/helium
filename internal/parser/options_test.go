package parser_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/internal/bitset"
	"github.com/lestrrat-go/helium/internal/parser"
	"github.com/stretchr/testify/require"
)

func TestOptionBitPositions(t *testing.T) {
	t.Parallel()

	// Verify bit positions match libxml2's XML_PARSE_* constants.
	require.Equal(t, parser.Option(1), parser.Recover)
	require.Equal(t, parser.Option(2), parser.NoEnt)
	require.Equal(t, parser.Option(4), parser.DTDLoad)
	require.Equal(t, parser.Option(8), parser.DTDAttr)
	require.Equal(t, parser.Option(16), parser.DTDValid)
	require.Equal(t, parser.Option(32), parser.NoError)
	require.Equal(t, parser.Option(64), parser.NoWarning)
	require.Equal(t, parser.Option(128), parser.Pedantic)
	require.Equal(t, parser.Option(256), parser.NoBlanks)
	require.Equal(t, parser.Option(1024), parser.XInclude)
	require.Equal(t, parser.Option(2048), parser.NoNet)
	require.Equal(t, parser.Option(4096), parser.NoDict)
	require.Equal(t, parser.Option(8192), parser.NsClean)
	require.Equal(t, parser.Option(16384), parser.NoCDATA)
	require.Equal(t, parser.Option(32768), parser.NoXIncNode)
	require.Equal(t, parser.Option(65536), parser.Compact)
	require.Equal(t, parser.Option(262144), parser.NoBaseFix)
	require.Equal(t, parser.Option(524288), parser.Huge)
	require.Equal(t, parser.Option(2097152), parser.IgnoreEnc)
	require.Equal(t, parser.Option(4194304), parser.BigLines)
	require.Equal(t, parser.Option(8388608), parser.NoXXE)
	require.Equal(t, parser.Option(16777216), parser.NoUnzip)
	require.Equal(t, parser.Option(33554432), parser.NoSysCatalog)
	require.Equal(t, parser.Option(67108864), parser.CatalogPI)
	require.Equal(t, parser.Option(134217728), parser.SkipIDs)
	require.Equal(t, parser.Option(268435456), parser.LenientXMLDecl)
}

func TestOptionFlagsAreUnique(t *testing.T) {
	t.Parallel()

	assertUniqueFlags(t, []parser.Option{
		parser.Recover,
		parser.NoEnt,
		parser.DTDLoad,
		parser.DTDAttr,
		parser.DTDValid,
		parser.NoError,
		parser.NoWarning,
		parser.Pedantic,
		parser.NoBlanks,
		parser.XInclude,
		parser.NoNet,
		parser.NoDict,
		parser.NsClean,
		parser.NoCDATA,
		parser.NoXIncNode,
		parser.Compact,
		parser.NoBaseFix,
		parser.Huge,
		parser.IgnoreEnc,
		parser.BigLines,
		parser.NoXXE,
		parser.NoUnzip,
		parser.NoSysCatalog,
		parser.CatalogPI,
		parser.SkipIDs,
		parser.LenientXMLDecl,
	})
}

func TestSetAndIsSet(t *testing.T) {
	t.Parallel()

	var p parser.Option
	require.False(t, p.IsSet(parser.NoEnt))

	p.Set(parser.NoEnt)
	require.True(t, p.IsSet(parser.NoEnt))
	require.False(t, p.IsSet(parser.DTDLoad))

	p.Set(parser.DTDLoad)
	require.True(t, p.IsSet(parser.NoEnt))
	require.True(t, p.IsSet(parser.DTDLoad))
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
