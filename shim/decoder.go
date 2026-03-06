package shim

import (
	"context"
	stdxml "encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
)

type tokenEvent struct {
	tok    Token
	rawTok Token // raw variant (prefix:local instead of namespace URI)
	line   int
	col    int
	err    error
}

// Decoder reads XML tokens from a stream. It is a drop-in replacement for
// encoding/xml.Decoder backed by helium's SAX parser.
type Decoder struct {
	// Strict mode. When true (default), the parser requires strict XML conformance.
	Strict bool

	// AutoClose lists element names that should be auto-closed.
	AutoClose []string

	// Entity maps entity names to replacement text.
	Entity map[string]string

	// CharsetReader, if non-nil, defines a function to generate charset-conversion
	// readers, converting from the provided charset into UTF-8.
	CharsetReader func(charset string, input io.Reader) (io.Reader, error)

	// DefaultSpace sets the default namespace for elements without an explicit namespace.
	DefaultSpace string

	tokenReader TokenReader
	events      chan tokenEvent
	ctx         context.Context
	cancel      context.CancelFunc
	lastToken   Token
	savedErr    error
	offset      int64
	line        int
	column      int
}

func newDecoderFromReader(r io.Reader) (*Decoder, error) {
	ctx, cancel := context.WithCancel(context.Background())
	d := &Decoder{
		Strict: true,
		events: make(chan tokenEvent, 64),
		ctx:    ctx,
		cancel: cancel,
		line:   1,
		column: 1,
	}
	d.startSAXEmitter(r)
	return d, nil
}

func newDecoderFromTokenReader(tr TokenReader) *Decoder {
	return &Decoder{
		Strict:      true,
		tokenReader: tr,
		line:        1,
		column:      1,
	}
}

