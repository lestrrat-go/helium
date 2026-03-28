package html

import (
	"bytes"
	"context"

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
//
// (libxml2: htmlCreatePushParserCtxt)
type PushParser struct {
	buf bytes.Buffer
	sax SAXHandler
	cfg parseConfig
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
// When created with [Parser.NewSAXPushParser], it fires SAX events and always
// returns a nil Document; the returned error is non-nil only on parse failure.
func (pp *PushParser) Close(ctx context.Context) (*helium.Document, error) {
	data := pp.buf.Bytes()
	if pp.sax != nil {
		hp := newParser(ctx, data, pp.sax, pp.cfg)
		return nil, hp.parse(ctx)
	}
	tb := newTreeBuilder()
	hp := newParser(ctx, data, tb, pp.cfg)
	if err := hp.parse(ctx); err != nil {
		return nil, err
	}
	if enc := hp.detectedEncoding; enc != "" {
		tb.doc.SetEncoding(enc)
	}
	return tb.doc, nil
}
