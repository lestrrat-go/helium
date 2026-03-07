package shim

import (
	"bytes"
	stdxml "encoding/xml"
	"io"
)

const Header = stdxml.Header

var HTMLEntity = stdxml.HTMLEntity

func Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	enc.Indent(prefix, indent)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func NewDecoder(r io.Reader) *Decoder {
	dec, _ := newDecoderFromReader(r)
	return dec
}

func NewTokenDecoder(t TokenReader) *Decoder {
	if d, ok := t.(*Decoder); ok {
		return d
	}
	return newDecoderFromTokenReader(t)
}

func EscapeText(w io.Writer, s []byte) error {
	return stdxml.EscapeText(w, s)
}

func CopyToken(t Token) Token {
	return stdxml.CopyToken(t)
}
