package helium

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	henc "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/pdebug"
)

var (
	qch_dquote = []byte{'"'}
	qch_quote  = []byte{'\''}
)

func dumpQuotedString(out io.Writer, s string) error {
	dqi := strings.IndexByte(s, qch_dquote[0])
	if dqi < 0 {
		// double quote is allowed, cool!
		if _, err := out.Write(qch_dquote); err != nil {
			return err
		}
		if _, err := io.WriteString(out, s); err != nil {
			return err
		}
		if _, err := out.Write(qch_dquote); err != nil {
			return err
		}
		return nil
	}

	if qi := strings.IndexByte(s, qch_quote[0]); qi < 0 {
		// single quotes, then
		if _, err := out.Write(qch_quote); err != nil {
			return err
		}
		if _, err := io.WriteString(out, s); err != nil {
			return err
		}
		if _, err := out.Write(qch_quote); err != nil {
			return err
		}
		return nil
	}

	// Grr, can't use " or '. Well, let's escape all the double
	// quotes to &quot;, and quote the string

	if _, err := out.Write(qch_dquote); err != nil {
		return err
	}
	for len(s) > 0 && dqi > -1 {
		if _, err := io.WriteString(out, s[:dqi]); err != nil {
			return err
		}
		s = s[dqi+1:]
		dqi = strings.IndexByte(s, qch_dquote[0])
	}

	if len(s) > 0 {
		if _, err := io.WriteString(out, s); err != nil {
			return err
		}
	}
	if _, err := out.Write(qch_dquote); err != nil {
		return err
	}
	return nil
}

var (
	esc_quot = []byte("&quot;")
	// esc_apos = []byte("&#39;") // shorter than "&apos;"
	esc_amp  = []byte("&amp;")
	esc_lt   = []byte("&lt;")
	esc_gt   = []byte("&gt;")
	esc_tab  = []byte("&#9;")
	esc_nl   = []byte("&#10;")
	esc_cr   = []byte("&#13;")
	esc_fffd = []byte("\uFFFD") // Unicode replacement character
)

// Decide whether the given rune is in the XML Character Range, per
// the Char production of http://www.xml.com/axml/testaxml.htm,
// Section 2.2 Characters.
func isInCharacterRange(r rune) (inrange bool) {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xDF77 ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}

func escapeAttrValue(w io.Writer, s []byte, escapeNonASCII bool) error {
	if pdebug.Enabled {
		debugbuf := bytes.Buffer{}
		w = io.MultiWriter(w, &debugbuf)
		g := pdebug.Marker("escapeAttrValue '%s'", s)
		defer func() {
			pdebug.Printf("escaped value '%s'", debugbuf.Bytes())
			g.End()
		}()
	}
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		switch r {
		case '"':
			esc = esc_quot
		case '&':
			esc = esc_amp
		case '<':
			esc = esc_lt
		case '>':
			esc = esc_gt
		case '\n':
			esc = esc_nl
		case '\r':
			esc = esc_cr
		case '\t':
			esc = esc_tab
		default:
			if escapeNonASCII && !(0x20 <= r && r < 0x80) { // nolint:staticcheck
				if r < 0x100 {
					esc = []byte(fmt.Sprintf("&#x%X;", r))
					break
				}
			}
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = esc_fffd
				break
			}
			continue
		}

		if _, err := w.Write(s[last : i-width]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		last = i
	}

	if _, err := w.Write(s[last:]); err != nil {
		return err
	}
	return nil
}

// escapeText writes to w the properly escaped XML equivalent
// of the plain text data s. If escapeNewline is true, newline
// characters will be escaped.
func escapeText(w io.Writer, s []byte, escapeNewline bool, escapeNonASCII bool) error {
	if pdebug.Enabled {
		debugbuf := bytes.Buffer{}
		w = io.MultiWriter(w, &debugbuf)
		g := pdebug.IPrintf("START escapeText = '%s'", s)
		defer func() {
			g.IRelease("END escapeText = '%s'", debugbuf.Bytes())
		}()
	}
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		switch r {
		case '&':
			esc = esc_amp
		case '<':
			esc = esc_lt
		case '>':
			esc = esc_gt
		case '\n':
			if !escapeNewline {
				continue
			}
			esc = esc_nl
		case '\r':
			esc = esc_cr
		default:
			if escapeNonASCII && !(r == '\t' || (0x20 <= r && r < 0x80)) { // nolint:staticcheck
				if r < 0x100 {
					esc = []byte(fmt.Sprintf("&#x%X;", r))
					break
				}
			}
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = esc_fffd
				break
			}
			continue
		}

		if _, err := w.Write(s[last : i-width]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		last = i
	}

	if _, err := w.Write(s[last:]); err != nil {
		return err
	}
	return nil
}

