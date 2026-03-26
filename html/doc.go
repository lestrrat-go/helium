// Package html implements an HTML parser compatible with libxml2's HTMLparser.
//
// It parses HTML 4.01 documents, producing a helium DOM tree or firing SAX
// events. Unlike the XML parser in the parent package, the HTML parser is
// case-insensitive, handles void elements, auto-closes elements, and inserts
// implied html/head/body elements.
//
// # Parsing
//
// Use [NewParser] to create a parser, configure it with fluent builder
// methods, and call a terminal method:
//
//	doc, err := html.NewParser().Parse(ctx, htmlBytes)
//
// File-based, SAX-based, and push-parser modes are also available via
// [Parser.ParseFile], [Parser.ParseWithSAX], and [Parser.NewPushParser].
//
// # Serialization
//
// Use [NewWriter] to serialize HTML documents:
//
//	err := html.NewWriter().Format(true).WriteDoc(os.Stdout, doc)
package html
