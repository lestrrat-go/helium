package helium

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	catalog        CatalogResolver
	maxDepth       int
	errorHandler   ErrorHandler
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

// RecoverOnError controls whether the parser attempts to recover from
// well-formedness errors and returns a partial document.
// libxml2: XML_PARSE_RECOVER
// Default: false
func (p Parser) RecoverOnError(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseRecover)
		return p
	}
	p.cfg.options.Set(parseRecover)
	return p
}

// SubstituteEntities controls whether entity references are replaced
// with their substitution text during parsing.
// libxml2: XML_PARSE_NOENT
// Default: false
func (p Parser) SubstituteEntities(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoEnt)
		return p
	}
	p.cfg.options.Set(parseNoEnt)
	return p
}

// LoadExternalDTD controls whether the parser loads the external DTD subset.
// libxml2: XML_PARSE_DTDLOAD
// Default: false
func (p Parser) LoadExternalDTD(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseDTDLoad)
		return p
	}
	p.cfg.options.Set(parseDTDLoad)
	return p
}

// DefaultDTDAttributes controls whether the parser adds default attributes
// defined in the DTD. When set to true, also enables LoadExternalDTD.
// libxml2: XML_PARSE_DTDATTR
// Default: false
func (p Parser) DefaultDTDAttributes(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseDTDAttr)
		return p
	}
	p.cfg.options.Set(parseDTDAttr)
	p.cfg.options.Set(parseDTDLoad)
	return p
}

// ValidateDTD controls whether the parser validates the document against
// its DTD after parsing. When set to true, also enables LoadExternalDTD.
// libxml2: XML_PARSE_DTDVALID
// Default: false
func (p Parser) ValidateDTD(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseDTDValid)
		return p
	}
	p.cfg.options.Set(parseDTDValid)
	p.cfg.options.Set(parseDTDLoad)
	return p
}

// SuppressErrors controls whether error reports from the parser are
// suppressed. When true, the SAX error callback is not invoked.
// libxml2: XML_PARSE_NOERROR
// Default: false
func (p Parser) SuppressErrors(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoError)
		return p
	}
	p.cfg.options.Set(parseNoError)
	return p
}

// SuppressWarnings controls whether warning reports from the parser are
// suppressed. When true, the SAX warning callback is not invoked.
// libxml2: XML_PARSE_NOWARNING
// Default: false
func (p Parser) SuppressWarnings(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoWarning)
		return p
	}
	p.cfg.options.Set(parseNoWarning)
	return p
}

// PedanticErrors controls whether the parser reports pedantic warnings
// for minor specification violations.
// libxml2: XML_PARSE_PEDANTIC
// Default: false
func (p Parser) PedanticErrors(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parsePedantic)
		return p
	}
	p.cfg.options.Set(parsePedantic)
	return p
}

// StripBlanks controls whether whitespace-only text nodes are removed
// from the resulting DOM tree.
// libxml2: XML_PARSE_NOBLANKS
// Default: false
func (p Parser) StripBlanks(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoBlanks)
		return p
	}
	p.cfg.options.Set(parseNoBlanks)
	return p
}

// ProcessXInclude controls whether XInclude substitution is performed
// during parsing.
// libxml2: XML_PARSE_XINCLUDE
// Default: false
func (p Parser) ProcessXInclude(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseXInclude)
		return p
	}
	p.cfg.options.Set(parseXInclude)
	return p
}

// AllowNetwork controls whether the parser is allowed to fetch resources
// over the network (e.g. external DTDs, entities). When set to false,
// all network access is forbidden.
// libxml2: XML_PARSE_NONET (note: semantics are inverted — libxml2 sets
// this flag to *forbid* network access, whereas AllowNetwork(true)
// *permits* it)
// Default: true (network access is allowed)
func (p Parser) AllowNetwork(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Set(parseNoNet)
		return p
	}
	p.cfg.options.Clear(parseNoNet)
	return p
}

// CleanNamespaces controls whether redundant namespace declarations are
// removed from the resulting DOM tree.
// libxml2: XML_PARSE_NSCLEAN
// Default: false
func (p Parser) CleanNamespaces(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNsClean)
		return p
	}
	p.cfg.options.Set(parseNsClean)
	return p
}

// MergeCDATA controls whether CDATA sections are merged into adjacent
// text nodes instead of being represented as separate CDATA nodes.
// libxml2: XML_PARSE_NOCDATA
// Default: false
func (p Parser) MergeCDATA(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoCDATA)
		return p
	}
	p.cfg.options.Set(parseNoCDATA)
	return p
}

// XIncludeNodes controls whether XINCLUDE START/END marker nodes are
// generated in the DOM tree during XInclude processing.
// libxml2: XML_PARSE_NOXINCNODE (note: semantics are inverted — libxml2
// sets this flag to *suppress* marker nodes, whereas XIncludeNodes(false)
// suppresses them)
// Default: true (marker nodes are generated)
func (p Parser) XIncludeNodes(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Set(parseNoXIncNode)
		return p
	}
	p.cfg.options.Clear(parseNoXIncNode)
	return p
}

// CompactTextNodes controls whether the parser compacts small text nodes
// to reduce memory usage.
// libxml2: XML_PARSE_COMPACT
// Default: false
func (p Parser) CompactTextNodes(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseCompact)
		return p
	}
	p.cfg.options.Set(parseCompact)
	return p
}

