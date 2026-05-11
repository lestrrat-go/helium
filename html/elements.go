package html

import "strings"

// HTML element name constants used throughout the html package.
// Extracted to satisfy the goconst linter; these names appear many times
// in the element tables and parser logic below.
const (
	elemAddress  = "address"
	elemBody     = "body"
	elemCaption  = "caption"
	elemCenter   = "center"
	elemCol      = "col"
	elemColgroup = "colgroup"
	elemDir      = "dir"
	elemDiv      = "div"
	elemFieldset = "fieldset"
	elemFont     = "font"
	elemForm     = "form"
	elemFrameset = "frameset"
	elemHead     = "head"
	elemHTML     = "html"
	elemListing  = "listing"
	elemMenu     = "menu"
	elemPre      = "pre"
	elemTable    = "table"
	elemTbody    = "tbody"
	elemTfoot    = "tfoot"
	elemThead    = "thead"
	elemTitle    = "title"
	elemXmp      = "xmp"
)

// HTML5 named character entity keys that collide with element-name literals.
// These are entity names (e.g. &div; == U+00F7), not HTML element references,
// so they are kept distinct from the elem* constants.
const (
	entityDiv = "div"
	entityPre = "pre"
)

// dataMode controls how element content is parsed.
type dataMode int

const (
	dataNormal    dataMode = 0 // normal HTML parsing
	dataRCDATA    dataMode = 1 // RCDATA: entities expanded but no tag parsing (title, textarea)
	dataRawText   dataMode = 2 // raw text: no parsing at all (script, style, iframe, xmp, noembed, noframes)
	dataScript    dataMode = 3 // script: like raw text but with special </script> handling
	dataPlaintext dataMode = 4 // plaintext: raw text until end of file
)

// htmlElemDesc describes an HTML element's parsing behavior.
// Ported from libxml2's html40ElementTable (HTMLparser.c:502).
type htmlElemDesc struct {
	name     string
	startTag int      // 0=required, 1=optional
	endTag   int      // 0=required, 1=optional, 2=forbidden(void), 3=stylistic(close easily)
	saveEnd  int      // save end tag
	empty    bool     // void element (no content)
	depr     bool     // deprecated
	dtd      int      // 0=strict, 1=loose only, 2=frameset only
	inline   int      // 0=block, 1=inline, 2=both(inline+block)
	dataMode dataMode // how to parse element content
}

