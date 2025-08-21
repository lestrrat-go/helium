package helium

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/lestrrat-go/helium/sax"
)

type TreeBuilder struct {
}

func NewTreeBuilder() *TreeBuilder {
	return &TreeBuilder{}
}

func (t *TreeBuilder) SetDocumentLocator(ctx context.Context, ctxif sax.Context, loc sax.DocumentLocator) error {
	return nil
}

func (t *TreeBuilder) StartDocument(ctx context.Context, ctxif sax.Context) error {
	pctx := ctxif.(*parserCtx)

	ctx, span := StartSpan(ctx, "TreeBuilder.StartDocument")
	defer span.End()

	TraceEvent(ctx, "parsing started",
		slog.String("version", pctx.version),
		slog.String("encoding", pctx.encoding),
		slog.Int("standalone", int(pctx.standalone)))

	pctx.doc = NewDocument(pctx.version, pctx.encoding, pctx.standalone)

	TraceEvent(ctx, "document created successfully")
	return nil
}

func (t *TreeBuilder) EndDocument(ctx context.Context, ctxif sax.Context) error {
	ctx, span := StartSpan(ctx, "TreeBuilder.EndDocument")
	defer span.End()

	TraceEvent(ctx, "parsing completed")
	return nil
}

func (t *TreeBuilder) ProcessingInstruction(ctx context.Context, ctxif sax.Context, target, data string) error {
	pctx := ctxif.(*parserCtx)

	ctx, span := StartSpan(ctx, "TreeBuilder.ProcessingInstruction")
	defer span.End()

	TraceEvent(ctx, "processing instruction",
		slog.String("target", target),
		slog.String("data", data))

	doc := pctx.doc
	pi, err := doc.CreatePI(target, data)
	if err != nil {
		return err
	}

	switch pctx.inSubset {
	case 1:
		if err := doc.IntSubset().AddChild(pi); err != nil {
			return err
		}
		TraceEvent(ctx, "PI added to internal subset")
	case 2:
		if err := doc.ExtSubset().AddChild(pi); err != nil {
			return err
		}
		TraceEvent(ctx, "PI added to external subset")
	}

	parent := pctx.elem
	if parent == nil {
		if err := doc.AddChild(pi); err != nil {
			return err
		}
		TraceEvent(ctx, "PI added to document")
	} else if parent.Type() == ElementNode {
		if err := parent.AddChild(pi); err != nil {
			return err
		}
		TraceEvent(ctx, "PI added as child element")
	} else {
		if err := parent.AddSibling(pi); err != nil {
			return err
		}
		TraceEvent(ctx, "PI added as sibling")
	}
	TraceEvent(ctx, "processing instruction completed successfully")
	return nil
}

func (t *TreeBuilder) StartElementNS(ctx context.Context, ctxif sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
	pctx := ctxif.(*parserCtx)

	ctx, span := StartSpan(ctx, "TreeBuilder.StartElementNS")
	defer span.End()

	TraceEvent(ctx, "element start",
		slog.String("element_name", localname),
		slog.String("prefix", prefix),
		slog.String("uri", uri),
		slog.Int("namespace_count", len(namespaces)),
		slog.Int("attribute_count", len(attrs)))

	doc := pctx.doc
	e, err := doc.CreateElement(localname)
	if err != nil {
		return err
	}

	if uri != "" {
		if err := e.SetNamespace(prefix, uri, true); err != nil {
			return err
		}
	}

	for _, ns := range namespaces {
		if err := e.SetNamespace(ns.Prefix(), ns.URI(), false); err != nil {
			return err
		}
	}

	for _, attr := range attrs {
		if attr.IsDefault() && !pctx.loadsubset.IsSet(CompleteAttrs) {
			continue
		}
		if err := e.SetAttribute(attr.Name(), attr.Value()); err != nil {
			return err
		}
	}

	var parent Node
	if e := pctx.elem; e != nil {
		parent = e
	}
	if parent == nil {
		if err := doc.AddChild(e); err != nil {
			return err
		}
		TraceEvent(ctx, "element added to document as root")
	} else if parent.Type() == ElementNode {
		if err := parent.AddChild(e); err != nil {
			return err
		}
		TraceEvent(ctx, "element added as child")
	} else {
		if err := parent.AddSibling(e); err != nil {
			return err
		}
		TraceEvent(ctx, "element added as sibling")
	}

	pctx.elem = e
	TraceEvent(ctx, "element processing completed successfully")
	return nil
}