// Dumper serializes an XML document tree.
// escapeNonASCII controls whether characters U+0080–U+00FF are emitted as
// numeric character references (&#xNN;).  libxml2 only does this when the
// output encoding is UTF-8; when an encoding handler is present the
// characters pass through and the encoder converts them.
type Dumper struct {
	// Format enables indented output when set to true.
	Format bool
	// IndentString is the string used for each indent level (default "  ").
	IndentString string

	escapeNonASCII bool
	isXHTML        bool
	encoding       string // document encoding, used for XHTML meta injection
	indent         int    // current indent depth (used when Format is true)
}

func (d *Dumper) indentStr() string {
	if d.IndentString == "" {
		return "  "
	}
	return d.IndentString
}

func (d *Dumper) writeIndent(out io.Writer) {
	if !d.Format || d.indent <= 0 {
		return
	}
	s := d.indentStr()
	for i := 0; i < d.indent; i++ {
		_, _ = io.WriteString(out, s)
	}
}

// hasOnlyTextChildren returns true when every child is a text or entity-ref node.
func hasOnlyTextChildren(n Node) bool {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c.Type() {
		case TextNode, EntityRefNode, CDATASectionNode:
			// ok
		default:
			return false
		}
	}
	return true
}

// isXHTMLDTD returns true if the DTD identifies an XHTML document.
// Mirrors libxml2's xmlIsXHTML (tree.c).
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

// xhtmlVoidElements is the set of XHTML empty elements that use " />"
// self-closing syntax. Mirrors xhtmlIsEmpty in xmlsave.c.
var xhtmlVoidElements = map[string]bool{
	"area": true, "base": true, "basefont": true, "br": true,
	"col": true, "frame": true, "hr": true, "img": true,
	"input": true, "isindex": true, "link": true, "meta": true,
	"param": true,
}

// xhtmlNameIDElements is the set of elements where name→id mirroring
// applies (C.8). Mirrors xhtmlAttrListDumpOutput in xmlsave.c.
var xhtmlNameIDElements = map[string]bool{
	"a": true, "p": true, "div": true, "img": true,
	"map": true, "applet": true, "form": true, "frame": true,
	"iframe": true,
}

// htmlBooleanAttrs is the set of HTML boolean attributes.
// Mirrors htmlIsBooleanAttr in HTMLtree.c.
var htmlBooleanAttrs = map[string]bool{
	"checked": true, "compact": true, "declare": true, "defer": true,
	"disabled": true, "ismap": true, "multiple": true, "nohref": true,
	"noresize": true, "noshade": true, "nowrap": true, "readonly": true,
	"selected": true,
}

func (d *Dumper) DumpDoc(out io.Writer, doc *Document) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Dumper.DumpDoc")
		defer g.IRelease("END Dumper.DumpDoc")
	}

	// Mirrors libxml2's xmlSaveWriteText: when output encoding is UTF-8
	// (no encoder), escape non-ASCII chars 0x80-0xDF as numeric refs.
	// When an encoder is present, pass them through for re-encoding.
	d.escapeNonASCII = true
	if enc := doc.encoding; enc != "" {
		lower := strings.ToLower(enc)
		if lower != "utf-8" && lower != "utf8" && lower != "us-ascii" && lower != "ascii" {
			if e := henc.Load(enc); e != nil {
				d.escapeNonASCII = false
				w := e.NewEncoder().Writer(out)
				if closer, ok := w.(io.Closer); ok {
					defer func() { _ = closer.Close() }()
				}
				out = w
			}
		}
	}

	// Detect XHTML. Mirrors xmlSaveDocInternal in xmlsave.c.
	d.isXHTML = false
	d.encoding = doc.encoding
	if dtd := doc.intSubset; dtd != nil {
		d.isXHTML = isXHTMLDTD(dtd)
	}

	if err := d.DumpNode(out, doc); err != nil {
		return err
	}

	for e := doc.FirstChild(); e != nil; e = e.NextSibling() {
		if d.isXHTML && e.Type() == ElementNode {
			if err := d.dumpXHTMLNode(out, e); err != nil {
				return err
			}
		} else {
			if err := d.DumpNode(out, e); err != nil {
				return err
			}
		}
		_, _ = io.WriteString(out, "\n")
	}
	return nil
}

