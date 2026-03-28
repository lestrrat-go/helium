package helium

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
)

// fileParseInput wraps an os.File as a sax.ParseInput.
type fileParseInput struct {
	io.ReadCloser
	uri string
}

func (f *fileParseInput) URI() string { return f.uri }

// TreeBuilder is a SAX2 handler that builds a DOM tree from SAX events,
// analogous to libxml2's default SAX handler (xmlSAX2InitDefaultSAXHandler).
type TreeBuilder struct {
	cachedCtx  context.Context // last context seen
	cachedPCtx *parserCtx      // cached parserCtx for that context
}

func (t *TreeBuilder) pctx(ctxif context.Context) *parserCtx {
	if ctxif == t.cachedCtx {
		return t.cachedPCtx
	}
	p := getParserCtx(ctxif)
	t.cachedCtx = ctxif
	t.cachedPCtx = p
	return p
}

// NewTreeBuilder creates a new TreeBuilder that builds a DOM tree from SAX events.
func NewTreeBuilder() *TreeBuilder {
	return &TreeBuilder{}
}

func (t *TreeBuilder) SetDocumentLocator(ctxif context.Context, loc sax.DocumentLocator) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.SetDocumentLocator")
		defer g.IRelease("END tree.SetDocumentLocator")
	}

	return nil
}

func (t *TreeBuilder) StartDocument(ctxif context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.StartDocument")
		defer g.IRelease("END tree.StartDocument")
	}

	ctx := t.pctx(ctxif)
	ctx.doc = NewDocument(ctx.version, ctx.encoding, ctx.standalone)
	ctx.doc.idsSkip = ctx.loadsubset.IsSet(SkipIDs)
	ctx.doc.url = ctx.baseURI
	return nil
}

func (t *TreeBuilder) EndDocument(ctxif context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.EndDocument")
		defer g.IRelease("END tree.EndDocument")
	}

	ctx := t.pctx(ctxif)
	if ctx.doc != nil && ctx.wellFormed {
		ctx.doc.properties |= DocWellFormed
		if ctx.valid {
			ctx.doc.properties |= DocDTDValid
		}
	}
	return nil
}

func (t *TreeBuilder) ProcessingInstruction(ctxif context.Context, target, data string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.ProcessingInstruction")
		defer g.IRelease("END tree.ProcessingInstruction")
	}
	ctx := t.pctx(ctxif)
	doc := ctx.doc
	pi := doc.CreatePI(target, data)

	// Track external entity base URI for base-uri() resolution.
	if ctx.currentEntityURI != "" {
		pi.entityBaseURI = ctx.currentEntityURI
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

func (t *TreeBuilder) StartElementNS(ctxif context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
	//	ctx := t.pctx(ctxif)
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

	ctx := t.pctx(ctxif)
	doc := ctx.doc

	e := doc.CreateElement(localname)

	e.SetLine(ctx.LineNumber())

	// When this element is being created as part of external entity
	// expansion, record the entity's URI so base-uri() returns the
	// correct value without needing a synthetic xml:base attribute.
	if ctx.currentEntityURI != "" {
		e.entityBaseURI = ctx.currentEntityURI
	}

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
			if ctx.replaceEntities {
				// When replaceEntities is true (ParseNoEnt), entity
				// references are already resolved in the attribute
				// value. Use literal mode to avoid re-parsing & as
				// new entity reference starts.
				_ = e.SetLiteralAttributeNS(attr.LocalName(), attr.Value(), ns)
			} else {
				if _, err := e.SetAttributeNS(attr.LocalName(), attr.Value(), ns); err != nil {
					return err
				}
			}
		} else {
			if ctx.replaceEntities {
				_ = e.SetLiteralAttribute(attr.Name(), attr.Value())
			} else {
				if _, err := e.SetAttribute(attr.Name(), attr.Value()); err != nil {
					return err
				}
			}
		}
	}

	// Propagate attribute types from DTD declarations and register IDs.
	elemName := localname
	if prefix != "" {
		elemName = prefix + ":" + localname
	}
	registerIDs := !ctx.loadsubset.IsSet(SkipIDs)
	e.ForEachAttribute(func(a *Attribute) bool {
		aLocalName := a.LocalName()
		aPrefix := a.Prefix()
		if decl := lookupAttributeDecl(doc, aLocalName, aPrefix, elemName); decl != nil {
			a.SetAType(decl.AType())
		}
		if registerIDs {
			if a.Name() == lexicon.QNameXMLID || a.AType() == enum.AttrID {
				doc.RegisterID(a.Value(), e)
			}
		}
		return true
	})

	var parent MutableNode
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

