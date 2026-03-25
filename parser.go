package helium

import (
	"bytes"
	"context"
	"errors"
	"io"

	icatalog "github.com/lestrrat-go/helium/internal/catalog"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
)

type stopFuncKey struct{}

// StopParser tells the parser to stop at the next opportunity. Call this
// from any SAX callback to abort parsing early. The parse functions will
// return the partial document built so far with a nil error.
func StopParser(ctx context.Context) {
	if ctx == nil {
		return
	}
	if fn, _ := ctx.Value(stopFuncKey{}).(func()); fn != nil {
		fn()
	}
}

// parserConfig holds the mutable configuration behind a Parser.
type parserConfig struct {
	sax            sax.SAX2Handler
	charBufferSize int
	options        parseOption
	baseURI        string
	catalog        icatalog.Resolver
	maxDepth       int
}

// Parser holds configuration for XML parsing (libxml2: xmlParserCtxt).
// It uses clone-on-write semantics: each builder method returns
// a new Parser sharing the underlying config until mutation.
type Parser struct {
	cfg *parserConfig
}

// NewParser creates a new Parser with default settings.
func NewParser() Parser {
	return Parser{cfg: &parserConfig{
		sax: NewTreeBuilder(),
	}}
}

func (p Parser) clone() Parser {
	cp := *p.cfg
	return Parser{cfg: &cp}
}

// --- Flag methods (each sets/clears the corresponding bit) ---

// Recover enables recovery on parse errors (libxml2: XML_PARSE_RECOVER).
func (p Parser) Recover(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseRecover)
	} else {
		p.cfg.options.Clear(parseRecover)
	}
	return p
}

// NoEnt enables entity substitution (libxml2: XML_PARSE_NOENT).
func (p Parser) NoEnt(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseNoEnt)
	} else {
		p.cfg.options.Clear(parseNoEnt)
	}
	return p
}

// DTDLoad enables loading the external DTD subset (libxml2: XML_PARSE_DTDLOAD).
func (p Parser) DTDLoad(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseDTDLoad)
	} else {
		p.cfg.options.Clear(parseDTDLoad)
	}
	return p
}

// DTDAttr enables defaulting DTD attributes (libxml2: XML_PARSE_DTDATTR).
// Also implies DTDLoad.
func (p Parser) DTDAttr(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseDTDAttr)
		p.cfg.options.Set(parseDTDLoad)
	} else {
		p.cfg.options.Clear(parseDTDAttr)
	}
	return p
}

// DTDValid enables DTD validation (libxml2: XML_PARSE_DTDVALID).
// Also implies DTDLoad.
func (p Parser) DTDValid(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseDTDValid)
		p.cfg.options.Set(parseDTDLoad)
	} else {
		p.cfg.options.Clear(parseDTDValid)
	}
	return p
}

// NoError suppresses error reports (libxml2: XML_PARSE_NOERROR).
func (p Parser) NoError(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseNoError)
	} else {
		p.cfg.options.Clear(parseNoError)
	}
	return p
}

// NoWarning suppresses warning reports (libxml2: XML_PARSE_NOWARNING).
func (p Parser) NoWarning(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseNoWarning)
	} else {
		p.cfg.options.Clear(parseNoWarning)
	}
	return p
}

// Pedantic enables pedantic error reporting (libxml2: XML_PARSE_PEDANTIC).
func (p Parser) Pedantic(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parsePedantic)
	} else {
		p.cfg.options.Clear(parsePedantic)
	}
	return p
}

// NoBlanks removes blank nodes (libxml2: XML_PARSE_NOBLANKS).
func (p Parser) NoBlanks(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseNoBlanks)
	} else {
		p.cfg.options.Clear(parseNoBlanks)
	}
	return p
}

// XInclude enables XInclude substitution (libxml2: XML_PARSE_XINCLUDE).
func (p Parser) XInclude(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseXInclude)
	} else {
		p.cfg.options.Clear(parseXInclude)
	}
	return p
}

