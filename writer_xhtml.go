package helium

import (
	"io"
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

func isXHTMLDTD(dtd *DTD) bool {
	switch dtd.externalID {
	case "-//W3C//DTD XHTML 1.0 Strict//EN",
		"-//W3C//DTD XHTML 1.0 Transitional//EN",
		"-//W3C//DTD XHTML 1.0 Frameset//EN":
		return true
	}
	switch dtd.systemID {
	case "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd",
		"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd",
		"http://www.w3.org/TR/xhtml1/DTD/xhtml1-frameset.dtd":
		return true
	}
	return false
}

var xhtmlVoidElements = map[string]bool{
	"area": true, "base": true, "basefont": true, "br": true,
	"col": true, "frame": true, "hr": true, "img": true,
	"input": true, "isindex": true, "link": true, "meta": true,
	"param": true,
}

var xhtmlNameIDElements = map[string]bool{
	"a": true, "p": true, "div": true, "img": true,
	"map": true, "applet": true, "form": true, "frame": true,
	"iframe": true,
}

var htmlBooleanAttrs = map[string]bool{
	"checked": true, "compact": true, "declare": true, "defer": true,
	"disabled": true, "ismap": true, "multiple": true, "nohref": true,
	"noresize": true, "noshade": true, "nowrap": true, "readonly": true,
	"selected": true,
}

func (d *writeSession) dumpXHTMLNode(out io.Writer, n Node) error {
	switch n.Type() {
	case ElementNode:
	default:
		return d.writeNode(out, n)
	}

	e, ok := AsNode[*Element](n)
	if !ok {
		return d.writeNode(out, n)
	}
	localName := e.LocalName()

	var name string
	if nser, ok := n.(Namespacer); ok {
		if prefix := nser.Prefix(); prefix != "" {
			name = prefix + ":" + localName
		} else {
			name = localName
		}
	} else {
		name = n.Name()
	}

	_, _ = io.WriteString(out, "<")
	_, _ = io.WriteString(out, name)

	nslist := e.Namespaces()
	if len(nslist) > 0 {
		if err := d.dumpNsList(out, nslist); err != nil {
			return err
		}
	}

	hasDefaultNs := false
	for _, ns := range nslist {
		if ns.prefix == "" {
			hasDefaultNs = true
			break
		}
	}
	if localName == "html" && e.ns == nil && !hasDefaultNs {
		_, _ = io.WriteString(out, ` xmlns="http://www.w3.org/1999/xhtml"`)
	}

	d.dumpXHTMLAttrList(out, e)

	addMeta := false
	if localName == "head" {
		if parent := e.parent; parent != nil {
			if pe, ok := parent.(*Element); ok {
				if pe.LocalName() == "html" && pe.parent != nil && pe.parent.Type() == DocumentNode {
					addMeta = !d.headHasContentTypeMeta(e)
				}
			}
		}
	}

	if e.FirstChild() == nil {
		if (e.ns == nil || e.ns.Prefix() == "") && xhtmlVoidElements[localName] && !addMeta {
			_, _ = io.WriteString(out, " />")
		} else {
			if addMeta {
				_, _ = io.WriteString(out, ">")
				if d.format {
					_, _ = io.WriteString(out, "\n")
					d.indent++
					d.writeIndent(out)
				}
				d.writeMetaContentType(out)
				if d.format {
					_, _ = io.WriteString(out, "\n")
					d.indent--
					d.writeIndent(out)
				}
			} else {
				_, _ = io.WriteString(out, ">")
			}
			_, _ = io.WriteString(out, "</")
			_, _ = io.WriteString(out, name)
			_, _ = io.WriteString(out, ">")
		}
		return nil
	}

	_, _ = io.WriteString(out, ">")

	textOnly := d.format && hasOnlyTextChildren(e)
	if d.format && !textOnly {
		_, _ = io.WriteString(out, "\n")
		d.indent++
	}

	if addMeta {
		if d.format && !textOnly {
			d.writeIndent(out)
		}
		d.writeMetaContentType(out)
		if d.format && !textOnly {
			_, _ = io.WriteString(out, "\n")
		}
	}

	for child := range Children(e) {
		if d.format && !textOnly {
			d.writeIndent(out)
		}
		if child.Type() == ElementNode {
			if err := d.dumpXHTMLNode(out, child); err != nil {
				return err
			}
		} else {
			if err := d.writeNode(out, child); err != nil {
				return err
			}
		}
		if d.format && !textOnly {
			_, _ = io.WriteString(out, "\n")
		}
	}

	if d.format && !textOnly {
		d.indent--
		d.writeIndent(out)
	}

	_, _ = io.WriteString(out, "</")
	_, _ = io.WriteString(out, name)
	_, _ = io.WriteString(out, ">")
	return nil
}

func (d *writeSession) dumpXHTMLAttrList(out io.Writer, e *Element) {
	var langAttr, xmlLangAttr, nameAttr, idAttr *Attribute
	localName := e.LocalName()

	for attr := e.properties; attr != nil; {
		attrName := attr.Name()

		switch attrName {
		case "id":
			idAttr = attr
		case "name":
			nameAttr = attr
		case "lang":
			langAttr = attr
		case lexicon.QNameXMLLang:
			xmlLangAttr = attr
		}

		_, _ = io.WriteString(out, " ")
		_, _ = io.WriteString(out, attrName)
		_, _ = io.WriteString(out, `="`)

		attrValue := attr.Value()
		if attrValue == "" && htmlBooleanAttrs[attrName] {
			_, _ = io.WriteString(out, attrName)
		} else {
			for achld := range Children(attr) {
				if achld.Type() == TextNode {
					_ = escapeAttrValue(out, achld.Content(), d.escapeNonASCII)
				} else {
					_ = d.writeNode(out, achld)
				}
			}
		}
		_, _ = io.WriteString(out, `"`)

		next, ok := AsNode[*Attribute](attr.NextSibling())
		if !ok {
			break
		}
		attr = next
	}

	if nameAttr != nil && idAttr == nil && xhtmlNameIDElements[localName] {
		_, _ = io.WriteString(out, ` id="`)
		_ = escapeAttrValue(out, nameAttr.Content(), d.escapeNonASCII)
		_, _ = io.WriteString(out, `"`)
	}

	if langAttr != nil && xmlLangAttr == nil {
		_, _ = io.WriteString(out, ` xml:lang="`)
		_ = escapeAttrValue(out, langAttr.Content(), d.escapeNonASCII)
		_, _ = io.WriteString(out, `"`)
	} else if xmlLangAttr != nil && langAttr == nil {
		_, _ = io.WriteString(out, ` lang="`)
		_ = escapeAttrValue(out, xmlLangAttr.Content(), d.escapeNonASCII)
		_, _ = io.WriteString(out, `"`)
	}
}

func (d *writeSession) headHasContentTypeMeta(head *Element) bool {
	for child := range Children(head) {
		if child.Type() != ElementNode {
			continue
		}
		ce, ok := child.(*Element)
		if !ok || ce.LocalName() != "meta" {
			continue
		}
		for attr := ce.properties; attr != nil; {
			if attr.ns == nil && strings.EqualFold(attr.Name(), "http-equiv") {
				if strings.EqualFold(attr.Value(), "Content-Type") {
					return true
				}
			}
			next, ok := AsNode[*Attribute](attr.NextSibling())
			if !ok {
				break
			}
			attr = next
		}
	}
	return false
}

func (d *writeSession) writeMetaContentType(out io.Writer) {
	enc := d.encoding
	if enc == "" {
		enc = "UTF-8"
	}
	_, _ = io.WriteString(out, `<meta http-equiv="Content-Type" content="text/html; charset=`)
	_, _ = io.WriteString(out, enc)
	_, _ = io.WriteString(out, `" />`)
}
