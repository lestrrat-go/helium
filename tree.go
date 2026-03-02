package helium

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
	"github.com/lestrrat-go/strcursor"
)

// BuildURI resolves a relative system ID against a base URI.
// For local file paths (no scheme or file: scheme), it uses filepath.Join.
// For other schemes, it uses url.ResolveReference.
func BuildURI(systemID, base string) string {
	u, err := url.Parse(systemID)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		return systemID
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	if baseURL.Scheme == "" || baseURL.Scheme == "file" {
		basePath := baseURL.Path
		if basePath == "" {
			basePath = base
		}
		// When the base ends with "/" it represents a directory;
		// use it directly instead of calling filepath.Dir which
		// would strip the last component.
		dir := filepath.Dir(basePath)
		if strings.HasSuffix(basePath, "/") {
			dir = strings.TrimRight(basePath, "/")
		}
		result := filepath.Join(dir, systemID)
		// Preserve trailing slash from systemID (indicates a directory
		// base for xml:base chaining).
		if strings.HasSuffix(systemID, "/") && !strings.HasSuffix(result, "/") {
			result += "/"
		}
		return result
	}

	return baseURL.ResolveReference(u).String()
}

// fileParseInput wraps an os.File as a sax.ParseInput.
type fileParseInput struct {
	io.ReadCloser
	uri string
}

func (f *fileParseInput) URI() string { return f.uri }

// TreeBuilder is a SAX2 handler that builds a DOM tree from SAX events,
// analogous to libxml2's default SAX handler (xmlSAX2InitDefaultSAXHandler).
type TreeBuilder struct {
}

// NewTreeBuilder creates a new TreeBuilder that builds a DOM tree from SAX events.
func NewTreeBuilder() *TreeBuilder {
	return &TreeBuilder{}
}

func (t *TreeBuilder) SetDocumentLocator(ctxif sax.Context, loc sax.DocumentLocator) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.SetDocumentLocator")
		defer g.IRelease("END tree.SetDocumentLocator")
	}

	return nil
}

func (t *TreeBuilder) StartDocument(ctxif sax.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.StartDocument")
		defer g.IRelease("END tree.StartDocument")
	}

	ctx := ctxif.(*parserCtx)
	ctx.doc = NewDocument(ctx.version, ctx.encoding, ctx.standalone)
	ctx.doc.ids = make(map[string]*Element)
	ctx.doc.url = ctx.baseURI
	return nil
}

func (t *TreeBuilder) EndDocument(ctxif sax.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.EndDocument")
		defer g.IRelease("END tree.EndDocument")
	}

	ctx := ctxif.(*parserCtx)
	if ctx.doc != nil && ctx.wellFormed {
		ctx.doc.properties |= DocWellFormed
		if ctx.valid {
			ctx.doc.properties |= DocDTDValid
		}
	}
	return nil
}

func (t *TreeBuilder) ProcessingInstruction(ctxif sax.Context, target, data string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.ProcessingInstruction")
		defer g.IRelease("END tree.ProcessingInstruction")
	}
	ctx := ctxif.(*parserCtx)
	doc := ctx.doc
	pi, err := doc.CreatePI(target, data)
	if err != nil {
		return err
	}

	switch ctx.inSubset {
	case 1:
		if err := doc.IntSubset().AddChild(pi); err != nil {
			return err
		}
	case 2:
		if err := doc.ExtSubset().AddChild(pi); err != nil {
			return err
		}
	}

	parent := ctx.elem
	if parent == nil {
		if err := doc.AddChild(pi); err != nil {
			return err
		}
	} else if parent.Type() == ElementNode {
		if err := parent.AddChild(pi); err != nil {
			return err
		}
	} else {
		if err := parent.AddSibling(pi); err != nil {
			return err
		}
	}
	return nil
}

