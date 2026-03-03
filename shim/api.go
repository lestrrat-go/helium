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

func Unmarshal(data []byte, v any) error {
	return stdxml.Unmarshal(data, v)
}

func NewDecoder(r io.Reader) *Decoder {
	return stdxml.NewDecoder(r)
}

func NewTokenDecoder(t TokenReader) *Decoder {
	return stdxml.NewTokenDecoder(t)
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
