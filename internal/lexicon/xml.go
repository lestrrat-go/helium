package lexicon

const (
	PrefixXML   = "xml"
	PrefixXMLNS = "xmlns"

	AttrBase  = "base"
	AttrID    = "id"
	AttrLang  = "lang"
	AttrSpace = "space"

	QNameXMLBase  = "xml:base"
	QNameXMLID    = "xml:id"
	QNameXMLLang  = "xml:lang"
	QNameXMLSpace = "xml:space"

	SpaceDefault  = "default"
	SpacePreserve = "preserve"

	ValueYes = "yes"
	ValueNo  = "no"

	// XML declaration pseudo-attributes.
	DeclVersion    = "version"
	DeclEncoding   = "encoding"
	DeclStandalone = "standalone"

	// Common attribute names used across XML vocabularies.
	// Note: AttrName is defined in catalog.go.
	AttrValue   = "value"
	AttrType    = "type"
	AttrClass   = "class"
	AttrStyle   = "style"
	AttrHref    = "href"
	AttrSrc     = "src"
	AttrAlt     = "alt"
	AttrTitle   = "title"
	AttrRef     = "ref"
	AttrSelect  = "select"
	AttrMatch   = "match"
	AttrTest    = "test"
	AttrUse     = "use"
	AttrMode    = "mode"
)

// WellKnownNames is the set of all lexicon string constants, suitable for
// seeding a string intern table. The parser uses this to avoid allocating
// new strings for names that match well-known constants.
var WellKnownNames = [...]string{
	PrefixXML, PrefixXMLNS,
	AttrBase, AttrID, AttrLang, AttrSpace,
	QNameXMLBase, QNameXMLID, QNameXMLLang, QNameXMLSpace,
	SpaceDefault, SpacePreserve,
	ValueYes, ValueNo,
	DeclVersion, DeclEncoding, DeclStandalone,
	AttrName, AttrValue, AttrType, AttrClass, AttrStyle,
	AttrHref, AttrSrc, AttrAlt, AttrTitle,
	AttrRef, AttrSelect, AttrMatch, AttrTest,
	AttrUse, AttrMode,
	// Catalog constants.
	ElemGroup, ElemPublic, ElemSystem, ElemURI,
	AttrPrefer, AttrPublicID, AttrSystemID, AttrCatalog,
}
