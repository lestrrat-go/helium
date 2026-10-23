package helium_test

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

func TestXML11LineEndings(t *testing.T) {
	t.Parallel()
	t.Run("literal contexts", func(t *testing.T) {
		for _, ending := range []struct {
			name, input, xml10 string
		}{
			{"NEL", "\u0085", "\u0085"},
			{"LS", "\u2028", "\u2028"},
			{"CR NEL", "\r\u0085", "\n\u0085"},
			{"CR LF", "\r\n", "\n"},
			{"CR", "\r", "\n"},
		} {
			for _, version := range []string{ver10, ver11} {
				// XML 1.0 controls cover the additional XML 1.1 line endings.
				// Its existing CR handling is outside this change.
				if version == ver10 && strings.Contains(ending.input, "\r") {
					continue
				}
				t.Run(ending.name+" "+version, func(t *testing.T) {
					want := ending.xml10
					if version == ver11 {
						want = "\n"
					}
					for _, template := range []string{
						`<r>VALUE</r>`, `<r><![CDATA[VALUE]]></r>`,
						`<r><!--VALUE--></r>`, `<r><?pi VALUE?></r>`,
						`<r a="VALUE"/>`,
					} {
						source := `<?xml version="` + version + `"?>` + strings.ReplaceAll(template, "VALUE", "a"+ending.input+"b")
						doc, err := helium.NewParser().Parse(t.Context(), []byte(source))
						require.NoError(t, err, template)
						root := doc.DocumentElement()
						if attr, ok := root.GetAttribute("a"); ok {
							require.Equal(t, "a"+strings.ReplaceAll(want, "\n", " ")+"b", attr, template)
							continue
						}
						require.Equal(t, "a"+want+"b", string(root.FirstChild().Content()), template)
					}
				})
			}
		}
	})

	t.Run("entry points and read boundaries", func(t *testing.T) {
		const source = "<?xml version=\"1.1\"?><r>a\r\u0085b\u0085c\u2028d\r\ne\rf&#x85;&#x2028;</r>"
		const want = "a\nb\nc\nd\ne\nf\u0085\u2028"
		for _, chunk := range []int{1, 2, 3, 4096} {
			doc, err := helium.NewParser().ParseReader(t.Context(), &parserChunkedReader{data: []byte(source), chunk: chunk})
			require.NoError(t, err)
			require.Equal(t, want, string(doc.DocumentElement().Content()))
		}
		pp := helium.NewParser().NewPushParser(t.Context())
		for i := range len(source) {
			require.NoError(t, pp.Push([]byte(source[i:i+1])))
		}
		doc, err := pp.Close()
		require.NoError(t, err)
		require.Equal(t, want, string(doc.DocumentElement().Content()))

		var text strings.Builder
		h := sax.New()
		h.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, data []byte) error {
			text.Write(data)
			return nil
		}))
		_, err = helium.NewParser().SAXHandler(h).CharBufferSize(4).
			ParseReader(t.Context(), &parserChunkedReader{data: []byte(source), chunk: 1})
		require.NoError(t, err)
		require.Equal(t, want, text.String())
	})

	t.Run("markup and entity reference provenance", func(t *testing.T) {
		source := "<?xml version=\"1.1\"?>\u0085<!DOCTYPE r [" +
			"<!ENTITY literal 'a\u0085b'><!ENTITY ref 'c&#x85;d&#x2028;e'>" +
			"<!ENTITY nested '&ref;'>]>\u2028<r\u0085a='value'>&literal;|&nested;</r>"
		doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(source))
		require.NoError(t, err)
		require.Equal(t, "a\nb|c\u0085d\u2028e", string(doc.DocumentElement().Content()))

		node, err := helium.NewParser().ParseInNodeContext(t.Context(), doc.DocumentElement(), []byte("<c>a\u0085b&#x85;</c>"))
		require.NoError(t, err)
		require.Equal(t, "a\nb\u0085", string(node.Content()))
	})

	t.Run("external input versions", func(t *testing.T) {
		const externalSubsetCase = "external subset"
		for _, version := range []string{"", ver10, ver11} {
			decl := ""
			want := "a\nb\n\u0085"
			if version != "" {
				decl = `<?xml version="` + version + `" encoding="UTF-8"?>`
			}
			if version == ver10 {
				want = "a\u0085b\n\u0085"
			}
			for _, kind := range []string{"general entity", "parameter entity", externalSubsetCase} {
				t.Run(kind+" "+version, func(t *testing.T) {
					var source, external string
					switch kind {
					case "general entity":
						source = `<!DOCTYPE r [<!ENTITY e SYSTEM "external.xml">]>`
						external = "a\u0085b"
					case "parameter entity":
						source = `<!DOCTYPE r [<!ENTITY % pe SYSTEM "external.xml">%pe;]>`
						external = "<!ENTITY e 'a\u0085b'>"
					case externalSubsetCase:
						source = `<!DOCTYPE r SYSTEM "external.xml">`
						external = "<!ENTITY e 'a\u0085b'>"
					}
					source = `<?xml version="1.1"?>` + source + "<r>&e;\u0085&#x85;</r>"
					fsys := fstest.MapFS{"external.xml": &fstest.MapFile{Data: []byte(decl + external)}}
					doc, err := helium.NewParser().BlockXXE(false).SubstituteEntities(true).
						LoadExternalDTD(true).FS(fsys).Parse(t.Context(), []byte(source))
					require.NoError(t, err)
					require.Equal(t, want, string(doc.DocumentElement().Content()))
				})
			}
		}
	})

	t.Run("decoded encodings", func(t *testing.T) {
		for _, e := range contentEncodings {
			if !strings.HasPrefix(e.name, "UTF-") && e.name != ucs4Name && e.name != latin1Name && e.name != "IBM037" {
				continue
			}
			t.Run(e.label, func(t *testing.T) {
				e.version = ver11
				input := e.encode(t, e.firstLineDecl()+"<r>a\r\u0085b\u0085c</r>")
				doc, err := helium.NewParser().ParseReader(t.Context(), &parserChunkedReader{data: input, chunk: 1})
				require.NoError(t, err)
				require.Equal(t, "a\nb\nc", string(doc.DocumentElement().Content()))
			})
		}
	})

	t.Run("line endings forbidden in declarations", func(t *testing.T) {
		for _, ending := range []string{"\u0085", "\u2028"} {
			_, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.1"`+ending+`?><r/>`))
			require.Error(t, err)
		}
	})

	t.Run("line positions", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte("<?xml version=\"1.1\"?><r>\u0085<a/>\r\u0085<b/>\u2028<c/></r>"))
		require.NoError(t, err)
		line := 2
		for n := range helium.Children(doc.DocumentElement()) {
			if n.Type() == helium.ElementNode {
				require.Equal(t, line, n.Line())
				line++
			}
		}
		require.Equal(t, 5, line)
	})
}
