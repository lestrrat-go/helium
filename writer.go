package helium

import (
	"io"
	"strings"

	henc "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/pdebug"
)

// Write serializes a node (document or element) to the given writer using
// default settings.
func Write(out io.Writer, node Node) error {
	return NewWriter().WriteTo(out, node)
}

// WriteString serializes a node (document or element) to a string using
// default settings.
func WriteString(node Node) (string, error) {
	var buf strings.Builder
	if err := Write(&buf, node); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const xmlTextNoEnc = "textnoenc"

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
// It is created inside WriteTo and threaded through the internal helper
// methods so that Writer itself stays immutable.
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
	for range s.indent {
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

// WriteTo serializes a node (document or element) to the given writer.
// When the node is a Document, document-level setup (encoding, XHTML
// detection, DTD filtering) is applied automatically.
func (d Writer) WriteTo(out io.Writer, node Node) error {
	if doc, ok := node.(*Document); ok {
		return d.writeDoc(out, doc)
	}
	s := writeSession{Writer: d, escapeNonASCII: !d.noEscapeNonASCII}
	return s.writeNode(out, node)
}

func (d Writer) writeDoc(out io.Writer, doc *Document) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Writer.writeDoc")
		defer g.IRelease("END Writer.writeDoc")
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

	doc := n.(*Document) //nolint:forcetypeassert
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

// writeNode is the internal implementation for node serialization.
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
		pi := n.(*ProcessingInstruction) //nolint:forcetypeassert
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
		if n.Name() == xmlTextNoEnc {
			// xmlTextNoEnc is a libxml2 marker (set on the node's name, not
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
		if err = d.dumpElementDecl(out, n.(*ElementDecl)); err != nil { //nolint:forcetypeassert
			return err
		}
		return nil
	case AttributeDeclNode:
		if err = d.dumpAttributeDecl(out, n.(*AttributeDecl)); err != nil { //nolint:forcetypeassert
			return err
		}
		return nil
	case EntityNode:
		if err = d.dumpEntityDecl(out, n.(*Entity)); err != nil { //nolint:forcetypeassert
			return err
		}
		return nil
	case NotationNode:
		if err = d.dumpNotationDecl(out, n.(*Notation)); err != nil { //nolint:forcetypeassert
			return err
		}
		return nil
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
			attr = a.(*Attribute) //nolint:forcetypeassert
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
