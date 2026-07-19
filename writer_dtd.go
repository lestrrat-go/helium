package helium

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

func dtdQuoteChar(value string) byte {
	if strings.ContainsRune(value, '"') {
		return '\''
	}
	return '"'
}

// checkPubid records a sticky error when s carries a character outside the
// PubidChar production (isPubidChar), i.e. a public identifier that cannot be
// serialized as a well-formed PubidLiteral. It is the shared check for the
// DOCTYPE and notation public-identifier paths. Returns true when it recorded
// the error.
func (d *writeSession) checkPubid(s string) bool {
	for _, r := range s {
		if !isPubidChar(r) {
			d.check(fmt.Errorf("helium: public identifier %q contains an invalid PubidChar: %w", s, ErrWriterInvalidDTDNode))
			return true
		}
	}
	return false
}

// checkSystemLiteral records a sticky error when s contains BOTH quote
// characters and therefore cannot be delimited as a SystemLiteral (which admits
// no character references). Returns true when it recorded the error.
func (d *writeSession) checkSystemLiteral(s string) bool {
	if strings.ContainsRune(s, '"') && strings.ContainsRune(s, '\'') {
		d.check(fmt.Errorf("helium: system literal %q contains both quote characters: %w", s, ErrWriterInvalidDTDNode))
		return true
	}
	return false
}