func (t *TreeBuilder) EndElementNS(ctxif context.Context, localname, prefix, uri string) error {
	if pdebug.Enabled {
		if prefix != "" {
			pdebug.Printf("tree.EndElement: %s:%s", prefix, localname)
		} else {
			pdebug.Printf("tree.EndElement: %s", localname)
		}
	}

	ctx := t.pctx(ctxif)
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

func (t *TreeBuilder) Characters(ctxif context.Context, data []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.Characters: '%s' (%v)", data, data)
		defer g.IRelease("END tree.Characters")
	}

	ctx := t.pctx(ctxif)
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
func (t *TreeBuilder) CDataBlock(ctxif context.Context, data []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.CDATABlock")
		defer g.IRelease("END tree.CDATABlock")
	}

	ctx := t.pctx(ctxif)
	parent := ctx.elem
	if parent == nil {
		return nil
	}

	doc := ctx.doc
	cdata := doc.CreateCDATASection(data)

	return parent.AddChild(cdata)
}

// Comment mirrors xmlSAX2Comment in libxml2's SAX2.c, which delegates
// parent selection to xmlSAX2AppendChild. When inside a DTD subset the
// comment is added to the DTD, not the document.
func (t *TreeBuilder) Comment(ctxif context.Context, data []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.Comment: %s", data)
		defer g.IRelease("END tree.Comment")
	}

	ctx := t.pctx(ctxif)
	doc := ctx.doc
	if doc == nil {
		return errors.New("comment placed in wrong location")
	}

	e := doc.CreateComment(data)

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

func (t *TreeBuilder) InternalSubset(ctxif context.Context, name, eid, uri string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.InternalSubset %s,%s,%s", name, eid, uri)
		defer g.IRelease("END tree.InternalSubset")
	}

	ctx := t.pctx(ctxif)
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

func (t *TreeBuilder) ExternalSubset(ctxif context.Context, name, eid, uri string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.ExternalSubset %s,%s,%s", name, eid, uri)
		defer g.IRelease("END tree.ExternalSubset")
	}

	ctx := t.pctx(ctxif)

	if ctx.options.IsSet(parseNoXXE) {
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
	savedBaseURI := ctx.baseURI
	ctx.external = true
	ctx.baseURI = resolved

	baseLen := ctx.inputTab.Len()
	ctx.pushInput(strcursor.NewByteCursor(bytes.NewReader(data)))

	for ctx.inputTab.Len() > baseLen {
		top := ctx.adaptCursor(ctx.inputTab.PeekOne())
		if top == nil || top.Done() {
			break
		}

		ctx.skipBlanks(ctxif)

		if ctx.inputTab.Len() <= baseLen {
			break
		}
		top = ctx.adaptCursor(ctx.inputTab.PeekOne())
		if top == nil || top.Done() {
			break
		}

		cur := ctx.getCursor()
		if cur != nil && cur.Peek() == '<' && cur.PeekAt(1) == '!' && cur.PeekAt(2) == '[' {
			if err := ctx.parseConditionalSections(ctxif); err != nil {
				break
			}
			continue
		}

		if err := ctx.parseMarkupDecl(ctxif); err != nil {
			break
		}
	}

	// Clean up: ensure our pushed input is removed
	for ctx.inputTab.Len() > baseLen {
		ctx.popInput()
	}

	ctx.external = savedExternal
	ctx.baseURI = savedBaseURI

	return nil
}

func (t *TreeBuilder) HasInternalSubset(ctxif context.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) HasExternalSubset(ctxif context.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) IsStandalone(ctxif context.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) GetEntity(ctxif context.Context, name string) (ent sax.Entity, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.GetEntity '%s'", name)
		defer func() {
			g.IRelease("END tree.GetEntity = '%v'", ent)
		}()
	}

	ctx := t.pctx(ctxif)
	doc := ctx.doc
	x, ok := doc.GetEntity(name)
	if !ok {
		err = errors.New("entity not found")
	} else {
		ent = x
	}
	return
}

func (t *TreeBuilder) GetParameterEntity(ctxif context.Context, name string) (sax.Entity, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.GetParameterEntity '%s'", name)
		defer g.IRelease("END tree.GetParameterEntity")
	}

	if ctxif == nil {
		return nil, ErrInvalidParserCtx
	}

	ctx := t.pctx(ctxif)
	doc := ctx.doc
	if doc == nil {
		return nil, ErrInvalidDocument
	}

	if ret, ok := doc.GetParameterEntity(name); ok {
		return ret, nil
	}

	return nil, ErrEntityNotFound
}

func (t *TreeBuilder) AttributeDecl(ctxif context.Context, eName string, aName string, typ enum.AttributeType, deftype enum.AttributeDefault, value string, enumif sax.Enumeration) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.AttributeDecl name = '%s', elem = '%s'", aName, eName)
		defer g.IRelease("END tree.AttributeDecl")
	}

	ctx := t.pctx(ctxif)

	if aName == lexicon.QNameXMLID && typ != enum.AttrID {
		// libxml2 says "raise the error but keep the validity flag"
		// but I don't know if we can do that..
		return errors.New("xml:id: attribute type should be enum.AttrID")
	}
	var prefix string
	var local string
	if i := strings.IndexByte(aName, ':'); i > -1 {
		prefix = aName[:i]
		local = aName[i+1:]
	} else {
		local = aName
	}

	enum := enumif.(Enumeration) //nolint:forcetypeassert

	doc := ctx.doc
	switch ctx.inSubset {
	case 1:
		if pdebug.Enabled {
			pdebug.Printf("Processing intSubset...")
		}
		if _, err := ctx.addAttributeDecl(doc.intSubset, eName, local, prefix, typ, deftype, value, enum); err != nil {
			return err
		}
	case 2:
		if pdebug.Enabled {
			pdebug.Printf("Processing extSubset...")
		}
		if _, err := ctx.addAttributeDecl(doc.extSubset, eName, local, prefix, typ, deftype, value, enum); err != nil {
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

func (t *TreeBuilder) ElementDecl(ctxif context.Context, name string, typ enum.ElementType, content sax.ElementContent) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.ElementDecl")
		defer g.IRelease("END tree.ElementDecl")
	}

	ctx := t.pctx(ctxif)
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

	_, err := dtd.AddElementDecl(name, typ, content.(*ElementContent)) //nolint:forcetypeassert
	if err != nil {
		return err
	}

	return nil
}

