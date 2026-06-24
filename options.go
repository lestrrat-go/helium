package helium

import (
	"github.com/lestrrat-go/helium/internal/bitset"
	"github.com/lestrrat-go/helium/internal/parser"
)

type LoadSubsetOption int

const (
	DetectIDs     LoadSubsetOption = 1 << (iota + 1)
	CompleteAttrs                  // 4
	SkipIDs                        // 8
)

func (p *LoadSubsetOption) Set(n LoadSubsetOption) {
	bitset.Set(p, n)
}

func (p LoadSubsetOption) IsSet(n LoadSubsetOption) bool {
	return bitset.IsSet(p, n)
}

// parseOption is an alias for parser.Option so that existing code in the
// helium package can continue to use the unexported name unchanged.
type parseOption = parser.Option

const (
	parseRecover        = parser.Recover
	parseNoEnt          = parser.NoEnt
	parseDTDLoad        = parser.DTDLoad
	parseDTDAttr        = parser.DTDAttr
	parseDTDValid       = parser.DTDValid
	parseNoError        = parser.NoError
	parseNoWarning      = parser.NoWarning
	parsePedantic       = parser.Pedantic
	parseNoBlanks       = parser.NoBlanks
	parseXInclude       = parser.XInclude
	parseNoNet          = parser.NoNet
	parseNoDict         = parser.NoDict
	parseNsClean        = parser.NsClean
	parseNoCDATA        = parser.NoCDATA
	parseNoXIncNode     = parser.NoXIncNode
	parseCompact        = parser.Compact
	parseNoBaseFix      = parser.NoBaseFix
	parseHuge           = parser.Huge
	parseIgnoreEnc      = parser.IgnoreEnc
	parseBigLines       = parser.BigLines
	parseNoXXE          = parser.NoXXE
	parseNoUnzip        = parser.NoUnzip
	parseNoSysCatalog   = parser.NoSysCatalog
	parseCatalogPI      = parser.CatalogPI
	parseSkipIDs        = parser.SkipIDs
	parseLenientXMLDecl = parser.LenientXMLDecl
)
