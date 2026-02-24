package helium

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

func nullOrString(s string) string {
	if s == "" {
		return "(null)"
	}
	return s
}

func newLibxml2EventEmitter(out io.Writer) sax.SAX2Handler {
	entities := map[string]*Entity{}
	peEntities := map[string]*Entity{}
	s := sax.New()

	s.SetDocumentLocatorHandler = func(_ sax.Context, _ sax.DocumentLocator) error {
		_, _ = fmt.Fprintf(out, "SAX.setDocumentLocator()\n")
		return nil
	}
	s.StartDocumentHandler = func(_ sax.Context) error {
		_, _ = fmt.Fprintf(out, "SAX.startDocument()\n")
		return nil
	}
	s.EndDocumentHandler = func(_ sax.Context) error {
		_, _ = fmt.Fprintf(out, "SAX.endDocument()\n")
		return nil
	}
	s.InternalSubsetHandler = func(_ sax.Context, name, externalID, systemID string) error {
		_, _ = fmt.Fprintf(out, "SAX.internalSubset(%s, %s, %s)\n", name, externalID, systemID)
		return nil
	}
	s.ExternalSubsetHandler = func(_ sax.Context, name, externalID, systemID string) error {
		_, _ = fmt.Fprintf(out, "SAX.externalSubset(%s, %s, %s)\n", name, externalID, systemID)
		return nil
	}
	s.EntityDeclHandler = func(_ sax.Context, name string, typ int, publicID string, systemID string, content string) error {
		// External entities (types 2, 3, 5) have no content — libxml2 prints (null).
		contentStr := content
		et := EntityType(typ)
		if content == "" && (et == ExternalGeneralParsedEntity || et == ExternalGeneralUnparsedEntity || et == ExternalParameterEntity) {
			contentStr = "(null)"
		}
		_, _ = fmt.Fprintf(out, "SAX.entityDecl(%s, %d, %s, %s, %s)\n",
			name, typ, nullOrString(publicID), nullOrString(systemID), contentStr)
		ent := newEntity(name, EntityType(typ), publicID, systemID, content, "")
		et = EntityType(typ)
		if et == InternalParameterEntity || et == ExternalParameterEntity {
			peEntities[name] = ent
		} else {
			entities[name] = ent
		}
		return nil
	}
	s.UnparsedEntityDeclHandler = func(_ sax.Context, name string, publicID string, systemID string, notationName string) error {
		_, _ = fmt.Fprintf(out, "SAX.unparsedEntityDecl(%s, %s, %s, %s)\n",
			name, nullOrString(publicID), systemID, notationName)
		return nil
	}
	s.NotationDeclHandler = func(_ sax.Context, name string, publicID string, systemID string) error {
		_, _ = fmt.Fprintf(out, "SAX.notationDecl(%s, %s, %s)\n", name, nullOrString(publicID), nullOrString(systemID))
		return nil
	}
	s.AttributeDeclHandler = func(_ sax.Context, elemName string, attrName string, typ int, deftype int, defvalue string, _ sax.Enumeration) error {
		if defvalue == "" {
			defvalue = "NULL"
		}
		_, _ = fmt.Fprintf(out, "SAX.attributeDecl(%s, %s, %d, %d, %s, ...)\n", elemName, attrName, typ, deftype, defvalue)
		return nil
	}
	s.ElementDeclHandler = func(_ sax.Context, name string, typ int, _ sax.ElementContent) error {
		_, _ = fmt.Fprintf(out, "SAX.elementDecl(%s, %d, ...)\n", name, typ)
		return nil
	}
	s.GetEntityHandler = func(_ sax.Context, name string) (sax.Entity, error) {
		_, _ = fmt.Fprintf(out, "SAX.getEntity(%s)\n", name)
		ent, ok := entities[name]
		if !ok {
			return nil, nil
		}
		return ent, nil
	}
	s.GetParameterEntityHandler = func(_ sax.Context, name string) (sax.Entity, error) {
		_, _ = fmt.Fprintf(out, "SAX.getParameterEntity(%s)\n", name)
		ent, ok := peEntities[name]
		if !ok {
			return nil, nil
		}
		return ent, nil
	}
	s.ReferenceHandler = func(_ sax.Context, name string) error {
		_, _ = fmt.Fprintf(out, "SAX.reference(%s)\n", name)
		return nil
	}
	s.CommentHandler = func(_ sax.Context, data []byte) error {
		_, _ = fmt.Fprintf(out, "SAX.comment(%s)\n", data)
		return nil
	}
	s.ProcessingInstructionHandler = func(_ sax.Context, target string, data string) error {
		_, _ = fmt.Fprintf(out, "SAX.processingInstruction(%s, %s)\n", target, data)
		return nil
	}
	s.CDataBlockHandler = func(_ sax.Context, data []byte) error {
		output := string(data)
		if len(output) > 20 {
			output = output[:20]
		}
		_, _ = fmt.Fprintf(out, "SAX.pcdata(%s, %d)\n", output, len(data))
		return nil
	}
	charHandler := func(name string, _ sax.Context, data []byte) error {
		output := string(data)
		if len(output) > 30 {
			output = output[:30]
		}
		_, _ = fmt.Fprintf(out, "SAX.%s(%s, %d)\n", name, output, len(data))
		return nil
	}
	s.CharactersHandler = func(ctx sax.Context, data []byte) error {
		return charHandler("characters", ctx, data)
	}
	// libxml2 in non-validating mode always emits characters(), never
	// ignorableWhitespace(). Map accordingly.
	s.IgnorableWhitespaceHandler = func(ctx sax.Context, data []byte) error {
		return charHandler("characters", ctx, data)
	}
	s.StartElementNSHandler = func(_ sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		_, _ = fmt.Fprintf(out, "SAX.startElementNs(%s, ", localname)

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
			if p := ns.Prefix(); p != "" {
				_, _ = fmt.Fprintf(out, "xmlns:%s='%s'", p, ns.URI())
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
			defaulted,
		)

		if len(attrs) > 0 {
			_, _ = fmt.Fprintf(out, ", ")
			for i, attr := range attrs {
				// Truncate attribute value preview at 4 bytes (matching
				// libxml2's C-level %.4s which is byte-based, unlike Go's
				// %.4s which is rune-based).
				val := attr.Value()
				preview := val
				if len(preview) > 4 {
					preview = preview[:4]
				}
				_, _ = fmt.Fprintf(out, "%s='%s...', %d", attr.Name(), preview, len(val))
				if i < len(attrs)-1 {
					_, _ = fmt.Fprintf(out, ", ")
				}
			}
		}

		_, _ = fmt.Fprintln(out, ")")
		return nil
	}
	s.EndElementNSHandler = func(_ sax.Context, localname, prefix, uri string) error {
		_, _ = fmt.Fprintf(out, "SAX.endElementNs(%s, ", localname)

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
	}
	return s
}