// FixBaseURIs controls whether xml:base URIs are fixed up during
// XInclude processing. When set to false, xml:base attributes are
// not adjusted on included content.
// libxml2: XML_PARSE_NOBASEFIX (note: semantics are inverted — libxml2
// sets this flag to *disable* fixup, whereas FixBaseURIs(false)
// disables it)
// Default: true (xml:base URIs are fixed up)
func (p Parser) FixBaseURIs(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Set(parseNoBaseFix)
		return p
	}
	p.cfg.options.Clear(parseNoBaseFix)
	return p
}

// RelaxLimits controls whether hardcoded parser limits (name length,
// entity expansion) are relaxed. Use with caution — disabling limits
// may expose the parser to denial-of-service attacks.
// libxml2: XML_PARSE_HUGE
// Default: false
func (p Parser) RelaxLimits(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseHuge)
		return p
	}
	p.cfg.options.Set(parseHuge)
	return p
}

// IgnoreEncoding controls whether the parser ignores the encoding
// declaration inside the document and uses the transport-level encoding
// instead.
// libxml2: XML_PARSE_IGNORE_ENC
// Default: false
func (p Parser) IgnoreEncoding(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseIgnoreEnc)
		return p
	}
	p.cfg.options.Set(parseIgnoreEnc)
	return p
}

// BigLineNumbers controls whether large line numbers are stored in the
// text PSVI field, allowing line numbers above 65535.
// libxml2: XML_PARSE_BIG_LINES
// Default: false
func (p Parser) BigLineNumbers(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseBigLines)
		return p
	}
	p.cfg.options.Set(parseBigLines)
	return p
}

// BlockXXE controls whether loading of external entities and DTDs is
// blocked, preventing XML External Entity (XXE) attacks.
// libxml2: XML_PARSE_NOXXE
// Default: false
func (p Parser) BlockXXE(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseNoXXE)
		return p
	}
	p.cfg.options.Set(parseNoXXE)
	return p
}

// ReuseDict controls whether the parser reuses the context dictionary
// for interned strings. When set to false, a fresh dictionary is used.
// libxml2: XML_PARSE_NODICT (note: semantics are inverted — libxml2
// sets this flag to *disable* dictionary reuse, whereas ReuseDict(false)
// disables it)
// Default: true (dictionary is reused)
func (p Parser) ReuseDict(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Set(parseNoDict)
		return p
	}
	p.cfg.options.Clear(parseNoDict)
	return p
}

// SkipIDs controls whether ID attribute interning is skipped during
// parsing. When true, the parser does not build the ID table.
// libxml2: XML_PARSE_SKIP_IDS
// Default: false
func (p Parser) SkipIDs(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseSkipIDs)
		return p
	}
	p.cfg.options.Set(parseSkipIDs)
	return p
}

// LenientXMLDecl relaxes XML declaration parsing so that the version,
// encoding, and standalone pseudo-attributes may appear in any order.
// Per the XML spec (section 2.8) the order MUST be version, encoding,
// standalone, but some real-world producers emit them differently.
// This is a helium extension not present in libxml2.
// Default: false
func (p Parser) LenientXMLDecl(v bool) Parser {
	p = p.clone()
	if !v {
		p.cfg.options.Clear(parseLenientXMLDecl)
		return p
	}
	p.cfg.options.Set(parseLenientXMLDecl)
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
func (p Parser) Catalog(c CatalogResolver) Parser {
	p = p.clone()
	p.cfg.catalog = c
	return p
}

// ErrorHandler sets the handler for validation errors produced during
// DTD validation ([ValidateDTD]). When set, individual errors are delivered
// to the handler as they occur. The returned error from Parse is
// [ErrDTDValidationFailed] on failure.
func (p Parser) ErrorHandler(h ErrorHandler) Parser {
	p = p.clone()
	p.cfg.errorHandler = h
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
		handler := p.cfg.errorHandler
		if handler == nil {
			handler = NilErrorHandler{}
		}
		if err := validateDocument(ctx, pctx.doc, handler); err != nil {
			return pctx.doc, err
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
		handler := p.cfg.errorHandler
		if handler == nil {
			handler = NilErrorHandler{}
		}
		if err := validateDocument(ctx, pctx.doc, handler); err != nil {
			return pctx.doc, err
		}
	}

	return pctx.doc, nil
}

// ParseFile reads and parses an XML file. The document's URL is set to the
// absolute path of the file, and the file path is used as the base URI for
// relative URI resolution during parsing.
func (p Parser) ParseFile(ctx context.Context, path string) (*Document, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is caller-supplied
	if err != nil {
		return nil, fmt.Errorf("helium: failed to read %q: %w", path, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("helium: failed to resolve path %q: %w", path, err)
	}
	doc, err := p.BaseURI(abs).Parse(ctx, data)
	if err != nil {
		return nil, err
	}
	doc.SetURL(abs)
	return doc, nil
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
	newRoot := doc.CreateElement("pseudoroot")
	newctx.pushNodeEntry(nodeEntry{local: "pseudoroot", qname: "pseudoroot"})
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
				e.(MutableNode).SetTreeDoc(doc)
				e.baseDocNode().parent = nil
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