func (d *Decoder) startSAXEmitter(r io.Reader) {
	var locator sax.DocumentLocator

	push := func(tok, rawTok Token, line, col int) error {
		select {
		case d.events <- tokenEvent{tok: stdxml.CopyToken(tok), rawTok: stdxml.CopyToken(rawTok), line: line, col: col}:
			return nil
		case <-d.ctx.Done():
			return d.ctx.Err()
		}
	}

	h := sax.New()
	h.OnStartDocument = sax.StartDocumentFunc(func(_ sax.Context) error { return nil })
	h.OnEndDocument = sax.EndDocumentFunc(func(_ sax.Context) error { return nil })
	h.OnSetDocumentLocator = sax.SetDocumentLocatorFunc(func(_ sax.Context, loc2 sax.DocumentLocator) error {
		locator = loc2
		return nil
	})
	h.OnStartElementNS = sax.StartElementNSFunc(func(_ sax.Context, localname, prefix string, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		line, col := 0, 0
		if locator != nil {
			line = locator.LineNumber()
			col = locator.ColumnNumber()
		}

		// Build namespace map for attribute URI resolution
		attrNS := make(map[string]string, len(namespaces))
		for _, ns := range namespaces {
			attrNS[ns.Prefix()] = ns.URI()
		}

		// Resolved token (for Token())
		se := StartElement{Name: Name{Space: uri, Local: localname}}
		// Raw token (for RawToken()) — uses "prefix:local" form
		rawLocal := localname
		if prefix != "" {
			rawLocal = prefix + ":" + localname
		}
		rawSE := StartElement{Name: Name{Local: rawLocal}}

		if len(attrs) > 0 {
			se.Attr = make([]Attr, 0, len(attrs))
			rawSE.Attr = make([]Attr, 0, len(attrs))
			for _, attr := range attrs {
				space := ""
				if p := attr.Prefix(); p != "" {
					space = attrNS[p]
				}
				se.Attr = append(se.Attr, Attr{
					Name:  Name{Space: space, Local: attr.LocalName()},
					Value: attr.Value(),
				})
				rawAttrLocal := attr.LocalName()
				if p := attr.Prefix(); p != "" {
					rawAttrLocal = p + ":" + attr.LocalName()
				}
				rawSE.Attr = append(rawSE.Attr, Attr{
					Name:  Name{Local: rawAttrLocal},
					Value: attr.Value(),
				})
			}
		}
		return push(se, rawSE, line, col)
	})
	h.OnEndElementNS = sax.EndElementNSFunc(func(_ sax.Context, localname, prefix string, uri string) error {
		line, col := 0, 0
		if locator != nil {
			line = locator.LineNumber()
			col = locator.ColumnNumber()
		}
		rawLocal := localname
		if prefix != "" {
			rawLocal = prefix + ":" + localname
		}
		ee := EndElement{Name: Name{Space: uri, Local: localname}}
		rawEE := EndElement{Name: Name{Local: rawLocal}}
		return push(ee, rawEE, line, col)
	})
	h.OnCharacters = sax.CharactersFunc(func(_ sax.Context, ch []byte) error {
		line, col := 0, 0
		if locator != nil {
			line = locator.LineNumber()
			col = locator.ColumnNumber()
		}
		cd := CharData(append([]byte(nil), ch...))
		return push(cd, cd, line, col)
	})
	h.OnIgnorableWhitespace = sax.IgnorableWhitespaceFunc(func(_ sax.Context, ch []byte) error {
		line, col := 0, 0
		if locator != nil {
			line = locator.LineNumber()
			col = locator.ColumnNumber()
		}
		cd := CharData(append([]byte(nil), ch...))
		return push(cd, cd, line, col)
	})
	h.OnCDataBlock = sax.CDataBlockFunc(func(_ sax.Context, value []byte) error {
		line, col := 0, 0
		if locator != nil {
			line = locator.LineNumber()
			col = locator.ColumnNumber()
		}
		cd := CharData(append([]byte(nil), value...))
		return push(cd, cd, line, col)
	})
	h.OnComment = sax.CommentFunc(func(_ sax.Context, value []byte) error {
		line, col := 0, 0
		if locator != nil {
			line = locator.LineNumber()
			col = locator.ColumnNumber()
		}
		c := Comment(append([]byte(nil), value...))
		return push(c, c, line, col)
	})
	h.OnProcessingInstruction = sax.ProcessingInstructionFunc(func(_ sax.Context, target, data string) error {
		if target == "xml" {
			return nil // skip XML declaration
		}
		line, col := 0, 0
		if locator != nil {
			line = locator.LineNumber()
			col = locator.ColumnNumber()
		}
		pi := ProcInst{Target: target, Inst: []byte(data)}
		return push(pi, pi, line, col)
	})

	// Stubs for callbacks we don't use
	h.OnInternalSubset = sax.InternalSubsetFunc(func(_ sax.Context, _ string, _ string, _ string) error { return nil })
	h.OnExternalSubset = sax.ExternalSubsetFunc(func(_ sax.Context, _ string, _ string, _ string) error { return nil })
	h.OnReference = sax.ReferenceFunc(func(_ sax.Context, _ string) error { return nil })
	h.OnEntityDecl = sax.EntityDeclFunc(func(_ sax.Context, _ string, _ enum.EntityType, _ string, _ string, _ string) error { return nil })
	h.OnElementDecl = sax.ElementDeclFunc(func(_ sax.Context, _ string, _ enum.ElementType, _ sax.ElementContent) error { return nil })
	h.OnAttributeDecl = sax.AttributeDeclFunc(func(_ sax.Context, _ string, _ string, _ enum.AttributeType, _ enum.AttributeDefault, _ string, _ sax.Enumeration) error {
		return nil
	})
	h.OnNotationDecl = sax.NotationDeclFunc(func(_ sax.Context, _ string, _ string, _ string) error { return nil })
	h.OnUnparsedEntityDecl = sax.UnparsedEntityDeclFunc(func(_ sax.Context, _ string, _ string, _ string, _ string) error { return nil })
	h.OnGetEntity = sax.GetEntityFunc(func(_ sax.Context, _ string) (sax.Entity, error) { return nil, nil })
	h.OnGetParameterEntity = sax.GetParameterEntityFunc(func(_ sax.Context, _ string) (sax.Entity, error) { return nil, nil })
	h.OnResolveEntity = sax.ResolveEntityFunc(func(_ sax.Context, _ string, _ string) (sax.ParseInput, error) { return nil, nil })
	h.OnHasExternalSubset = sax.HasExternalSubsetFunc(func(_ sax.Context) (bool, error) { return false, nil })
	h.OnHasInternalSubset = sax.HasInternalSubsetFunc(func(_ sax.Context) (bool, error) { return false, nil })
	h.OnIsStandalone = sax.IsStandaloneFunc(func(_ sax.Context) (bool, error) { return false, nil })
	h.OnError = sax.ErrorFunc(func(_ sax.Context, err error) error { return err })
	h.OnWarning = sax.WarningFunc(func(_ sax.Context, _ error) error { return nil })

	go func() {
		defer close(d.events)
		p := helium.NewParser()
		p.SetSAXHandler(h)
		_, err := p.ParseReader(d.ctx, r)
		if err != nil {
			select {
			case d.events <- tokenEvent{err: err}:
			case <-d.ctx.Done():
			}
		}
	}()
}

