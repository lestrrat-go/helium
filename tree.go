package helium

import (
	"errors"
	"strings"

	"github.com/lestrrat/helium/internal/debug"
	"github.com/lestrrat/helium/sax"
)

type TreeBuilder struct {
	doc  *Document
	node Node
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

	t.doc = NewDocument(ctx.version, ctx.encoding, ctx.standalone)
	return nil
}

func (t *TreeBuilder) EndDocument(ctxif sax.Context) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.EndDocument")
		defer g.IRelease("END tree.EndDocument")
	}
	ctx := ctxif.(*parserCtx)
	ctx.doc = t.doc
	t.doc = nil
	return nil
}

func (t *TreeBuilder) ProcessingInstruction(ctxif sax.Context, target, data string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.ProcessingInstruction")
		defer g.IRelease("END tree.ProcessingInstruction")
	}

	pi, err := t.doc.CreatePI(target, data)
	if err != nil {
		return err
	}

	ctx := ctxif.(*parserCtx)
	switch ctx.inSubset {
	case 1:
		t.doc.IntSubset().AddChild(pi)
	case 2:
		t.doc.ExtSubset().AddChild(pi)
	}

	if t.node == nil {
		t.doc.AddChild(pi)
	} else if t.node.Type() == ElementNode {
		t.node.AddChild(pi)
	} else {
		t.node.AddSibling(pi)
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
	e, err := t.doc.CreateElement(localname)
	if err != nil {
		return err
	}

	for _, attr := range attrs {
		e.SetAttribute(attr.Name(), attr.Value())
	}

	if t.node == nil {
		t.doc.AddChild(e)
	} else {
		t.node.AddChild(e)
	}

	t.node = e

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

	if e, ok := t.node.(*Element); ok && e.LocalName() == localname && e.Prefix() == prefix && e.URI() == uri {
		parent := t.node.Parent()
		if debug.Enabled {
			pname := "(null)"
			if parent != nil {
				pname = parent.Name()
			}
			debug.Printf("Setting t.node to '%s' (t.node.Parent())", pname)
		}
		t.node = parent
	}
	return nil
}

func (t *TreeBuilder) Characters(ctxif sax.Context, data []byte) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.Characters: '%s' (%v)", data, data)
		defer g.IRelease("END tree.Characters")
	}

	if t.node == nil {
		return errors.New("text content placed in wrong location")
	}

	if debug.Enabled {
		debug.Printf("Calling AddContent() on '%s' node", t.node.Name())
	}

	return t.node.AddContent(data)
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

	if t.doc == nil {
		return errors.New("comment placed in wrong location")
	}

	e, err := t.doc.CreateComment(data)
	if err != nil {
		return err
	}

	if t.node == nil {
		t.doc.AddChild(e)
	} else if t.node.Type() == ElementNode {
		t.node.AddChild(e)
	} else {
		t.node.AddSibling(e)
	}
	return nil
}

func (t *TreeBuilder) InternalSubset(ctxif sax.Context, name, eid, uri string) error {
	dtd, err := t.doc.CreateDTD()
	if err != nil {
		return err
	}
	t.doc.intSubset = dtd
	t.doc.AddChild(t.doc.intSubset)
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

	err = sax.ErrHandlerUnspecified
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

	switch ctx.inSubset {
	case 1:
		if debug.Enabled {
			debug.Printf("Processing intSubset...")
		}
		if _, err := ctx.addAttributeDecl(t.doc.intSubset, eName, local, prefix, AttributeType(typ), AttributeDefault(deftype), value, enum); err != nil {
			return err
		}
	case 2:
		if debug.Enabled {
			debug.Printf("Processing extSubset...")
		}
		if _, err := ctx.addAttributeDecl(t.doc.extSubset, eName, local, prefix, AttributeType(typ), AttributeDefault(deftype), value, enum); err != nil {
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

func (t *TreeBuilder) ElementDecl(ctx sax.Context, name string, typ int, content sax.ElementContent) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.ElementDecl")
		defer g.IRelease("END tree.ElementDecl")
	}

	return nil
}

func (t *TreeBuilder) EndDTD(ctx sax.Context) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.EndDTD")
		defer g.IRelease("END tree.EndDTD")
	}

	return nil
}

func (t *TreeBuilder) EndEntity(ctx sax.Context, name string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.EndEntity")
		defer g.IRelease("END tree.EndEntity")
	}

	return nil
}
func (t *TreeBuilder) ExternalEntityDecl(ctx sax.Context, name string, publicID string, systemID string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.ExternalEntityDecl")
		defer g.IRelease("END tree.ExternalEntityDecl")
	}

	return nil
}

func (t *TreeBuilder) GetExternalSubset(ctx sax.Context, name string, baseURI string) error {
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

func (t *TreeBuilder) InternalEntityDecl(ctx sax.Context, name string, value string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.InternalEntityDecl")
		defer g.IRelease("END tree.InternalEntityDecl")
	}

	return nil
}

func (t *TreeBuilder) NotationDecl(ctx sax.Context, name string, publicID string, systemID string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.NotationDecl")
		defer g.IRelease("END tree.NotationDecl")
	}

	return nil
}

func (t *TreeBuilder) Reference(ctx sax.Context, name string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.Reference '%s'", name)
		defer g.IRelease("END tree.Reference")
	}

	return nil
}

func (t *TreeBuilder) ResolveEntity(ctx sax.Context, publicID string, systemID string) (sax.ParseInput, error) {
	if debug.Enabled {
		g := debug.IPrintf("START tree.ResolveEntity '%s'", publicID, systemID)
		defer g.IRelease("END tree.ResolveEntity")
	}

	return nil, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) SkippedEntity(ctx sax.Context, name string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.SkippedEntity '%s'", name)
		defer g.IRelease("END tree.SkippedEntity")
	}

	return nil
}

func (t *TreeBuilder) StartDTD(ctx sax.Context, name string, publicID string, systemID string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.StartDTD")
		defer g.IRelease("END tree.StartDTD")
	}

	return nil
}

func (t *TreeBuilder) StartEntity(ctx sax.Context, name string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.StartEntity")
		defer g.IRelease("END tree.StartEntity")
	}

	return nil
}

func (t *TreeBuilder) EntityDecl(ctx sax.Context, name string, typ int, publicID string, systemID string, notation string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.EntityDecl '%s'", name)
		defer g.IRelease("END tree.EntityDecl")
	}
	return nil
}

func (t *TreeBuilder) UnparsedEntityDecl(ctx sax.Context, name string, publicID string, systemID string, notation string) error {
	if debug.Enabled {
		g := debug.IPrintf("START tree.UnparsedEntityDecl '%s'", name)
		defer g.IRelease("END tree.UnparsedEntityDecl")
	}

	// Because the parser needs to know about entities even in cases where
	// there isn't a SAX handler registered, call to Document.RegisterEntry
	// is done in the main parser -- and not here.
	return nil
}

func (t *TreeBuilder) Error(ctx sax.Context, message string, args ...interface{}) error {
	return nil
}
