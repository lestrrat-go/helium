package helium

import "github.com/lestrrat-go/option"

type Option = option.Interface

type identDocumentEncoding struct{}
type identDocumentStandalone struct{}
type identDocumentVersion struct{}
type identSAX struct{}

type DocumentOption interface {
	Option
	documentOption()
}

type documentOption struct { Option }
func (*documentOption) documentOption() {}

type ParseOption interface {
	Option
	parseOption()
}

type parseOption struct { Option }
func (*parseOption) parseOption() {}

// WithEncoding specifies the encoding of an XML document
func WithEncoding(v string) DocumentOption {
	return &documentOption{option.New(identDocumentEncoding{}, v)}
}

// WithSAX
func WithSAX(v interface{}) ParseOption {
	return &parseOption{option.New(identSAX{}, v)}
}


// WithStandalone specifies if the XML is a standlone XML document or not
func WithStandalone(v DocumentStandaloneType) DocumentOption {
	return &documentOption{option.New(identDocumentStandalone{}, v)}
}

// WithVersion specifies the XML version of an XML document
func WithVersion(v string) DocumentOption {
	return &documentOption{option.New(identDocumentVersion{}, v)}
}

