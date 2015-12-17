package helium

import (
	"errors"

	"github.com/lestrrat/helium/internal/debug"
)

type TreeBuilder struct {
	doc  *Document
	node Node
}

func (t *TreeBuilder) StartDocument(ctxif interface{}) error {
	ctx := ctxif.(*parserCtx)
	t.doc = NewDocument(ctx.version, ctx.encoding, ctx.standalone)
	return nil
}

func (t *TreeBuilder) EndDocument(ctxif interface{}) error {
	ctx := ctxif.(*parserCtx)
	ctx.doc = t.doc
	t.doc = nil
	return nil
}

func (t *TreeBuilder) ProcessingInstruction(ctxif interface{}, target, data string) error {
	//	ctx := ctxif.(*parserCtx)
	pi, err := t.doc.CreatePI(target, data)
	if err != nil {
		return err
	}

	// register to the document
	t.doc.IntSubset().AddChild(pi)
	if t.node == nil {
		t.doc.AddChild(pi)
		return nil
	}

	// what's the "current" node?
	if t.node.Type() == ElementNode {
		t.node.AddChild(pi)
	} else {
		t.node.AddSibling(pi)
	}
	return nil
}

func (t *TreeBuilder) StartElement(ctxif interface{}, elem *ParsedElement) error {
	//	ctx := ctxif.(*parserCtx)
	if debug.Enabled {
		debug.Printf("tree.StartElement: %#v", elem)
	}
	e, err := t.doc.CreateElement(elem.Local())
	if err != nil {
		return err
	}

	// attrdata = []string{ local, value, prefix }
	for _, data := range elem.Attributes() {
		e.SetAttribute(data.prefix+":"+data.local, data.value)
	}

	if t.node == nil {
		t.doc.AddChild(e)
	} else {
		t.node.AddChild(e)
	}

	t.node = e

	return nil
}

func (t *TreeBuilder) EndElement(ctxif interface{}, elem *ParsedElement) error {
	if debug.Enabled {
		debug.Printf("tree.EndElement: %#v", elem)
	}
	return nil
}

func (t *TreeBuilder) Characters(ctxif interface{}, data []byte) error {
	if debug.Enabled {
		debug.Printf("tree.Characters: %s", data)
	}

	if t.node == nil {
		return errors.New("text content placed in wrong location")
	}

	e, err := t.doc.CreateText(data)
	if err != nil {
		return err
	}
	t.node.AddChild(e)
	return nil
}

func (t *TreeBuilder) CDATABlock(ctxif interface{}, data []byte) error {
	return t.Characters(ctxif, data)
}

func (t *TreeBuilder) Comment(ctxif interface{}, data []byte) error {
	if debug.Enabled {
		debug.Printf("tree.Comment: %s", data)
	}

	if t.node == nil {
		return errors.New("comment placed in wrong location")
	}

	e, err := t.doc.CreateComment(data)
	if err != nil {
		return err
	}
	t.node.AddChild(e)
	return nil
}
