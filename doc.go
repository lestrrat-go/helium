// Package helium is a Go implementation of the libxml2 XML toolkit. It provides
// tree-based XML parsing, a DOM interface, SAX2 callbacks, namespace handling,
// DTD validation, and serialization.
//
// # Parsing
//
// Use [NewParser] to create a parser, configure it with fluent builder methods,
// and call a terminal method to parse XML:
//
//	doc, err := helium.NewParser().
//	    SubstituteEntities(true).
//	    Parse(ctx, xmlBytes)
//
// For file-based input, use [Parser.ParseFile]:
//
//	doc, err := helium.NewParser().ParseFile(ctx, "input.xml")
//
// For streaming input, use [Parser.ParseReader] or [Parser.NewPushParser].
//
// # Serialization
//
// Use [NewWriter] to serialize documents or nodes back to XML:
//
//	err := helium.NewWriter().Format(true).WriteDoc(os.Stdout, doc)
//
// # DOM
//
// The document tree consists of [Node] values. Concrete types include
// [Document], [Element], [Text], [Comment], [CDATASection],
// [ProcessingInstruction], [Attribute], [DTD], and [Namespace].
// Tree traversal helpers include [Walk], [Children], [Descendants], and
// [ChildElements].
//
// # Related packages
//
// Sub-packages provide additional XML processing:
//
//   - [github.com/lestrrat-go/helium/xpath1] — XPath 1.0
//   - [github.com/lestrrat-go/helium/xpath3] — XPath 3.1
//   - [github.com/lestrrat-go/helium/xslt3] — XSLT 3.0
//   - [github.com/lestrrat-go/helium/xsd] — XML Schema validation
//   - [github.com/lestrrat-go/helium/relaxng] — RELAX NG validation
//   - [github.com/lestrrat-go/helium/schematron] — Schematron validation
//   - [github.com/lestrrat-go/helium/c14n] — XML Canonicalization
//   - [github.com/lestrrat-go/helium/xinclude] — XInclude processing
//   - [github.com/lestrrat-go/helium/catalog] — OASIS XML Catalog
//   - [github.com/lestrrat-go/helium/stream] — Streaming XML writer
//   - [github.com/lestrrat-go/helium/sax] — SAX2 handler interfaces
//   - [github.com/lestrrat-go/helium/shim] — Drop-in encoding/xml replacement
//   - [github.com/lestrrat-go/helium/html] — HTML parser
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with helium_). Because examples are in
// a separate test module they do not appear in the generated documentation.
package helium