// NoNet forbids network access (libxml2: XML_PARSE_NONET).
func (p Parser) NoNet(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseNoNet)
	} else {
		p.cfg.options.Clear(parseNoNet)
	}
	return p
}

// NsClean removes redundant namespace declarations (libxml2: XML_PARSE_NSCLEAN).
func (p Parser) NsClean(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseNsClean)
	} else {
		p.cfg.options.Clear(parseNsClean)
	}
	return p
}

// NoCDATA merges CDATA as text nodes (libxml2: XML_PARSE_NOCDATA).
func (p Parser) NoCDATA(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseNoCDATA)
	} else {
		p.cfg.options.Clear(parseNoCDATA)
	}
	return p
}

// NoXIncNode suppresses XINCLUDE START/END nodes (libxml2: XML_PARSE_NOXINCNODE).
func (p Parser) NoXIncNode(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseNoXIncNode)
	} else {
		p.cfg.options.Clear(parseNoXIncNode)
	}
	return p
}

// NoBaseFix disables xml:base fixup for XInclude (libxml2: XML_PARSE_NOBASEFIX).
func (p Parser) NoBaseFix(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseNoBaseFix)
	} else {
		p.cfg.options.Clear(parseNoBaseFix)
	}
	return p
}

// Huge relaxes hardcoded parser limits (libxml2: XML_PARSE_HUGE).
func (p Parser) Huge(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseHuge)
	} else {
		p.cfg.options.Clear(parseHuge)
	}
	return p
}

// IgnoreEnc ignores internal document encoding hint (libxml2: XML_PARSE_IGNORE_ENC).
func (p Parser) IgnoreEnc(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseIgnoreEnc)
	} else {
		p.cfg.options.Clear(parseIgnoreEnc)
	}
	return p
}

// NoXXE blocks external entity/DTD loading (libxml2: XML_PARSE_NOXXE).
func (p Parser) NoXXE(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseNoXXE)
	} else {
		p.cfg.options.Clear(parseNoXXE)
	}
	return p
}

// SkipIDs skips ID attribute interning (libxml2: XML_PARSE_SKIP_IDS).
func (p Parser) SkipIDs(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseSkipIDs)
	} else {
		p.cfg.options.Clear(parseSkipIDs)
	}
	return p
}

// LenientXMLDecl relaxes XML declaration parsing so pseudo-attributes
// may appear in any order (helium extension).
func (p Parser) LenientXMLDecl(v bool) Parser {
	p = p.clone()
	if v {
		p.cfg.options.Set(parseLenientXMLDecl)
	} else {
		p.cfg.options.Clear(parseLenientXMLDecl)
	}
	return p
}

// --- Non-flag configuration ---

// SAXHandler sets the SAX2 event handler for parsing.
func (p Parser) SAXHandler(s sax.SAX2Handler) Parser {
	p = p.clone()
	p.cfg.sax = s
	return p
}

// BaseURI sets the document's base URI, used for resolving relative
// references such as external DTD system identifiers.
func (p Parser) BaseURI(uri string) Parser {
	p = p.clone()
	p.cfg.baseURI = uri
	return p
}

// CharBufferSize sets the maximum number of bytes delivered in a single
// Characters or IgnorableWhitespace SAX callback. When size <= 0 (the
// default), all character data is delivered in one call. When size > 0,
// data longer than size bytes is split into chunks of at most size bytes,
// always respecting UTF-8 character boundaries.
func (p Parser) CharBufferSize(size int) Parser {
	p = p.clone()
	p.cfg.charBufferSize = size
	return p
}

// MaxDepth sets the maximum element nesting depth allowed during parsing.
// When depth is greater than zero, the parser returns an error if the input
// document contains elements nested deeper than this limit. A value of zero
// (the default) means no limit is enforced.
func (p Parser) MaxDepth(depth int) Parser {
	p = p.clone()
	p.cfg.maxDepth = depth
	return p
}

