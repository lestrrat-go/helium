package examples_test

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/lestrrat-go/helium"
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
