package parser

import "github.com/lestrrat-go/helium/internal/bitset"

// Option is the bitset for parser flags.
// Bit positions match libxml2's XML_PARSE_* constants.
type Option int

const (
	Recover    Option = 1 << iota       /* recover on errors */
	NoEnt      Option = 1 << iota       /* substitute entities */
	DTDLoad    Option = 1 << iota       /* load the external subset */
	DTDAttr    Option = 1 << iota       /* default DTD attributes */
	DTDValid   Option = 1 << iota       /* validate with the DTD */
	NoError    Option = 1 << iota       /* suppress error reports */
	NoWarning  Option = 1 << iota       /* suppress warning reports */
	Pedantic   Option = 1 << iota       /* pedantic error reporting */
	NoBlanks   Option = 1 << iota       /* remove blank nodes */
	XInclude   Option = 1 << (iota + 1) /* Implement XInclude substitution */
	NoNet      Option = 1 << (iota + 1) /* Forbid network access */
	NoDict     Option = 1 << (iota + 1) /* Do not reuse the context dictionary */
	NsClean    Option = 1 << (iota + 1) /* remove redundant namespaces declarations */
	NoCDATA    Option = 1 << (iota + 1) /* merge CDATA as text nodes */
	NoXIncNode Option = 1 << (iota + 1) /* do not generate XINCLUDE START/END nodes */
	Compact    Option = 1 << (iota + 1) /* compact small text nodes */
	NoBaseFix  Option = 1 << (iota + 2) /* do not fixup XINCLUDE xml:base uris */
	Huge       Option = 1 << (iota + 2) /* relax any hardcoded limit from the parser */
	IgnoreEnc      Option = 1 << (iota + 3) /* ignore internal document encoding hint */
	BigLines       Option = 1 << (iota + 3) /* Store big lines numbers in text PSVI field */
	NoXXE          Option = 1 << (iota + 3) /* block external entity/DTD loading */
	NoUnzip        Option = 1 << (iota + 3) /* no-op: helium has no built-in decompression */
	NoSysCatalog   Option = 1 << (iota + 3) /* no-op: helium has no global system catalog */
	CatalogPI      Option = 1 << (iota + 3) /* no-op: catalog PIs not yet supported */
	SkipIDs        Option = 1 << (iota + 3) /* skip ID attribute interning */

	// Helium extensions — not present in libxml2.

	// LenientXMLDecl relaxes XML declaration parsing so that the
	// version, encoding, and standalone pseudo-attributes may appear in
	// any order.
	LenientXMLDecl Option = 1 << (iota + 3)
)

func (p *Option) Set(n Option) {
	bitset.Set(p, n)
}

func (p *Option) Clear(n Option) {
	*p &^= n
}

func (p Option) IsSet(n Option) bool {
	return bitset.IsSet(p, n)
}
