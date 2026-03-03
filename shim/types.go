package shim

import (
	stdxml "encoding/xml"
)

type Name = stdxml.Name

type Attr = stdxml.Attr

type Token = stdxml.Token

type StartElement = stdxml.StartElement

type EndElement = stdxml.EndElement

type CharData = stdxml.CharData

type Comment = stdxml.Comment

type Directive = stdxml.Directive

type ProcInst = stdxml.ProcInst

type Encoder = stdxml.Encoder

type Marshaler = stdxml.Marshaler

type MarshalerAttr = stdxml.MarshalerAttr

type Unmarshaler = stdxml.Unmarshaler

type UnmarshalerAttr = stdxml.UnmarshalerAttr

type TokenReader = stdxml.TokenReader

type SyntaxError = stdxml.SyntaxError

type TagPathError = stdxml.TagPathError

type UnsupportedTypeError = stdxml.UnsupportedTypeError

type UnmarshalError = stdxml.UnmarshalError