// LookupNSByPrefix walks the element and its ancestors to find a namespace
// declaration matching the given prefix. The "xml" prefix is always
// implicitly bound to the XML namespace.
func LookupNSByPrefix(e *Element, prefix string) *Namespace {
	var node Node = e
	for node != nil {
		if el, ok := node.(*Element); ok {
			for _, ns := range el.Namespaces() {
				if ns.Prefix() == prefix {
					return ns
				}
			}
		}
		node = node.Parent()
	}
	if prefix == "xml" {
		return NewNamespace("xml", XMLNamespace)
	}
	return nil
}

// lookupNSByPrefix is the unexported alias for internal callers.
func lookupNSByPrefix(e *Element, prefix string) *Namespace {
	return LookupNSByPrefix(e, prefix)
}

// LookupNSByHref walks the element and its ancestors to find a namespace
// declaration matching the given URI.
func LookupNSByHref(e *Element, href string) *Namespace {
	if href == XMLNamespace {
		return NewNamespace("xml", XMLNamespace)
	}
	var node Node = e
	for node != nil {
		if el, ok := node.(*Element); ok {
			for _, ns := range el.Namespaces() {
				if ns.URI() == href {
					return ns
				}
			}
		}
		node = node.Parent()
	}
	return nil
}

// lookupAttributeDecl looks up an attribute declaration in the document's
// internal and external DTD subsets.
func lookupAttributeDecl(doc *Document, name, prefix, elem string) *AttributeDecl {
	if doc == nil {
		return nil
	}
	if dtd := doc.IntSubset(); dtd != nil {
		if decl, ok := dtd.LookupAttribute(name, prefix, elem); ok {
			return decl
		}
	}
	if dtd := doc.ExtSubset(); dtd != nil {
		if decl, ok := dtd.LookupAttribute(name, prefix, elem); ok {
			return decl
		}
	}
	return nil
}

func (t *TreeBuilder) StartElementNS(ctxif sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
	//	ctx := ctxif.(*parserCtx)
	if pdebug.Enabled {
		var name string
		if prefix != "" {
			name = prefix + ":" + localname
		} else {
			name = localname
		}
		g := pdebug.IPrintf("START tree.StartElement: %s", name)
		defer g.IRelease("END tree.StartElement")
	}

	ctx := ctxif.(*parserCtx)
	doc := ctx.doc

	e, err := doc.CreateElement(localname)
	if err != nil {
		return err
	}

	e.SetLine(ctx.LineNumber())

	if uri != "" {
		if err := e.SetActiveNamespace(prefix, uri); err != nil {
			return err
		}
	}

	for _, ns := range namespaces {
		if err := e.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
			return err
		}
	}

	pdebug.Printf("We got %d attributes", len(attrs))
	for _, attr := range attrs {
		if attr.IsDefault() && !ctx.loadsubset.IsSet(CompleteAttrs) {
			if pdebug.Enabled {
				pdebug.Printf("Skipping default attribute %s", attr.Name())
			}
			continue
		}
		if p := attr.Prefix(); p != "" {
			// Prefixed attribute — look up the namespace from the
			// element itself or from the parent context (the element
			// hasn't been linked to the tree yet).
			ns := lookupNSByPrefix(e, p)
			if ns == nil && ctx.elem != nil {
				ns = lookupNSByPrefix(ctx.elem, p)
			}
			if err := e.SetAttributeNS(attr.LocalName(), attr.Value(), ns); err != nil {
				return err
			}
		} else {
			if err := e.SetAttribute(attr.Name(), attr.Value()); err != nil {
				return err
			}
		}
	}

	// Propagate attribute types from DTD declarations.
	elemName := localname
	if prefix != "" {
		elemName = prefix + ":" + localname
	}
	for _, a := range e.Attributes() {
		aLocalName := a.LocalName()
		aPrefix := a.Prefix()
		if decl := lookupAttributeDecl(doc, aLocalName, aPrefix, elemName); decl != nil {
			a.SetAType(decl.AType())
		}
	}

	// Register ID attributes in the document's ID table for O(1) lookup.
	if !ctx.loadsubset.IsSet(SkipIDs) {
		for _, a := range e.Attributes() {
			if a.Name() == "xml:id" || a.AType() == AttrID {
				doc.RegisterID(a.Value(), e)
			}
		}
	}

	var parent Node
	if e := ctx.elem; e != nil {
		parent = e
	}
	if parent == nil {
		if err := doc.AddChild(e); err != nil {
			return err
		}
	} else if parent.Type() == ElementNode {
		if err := parent.AddChild(e); err != nil {
			return err
		}
	} else {
		if err := parent.AddSibling(e); err != nil {
			return err
		}
	}

	ctx.elem = e

	return nil
}