func (d *Dumper) dumpDocContent(out io.Writer, n Node) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Dumper.dumpDocContent")
		defer g.IRelease("END Dumper.dumpDocContent")
	}

	doc := n.(*Document)
	_, _ = io.WriteString(out, `<?xml version="`)
	version := doc.Version()
	if version == "" {
		version = "1.0"
	}
	_, _ = io.WriteString(out, version+`"`)

	if encoding := doc.encoding; encoding != "" {
		_, _ = io.WriteString(out, ` encoding="`+encoding+`"`)
	}

	switch doc.Standalone() {
	case StandaloneExplicitNo:
		_, _ = io.WriteString(out, ` standalone="no"`)
	case StandaloneExplicitYes:
		_, _ = io.WriteString(out, ` standalone="yes"`)
	}
	_, _ = io.WriteString(out, "?>\n")
	return nil
}

func (d *Dumper) dumpDTD(out io.Writer, n Node) error {
	dtd := n.(*DTD)
	_, _ = io.WriteString(out, "<!DOCTYPE ")
	_, _ = io.WriteString(out, dtd.Name())

	if dtd.externalID != "" {
		_, _ = io.WriteString(out, " PUBLIC \"")
		_, _ = io.WriteString(out, dtd.externalID)
		_, _ = io.WriteString(out, "\" \"")
		_, _ = io.WriteString(out, dtd.systemID)
		_, _ = io.WriteString(out, "\"")
	} else if dtd.systemID != "" {
		_, _ = io.WriteString(out, " SYSTEM \"")
		_, _ = io.WriteString(out, dtd.systemID)
		_, _ = io.WriteString(out, "\"")
	}

	if len(dtd.entities) == 0 && len(dtd.elements) == 0 && len(dtd.pentities) == 0 && len(dtd.attributes) == 0 {
		/* (dtd.notations == NULL) && */
		_, _ = io.WriteString(out, ">")
		return nil
	}

	_, _ = io.WriteString(out, " [\n")

	for e := dtd.FirstChild(); e != nil; e = e.NextSibling() {
		if err := d.DumpNode(out, e); err != nil {
			return err
		}
	}

	_, _ = io.WriteString(out, "]>")
	return nil
}