// html40ElementTable is the full HTML 4.0 element table from libxml2.
// name, startTag, endTag, saveEnd, empty, depr, dtd, inline, dataMode
var html40ElementTable = []htmlElemDesc{
	{"a", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"abbr", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"acronym", 0, 0, 0, false, false, 0, 1, dataNormal},
	{elemAddress, 0, 0, 0, false, false, 0, 0, dataNormal},
	{"applet", 0, 0, 0, false, true, 1, 2, dataNormal},
	{"area", 0, 2, 2, true, false, 0, 0, dataNormal},
	{"b", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"base", 0, 2, 2, true, false, 0, 0, dataNormal},
	{"basefont", 0, 2, 2, true, true, 1, 1, dataNormal},
	{"bdo", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"bgsound", 0, 0, 2, true, false, 0, 0, dataNormal},
	{"big", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"blockquote", 0, 0, 0, false, false, 0, 0, dataNormal},
	{elemBody, 1, 1, 0, false, false, 0, 0, dataNormal},
	{"br", 0, 2, 2, true, false, 0, 1, dataNormal},
	{"button", 0, 0, 0, false, false, 0, 2, dataNormal},
	{elemCaption, 0, 0, 0, false, false, 0, 0, dataNormal},
	{elemCenter, 0, 3, 0, false, true, 1, 0, dataNormal},
	{"cite", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"code", 0, 0, 0, false, false, 0, 1, dataNormal},
	{elemCol, 0, 2, 2, true, false, 0, 0, dataNormal},
	{elemColgroup, 0, 1, 0, false, false, 0, 0, dataNormal},
	{"dd", 0, 1, 0, false, false, 0, 0, dataNormal},
	{"del", 0, 0, 0, false, false, 0, 2, dataNormal},
	{"dfn", 0, 0, 0, false, false, 0, 1, dataNormal},
	{elemDir, 0, 0, 0, false, true, 1, 0, dataNormal},
	{elemDiv, 0, 0, 0, false, false, 0, 0, dataNormal},
	{"dl", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"dt", 0, 1, 0, false, false, 0, 0, dataNormal},
	{"em", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"embed", 0, 1, 2, true, true, 1, 1, dataNormal},
	{elemFieldset, 0, 0, 0, false, false, 0, 0, dataNormal},
	{elemFont, 0, 3, 0, false, true, 1, 1, dataNormal},
	{elemForm, 0, 0, 0, false, false, 0, 0, dataNormal},
	{"frame", 0, 2, 2, true, false, 2, 0, dataNormal},
	{elemFrameset, 0, 0, 0, false, false, 2, 0, dataNormal},
	{"h1", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"h2", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"h3", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"h4", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"h5", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"h6", 0, 0, 0, false, false, 0, 0, dataNormal},
	{elemHead, 1, 1, 0, false, false, 0, 0, dataNormal},
	{"hr", 0, 2, 2, true, false, 0, 0, dataNormal},
	{elemHTML, 1, 1, 0, false, false, 0, 0, dataNormal},
	{"i", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"iframe", 0, 0, 0, false, false, 1, 2, dataRawText},
	{"img", 0, 2, 2, true, false, 0, 1, dataNormal},
	{"input", 0, 2, 2, true, false, 0, 1, dataNormal},
	{"ins", 0, 0, 0, false, false, 0, 2, dataNormal},
	{"isindex", 0, 2, 2, true, true, 1, 0, dataNormal},
	{"kbd", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"keygen", 0, 0, 2, true, false, 0, 0, dataNormal},
	{"label", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"legend", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"li", 0, 1, 1, false, false, 0, 0, dataNormal},
	{"link", 0, 2, 2, true, false, 0, 0, dataNormal},
	{"map", 0, 0, 0, false, false, 0, 2, dataNormal},
	{elemMenu, 0, 0, 0, false, true, 1, 0, dataNormal},
	{"meta", 0, 2, 2, true, false, 0, 0, dataNormal},
	{"noembed", 0, 0, 0, false, false, 0, 0, dataRawText},
	{"noframes", 0, 0, 0, false, false, 2, 0, dataRawText},
	{"nobr", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"noscript", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"object", 0, 0, 0, false, false, 0, 2, dataNormal},
	{"ol", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"optgroup", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"option", 0, 1, 0, false, false, 0, 0, dataNormal},
	{"p", 0, 1, 0, false, false, 0, 0, dataNormal},
	{"param", 0, 2, 2, true, false, 0, 0, dataNormal},
	{"plaintext", 0, 0, 0, false, false, 0, 0, dataPlaintext},
	{elemPre, 0, 0, 0, false, false, 0, 0, dataNormal},
	{"q", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"s", 0, 3, 0, false, true, 1, 1, dataNormal},
	{"samp", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"script", 0, 0, 0, false, false, 0, 2, dataScript},
	{"select", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"small", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"source", 0, 0, 2, true, false, 0, 0, dataNormal},
	{"span", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"strike", 0, 3, 0, false, true, 1, 1, dataNormal},
	{"strong", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"style", 0, 0, 0, false, false, 0, 0, dataRawText},
	{"sub", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"sup", 0, 3, 0, false, false, 0, 1, dataNormal},
	{elemTable, 0, 0, 0, false, false, 0, 0, dataNormal},
	{elemTbody, 1, 0, 0, false, false, 0, 0, dataNormal},
	{"td", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"textarea", 0, 0, 0, false, false, 0, 1, dataRCDATA},
	{elemTfoot, 0, 1, 0, false, false, 0, 0, dataNormal},
	{"th", 0, 1, 0, false, false, 0, 0, dataNormal},
	{elemThead, 0, 1, 0, false, false, 0, 0, dataNormal},
	{elemTitle, 0, 0, 0, false, false, 0, 0, dataRCDATA},
	{"tr", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"track", 0, 0, 2, true, false, 0, 0, dataNormal},
	{"tt", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"u", 0, 3, 0, false, true, 1, 1, dataNormal},
	{"ul", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"var", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"wbr", 0, 0, 2, true, false, 0, 0, dataNormal},
	{elemXmp, 0, 0, 0, false, false, 0, 1, dataRawText},
}

// elemMap is a lookup map for element descriptors.
var elemMap map[string]*htmlElemDesc

func init() {
	elemMap = make(map[string]*htmlElemDesc, len(html40ElementTable))
	for i := range html40ElementTable {
		elemMap[html40ElementTable[i].name] = &html40ElementTable[i]
	}
}

// lookupElement returns the element descriptor for the given tag name (case-insensitive).
func lookupElement(name string) *htmlElemDesc {
	return elemMap[strings.ToLower(name)]
}

// isHeadElement returns true if the tag belongs in <head>.
func isHeadElement(name string) bool {
	switch strings.ToLower(name) {
	case elemTitle, "base", "link", "meta", "style", "script":
		return true
	}
	return false
}