func (t *TreeBuilder) EndElementNS(ctxif sax.Context, localname, prefix, uri string) error {
	if pdebug.Enabled {
		if prefix != "" {
			pdebug.Printf("tree.EndElement: %s:%s", prefix, localname)
		} else {
			pdebug.Printf("tree.EndElement: %s", localname)
		}
	}

	ctx := ctxif.(*parserCtx)
	cur := ctx.elem
	if cur == nil {
		return errors.New("no context node to end")
	}

	p := cur.Parent()
	if e, ok := p.(*Element); ok {
		ctx.elem = e
	} else {
		ctx.elem = nil
	}
	return nil
}

func (t *TreeBuilder) Characters(ctxif sax.Context, data []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.Characters: '%s' (%v)", data, data)
		defer g.IRelease("END tree.Characters")
	}

	ctx := ctxif.(*parserCtx)
	n := ctx.elem
	if n == nil {
		return errors.New("text content placed in wrong location")
	}

	if pdebug.Enabled {
		pdebug.Printf("Calling AppendText() on '%s' node", n.Name())
	}

	return n.AppendText(data)
}

// CDataBlock mirrors xmlSAX2Text(ctxt, value, len, XML_CDATA_SECTION_NODE)
// in libxml2's SAX2.c. Unlike text nodes, adjacent CDATA sections are NOT
// merged — each callback creates a new CDATASection node.
func (t *TreeBuilder) CDataBlock(ctxif sax.Context, data []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.CDATABlock")
		defer g.IRelease("END tree.CDATABlock")
	}

	ctx := ctxif.(*parserCtx)
	parent := ctx.elem
	if parent == nil {
		return nil
	}

	doc := ctx.doc
	cdata, err := doc.CreateCDATASection(data)
	if err != nil {
		return err
	}

	return parent.AddChild(cdata)
}

// Comment mirrors xmlSAX2Comment in libxml2's SAX2.c, which delegates
// parent selection to xmlSAX2AppendChild. When inside a DTD subset the
// comment is added to the DTD, not the document.
func (t *TreeBuilder) Comment(ctxif sax.Context, data []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.Comment: %s", data)
		defer g.IRelease("END tree.Comment")
	}

	ctx := ctxif.(*parserCtx)
	doc := ctx.doc
	if doc == nil {
		return errors.New("comment placed in wrong location")
	}

	e, err := doc.CreateComment(data)
	if err != nil {
		return err
	}

	// Mirror xmlSAX2AppendChild parent selection (SAX2.c:899-907).
	switch ctx.inSubset {
	case inInternalSubset:
		return doc.IntSubset().AddChild(e)
	case inExternalSubset:
		return doc.ExtSubset().AddChild(e)
	}

	n := ctx.elem
	if n == nil {
		if err := doc.AddChild(e); err != nil {
			return err
		}
	} else if n.Type() == ElementNode {
		if err := n.AddChild(e); err != nil {
			return err
		}
	} else {
		if err := n.AddSibling(e); err != nil {
			return err
		}
	}
	return nil
}

func (t *TreeBuilder) InternalSubset(ctxif sax.Context, name, eid, uri string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.InternalSubset %s,%s,%s", name, eid, uri)
		defer g.IRelease("END tree.InternalSubset")
	}

	ctx := ctxif.(*parserCtx)
	doc := ctx.doc

	dtd, err := doc.InternalSubset()
	if err == nil {
		// HTML mode would skip freeing the DTD here.
		dtd.Free()
		doc.intSubset = nil // hmm, do we need this?
	}

	_, err = doc.CreateInternalSubset(name, eid, uri)
	if err != nil {
		return err
	}

	return nil
}