func (t *TreeBuilder) EndElementNS(ctx context.Context, ctxif sax.Context, localname, prefix, uri string) error {
	pctx := ctxif.(*parserCtx)

	ctx, span := StartSpan(ctx, "TreeBuilder.EndElementNS")
	defer span.End()

	TraceEvent(ctx, "element end",
		slog.String("element_name", localname),
		slog.String("prefix", prefix),
		slog.String("uri", uri))

	cur := pctx.elem
	if cur == nil {
		return errors.New("no context node to end")
	}

	p := cur.Parent()
	if e, ok := p.(*Element); ok {
		pctx.elem = e
		TraceEvent(ctx, "moved up to parent element", slog.String("parent_name", e.LocalName()))
	} else {
		pctx.elem = nil
		TraceEvent(ctx, "moved up to document root")
	}
	TraceEvent(ctx, "element end processing completed successfully")
	return nil
}

func (t *TreeBuilder) Characters(ctx context.Context, ctxif sax.Context, data []byte) error {
	pctx := ctxif.(*parserCtx)

	TraceEvent(ctx, "character data",
		slog.Int("data_length", len(data)))

	n := pctx.elem
	if n == nil {
		return errors.New("text content placed in wrong location")
	}

	if err := n.AddContent(data); err != nil {
		return err
	}

	TraceEvent(ctx, "character data added successfully")
	return nil
}

func (t *TreeBuilder) CDataBlock(ctx context.Context, ctxif sax.Context, data []byte) error {
	TraceEvent(ctx, "CDATA block",
		slog.Int("data_length", len(data)))
	return nil
}

func (t *TreeBuilder) Comment(ctx context.Context, ctxif sax.Context, data []byte) error {
	pctx := ctxif.(*parserCtx)

	ctx, span := StartSpan(ctx, "TreeBuilder.Comment")
	defer span.End()

	TraceEvent(ctx, "comment",
		slog.Int("comment_length", len(data)))

	doc := pctx.doc
	if doc == nil {
		return errors.New("comment placed in wrong location")
	}

	e, err := doc.CreateComment(data)
	if err != nil {
		return err
	}

	n := pctx.elem
	if n == nil {
		if err := doc.AddChild(e); err != nil {
			return err
		}
		TraceEvent(ctx, "comment added to document")
	} else if n.Type() == ElementNode {
		if err := n.AddChild(e); err != nil {
			return err
		}
		TraceEvent(ctx, "comment added as child")
	} else {
		if err := n.AddSibling(e); err != nil {
			return err
		}
		TraceEvent(ctx, "comment added as sibling")
	}
	TraceEvent(ctx, "comment processing completed successfully")
	return nil
}

func (t *TreeBuilder) InternalSubset(ctx context.Context, ctxif sax.Context, name, eid, uri string) error {
	pctx := ctxif.(*parserCtx)
	doc := pctx.doc

	dtd, err := doc.InternalSubset()
	if err == nil {
		/* TODO
		if ctx.html {
			return nil
		} */
		dtd.Free()
		doc.intSubset = nil // hmm, do we need this?
	}

	_, err = doc.CreateInternalSubset(name, eid, uri)
	if err != nil {
		return err
	}

	return nil
}

func (t *TreeBuilder) ExternalSubset(ctx context.Context, ctxif sax.Context, name, eid, uri string) error {
	return nil
}

func (t *TreeBuilder) HasInternalSubset(ctx context.Context, ctxif sax.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) HasExternalSubset(ctx context.Context, ctxif sax.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) IsStandalone(ctx context.Context, ctxif sax.Context) (bool, error) {
	return false, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) GetEntity(ctx context.Context, ctxif sax.Context, name string) (ent sax.Entity, err error) {
	pctx := ctxif.(*parserCtx)
	doc := pctx.doc
	x, ok := doc.GetEntity(name)
	if !ok {
		err = errors.New("entity not found")
	} else {
		ent = x
	}
	return
}

func (t *TreeBuilder) GetParameterEntity(ctx context.Context, ctxif sax.Context, name string) (sax.Entity, error) {
	if ctxif == nil {
		return nil, ErrInvalidParserCtx
	}

	pctx := ctxif.(*parserCtx)
	doc := pctx.doc
	if doc == nil {
		return nil, ErrInvalidDocument
	}

	if ret, ok := doc.GetParameterEntity(name); ok {
		return ret, nil
	}

	return nil, ErrEntityNotFound
}

