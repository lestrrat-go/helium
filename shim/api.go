package shim

import (
	stdxml "encoding/xml"
	"io"
)

const Header = stdxml.Header

var HTMLEntity = stdxml.HTMLEntity

func Marshal(v any) ([]byte, error) {
	return stdxml.Marshal(v)
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return stdxml.MarshalIndent(v, prefix, indent)
}

func NewDecoder(r io.Reader) *Decoder {
	dec, err := newDecoderFromReader(r)
	if err != nil {
		return &Decoder{line: 1, column: 1}
	}
	return dec
}

func NewTokenDecoder(t TokenReader) *Decoder {
	return newDecoderFromTokenReader(t)
}

func NewEncoder(w io.Writer) *Encoder {
	return stdxml.NewEncoder(w)
}

func EscapeText(w io.Writer, s []byte) error {
	return stdxml.EscapeText(w, s)
}

func CopyToken(t Token) Token {
	return stdxml.CopyToken(t)
}