func (t *TreeBuilder) EndDTD(ctxif context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.EndDTD")
		defer g.IRelease("END tree.EndDTD")
	}

	return nil
}

func (t *TreeBuilder) EndEntity(ctxif context.Context, name string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.EndEntity")
		defer g.IRelease("END tree.EndEntity")
	}

	return nil
}
func (t *TreeBuilder) ExternalEntityDecl(ctxif context.Context, name string, publicID string, systemID string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.ExternalEntityDecl")
		defer g.IRelease("END tree.ExternalEntityDecl")
	}

	return nil
}

func (t *TreeBuilder) GetExternalSubset(ctxif context.Context, name string, baseURI string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.GetExternalSubset")
		defer g.IRelease("END tree.GetExternalSubset")
	}

	return nil
}

func (t *TreeBuilder) IgnorableWhitespace(ctxif context.Context, content []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.IgnorableWhitespace (%v)", content)
		defer g.IRelease("END tree.IgnorableWhitespace")
	}

	ctx := t.pctx(ctxif)
	if ctx.keepBlanks {
		return t.Characters(ctxif, content)
	}

	return nil
}

func (t *TreeBuilder) InternalEntityDecl(ctxif context.Context, name string, value string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.InternalEntityDecl")
		defer g.IRelease("END tree.InternalEntityDecl")
	}

	return nil
}

func (t *TreeBuilder) NotationDecl(ctxif context.Context, name string, publicID string, systemID string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.NotationDecl")
		defer g.IRelease("END tree.NotationDecl")
	}

	ctx := t.pctx(ctxif)
	dtd := ctx.doc.intSubset
	if dtd == nil {
		return nil
	}
	_, err := dtd.AddNotation(name, publicID, systemID)
	return err
}

func (t *TreeBuilder) Reference(ctxif context.Context, name string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.Reference '%s'", name)
		defer g.IRelease("END tree.Reference")
	}

	ctx := t.pctx(ctxif)
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

func (t *TreeBuilder) ResolveEntity(ctxif context.Context, publicID string, systemID string) (sax.ParseInput, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.ResolveEntity '%s' '%s'", publicID, systemID)
		defer g.IRelease("END tree.ResolveEntity")
	}

	ctx := t.pctx(ctxif)
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

func (t *TreeBuilder) SkippedEntity(ctxif context.Context, name string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.SkippedEntity '%s'", name)
		defer g.IRelease("END tree.SkippedEntity")
	}

	return nil
}

func (t *TreeBuilder) StartDTD(ctxif context.Context, name string, publicID string, systemID string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.StartDTD")
		defer g.IRelease("END tree.StartDTD")
	}

	return nil
}

func (t *TreeBuilder) StartEntity(ctxif context.Context, name string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.StartEntity")
		defer g.IRelease("END tree.StartEntity")
	}

	return nil
}

func (t *TreeBuilder) EntityDecl(ctxif context.Context, name string, typ enum.EntityType, publicID string, systemID string, notation string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.EntityDecl '%s' -> '%s'", name, notation)
		defer g.IRelease("END tree.EntityDecl")
	}

	ctx := t.pctx(ctxif)
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

	ent, err := dtd.AddEntity(name, typ, publicID, systemID, notation)
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

func (t *TreeBuilder) UnparsedEntityDecl(ctxif context.Context, name string, publicID string, systemID string, notation string) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START tree.UnparsedEntityDecl '%s'", name)
		defer g.IRelease("END tree.UnparsedEntityDecl")
	}

	// Mirror xmlSAX2UnparsedEntityDecl: register the NDATA entity in the DTD.
	ctx := t.pctx(ctxif)
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

	ent, _ := dtd.AddEntity(name, enum.ExternalGeneralUnparsedEntity, publicID, systemID, notation)

	// Build the full URI for unparsed entities by resolving the system ID
	// against the document's base URI (mirrors libxml2's xmlSAX2UnparsedEntityDecl).
	if ent != nil && ent.uri == "" && systemID != "" {
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

func (t *TreeBuilder) Error(ctxif context.Context, err error) error {
	return nil
}

func (t *TreeBuilder) Warning(ctxif context.Context, err error) error {
	return nil
}