func (d *writeSession) dumpDTD(out io.Writer, n Node) error {
	dtd, ok := AsNode[*DTD](n)
	if !ok {
		return nil
	}
	d.writeString(out, "<!DOCTYPE ")
	// A DOCTYPE name must be a non-empty XML Name; an empty or all-whitespace name
	// serializes as "<!DOCTYPE >", which no parser accepts.
	if strings.TrimSpace(dtd.Name()) == "" {
		d.check(fmt.Errorf("helium: empty DOCTYPE name: %w", ErrWriterInvalidDTDNode))
		return d.err
	}
	// A DOCTYPE name is emitted verbatim and must be a valid XML Name (which
	// subsumes the character-range check and the US-ASCII guard). Guard before the
	// name write so no raw octet leaks ahead of the sticky error.
	if !d.checkVerbatimName("DOCTYPE name", dtd.Name()) {
		return d.err
	}
	d.writeString(out, dtd.Name())

	// The DOCTYPE external-ID public/system literals are reference-less DTD
	// literals: an XML-invalid character is rejected (default) or U+FFFD-substituted.
	pubLit, stop := d.dtdLiteral("DOCTYPE public-ID literal", dtd.externalID)
	if stop {
		return d.err
	}
	sysLit, stop := d.dtdLiteral("DOCTYPE system-ID literal", dtd.systemID)
	if stop {
		return d.err
	}

	if d.err == nil && pubLit != "" {
		// A non-ASCII public id has no faithful US-ASCII serialization; reject that
		// (encoding error) before the PubidChar check so the US-ASCII path keeps
		// reporting the encoding failure rather than a PubidChar failure.
		if d.rejectNonASCIIStr("DOCTYPE public identifier", pubLit) {
			return d.err
		}
		if d.checkPubid(pubLit) || d.checkSystemLiteral(sysLit) {
			return d.err
		}
		pubQ := dtdQuoteChar(pubLit)
		sysQ := dtdQuoteChar(sysLit)
		d.writeString(out, fmt.Sprintf(" PUBLIC %c%s%c %c%s%c", pubQ, pubLit, pubQ, sysQ, sysLit, sysQ))
	} else if d.err == nil && sysLit != "" {
		if d.checkSystemLiteral(sysLit) {
			return d.err
		}
		sysQ := dtdQuoteChar(sysLit)
		d.writeString(out, fmt.Sprintf(" SYSTEM %c%s%c", sysQ, sysLit, sysQ))
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

func (d *writeSession) dumpEnumeration(out io.Writer, atype enum.AttributeType, n Enumeration) error {
	if len(n) == 0 {
		d.check(fmt.Errorf("helium: attribute type %d enumeration must contain at least one token: %w", atype, ErrWriterInvalidName))
		return d.err
	}

	seen := make(map[string]struct{}, len(n))
	for _, v := range n {
		var valid bool
		switch atype {
		case enum.AttrEnumeration:
			valid = isValidNmtoken(v)
		case enum.AttrNotation:
			valid = xmlchar.IsValidName(v)
		default:
			d.check(fmt.Errorf("helium: invalid enumeration attribute type: %w", ErrWriterInvalidDTDNode))
			return d.err
		}
		if !valid {
			d.check(fmt.Errorf("helium: invalid enumeration token %q: %w", v, ErrWriterInvalidName))
			return d.err
		}
		if _, duplicate := seen[v]; duplicate {
			d.check(fmt.Errorf("helium: duplicate enumeration token %q: %w", v, ErrWriterInvalidName))
			return d.err
		}
		seen[v] = struct{}{}
		// Enumeration members are grammar tokens, not character content. Validate
		// before this US-ASCII check and never substitute an invalid token.
		if d.rejectNonASCIIStr("enumeration token", v) {
			return d.err
		}
	}

	for i, v := range n {
		d.writeString(out, v)
		if i != len(n)-1 {
			d.writeString(out, " | ")
		}
	}
	d.writeString(out, ")")
	return d.err
}

// writePrefixedName emits a DTD Name held in split storage. The split is not a
// QName interpretation: dtdName restores the exact original spelling before
// validating it with the DTD Name grammar.
func (d *writeSession) writePrefixedName(out io.Writer, prefix, name string) {
	fullName := dtdName(prefix, name)
	if !d.checkVerbatimName("DTD declaration name", fullName) {
		return
	}
	d.writeString(out, fullName)
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
	// An entity value screens both its literal runes (the reference-less DTD-literal
	// policy) and every character reference it carries: a &#N;/&#xN; target must be
	// serializable in the target XML version, else it is rejected (default) or its
	// reference replaced by U+FFFD.
	content, stop := d.entityValueLiteral("entity value", content, false)
	if stop {
		return d.err
	}
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

	// Entity names are intentionally NCNames: Helium's parser and streaming writer
	// reject colon-bearing names.
	if !d.checkVerbatimNCName("entity name", ent.name) {
		return d.err
	}

	switch etype := ent.entityType; etype {
	case enum.InternalGeneralEntity:
		d.writeString(out, "<!ENTITY ")
		d.writeString(out, ent.name)
		d.writeString(out, " ")
		if ent.orig != "" {
			orig, stop := d.entityValueLiteral("entity value", ent.orig, true)
			if stop {
				return d.err
			}
			if err := dumpQuotedString(out, orig); err != nil {
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
		if err := d.dumpExternalEntityID(out, ent); err != nil {
			return err
		}

		if etype == enum.ExternalGeneralUnparsedEntity {
			if ent.content != "" {
				// The NDATA notation name is emitted verbatim and must be a valid XML
				// Name. Guard before the " NDATA " write so no raw octet leaks ahead of
				// the sticky error.
				notationName := ent.content
				if ent.orig != "" {
					notationName = ent.orig
				}
				if !d.checkVerbatimName("NDATA notation name", notationName) {
					return d.err
				}
				d.writeString(out, " NDATA ")
				d.writeString(out, notationName)
			}
		}
		d.writeString(out, ">\n")
	case enum.InternalParameterEntity:
		d.writeString(out, "<!ENTITY % ")
		d.writeString(out, ent.name)
		d.writeString(out, " ")
		if ent.orig != "" {
			orig, stop := d.entityValueLiteral("parameter-entity value", ent.orig, true)
			if stop {
				return d.err
			}
			if err := dumpQuotedString(out, orig); err != nil {
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
		if err := d.dumpExternalEntityID(out, ent); err != nil {
			return err
		}
		d.writeString(out, ">\n")
	default:
		return fmt.Errorf("invalid entity type: %w", ErrWriterInvalidDTDNode)
	}
	return d.err
}

// dumpExternalEntityID writes the external identifier for an entity type that
// emits one. Internal entity declarations retain these fields in the DOM but do
// not consume them in XML, so their EntityValue paths handle their own content.
func (d *writeSession) dumpExternalEntityID(out io.Writer, ent *Entity) error {
	pubLit, stop := d.dtdLiteral("entity public-ID literal", ent.externalID)
	if stop {
		return d.err
	}
	sysLit, stop := d.dtdLiteral("entity system-ID literal", ent.systemID)
	if stop {
		return d.err
	}
	if pubLit != "" {
		if d.checkPubid(pubLit) {
			return d.err
		}
	}
	if d.checkSystemLiteral(sysLit) {
		return d.err
	}
	if pubLit != "" {
		d.writeString(out, " PUBLIC ")
		d.check(dumpQuotedString(out, pubLit))
		d.writeString(out, " ")
		d.check(dumpQuotedString(out, sysLit))
		return d.err
	}

	d.writeString(out, " SYSTEM ")
	d.check(dumpQuotedString(out, sysLit))
	return d.err
}

func (d *writeSession) dumpNotationDecl(out io.Writer, n *Notation) error {
	// The parser requires a notation declaration name to be an NCName.
	if !d.checkVerbatimNCName("notation name", n.name) {
		return d.err
	}
	// The public/system-ID literals are reference-less DTD literals: an XML-invalid
	// character is rejected (default) or U+FFFD-substituted.
	pubLit, stop := d.dtdLiteral("notation public-ID literal", n.publicID)
	if stop {
		return d.err
	}
	sysLit, stop := d.dtdLiteral("notation system-ID literal", n.systemID)
	if stop {
		return d.err
	}

	// A non-ASCII public id has no faithful US-ASCII serialization; reject that
	// (encoding error) before the PubidChar check so the US-ASCII path keeps
	// reporting the encoding failure rather than a PubidChar failure.
	if pubLit != "" {
		if d.rejectNonASCIIStr("notation public identifier", pubLit) {
			return d.err
		}
		if d.checkPubid(pubLit) {
			return d.err
		}
	}
	if d.checkSystemLiteral(sysLit) {
		return d.err
	}
	d.writeString(out, "<!NOTATION ")
	d.writeString(out, n.name)
	if pubLit != "" {
		d.writeString(out, " PUBLIC ")
		d.check(dumpQuotedString(out, pubLit))
		if sysLit != "" {
			d.writeString(out, " ")
			d.check(dumpQuotedString(out, sysLit))
		}
	} else {
		d.writeString(out, " SYSTEM ")
		d.check(dumpQuotedString(out, sysLit))
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
	// The element name is emitted verbatim and must be a valid XML Name (which
	// subsumes the character-range check and the US-ASCII guard). Guard before the
	// write. (The attribute name/prefix are guarded inside writePrefixedName.)
	if !d.checkVerbatimName("attribute-list element name", n.elem) {
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
		if err := d.dumpEnumeration(out, n.atype, n.tree); err != nil {
			return err
		}
	case enum.AttrNotation:
		d.writeString(out, " NOTATION (")
		if err := d.dumpEnumeration(out, n.atype, n.tree); err != nil {
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
		d.check(escapeAttrValue(out, []byte(n.defvalue), d.escapeNonASCII, d.asciiOutput, d.asciiReject(), !d.replaceInvalidChars, d.xml11, nil))
		d.writeString(out, `"`)
	}
	d.writeString(out, ">\n")
	return d.err
}

func (d *writeSession) dumpNsList(out io.Writer, nslist []*Namespace) error {
	// A per-prefix seen guard keeps the start tag reparseable even if some path
	// left a duplicate prefix in nsDefs: at most one declaration per prefix is
	// emitted. This mirrors the attribute-chain guard in reconcileNamespaces and
	// is correct-by-construction backup to DeclareNamespace's collapse rule.
	var seen map[string]struct{}
	for _, ns := range nslist {
		if seen == nil {
			seen = make(map[string]struct{}, len(nslist))
		}
		if _, dup := seen[ns.prefix]; dup {
			continue
		}
		seen[ns.prefix] = struct{}{}
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
	if err := escapeAttrValue(out, []byte(ns.href), d.escapeNonASCII, d.asciiOutput, d.asciiReject(), !d.replaceInvalidChars, d.xml11, nil); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"`); err != nil {
		return err
	}
	return nil
}