// Close cancels the SAX goroutine and releases resources.
func (d *Decoder) Close() {
	if d.cancel != nil {
		d.cancel()
	}
}

func (d *Decoder) advancePosition(tok Token) {
	// Estimate byte size from token for InputOffset tracking.
	n := tokenSize(tok)
	d.offset += int64(n)
}

// tokenSize returns an estimated byte size of the serialized token,
// matching encoding/xml's offset accounting.
func tokenSize(tok Token) int {
	switch v := tok.(type) {
	case StartElement:
		// <name attr="val">
		n := 1 + len(v.Name.Local) + 1 // < name >
		if v.Name.Space != "" {
			// This is an approximation since we don't have the prefix
		}
		for _, a := range v.Attr {
			n += 1 + len(a.Name.Local) + 2 + len(a.Value) + 1 // space name="val"
		}
		return n
	case EndElement:
		return 2 + len(v.Name.Local) + 1 // </name>
	case CharData:
		return len(v)
	case Comment:
		return 7 + len(v) // <!--...-->
	case ProcInst:
		return 4 + len(v.Target) + 1 + len(v.Inst) + 2 // <?target data?>
	case Directive:
		return 3 + len(v) + 1 // <!...>
	}
	return 0
}

// Token returns the next XML token in the input stream.
// Namespace URIs are resolved in the Name.Space field.
func (d *Decoder) Token() (Token, error) {
	tok, err := d.readToken(false)
	if err != nil {
		return nil, err
	}
	if d.DefaultSpace != "" {
		tok = applyDefaultSpace(tok, d.DefaultSpace)
	}
	return tok, nil
}

// RawToken returns the next XML token without namespace resolution.
// Element names use prefix:local form instead of resolved namespace URIs.
func (d *Decoder) RawToken() (Token, error) {
	return d.readToken(true)
}

func (d *Decoder) readToken(raw bool) (Token, error) {
	var tok Token

	if d.tokenReader != nil {
		if d.savedErr != nil {
			err := d.savedErr
			d.savedErr = nil
			return nil, err
		}
		for {
			nextTok, err := d.tokenReader.Token()
			if nextTok == nil && err == nil {
				continue
			}
			if err != nil {
				if nextTok != nil {
					d.savedErr = err
					tok = nextTok
					break
				}
				return nil, err
			}
			tok = nextTok
			break
		}
	} else {
		event, ok := <-d.events
		if !ok {
			return nil, io.EOF
		}
		if event.err != nil {
			return nil, convertParseError(event.err)
		}
		if event.line > 0 {
			d.line = event.line
			d.column = event.col
		}
		if raw {
			tok = event.rawTok
		} else {
			tok = event.tok
		}
	}

	tok = stdxml.CopyToken(tok)

	// Check encoding attribute in XML declaration
	if pi, ok := tok.(ProcInst); ok && pi.Target == "xml" {
		if err := d.checkProcInstEncoding(string(pi.Inst)); err != nil {
			return nil, err
		}
	}

	d.lastToken = tok
	d.advancePosition(tok)
	return tok, nil
}

// checkProcInstEncoding validates the encoding attribute in an XML declaration.
// UTF-8 (case-insensitive) is always accepted. Non-UTF-8 requires CharsetReader.
func (d *Decoder) checkProcInstEncoding(data string) error {
	enc := procInstValue(data, "encoding")
	if enc == "" {
		return nil
	}
	if strings.EqualFold(enc, "utf-8") {
		return nil
	}
	if d.CharsetReader == nil {
		return fmt.Errorf("xml: encoding %q declared but Decoder.CharsetReader is nil", enc)
	}
	return nil
}