func (t *TreeBuilder) ExternalSubset(ctxif sax.Context, name, eid, uri string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.ExternalSubset %s,%s,%s", name, eid, uri)
		defer g.IRelease("END tree.ExternalSubset")
	}

	ctx := ctxif.(*parserCtx)

	if ctx.options.IsSet(ParseNoXXE) {
		return nil
	}

	if !ctx.loadsubset.IsSet(DetectIDs) {
		return nil
	}

	// Try catalog resolution first.
	if ctx.catalog != nil {
		if catalogURI := ctx.catalog.Resolve(eid, uri); catalogURI != "" {
			uri = catalogURI
		}
	}

	if uri == "" {
		return nil
	}

	// Resolve system URI against document's base URI
	resolved := uri
	if !filepath.IsAbs(uri) && ctx.baseURI != "" {
		resolved = filepath.Join(filepath.Dir(ctx.baseURI), uri)
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		// Silently ignore missing external DTDs
		return nil
	}

	doc := ctx.doc

	// Create the external subset DTD
	dtd := newDTD()
	dtd.name = name
	dtd.externalID = eid
	dtd.systemID = uri
	dtd.doc = doc
	doc.extSubset = dtd

	// Parse markup declarations from the DTD content.
	// Push content onto the input stack and loop until exhausted.
	savedExternal := ctx.external
	ctx.external = true

	baseLen := ctx.inputTab.Len()
	ctx.pushInput(strcursor.NewByteCursor(bytes.NewReader(data)))

	for ctx.inputTab.Len() > baseLen {
		top, ok := ctx.inputTab.PeekOne().(strcursor.Cursor)
		if !ok || top.Done() {
			break
		}

		ctx.skipBlanks()

		if ctx.inputTab.Len() <= baseLen {
			break
		}
		top, ok = ctx.inputTab.PeekOne().(strcursor.Cursor)
		if !ok || top.Done() {
			break
		}

		cur := ctx.getCursor()
		if cur != nil && cur.Peek() == '<' && cur.PeekN(2) == '!' && cur.PeekN(3) == '[' {
			if err := ctx.parseConditionalSections(); err != nil {
				break
			}
			continue
		}

		if err := ctx.parseMarkupDecl(); err != nil {
			break
		}
	}

	// Clean up: ensure our pushed input is removed
	for ctx.inputTab.Len() > baseLen {
		ctx.popInput()
	}

	ctx.external = savedExternal

	return nil
}

func (t *TreeBuilder) HasInternalSubset(ctxif sax.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) HasExternalSubset(ctxif sax.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) IsStandalone(ctxif sax.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) GetEntity(ctxif sax.Context, name string) (ent sax.Entity, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.GetEntity '%s'", name)
		defer func() {
			g.IRelease("END tree.GetEntity = '%v'", ent)
		}()
	}

	ctx := ctxif.(*parserCtx)
	doc := ctx.doc
	x, ok := doc.GetEntity(name)
	if !ok {
		err = errors.New("entity not found")
	} else {
		ent = x
	}
	return
}

func (t *TreeBuilder) GetParameterEntity(ctxif sax.Context, name string) (sax.Entity, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.GetParameterEntity '%s'", name)
		defer g.IRelease("END tree.GetParameterEntity")
	}

	if ctxif == nil {
		return nil, ErrInvalidParserCtx
	}

	ctx := ctxif.(*parserCtx)
	doc := ctx.doc
	if doc == nil {
		return nil, ErrInvalidDocument
	}

	if ret, ok := doc.GetParameterEntity(name); ok {
		return ret, nil
	}

	return nil, ErrEntityNotFound
}

