package helium_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
	"github.com/stretchr/testify/require"
)

func newEventEmitter(out io.Writer) sax.SAX2Handler {
	entities := map[string]*helium.Entity{}
	s := sax.New()
	s.SetOnSetDocumentLocator(sax.SetDocumentLocatorFunc(func(_ context.Context, loc sax.DocumentLocator) error {
		_, _ = fmt.Fprintf(out, "SAX.SetDocumentLocator()\n")
		return nil
	}))
	s.SetOnAttributeDecl(sax.AttributeDeclFunc(func(_ context.Context, elemName string, attrName string, typ enum.AttributeType, deftype enum.AttributeDefault, defvalue string, enum sax.Enumeration) error {
		// eek, defvalue is an interface, and interface == nil is only true
		// if the interface has no value AND not type, so.. hmmm.
		if defvalue == "" {
			defvalue = "NULL"
		}
		_, _ = fmt.Fprintf(out, "SAX.AttributeDecl(%s, %s, %d, %d, %s, ...)\n", elemName, attrName, typ, deftype, defvalue)
		return nil
	}))
	s.SetOnInternalSubset(sax.InternalSubsetFunc(func(_ context.Context, name, externalID, systemID string) error {
		_, _ = fmt.Fprintf(out, "SAX.InternalSubset(%s, %s, %s)\n", name, externalID, systemID)
		return nil
	}))
	s.SetOnReference(sax.ReferenceFunc(func(_ context.Context, name string) error {
		_, _ = fmt.Fprintf(out, "SAX.Reference(%s)\n", name)
		return nil
	}))
	s.SetOnGetEntity(sax.GetEntityFunc(func(_ context.Context, name string) (sax.Entity, error) {
		_, _ = fmt.Fprintf(out, "SAX.ResolveEntity(%s)\n", name)

		ent, ok := entities[name]
		if !ok {
			return nil, errors.New("entity not found")
		}
		return ent, nil
	}))

	s.SetOnGetParameterEntity(sax.GetParameterEntityFunc(func(_ context.Context, name string) (sax.Entity, error) {
		_, _ = fmt.Fprintf(out, "SAX.ResolveEntity(%s)\n", name)

		ent, ok := entities[name]
		if !ok {
			return nil, errors.New("entity not found")
		}
		return ent, nil
	}))

	s.SetOnEntityDecl(sax.EntityDeclFunc(func(ctxif context.Context, name string, typ enum.EntityType, publicID string, systemID string, notation string) error {
		if pdebug.Enabled {
			g := pdebug.Marker("EntityDecl handler for sax_test.go")
			defer g.End()
		}
		_, _ = fmt.Fprintf(out, "SAX.UnparsedEntityDecl(%s, %d, %s, %s, %s)\n",
			name, typ, publicID, systemID, notation)

		doc := helium.NewDefaultDocument()
		dtd, err := doc.CreateInternalSubset("root", "", "")
		if err != nil {
			return err
		}
		if typ == enum.ExternalGeneralUnparsedEntity {
			if _, err := dtd.AddNotation(notation, "", ""); err != nil {
				return err
			}
		}
		ent, err := dtd.AddEntity(name, typ, publicID, systemID, notation)
		if err != nil {
			return err
		}
		entities[name] = ent
		if pdebug.Enabled {
			pdebug.Printf("registered entity '%s' (entity type = '%s', publicID = '%s', systemID = '%s', notation = '%s')", name, typ, publicID, systemID, notation)
		}
		return nil
	}))
	s.SetOnExternalSubset(sax.ExternalSubsetFunc(func(_ context.Context, name, externalID, systemID string) error {
		_, _ = fmt.Fprintf(out, "SAX.ExternalSubset(%s, %s, %s)\n", name, externalID, systemID)
		return nil
	}))
	s.SetOnElementDecl(sax.ElementDeclFunc(func(_ context.Context, name string, typ enum.ElementType, content sax.ElementContent) error {
		_, _ = fmt.Fprintf(out, "SAX.ElementDecl(%s, %d, ...)\n", name, typ)
		return nil
	}))
	s.SetOnStartDocument(sax.StartDocumentFunc(func(_ context.Context) error {
		_, _ = fmt.Fprintf(out, "SAX.StartDocument()\n")
		return nil
	}))
	s.SetOnEndDocument(sax.EndDocumentFunc(func(_ context.Context) error {
		_, _ = fmt.Fprintf(out, "SAX.EndDocument()\n")
		return nil
	}))
	s.SetOnComment(sax.CommentFunc(func(_ context.Context, data []byte) error {
		_, _ = fmt.Fprintf(out, "SAX.Comment(%s)\n", data)
		return nil
	}))
	charHandler := func(name string, _ context.Context, data []byte) error {
		var output string
		if len(data) > 30 {
			output = string(data[:30])
		} else {
			output = string(data)
		}

		_, _ = fmt.Fprintf(out, "SAX.%s(%s, %d)\n", name, output, len(data))
		return nil
	}
	s.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(ctx context.Context, data []byte) error {
		return charHandler("IgnorableWhitespace", ctx, data)
	}))
	s.SetOnCharacters(sax.CharactersFunc(func(ctx context.Context, data []byte) error {
		return charHandler("Characters", ctx, data)
	}))
	s.SetOnStartElementNS(sax.StartElementNSFunc(func(_ context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		_, _ = fmt.Fprintf(out, "SAX.StartElementNS(%s, ", localname)

		if prefix != "" {
			_, _ = fmt.Fprintf(out, "%s, ", prefix)
		} else {
			_, _ = fmt.Fprintf(out, "NULL, ")
		}

		if uri != "" {
			_, _ = fmt.Fprintf(out, "'%s', ", uri)
		} else {
			_, _ = fmt.Fprintf(out, "NULL, ")
		}

		lns := len(namespaces)
		_, _ = fmt.Fprintf(out, "%d, ", lns)
		for _, ns := range namespaces {
			if prefix := ns.Prefix(); prefix != "" {
				_, _ = fmt.Fprintf(out, "xmlns:%s='%s'", ns.Prefix(), ns.URI())
			} else {
				_, _ = fmt.Fprintf(out, "xmlns='%s'", ns.URI())
			}
			_, _ = fmt.Fprintf(out, ", ")
		}

		defaulted := 0
		for _, attr := range attrs {
			if attr.IsDefault() {
				defaulted++
			}
		}

		_, _ = fmt.Fprintf(out, "%d, %d",
			len(attrs),
			defaulted, /* TODO - number of defaulted attributes */
		)

		if len(attrs) > 0 {
			_, _ = fmt.Fprintf(out, ", ")
			for i, attr := range attrs {
				_, _ = fmt.Fprintf(out, "%s='%.4s...', %d", attr.Name(), attr.Value(), len(attr.Value()))
				if i < len(attrs)-1 {
					_, _ = fmt.Fprintf(out, ", ")
				}
			}
		}

		_, _ = fmt.Fprintln(out, ")")

		return nil
	}))
	s.SetOnEndElementNS(sax.EndElementNSFunc(func(_ context.Context, localname, prefix, uri string) error {
		_, _ = fmt.Fprintf(out, "SAX.EndElementNS(%s, ", localname)

		if prefix != "" {
			_, _ = fmt.Fprintf(out, "%s, ", prefix)
		} else {
			_, _ = fmt.Fprintf(out, "NULL, ")
		}

		if uri != "" {
			_, _ = fmt.Fprintf(out, "'%s')\n", uri)
		} else {
			_, _ = fmt.Fprintf(out, "NULL)\n")
		}

		return nil
	}))
	return s
}

