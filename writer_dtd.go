package helium

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/lestrrat-go/helium/enum"
)

func dtdQuoteChar(value string) byte {
	if strings.ContainsRune(value, '"') {
		return '\''
	}
	return '"'
}

func (d *writeSession) dumpDTD(out io.Writer, n Node) error {
	dtd, ok := AsNode[*DTD](n)
	if !ok {
		return nil
	}
	d.writeString(out, "<!DOCTYPE ")
	// A DOCTYPE name is emitted verbatim and cannot hold a character reference, so
	// a non-ASCII name has no faithful US-ASCII serialization. Guard before the
	// write so no raw octet leaks ahead of the sticky error.
	if d.rejectNonASCIIStr("DOCTYPE name", dtd.Name()) {
		return d.err
	}
	d.writeString(out, dtd.Name())

	if d.err == nil && dtd.externalID != "" {
		pubQ := dtdQuoteChar(dtd.externalID)
		sysQ := dtdQuoteChar(dtd.systemID)
		d.writeString(out, fmt.Sprintf(" PUBLIC %c%s%c %c%s%c", pubQ, dtd.externalID, pubQ, sysQ, dtd.systemID, sysQ))
	} else if d.err == nil && dtd.systemID != "" {
		sysQ := dtdQuoteChar(dtd.systemID)
		d.writeString(out, fmt.Sprintf(" SYSTEM %c%s%c", sysQ, dtd.systemID, sysQ))
	}

	if len(dtd.entities) == 0 && len(dtd.elements) == 0 && len(dtd.pentities) == 0 && len(dtd.attributes) == 0 && len(dtd.notations) == 0 {
		d.writeString(out, ">")
		return d.err
	}

	d.writeString(out, " [\n")

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

	d.writeString(out, "]>")
	return d.err
}

func (d *writeSession) dumpEnumeration(out io.Writer, n Enumeration) error {
	l := len(n)
	for i, v := range n {
		// An enumeration token (NMTOKEN / NOTATION name) is emitted verbatim and
		// cannot hold a character reference, so a non-ASCII token has no faithful
		// US-ASCII serialization. Guard before the write.
		if d.rejectNonASCIIStr("enumeration token", v) {
			return d.err
		}
		d.writeString(out, v)
		if i != l-1 {
			d.writeString(out, " | ")
		}
	}
	d.writeString(out, ")")
	return d.err
}

// writePrefixedName emits an optionally namespace-prefixed name as
// "prefix:name" (or just "name" when prefix is empty), preserving the
// writeString order so the sticky-error handling is identical to the
// inline sequences it replaces.
func (d *writeSession) writePrefixedName(out io.Writer, prefix, name string) {
	// The prefix and name are emitted verbatim and cannot hold a character
	// reference, so a non-ASCII component has no faithful US-ASCII serialization.
	// Guard before the first write so no raw octet leaks ahead of the sticky
	// error. This is the shared chokepoint for <!ELEMENT>/<!ATTLIST> names and
	// element-content child names.
	if d.rejectNonASCIIStr("DTD declaration name", prefix) || d.rejectNonASCIIStr("DTD declaration name", name) {
		return
	}
	if prefix != "" {
		d.writeString(out, prefix)
		d.writeString(out, ":")
	}
	d.writeString(out, name)
}

func (d *writeSession) dumpElementDeclPrologue(out io.Writer, n *ElementDecl) {
	d.writeString(out, "<!ELEMENT ")
	d.writePrefixedName(out, n.prefix, n.name)
}

