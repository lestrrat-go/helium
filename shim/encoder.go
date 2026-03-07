package shim

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

var (
	begComment  = []byte("<!--")
	endComment  = []byte("-->")
	endProcInst = []byte("?>")
)

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

func (enc *Encoder) writeIndent(depth int) error {
	if enc.indent == "" && enc.prefix == "" {
		return nil
	}
	if err := enc.w.WriteByte('\n'); err != nil {
		return err
	}
	if enc.prefix != "" {
		if _, err := enc.w.WriteString(enc.prefix); err != nil {
			return err
		}
	}
	for i := 0; i < depth; i++ {
		if _, err := enc.w.WriteString(enc.indent); err != nil {
			return err
		}
	}
	return nil
}

func (enc *Encoder) writeStartElement(se StartElement) error {
	if se.Name.Local == "" {
		return fmt.Errorf("xml: start tag with no name")
	}

	if enc.indent != "" || enc.prefix != "" {
		if enc.depth > 0 {
			if err := enc.writeIndent(enc.depth); err != nil {
				return err
			}
		}
	}

	enc.tags = append(enc.tags, se.Name)
	enc.nsStack.push()

	if err := enc.w.WriteByte('<'); err != nil {
		return err
	}
	if _, err := enc.w.WriteString(se.Name.Local); err != nil {
		return err
	}

	if se.Name.Space != "" {
		if _, err := enc.w.WriteString(` xmlns="`); err != nil {
			return err
		}
		if err := escapeAttrVal(enc.w, se.Name.Space); err != nil {
			return err
		}
		if err := enc.w.WriteByte('"'); err != nil {
			return err
		}
	}

	// Write attributes, interleaving xmlns declarations as needed.
	for _, attr := range se.Attr {
		name := attr.Name
		if name.Local == "" {
			continue
		}
		if err := enc.w.WriteByte(' '); err != nil {
			return err
		}
		if name.Space != "" {
			p := enc.nsStack.createAttrPrefix(enc.w, name.Space)
			if _, err := enc.w.WriteString(p); err != nil {
				return err
			}
			if err := enc.w.WriteByte(':'); err != nil {
				return err
			}
		}
		if _, err := enc.w.WriteString(name.Local); err != nil {
			return err
		}
		if _, err := enc.w.WriteString(`="`); err != nil {
			return err
		}
		if err := escapeAttrVal(enc.w, attr.Value); err != nil {
			return err
		}
		if err := enc.w.WriteByte('"'); err != nil {
			return err
		}
	}

	if err := enc.w.WriteByte('>'); err != nil {
		return err
	}

	enc.depth++
	enc.hasTokens = true
	enc.lastWasStart = true
	enc.lastWasText = false
	return nil
}

func (enc *Encoder) writeEndElement(ee EndElement) error {
	if ee.Name.Local == "" {
		return fmt.Errorf("xml: end tag with no name")
	}
	if len(enc.tags) == 0 || enc.tags[len(enc.tags)-1].Local == "" {
		return fmt.Errorf("xml: end tag </%s> without start tag", ee.Name.Local)
	}
	if top := enc.tags[len(enc.tags)-1]; top != ee.Name {
		if top.Local != ee.Name.Local {
			return fmt.Errorf("xml: end tag </%s> does not match start tag <%s>", ee.Name.Local, top.Local)
		}
		return fmt.Errorf("xml: end tag </%s> in namespace %s does not match start tag <%s> in namespace %s", ee.Name.Local, ee.Name.Space, top.Local, top.Space)
	}
	enc.tags = enc.tags[:len(enc.tags)-1]

	enc.depth--

	if (enc.indent != "" || enc.prefix != "") && !enc.lastWasStart && !enc.lastWasText {
		if err := enc.writeIndent(enc.depth); err != nil {
			return err
		}
	}

	if _, err := enc.w.WriteString("</"); err != nil {
		return err
	}
	if _, err := enc.w.WriteString(ee.Name.Local); err != nil {
		return err
	}
	if err := enc.w.WriteByte('>'); err != nil {
		return err
	}

	enc.nsStack.pop()

	enc.lastWasStart = false
	enc.lastWasText = false
	return nil
}

func (enc *Encoder) writeCharData(cd CharData) error {
	if err := escapeText(enc.w, []byte(cd)); err != nil {
		return err
	}
	enc.hasTokens = true
	enc.lastWasStart = false
	enc.lastWasText = true
	return nil
}

