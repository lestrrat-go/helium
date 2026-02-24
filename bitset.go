package helium

type LoadSubsetOption int

const (
	DetectIDs LoadSubsetOption = 1<<iota + 1
	CompleteAttrs
	SkipIDs
)

type ParseOption int

// Note: Many of these are totally unimplemented at this point
const (
	ParseRecover   ParseOption = 1 << iota /* recover on errors */
	ParseNoEnt                             /* substitute entities */
	ParseDTDLoad                           /* load the external subset */
	ParseDTDAttr                           /* default DTD attributes */
	ParseDTDValid                          /* validate with the DTD */
	ParseNoError                           /* suppress error reports */
	ParseNoWarning                         /* suppress warning reports */
	ParsePedantic                          /* pedantic error reporting */
	ParseNoBlanks                          /* remove blank nodes */
	// gap here: ParseSAX1 is not implemented
	ParseXInclude   ParseOption = 1<<iota + 10 /* Implement XInclude substitition  */
	ParseNoNet                                 /* Forbid network access */
	ParseNoDict                                /* Do not reuse the context dictionnary */
	ParseNsClean                               /* remove redundant namespaces declarations */
	ParseNoCDATA                               /* merge CDATA as text nodes */
	ParseNoXIncNode                            /* do not generate XINCLUDE START/END nodes */
	ParseCompact                               /* compact small text nodes; no modification of the tree allowed afterwards (will possibly crash if you try to modify the tree) */
	// ParseOld10 is not implemented
	ParseNoBaseFix /* do not fixup XINCLUDE xml:base uris */
	ParseHuge      /* relax any hardcoded limit from the parser */
	// ParseOldSAX is not implemented
	ParseIgnoreEnc ParseOption = 1<<iota + 21 /* ignore internal document encoding hint */
	ParseBigLines  ParseOption = 1 << 22      /* Store big lines numbers in text PSVI field */
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

