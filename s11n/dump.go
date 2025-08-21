package s11n

import (
	"bytes"
	"io"
	"strings"

	"github.com/lestrrat-go/helium/node"
)

type Dumper struct{}

func (d *Dumper) DumpDoc(out io.Writer, doc *node.Document) error {
	// Note: s11n dumping functions lack context for tracing

	if err := d.DumpNode(out, doc); err != nil {
		return err
	}

	for e := doc.FirstChild(); e != nil; e = e.NextSibling() {
		if err := d.DumpNode(out, e); err != nil {
			return err
		}
		_, _ = io.WriteString(out, "\n")
	}
	return nil
}

func (d *Dumper) dumpDocContent(out io.Writer, n node.Node) error {
	// Note: s11n dumping functions lack context for tracing

	doc := n.(*node.Document)
	_, _ = io.WriteString(out, `<?xml version="`)
	version := doc.Version()
	if version == "" {
		version = "1.0"
	}
	_, _ = io.WriteString(out, version+`"`)

	if encoding := doc.Encoding(); encoding != "" && encoding != "utf8" {
		_, _ = io.WriteString(out, ` encoding="`+encoding+`"`)
	}

	switch doc.Standalone() {
	case node.StandaloneExplicitNo:
		_, _ = io.WriteString(out, ` standalone="no"`)
	case node.StandaloneExplicitYes:
		_, _ = io.WriteString(out, ` standalone="yes"`)
	}
	_, _ = io.WriteString(out, "?>\n")
	return nil
}

func (d *Dumper) dumpDTD(out io.Writer, n node.Node) error {
	// DTD type doesn't fully implement Node interface yet
	// Using a basic implementation for now
	_, _ = io.WriteString(out, "<!DOCTYPE ")
	_, _ = io.WriteString(out, n.LocalName())
	_, _ = io.WriteString(out, " ")

	_, _ = io.WriteString(out, "[\n")

	for e := n.FirstChild(); e != nil; e = e.NextSibling() {
		if err := d.DumpNode(out, e); err != nil {
			return err
		}
	}

	_, _ = io.WriteString(out, "]>")
	return nil
}

func (d *Dumper) dumpEnumeration(out io.Writer, n node.Enumeration) error {
	l := len(n)
	for i, v := range n {
		_, _ = io.WriteString(out, v)
		if i != l-1 {
			_, _ = io.WriteString(out, " | ")
		}
	}
	_, _ = io.WriteString(out, ")")
	return nil
}

func dumpElementDeclPrologue(out io.Writer, n node.Node) {
	_, _ = io.WriteString(out, "<!ELEMENT ")
	_, _ = io.WriteString(out, n.LocalName())
}

func dumpElementContent(out io.Writer, n *node.ElementContent, glob bool) error {
	// Note: s11n dumping functions lack context for tracing
	if n == nil {
		return nil
	}

	if glob {
		_, _ = io.WriteString(out, "(")
	}

	// Note: These fields need to be accessible or we need getter methods
	// For now, using placeholder implementation
	_, _ = io.WriteString(out, "#PCDATA")

	if glob {
		_, _ = io.WriteString(out, ")")
	}

	return nil
}

func dumpEntityContent(out io.Writer, content string) error {
	if strings.IndexByte(content, '%') == -1 {
		if err := DumpQuotedString(out, content); err != nil {
			return err
		}
		return nil
	}

	_, _ = io.WriteString(out, `"`)
	rdr := strings.NewReader(content)
	buf := bytes.Buffer{}
	for rdr.Len() > 0 {
		c, err := rdr.ReadByte()
		if err != nil {
			return err
		}
		switch c {
		case '"':
			if buf.Len() > 0 {
				if _, err := buf.WriteTo(out); err != nil {
					return err
				}
				buf.Reset()
			}
			_, _ = io.WriteString(out, "&quot;")
		case '%':
			if buf.Len() > 0 {
				if _, err := buf.WriteTo(out); err != nil {
					return err
				}
				buf.Reset()
			}
			_, _ = io.WriteString(out, "&#x25;")
		default:
			_ = buf.WriteByte(c)
		}
	}
	if buf.Len() > 0 {
		if _, err := buf.WriteTo(out); err != nil {
			return err
		}
	}
	_, _ = io.WriteString(out, `"`)

	return nil
}