func (d *writeSession) dumpElementContent(out io.Writer, n *ElementContent, glob bool) error {
	if n == nil {
		return nil
	}

	if glob {
		d.writeString(out, "(")
	}

	switch n.ctype {
	case ElementContentPCDATA:
		d.writeString(out, "#PCDATA")
	case ElementContentElement:
		d.writePrefixedName(out, n.prefix, n.name)
	case ElementContentSeq:
		switch n.c1.ctype {
		case ElementContentOr, ElementContentSeq:
			if err := d.dumpElementContent(out, n.c1, true); err != nil {
				return err
			}
		default:
			if err := d.dumpElementContent(out, n.c1, false); err != nil {
				return err
			}
		}
		d.writeString(out, " , ")

		if ctype := n.c2.ctype; ctype == ElementContentOr || (ctype == ElementContentSeq && n.c2.coccur != ElementContentOnce) {
			if err := d.dumpElementContent(out, n.c2, true); err != nil {
				return err
			}
		} else {
			if err := d.dumpElementContent(out, n.c2, false); err != nil {
				return err
			}
		}
	case ElementContentOr:
		switch n.c1.ctype {
		case ElementContentOr, ElementContentSeq:
			if err := d.dumpElementContent(out, n.c1, true); err != nil {
				return err
			}
		default:
			if err := d.dumpElementContent(out, n.c1, false); err != nil {
				return err
			}
		}
		d.writeString(out, " | ")

		if ctype := n.c2.ctype; ctype == ElementContentSeq || (ctype == ElementContentOr && n.c2.coccur != ElementContentOnce) {
			if err := d.dumpElementContent(out, n.c2, true); err != nil {
				return err
			}
		} else {
			if err := d.dumpElementContent(out, n.c2, false); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("invalid ElementContent: %w", ErrWriterInvalidDTDNode)
	}

	if glob {
		d.writeString(out, ")")
	}

	switch n.coccur {
	case ElementContentOnce:
	case ElementContentOpt:
		d.writeString(out, "?")
	case ElementContentMult:
		d.writeString(out, "*")
	case ElementContentPlus:
		d.writeString(out, "+")
	}

	return d.err
}

func (d *writeSession) dumpEntityContent(out io.Writer, content string) error {
	if strings.IndexByte(content, '%') == -1 {
		if err := dumpQuotedString(out, content); err != nil {
			d.check(err)
			return d.err
		}
		return d.err
	}

	d.writeString(out, `"`)
	rdr := strings.NewReader(content)
	buf := bytes.Buffer{}
	for rdr.Len() > 0 {
		c, err := rdr.ReadByte()
		if err != nil {
			d.check(err)
			return d.err
		}
		switch c {
		case '"':
			if buf.Len() > 0 {
				d.writeBytes(out, buf.Bytes())
				buf.Reset()
			}
			d.writeString(out, "&quot;")
		case '%':
			if buf.Len() > 0 {
				d.writeBytes(out, buf.Bytes())
				buf.Reset()
			}
			d.writeString(out, "&#x25;")
		default:
			_ = buf.WriteByte(c)
		}
	}
	if buf.Len() > 0 {
		d.writeBytes(out, buf.Bytes())
	}
	d.writeString(out, `"`)

	return d.err
}

func (d *writeSession) dumpEntityDecl(out io.Writer, ent *Entity) error {
	if ent == nil {
		return nil
	}

	// The entity name is emitted verbatim in every branch below and cannot hold a
	// character reference, so a non-ASCII name has no faithful US-ASCII
	// serialization. Guard once before any write so no raw octet leaks ahead of
	// the sticky error.
	if d.rejectNonASCIIStr("entity name", ent.name) {
		return d.err
	}

	switch etype := ent.entityType; etype {
	case enum.InternalGeneralEntity:
		d.writeString(out, "<!ENTITY ")
		d.writeString(out, ent.name)
		d.writeString(out, " ")
		if ent.orig != "" {
			if err := dumpQuotedString(out, ent.orig); err != nil {
				d.check(err)
				return d.err
			}
		} else {
			if err := d.dumpEntityContent(out, ent.content); err != nil {
				return err
			}
		}
		d.writeString(out, ">\n")
	case enum.ExternalGeneralParsedEntity, enum.ExternalGeneralUnparsedEntity:
		d.writeString(out, "<!ENTITY ")
		d.writeString(out, ent.name)
		if ent.externalID != "" {
			d.writeString(out, " PUBLIC ")
			d.check(dumpQuotedString(out, ent.externalID))
			d.writeString(out, " ")
			d.check(dumpQuotedString(out, ent.systemID))
		} else {
			d.writeString(out, " SYSTEM ")
			d.check(dumpQuotedString(out, ent.systemID))
		}

		if etype == enum.ExternalGeneralUnparsedEntity {
			if ent.content != "" {
				d.writeString(out, " NDATA ")
				if ent.orig != "" {
					d.writeString(out, ent.orig)
				} else {
					d.writeString(out, ent.content)
				}
			}
		}
		d.writeString(out, ">\n")
	case enum.InternalParameterEntity:
		d.writeString(out, "<!ENTITY % ")
		d.writeString(out, ent.name)
		d.writeString(out, " ")
		if ent.orig != "" {
			if err := dumpQuotedString(out, ent.orig); err != nil {
				d.check(err)
				return d.err
			}
		} else {
			if err := d.dumpEntityContent(out, ent.content); err != nil {
				return err
			}
		}
		d.writeString(out, ">\n")
	case enum.ExternalParameterEntity:
		d.writeString(out, "<!ENTITY % ")
		d.writeString(out, ent.name)
		if ent.externalID != "" {
			d.writeString(out, " PUBLIC ")
			d.check(dumpQuotedString(out, ent.externalID))
			d.writeString(out, " ")
			d.check(dumpQuotedString(out, ent.systemID))
		} else {
			d.writeString(out, " SYSTEM ")
			d.check(dumpQuotedString(out, ent.systemID))
		}
	default:
		return fmt.Errorf("invalid entity type: %w", ErrWriterInvalidDTDNode)
	}
	return d.err
}

func (d *writeSession) dumpNotationDecl(out io.Writer, n *Notation) error {
	// A notation name cannot hold a character reference, so a non-ASCII name has
	// no faithful US-ASCII serialization.
	if d.rejectNonASCIIStr("notation name", n.name) {
		return d.err
	}
	d.writeString(out, "<!NOTATION ")
	d.writeString(out, n.name)
	if n.publicID != "" {
		d.writeString(out, " PUBLIC ")
		d.check(dumpQuotedString(out, n.publicID))
		if n.systemID != "" {
			d.writeString(out, " ")
			d.check(dumpQuotedString(out, n.systemID))
		}
	} else {
		d.writeString(out, " SYSTEM ")
		d.check(dumpQuotedString(out, n.systemID))
	}
	d.writeString(out, " >\n")
	return d.err
}

func (d *writeSession) dumpElementDecl(out io.Writer, n *ElementDecl) error {
	switch n.decltype {
	case enum.EmptyElementType:
		d.dumpElementDeclPrologue(out, n)
		d.writeString(out, " EMPTY>\n")
	case enum.AnyElementType:
		d.dumpElementDeclPrologue(out, n)
		d.writeString(out, " ANY>\n")
	case enum.MixedElementType, enum.ElementElementType:
		d.dumpElementDeclPrologue(out, n)
		d.writeString(out, " ")
		if err := d.dumpElementContent(out, n.content, true); err != nil {
			return err
		}
		d.writeString(out, ">\n")
	default:
		return fmt.Errorf("invalid element decl: %w", ErrWriterInvalidDTDNode)
	}
	return d.err
}

func (d *writeSession) dumpAttributeDecl(out io.Writer, n *AttributeDecl) error {
	d.writeString(out, "<!ATTLIST ")
	// The element name is emitted verbatim and cannot hold a character reference,
	// so a non-ASCII name has no faithful US-ASCII serialization. Guard before the
	// write. (The attribute name/prefix are guarded inside writePrefixedName.)
	if d.rejectNonASCIIStr("attribute-list element name", n.elem) {
		return d.err
	}
	d.writeString(out, n.elem)
	d.writeString(out, " ")
	d.writePrefixedName(out, n.prefix, n.name)
	switch n.atype {
	case enum.AttrCDATA:
		d.writeString(out, " CDATA")
	case enum.AttrID:
		d.writeString(out, " ID")
	case enum.AttrIDRef:
		d.writeString(out, " IDREF")
	case enum.AttrIDRefs:
		d.writeString(out, " IDREFS")
	case enum.AttrEntity:
		d.writeString(out, " ENTITY")
	case enum.AttrEntities:
		d.writeString(out, " ENTITIES")
	case enum.AttrNmtoken:
		d.writeString(out, " NMTOKEN")
	case enum.AttrNmtokens:
		d.writeString(out, " NMTOKENS")
	case enum.AttrEnumeration:
		d.writeString(out, " (")
		if err := d.dumpEnumeration(out, n.tree); err != nil {
			return err
		}
	case enum.AttrNotation:
		d.writeString(out, " NOTATION (")
		if err := d.dumpEnumeration(out, n.tree); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid AttributeDecl type: %w", ErrWriterInvalidDTDNode)
	}

	switch n.def {
	case enum.AttrDefaultNone:
	case enum.AttrDefaultRequired:
		d.writeString(out, " #REQUIRED")
	case enum.AttrDefaultImplied:
		d.writeString(out, " #IMPLIED")
	case enum.AttrDefaultFixed:
		d.writeString(out, " #FIXED")
	default:
		return fmt.Errorf("invalid AttributeDecl default value type: %w", ErrWriterInvalidDTDNode)
	}

	if n.defvalue != "" {
		d.writeString(out, ` "`)
		d.check(escapeAttrValue(out, []byte(n.defvalue), d.escapeNonASCII, d.asciiOutput, d.asciiReject(), d.rejectInvalidChars, d.xml11, nil))
		d.writeString(out, `"`)
	}
	d.writeString(out, ">\n")
	return d.err
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

	// The prefix is emitted verbatim as "xmlns:"+prefix below. Reject any
	// prefix that is not a valid NCName so a crafted prefix (whitespace,
	// quotes, '>') cannot inject raw markup into the start tag.
	if !d.checkNamespacePrefix(ns.prefix) {
		return d.err
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
	if err := escapeAttrValue(out, []byte(ns.href), d.escapeNonASCII, d.asciiOutput, d.asciiReject(), d.rejectInvalidChars, d.xml11, nil); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"`); err != nil {
		return err
	}
	return nil
}
