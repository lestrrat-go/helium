package xpath3

const (
	// XPath 3.1 reserved keywords. The lexer and token-string tables use
	// these; they also serve as the conventional default namespace prefixes
	// for the map/array function namespaces per the F&O spec.
	keywordMap   = "map"
	keywordArray = "array"

	// JSON kind labels used by fn:json-to-xml / fn:xml-to-json and the
	// JSON serializer. The string values match the local-element names
	// defined by the W3C XPath/XQuery 3.1 F&O spec for the
	// http://www.w3.org/2005/xpath-functions namespace. Kept separate from
	// keywordMap/keywordArray because the conceptual roles are unrelated
	// even though the string values coincide.
	jsonKindNull  = "null"
	jsonKindMap   = "map"
	jsonKindArray = "array"

	// Duplicate-key handling option values for JSON-related functions.
	duplicatesUseFirst = "use-first"
	duplicatesReject   = "reject"
)
