package shim

import (
	"bytes"
	"context"
	stdxml "encoding/xml"
	"errors"
	"io"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
)

type tokenEvent struct {
	tok Token
	err error
}

type Decoder struct {
	tokenReader TokenReader
	events      chan tokenEvent
	initErr     error
	lastToken   Token
	offset      int64
	line        int
	column      int
}

func newDecoderFromReader(r io.Reader) (*Decoder, error) {
	d := &Decoder{
		events: make(chan tokenEvent, 64),
		line:   1,
		column: 1,
	}
	d.startSAXEmitter(r)
	return d, nil
}

func newDecoderFromTokenReader(tr TokenReader) *Decoder {
	return &Decoder{
		tokenReader: tr,
		line:        1,
		column:      1,
	}
}

func (d *Decoder) startSAXEmitter(r io.Reader) {
	push := func(tok Token) error {
		d.events <- tokenEvent{tok: stdxml.CopyToken(tok)}
		return nil
	}

	h := sax.New()
	h.OnStartDocument = sax.StartDocumentFunc(func(_ sax.Context) error { return nil })
	h.OnEndDocument = sax.EndDocumentFunc(func(_ sax.Context) error { return nil })
	h.OnSetDocumentLocator = sax.SetDocumentLocatorFunc(func(_ sax.Context, _ sax.DocumentLocator) error { return nil })
	h.OnStartElementNS = sax.StartElementNSFunc(func(_ sax.Context, localname, _ string, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		attrNS := make(map[string]string, len(namespaces))
		for _, ns := range namespaces {
			attrNS[ns.Prefix()] = ns.URI()
		}

		se := StartElement{Name: Name{Space: uri, Local: localname}}
		if len(attrs) > 0 {
			se.Attr = make([]Attr, 0, len(attrs))
			for _, attr := range attrs {
				space := ""
				if p := attr.Prefix(); p != "" {
					space = attrNS[p]
				}
				se.Attr = append(se.Attr, Attr{
					Name:  Name{Space: space, Local: attr.LocalName()},
					Value: attr.Value(),
				})
			}
		}
		return push(se)
	})
	h.OnEndElementNS = sax.EndElementNSFunc(func(_ sax.Context, localname, _ string, uri string) error {
		return push(EndElement{Name: Name{Space: uri, Local: localname}})
	})
	h.OnCharacters = sax.CharactersFunc(func(_ sax.Context, ch []byte) error {
		return push(CharData(append([]byte(nil), ch...)))
	})
	h.OnIgnorableWhitespace = sax.IgnorableWhitespaceFunc(func(_ sax.Context, ch []byte) error {
		return push(CharData(append([]byte(nil), ch...)))
	})
	h.OnCDataBlock = sax.CDataBlockFunc(func(_ sax.Context, value []byte) error {
		return push(CharData(append([]byte(nil), value...)))
	})
	h.OnComment = sax.CommentFunc(func(_ sax.Context, value []byte) error {
		return push(Comment(append([]byte(nil), value...)))
	})
	h.OnProcessingInstruction = sax.ProcessingInstructionFunc(func(_ sax.Context, target, data string) error {
		return push(ProcInst{Target: target, Inst: []byte(data)})
	})
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
		_, err := p.ParseReader(context.Background(), r)
		if err != nil {
			d.events <- tokenEvent{err: err}
		}
	}()
}

func (d *Decoder) advancePosition(tok Token) {
	b, err := tokenBytes(tok)
	if err != nil {
		return
	}
	d.offset += int64(len(b))
	for _, ch := range b {
		if ch == '\n' {
			d.line++
			d.column = 1
			continue
		}
		d.column++
	}
}

func tokenBytes(tok Token) ([]byte, error) {
	var buf bytes.Buffer
	enc := stdxml.NewEncoder(&buf)
	if err := enc.EncodeToken(stdxml.CopyToken(tok)); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (d *Decoder) Token() (Token, error) {
	var (
		tok Token
	)

	if d.tokenReader != nil {
		nextTok, err := d.tokenReader.Token()
		if err != nil {
			return nil, err
		}
		tok = nextTok
	} else {
		event, ok := <-d.events
		if !ok {
			return nil, io.EOF
		}
		if event.err != nil {
			return nil, event.err
		}
		tok = event.tok
	}

	tok = stdxml.CopyToken(tok)
	d.lastToken = tok
	d.advancePosition(tok)
	return tok, nil
}

func (d *Decoder) RawToken() (Token, error) {
	return d.Token()
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

	elemBytes, err := d.captureElementBytes(*start)
	if err != nil {
		return err
	}
	return Unmarshal(elemBytes, v)
}

func (d *Decoder) captureElementBytes(start stdxml.StartElement) ([]byte, error) {
	tokens := []stdxml.Token{stdxml.CopyToken(start)}
	depth := 1

	for depth > 0 {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, err
		}
		tokens = append(tokens, stdxml.CopyToken(tok))

		switch tok.(type) {
		case stdxml.StartElement:
			depth++
		case stdxml.EndElement:
			depth--
		}
	}

	var buf bytes.Buffer
	enc := stdxml.NewEncoder(&buf)
	for _, tok := range tokens {
		if err := enc.EncodeToken(tok); err != nil {
			return nil, err
		}
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
