package html

import (
	"bytes"

	"github.com/lestrrat-go/helium"
)

// PushParser provides an incremental HTML parsing interface. Data is
// accumulated internally via Push or Write. When Close is called, the
// accumulated data is parsed in one shot and the resulting Document is
// returned.
//
// Unlike the XML PushParser which parses concurrently, the HTML PushParser
// buffers all data because the HTML parser operates on []byte directly
// rather than an io.Reader.
type PushParser struct {
	buf     bytes.Buffer
	sax     SAXHandler
	options []ParseOption
}

// NewPushParser creates an HTML PushParser that builds a DOM tree.
func NewPushParser(options ...ParseOption) *PushParser {
	return &PushParser{options: options}
}

// NewPushParserWithSAX creates an HTML PushParser that fires SAX events
// to the given handler instead of building a DOM tree.
func NewPushParserWithSAX(h SAXHandler, options ...ParseOption) *PushParser {
	return &PushParser{sax: h, options: options}
}

// Push appends a chunk of HTML data to the internal buffer.
func (pp *PushParser) Push(chunk []byte) error {
	_, err := pp.buf.Write(chunk)
	return err
}

// Write implements io.Writer, allowing use with io.Copy and similar functions.
func (pp *PushParser) Write(p []byte) (int, error) {
	return pp.buf.Write(p)
}

// Close parses the accumulated HTML data and returns the resulting Document.
// In SAX mode, it returns (nil, error) after firing all SAX events.
func (pp *PushParser) Close() (*helium.Document, error) {
	data := pp.buf.Bytes()
	if pp.sax != nil {
		return nil, ParseWithSAX(data, pp.sax, pp.options...)
	}
	return Parse(data, pp.options...)
}
