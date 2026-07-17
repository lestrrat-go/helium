package helium

import (
	"github.com/lestrrat-go/helium/internal/bitset"
	"github.com/lestrrat-go/helium/internal/parser"
)

// LoadSubsetOption is a bitset of parser augmentation flags applied while
// building the tree: whether to load the external DTD subset, whether to fill
// in DTD-defaulted attributes, and whether to skip ID-attribute interning (the
// last applies to any recognized ID attribute, including xml:id without a DTD).
// The parser derives it from the configured Parser options; the exported
// constants name the individual bits.
type LoadSubsetOption int

const (
	// DetectIDs is the external-DTD-loading intent bit (derived from
	// LoadExternalDTD): it is one of the intents that cause the external subset
	// to be loaded, which lets DTD-declared ID attributes be recognized. The bit
	// itself does not populate the ID table — ID interning happens for any parse
	// unless SkipIDs is set.
	DetectIDs LoadSubsetOption = 1 << (iota + 1)
	// CompleteAttrs adds attributes that the DTD declares with a default value
	// but that the instance omits.
	CompleteAttrs // 4
	// SkipIDs suppresses ID-attribute interning, so no ID table is populated.
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
