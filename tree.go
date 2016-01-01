package helium

import (
	"errors"
	"strings"

	"github.com/lestrrat/helium/internal/debug"
	"github.com/lestrrat/helium/sax"
)

type TreeBuilder struct {
	elem *Element
}

func NewTreeBuilder() *TreeBuilder {
	return &TreeBuilder{}
}

func (t *TreeBuilder) SetDocumentLocator(ctxif sax.Context, loc sax.DocumentLocator) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.SetDocumentLocator")
		defer g.IRelease("END tree.SetDocumentLocator")
	}

	return nil
}

func (t *TreeBuilder) StartDocument(ctxif sax.Context) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.StartDocument")
		defer g.IRelease("END tree.StartDocument")
	}

	ctx := ctxif.(*parserCtx)
	ctx.doc = NewDocument(ctx.version, ctx.encoding, ctx.standalone)
	return nil
}

func (t *TreeBuilder) EndDocument(ctxif sax.Context) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.EndDocument")
		defer g.IRelease("END tree.EndDocument")
	}
	return nil
}

func (t *TreeBuilder) ProcessingInstruction(ctxif sax.Context, target, data string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.ProcessingInstruction")
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
		doc.IntSubset().AddChild(pi)
	case 2:
		doc.ExtSubset().AddChild(pi)
	}

	parent := t.elem
	if parent == nil {
		doc.AddChild(pi)
	} else if parent.Type() == ElementNode {
		parent.AddChild(pi)
	} else {
		parent.AddSibling(pi)
	}
	return nil
}

func (t *TreeBuilder) StartElementNS(ctxif sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
	//	ctx := ctxif.(*parserCtx)
	if debug.Enabled {
		var name string
		if prefix != "" {
			name = prefix + ":" + localname
		} else {
			name = localname
		}
		g := debug.IPrintf("START tree.StartElement: %s", name)
		defer g.IRelease("END tree.StartElement")
	}
	ctx := ctxif.(*parserCtx)
	doc := ctx.doc
	elem := t.elem

	e, err := doc.CreateElement(localname)
	if err != nil {
		return err
	}

	if uri != "" {
		e.SetNamespace(prefix, uri, true)
	}

	for _, ns := range namespaces {
		e.SetNamespace(ns.Prefix(), ns.URI(), false)
	}

	for _, attr := range attrs {
		e.SetAttribute(attr.Name(), attr.Value())
	}

	if elem == nil {
		if debug.Enabled {
			debug.Printf("parent is nil, adding a child directly to document")
		}
		doc.AddChild(e)
	} else {
		if debug.Enabled {
			debug.Printf("parent is NOT nil, adding a child to '%s'", elem.Name())
		}
		elem.AddChild(e)
	}

	t.elem = e

	return nil
}

func (t *TreeBuilder) EndElementNS(ctxif sax.Context, localname, prefix, uri string) error {
	if debug.Enabled {
		if prefix != "" {
			debug.Printf("tree.EndElement: %s:%s", prefix, localname)
		} else {
			debug.Printf("tree.EndElement: %s", localname)
		}
	}

	cur := t.elem
	if cur == nil {
		return errors.New("no context node to end")
	}

	p := cur.Parent()
	if e, ok := p.(*Element); ok {
		t.elem = e
	} else {
		t.elem = nil
	}
	return nil
}

func (t *TreeBuilder) Characters(ctxif sax.Context, data []byte) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.Characters: '%s' (%v)", data, data)
		defer g.IRelease("END tree.Characters")
	}

	n := t.elem
	if n == nil {
		return errors.New("text content placed in wrong location")
	}

	if debug.Enabled {
		debug.Printf("Calling AddContent() on '%s' node", n.Name())
	}

	return n.AddContent(data)
}

func (t *TreeBuilder) CDataBlock(_ sax.Context, data []byte) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.CDATABlock")
		defer g.IRelease("END tree.CDATABlock")
	}
	return nil
}

func (t *TreeBuilder) Comment(ctxif sax.Context, data []byte) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.Comment: %s", data)
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

	n := t.elem
	if n == nil {
		doc.AddChild(e)
	} else if n.Type() == ElementNode {
		n.AddChild(e)
	} else {
		n.AddSibling(e)
	}
	return nil
}

func (t *TreeBuilder) InternalSubset(ctxif sax.Context, name, eid, uri string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.InternalSubset %s,%s,%s", name, eid, uri)
		defer g.IRelease("END tree.InternalSubset")
	}

	ctx := ctxif.(*parserCtx)
	doc := ctx.doc
	dtd, err := doc.CreateDTD()
	if err != nil {
		return err
	}

	doc.intSubset = dtd
	doc.AddChild(doc.intSubset)
	dtd.externalID = eid
	dtd.name = name
	dtd.systemID = uri
	return nil
}