// procInstValue extracts the value of an attribute from a processing instruction's data.
func procInstValue(data, param string) string {
	idx := strings.Index(data, param)
	if idx < 0 {
		return ""
	}
	s := data[idx+len(param):]
	s = strings.TrimSpace(s)
	if s == "" || s[0] != '=' {
		return ""
	}
	s = strings.TrimSpace(s[1:])
	if s == "" {
		return ""
	}
	q := s[0]
	if q != '\'' && q != '"' {
		return ""
	}
	end := strings.IndexByte(s[1:], q)
	if end < 0 {
		return ""
	}
	return s[1 : end+1]
}

func applyDefaultSpace(tok Token, space string) Token {
	switch v := tok.(type) {
	case StartElement:
		if v.Name.Space == "" {
			v.Name.Space = space
			return v
		}
	case EndElement:
		if v.Name.Space == "" {
			v.Name.Space = space
			return v
		}
	}
	return tok
}

func (d *Decoder) Skip() error {
	if d.lastToken == nil {
		return errors.New("shim: Skip called before reading start element")
	}
	if _, ok := d.lastToken.(StartElement); !ok {
		return nil
	}

	depth := 1
	for depth > 0 {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch tok.(type) {
		case StartElement:
			depth++
		case EndElement:
			depth--
		}
	}
	return nil
}

func (d *Decoder) InputOffset() int64 {
	return d.offset
}

func (d *Decoder) InputPos() (line, column int) {
	if d.line == 0 {
		return 1, 1
	}
	return d.line, d.column
}

func (d *Decoder) Decode(v any) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		start, ok := tok.(stdxml.StartElement)
		if !ok {
			continue
		}
		return d.DecodeElement(v, &start)
	}
}

func (d *Decoder) DecodeElement(v any, start *StartElement) error {
	if start == nil {
		return d.Decode(v)
	}

	elem, err := d.buildElementFromTokens(*start)
	if err != nil {
		return err
	}
	return decodeElementInto(reflectValueOf(v), elem)
}

func reflectValueOf(v any) reflect.Value {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		return rv.Elem()
	}
	return rv
}

// buildElementFromTokens reads tokens from the decoder and builds
// a helium Element subtree. This avoids the previous approach of
// serializing tokens to bytes and re-parsing.
func (d *Decoder) buildElementFromTokens(start stdxml.StartElement) (*helium.Element, error) {
	doc := helium.NewDefaultDocument()

	root, err := doc.CreateElement(start.Name.Local)
	if err != nil {
		return nil, err
	}

	// Set namespace if present
	if start.Name.Space != "" {
		ns, nsErr := doc.CreateNamespace("", start.Name.Space)
		if nsErr == nil {
			root.SetAttributeNS(root.Name(), root.Name(), ns)
		}
	}

	// Set attributes
	for _, attr := range start.Attr {
		root.SetAttribute(attr.Name.Local, attr.Value)
	}

	if err := doc.SetDocumentElement(root); err != nil {
		return nil, err
	}

	// Read children
	if err := d.populateElement(doc, root, start.Name); err != nil {
		return nil, err
	}

	return root, nil
}

func (d *Decoder) populateElement(doc *helium.Document, parent *helium.Element, name Name) error {
	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}

		switch v := tok.(type) {
		case StartElement:
			child, cErr := doc.CreateElement(v.Name.Local)
			if cErr != nil {
				return cErr
			}
			for _, attr := range v.Attr {
				child.SetAttribute(attr.Name.Local, attr.Value)
			}
			if err := parent.AddChild(child); err != nil {
				return err
			}
			if err := d.populateElement(doc, child, v.Name); err != nil {
				return err
			}
		case EndElement:
			if v.Name.Local != name.Local || v.Name.Space != name.Space {
				return &SyntaxError{
					Msg:  "element <" + name.Local + "> closed by </" + v.Name.Local + ">",
					Line: d.line,
				}
			}
			return nil
		case CharData:
			text, tErr := doc.CreateText([]byte(v))
			if tErr != nil {
				return tErr
			}
			if err := parent.AddChild(text); err != nil {
				return err
			}
		case Comment:
			comment, cErr := doc.CreateComment([]byte(v))
			if cErr != nil {
				return cErr
			}
			if err := parent.AddChild(comment); err != nil {
				return err
			}
		case ProcInst:
			pi, pErr := doc.CreatePI(v.Target, string(v.Inst))
			if pErr != nil {
				return pErr
			}
			if err := parent.AddChild(pi); err != nil {
				return err
			}
		}
	}
}
