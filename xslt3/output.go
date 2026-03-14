package xslt3

import (
	"io"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/stream"
	"github.com/lestrrat-go/helium/xpath3"
)

// outputFrame represents the current output target during transformation.
type outputFrame struct {
	doc          *helium.Document // result document being built
	current      helium.Node      // current insertion point
	captureItems bool             // when true, xsl:sequence adds to pendingItems instead of DOM
	pendingItems xpath3.Sequence  // captured items from xsl:sequence
}

// serializeResult writes the result document to a writer according to the
// output definition.
func serializeResult(w io.Writer, doc *helium.Document, outDef *OutputDef) error {
	if outDef == nil {
		outDef = defaultOutputDef()
	}

	switch outDef.Method {
	case "text":
		return serializeText(w, doc)
	case "html":
		return serializeHTML(w, doc, outDef)
	default:
		return serializeXML(w, doc, outDef)
	}
}

func defaultOutputDef() *OutputDef {
	return &OutputDef{
		Method:   "xml",
		Encoding: "UTF-8",
		Version:  "1.0",
	}
}

func serializeXML(w io.Writer, doc *helium.Document, outDef *OutputDef) error {
	var opts []helium.WriteOption
	if outDef.Indent {
		opts = append(opts, helium.WithFormat())
	}
	if outDef.OmitDeclaration {
		opts = append(opts, helium.WithNoDecl())
	}
	return doc.XML(w, opts...)
}

func serializeText(w io.Writer, doc *helium.Document) error {
	// Text output: just write the text content of the document
	sw := stream.NewWriter(w)
	err := helium.Walk(doc, func(n helium.Node) error {
		switch n.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			return sw.WriteRaw(string(n.Content()))
		}
		return nil
	})
	if err != nil {
		return err
	}
	return sw.Flush()
}

func serializeHTML(w io.Writer, doc *helium.Document, outDef *OutputDef) error {
	// For now, use XML serialization with some HTML tweaks
	var opts []helium.WriteOption
	if outDef.Indent {
		opts = append(opts, helium.WithFormat())
	}
	opts = append(opts, helium.WithNoDecl())
	return doc.XML(w, opts...)
}
