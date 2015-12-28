package helium

import (
	"fmt"
	"io"
	"unicode/utf8"
)

var (
	esc_quot = []byte("&#34;") // shorter than "&quot;"
	esc_apos = []byte("&#39;") // shorter than "&apos;"
	esc_amp  = []byte("&amp;")
	esc_lt   = []byte("&lt;")
	esc_gt   = []byte("&gt;")
	esc_tab  = []byte("&#09;")
	esc_nl   = []byte("&#10;")
	esc_cr   = []byte("&#xD;")
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

// escapeText writes to w the properly escaped XML equivalent
// of the plain text data s. If escapeNewline is true, newline
// characters will be escaped.
func escapeText(w io.Writer, s []byte, escapeNewline bool) error {
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		switch r {
		case '"':
			esc = esc_quot
		case '\'':
			esc = esc_apos
		case '&':
			esc = esc_amp
		case '<':
			esc = esc_lt
		case '>':
			esc = esc_gt
		case '\t':
			esc = esc_tab
		case '\n':
			if !escapeNewline {
				continue
			}
			esc = esc_nl
		case '\r':
			esc = esc_cr
		default:
			if !(0x20 <= r && r < 0x80) {
				if r < 0xE0 {
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
	case EntityRefNode:
		io.WriteString(out, "&")
		io.WriteString(out, n.Name())
		io.WriteString(out, ";")
		return nil
	case TextNode:
		c := n.Content()
		if string(c) == XMLTextNoEnc {
			panic("unimplemented")
		} else {
			escapeText(out, c, false)
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
			io.WriteString(out, " "+attr.Name()+`="`)
			escapeText(out, []byte(attr.Value()), true)
			io.WriteString(out, `"`)
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