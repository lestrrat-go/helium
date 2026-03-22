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
	return xslt3.CompileStylesheet(ctx, doc)
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
	parts := make([]string, 0, len(seq))
	for _, item := range seq {
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
