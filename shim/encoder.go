package shim

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

var ddBytes = []byte("--")

// Encoder writes XML tokens to an output stream.
type Encoder struct {
	w            *bufio.Writer
	prefix       string
	indent       string
	depth        int
	tags         []Name
	nsStack      nsStack
	err          error
	closed       bool
	lastWasStart bool
	lastWasText  bool
	hasTokens    bool
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: bufio.NewWriter(w),
	}
}

// Indent sets the encoder to generate XML in which each element begins
// on a new indented line that starts with prefix and is followed by
// one or more copies of indent according to the nesting depth.
func (enc *Encoder) Indent(prefix, indent string) {
	enc.prefix = prefix
	enc.indent = indent
}

// EncodeToken writes the given XML token to the stream.
func (enc *Encoder) EncodeToken(t Token) error {
	if enc.err != nil {
		return enc.err
	}
	if enc.closed {
		return fmt.Errorf("xml: EncodeToken called after Close")
	}

	switch v := t.(type) {
	case StartElement:
		return enc.writeStartElement(v)
	case EndElement:
		return enc.writeEndElement(v)
	case CharData:
		return enc.writeCharData(v)
	case Comment:
		return enc.writeComment(v)
	case ProcInst:
		return enc.writeProcInst(v)
	case Directive:
		return enc.writeDirective(v)
	}
	return fmt.Errorf("xml: EncodeToken of invalid token type")
}

func (enc *Encoder) writeIndent(depth int) {
	if enc.indent == "" && enc.prefix == "" {
		return
	}
	enc.w.WriteByte('\n')
	if enc.prefix != "" {
		enc.w.WriteString(enc.prefix)
	}
	for i := 0; i < depth; i++ {
		enc.w.WriteString(enc.indent)
	}
}

func (enc *Encoder) writeStartElement(se StartElement) error {
	if enc.indent != "" || enc.prefix != "" {
		if enc.depth > 0 {
			enc.writeIndent(enc.depth)
		}
	}

	enc.nsStack.push()

	enc.w.WriteByte('<')

	// Determine if we need to declare namespaces
	prefix := ""
	needsDefaultNS := false
	if se.Name.Space != "" {
		existingPrefix, found := enc.nsStack.resolve(se.Name.Space)
		if found {
			prefix = existingPrefix
		} else {
			// Auto-declare namespace
			needsDefaultNS = true
			enc.nsStack.addBinding("", se.Name.Space)
		}
	}

	if prefix != "" {
		enc.w.WriteString(prefix)
		enc.w.WriteByte(':')
	}
	enc.w.WriteString(se.Name.Local)

	if needsDefaultNS {
		enc.w.WriteString(` xmlns="`)
		escapeAttrVal(enc.w, se.Name.Space)
		enc.w.WriteByte('"')
	}

	// Collect xmlns attrs from Attr list and attr namespace bindings
	attrNSDecls := map[string]string{} // prefix → URI for attrs
	for _, attr := range se.Attr {
		if attr.Name.Space == "" {
			continue
		}
		if attr.Name.Space == "xmlns" {
			continue
		}
		if _, found := enc.nsStack.resolve(attr.Name.Space); !found {
			if _, already := attrNSDecls[attr.Name.Space]; !already {
				p := enc.nsStack.allocPrefix()
				enc.nsStack.addBinding(p, attr.Name.Space)
				attrNSDecls[attr.Name.Space] = p
			}
		}
	}

	// Write xmlns declarations for attribute namespaces
	for uri, p := range attrNSDecls {
		enc.w.WriteString(` xmlns:`)
		enc.w.WriteString(p)
		enc.w.WriteString(`="`)
		escapeAttrVal(enc.w, uri)
		enc.w.WriteByte('"')
	}

	// Write attributes
	for _, attr := range se.Attr {
		enc.w.WriteByte(' ')
		if attr.Name.Space != "" && attr.Name.Space != "xmlns" {
			p, _ := enc.nsStack.resolve(attr.Name.Space)
			if p != "" {
				enc.w.WriteString(p)
				enc.w.WriteByte(':')
			}
		}
		enc.w.WriteString(attr.Name.Local)
		enc.w.WriteString(`="`)
		escapeAttrVal(enc.w, attr.Value)
		enc.w.WriteByte('"')
	}

	enc.w.WriteByte('>')

	enc.tags = append(enc.tags, se.Name)
	enc.depth++
	enc.hasTokens = true
	enc.lastWasStart = true
	enc.lastWasText = false
	return nil
}