func (t *TreeBuilder) AttributeDecl(ctxif sax.Context, eName string, aName string, typ int, deftype int, value string, enumif sax.Enumeration) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.AttributeDecl name = '%s', elem = '%s'", aName, eName)
		defer g.IRelease("END tree.AttributeDecl")
	}

	ctx := ctxif.(*parserCtx)

	if aName == "xml:id" && typ != int(AttrID) {
		// libxml2 says "raise the error but keep the validity flag"
		// but I don't know if we can do that..
		return errors.New("xml:id: attribute type should be AttrID")
	}
	var prefix string
	var local string
	if i := strings.IndexByte(aName, ':'); i > -1 {
		prefix = aName[:i]
		local = aName[i+1:]
	} else {
		local = aName
	}

	enum := enumif.(Enumeration)

	doc := ctx.doc
	switch ctx.inSubset {
	case 1:
		if pdebug.Enabled {
			pdebug.Printf("Processing intSubset...")
		}
		if _, err := ctx.addAttributeDecl(doc.intSubset, eName, local, prefix, AttributeType(typ), AttributeDefault(deftype), value, enum); err != nil {
			return err
		}
	case 2:
		if pdebug.Enabled {
			pdebug.Printf("Processing extSubset...")
		}
		if _, err := ctx.addAttributeDecl(doc.extSubset, eName, local, prefix, AttributeType(typ), AttributeDefault(deftype), value, enum); err != nil {
			return err
		}
	default:
		if pdebug.Enabled {
			pdebug.Printf("uh-oh we have a problem inSubset = %d", ctx.inSubset)
		}
		return errors.New("TreeBuilder.AttributeDecl called while not in subset")
	}
	// NOTE: Attribute declaration validation (xmlValidateAttributeDecl in
	// libxml2) is now handled post-parse via validateDocument() when
	// ParseDTDValid is set.
	return nil
}

func (t *TreeBuilder) ElementDecl(ctxif sax.Context, name string, typ int, content sax.ElementContent) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.ElementDecl")
		defer g.IRelease("END tree.ElementDecl")
	}

	ctx := ctxif.(*parserCtx)
	doc := ctx.doc
	var dtd *DTD
	switch ctx.inSubset {
	case 1:
		dtd = doc.intSubset
	case 2:
		dtd = doc.extSubset
	default:
		return errors.New("sax.ElementDecl called while not in subset")
	}

	_, err := dtd.AddElementDecl(name, ElementTypeVal(typ), content.(*ElementContent))
	if err != nil {
		return err
	}

	return nil
}

func (t *TreeBuilder) EndDTD(ctxif sax.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.EndDTD")
		defer g.IRelease("END tree.EndDTD")
	}

	return nil
}

func (t *TreeBuilder) EndEntity(ctxif sax.Context, name string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.EndEntity")
		defer g.IRelease("END tree.EndEntity")
	}

	return nil
}
func (t *TreeBuilder) ExternalEntityDecl(ctxif sax.Context, name string, publicID string, systemID string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.ExternalEntityDecl")
		defer g.IRelease("END tree.ExternalEntityDecl")
	}

	return nil
}

func (t *TreeBuilder) GetExternalSubset(ctxif sax.Context, name string, baseURI string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.GetExternalSubset")
		defer g.IRelease("END tree.GetExternalSubset")
	}

	return nil
}

func (t *TreeBuilder) IgnorableWhitespace(ctxif sax.Context, content []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.IgnorableWhitespace (%v)", content)
		defer g.IRelease("END tree.IgnorableWhitespace")
	}

	ctx := ctxif.(*parserCtx)
	if ctx.keepBlanks {
		return t.Characters(ctx, content)
	}

	return nil
}

func (t *TreeBuilder) InternalEntityDecl(ctxif sax.Context, name string, value string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.InternalEntityDecl")
		defer g.IRelease("END tree.InternalEntityDecl")
	}

	return nil
}

func (t *TreeBuilder) NotationDecl(ctxif sax.Context, name string, publicID string, systemID string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.NotationDecl")
		defer g.IRelease("END tree.NotationDecl")
	}

	ctx := ctxif.(*parserCtx)
	dtd := ctx.doc.intSubset
	if dtd == nil {
		return nil
	}
	_, err := dtd.AddNotation(name, publicID, systemID)
	return err
}

