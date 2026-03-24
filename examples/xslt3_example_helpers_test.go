package examples_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
)

// exampleMessageReceiver implements xslt3.MessageHandler for examples.
type exampleMessageReceiver struct{}

func (r *exampleMessageReceiver) HandleMessage(msg string, terminate bool) error {
	fmt.Printf("message: %s (terminate=%t)\n", msg, terminate)
	return nil
}

// examplePrimaryItemsReceiver implements xslt3.PrimaryItemsHandler.
type examplePrimaryItemsReceiver struct {
	items xpath3.Sequence
}

func (r *examplePrimaryItemsReceiver) HandlePrimaryItems(seq xpath3.Sequence) error {
	r.items = xpath3.ItemSlice(append([]xpath3.Item(nil), seq.Materialize()...))
	return nil
}

// exampleRawResultReceiver implements xslt3.RawResultHandler.
type exampleRawResultReceiver struct {
	result xpath3.Sequence
}

func (r *exampleRawResultReceiver) HandleRawResult(seq xpath3.Sequence) error {
	r.result = xpath3.ItemSlice(append([]xpath3.Item(nil), seq.Materialize()...))
	return nil
}

// exampleResultDocReceiver implements xslt3.ResultDocumentHandler.
type exampleResultDocReceiver struct {
	docs    map[string]*helium.Document
	outDefs map[string]*xslt3.OutputDef
}

func newExampleResultDocReceiver() *exampleResultDocReceiver {
	return &exampleResultDocReceiver{
		docs:    make(map[string]*helium.Document),
		outDefs: make(map[string]*xslt3.OutputDef),
	}
}

func (r *exampleResultDocReceiver) HandleResultDocument(href string, doc *helium.Document, outDef *xslt3.OutputDef) error {
	r.docs[href] = doc
	if outDef != nil {
		r.outDefs[href] = outDef
	}
	return nil
}

type exampleXSLTResolver map[string]string

func (r exampleXSLTResolver) Resolve(uri string) (io.ReadCloser, error) {
	data, ok := r[uri]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(strings.NewReader(data)), nil
}

func serializeExampleDocument(doc *helium.Document) (string, error) {
	var buf bytes.Buffer
	if err := doc.XML(&buf, helium.WithNoDecl()); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func compileExampleStylesheet(ctx context.Context, src string) (*xslt3.Stylesheet, error) {
	doc, err := helium.Parse(ctx, []byte(src))
	if err != nil {
		return nil, err
	}
	return xslt3.NewCompiler().Compile(ctx, doc)
}

func parseExampleDocument(ctx context.Context, src string) (*helium.Document, error) {
	return helium.Parse(ctx, []byte(src))
}

func serializeExampleResult(doc *helium.Document, outDef *xslt3.OutputDef) (string, error) {
	var buf bytes.Buffer
	if err := xslt3.SerializeResult(&buf, doc, outDef); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func serializeExampleItems(items xpath3.Sequence, doc *helium.Document, outDef *xslt3.OutputDef) (string, error) {
	var buf bytes.Buffer
	if err := xslt3.SerializeItems(&buf, items, doc, outDef); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func formatExampleAtomicSequence(seq xpath3.Sequence) (string, error) {
	parts := make([]string, 0, seq.Len())
	for item := range seq.Items() {
		atomic, ok := item.(xpath3.AtomicValue)
		if !ok {
			return "", fmt.Errorf("unexpected non-atomic item %T", item)
		}
		value, err := xpath3.AtomicToString(atomic)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("%s=%s", atomic.TypeName, value))
	}
	return strings.Join(parts, ", "), nil
}