func (d *Dumper) dumpEntityDecl(out io.Writer, n node.Node) error {
	if n == nil {
		return nil
	}

	_, _ = io.WriteString(out, "<!ENTITY ")
	_, _ = io.WriteString(out, n.LocalName())
	_, _ = io.WriteString(out, " \"content\">")

	return nil
}

func (d *Dumper) dumpElementDecl(out io.Writer, n node.Node) error {
	dumpElementDeclPrologue(out, n)
	_, _ = io.WriteString(out, " ANY>\n")
	return nil
}

func (d *Dumper) dumpAttributeDecl(out io.Writer, n node.Node) error {
	_, _ = io.WriteString(out, "<!ATTLIST ")
	_, _ = io.WriteString(out, n.LocalName())
	_, _ = io.WriteString(out, " CDATA>")
	return nil
}

func (d *Dumper) dumpNsList(out io.Writer, nslist []*node.Namespace) error {
	for _, ns := range nslist {
		if err := d.dumpNs(out, ns); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dumper) dumpNs(out io.Writer, ns *node.Namespace) error {
	// Note: These fields need to be accessible
	// For now, returning without doing anything
	return nil
}

func (d *Dumper) DumpNode(out io.Writer, n node.Node) error {
	// Note: s11n dumping functions lack context for tracing

	var err error
	switch n.Type() {
	case node.DocumentNodeType:
		if err = d.dumpDocContent(out, n); err != nil {
			return err
		}
		return nil
	case node.DTDNodeType:
		if err = d.dumpDTD(out, n); err != nil {
			return err
		}
		return nil
	case node.CommentNodeType:
		_, _ = io.WriteString(out, "<!--")
		content, err := n.Content(nil)
		if err != nil {
			return err
		}
		_, _ = out.Write(content)
		_, _ = io.WriteString(out, "-->")
		return nil
	case node.EntityRefNodeType:
		_, _ = io.WriteString(out, "&")
		_, _ = io.WriteString(out, n.LocalName())
		_, _ = io.WriteString(out, ";")
		return nil
	case node.TextNodeType:
		c, err := n.Content(nil)
		if err != nil {
			return err
		}
		if string(c) == "textnoenc" {
			panic("unimplemented")
		} else {
			if err := EscapeText(out, c, false); err != nil {
				return err
			}
		}
		return nil // no recursing down
	case node.ElementDeclNodeType:
		if err = d.dumpElementDecl(out, n); err != nil {
			return err
		}
		return nil
	case node.AttributeDeclNodeType:
		if err = d.dumpAttributeDecl(out, n); err != nil {
			return err
		}
		return nil
	case node.EntityNodeType:
		if err = d.dumpEntityDecl(out, n); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	// Note: s11n dumping functions lack context for tracing

	// if it got here it's some sort of an element
	var name string
	var nslist []*node.Namespace
	if nser, ok := n.(node.Namespacer); ok {
		if prefix := nser.Prefix(); prefix != "" {
			name = prefix + ":" + nser.LocalName()
		} else {
			name = nser.LocalName()
		}
		// nslist = nser.Namespaces() // Not available in current implementation
	} else {
		name = n.LocalName()
	}

	_, _ = io.WriteString(out, "<")
	_, _ = io.WriteString(out, name)

	if len(nslist) > 0 {
		if err := d.dumpNsList(out, nslist); err != nil {
			return err
		}
	}

	if e, ok := n.(*node.Element); ok {
		// TODO: Implement attribute dumping when attribute types are properly defined
		// attrs := e.Attributes(nil)
		// for _, attr := range attrs {
		//     // Process attributes
		// }

		if child := e.FirstChild(); child == nil {
			_, _ = io.WriteString(out, "/>")
			return nil
		}
	}

	_, _ = io.WriteString(out, ">")

	if child := n.FirstChild(); child != nil {
		for ; child != nil; child = child.NextSibling() {
			if err := d.DumpNode(out, child); err != nil {
				return err
			}
		}
	}

	_, _ = io.WriteString(out, "</")
	_, _ = io.WriteString(out, name)
	_, _ = io.WriteString(out, ">")

	return nil
}