func (enc *Encoder) writeComment(c Comment) error {
	if bytes.Contains([]byte(c), endComment) {
		return fmt.Errorf("xml: EncodeToken of Comment containing --> marker")
	}
	if enc.indent != "" || enc.prefix != "" {
		if enc.depth > 0 && !enc.lastWasStart {
			if err := enc.writeIndent(enc.depth); err != nil {
				return err
			}
		}
	}
	if _, err := enc.w.WriteString("<!--"); err != nil {
		return err
	}
	if _, err := enc.w.Write([]byte(c)); err != nil {
		return err
	}
	if _, err := enc.w.WriteString("-->"); err != nil {
		return err
	}
	enc.hasTokens = true
	enc.lastWasStart = false
	enc.lastWasText = false
	return nil
}

func (enc *Encoder) writeProcInst(pi ProcInst) error {
	if pi.Target == "xml" && enc.hasTokens {
		return fmt.Errorf("xml: EncodeToken of ProcInst xml target only valid for xml declaration, first token encoded")
	}
	if !isXMLName(pi.Target) {
		return fmt.Errorf("xml: EncodeToken of ProcInst with invalid Target")
	}
	if bytes.Contains(pi.Inst, endProcInst) {
		return fmt.Errorf("xml: EncodeToken of ProcInst containing ?> marker")
	}
	if enc.indent != "" || enc.prefix != "" {
		if enc.depth > 0 && !enc.lastWasStart {
			if err := enc.writeIndent(enc.depth); err != nil {
				return err
			}
		}
	}
	if _, err := enc.w.WriteString("<?"); err != nil {
		return err
	}
	if _, err := enc.w.WriteString(pi.Target); err != nil {
		return err
	}
	if len(pi.Inst) > 0 {
		if err := enc.w.WriteByte(' '); err != nil {
			return err
		}
		if _, err := enc.w.Write(pi.Inst); err != nil {
			return err
		}
	}
	if _, err := enc.w.WriteString("?>"); err != nil {
		return err
	}
	enc.hasTokens = true
	enc.lastWasStart = false
	enc.lastWasText = false
	return nil
}

func (enc *Encoder) writeDirective(d Directive) error {
	if !isValidDirective(d) {
		return fmt.Errorf("xml: EncodeToken of Directive containing wrong < or > markers")
	}
	if _, err := enc.w.WriteString("<!"); err != nil {
		return err
	}
	if _, err := enc.w.Write([]byte(d)); err != nil {
		return err
	}
	if err := enc.w.WriteByte('>'); err != nil {
		return err
	}
	enc.hasTokens = true
	enc.lastWasStart = false
	enc.lastWasText = false
	return nil
}

// isValidDirective reports whether dir is a valid directive text,
// meaning angle brackets are matched, ignoring comments and strings.
func isValidDirective(dir Directive) bool {
	var (
		depth     int
		inquote   uint8
		incomment bool
	)
	for i, c := range dir {
		switch {
		case incomment:
			if c == '>' {
				if n := 1 + i - len(endComment); n >= 0 && bytes.Equal(dir[n:i+1], endComment) {
					incomment = false
				}
			}
		case inquote != 0:
			if c == inquote {
				inquote = 0
			}
		case c == '\'' || c == '"':
			inquote = c
		case c == '<':
			if i+len(begComment) < len(dir) && bytes.Equal(dir[i:i+len(begComment)], begComment) {
				incomment = true
			} else {
				depth++
			}
		case c == '>':
			if depth == 0 {
				return false
			}
			depth--
		}
	}
	return depth == 0 && inquote == 0 && !incomment
}

// Flush flushes any buffered XML to the underlying writer.
func (enc *Encoder) Flush() error {
	if enc.err != nil {
		return enc.err
	}
	if err := enc.w.Flush(); err != nil {
		enc.err = err
		return err
	}
	return nil
}

// Close flushes the encoder and returns an error if there are unclosed tags.
func (enc *Encoder) Close() error {
	if enc.closed {
		return nil
	}
	enc.closed = true
	if err := enc.w.Flush(); err != nil {
		return err
	}
	if enc.depth > 0 {
		tag := enc.tags[len(enc.tags)-1]
		return fmt.Errorf("unclosed tag <%s>", tag.Local)
	}
	return nil
}

// Encode writes the XML encoding of v to the stream.
func (enc *Encoder) Encode(v any) error {
	err := enc.marshalValue(v, nil)
	if err != nil {
		return err
	}
	return enc.Flush()
}

// EncodeElement writes the XML encoding of v to the stream,
// using start as the element tag.
func (enc *Encoder) EncodeElement(v any, start StartElement) error {
	err := enc.marshalValue(v, &start)
	if err != nil {
		return err
	}
	return enc.Flush()
}
