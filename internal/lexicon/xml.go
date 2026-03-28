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
	AttrValue  = "value"
	AttrType   = "type"
	AttrClass  = "class"
	AttrStyle  = "style"
	AttrHref   = "href"
	AttrSrc    = "src"
	AttrAlt    = "alt"
	AttrTitle  = "title"
	AttrRef    = "ref"
	AttrSelect = "select"
	AttrMatch  = "match"
	AttrTest   = "test"
	AttrUse    = "use"
	AttrMode   = "mode"
	AttrCount  = "count"
	AttrFrom   = "from"
)

// Boolean and common attribute value string constants.
const (
	ValueTrue  = "true"
	ValueFalse = "false"
	ValueOmit  = "omit"
)

// URL scheme constants.
const (
	SchemeHTTP  = "http"
	SchemeHTTPS = "https"
	SchemeFile  = "file"
)

// XSD prefixed type name constants (xs: prefix).
const (
	XSUntyped       = "xs:untyped"
	XSAnyType       = "xs:anyType"
	XSAnyAtomicType = "xs:anyAtomicType"
	XSUntypedAtomic = "xs:untypedAtomic"
	XSInteger       = "xs:integer"
	XSBoolean       = "xs:boolean"
)

// XSD type local name constants (unprefixed).
const (
	TypeBoolean            = "boolean"
	TypeString             = "string"
	TypeInteger            = "integer"
	TypeDecimal            = "decimal"
	TypeFloat              = "float"
	TypeDouble             = "double"
	TypeDate               = "date"
	TypeDateTime           = "dateTime"
	TypeTime               = "time"
	TypeDuration           = "duration"
	TypeDayTimeDuration    = "dayTimeDuration"
	TypeYearMonthDuration  = "yearMonthDuration"
	TypeAnyURI             = "anyURI"
	TypeUntypedAtomic      = "untypedAtomic"
	TypePositiveInteger    = "positiveInteger"
	TypeNonNegativeInteger = "nonNegativeInteger"
	TypeIDREF              = "IDREF"
	TypeToken              = "token"
	TypeNumber             = "number"
)

// XPath node test string constants (with parentheses).
const (
	NodeTestNode         = "node()"
	NodeTestElement      = "element()"
	NodeTestAttribute    = "attribute()"
	NodeTestText         = "text()"
	NodeTestComment      = "comment()"
	NodeTestPI           = "processing-instruction()"
	NodeTestDocumentNode = "document-node()"
	NodeTestItem         = "item()"
)

// XPath kind test name constants (without parentheses).
const (
	KindNode            = "node"
	KindText            = "text"
	KindComment         = "comment"
	KindPI              = "processing-instruction"
	KindElement         = "element"
	KindAttribute       = "attribute"
	KindDocumentNode    = "document-node"
	KindSchemaElement   = "schema-element"
	KindSchemaAttribute = "schema-attribute"
	KindNamespaceNode   = "namespace-node"
	KindItem            = "item"
	KindEmptySequence   = "empty-sequence"
)

// Numeric special value string constants.
const (
	FloatINF    = "INF"
	FloatPosINF = "+INF"
	FloatNegINF = "-INF"
	FloatNaN    = "NaN"
)

// XSLT mode value constants.
const (
	ModeAll     = "#all"
	ModeDefault = "#default"
	ModeUnnamed = "#unnamed"
)

// XSD prefix constants.
const (
	PrefixXS  = "xs"
	PrefixXSD = "xsd"
)

// Output encoding constants.
const (
	EncodingUTF8  = "utf-8"
	EncodingUTF8U = "UTF-8"
	EncodingUTF16 = "utf-16"
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
