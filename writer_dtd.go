package helium

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/pdebug"
)

func dtdQuoteChar(value string) byte {
	if strings.ContainsRune(value, '"') {
		return '\''
	}
	return '"'
}

func (d *writeSession) dumpDTD(out io.Writer, n Node) error {
	dtd, ok := AsType[*DTD](n)
	if !ok {
		return nil
	}
	_, _ = io.WriteString(out, "<!DOCTYPE ")
	_, _ = io.WriteString(out, dtd.Name())

	if dtd.externalID != "" {
		pubQ := dtdQuoteChar(dtd.externalID)
		sysQ := dtdQuoteChar(dtd.systemID)
		_, _ = fmt.Fprintf(out, " PUBLIC %c%s%c %c%s%c", pubQ, dtd.externalID, pubQ, sysQ, dtd.systemID, sysQ)
	} else if dtd.systemID != "" {
		sysQ := dtdQuoteChar(dtd.systemID)
		_, _ = fmt.Fprintf(out, " SYSTEM %c%s%c", sysQ, dtd.systemID, sysQ)
	}

	if len(dtd.entities) == 0 && len(dtd.elements) == 0 && len(dtd.pentities) == 0 && len(dtd.attributes) == 0 && len(dtd.notations) == 0 {
		_, _ = io.WriteString(out, ">")
		return nil
	}

	_, _ = io.WriteString(out, " [\n")

	savedFormat := d.format
	savedIndent := d.indent
	d.format = false
	d.indent = -1

	for e := range Children(dtd) {
		if err := d.writeNode(out, e); err != nil {
			d.format = savedFormat
			d.indent = savedIndent
			return err
		}
	}

	d.format = savedFormat
	d.indent = savedIndent

	_, _ = io.WriteString(out, "]>")
	return nil
}

func (d *writeSession) dumpEnumeration(out io.Writer, n Enumeration) error {
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

func dumpElementDeclPrologue(out io.Writer, n *ElementDecl) {
	_, _ = io.WriteString(out, "<!ELEMENT ")
	if n.prefix != "" {
		_, _ = io.WriteString(out, n.prefix)
		_, _ = io.WriteString(out, ":")
	}
	_, _ = io.WriteString(out, n.name)
}

func dumpElementContent(out io.Writer, n *ElementContent, glob bool) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Writer.dumpElementContent n = '%s'", n.name)
		defer g.IRelease("END Writer.dumpElementContent")
	}
	if n == nil {
		return nil
	}

	if glob {
		_, _ = io.WriteString(out, "(")
	}

	switch n.ctype {
	case ElementContentPCDATA:
		_, _ = io.WriteString(out, "#PCDATA")
	case ElementContentElement:
		if n.prefix != "" {
			_, _ = io.WriteString(out, n.prefix)
			_, _ = io.WriteString(out, ":")
		}
		_, _ = io.WriteString(out, n.name)
	case ElementContentSeq:
		switch n.c1.ctype {
		case ElementContentOr, ElementContentSeq:
			if err := dumpElementContent(out, n.c1, true); err != nil {
				return err
			}
		default:
			if err := dumpElementContent(out, n.c1, false); err != nil {
				return err
			}
		}
		_, _ = io.WriteString(out, " , ")

		if ctype := n.c2.ctype; ctype == ElementContentOr || (ctype == ElementContentSeq && n.c2.coccur != ElementContentOnce) {
			if err := dumpElementContent(out, n.c2, true); err != nil {
				return err
			}
		} else {
			if err := dumpElementContent(out, n.c2, false); err != nil {
				return err
			}
		}
	case ElementContentOr:
		switch n.c1.ctype {
		case ElementContentOr, ElementContentSeq:
			if err := dumpElementContent(out, n.c1, true); err != nil {
				return err
			}
		default:
			if err := dumpElementContent(out, n.c1, false); err != nil {
				return err
			}
		}
		_, _ = io.WriteString(out, " | ")

		if ctype := n.c2.ctype; ctype == ElementContentSeq || (ctype == ElementContentOr && n.c2.coccur != ElementContentOnce) {
			if err := dumpElementContent(out, n.c2, true); err != nil {
				return err
			}
		} else {
			if err := dumpElementContent(out, n.c2, false); err != nil {
				return err
			}
		}
	default:
		return errors.New("invalid ElementContent")
	}

	if glob {
		_, _ = io.WriteString(out, ")")
	}

	switch n.coccur {
	case ElementContentOnce:
	case ElementContentOpt:
		_, _ = io.WriteString(out, "?")
	case ElementContentMult:
		_, _ = io.WriteString(out, "*")
	case ElementContentPlus:
		_, _ = io.WriteString(out, "+")
	}

	return nil
}

