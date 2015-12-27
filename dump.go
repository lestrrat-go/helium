package helium

import "io"

type Dumper struct{}

func (d *Dumper) writeString(out io.Writer, content string) error {
	// punt all the magic for now
	_, err := io.WriteString(out, content)
	return err
}

func (d *Dumper) DumpDoc(out io.Writer, doc *Document) error {
	for e := doc.FirstChild(); e != nil; e = e.NextSibling() {
		if err := d.DumpNode(out, e); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dumper) DumpNode(out io.Writer, n Node) error {
	var err error
	switch n.Type() {
//	case DTDNode:
//		err = d.DumpDTD(out, n.(*DTD))
	case TextNode:
		c := n.Content()
		if string(c) == XMLTextNoEnc {
			panic("unimplemented")
		} else {
			err = d.writeString(out, string(c))
		}
		return nil // no recursing down
	}

	if err != nil {
		return err
	}

	// if it got here it's some sort of an element

	name := n.Name()
	if nser, ok := n.(Namespacer); ok {
		if prefix := nser.Prefix(); prefix != "" {
			name = prefix + ":" + name
		}
	}
	io.WriteString(out, "<")
	io.WriteString(out, name)
	io.WriteString(out, ">")

	if child := n.FirstChild(); child != nil {
		for ; child != nil; child = child.NextSibling() {
			if err := d.DumpNode(out, child); err != nil {
				return err
			}
		}
	}

	io.WriteString(out, "</")
	io.WriteString(out, name)
	io.WriteString(out, ">")
	

	return nil
}