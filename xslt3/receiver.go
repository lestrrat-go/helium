package xslt3

import (
	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// MessageHandler handles xsl:message output during transformation.
// A non-nil error aborts the transform immediately.
//
// Handler methods are called from the goroutine executing Do/Serialize/WriteTo.
// If you run transforms concurrently, your implementation must be safe for
// concurrent use.
type MessageHandler interface {
	HandleMessage(msg string, terminate bool) error
}

// ResultDocumentHandler handles secondary result documents produced
// by xsl:result-document. The outDef contains the effective output
// definition (method, encoding, indent, etc.) for this result document.
// A non-nil error aborts the transform.
//
// Handler methods are called from the goroutine executing Do/Serialize/WriteTo.
// If you run transforms concurrently, your implementation must be safe for
// concurrent use.
type ResultDocumentHandler interface {
	HandleResultDocument(href string, doc *helium.Document, outDef *OutputDef) error
}

// RawResultHandler receives the raw XDM result sequence from the primary
// output before it is serialized into the result document tree.
// A non-nil error aborts the transform.
//
// Handler methods are called from the goroutine executing Do/Serialize/WriteTo.
// If you run transforms concurrently, your implementation must be safe for
// concurrent use.
type RawResultHandler interface {
	HandleRawResult(seq xpath3.Sequence) error
}

// PrimaryItemsHandler receives non-node items captured from the primary
// output during transformation (needed for json/adaptive serialization).
// A non-nil error aborts the transform.
//
// Handler methods are called from the goroutine executing Do/Serialize/WriteTo.
// If you run transforms concurrently, your implementation must be safe for
// concurrent use.
type PrimaryItemsHandler interface {
	HandlePrimaryItems(seq xpath3.Sequence) error
}

// AnnotationHandler receives type annotations and schema declarations
// from schema-aware transformations. A non-nil error aborts the transform.
//
// Handler methods are called from the goroutine executing Do/Serialize/WriteTo.
// If you run transforms concurrently, your implementation must be safe for
// concurrent use.
type AnnotationHandler interface {
	HandleAnnotations(annotations map[helium.Node]string, declarations xpath3.SchemaDeclarations) error
}
