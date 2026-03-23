package xslt3

import (
	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// MessageReceiver handles xsl:message output during transformation.
// A non-nil error aborts the transform immediately.
type MessageReceiver interface {
	HandleMessage(msg string, terminate bool) error
}

// ResultDocumentReceiver handles secondary result documents produced
// by xsl:result-document. A non-nil error aborts the transform.
type ResultDocumentReceiver interface {
	HandleResultDocument(href string, doc *helium.Document) error
}

// ResultDocumentOutputReceiver receives the effective output definition
// for each secondary result document. A non-nil error aborts the transform.
type ResultDocumentOutputReceiver interface {
	HandleResultDocumentOutput(href string, outDef *OutputDef) error
}

// RawResultReceiver receives the raw XDM result sequence from the primary
// output before it is serialized into the result document tree.
// A non-nil error aborts the transform.
type RawResultReceiver interface {
	HandleRawResult(seq xpath3.Sequence) error
}

// PrimaryItemsReceiver receives non-node items captured from the primary
// output during transformation (needed for json/adaptive serialization).
// A non-nil error aborts the transform.
type PrimaryItemsReceiver interface {
	HandlePrimaryItems(seq xpath3.Sequence) error
}

// AnnotationReceiver receives type annotations and schema declarations
// from schema-aware transformations. A non-nil error aborts the transform.
type AnnotationReceiver interface {
	HandleAnnotations(annotations map[helium.Node]string, declarations xpath3.SchemaDeclarations) error
}

// receiverSet holds the extracted receiver implementations from a single
// receiver object. All fields may be nil.
type receiverSet struct {
	message              MessageReceiver
	resultDocument       ResultDocumentReceiver
	resultDocumentOutput ResultDocumentOutputReceiver
	rawResult            RawResultReceiver
	primaryItems         PrimaryItemsReceiver
	annotations          AnnotationReceiver
}

// extractReceivers type-asserts the receiver object against all known
// receiver interfaces and returns a receiverSet with the matching ones.
func extractReceivers(r any) receiverSet {
	if r == nil {
		return receiverSet{}
	}
	var rs receiverSet
	if v, ok := r.(MessageReceiver); ok {
		rs.message = v
	}
	if v, ok := r.(ResultDocumentReceiver); ok {
		rs.resultDocument = v
	}
	if v, ok := r.(ResultDocumentOutputReceiver); ok {
		rs.resultDocumentOutput = v
	}
	if v, ok := r.(RawResultReceiver); ok {
		rs.rawResult = v
	}
	if v, ok := r.(PrimaryItemsReceiver); ok {
		rs.primaryItems = v
	}
	if v, ok := r.(AnnotationReceiver); ok {
		rs.annotations = v
	}
	return rs
}
