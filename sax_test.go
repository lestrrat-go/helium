package helium

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat/helium/sax"
	"github.com/stretchr/testify/assert"
)

func newEventEmitter(out io.Writer) SAX {
	entities := map[string]*Entity{}
	s := sax.New()
	s.SetDocumentLocatorHandler = func(_ sax.Context, loc sax.DocumentLocator) error {
		fmt.Fprintf(out, "SAX.SetDocumentLocator()\n")
		return nil
	}
	s.AttributeDeclHandler = func(_ sax.Context, elemName string, attrName string, typ int, deftype int, defvalue sax.AttributeDefaultValue, enum sax.Enumeration) error {
		// eek, defvalue is an interface, and interface == nil is only true
		// if the interface has no value AND not type, so.. hmmm.
		if fmt.Sprintf("%s", defvalue) == "" {
			defvalue = "NULL"
		}
		fmt.Fprintf(out, "SAX.AttributeDecl(%s, %s, %d, %d, %s, ...)\n", elemName, attrName, typ, deftype, defvalue)
		return nil
	}
	s.InternalSubsetHandler = func(_ sax.Context, name, externalID, systemID string) error {
		fmt.Fprintf(out, "SAX.InternalSubset(%s, %s, %s)\n", name, externalID, systemID)
		return nil
	}
	s.ReferenceHandler = func(_ sax.Context, name string) error {
		fmt.Fprintf(out, "SAX.Reference(%s)\n", name)
		return nil
	}
	s.ResolveEntityHandler = func(_ sax.Context, name string, publicID string, baseURI string, systemID string) (sax.Entity, error) {
		fmt.Fprintf(out, "SAX.ResolveEntity(%s)\n", name)

		ent, ok := entities[name]
		if !ok {
			return nil, errors.New("entity not found")
		}
		return ent, nil
	}

	s.UnparsedEntityDeclHandler = func(ctxif sax.Context, name string, typ int, publicID string, systemID string, notation string) error {
		fmt.Fprintf(out, "SAX.UnparsedEntityDecl(%s, %d, %s, %s, %s)\n",
			name, typ, publicID, systemID, notation)

		entities[name] = newEntity(name, EntityType(typ), publicID, systemID, notation, "")
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
	s.StartElementHandler = func(_ sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		fmt.Fprintf(out, "SAX.StartElementNS(%s, ", localname)

		if prefix != "" {
			fmt.Fprintf(out, "%s, ", prefix)
		} else {
			fmt.Fprintf(out, "NULL, ")
		}

		if uri != "" {
			fmt.Fprintf(out, "'%s', ", uri)
		} else {
			fmt.Fprintf(out, "NULL, ")
		}

		lns := len(namespaces)
		fmt.Fprintf(out, "%d, ", lns)
		for _, ns := range namespaces {
			if prefix := ns.Prefix(); prefix != "" {
				fmt.Fprintf(out, "xmlns:%s='%s'", ns.Prefix(), ns.URI())
			} else {
				fmt.Fprintf(out, "xmlns='%s'", ns.URI())
			}
			fmt.Fprintf(out, ", ")
		}

		defaulted := 0
		for _, attr := range attrs {
			if attr.IsDefault() {
				defaulted++
			}
		}

		fmt.Fprintf(out, "%d, %d",
			len(attrs),
			defaulted, /* TODO - number of defaulted attributes */
		)

		if len(attrs) > 0 {
			fmt.Fprintf(out, ", ")
			for i, attr := range attrs {
				fmt.Fprintf(out, "%s='%.4s...', %d", attr.Name(), attr.Value(), len(attr.Value()))
				if i < len(attrs)-1 {
					fmt.Fprintf(out, ", ")
				}
			}
		}

		fmt.Fprintln(out, ")")

		return nil
	}
	s.EndElementHandler = func(_ sax.Context, localname, prefix, uri string) error {
		fmt.Fprintf(out, "SAX.EndElementNS(%s, ", localname)

		if prefix != "" {
			fmt.Fprintf(out, "%s, ", prefix)
		} else {
			fmt.Fprintf(out, "NULL, ")
		}

		if uri != "" {
			fmt.Fprintf(out, "'%s')\n", uri)
		} else {
			fmt.Fprintf(out, "NULL)\n")
		}

		return nil
	}
	return s
}

func TestSAXEvents(t *testing.T) {
	skipped := map[string]struct{}{
		"xml2.xml": {},
		"att11.xml": {},
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

		goldenfn := strings.Replace(fn, ".xml", ".sax2", -1)
		if _, err := os.Stat(goldenfn); err != nil {
			continue
		}

		t.Logf("Testing %s...", fn)

		in, err := ioutil.ReadFile(fn)
		if !assert.NoError(t, err, "ioutil.ReadFile should succeed") {
			return
		}

		golden, err := ioutil.ReadFile(goldenfn)
		if !assert.NoError(t, err, "ioutil.ReadFile should succeed") {
			return
		}

		out := bytes.Buffer{}
		p := NewParser()
		p.SetSAXHandler(newEventEmitter(&out))

		_, err = p.Parse(in)
		if !assert.NoError(t, err, "Parse should succeed (file = %s)", fn) {
			t.Logf("source XML: %s", in)
			return
		}

		if !assert.Equal(t, string(golden), string(out.Bytes()), "SAX event streams should match (file = %s)", fn) {
			errout, err := os.OpenFile(fn+".err", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
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