func (d *Dumper) dumpEnumeration(out io.Writer, n Enumeration) error {
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
		g := pdebug.IPrintf("START Dumper.dumpElementContent n = '%s'", n.name)
		defer g.IRelease("END Dumper.dumpElementContent")
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
		// no op
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

func (d *Dumper) dumpEntityDecl(out io.Writer, ent *Entity) error {
	if ent == nil {
		return nil
	}

	switch etype := ent.entityType; etype {
	case InternalGeneralEntity:
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
	case ExternalGeneralParsedEntity, ExternalGeneralUnparsedEntity:
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

		if etype == ExternalGeneralUnparsedEntity {
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
	case InternalParameterEntity:
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
	case ExternalParameterEntity:
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

func (d *Dumper) dumpElementDecl(out io.Writer, n *ElementDecl) error {
	switch n.decltype {
	case EmptyElementType:
		dumpElementDeclPrologue(out, n)
		_, _ = io.WriteString(out, " EMPTY>\n")
	case AnyElementType:
		dumpElementDeclPrologue(out, n)
		_, _ = io.WriteString(out, " ANY>\n")
	case MixedElementType, ElementElementType:
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

func (d *Dumper) dumpAttributeDecl(out io.Writer, n *AttributeDecl) error {
	_, _ = io.WriteString(out, "<!ATTLIST ")
	_, _ = io.WriteString(out, n.elem)
	_, _ = io.WriteString(out, " ")
	if n.prefix != "" {
		_, _ = io.WriteString(out, n.prefix)
		_, _ = io.WriteString(out, ":")
	}
	_, _ = io.WriteString(out, n.name)
	switch n.atype {
	case AttrCDATA:
		_, _ = io.WriteString(out, " CDATA")
	case AttrID:
		_, _ = io.WriteString(out, " ID")
	case AttrIDRef:
		_, _ = io.WriteString(out, " IDREF")
	case AttrIDRefs:
		_, _ = io.WriteString(out, " IDREFS")
	case AttrEntity:
		_, _ = io.WriteString(out, " ENTITY")
	case AttrNmtoken:
		_, _ = io.WriteString(out, " NMTOKEN")
	case AttrNmtokens:
		_, _ = io.WriteString(out, " NMTOKENS")
	case AttrEnumeration:
		_, _ = io.WriteString(out, " (")
		if err := d.dumpEnumeration(out, n.tree); err != nil {
			return err
		}
	case AttrNotation:
		_, _ = io.WriteString(out, " NOTATION (")
		if err := d.dumpEnumeration(out, n.tree); err != nil {
			return err
		}
	default:
		return errors.New("invalid AttributeDecl type")
	}

	switch n.def {
	case AttrDefaultNone:
		// no op
	case AttrDefaultRequired:
		_, _ = io.WriteString(out, " #REQUIRED")
	case AttrDefaultImplied:
		_, _ = io.WriteString(out, " #IMPLIED")
	case AttrDefaultFixed:
		_, _ = io.WriteString(out, " #FIXED")
	default:
		return errors.New("invalid AttributeDecl default value type")
	}

	if n.defvalue != "" {
		// Mirrors libxml2's xmlSaveWriteAttributeDecl: always use double
		// quotes and escape <, >, ", & via escapeAttrValue.
		_, _ = io.WriteString(out, ` "`)
		_ = escapeAttrValue(out, []byte(n.defvalue), d.escapeNonASCII)
		_, _ = io.WriteString(out, `"`)
	}
	_, _ = io.WriteString(out, ">\n")
	return nil
}

func (d *Dumper) dumpNsList(out io.Writer, nslist []*Namespace) error {
	for _, ns := range nslist {
		if err := d.dumpNs(out, ns); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dumper) dumpNs(out io.Writer, ns *Namespace) error {
	if ns.href == "" {
		// no op
		return nil
	}

	_, _ = io.WriteString(out, " ")

	if ns.prefix == "" {
		_, _ = io.WriteString(out, "xmlns")
	} else {
		_, _ = io.WriteString(out, "xmlns:")
		_, _ = io.WriteString(out, ns.prefix)
	}
	_, _ = io.WriteString(out, `="`)
	_ = escapeAttrValue(out, []byte(ns.href), d.escapeNonASCII)
	_, _ = io.WriteString(out, `"`)
	return nil
}

func (d *Dumper) DumpNode(out io.Writer, n Node) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Dumper.DumpNode '%s'", n.Name())
		defer g.IRelease("END Dumper.DumpNode")
	}

	var err error
	switch n.Type() {
	case DocumentNode:
		if err = d.dumpDocContent(out, n); err != nil {
			return err
		}
		return nil
	case DTDNode:
		if err = d.dumpDTD(out, n); err != nil {
			return err
		}
		return nil
	case CommentNode:
		_, _ = io.WriteString(out, "<!--")
		_, _ = out.Write(n.Content())
		_, _ = io.WriteString(out, "-->")
		return nil
	case ProcessingInstructionNode:
		// Mirrors xmlsave.c XML_PI_NODE handling.
		pi := n.(*ProcessingInstruction)
		_, _ = io.WriteString(out, "<?")
		_, _ = io.WriteString(out, pi.target)
		if pi.data != "" {
			_, _ = io.WriteString(out, " ")
			_, _ = io.WriteString(out, pi.data)
		}
		_, _ = io.WriteString(out, "?>")
		return nil
	case EntityRefNode:
		_, _ = io.WriteString(out, "&")
		_, _ = io.WriteString(out, n.Name())
		_, _ = io.WriteString(out, ";")
		return nil
	case TextNode:
		c := n.Content()
		if n.Name() == XMLTextNoEnc {
			// XMLTextNoEnc is a libxml2 marker (set on the node's name, not
			// its content) indicating the text should be emitted without
			// XML-escaping.  This is used during entity expansion
			// serialization where the replacement text is already encoded.
			if _, err := out.Write(c); err != nil {
				return err
			}
		} else {
			if err := escapeText(out, c, false, d.escapeNonASCII); err != nil {
				return err
			}
		}
		return nil // no recursing down
	case CDATASectionNode:
		// Mirrors xmlsave.c XML_CDATA_SECTION_NODE handling.
		// Splits content on "]]>" sequences so the output is well-formed.
		c := n.Content()
		if len(c) == 0 {
			_, _ = io.WriteString(out, "<![CDATA[]]>")
		} else {
			start := 0
			for i := 0; i+2 < len(c); i++ {
				if c[i] == ']' && c[i+1] == ']' && c[i+2] == '>' {
					end := i + 2
					_, _ = io.WriteString(out, "<![CDATA[")
					_, _ = out.Write(c[start:end])
					_, _ = io.WriteString(out, "]]>")
					start = end
				}
			}
			if start < len(c) {
				_, _ = io.WriteString(out, "<![CDATA[")
				_, _ = out.Write(c[start:])
				_, _ = io.WriteString(out, "]]>")
			}
		}
		return nil
	case ElementDeclNode:
		if err = d.dumpElementDecl(out, n.(*ElementDecl)); err != nil {
			return err
		}
		return nil
	case AttributeDeclNode:
		if err = d.dumpAttributeDecl(out, n.(*AttributeDecl)); err != nil {
			return err
		}
		return nil
	case EntityNode:
		if err = d.dumpEntityDecl(out, n.(*Entity)); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	if pdebug.Enabled {
		g := pdebug.IPrintf("START DumpNode(fallthrough)")
		defer g.IRelease("END DUmpNode(fallthrough)")
	}

	// if it got here it's some sort of an element
	var name string
	var nslist []*Namespace
	if nser, ok := n.(Namespacer); ok {
		if prefix := nser.Prefix(); prefix != "" {
			name = prefix + ":" + nser.LocalName()
		} else {
			name = nser.LocalName()
		}
		nslist = nser.Namespaces()
	} else {
		name = n.Name()
	}

	_, _ = io.WriteString(out, "<")
	_, _ = io.WriteString(out, name)

	if len(nslist) > 0 {
		if err := d.dumpNsList(out, nslist); err != nil {
			return err
		}
	}

	if e, ok := n.(*Element); ok {
		for attr := e.properties; attr != nil; {
			g := pdebug.IPrintf("START DumpNode(fallthrough->attribute(%s))", attr.Name())
			_, _ = io.WriteString(out, " "+attr.Name()+`="`)
			count := 0
			for achld := attr.FirstChild(); achld != nil; achld = achld.NextSibling() {
				count++
				if achld.Type() == TextNode {
					if err := escapeAttrValue(out, achld.Content(), d.escapeNonASCII); err != nil {
						return err
					}
				} else {
					if err := d.DumpNode(out, achld); err != nil {
						return err
					}
				}
			}
			_, _ = io.WriteString(out, `"`)
			g.IRelease("END DUmpNode(fallthrough->attribute(%s))", attr.Name())
			a := attr.NextSibling()
			if a == nil {
				break
			}
			attr = a.(*Attribute)
		}

		if child := e.FirstChild(); child == nil {
			_, _ = io.WriteString(out, "/>")
			return nil
		}
	}

	_, _ = io.WriteString(out, ">")

	if child := n.FirstChild(); child != nil {
		textOnly := d.Format && hasOnlyTextChildren(n)
		if d.Format && !textOnly {
			_, _ = io.WriteString(out, "\n")
			d.indent++
		}
		for ; child != nil; child = child.NextSibling() {
			if d.Format && !textOnly {
				d.writeIndent(out)
			}
			if err := d.DumpNode(out, child); err != nil {
				return err
			}
			if d.Format && !textOnly {
				_, _ = io.WriteString(out, "\n")
			}
		}
		if d.Format && !textOnly {
			d.indent--
			d.writeIndent(out)
		}
	}

	_, _ = io.WriteString(out, "</")
	_, _ = io.WriteString(out, name)
	_, _ = io.WriteString(out, ">")

	return nil
}

// dumpXHTMLNode serializes a node using XHTML rules.
// Mirrors xhtmlNodeDumpOutput in xmlsave.c.
func (d *Dumper) dumpXHTMLNode(out io.Writer, n Node) error {
	switch n.Type() {
	case ElementNode:
		// handled below
	default:
		return d.DumpNode(out, n)
	}

	e := n.(*Element)
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

	// Dump namespace declarations
	nslist := e.Namespaces()
	if len(nslist) > 0 {
		if err := d.dumpNsList(out, nslist); err != nil {
			return err
		}
	}

	// C.1: Inject xmlns="http://www.w3.org/1999/xhtml" on <html> if missing.
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

	// Dump attributes with XHTML mirroring rules.
	d.dumpXHTMLAttrList(out, e)

	// Determine if we need to inject <meta> Content-Type in <head>.
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
		if e.ns == nil && xhtmlVoidElements[localName] && !addMeta {
			// C.2: Empty void elements: " />"
			_, _ = io.WriteString(out, " />")
		} else {
			if addMeta {
				_, _ = io.WriteString(out, ">")
				d.writeMetaContentType(out)
			} else {
				_, _ = io.WriteString(out, ">")
			}
			// C.3: Non-void elements must use open+close tags
			_, _ = io.WriteString(out, "</")
			_, _ = io.WriteString(out, name)
			_, _ = io.WriteString(out, ">")
		}
		return nil
	}

	_, _ = io.WriteString(out, ">")
	if addMeta {
		d.writeMetaContentType(out)
	}

	for child := e.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == ElementNode {
			if err := d.dumpXHTMLNode(out, child); err != nil {
				return err
			}
		} else {
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

// dumpXHTMLAttrList dumps attributes with XHTML rules:
// - Boolean attribute normalization (C.5)
// - lang/xml:lang mirroring (C.7)
// - name/id mirroring (C.8)
func (d *Dumper) dumpXHTMLAttrList(out io.Writer, e *Element) {
	var langAttr, xmlLangAttr, nameAttr, idAttr *Attribute
	localName := e.LocalName()

	for attr := e.properties; attr != nil; {
		attrName := attr.Name()

		// Track special attributes for mirroring.
		// xml:lang is stored with full qualified name "xml:lang".
		switch attrName {
		case "id":
			idAttr = attr
		case "name":
			nameAttr = attr
		case "lang":
			langAttr = attr
		case "xml:lang":
			xmlLangAttr = attr
		}

		// Write the attribute (Name() already includes any namespace prefix)
		_, _ = io.WriteString(out, " ")
		_, _ = io.WriteString(out, attrName)
		_, _ = io.WriteString(out, `="`)

		// C.5: Boolean attribute normalization
		attrValue := attr.Value()
		if attrValue == "" && htmlBooleanAttrs[attrName] {
			_, _ = io.WriteString(out, attrName)
		} else {
			for achld := attr.FirstChild(); achld != nil; achld = achld.NextSibling() {
				if achld.Type() == TextNode {
					_ = escapeAttrValue(out, achld.Content(), d.escapeNonASCII)
				} else {
					_ = d.DumpNode(out, achld)
				}
			}
		}
		_, _ = io.WriteString(out, `"`)

		a := attr.NextSibling()
		if a == nil {
			break
		}
		attr = a.(*Attribute)
	}

	// C.8: name→id mirroring (only for specific elements)
	if nameAttr != nil && idAttr == nil && xhtmlNameIDElements[localName] {
		_, _ = io.WriteString(out, ` id="`)
		_ = escapeAttrValue(out, nameAttr.Content(), d.escapeNonASCII)
		_, _ = io.WriteString(out, `"`)
	}

	// C.7: lang/xml:lang mirroring
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

// headHasContentTypeMeta checks if a <head> element already has a
// <meta http-equiv="Content-Type"> child.
func (d *Dumper) headHasContentTypeMeta(head *Element) bool {
	for child := head.FirstChild(); child != nil; child = child.NextSibling() {
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
			a := attr.NextSibling()
			if a == nil {
				break
			}
			attr = a.(*Attribute)
		}
	}
	return false
}

// writeMetaContentType writes the XHTML meta Content-Type tag.
func (d *Dumper) writeMetaContentType(out io.Writer) {
	enc := d.encoding
	if enc == "" {
		enc = "UTF-8"
	}
	_, _ = io.WriteString(out, `<meta http-equiv="Content-Type" content="text/html; charset=`)
	_, _ = io.WriteString(out, enc)
	_, _ = io.WriteString(out, `" />`)
}