// TestLibxml2CompatSAX2 runs helium's SAX2 event stream against libxml2's
// SAX2 golden files (.sax2.expected) in testdata/libxml2-compat/.
//
// Environment variable HELIUM_LIBXML2_SAX2_TEST_FILES can be set to test
// only specific files:
//
//	HELIUM_LIBXML2_SAX2_TEST_FILES=att1,xml2 go test -run TestLibxml2CompatSAX2
func TestLibxml2CompatSAX2(t *testing.T) {
	dir := "testdata/libxml2-compat"

	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/libxml2-compat not found; run testdata/libxml2/generate.sh first")
	}

	skipped := map[string]string{
		// Character event splitting: libxml2 emits multiple smaller characters()
		// events at buffer boundaries; helium may merge or split differently.
		"isolat1":                          "character event splitting differs",
		"isolat2":                          "character event splitting differs",
		"icu_parse_test.xml":               "character event splitting differs",
		"rdf2":                             "character event splitting differs",
		"winblanks.xml":                    "character event splitting differs",
		"text-4-byte-UTF-16-BE.xml":        "character event splitting differs",
		"text-4-byte-UTF-16-BE-offset.xml": "character event splitting differs",
		"text-4-byte-UTF-16-LE.xml":        "character event splitting differs",
		"text-4-byte-UTF-16-LE-offset.xml": "character event splitting differs",

		// Parser behavior differences: entity handling, namespace propagation,
		// default attributes, or other structural differences.
		"undeclared-entity.xml": "requires SAX Warning callback (not in interface)",
	}

	only := map[string]struct{}{}
	if v := os.Getenv("HELIUM_LIBXML2_SAX2_TEST_FILES"); v != "" {
		for _, f := range strings.Split(v, ",") {
			only[strings.TrimSpace(f)] = struct{}{}
		}
	}

	files, err := os.ReadDir(dir)
	require.NoError(t, err, "os.ReadDir should succeed")

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}

		name := fi.Name()

		// Skip golden/err files — only process XML input files
		if strings.HasSuffix(name, ".expected") || strings.HasSuffix(name, ".err") ||
			strings.HasSuffix(name, ".sax2.expected") || strings.HasSuffix(name, ".sax2.err") {
			continue
		}

		// Check if a SAX2 golden file exists for this input
		sax2ExpectedPath := filepath.Join(dir, name+".sax2.expected")
		if _, err := os.Stat(sax2ExpectedPath); err != nil {
			continue
		}

		if len(only) > 0 {
			if _, ok := only[name]; !ok {
				continue
			}
		}

		if reason, ok := skipped[name]; ok {
			t.Logf("Skipping %s: %s", name, reason)
			continue
		}

		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()

			input, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err, "reading input file")

			expected, err := os.ReadFile(sax2ExpectedPath)
			require.NoError(t, err, "reading expected SAX2 file")

			var buf bytes.Buffer
			p := NewParser()
			p.SetSAXHandler(newLibxml2EventEmitter(&buf))

			_, err = p.Parse(input)
			if err != nil {
				t.Logf("source XML: %s", input)
			}
			require.NoError(t, err, "Parse should succeed (file = %s)", name)

			actual := buf.String()
			if string(expected) != actual {
				errPath := filepath.Join(dir, name+".sax2.err")
				_ = os.WriteFile(errPath, []byte(actual), 0600)
				t.Logf("Actual output saved to %s", errPath)
			}
			require.Equal(t, string(expected), actual, "SAX2 event streams should match (file = %s)", name)
		})
	}
}
