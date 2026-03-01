package helium

import (
	"bytes"
	"errors"
	"io"

	icatalog "github.com/lestrrat-go/helium/internal/catalog"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
)

// ParserStopper is implemented by the parser context passed as sax.Context
// to SAX callbacks. SAX handlers can type-assert the context to this
// interface and call StopParser to abort parsing early without error.
type ParserStopper interface {
	StopParser()
}

// StopParser tells the parser to stop at the next opportunity. Call this
// from any SAX callback to abort parsing early. The parse functions will
// return the partial document built so far with a nil error.
func StopParser(ctx sax.Context) {
	if s, ok := ctx.(ParserStopper); ok {
		s.StopParser()
	}
}

// Parser holds configuration for XML parsing (libxml2: xmlParserCtxt).
type Parser struct {
	sax            sax.SAX2Handler
	charBufferSize int
	options        ParseOption
	baseURI        string
	catalog        icatalog.Resolver
}

// Parse parses XML from a byte slice and returns the resulting Document
// (libxml2: xmlParseDoc / xmlParseMemory).
func Parse(b []byte) (*Document, error) {
	p := NewParser()
	return p.Parse(b)
}

func NewParser() *Parser {
	return &Parser{
		sax: NewTreeBuilder(),
	}
}

func (p *Parser) Parse(b []byte) (*Document, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("=== START Parser.Parse ===")
		defer g.IRelease("=== END Parser.Parse ===")
	}

	ctx := &parserCtx{rawInput: b, baseURI: p.baseURI}
	if err := ctx.init(p, bytes.NewReader(b)); err != nil {
		return nil, err
	}
	defer func() {
		if err := ctx.release(); err != nil {
			// Log error but don't override the main return error
			if pdebug.Enabled {
				pdebug.Printf("ctx.release() failed: %s", err)
			}
		}
	}()

	if err := ctx.parseDocument(); err != nil {
		if errors.Is(err, errParserStopped) {
			return ctx.doc, nil
		}
		if p != nil && p.options.IsSet(ParseRecover) {
			// ParseRecover: return the partial document along with the error
			return ctx.doc, err
		}
		return nil, err
	}

	// DTD validation: run post-parse document validation when requested.
	if p != nil && p.options.IsSet(ParseDTDValid) && ctx.doc != nil {
		if ve := validateDocument(ctx.doc); ve != nil {
			return ctx.doc, ve
		}
	}

	return ctx.doc, nil
}

// ParseReader parses XML from an io.Reader and returns the resulting Document
// (libxml2: xmlReadIO).
// This is identical to Parse but reads from a stream instead of a byte slice.
// EBCDIC encoding detection is not supported when parsing from a reader.
func ParseReader(r io.Reader) (*Document, error) {
	return NewParser().ParseReader(r)
}

// ParseReader parses XML from an io.Reader and returns the resulting Document.
// This is identical to Parse but reads from a stream instead of a byte slice.
// EBCDIC encoding detection is not supported when parsing from a reader.
func (p *Parser) ParseReader(r io.Reader) (*Document, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("=== START Parser.ParseReader ===")
		defer g.IRelease("=== END Parser.ParseReader ===")
	}

	ctx := &parserCtx{baseURI: p.baseURI}
	if err := ctx.init(p, r); err != nil {
		return nil, err
	}
	defer func() {
		if err := ctx.release(); err != nil {
			if pdebug.Enabled {
				pdebug.Printf("ctx.release() failed: %s", err)
			}
		}
	}()

	if err := ctx.parseDocument(); err != nil {
		if errors.Is(err, errParserStopped) {
			return ctx.doc, nil
		}
		if p != nil && p.options.IsSet(ParseRecover) {
			return ctx.doc, err
		}
		return nil, err
	}

	if p != nil && p.options.IsSet(ParseDTDValid) && ctx.doc != nil {
		if ve := validateDocument(ctx.doc); ve != nil {
			return ctx.doc, ve
		}
	}

	return ctx.doc, nil
}

func (p *Parser) SetSAXHandler(s sax.SAX2Handler) {
	p.sax = s
}

func (p *Parser) SetOption(opt ParseOption) {
	p.options.Set(opt)
}

// SetCharBufferSize sets the maximum number of bytes delivered in a single
// Characters or IgnorableWhitespace SAX callback. When size <= 0 (the
// default), all character data is delivered in one call. When size > 0,
// data longer than size bytes is split into chunks of at most size bytes,
// always respecting UTF-8 character boundaries.
func (p *Parser) SetCharBufferSize(size int) {
	p.charBufferSize = size
}