func (t *TreeBuilder) ExternalSubset(ctxif sax.Context, name, eid, uri string) error {
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
	if debug.Enabled {
		g := debug.IPrintf("START tree.GetEntity '%s'", name)
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
	if debug.Enabled {
		g := debug.IPrintf("START tree.GetParameterEntity")
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
	if debug.Enabled {
		g := debug.IPrintf("START tree.AttributeDecl name = '%s', elem = '%s'", aName, eName)
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
		if debug.Enabled {
			debug.Printf("Processing intSubset...")
		}
		if _, err := ctx.addAttributeDecl(doc.intSubset, eName, local, prefix, AttributeType(typ), AttributeDefault(deftype), value, enum); err != nil {
			return err
		}
	case 2:
		if debug.Enabled {
			debug.Printf("Processing extSubset...")
		}
		if _, err := ctx.addAttributeDecl(doc.extSubset, eName, local, prefix, AttributeType(typ), AttributeDefault(deftype), value, enum); err != nil {
			return err
		}
	default:
		if debug.Enabled {
			debug.Printf("uh-oh we have a problem inSubset = %d", ctx.inSubset)
		}
		return errors.New("TreeBuilder.AttributeDecl called while not in subset")
	}
	/*
	   if (ctxt->vctxt.valid == 0)
	       ctxt->valid = 0;
	   if ((attr != NULL) && (ctxt->validate) && (ctxt->wellFormed) &&
	       (ctxt->myDoc->intSubset != NULL))
	       ctxt->valid &= xmlValidateAttributeDecl(&ctxt->vctxt, ctxt->myDoc,
	                                               attr);
	*/
	return nil
}

func (t *TreeBuilder) ElementDecl(ctxif sax.Context, name string, typ int, content sax.ElementContent) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.ElementDecl")
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
	if debug.Enabled {
		g := debug.IPrintf("START tree.EndDTD")
		defer g.IRelease("END tree.EndDTD")
	}

	return nil
}

func (t *TreeBuilder) EndEntity(ctxif sax.Context, name string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.EndEntity")
		defer g.IRelease("END tree.EndEntity")
	}

	return nil
}
func (t *TreeBuilder) ExternalEntityDecl(ctxif sax.Context, name string, publicID string, systemID string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.ExternalEntityDecl")
		defer g.IRelease("END tree.ExternalEntityDecl")
	}

	return nil
}

func (t *TreeBuilder) GetExternalSubset(ctxif sax.Context, name string, baseURI string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.GetExternalSubset")
		defer g.IRelease("END tree.GetExternalSubset")
	}

	return nil
}

func (t *TreeBuilder) IgnorableWhitespace(ctxif sax.Context, content []byte) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.IgnorableWhitespace (%v)", content)
		defer g.IRelease("END tree.IgnorableWhitespace")
	}

	ctx := ctxif.(*parserCtx)
	if ctx.keepBlanks {
		return t.Characters(ctx, content)
	}

	return nil
}

func (t *TreeBuilder) InternalEntityDecl(ctxif sax.Context, name string, value string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.InternalEntityDecl")
		defer g.IRelease("END tree.InternalEntityDecl")
	}

	return nil
}

func (t *TreeBuilder) NotationDecl(ctxif sax.Context, name string, publicID string, systemID string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.NotationDecl")
		defer g.IRelease("END tree.NotationDecl")
	}

	return nil
}

func (t *TreeBuilder) Reference(ctxif sax.Context, name string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.Reference '%s'", name)
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

	parent := t.elem
	return parent.AddChild(n)
}

func (t *TreeBuilder) ResolveEntity(ctxif sax.Context, publicID string, systemID string) (sax.ParseInput, error) {
	if debug.Enabled {
		g := debug.IPrintf("START tree.ResolveEntity '%s'", publicID, systemID)
		defer g.IRelease("END tree.ResolveEntity")
	}

	return nil, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) SkippedEntity(ctxif sax.Context, name string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.SkippedEntity '%s'", name)
		defer g.IRelease("END tree.SkippedEntity")
	}

	return nil
}

func (t *TreeBuilder) StartDTD(ctxif sax.Context, name string, publicID string, systemID string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.StartDTD")
		defer g.IRelease("END tree.StartDTD")
	}

	return nil
}

func (t *TreeBuilder) StartEntity(ctxif sax.Context, name string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.StartEntity")
		defer g.IRelease("END tree.StartEntity")
	}

	return nil
}

func (t *TreeBuilder) EntityDecl(ctxif sax.Context, name string, typ int, publicID string, systemID string, notation string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.EntityDecl '%s'", name)
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

	ent, err := dtd.RegisterEntity(name, EntityType(typ), publicID, systemID, notation)
	if err != nil {
		return err
	}

	if ent.uri == "" && systemID != "" {
			/*
            xmlChar *URI;
            const char *base = NULL;

            if (ctxt->input != NULL)
                base = ctxt->input->filename;
            if (base == NULL)
                base = ctxt->directory;

            URI = xmlBuildURI(systemId, (const xmlChar *) base);
            ent->URI = URI;
			*/
	}

	return nil
}

func (t *TreeBuilder) UnparsedEntityDecl(ctxif sax.Context, name string, publicID string, systemID string, notation string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.UnparsedEntityDecl '%s'", name)
		defer g.IRelease("END tree.UnparsedEntityDecl")
	}

	// Because the parser needs to know about entities even in cases where
	// there isn't a SAX handler registered, call to Document.RegisterEntry
	// is done in the main parser -- and not here.
	return nil
}

func (t *TreeBuilder) Error(ctxif sax.Context, message string, args ...interface{}) error {
	return nil
}
