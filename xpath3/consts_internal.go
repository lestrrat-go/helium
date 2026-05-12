package xpath3

// Internal string constants shared across the xpath3 package, factored out to
// avoid repeated literals (and the goconst linter complaining about them).

// Common error message fragments.
const (
	msgContextItemAbsent = "context item is absent"
	msgMalformedTextDecl = "parse-xml-fragment: malformed text declaration"
)

// Conventional XPath namespace prefixes that appear in user-facing values
// (QName prefixes, namespace bindings, etc.).
const (
	prefixXML = "xml"
	prefixErr = "err"
)

// XPath 3.1 map/array keyword strings used as token keywords, JSON XML element
// names, and namespace prefix labels.
const (
	keywordMap   = "map"
	keywordArray = "array"
)

// JSON serialization atom names.
const jsonNull = "null"

// "duplicates" option values for map:merge / json-to-xml / parse-json.
const (
	duplicatesUseFirst = "use-first"
	duplicatesUseLast  = "use-last"
	duplicatesReject   = "reject"
	duplicatesRetain   = "retain"
)

// Serialization method names.
const serializeMethodXML = "xml"
