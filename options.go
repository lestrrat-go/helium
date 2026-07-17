package helium

import (
	"github.com/lestrrat-go/helium/internal/bitset"
	"github.com/lestrrat-go/helium/internal/parser"
)

// LoadSubsetOption is a bitset of parser tree-building flags: load the external
// DTD subset, fill in DTD-defaulted attributes, and skip ID-attribute interning.
// The parser sets its bits from the configured Parser options; the exported
// constants name the individual bits.
type LoadSubsetOption int

const (
	// DetectIDs is the bit the parser sets from the LoadExternalDTD option. The
	// parser loads the external DTD subset if any of DetectIDs, CompleteAttrs, or
	// DTD validation is set. ID-attribute interning is governed by SkipIDs, not
	// this bit.
	DetectIDs LoadSubsetOption = 1 << (iota + 1)
	// CompleteAttrs is the bit the parser sets from the DefaultDTDAttributes
	// option. With it set, the parser adds attributes that the DTD declares with a
	// default value and that the instance omits.
	CompleteAttrs // 4
	// SkipIDs suppresses ID-attribute interning.
	SkipIDs // 8
)

// Set turns on the bits in n.
func (p *LoadSubsetOption) Set(n LoadSubsetOption) {
	bitset.Set(p, n)
}

// IsSet reports whether any bit in n is turned on.
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
	parseNoNet          = parser.NoNet
	parseNsClean        = parser.NsClean
	parseNoCDATA        = parser.NoCDATA
	parseNoBaseFix      = parser.NoBaseFix
	parseIgnoreEnc      = parser.IgnoreEnc
	parseNoXXE          = parser.NoXXE
	parseNoUnzip        = parser.NoUnzip
	parseNoSysCatalog   = parser.NoSysCatalog
	parseCatalogPI      = parser.CatalogPI
	parseSkipIDs        = parser.SkipIDs
	parseLenientXMLDecl = parser.LenientXMLDecl
)