// htmlStartClose is the auto-close rules table from libxml2.
// If we encounter newTag while oldTag is open, we close oldTag first.
// Stored as map[oldTag]map[newTag]bool for O(1) lookup.
var htmlStartClose map[string]map[string]bool

func init() {
	entries := [][2]string{
		{"a", "a"},
		{"a", elemFieldset},
		{"a", elemTable},
		{"a", "td"},
		{"a", "th"},
		{elemAddress, "dd"},
		{elemAddress, "dl"},
		{elemAddress, "dt"},
		{elemAddress, elemForm},
		{elemAddress, "li"},
		{elemAddress, "ul"},
		{"b", elemCenter},
		{"b", "p"},
		{"b", "td"},
		{"b", "th"},
		{"big", "p"},
		{elemCaption, elemCol},
		{elemCaption, elemColgroup},
		{elemCaption, elemTbody},
		{elemCaption, elemTfoot},
		{elemCaption, elemThead},
		{elemCaption, "tr"},
		{elemCol, elemCol},
		{elemCol, elemColgroup},
		{elemCol, elemTbody},
		{elemCol, elemTfoot},
		{elemCol, elemThead},
		{elemCol, "tr"},
		{elemColgroup, elemColgroup},
		{elemColgroup, elemTbody},
		{elemColgroup, elemTfoot},
		{elemColgroup, elemThead},
		{elemColgroup, "tr"},
		{"dd", "dt"},
		{elemDir, "dd"},
		{elemDir, "dl"},
		{elemDir, "dt"},
		{elemDir, elemForm},
		{elemDir, "ul"},
		{"dl", elemForm},
		{"dl", "li"},
		{"dt", "dd"},
		{"dt", "dl"},
		{elemFont, elemCenter},
		{elemFont, "td"},
		{elemFont, "th"},
		{elemForm, elemForm},
		{"h1", elemFieldset},
		{"h1", elemForm},
		{"h1", "li"},
		{"h1", "p"},
		{"h1", elemTable},
		{"h2", elemFieldset},
		{"h2", elemForm},
		{"h2", "li"},
		{"h2", "p"},
		{"h2", elemTable},
		{"h3", elemFieldset},
		{"h3", elemForm},
		{"h3", "li"},
		{"h3", "p"},
		{"h3", elemTable},
		{"h4", elemFieldset},
		{"h4", elemForm},
		{"h4", "li"},
		{"h4", "p"},
		{"h4", elemTable},
		{"h5", elemFieldset},
		{"h5", elemForm},
		{"h5", "li"},
		{"h5", "p"},
		{"h5", elemTable},
		{"h6", elemFieldset},
		{"h6", elemForm},
		{"h6", "li"},
		{"h6", "p"},
		{"h6", elemTable},
		{elemHead, "a"},
		{elemHead, "abbr"},
		{elemHead, "acronym"},
		{elemHead, elemAddress},
		{elemHead, "b"},
		{elemHead, "bdo"},
		{elemHead, "big"},
		{elemHead, "blockquote"},
		{elemHead, elemBody},
		{elemHead, "br"},
		{elemHead, elemCenter},
		{elemHead, "cite"},
		{elemHead, "code"},
		{elemHead, "dd"},
		{elemHead, "dfn"},
		{elemHead, elemDir},
		{elemHead, elemDiv},
		{elemHead, "dl"},
		{elemHead, "dt"},
		{elemHead, "em"},
		{elemHead, elemFieldset},
		{elemHead, elemFont},
		{elemHead, elemForm},
		{elemHead, elemFrameset},
		{elemHead, "h1"},
		{elemHead, "h2"},
		{elemHead, "h3"},
		{elemHead, "h4"},
		{elemHead, "h5"},
		{elemHead, "h6"},
		{elemHead, "hr"},
		{elemHead, "i"},
		{elemHead, "iframe"},
		{elemHead, "img"},
		{elemHead, "kbd"},
		{elemHead, "li"},
		{elemHead, elemListing},
		{elemHead, "map"},
		{elemHead, elemMenu},
		{elemHead, "ol"},
		{elemHead, "p"},
		{elemHead, elemPre},
		{elemHead, "q"},
		{elemHead, "s"},
		{elemHead, "samp"},
		{elemHead, "small"},
		{elemHead, "span"},
		{elemHead, "strike"},
		{elemHead, "strong"},
		{elemHead, "sub"},
		{elemHead, "sup"},
		{elemHead, elemTable},
		{elemHead, "tt"},
		{elemHead, "u"},
		{elemHead, "ul"},
		{elemHead, "var"},
		{elemHead, elemXmp},
		{"hr", elemForm},
		{"i", elemCenter},
		{"i", "p"},
		{"i", "td"},
		{"i", "th"},
		{"legend", elemFieldset},
		{"li", "li"},
		{"link", elemBody},
		{"link", elemFrameset},
		{elemListing, "dd"},
		{elemListing, "dl"},
		{elemListing, "dt"},
		{elemListing, elemFieldset},
		{elemListing, elemForm},
		{elemListing, "li"},
		{elemListing, elemTable},
		{elemListing, "ul"},
		{elemMenu, "dd"},
		{elemMenu, "dl"},
		{elemMenu, "dt"},
		{elemMenu, elemForm},
		{elemMenu, "ul"},
		{"ol", elemForm},
		{"option", "optgroup"},
		{"option", "option"},
		{"p", elemAddress},
		{"p", "blockquote"},
		{"p", elemBody},
		{"p", elemCaption},
		{"p", elemCenter},
		{"p", elemCol},
		{"p", elemColgroup},
		{"p", "dd"},
		{"p", elemDir},
		{"p", elemDiv},
		{"p", "dl"},
		{"p", "dt"},
		{"p", elemFieldset},
		{"p", elemForm},
		{"p", elemFrameset},
		{"p", "h1"},
		{"p", "h2"},
		{"p", "h3"},
		{"p", "h4"},
		{"p", "h5"},
		{"p", "h6"},
		{"p", elemHead},
		{"p", "hr"},
		{"p", "li"},
		{"p", elemListing},
		{"p", elemMenu},
		{"p", "ol"},
		{"p", "p"},
		{"p", elemPre},
		{"p", elemTable},
		{"p", elemTbody},
		{"p", "td"},
		{"p", elemTfoot},
		{"p", "th"},
		{"p", elemTitle},
		{"p", "tr"},
		{"p", "ul"},
		{"p", elemXmp},
		{elemPre, "dd"},
		{elemPre, "dl"},
		{elemPre, "dt"},
		{elemPre, elemFieldset},
		{elemPre, elemForm},
		{elemPre, "li"},
		{elemPre, elemTable},
		{elemPre, "ul"},
		{"s", "p"},
		{"script", "noscript"},
		{"small", "p"},
		{"span", "td"},
		{"span", "th"},
		{"strike", "p"},
		{"style", elemBody},
		{"style", elemFrameset},
		{elemTbody, elemTbody},
		{elemTbody, elemTfoot},
		{"td", elemTbody},
		{"td", "td"},
		{"td", elemTfoot},
		{"td", "th"},
		{"td", "tr"},
		{elemTfoot, elemTbody},
		{"th", elemTbody},
		{"th", "td"},
		{"th", elemTfoot},
		{"th", "th"},
		{"th", "tr"},
		{elemThead, elemTbody},
		{elemThead, elemTfoot},
		{elemTitle, elemBody},
		{elemTitle, elemFrameset},
		{"tr", elemTbody},
		{"tr", elemTfoot},
		{"tr", "tr"},
		{"tt", "p"},
		{"u", "p"},
		{"u", "td"},
		{"u", "th"},
		{"ul", elemAddress},
		{"ul", elemForm},
		{"ul", elemMenu},
		{"ul", elemPre},
		{elemXmp, "dd"},
		{elemXmp, "dl"},
		{elemXmp, "dt"},
		{elemXmp, elemFieldset},
		{elemXmp, elemForm},
		{elemXmp, "li"},
		{elemXmp, elemTable},
		{elemXmp, "ul"},
	}

	htmlStartClose = make(map[string]map[string]bool, 64)
	for _, e := range entries {
		if htmlStartClose[e[0]] == nil {
			htmlStartClose[e[0]] = make(map[string]bool, 4)
		}
		htmlStartClose[e[0]][e[1]] = true
	}
}

// shouldAutoClose returns true if opening newTag should auto-close oldTag.
func shouldAutoClose(oldTag, newTag string) bool {
	m, ok := htmlStartClose[oldTag]
	if !ok {
		return false
	}
	return m[newTag]
}

// htmlEndPriority is used to determine how end tags are handled.
// Higher priority end tags can close lower priority elements.
// Ported from libxml2.
var htmlEndPriority = map[string]int{
	elemDiv:   150,
	"td":      160,
	"th":      160,
	"tr":      170,
	elemThead: 180,
	elemTbody: 180,
	elemTfoot: 180,
	elemTable: 190,
	elemHead:  200,
	elemBody:  200,
	elemHTML:  220,
}
