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

	// The element name is emitted verbatim here and on the closing tag below.
	// Validate it just like writeNode so an injected name (e.g. from
	// CreateElement) cannot inject raw markup through the XHTML path.
	if !d.checkElementName(name) {
		return d.err
	}

	d.writeString(out, "<")
	d.writeString(out, name)

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
		d.writeString(out, ` xmlns="http://www.w3.org/1999/xhtml"`)
	}

	// dumpXHTMLAttrList returns a non-nil error (e.g. an invalid/reserved
	// attribute name) when it stopped early. Abort here, mirroring writeNode,
	// so no element body, child content, or closing tag is emitted past the
	// error.
	if err := d.dumpXHTMLAttrList(out, e); err != nil {
		return err
	}

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
			d.writeString(out, " />")
		} else {
			if addMeta {
				d.writeString(out, ">")
				if d.format {
					d.writeString(out, "\n")
					d.indent++
					d.writeIndent(out)
				}
				d.writeMetaContentType(out)
				if d.format {
					d.writeString(out, "\n")
					d.indent--
					d.writeIndent(out)
				}
			} else {
				d.writeString(out, ">")
			}
			d.writeString(out, "</")
			d.writeString(out, name)
			d.writeString(out, ">")
		}
		return d.err
	}

	d.writeString(out, ">")

	textOnly := d.format && hasOnlyTextChildren(e)
	if d.format && !textOnly {
		d.writeString(out, "\n")
		d.indent++
	}

	if addMeta {
		if d.format && !textOnly {
			d.writeIndent(out)
		}
		d.writeMetaContentType(out)
		if d.format && !textOnly {
			d.writeString(out, "\n")
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
			d.writeString(out, "\n")
		}
	}

	if d.format && !textOnly {
		d.indent--
		d.writeIndent(out)
	}

	d.writeString(out, "</")
	d.writeString(out, name)
	d.writeString(out, ">")
	return d.err
}

func (d *writeSession) dumpXHTMLAttrList(out io.Writer, e *Element) error {
	var langAttr, xmlLangAttr, nameAttr, idAttr *Attribute
	localName := e.LocalName()

	for attr := e.properties; attr != nil; attr = attr.NextAttribute() {
		attrName := attr.Name()

		// The attribute name is emitted verbatim below. Validate it just like
		// writeNode so an injected name cannot inject raw markup. Stop on the
		// first invalid name and return the sticky error so the caller aborts
		// before emitting any element body or child content.
		if !d.checkAttributeName(attrName) {
			return d.err
		}

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

		d.writeString(out, " ")
		d.writeString(out, attrName)
		d.writeString(out, `="`)

		attrValue := attr.Value()
		if attrValue == "" && htmlBooleanAttrs[attrName] {
			d.writeString(out, attrName)
		} else {
			for achld := range Children(attr) {
				if achld.Type() == TextNode {
					// Read-only escape pass: use the internal slice without a copy.
					d.check(escapeAttrValue(out, rawContent(achld), d.escapeNonASCII, d.rejectInvalidChars, d.xml11, nil))
				} else {
					d.check(d.writeNode(out, achld))
				}
			}
		}
		d.writeString(out, `"`)
	}

	if nameAttr != nil && idAttr == nil && xhtmlNameIDElements[localName] {
		d.writeString(out, ` id="`)
		d.check(escapeAttrValue(out, nameAttr.Content(), d.escapeNonASCII, d.rejectInvalidChars, d.xml11, nil))
		d.writeString(out, `"`)
	}

	if langAttr != nil && xmlLangAttr == nil {
		d.writeString(out, ` xml:lang="`)
		d.check(escapeAttrValue(out, langAttr.Content(), d.escapeNonASCII, d.rejectInvalidChars, d.xml11, nil))
		d.writeString(out, `"`)
	} else if xmlLangAttr != nil && langAttr == nil {
		d.writeString(out, ` lang="`)
		d.check(escapeAttrValue(out, xmlLangAttr.Content(), d.escapeNonASCII, d.rejectInvalidChars, d.xml11, nil))
		d.writeString(out, `"`)
	}
	return d.err
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
		for attr := ce.properties; attr != nil; attr = attr.NextAttribute() {
			if attr.ns == nil && strings.EqualFold(attr.Name(), "http-equiv") {
				if strings.EqualFold(attr.Value(), "Content-Type") {
					return true
				}
			}
		}
	}
	return false
}

func (d *writeSession) writeMetaContentType(out io.Writer) {
	enc := d.encoding
	if enc == "" {
		enc = "UTF-8"
	}
	d.writeString(out, `<meta http-equiv="Content-Type" content="text/html; charset=`)
	d.writeString(out, enc)
	d.writeString(out, `" />`)
}
