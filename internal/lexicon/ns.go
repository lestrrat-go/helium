// Package lexicon defines shared spec vocabulary reused across helium packages.
package lexicon

const (
	// NamespaceCatalog is the OASIS XML Catalog namespace.
	NamespaceCatalog = "urn:oasis:names:tc:entity:xmlns:xml:catalog"

	// NamespaceXSLT is the XSLT 1.0/2.0/3.0 namespace.
	NamespaceXSLT = "http://www.w3.org/1999/XSL/Transform"

	// NamespaceXSD is the W3C XML Schema namespace.
	NamespaceXSD = "http://www.w3.org/2001/XMLSchema"

	// NamespaceXSI is the XML Schema instance namespace.
	NamespaceXSI = "http://www.w3.org/2001/XMLSchema-instance"

	// NamespaceXSDVersioning is the XML Schema 1.1 version-control namespace,
	// used for vc:minVersion/vc:maxVersion (and vc:*Available) conditional-
	// inclusion hints on schema documents.
	NamespaceXSDVersioning = "http://www.w3.org/2007/XMLSchema-versioning"

	// NamespaceFn is the XPath 3.1 standard function namespace.
	NamespaceFn = "http://www.w3.org/2005/xpath-functions"

	// NamespaceMath is the XPath 3.1 math function namespace.
	NamespaceMath = "http://www.w3.org/2005/xpath-functions/math"

	// NamespaceMap is the XPath 3.1 map function namespace.
	NamespaceMap = "http://www.w3.org/2005/xpath-functions/map"

	// NamespaceArray is the XPath 3.1 array function namespace.
	NamespaceArray = "http://www.w3.org/2005/xpath-functions/array"

	// NamespaceErr is the XPath/XQuery error namespace.
	NamespaceErr = "http://www.w3.org/2005/xqt-errors"

	// ErrXPTY0004 is the W3C XPath/XQuery type-error code raised when an
	// operand's type is not appropriate for the operation. Used by xpath3
	// internally and asserted by both xpath3 and xslt3 test fixtures.
	ErrXPTY0004 = "XPTY0004"

	// ErrFOCH0001 is the W3C XPath/XQuery error code raised when a codepoint
	// is not a valid XML character (e.g. fn:codepoints-to-string with an
	// out-of-range value). Used by xpath3 and asserted by test fixtures.
	ErrFOCH0001 = "FOCH0001"

	// NamespaceXML is the XML namespace URI (predeclared, prefix "xml").
	NamespaceXML = "http://www.w3.org/XML/1998/namespace"

	// NamespaceXMLNS is the XML Namespaces namespace URI (prefix "xmlns").
	NamespaceXMLNS = "http://www.w3.org/2000/xmlns/"

	// NamespaceXSDDatatypes is the XSD datatypes library namespace (used by RelaxNG).
	NamespaceXSDDatatypes = "http://www.w3.org/2001/XMLSchema-datatypes"

	// NamespaceXHTML is the XHTML namespace.
	NamespaceXHTML = "http://www.w3.org/1999/xhtml"

	// NamespaceSVG is the SVG namespace.
	NamespaceSVG = "http://www.w3.org/2000/svg"

	// NamespaceMathML is the MathML namespace.
	NamespaceMathML = "http://www.w3.org/1998/Math/MathML"

	// NamespaceXInclude is the XInclude 1.0 (Second Edition) namespace.
	NamespaceXInclude = "http://www.w3.org/2001/XInclude"

	// NamespaceXInclude11 is the XInclude 1.1 namespace.
	NamespaceXInclude11 = "http://www.w3.org/2003/XInclude"

	// NamespaceRelaxNG is the RELAX NG structure namespace.
	NamespaceRelaxNG = "http://relaxng.org/ns/structure/1.0"

	// NamespaceSerialization is the XSLT/XQuery serialization parameters namespace.
	NamespaceSerialization = "http://www.w3.org/2010/xslt-xquery-serialization"

	// CollationCodepoint is the XPath codepoint collation URI.
	CollationCodepoint = "http://www.w3.org/2005/xpath-functions/collation/codepoint"

	// CollationUCA is the Unicode Collation Algorithm (UCA) collation URI.
	CollationUCA = "http://www.w3.org/2013/collation/UCA"

	// CollationHTMLASCII is the HTML ASCII case-insensitive collation URI.
	CollationHTMLASCII = "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"

	// NamespaceDSig is the W3C XML Digital Signatures namespace.
	NamespaceDSig = "http://www.w3.org/2000/09/xmldsig#"

	// NamespaceDSig11 is the W3C XML Digital Signatures 1.1 namespace.
	NamespaceDSig11 = "http://www.w3.org/2009/xmldsig11#"

	// NamespaceXMLEnc is the W3C XML Encryption namespace.
	NamespaceXMLEnc = "http://www.w3.org/2001/04/xmlenc#"

	// NamespaceXMLEnc11 is the W3C XML Encryption 1.1 namespace.
	NamespaceXMLEnc11 = "http://www.w3.org/2009/xmlenc11#"
)