func (enc *Encoder) writeEndElement(ee EndElement) error {
	if enc.depth == 0 {
		return fmt.Errorf("xml: EndElement </...> without StartElement")
	}

	enc.depth--

	if (enc.indent != "" || enc.prefix != "") && !enc.lastWasStart && !enc.lastWasText {
		enc.writeIndent(enc.depth)
	}

	enc.w.WriteString("</")
	// Use the same prefix as the start element
	if ee.Name.Space != "" {
		if p, found := enc.nsStack.resolve(ee.Name.Space); found && p != "" {
			enc.w.WriteString(p)
			enc.w.WriteByte(':')
		}
	}
	enc.w.WriteString(ee.Name.Local)
	enc.w.WriteByte('>')

	enc.nsStack.pop()
	if len(enc.tags) > 0 {
		enc.tags = enc.tags[:len(enc.tags)-1]
	}

	enc.lastWasStart = false
	enc.lastWasText = false
	return nil
}

func (enc *Encoder) writeCharData(cd CharData) error {
	escapeText(enc.w, []byte(cd))
	enc.hasTokens = true
	enc.lastWasStart = false
	enc.lastWasText = true
	return nil
}

func (enc *Encoder) writeComment(c Comment) error {
	if bytes.Contains([]byte(c), ddBytes) {
		return fmt.Errorf("xml: comments must not contain \"--\"")
	}
	if enc.indent != "" || enc.prefix != "" {
		if enc.depth > 0 && !enc.lastWasStart {
			enc.writeIndent(enc.depth)
		}
	}
	enc.w.WriteString("<!--")
	enc.w.Write([]byte(c))
	if len(c) > 0 && c[len(c)-1] == '-' {
		enc.w.WriteByte(' ')
	}
	enc.w.WriteString("-->")
	enc.hasTokens = true
	enc.lastWasStart = false
	enc.lastWasText = false
	return nil
}

func (enc *Encoder) writeProcInst(pi ProcInst) error {
	if pi.Target == "xml" && enc.hasTokens {
		return fmt.Errorf("xml: EncodeToken of ProcInst xml target only valid for first token")
	}
	if enc.indent != "" || enc.prefix != "" {
		if enc.depth > 0 && !enc.lastWasStart {
			enc.writeIndent(enc.depth)
		}
	}
	enc.w.WriteString("<?")
	enc.w.WriteString(pi.Target)
	if len(pi.Inst) > 0 {
		enc.w.WriteByte(' ')
		enc.w.Write(pi.Inst)
	}
	enc.w.WriteString("?>")
	enc.hasTokens = true
	enc.lastWasStart = false
	enc.lastWasText = false
	return nil
}

func (enc *Encoder) writeDirective(d Directive) error {
	enc.w.WriteString("<!")
	enc.w.Write([]byte(d))
	enc.w.WriteByte('>')
	enc.hasTokens = true
	enc.lastWasStart = false
	enc.lastWasText = false
	return nil
}

// Flush flushes any buffered XML to the underlying writer.
func (enc *Encoder) Flush() error {
	if enc.err != nil {
		return enc.err
	}
	return enc.w.Flush()
}

// Close flushes the encoder and returns an error if there are unclosed tags.
func (enc *Encoder) Close() error {
	if enc.closed {
		return nil
	}
	enc.closed = true
	enc.w.Flush()
	if enc.depth > 0 {
		tag := enc.tags[len(enc.tags)-1]
		return fmt.Errorf("unclosed tag <%s>", tag.Local)
	}
	return nil
}

// Encode writes the XML encoding of v to the stream.
func (enc *Encoder) Encode(v any) error {
	return enc.marshalValue(v, nil)
}

// EncodeElement writes the XML encoding of v to the stream,
// using start as the element tag.
func (enc *Encoder) EncodeElement(v any, start StartElement) error {
	return enc.marshalValue(v, &start)
}