func TestDocumentLocatorIDs(t *testing.T) {
	const baseURI = "test://document.xml"
	var gotPublicID, gotSystemID string

	s := sax.New()
	s.SetOnSetDocumentLocator(sax.SetDocumentLocatorFunc(func(_ context.Context, loc sax.DocumentLocator) error {
		gotPublicID = loc.GetPublicID()
		gotSystemID = loc.GetSystemID()
		return nil
	}))
	s.SetOnStartDocument(sax.StartDocumentFunc(func(_ context.Context) error { return nil }))
	s.SetOnEndDocument(sax.EndDocumentFunc(func(_ context.Context) error { return nil }))
	s.SetOnStartElementNS(sax.StartElementNSFunc(func(_ context.Context, _, _, _ string, _ []sax.Namespace, _ []sax.Attribute) error {
		return nil
	}))
	s.SetOnEndElementNS(sax.EndElementNSFunc(func(_ context.Context, _, _, _ string) error { return nil }))
	s.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, _ []byte) error { return nil }))

	p := helium.NewParser()
	p.SetSAXHandler(s)
	p.SetBaseURI(baseURI)

	_, err := p.Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err, "Parse should succeed")
	require.Equal(t, "", gotPublicID, "GetPublicID should return empty string")
	require.Equal(t, baseURI, gotSystemID, "GetSystemID should return base URI")
}

func TestSAXEvents(t *testing.T) {
	skipped := map[string]struct{}{}

	dir := "test"
	files, err := os.ReadDir(dir)
	require.NoError(t, err, "os.ReadDir should succeed")

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

		goldenfn := strings.ReplaceAll(fn, ".xml", ".sax2")
		if _, err := os.Stat(goldenfn); err != nil {
			continue
		}

		t.Logf("Testing %s...", fn)

		in, err := os.ReadFile(fn)
		require.NoError(t, err, "os.ReadFile should succeed")

		golden, err := os.ReadFile(goldenfn)
		require.NoError(t, err, "os.ReadFile should succeed")

		out := bytes.Buffer{}
		p := helium.NewParser()
		p.SetSAXHandler(newEventEmitter(&out))

		_, err = p.Parse(t.Context(), in)
		if err != nil {
			t.Logf("source XML: %s", in)
		}
		require.NoError(t, err, "Parse should succeed (file = %s)", fn)

		if string(golden) != out.String() {
			errout, err := os.OpenFile(fn+".err", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			if err != nil {
				t.Logf("Failed to file to save output: %s", err)
				return
			}
			defer func() { _ = errout.Close() }()

			_, _ = errout.Write(out.Bytes())
		}
		require.Equal(t, string(golden), out.String(), "SAX event streams should match (file = %s)", fn)
	}
}