func dumpEntityContent(out io.Writer, content string) error {
	if strings.IndexByte(content, '%') == -1 {
		if err := dumpQuotedString(out, content); err != nil {
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

func (d *writeSession) dumpEntityDecl(out io.Writer, ent *Entity) error {
	if ent == nil {
		return nil
	}

	switch etype := ent.entityType; etype {
	case enum.InternalGeneralEntity:
		_, _ = io.WriteString(out, "<!ENTITY ")
		_, _ = io.WriteString(out, ent.name)
		_, _ = io.WriteString(out, " ")
		if ent.orig != "" {
			if err := dumpQuotedString(out, ent.orig); err != nil {
				return err
			}
		} else {
			if err := dumpEntityContent(out, ent.content); err != nil {
				return err
			}
		}
		_, _ = io.WriteString(out, ">\n")
	case enum.ExternalGeneralParsedEntity, enum.ExternalGeneralUnparsedEntity:
		_, _ = io.WriteString(out, "<!ENTITY ")
		_, _ = io.WriteString(out, ent.name)
		if ent.externalID != "" {
			_, _ = io.WriteString(out, " PUBLIC ")
			_ = dumpQuotedString(out, ent.externalID)
			_, _ = io.WriteString(out, " ")
			_ = dumpQuotedString(out, ent.systemID)
		} else {
			_, _ = io.WriteString(out, " SYSTEM ")
			_ = dumpQuotedString(out, ent.systemID)
		}

		if etype == enum.ExternalGeneralUnparsedEntity {
			if ent.content != "" {
				_, _ = io.WriteString(out, " NDATA ")
				if ent.orig != "" {
					_, _ = io.WriteString(out, ent.orig)
				} else {
					_, _ = io.WriteString(out, ent.content)
				}
			}
		}
		_, _ = io.WriteString(out, ">\n")
	case enum.InternalParameterEntity:
		_, _ = io.WriteString(out, "<!ENTITY % ")
		_, _ = io.WriteString(out, ent.name)
		_, _ = io.WriteString(out, " ")
		if ent.orig != "" {
			if err := dumpQuotedString(out, ent.orig); err != nil {
				return err
			}
		} else {
			if err := dumpEntityContent(out, ent.content); err != nil {
				return err
			}
		}
		_, _ = io.WriteString(out, ">\n")
	case enum.ExternalParameterEntity:
		_, _ = io.WriteString(out, "<!ENTITY % ")
		_, _ = io.WriteString(out, ent.name)
		if ent.externalID != "" {
			_, _ = io.WriteString(out, " PUBLIC ")
			_ = dumpQuotedString(out, ent.externalID)
			_, _ = io.WriteString(out, " ")
			_ = dumpQuotedString(out, ent.systemID)
		} else {
			_, _ = io.WriteString(out, " SYSTEM ")
			_ = dumpQuotedString(out, ent.systemID)
		}
	default:
		return errors.New("invalid entity type")
	}
	return nil
}

func (d *writeSession) dumpNotationDecl(out io.Writer, n *Notation) error { //nolint:unparam // always nil but matches other dump methods
	_, _ = io.WriteString(out, "<!NOTATION ")
	_, _ = io.WriteString(out, n.name)
	if n.publicID != "" {
		_, _ = io.WriteString(out, " PUBLIC ")
		_ = dumpQuotedString(out, n.publicID)
		if n.systemID != "" {
			_, _ = io.WriteString(out, " ")
			_ = dumpQuotedString(out, n.systemID)
		}
	} else {
		_, _ = io.WriteString(out, " SYSTEM ")
		_ = dumpQuotedString(out, n.systemID)
	}
	_, _ = io.WriteString(out, " >\n")
	return nil
}

func (d *writeSession) dumpElementDecl(out io.Writer, n *ElementDecl) error {
	switch n.decltype {
	case enum.EmptyElementType:
		dumpElementDeclPrologue(out, n)
		_, _ = io.WriteString(out, " EMPTY>\n")
	case enum.AnyElementType:
		dumpElementDeclPrologue(out, n)
		_, _ = io.WriteString(out, " ANY>\n")
	case enum.MixedElementType, enum.ElementElementType:
		dumpElementDeclPrologue(out, n)
		_, _ = io.WriteString(out, " ")
		if err := dumpElementContent(out, n.content, true); err != nil {
			return err
		}
		_, _ = io.WriteString(out, ">\n")
	default:
		return errors.New("invalid element decl")
	}
	return nil
}

func (d *writeSession) dumpAttributeDecl(out io.Writer, n *AttributeDecl) error {
	_, _ = io.WriteString(out, "<!ATTLIST ")
	_, _ = io.WriteString(out, n.elem)
	_, _ = io.WriteString(out, " ")
	if n.prefix != "" {
		_, _ = io.WriteString(out, n.prefix)
		_, _ = io.WriteString(out, ":")
	}
	_, _ = io.WriteString(out, n.name)
	switch n.atype {
	case enum.AttrCDATA:
		_, _ = io.WriteString(out, " CDATA")
	case enum.AttrID:
		_, _ = io.WriteString(out, " ID")
	case enum.AttrIDRef:
		_, _ = io.WriteString(out, " IDREF")
	case enum.AttrIDRefs:
		_, _ = io.WriteString(out, " IDREFS")
	case enum.AttrEntity:
		_, _ = io.WriteString(out, " ENTITY")
	case enum.AttrEntities:
		_, _ = io.WriteString(out, " ENTITIES")
	case enum.AttrNmtoken:
		_, _ = io.WriteString(out, " NMTOKEN")
	case enum.AttrNmtokens:
		_, _ = io.WriteString(out, " NMTOKENS")
	case enum.AttrEnumeration:
		_, _ = io.WriteString(out, " (")
		if err := d.dumpEnumeration(out, n.tree); err != nil {
			return err
		}
	case enum.AttrNotation:
		_, _ = io.WriteString(out, " NOTATION (")
		if err := d.dumpEnumeration(out, n.tree); err != nil {
			return err
		}
	default:
		return errors.New("invalid AttributeDecl type")
	}

	switch n.def {
	case enum.AttrDefaultNone:
	case enum.AttrDefaultRequired:
		_, _ = io.WriteString(out, " #REQUIRED")
	case enum.AttrDefaultImplied:
		_, _ = io.WriteString(out, " #IMPLIED")
	case enum.AttrDefaultFixed:
		_, _ = io.WriteString(out, " #FIXED")
	default:
		return errors.New("invalid AttributeDecl default value type")
	}

	if n.defvalue != "" {
		_, _ = io.WriteString(out, ` "`)
		_ = escapeAttrValue(out, []byte(n.defvalue), d.escapeNonASCII)
		_, _ = io.WriteString(out, `"`)
	}
	_, _ = io.WriteString(out, ">\n")
	return nil
}

func (d *writeSession) dumpNsList(out io.Writer, nslist []*Namespace) error {
	for _, ns := range nslist {
		if err := d.dumpNs(out, ns); err != nil {
			return err
		}
	}
	return nil
}

func (d *writeSession) dumpNs(out io.Writer, ns *Namespace) error {
	if ns.href == "" && ns.prefix != "" {
		if !d.allowPrefixUndecl {
			return nil
		}
	}
	if ns.href == "" && ns.prefix == "" {
		_, err := io.WriteString(out, ` xmlns=""`)
		return err
	}

	if ns.prefix == "xml" {
		return nil
	}

	if _, err := io.WriteString(out, " "); err != nil {
		return err
	}

	if ns.prefix == "" {
		if _, err := io.WriteString(out, "xmlns"); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(out, "xmlns:"); err != nil {
			return err
		}
		if _, err := io.WriteString(out, ns.prefix); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, `="`); err != nil {
		return err
	}
	if err := escapeAttrValue(out, []byte(ns.href), d.escapeNonASCII); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"`); err != nil {
		return err
	}
	return nil
}