// Catalog sets an XML Catalog for resolving external entity identifiers
// (public/system IDs) during parsing. When set, the parser consults the
// catalog before attempting to load external DTDs and entities.
func (p Parser) Catalog(c icatalog.Resolver) Parser {
	p = p.clone()
	p.cfg.catalog = c
	return p
}

// --- Terminal methods ---

// Parse parses XML from a byte slice and returns the resulting Document
// (libxml2: xmlParseDoc / xmlParseMemory).
func (p Parser) Parse(ctx context.Context, b []byte) (*Document, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if pdebug.Enabled {
		g := pdebug.IPrintf("=== START Parser.Parse ===")
		defer g.IRelease("=== END Parser.Parse ===")
	}

	pctx := &parserCtx{rawInput: b, baseURI: p.cfg.baseURI}
	if err := pctx.init(p.cfg, bytes.NewReader(b)); err != nil {
		return nil, err
	}
	defer func() {
		if err := pctx.release(); err != nil {
			// Log error but don't override the main return error
			if pdebug.Enabled {
				pdebug.Printf("ctx.release() failed: %s", err)
			}
		}
	}()

	if err := pctx.parseDocument(ctx); err != nil {
		if errors.Is(err, errParserStopped) {
			return pctx.doc, nil
		}
		if p.cfg.options.IsSet(parseRecover) {
			// ParseRecover: return the partial document along with the error
			return pctx.doc, err
		}
		return nil, err
	}

	// DTD validation: run post-parse document validation when requested.
	if p.cfg.options.IsSet(parseDTDValid) && pctx.doc != nil {
		if ve := validateDocument(pctx.doc); ve != nil {
			return pctx.doc, ve
		}
	}

	return pctx.doc, nil
}

// ParseReader parses XML from an io.Reader and returns the resulting Document
// (libxml2: xmlReadIO).
// This is identical to Parse but reads from a stream instead of a byte slice.
// EBCDIC encoding detection is not supported when parsing from a reader.
func (p Parser) ParseReader(ctx context.Context, r io.Reader) (*Document, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if pdebug.Enabled {
		g := pdebug.IPrintf("=== START Parser.ParseReader ===")
		defer g.IRelease("=== END Parser.ParseReader ===")
	}

	pctx := &parserCtx{baseURI: p.cfg.baseURI}
	if err := pctx.init(p.cfg, r); err != nil {
		return nil, err
	}
	defer func() {
		if err := pctx.release(); err != nil {
			if pdebug.Enabled {
				pdebug.Printf("ctx.release() failed: %s", err)
			}
		}
	}()

	if err := pctx.parseDocument(ctx); err != nil {
		if errors.Is(err, errParserStopped) {
			return pctx.doc, nil
		}
		if p.cfg.options.IsSet(parseRecover) {
			return pctx.doc, err
		}
		return nil, err
	}

	if p.cfg.options.IsSet(parseDTDValid) && pctx.doc != nil {
		if ve := validateDocument(pctx.doc); ve != nil {
			return pctx.doc, ve
		}
	}

	return pctx.doc, nil
}

// ParseInNodeContext parses an XML fragment in the context of an existing
// node. The node provides in-scope namespace declarations and document-level
// DTD/entity context. Returns the first node of the parsed fragment list
// (siblings linked via NextSibling). The returned nodes are not attached
// to any parent.
func (p Parser) ParseInNodeContext(ctx context.Context, node Node, data []byte) (Node, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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
	if err := newctx.init(p.cfg, bytes.NewReader(data)); err != nil {
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
	innerCtx := withParserCtx(ctx, newctx)
	innerCtx = sax.WithDocumentLocator(innerCtx, newctx)
	innerCtx = context.WithValue(innerCtx, stopFuncKey{}, newctx.stop)
	if err := newctx.parseContent(innerCtx); err != nil {
		if !errors.Is(err, errParserStopped) {
			return nil, err
		}
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
