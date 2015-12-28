package helium

import (
	"io"

	"github.com/lestrrat/helium/internal/debug"
)

type Dumper struct{}

func (d *Dumper) writeString(out io.Writer, content string) error {
	// punt all the magic for now
	_, err := io.WriteString(out, content)
	return err
}

func (d *Dumper) DumpDoc(out io.Writer, doc *Document) error {
	if err := d.DumpNode(out, doc); err != nil {
		return err
	}

	for e := doc.FirstChild(); e != nil; e = e.NextSibling() {
		if err := d.DumpNode(out, e); err != nil {
			return err
		}
	}
	io.WriteString(out, "\n")
	return nil
}

func (d *Dumper) dumpDocContent(out io.Writer, n Node) error {
	doc := n.(*Document)
	io.WriteString(out, `<?xml version="`)
	version := doc.Version()
	if version == "" {
		version = "1.0"
	}
	io.WriteString(out, version+`"`)

	if encoding := doc.encoding; encoding != "" {
		io.WriteString(out, ` encoding="`+encoding+`"`)
	}

	switch doc.Standalone() {
	case StandaloneExplicitNo:
		io.WriteString(out, ` standalone="no"`)
	case StandaloneExplicitYes:
		io.WriteString(out, ` standalone="yes"`)
	}
	io.WriteString(out, "?>\n")
	return nil
}

func (d *Dumper) DumpNode(out io.Writer, n Node) error {
	var err error
	switch n.Type() {
	case DocumentNode:
		if err = d.dumpDocContent(out, n); err != nil {
			return err
		}
		return nil
		//	case DTDNode:
		//		err = d.DumpDTD(out, n.(*DTD))
	case TextNode:
		c := n.Content()
		if string(c) == XMLTextNoEnc {
			panic("unimplemented")
		} else {
			debug.Printf("Text node! -> '%s'", c)
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

	if e, ok := n.(*Element); ok {
		for attr := e.properties; attr != nil; {
			io.WriteString(out, " "+attr.Name()+`="`+attr.Value()+`"`)
			if a := attr.NextSibling(); a != nil {
				attr = a.(*Attribute)
			} else {
				attr = nil
			}
		}

		if child := e.FirstChild(); child == nil {
			io.WriteString(out, "/>")
			return nil
		}
	}


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