package helium_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat/helium"
	"github.com/lestrrat/helium/sax"
	"github.com/stretchr/testify/assert"
)

func newEventEmitter(out io.Writer) helium.SAX {
	s := sax.New()
	s.SetDocumentLocatorHandler = func(_ sax.Context, loc sax.DocumentLocator) error {
		fmt.Fprintf(out, "SAX.SetDocumentLocator()\n")
		return nil
	}
	s.AttributeDeclHandler = func(_ sax.Context, elemName string, attrName string, typ int, deftype int, defvalue sax.AttributeDefaultValue, enum sax.Enumeration) error {
		if defvalue == nil {
			defvalue = "NULL"
		}
		fmt.Fprintf(out, "SAX.AttributeDecl(%s, %s, %d, %d, %s, ...)\n", elemName, attrName, typ, deftype, defvalue)
		return nil
	}
	s.InternalSubsetHandler = func(_ sax.Context, name, externalID, systemID string) error {
		fmt.Fprintf(out, "SAX.InternalSubset(%s, %s, %s)\n", name, externalID, systemID)
		return nil
	}
	s.ExternalSubsetHandler = func(_ sax.Context, name, externalID, systemID string) error {
		fmt.Fprintf(out, "SAX.ExternalSubset(%s, %s, %s)\n", name, externalID, systemID)
		return nil
	}
	s.ElementDeclHandler = func(_ sax.Context, name string, typ int, content sax.ElementContent) error {
		fmt.Fprintf(out, "SAX.ElementDecl(%s, %d, ...)\n", name, typ)
		return nil
	}
	s.StartDocumentHandler = func(_ sax.Context) error {
		fmt.Fprintf(out, "SAX.StartDocument()\n")
		return nil
	}
	s.EndDocumentHandler = func(_ sax.Context) error {
		fmt.Fprintf(out, "SAX.EndDocument()\n")
		return nil
	}
	s.CommentHandler = func(_ sax.Context, data []byte) error {
		fmt.Fprintf(out, "SAX.Comment(%s)\n", data)
		return nil
	}
	charHandler := func(name string, _ sax.Context, data []byte) error {
		var output string
		if len(data) > 30 {
			output = string(data[:30])
		} else {
			output = string(data)
		}

		fmt.Fprintf(out, "SAX.%s(%s, %d)\n", name, output, len(data))
		return nil
	}
	s.IgnorableWhitespaceHandler = func(ctx sax.Context, data []byte) error {
		return charHandler("IgnorableWhitespace", ctx, data)
	}
	s.CharactersHandler = func(ctx sax.Context, data []byte) error {
		return charHandler("Characters", ctx, data)
	}
	s.StartElementHandler = func(_ sax.Context, elem sax.ParsedElement) error {
		prefix := elem.Prefix()
		if prefix == "" {
			prefix = "NULL"
		}
		uri := elem.URI()
		if uri == "" {
			uri = "NULL"
		}
		attrs := elem.Attributes()

		fmt.Fprintf(out, "SAX.StartElementNS(%s, %s, %s, %d, %d, %d",
			elem.Name(),
			prefix,
			uri,
			0, /* TODO - number of namespaces */
			len(attrs),
			0, /* TODO - number of defaulted attributes */
		)

		if len(attrs) > 0 {
			fmt.Fprintf(out, ", ")
			for i, attr := range attrs {
				fmt.Fprintf(out, "%s='%.4s...', %d", attr.LocalName(), attr.Value(), len(attr.Value()))
				if i < len(attrs)-1 {
					fmt.Fprintf(out, ", ")
				}
			}
		}

		fmt.Fprintln(out, ")")

		return nil
	}
	s.EndElementHandler = func(_ sax.Context, elem sax.ParsedElement) error {
		prefix := elem.Prefix()
		if prefix == "" {
			prefix = "NULL"
		}
		uri := elem.URI()
		if uri == "" {
			uri = "NULL"
		}
		fmt.Fprintf(out, "SAX.EndElementNS(%s, %s, %s)\n", elem.Name(), prefix, uri)
		return nil
	}
	return s
}

func TestSAXEvents(t *testing.T) {
	skipped := map[string]struct{} {
		"xml2.xml": {},
		"att4.xml": {},
		"att5.xml": {},
	}

	dir := "test"
	files, err := ioutil.ReadDir(dir)
	if !assert.NoError(t, err, "ioutil.ReadDir should succeed") {
		return
	}

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}

		if _, ok := skipped[fi.Name()]; ok {
			t.Logf("Skipping test for '%s' for now...", fi.Name())
			continue
		}

		fn := filepath.Join(dir, fi.Name())
		if !strings.HasSuffix(fn, ".xml") {
			continue
		}

		in, err := ioutil.ReadFile(fn)
		if !assert.NoError(t, err, "ioutil.ReadFile should succeed") {
			return
		}

		golden, err := ioutil.ReadFile(strings.Replace(fn, ".xml", ".sax2", -1))
		if !assert.NoError(t, err, "ioutil.ReadFile should succeed") {
			return
		}

		out := bytes.Buffer{}
		p := helium.NewParser()
		p.SetSAXHandler(newEventEmitter(&out))

		_, err = p.Parse(in)
		if !assert.NoError(t, err, "Parse should succeed") {
			t.Logf("source XML: %s", in)
			return
		}

		if !assert.Equal(t, string(golden), string(out.Bytes()), "SAX event streams should match (file = %s)", fn) {
			errout, err := os.OpenFile(fn+".err", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0744)
			if err != nil {
				t.Logf("Failed to file to save output: %s", err)
				return
			}
			defer errout.Close()

			errout.Write(out.Bytes())
			return
		}
	}
}