func (t *TreeBuilder) Reference(ctxif sax.Context, name string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.Reference '%s'", name)
		defer g.IRelease("END tree.Reference")
	}

	ctx := ctxif.(*parserCtx)
	doc := ctx.doc
	var n Node
	var err error
	if name[0] == '#' {
		if n, err = doc.CreateCharRef(name); err != nil {
			return err
		}
	} else {
		if n, err = doc.CreateReference(name); err != nil {
			return err
		}
	}

	parent := ctx.elem
	return parent.AddChild(n)
}

func (t *TreeBuilder) ResolveEntity(ctxif sax.Context, publicID string, systemID string) (sax.ParseInput, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.ResolveEntity '%s' '%s'", publicID, systemID)
		defer g.IRelease("END tree.ResolveEntity")
	}

	ctx := ctxif.(*parserCtx)
	if ctx.catalog != nil {
		if resolved := ctx.catalog.Resolve(publicID, systemID); resolved != "" {
			f, err := os.Open(resolved)
			if err == nil {
				return &fileParseInput{ReadCloser: f, uri: resolved}, nil
			}
		}
	}

	// Fall back to direct file-based resolution. The systemID at this point
	// is the entity's resolved URI (built from system ID + base URI in
	// EntityDecl). Try opening it as a file path.
	if systemID != "" {
		f, err := os.Open(systemID)
		if err == nil {
			return &fileParseInput{ReadCloser: f, uri: systemID}, nil
		}
	}

	return nil, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) SkippedEntity(ctxif sax.Context, name string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.SkippedEntity '%s'", name)
		defer g.IRelease("END tree.SkippedEntity")
	}

	return nil
}

func (t *TreeBuilder) StartDTD(ctxif sax.Context, name string, publicID string, systemID string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.StartDTD")
		defer g.IRelease("END tree.StartDTD")
	}

	return nil
}

func (t *TreeBuilder) StartEntity(ctxif sax.Context, name string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.StartEntity")
		defer g.IRelease("END tree.StartEntity")
	}

	return nil
}

func (t *TreeBuilder) EntityDecl(ctxif sax.Context, name string, typ int, publicID string, systemID string, notation string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.EntityDecl '%s' -> '%s'", name, notation)
		defer g.IRelease("END tree.EntityDecl")
	}

	ctx := ctxif.(*parserCtx)
	doc := ctx.doc
	var dtd *DTD
	switch ctx.inSubset {
	case 1:
		dtd = doc.intSubset
	case 2:
		dtd = doc.extSubset
	default:
		return errors.New("sax.EntityDecl called while note in subset")
	}

	ent, err := dtd.AddEntity(name, EntityType(typ), publicID, systemID, notation)
	if err != nil {
		return err
	}

	// Build the full URI for external entities by resolving the system ID
	// against the document's base URI (mirrors libxml2's xmlSAX2EntityDecl).
	if ent.uri == "" && systemID != "" {
		base := ctx.baseURI
		if base != "" {
			resolved := BuildURI(systemID, base)
			if resolved != "" {
				ent.uri = resolved
			}
		}
		if ent.uri == "" {
			ent.uri = systemID
		}
	}

	return nil
}

func (t *TreeBuilder) UnparsedEntityDecl(ctxif sax.Context, name string, publicID string, systemID string, notation string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.UnparsedEntityDecl '%s'", name)
		defer g.IRelease("END tree.UnparsedEntityDecl")
	}

	// Mirror xmlSAX2UnparsedEntityDecl: register the NDATA entity in the DTD.
	ctx := ctxif.(*parserCtx)
	doc := ctx.doc
	var dtd *DTD
	switch ctx.inSubset {
	case 1:
		dtd = doc.intSubset
	case 2:
		dtd = doc.extSubset
	default:
		return errors.New("sax.UnparsedEntityDecl called while not in subset")
	}

	_, _ = dtd.AddEntity(name, ExternalGeneralUnparsedEntity, publicID, systemID, notation)
	return nil
}

func (t *TreeBuilder) Error(ctxif sax.Context, err error) error {
	return nil
}

func (t *TreeBuilder) Warning(ctxif sax.Context, err error) error {
	return nil
}
