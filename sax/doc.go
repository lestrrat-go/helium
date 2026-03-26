// Package sax defines the SAX2 event-driven XML parsing interface.
//
// The central type is [SAX2Handler], which declares callback methods for all
// XML parsing events: document start/end, element start/end, character data,
// comments, processing instructions, DTD declarations, and error reporting.
//
// Implementations can satisfy individual callback interfaces (e.g.,
// [StartElement], [Characters]) rather than the full [SAX2Handler]. Adapter
// function types like [StartElementFunc] and [CharactersFunc] are provided
// for convenience.
//
// Pass a SAX2Handler to [helium.Parser.SAXHandler] to receive events during
// XML parsing without building a DOM tree.
package sax