func (t *TreeBuilder) AttributeDecl(ctx context.Context, ctxif sax.Context, eName string, aName string, typ int, deftype int, value string, enumif sax.Enumeration) error {
	pctx := ctxif.(*parserCtx)

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

	doc := pctx.doc
	switch pctx.inSubset {
	case 1:
		if _, err := pctx.addAttributeDecl(doc.intSubset, eName, local, prefix, AttributeType(typ), AttributeDefault(deftype), value, enum); err != nil {
			return err
		}
	case 2:
		if _, err := pctx.addAttributeDecl(doc.extSubset, eName, local, prefix, AttributeType(typ), AttributeDefault(deftype), value, enum); err != nil {
			return err
		}
	default:
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

func (t *TreeBuilder) ElementDecl(ctx context.Context, ctxif sax.Context, name string, typ int, content sax.ElementContent) error {
	pctx := ctxif.(*parserCtx)
	doc := pctx.doc
	var dtd *DTD
	switch pctx.inSubset {
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

func (t *TreeBuilder) EndDTD(ctx context.Context, ctxif sax.Context) error {
	return nil
}

func (t *TreeBuilder) EndEntity(ctx context.Context, ctxif sax.Context, name string) error {
	return nil
}
func (t *TreeBuilder) ExternalEntityDecl(ctx context.Context, ctxif sax.Context, name string, publicID string, systemID string) error {
	return nil
}

func (t *TreeBuilder) GetExternalSubset(ctx context.Context, ctxif sax.Context, name string, baseURI string) error {
	return nil
}

func (t *TreeBuilder) IgnorableWhitespace(ctx context.Context, ctxif sax.Context, content []byte) error {
	pctx := ctxif.(*parserCtx)
	if pctx.keepBlanks {
		return t.Characters(ctx, ctxif, content)
	}

	return nil
}

func (t *TreeBuilder) InternalEntityDecl(ctx context.Context, ctxif sax.Context, name string, value string) error {
	return nil
}

func (t *TreeBuilder) NotationDecl(ctx context.Context, ctxif sax.Context, name string, publicID string, systemID string) error {
	return nil
}

func (t *TreeBuilder) Reference(ctx context.Context, ctxif sax.Context, name string) error {
	pctx := ctxif.(*parserCtx)
	doc := pctx.doc
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

	parent := pctx.elem
	return parent.AddChild(n)
}

func (t *TreeBuilder) ResolveEntity(ctx context.Context, ctxif sax.Context, publicID string, systemID string) (sax.ParseInput, error) {
	return nil, sax.ErrHandlerUnspecified
}

func (t *TreeBuilder) SkippedEntity(ctx context.Context, ctxif sax.Context, name string) error {
	return nil
}

func (t *TreeBuilder) StartDTD(ctx context.Context, ctxif sax.Context, name string, publicID string, systemID string) error {
	return nil
}

func (t *TreeBuilder) StartEntity(ctx context.Context, ctxif sax.Context, name string) error {
	return nil
}

func (t *TreeBuilder) EntityDecl(ctx context.Context, ctxif sax.Context, name string, typ int, publicID string, systemID string, notation string) error {
	pctx := ctxif.(*parserCtx)
	doc := pctx.doc
	var dtd *DTD
	switch pctx.inSubset {
	case 1:
		dtd = doc.intSubset
	case 2:
		dtd = doc.extSubset
	default:
		return errors.New("sax.EntityDecl called while note in subset")
	}

	_, err := dtd.AddEntity(name, EntityType(typ), publicID, systemID, notation)
	if err != nil {
		return err
	}

	/*
		if ent.uri == "" && systemID != "" {
			   xmlChar *URI;
			   const char *base = NULL;

			   if (ctxt->input != NULL)
			       base = ctxt->input->filename;
			   if (base == NULL)
			       base = ctxt->directory;

			   URI = xmlBuildURI(systemId, (const xmlChar *) base);
			   ent->URI = URI;
		}
	*/

	return nil
}

func (t *TreeBuilder) UnparsedEntityDecl(ctx context.Context, ctxif sax.Context, name string, publicID string, systemID string, notation string) error {
	// Because the parser needs to know about entities even in cases where
	// there isn't a SAX handler registered, call to Document.RegisterEntry
	// is done in the main parser -- and not here.
	return nil
}

func (t *TreeBuilder) Error(ctx context.Context, ctxif sax.Context, message string, args ...interface{}) error {
	return nil
}
