package helium

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOptionBitPositions(t *testing.T) {
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
}
