package helium

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	henc "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/lexicon"
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
		if _, err := io.WriteString(out, "&quot;"); err != nil {
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
	esc_amp      = []byte("&amp;")
	esc_lt       = []byte("&lt;")
	esc_gt       = []byte("&gt;")
	esc_tab      = []byte("&#9;")
	esc_nl       = []byte("&#10;")
	esc_cr       = []byte("&#13;")
	esc_fffd     = []byte("\uFFFD")   // Unicode replacement character
	esc_fffd_ref = []byte("&#xFFFD;") // U+FFFD as a numeric character reference
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
			if r == 0xFFFD {
				esc = esc_fffd_ref
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
			if r == 0xFFFD {
				esc = esc_fffd_ref
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

// Writer serializes an XML document tree (libxml2: xmlSaveCtxt).
//
// It is a value-style wrapper: fluent methods return updated copies and the
// original is never mutated. Mutable runtime state (indent depth, resolved
// escapeNonASCII flag, XHTML detection) lives in a writeSession created
// inside each terminal method.
type Writer struct {
	format            bool
	indentString      string
	skipDTD           bool
	noEmpty           bool
	noDecl            bool
	noEscapeNonASCII  bool
	allowPrefixUndecl bool // emit xmlns:prefix="" undeclarations (XML 1.1)
}

// writeSession holds the mutable state for a single serialization pass.
// It is created inside WriteDoc / WriteNode and threaded through the
// internal helper methods so that Writer itself stays immutable.
type writeSession struct {
	Writer
	escapeNonASCII bool
	isXHTML        bool
	encoding       string // document encoding, used for XHTML meta injection
	indent         int    // current indent depth (used when format is true)
}

// NewWriter creates a new Writer with default settings.
func NewWriter() Writer {
	return Writer{}
}

// Format controls whether indented (pretty-printed) output is emitted.
func (w Writer) Format(v bool) Writer {
	w.format = v
	return w
}

// IndentString sets the string used for each indent level.
func (w Writer) IndentString(s string) Writer {
	w.indentString = s
	return w
}

// SelfCloseEmptyElements controls whether empty elements are serialized as
// self-closing tags (for example, <br/>). When false, they are emitted as
// explicit open+close pairs (for example, <br></br>).
func (w Writer) SelfCloseEmptyElements(v bool) Writer {
	w.noEmpty = !v
	return w
}

// XMLDeclaration controls whether the XML declaration is emitted.
func (w Writer) XMLDeclaration(v bool) Writer {
	w.noDecl = !v
	return w
}

// IncludeDTD controls whether DTD nodes are emitted.
func (w Writer) IncludeDTD(v bool) Writer {
	w.skipDTD = !v
	return w
}

// EscapeNonASCII controls whether non-ASCII characters are escaped as numeric
// character references when serializing UTF-8 output.
func (w Writer) EscapeNonASCII(v bool) Writer {
	w.noEscapeNonASCII = !v
	return w
}

// AllowPrefixUndeclarations controls whether xmlns:prefix="" undeclarations
// may be emitted.
func (w Writer) AllowPrefixUndeclarations(v bool) Writer {
	w.allowPrefixUndecl = v
	return w
}

func (s *writeSession) indentStr() string {
	if s.indentString == "" {
		return "  "
	}
	return s.indentString
}

func (s *writeSession) writeIndent(out io.Writer) {
	if !s.format || s.indent <= 0 {
		return
	}
	str := s.indentStr()
	for i := 0; i < s.indent; i++ {
		_, _ = io.WriteString(out, str)
	}
}

// hasOnlyTextChildren returns true when every child is a text or entity-ref node.
func hasOnlyTextChildren(n Node) bool {
	for c := range Children(n) {
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

// WriteDoc serializes a complete Document to the given writer
// (libxml2: xmlDocDumpFormatMemory / xmlSaveDoc).
func (d Writer) WriteDoc(out io.Writer, doc *Document) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Writer.WriteDoc")
		defer g.IRelease("END Writer.WriteDoc")
	}

	s := writeSession{Writer: d}

	// Mirrors libxml2's xmlSaveWriteText: when output encoding is UTF-8
	// (no encoder), escape non-ASCII chars 0x80-0xDF as numeric refs.
	// When an encoder is present, pass them through for re-encoding.
	s.escapeNonASCII = !d.noEscapeNonASCII
	if enc := doc.encoding; enc != "" {
		lower := strings.ToLower(enc)
		if lower != "utf-8" && lower != encUTF8 && lower != "us-ascii" && lower != "ascii" {
			if e := henc.Load(enc); e != nil {
				s.escapeNonASCII = false
				w := e.NewEncoder().Writer(out)
				if closer, ok := w.(io.Closer); ok {
					defer func() { _ = closer.Close() }()
				}
				out = w
			}
		}
	}

	// Detect XHTML. Mirrors xmlSaveDocInternal in xmlsave.c.
	s.isXHTML = false
	s.encoding = doc.encoding
	if dtd := doc.intSubset; dtd != nil {
		s.isXHTML = isXHTMLDTD(dtd)
	}

	if err := s.writeNode(out, doc); err != nil {
		return err
	}

	for e := range Children(doc) {
		if s.skipDTD && e.Type() == DTDNode {
			continue
		}
		if s.isXHTML && e.Type() == ElementNode {
			if err := s.dumpXHTMLNode(out, e); err != nil {
				return err
			}
		} else {
			if err := s.writeNode(out, e); err != nil {
				return err
			}
		}
		_, _ = io.WriteString(out, "\n")
	}
	return nil
}

func (d *writeSession) dumpDocContent(out io.Writer, n Node) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Writer.dumpDocContent")
		defer g.IRelease("END Writer.dumpDocContent")
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
		_, _ = io.WriteString(out, ` standalone="`+lexicon.ValueNo+`"`)
	case StandaloneExplicitYes:
		_, _ = io.WriteString(out, ` standalone="`+lexicon.ValueYes+`"`)
	}
	_, _ = io.WriteString(out, "?>\n")
	return nil
}

// dtdQuoteChar returns the appropriate quote character for a DTD identifier.
// Uses double quote by default, single quote if the value contains double quotes.
func dtdQuoteChar(value string) byte {
	if strings.ContainsRune(value, '"') {
		return '\''
	}
	return '"'
}

func (d *writeSession) dumpDTD(out io.Writer, n Node) error {
	dtd := n.(*DTD)
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

	// Suppress formatting for DTD children, matching libxml2's
	// xmlDtdDumpOutput which sets format=0, level=-1.
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

func (d *writeSession) dumpNotationDecl(out io.Writer, n *Notation) error {
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
		// no op
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
		// Mirrors libxml2's xmlSaveWriteAttributeDecl: always use double
		// quotes and escape <, >, ", & via escapeAttrValue.
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
		// Prefixed namespace with empty URI — skip unless serializer
		// opts in to XML 1.1 undeclarations (xmlns:prefix="").
		// The default XML 1.0 serialization does not support these.
		if !d.allowPrefixUndecl {
			return nil
		}
	}
	if ns.href == "" && ns.prefix == "" {
		// xmlns="" — namespace undeclaration; emit it
		_, err := io.WriteString(out, ` xmlns=""`)
		return err
	}

	// Skip the implicit xml: prefix namespace declaration.
	// libxml2: xmlNsDumpOutput skips prefix "xml" unconditionally.
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

// WriteNode serializes a single node and its subtree to the given writer
// (libxml2: xmlNodeDump).
func (d Writer) WriteNode(out io.Writer, n Node) error {
	s := writeSession{Writer: d, escapeNonASCII: !d.noEscapeNonASCII}
	return s.writeNode(out, n)
}

// writeNode is the internal implementation used by both WriteDoc and WriteNode.
func (d *writeSession) writeNode(out io.Writer, n Node) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Writer.WriteNode '%s'", n.Name())
		defer g.IRelease("END Writer.WriteNode")
	}

	var err error
	switch n.Type() {
	case DocumentNode:
		if !d.noDecl {
			if err = d.dumpDocContent(out, n); err != nil {
				return err
			}
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
	case NotationNode:
		if err = d.dumpNotationDecl(out, n.(*Notation)); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	if pdebug.Enabled {
		g := pdebug.IPrintf("START WriteNode(fallthrough)")
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
			g := pdebug.IPrintf("START WriteNode(fallthrough->attribute(%s))", attr.Name())
			_, _ = io.WriteString(out, " "+attr.Name()+`="`)
			count := 0
			for achld := range Children(attr) {
				count++
				if achld.Type() == TextNode {
					if err := escapeAttrValue(out, achld.Content(), d.escapeNonASCII); err != nil {
						return err
					}
				} else {
					if err := d.writeNode(out, achld); err != nil {
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
			if d.noEmpty {
				_, _ = io.WriteString(out, "></")
				_, _ = io.WriteString(out, name)
				_, _ = io.WriteString(out, ">")
			} else {
				_, _ = io.WriteString(out, "/>")
			}
			return nil
		}
	}

	_, _ = io.WriteString(out, ">")

	if child := n.FirstChild(); child != nil {
		textOnly := d.format && hasOnlyTextChildren(n)
		if d.format && !textOnly {
			_, _ = io.WriteString(out, "\n")
			d.indent++
		}
		for ; child != nil; child = child.NextSibling() {
			if d.format && !textOnly {
				d.writeIndent(out)
			}
			if err := d.writeNode(out, child); err != nil {
				return err
			}
			if d.format && !textOnly {
				_, _ = io.WriteString(out, "\n")
			}
		}
		if d.format && !textOnly {
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
func (d *writeSession) dumpXHTMLNode(out io.Writer, n Node) error {
	switch n.Type() {
	case ElementNode:
		// handled below
	default:
		return d.writeNode(out, n)
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
		if (e.ns == nil || e.ns.Prefix() == "") && xhtmlVoidElements[localName] && !addMeta {
			// C.2: Empty void elements: " />"
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
			// C.3: Non-void elements must use open+close tags
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

// dumpXHTMLAttrList dumps attributes with XHTML rules:
// - Boolean attribute normalization (C.5)
// - lang/xml:lang mirroring (C.7)
// - name/id mirroring (C.8)
func (d *writeSession) dumpXHTMLAttrList(out io.Writer, e *Element) {
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
		case lexicon.QNameXMLLang:
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
			for achld := range Children(attr) {
				if achld.Type() == TextNode {
					_ = escapeAttrValue(out, achld.Content(), d.escapeNonASCII)
				} else {
					_ = d.writeNode(out, achld)
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
// When formatting is enabled, a newline and indent are emitted before
// the meta tag, matching libxml2's behavior.
func (d *writeSession) writeMetaContentType(out io.Writer) {
	enc := d.encoding
	if enc == "" {
		enc = "UTF-8"
	}
	_, _ = io.WriteString(out, `<meta http-equiv="Content-Type" content="text/html; charset=`)
	_, _ = io.WriteString(out, enc)
	_, _ = io.WriteString(out, `" />`)
}
