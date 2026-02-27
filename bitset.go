package helium

type LoadSubsetOption int

const (
	DetectIDs LoadSubsetOption = 1<<iota + 1
	CompleteAttrs
	SkipIDs
)

type ParseOption int

// Bit positions match libxml2's XML_PARSE_* constants.
const (
	ParseRecover    ParseOption = 1 << iota       /* recover on errors */
	ParseNoEnt      ParseOption = 1 << iota       /* substitute entities */
	ParseDTDLoad    ParseOption = 1 << iota       /* load the external subset */
	ParseDTDAttr    ParseOption = 1 << iota       /* default DTD attributes */
	ParseDTDValid   ParseOption = 1 << iota       /* validate with the DTD */
	ParseNoError    ParseOption = 1 << iota       /* suppress error reports */
	ParseNoWarning  ParseOption = 1 << iota       /* suppress warning reports */
	ParsePedantic   ParseOption = 1 << iota       /* pedantic error reporting */
	ParseNoBlanks   ParseOption = 1 << iota       /* remove blank nodes */
	ParseXInclude   ParseOption = 1 << (iota + 1) /* Implement XInclude substitution */
	ParseNoNet      ParseOption = 1 << (iota + 1) /* Forbid network access */
	ParseNoDict     ParseOption = 1 << (iota + 1) /* Do not reuse the context dictionary */
	ParseNsClean    ParseOption = 1 << (iota + 1) /* remove redundant namespaces declarations */
	ParseNoCDATA    ParseOption = 1 << (iota + 1) /* merge CDATA as text nodes */
	ParseNoXIncNode ParseOption = 1 << (iota + 1) /* do not generate XINCLUDE START/END nodes */
	ParseCompact    ParseOption = 1 << (iota + 1) /* compact small text nodes */
	ParseNoBaseFix  ParseOption = 1 << (iota + 2) /* do not fixup XINCLUDE xml:base uris */
	ParseHuge       ParseOption = 1 << (iota + 2) /* relax any hardcoded limit from the parser */
	ParseIgnoreEnc  ParseOption = 1 << (iota + 3) /* ignore internal document encoding hint */
	ParseBigLines   ParseOption = 1 << (iota + 3) /* Store big lines numbers in text PSVI field */
)

func (p *ParseOption) Set(n ParseOption) {
	*p = *p | n
}

func (p ParseOption) IsSet(n ParseOption) bool {
	return p & n != 0
}

func (p *LoadSubsetOption) Set(n LoadSubsetOption) {
	*p = *p | n
}

func (p LoadSubsetOption) IsSet(n LoadSubsetOption) bool {
	return p & n != 0
}

