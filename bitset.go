package helium

import "github.com/lestrrat-go/helium/internal/bitset"

type LoadSubsetOption int

const (
	DetectIDs     LoadSubsetOption = 1 << (iota + 1)
	CompleteAttrs                                     // 4
	SkipIDs                                           // 8
)

func (p *LoadSubsetOption) Set(n LoadSubsetOption) {
	bitset.Set(p, n)
}

func (p LoadSubsetOption) IsSet(n LoadSubsetOption) bool {
	return bitset.IsSet(p, n)
}

// parseOption is the internal bitset for parser flags.
// Bit positions match libxml2's XML_PARSE_* constants.
type parseOption int

const (
	parseRecover    parseOption = 1 << iota       /* recover on errors */
	parseNoEnt      parseOption = 1 << iota       /* substitute entities */
	parseDTDLoad    parseOption = 1 << iota       /* load the external subset */
	parseDTDAttr    parseOption = 1 << iota       /* default DTD attributes */
	parseDTDValid   parseOption = 1 << iota       /* validate with the DTD */
	parseNoError    parseOption = 1 << iota       /* suppress error reports */
	parseNoWarning  parseOption = 1 << iota       /* suppress warning reports */
	parsePedantic   parseOption = 1 << iota       /* pedantic error reporting */
	parseNoBlanks   parseOption = 1 << iota       /* remove blank nodes */
	parseXInclude   parseOption = 1 << (iota + 1) /* Implement XInclude substitution */
	parseNoNet      parseOption = 1 << (iota + 1) /* Forbid network access */
	parseNoDict     parseOption = 1 << (iota + 1) /* Do not reuse the context dictionary */
	parseNsClean    parseOption = 1 << (iota + 1) /* remove redundant namespaces declarations */
	parseNoCDATA    parseOption = 1 << (iota + 1) /* merge CDATA as text nodes */
	parseNoXIncNode parseOption = 1 << (iota + 1) /* do not generate XINCLUDE START/END nodes */
	parseCompact    parseOption = 1 << (iota + 1) /* compact small text nodes */
	parseNoBaseFix  parseOption = 1 << (iota + 2) /* do not fixup XINCLUDE xml:base uris */
	parseHuge       parseOption = 1 << (iota + 2) /* relax any hardcoded limit from the parser */
	parseIgnoreEnc      parseOption = 1 << (iota + 3) /* ignore internal document encoding hint */
	parseBigLines       parseOption = 1 << (iota + 3) /* Store big lines numbers in text PSVI field */
	parseNoXXE          parseOption = 1 << (iota + 3) /* block external entity/DTD loading */
	parseNoUnzip        parseOption = 1 << (iota + 3) /* no-op: helium has no built-in decompression */
	parseNoSysCatalog   parseOption = 1 << (iota + 3) /* no-op: helium has no global system catalog */
	parseCatalogPI      parseOption = 1 << (iota + 3) /* no-op: catalog PIs not yet supported */
	parseSkipIDs        parseOption = 1 << (iota + 3) /* skip ID attribute interning */

	// Helium extensions — not present in libxml2.

	// parseLenientXMLDecl relaxes XML declaration parsing so that the
	// version, encoding, and standalone pseudo-attributes may appear in
	// any order.
	parseLenientXMLDecl parseOption = 1 << (iota + 3)
)

func (p *parseOption) Set(n parseOption) {
	bitset.Set(p, n)
}

func (p *parseOption) Clear(n parseOption) {
	*p &^= n
}

func (p parseOption) IsSet(n parseOption) bool {
	return bitset.IsSet(p, n)
}
