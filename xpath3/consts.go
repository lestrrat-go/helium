package xpath3

// Namespace prefixes and other string literals reused across the package.
const (
	nsPrefixMap   = "map"
	nsPrefixArray = "array"

	// JSON kind labels used by fn:json-to-xml / fn:xml-to-json and the
	// JSON serializer. The string values match the local-element names
	// defined by the W3C XPath/XQuery 3.1 Functions and Operators spec
	// for the http://www.w3.org/2005/xpath-functions namespace.
	jsonKindNull  = "null"
	jsonKindMap   = nsPrefixMap
	jsonKindArray = nsPrefixArray

	// Duplicate-key handling option values for JSON-related functions.
	duplicatesUseFirst = "use-first"
	duplicatesReject   = "reject"
)