// SetBaseURI sets the document's base URI, used for resolving relative
// references such as external DTD system identifiers.
func (p *Parser) SetBaseURI(uri string) {
	p.baseURI = uri
}

// SetCatalog sets an XML Catalog for resolving external entity identifiers
// (public/system IDs) during parsing. When set, the parser consults the
// catalog before attempting to load external DTDs and entities.
func (p *Parser) SetCatalog(c icatalog.Resolver) {
	p.catalog = c
}

// ParseInNodeContext parses an XML fragment in the context of an existing
// node. The node provides in-scope namespace declarations and document-level
// DTD/entity context. Returns the first node of the parsed fragment list
// (siblings linked via NextSibling). The returned nodes are not attached
// to any parent.
func ParseInNodeContext(node Node, data []byte) (Node, error) {
	return NewParser().ParseInNodeContext(node, data)
}

// ParseInNodeContext parses an XML fragment in the context of an existing
// node. The node provides in-scope namespace declarations and document-level
// DTD/entity context. Returns the first node of the parsed fragment list
// (siblings linked via NextSibling). The returned nodes are not attached
// to any parent.
func (p *Parser) ParseInNodeContext(node Node, data []byte) (Node, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("=== START Parser.ParseInNodeContext ===")
		defer g.IRelease("=== END Parser.ParseInNodeContext ===")
	}

	if node == nil {
		return nil, errors.New("node must not be nil")
	}

	// Walk up to the nearest element or document node.
	var ctxElem *Element
	var doc *Document
	cur := node
	for cur != nil {
		switch v := cur.(type) {
		case *Document:
			doc = v
			goto found
		case *Element:
			ctxElem = v
			doc = v.doc
			goto found
		}
		cur = cur.Parent()
	}
	return nil, errors.New("no element or document context found")

found:
	if doc == nil {
		doc = NewDocument("1.0", "", StandaloneImplicitNo)
	}

	newctx := &parserCtx{}
	if err := newctx.init(p, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	defer func() {
		if err := newctx.release(); err != nil {
			if pdebug.Enabled {
				pdebug.Printf("newctx.release() failed: %s", err)
			}
		}
	}()

	// Save the document's children and restore them afterward.
	fc := doc.FirstChild()
	lc := doc.LastChild()
	setFirstChild(doc, nil)
	setLastChild(doc, nil)
	defer func() {
		setFirstChild(doc, fc)
		setLastChild(doc, lc)
	}()

	newctx.doc = doc

	// Push in-scope namespaces from the context element into the parser's
	// namespace stack so that the fragment can resolve prefixed names.
	if ctxElem != nil {
		nsList := collectInScopeNamespaces(ctxElem)
		for _, ns := range nsList {
			newctx.pushNS(ns.Prefix(), ns.URI())
		}
	}

	// Create pseudoroot element, push to node stack.
	newRoot, err := doc.CreateElement("pseudoroot")
	if err != nil {
		return nil, err
	}
	newctx.pushNode(newRoot)
	newctx.elem = newRoot
	if err := doc.AddChild(newRoot); err != nil {
		return nil, err
	}

	if err := newctx.switchEncoding(); err != nil {
		return nil, err
	}
	if err := newctx.parseContent(); err != nil {
		return nil, err
	}

	// Extract children from pseudoroot.
	if child := doc.FirstChild(); child != nil {
		if grandchild := child.FirstChild(); grandchild != nil {
			for e := grandchild; e != nil; e = e.NextSibling() {
				e.SetTreeDoc(doc)
				e.SetParent(nil)
			}
			return grandchild, nil
		}
	}

	return nil, nil
}

// collectInScopeNamespaces walks up from elem collecting all namespace
// declarations. Inner declarations shadow outer ones (closer to elem wins).
func collectInScopeNamespaces(elem *Element) []*Namespace {
	seen := map[string]bool{}
	var result []*Namespace
	var cur Node = elem
	for cur != nil {
		if e, ok := cur.(*Element); ok {
			for _, ns := range e.Namespaces() {
				if !seen[ns.Prefix()] {
					seen[ns.Prefix()] = true
					result = append(result, ns)
				}
			}
		}
		cur = cur.Parent()
	}
	return result
}
