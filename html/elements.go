package html

import "strings"

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
	{"address", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"applet", 0, 0, 0, false, true, 1, 2, dataNormal},
	{"area", 0, 2, 2, true, false, 0, 0, dataNormal},
	{"b", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"base", 0, 2, 2, true, false, 0, 0, dataNormal},
	{"basefont", 0, 2, 2, true, true, 1, 1, dataNormal},
	{"bdo", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"bgsound", 0, 0, 2, true, false, 0, 0, dataNormal},
	{"big", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"blockquote", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"body", 1, 1, 0, false, false, 0, 0, dataNormal},
	{"br", 0, 2, 2, true, false, 0, 1, dataNormal},
	{"button", 0, 0, 0, false, false, 0, 2, dataNormal},
	{"caption", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"center", 0, 3, 0, false, true, 1, 0, dataNormal},
	{"cite", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"code", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"col", 0, 2, 2, true, false, 0, 0, dataNormal},
	{"colgroup", 0, 1, 0, false, false, 0, 0, dataNormal},
	{"dd", 0, 1, 0, false, false, 0, 0, dataNormal},
	{"del", 0, 0, 0, false, false, 0, 2, dataNormal},
	{"dfn", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"dir", 0, 0, 0, false, true, 1, 0, dataNormal},
	{"div", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"dl", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"dt", 0, 1, 0, false, false, 0, 0, dataNormal},
	{"em", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"embed", 0, 1, 2, true, true, 1, 1, dataNormal},
	{"fieldset", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"font", 0, 3, 0, false, true, 1, 1, dataNormal},
	{"form", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"frame", 0, 2, 2, true, false, 2, 0, dataNormal},
	{"frameset", 0, 0, 0, false, false, 2, 0, dataNormal},
	{"h1", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"h2", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"h3", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"h4", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"h5", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"h6", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"head", 1, 1, 0, false, false, 0, 0, dataNormal},
	{"hr", 0, 2, 2, true, false, 0, 0, dataNormal},
	{"html", 1, 1, 0, false, false, 0, 0, dataNormal},
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
	{"menu", 0, 0, 0, false, true, 1, 0, dataNormal},
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
	{"pre", 0, 0, 0, false, false, 0, 0, dataNormal},
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
	{"table", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"tbody", 1, 0, 0, false, false, 0, 0, dataNormal},
	{"td", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"textarea", 0, 0, 0, false, false, 0, 1, dataRCDATA},
	{"tfoot", 0, 1, 0, false, false, 0, 0, dataNormal},
	{"th", 0, 1, 0, false, false, 0, 0, dataNormal},
	{"thead", 0, 1, 0, false, false, 0, 0, dataNormal},
	{"title", 0, 0, 0, false, false, 0, 0, dataRCDATA},
	{"tr", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"track", 0, 0, 2, true, false, 0, 0, dataNormal},
	{"tt", 0, 3, 0, false, false, 0, 1, dataNormal},
	{"u", 0, 3, 0, false, true, 1, 1, dataNormal},
	{"ul", 0, 0, 0, false, false, 0, 0, dataNormal},
	{"var", 0, 0, 0, false, false, 0, 1, dataNormal},
	{"wbr", 0, 0, 2, true, false, 0, 0, dataNormal},
	{"xmp", 0, 0, 0, false, false, 0, 1, dataRawText},
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
	case "title", "base", "link", "meta", "style", "script":
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
		{"a", "fieldset"},
		{"a", "table"},
		{"a", "td"},
		{"a", "th"},
		{"address", "dd"},
		{"address", "dl"},
		{"address", "dt"},
		{"address", "form"},
		{"address", "li"},
		{"address", "ul"},
		{"b", "center"},
		{"b", "p"},
		{"b", "td"},
		{"b", "th"},
		{"big", "p"},
		{"caption", "col"},
		{"caption", "colgroup"},
		{"caption", "tbody"},
		{"caption", "tfoot"},
		{"caption", "thead"},
		{"caption", "tr"},
		{"col", "col"},
		{"col", "colgroup"},
		{"col", "tbody"},
		{"col", "tfoot"},
		{"col", "thead"},
		{"col", "tr"},
		{"colgroup", "colgroup"},
		{"colgroup", "tbody"},
		{"colgroup", "tfoot"},
		{"colgroup", "thead"},
		{"colgroup", "tr"},
		{"dd", "dt"},
		{"dir", "dd"},
		{"dir", "dl"},
		{"dir", "dt"},
		{"dir", "form"},
		{"dir", "ul"},
		{"dl", "form"},
		{"dl", "li"},
		{"dt", "dd"},
		{"dt", "dl"},
		{"font", "center"},
		{"font", "td"},
		{"font", "th"},
		{"form", "form"},
		{"h1", "fieldset"},
		{"h1", "form"},
		{"h1", "li"},
		{"h1", "p"},
		{"h1", "table"},
		{"h2", "fieldset"},
		{"h2", "form"},
		{"h2", "li"},
		{"h2", "p"},
		{"h2", "table"},
		{"h3", "fieldset"},
		{"h3", "form"},
		{"h3", "li"},
		{"h3", "p"},
		{"h3", "table"},
		{"h4", "fieldset"},
		{"h4", "form"},
		{"h4", "li"},
		{"h4", "p"},
		{"h4", "table"},
		{"h5", "fieldset"},
		{"h5", "form"},
		{"h5", "li"},
		{"h5", "p"},
		{"h5", "table"},
		{"h6", "fieldset"},
		{"h6", "form"},
		{"h6", "li"},
		{"h6", "p"},
		{"h6", "table"},
		{"head", "a"},
		{"head", "abbr"},
		{"head", "acronym"},
		{"head", "address"},
		{"head", "b"},
		{"head", "bdo"},
		{"head", "big"},
		{"head", "blockquote"},
		{"head", "body"},
		{"head", "br"},
		{"head", "center"},
		{"head", "cite"},
		{"head", "code"},
		{"head", "dd"},
		{"head", "dfn"},
		{"head", "dir"},
		{"head", "div"},
		{"head", "dl"},
		{"head", "dt"},
		{"head", "em"},
		{"head", "fieldset"},
		{"head", "font"},
		{"head", "form"},
		{"head", "frameset"},
		{"head", "h1"},
		{"head", "h2"},
		{"head", "h3"},
		{"head", "h4"},
		{"head", "h5"},
		{"head", "h6"},
		{"head", "hr"},
		{"head", "i"},
		{"head", "iframe"},
		{"head", "img"},
		{"head", "kbd"},
		{"head", "li"},
		{"head", "listing"},
		{"head", "map"},
		{"head", "menu"},
		{"head", "ol"},
		{"head", "p"},
		{"head", "pre"},
		{"head", "q"},
		{"head", "s"},
		{"head", "samp"},
		{"head", "small"},
		{"head", "span"},
		{"head", "strike"},
		{"head", "strong"},
		{"head", "sub"},
		{"head", "sup"},
		{"head", "table"},
		{"head", "tt"},
		{"head", "u"},
		{"head", "ul"},
		{"head", "var"},
		{"head", "xmp"},
		{"hr", "form"},
		{"i", "center"},
		{"i", "p"},
		{"i", "td"},
		{"i", "th"},
		{"legend", "fieldset"},
		{"li", "li"},
		{"link", "body"},
		{"link", "frameset"},
		{"listing", "dd"},
		{"listing", "dl"},
		{"listing", "dt"},
		{"listing", "fieldset"},
		{"listing", "form"},
		{"listing", "li"},
		{"listing", "table"},
		{"listing", "ul"},
		{"menu", "dd"},
		{"menu", "dl"},
		{"menu", "dt"},
		{"menu", "form"},
		{"menu", "ul"},
		{"ol", "form"},
		{"option", "optgroup"},
		{"option", "option"},
		{"p", "address"},
		{"p", "blockquote"},
		{"p", "body"},
		{"p", "caption"},
		{"p", "center"},
		{"p", "col"},
		{"p", "colgroup"},
		{"p", "dd"},
		{"p", "dir"},
		{"p", "div"},
		{"p", "dl"},
		{"p", "dt"},
		{"p", "fieldset"},
		{"p", "form"},
		{"p", "frameset"},
		{"p", "h1"},
		{"p", "h2"},
		{"p", "h3"},
		{"p", "h4"},
		{"p", "h5"},
		{"p", "h6"},
		{"p", "head"},
		{"p", "hr"},
		{"p", "li"},
		{"p", "listing"},
		{"p", "menu"},
		{"p", "ol"},
		{"p", "p"},
		{"p", "pre"},
		{"p", "table"},
		{"p", "tbody"},
		{"p", "td"},
		{"p", "tfoot"},
		{"p", "th"},
		{"p", "title"},
		{"p", "tr"},
		{"p", "ul"},
		{"p", "xmp"},
		{"pre", "dd"},
		{"pre", "dl"},
		{"pre", "dt"},
		{"pre", "fieldset"},
		{"pre", "form"},
		{"pre", "li"},
		{"pre", "table"},
		{"pre", "ul"},
		{"s", "p"},
		{"script", "noscript"},
		{"small", "p"},
		{"span", "td"},
		{"span", "th"},
		{"strike", "p"},
		{"style", "body"},
		{"style", "frameset"},
		{"tbody", "tbody"},
		{"tbody", "tfoot"},
		{"td", "tbody"},
		{"td", "td"},
		{"td", "tfoot"},
		{"td", "th"},
		{"td", "tr"},
		{"tfoot", "tbody"},
		{"th", "tbody"},
		{"th", "td"},
		{"th", "tfoot"},
		{"th", "th"},
		{"th", "tr"},
		{"thead", "tbody"},
		{"thead", "tfoot"},
		{"title", "body"},
		{"title", "frameset"},
		{"tr", "tbody"},
		{"tr", "tfoot"},
		{"tr", "tr"},
		{"tt", "p"},
		{"u", "p"},
		{"u", "td"},
		{"u", "th"},
		{"ul", "address"},
		{"ul", "form"},
		{"ul", "menu"},
		{"ul", "pre"},
		{"xmp", "dd"},
		{"xmp", "dl"},
		{"xmp", "dt"},
		{"xmp", "fieldset"},
		{"xmp", "form"},
		{"xmp", "li"},
		{"xmp", "table"},
		{"xmp", "ul"},
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
	"div":   150,
	"td":    160,
	"th":    160,
	"tr":    170,
	"thead": 180,
	"tbody": 180,
	"tfoot": 180,
	"table": 190,
	"head":  200,
	"body":  200,
	"html":  220,
}
