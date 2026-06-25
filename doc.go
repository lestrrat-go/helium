// Package helium is a fast, pure-Go XML toolkit covering XML parsing,
// SAX2-style streaming, XPath 3.1, XSLT 3.0, XInclude, XSD, Relax NG, and
// Schematron. This root package provides tree-based XML parsing, a DOM
// interface, SAX2 callbacks, namespace handling, DTD validation, and
// serialization. See the sub-packages listed below for additional features.
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
// # Security defaults
//
// [NewParser] is secure by default and safe for untrusted input: it blocks
// external entity and DTD loading ([Parser.BlockXXE] is on), exposes no
// filesystem ([Parser.FS] defaults to a deny-all FS), forbids network access
// ([Parser.AllowNetwork] is off), and caps element nesting depth at 256
// ([Parser.MaxDepth]). Entity substitution, external-DTD loading, XInclude, and
// DTD validation are likewise off by default.
//
// Because of this, a document that references an external DTD or entity — for
// example via [Parser.ParseFile] on a file with a SYSTEM DTD — will not load
// that resource by default. To deliberately load external resources from a
// trusted source, opt in explicitly and supply a filesystem:
//
//	doc, err := helium.NewParser().
//	    BlockXXE(false).
//	    LoadExternalDTD(true).
//	    FS(helium.PermissiveFS()). // any os.Open path; or a confined fs.FS
//	    Parse(ctx, xmlBytes)
//
// [PermissiveFS] restores the historical unsandboxed behavior; prefer a confined
// [io/fs.FS] when the document's external references are known.
//
// # Serialization
//
// Use [NewWriter] to serialize documents or nodes back to XML:
//
//	err := helium.NewWriter().Format(true).WriteTo(os.Stdout, doc)
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
