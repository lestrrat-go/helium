// Package html implements an HTML parser compatible with libxml2's HTMLparser.
//
// It parses HTML 4.01 documents, producing a helium DOM tree or firing SAX1
// events. Unlike the XML parser in the parent package, the HTML parser is
// case-insensitive, handles void elements, auto-closes elements, and inserts
// implied html/head/body elements.
package html

import (
	"os"

	"github.com/lestrrat-go/helium"
)

// Parse parses HTML data and returns a helium Document.
func Parse(data []byte) (*helium.Document, error) {
	tb := newTreeBuilder()
	p := newParser(data, tb)
	if err := p.parse(); err != nil {
		return nil, err
	}
	if enc := p.detectedEncoding; enc != "" {
		tb.doc.SetEncoding(enc)
	}
	return tb.doc, nil
}

// ParseFile reads and parses an HTML file.
func ParseFile(filename string) (*helium.Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// ParseWithSAX parses HTML data, firing SAX events to the given handler
// without building a DOM tree.
func ParseWithSAX(data []byte, handler SAXHandler) error {
	p := newParser(data, handler)
	return p.parse()
}
