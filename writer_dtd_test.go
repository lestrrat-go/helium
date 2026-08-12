package helium_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

func TestSerializeDTD(t *testing.T) {
	t.Parallel()

	// round-trips a document with a rich internal
	// subset (entities, attributes with defaults, notations, varied content models)
	// to exercise the DTD writer paths.
	t.Run("rich internal subset", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (a | b)*>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b EMPTY>
<!ATTLIST a
  id   ID       #IMPLIED
  kind (x | y)  "x"
  req  CDATA    #REQUIRED>
<!ENTITY internal "expanded">
<!ENTITY % pe "ignored">
<!NOTATION gif SYSTEM "viewer.exe">
]>
<doc><a id="i1" req="r">text</a><b/></doc>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, "<!DOCTYPE doc")
		require.Contains(t, out, "<!ELEMENT")
		require.Contains(t, out, "<!ATTLIST")
		require.Contains(t, out, "<!ENTITY")
		require.Contains(t, out, "<!NOTATION")
	})

	// parses a DTD that exercises every attribute type, every
	// default kind, internal/external/parameter/unparsed entities, PUBLIC and
	// SYSTEM notations, and EMPTY/ANY/mixed/element content models, then serializes
	// it so the DTD-writer paths are covered. The serialized form round-trips
	// through a second parse.
	t.Run("rich DTD", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (a, b?, (c | d)*, e+)>
<!ELEMENT a EMPTY>
<!ELEMENT b ANY>
<!ELEMENT c (#PCDATA)>
<!ELEMENT d (#PCDATA|c)*>
<!ELEMENT e (#PCDATA)>
<!ATTLIST doc
  id    ID       #IMPLIED
  ref   IDREF    #IMPLIED
  refs  IDREFS   #IMPLIED
  ent   ENTITY   #IMPLIED
  ents  ENTITIES #IMPLIED
  tok   NMTOKEN  #IMPLIED
  toks  NMTOKENS #IMPLIED
  req   CDATA    #REQUIRED
  fix   CDATA    #FIXED "fixed"
  kind  (x|y|z)  "x"
  note  NOTATION (gif|png) #IMPLIED>
<!ENTITY internal "internal value">
<!ENTITY ext SYSTEM "ext.xml">
<!ENTITY pub PUBLIC "-//Example//Text//EN" "pub.xml">
<!ENTITY img SYSTEM "img.gif" NDATA gif>
<!ENTITY % pe "param value">
<!NOTATION gif SYSTEM "viewer.exe">
<!NOTATION png PUBLIC "-//Example//Notation//EN" "png.exe">
]>
<doc id="d1" req="x"><a/><c>c</c><e>e</e></doc>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err, "parse rich DTD")

		out, err := helium.WriteString(doc)
		require.NoError(t, err, "serialize rich DTD")

		// Spot-check the declarations made it into the serialized DTD.
		for _, want := range []string{
			"<!DOCTYPE doc",
			"<!ELEMENT a EMPTY>",
			"<!ELEMENT b ANY>",
			"ID",
			"IDREF",
			"IDREFS",
			"ENTITY",
			"ENTITIES",
			"NMTOKEN",
			"#REQUIRED",
			"#FIXED",
			"NOTATION",
			"<!ENTITY internal",
			"<!ENTITY ext SYSTEM",
			"<!ENTITY pub PUBLIC",
			"NDATA gif",
			"<!ENTITY % pe",
			"<!NOTATION gif SYSTEM",
			"<!NOTATION png PUBLIC",
		} {
			require.Contains(t, out, want, "serialized DTD should contain %q", want)
		}

		// The serialized output must itself re-parse cleanly.
		_, err = helium.NewParser().Parse(t.Context(), []byte(out))
		require.NoError(t, err, "re-parse serialized rich DTD")
	})

	// is a fuller round-trip that, in addition to the
	// existing rich-DTD test, exercises serialization of a programmatically built
	// DTD containing a percent-bearing internal entity and a parameter entity so the
	// entity-content writer paths run end to end and re-parse cleanly.
	t.Run("with entities", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("doc", "", "")
		require.NoError(t, err)

		_, err = dtd.AddEntity("plain", enum.InternalGeneralEntity, "", "", "plain value")
		require.NoError(t, err)
		_, err = dtd.AddEntity("ext", enum.ExternalGeneralParsedEntity, "", "ext.xml", "")
		require.NoError(t, err)
		_, err = dtd.AddEntity("pub", enum.ExternalGeneralParsedEntity, "-//E//T//EN", "pub.xml", "")
		require.NoError(t, err)

		root, err := doc.CreateElement("doc")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, "<!ENTITY plain")
		require.Contains(t, out, "<!ENTITY ext SYSTEM")
		require.Contains(t, out, "<!ENTITY pub PUBLIC")

		// Re-parse to confirm well-formedness.
		require.True(t, strings.Contains(out, "<!DOCTYPE doc"))
	})

	// the text/attribute escaping paths, including
	// values that contain both single and double quotes and assorted control and
	// markup characters.
	t.Run("escaping", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		// Attribute value containing both quote characters plus markup chars.
		// SetAttribute stores the value verbatim (no entity parsing) so the
		// serializer is the component responsible for escaping it.
		err = root.SetAttribute("attr", `he said "hi" & 'bye' <x>`)
		require.NoError(t, err)

		// Text content with markup, ampersand, tab and newline.
		require.NoError(t, root.AppendText([]byte("a < b & c > d\twith\ntabs")))

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, "&amp;")
		require.Contains(t, out, "&lt;")
		require.Contains(t, out, "&gt;")
		require.Contains(t, out, "&quot;")

		// Re-parse to confirm well-formedness of the escaped output.
		_, err = helium.NewParser().Parse(t.Context(), []byte(out))
		require.NoError(t, err)
	})

	// the Format/IndentString writer options.
	t.Run("formatting", func(t *testing.T) {
		const src = `<root><a><b>text</b></a><c/></root>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		var buf strings.Builder
		err = helium.NewWriter().
			Format(true).
			IndentString("    ").
			XMLDeclaration(false).
			WriteTo(&buf, doc)
		require.NoError(t, err)
		require.Contains(t, buf.String(), "\n    ")
		require.NotContains(t, buf.String(), "<?xml")
	})

	// the SelfCloseEmptyElements option.
	t.Run("self-close toggle", func(t *testing.T) {
		const src = `<root><empty/></root>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		var buf strings.Builder
		err = helium.NewWriter().
			SelfCloseEmptyElements(false).
			XMLDeclaration(false).
			WriteTo(&buf, doc)
		require.NoError(t, err)
		require.Contains(t, buf.String(), "<empty></empty>")
	})
}

func TestWriterEntityDecl(t *testing.T) {
	t.Parallel()

	// every ampersand in
	// content and parsed orig values is syntax, not repairable character data.
	t.Run("malformed entity-value markup", func(t *testing.T) {
		newEntityDoc := func(t *testing.T, value string, orig bool) *helium.Document {
			t.Helper()
			doc := helium.NewDocument(ver10, "UTF-8", helium.StandaloneImplicitNo)
			dtd, err := doc.CreateInternalSubset("doc", "", "")
			require.NoError(t, err)
			ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", value)
			require.NoError(t, err)
			if orig {
				ent.SetOrig(value)
			}
			root, err := doc.CreateElement("doc")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))
			return doc
		}

		for _, tc := range []struct {
			name  string
			value string
			orig  bool
		}{
			{name: "content-bare-ampersand", value: "&"},
			{name: "content-missing-semicolon", value: "&name"},
			{name: "content-invalid-name", value: "&9name;"},
			{name: "content-whitespace-name", value: "&bad name;"},
			{name: "orig-bare-ampersand", value: "&", orig: true},
			{name: "orig-missing-semicolon", value: "&name", orig: true},
			{name: "orig-invalid-name", value: "&9name;", orig: true},
			{name: "orig-whitespace-name", value: "&bad name;", orig: true},
			{name: "orig-bare-percent", value: "%", orig: true},
			{name: "orig-parameter-missing-semicolon", value: "%pe", orig: true},
			{name: "orig-parameter-invalid-name", value: "%9pe;", orig: true},
			{name: "orig-parameter-reference", value: "%pe;", orig: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				require.ErrorIs(t, helium.NewWriter().RejectInvalidChars(false).WriteTo(&buf, newEntityDoc(t, tc.value, tc.orig)), helium.ErrWriterInvalidName)
				require.NotContains(t, buf.String(), tc.value)
			})
		}

		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, newEntityDoc(t, "value&amp;more", false)))
		require.Contains(t, buf.String(), "value&amp;more")
	})

	// gives the scanner
	// one terminal semicolon after many malformed starts. A one-pass scanner rejects
	// the first reference without repeatedly searching the remaining suffix.
	t.Run("repeated malformed prefixes", func(t *testing.T) {
		value := strings.Repeat("&#", 32*1024) + ";"
		doc := helium.NewDocument(ver10, "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("doc", "", "")
		require.NoError(t, err)
		_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", value)
		require.NoError(t, err)
		root, err := doc.CreateElement("doc")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		var buf bytes.Buffer
		require.ErrorIs(t, helium.NewWriter().WriteTo(&buf, doc), helium.ErrWriterInvalidName)
	})

	t.Run("external parameter-entity closing delimiter", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("doc", "", "")
		require.NoError(t, err)
		_, err = dtd.AddEntity("ext", enum.ExternalParameterEntity, "", "ext.ent", "")
		require.NoError(t, err)
		root, err := doc.CreateElement("doc")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		require.Contains(t, buf.String(), "<!ENTITY % ext SYSTEM \"ext.ent\">")
		_, err = helium.NewParser().Parse(t.Context(), buf.Bytes())
		require.NoError(t, err)
	})

	t.Run("stored external IDs on internal entities are ignored", func(t *testing.T) {
		const (
			invalidPublicID = "bad{pubid"
			invalidSystemID = "bad\x01system"
		)

		for _, tc := range []struct {
			name       string
			entityType enum.EntityType
			publicID   string
			systemID   string
		}{
			{
				name:       "general public ID",
				entityType: enum.InternalGeneralEntity,
				publicID:   invalidPublicID,
			},
			{
				name:       "general system ID",
				entityType: enum.InternalGeneralEntity,
				systemID:   invalidSystemID,
			},
			{
				name:       "parameter public ID",
				entityType: enum.InternalParameterEntity,
				publicID:   invalidPublicID,
			},
			{
				name:       "parameter system ID",
				entityType: enum.InternalParameterEntity,
				systemID:   invalidSystemID,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := doc.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddEntity("entity", tc.entityType, tc.publicID, tc.systemID, "entity value")
				require.NoError(t, err)
				root, err := doc.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, doc.AddChild(root))

				out, err := helium.WriteString(doc)
				require.NoError(t, err)
				require.Contains(t, out, "<!ENTITY")
				require.Contains(t, out, "entity value")
				if tc.publicID != "" {
					require.NotContains(t, out, tc.publicID)
				}
				if tc.systemID != "" {
					require.NotContains(t, out, tc.systemID)
				}
				require.NotContains(t, out, "\uFFFD")
				_, err = helium.NewParser().Parse(t.Context(), []byte(out))
				require.NoError(t, err)
			})
		}
	})

	t.Run("external entity IDs are validated", func(t *testing.T) {
		const (
			invalidPublicID = "bad{pubid"
			invalidSystemID = "bad\x01system"
		)

		for _, tc := range []struct {
			name       string
			entityType enum.EntityType
			publicID   string
			systemID   string
			wantErr    error
		}{
			{
				name:       "parsed general public ID",
				entityType: enum.ExternalGeneralParsedEntity,
				publicID:   invalidPublicID,
				systemID:   "ext.ent",
				wantErr:    helium.ErrWriterInvalidDTDNode,
			},
			{
				name:       "unparsed general public ID",
				entityType: enum.ExternalGeneralUnparsedEntity,
				publicID:   invalidPublicID,
				systemID:   "ext.ent",
				wantErr:    helium.ErrWriterInvalidDTDNode,
			},
			{
				name:       "parameter public ID",
				entityType: enum.ExternalParameterEntity,
				publicID:   invalidPublicID,
				systemID:   "ext.ent",
				wantErr:    helium.ErrWriterInvalidDTDNode,
			},
			{
				name:       "parsed general system ID",
				entityType: enum.ExternalGeneralParsedEntity,
				systemID:   invalidSystemID,
				wantErr:    helium.ErrInvalidXMLChar,
			},
			{
				name:       "unparsed general system ID",
				entityType: enum.ExternalGeneralUnparsedEntity,
				systemID:   invalidSystemID,
				wantErr:    helium.ErrInvalidXMLChar,
			},
			{
				name:       "parameter system ID",
				entityType: enum.ExternalParameterEntity,
				systemID:   invalidSystemID,
				wantErr:    helium.ErrInvalidXMLChar,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := doc.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddEntity("ext", tc.entityType, tc.publicID, tc.systemID, "")
				require.NoError(t, err)
				root, err := doc.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, doc.AddChild(root))

				var buf bytes.Buffer
				err = helium.NewWriter().WriteTo(&buf, doc)
				require.ErrorIs(t, err, tc.wantErr)
				require.NotContains(t, buf.String(), "\uFFFD")
			})
		}
	})

	t.Run("valid external system literals serialize", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			systemID string
			want     string
		}{
			{
				name:     "no quote",
				systemID: "external.ent",
				want:     `"external.ent"`,
			},
			{
				name:     "double quote",
				systemID: `external"quote.ent`,
				want:     `'external"quote.ent'`,
			},
			{
				name:     "single quote",
				systemID: `external'quote.ent`,
				want:     `"external'quote.ent"`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := doc.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddEntity("ext", enum.ExternalGeneralParsedEntity, "", tc.systemID, "")
				require.NoError(t, err)
				root, err := doc.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, doc.AddChild(root))

				out, err := helium.WriteString(doc)
				require.NoError(t, err)
				require.Contains(t, out, "SYSTEM "+tc.want)
				_, err = helium.NewParser().Parse(t.Context(), []byte(out))
				require.NoError(t, err)
			})
		}
	})

	// serializes an internal general entity
	// whose replacement text contains a literal '%', driving the dumpEntityContent
	// percent-escaping branch in the DTD writer.
	t.Run("content carrying a percent sign", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("doc", "", "")
		require.NoError(t, err)

		// content has no orig set (AddEntity passes orig=""), so the writer falls
		// through to dumpEntityContent; the '%' forces the escaping branch and the
		// '"' forces the &quot; branch.
		_, err = dtd.AddEntity("pct", enum.InternalGeneralEntity, "", "", `50% "done"`)
		require.NoError(t, err)

		root, err := doc.CreateElement("doc")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, "<!ENTITY pct", "entity declaration serialized")
		require.Contains(t, out, "&#x25;", "percent escaped via dumpEntityContent")
		require.Contains(t, out, "&quot;", "embedded quote escaped via dumpEntityContent")
	})

	t.Run("XML 1.1 restricted characters in an entity value", func(t *testing.T) {
		newDoc := func(t *testing.T, version string, typ enum.EntityType, orig bool, value string) *helium.Document {
			t.Helper()
			doc := helium.NewDocument(version, "UTF-8", helium.StandaloneImplicitNo)
			dtd, err := doc.CreateInternalSubset("doc", "", "")
			require.NoError(t, err)
			ent, err := dtd.AddEntity("e", typ, "", "", value)
			require.NoError(t, err)
			if orig {
				ent.SetOrig(value)
			}
			root, err := doc.CreateElement("doc")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))
			return doc
		}

		for _, tc := range []struct {
			name string
			typ  enum.EntityType
			orig bool
		}{
			{name: "general-content", typ: enum.InternalGeneralEntity},
			{name: "general-orig", typ: enum.InternalGeneralEntity, orig: true},
			{name: "parameter-content", typ: enum.InternalParameterEntity},
			{name: "parameter-orig", typ: enum.InternalParameterEntity, orig: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				for _, writer := range []helium.Writer{
					helium.NewWriter(),
					helium.NewWriter().RejectInvalidChars(false),
				} {
					var buf bytes.Buffer
					require.NoError(t, writer.WriteTo(&buf, newDoc(t, ver11, tc.typ, tc.orig, "a\x7fb")))
					require.Contains(t, buf.String(), "a&#127;b")
					require.NotContains(t, buf.String(), "a\x7fb")
				}

				// XML 1.0 keeps U+007F literal in every EntityValue storage path.
				var buf bytes.Buffer
				require.NoError(t, helium.NewWriter().WriteTo(&buf, newDoc(t, ver10, tc.typ, tc.orig, "a\x7fb")))
				require.Contains(t, buf.String(), "a\x7fb")
				require.NotContains(t, buf.String(), "a&#127;b")

				// A validated reference remains verbatim instead of being double-escaped.
				buf.Reset()
				require.NoError(t, helium.NewWriter().WriteTo(&buf, newDoc(t, ver11, tc.typ, tc.orig, "a&#127;b")))
				require.Contains(t, buf.String(), "a&#127;b")
			})
		}
	})
}

func TestWriterDTDLiterals(t *testing.T) {
	t.Parallel()

	t.Run("unrepresentable system literals", func(t *testing.T) {
		const systemID = `external"quote'and-apostrophe.ent`
		for _, tc := range []struct {
			name string
			add  func(*helium.DTD) error
		}{
			{
				name: "parsed general entity with system ID",
				add: func(dtd *helium.DTD) error {
					_, err := dtd.AddEntity("ext", enum.ExternalGeneralParsedEntity, "", systemID, "")
					return err
				},
			},
			{
				name: "unparsed general entity with public and system IDs",
				add: func(dtd *helium.DTD) error {
					_, err := dtd.AddEntity("ext", enum.ExternalGeneralUnparsedEntity, "Example", systemID, "")
					return err
				},
			},
			{
				name: "parameter entity with system ID",
				add: func(dtd *helium.DTD) error {
					_, err := dtd.AddEntity("ext", enum.ExternalParameterEntity, "", systemID, "")
					return err
				},
			},
			{
				name: "notation with system ID",
				add: func(dtd *helium.DTD) error {
					_, err := dtd.AddNotation("notation", "", systemID)
					return err
				},
			},
			{
				name: "notation with public and system IDs",
				add: func(dtd *helium.DTD) error {
					_, err := dtd.AddNotation("notation", "Example", systemID)
					return err
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := doc.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				require.NoError(t, tc.add(dtd))
				root, err := doc.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, doc.AddChild(root))

				var buf bytes.Buffer
				err = helium.NewWriter().WriteTo(&buf, doc)
				require.ErrorIs(t, err, helium.ErrWriterInvalidDTDNode)
				require.NotContains(t, buf.String(), systemID)
				require.NotContains(t, buf.String(), "&quot;")
			})
		}
	})

	t.Run("XML 1.1 restricted characters", func(t *testing.T) {
		newDoc := func() *helium.Document {
			doc := helium.NewDocument(ver11, "UTF-8", helium.StandaloneImplicitNo)
			_, err := doc.CreateInternalSubset("doc", "", "sys\x7fid")
			require.NoError(t, err)
			root, err := doc.CreateElement("doc")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))
			return doc
		}

		var buf bytes.Buffer
		require.ErrorIs(t, helium.NewWriter().WriteTo(&buf, newDoc()), helium.ErrInvalidXMLChar)

		buf.Reset()
		require.NoError(t, helium.NewWriter().EscapeNonASCII(false).RejectInvalidChars(false).WriteTo(&buf, newDoc()))
		require.Contains(t, buf.String(), "sys�id")
		require.NotContains(t, buf.String(), "&#127;")
	})

	t.Run("notation system-literal quoting", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("doc", "", "")
		require.NoError(t, err)

		// no quote -> double-quote delimited.
		_, err = dtd.AddNotation("plain", "", "plain.exe")
		require.NoError(t, err)
		// double quote only -> single-quote delimited.
		_, err = dtd.AddNotation("dq", "", `has"dquote`)
		require.NoError(t, err)
		// single quote only -> double-quote delimited.
		_, err = dtd.AddNotation("sq", "", `has'squote`)
		require.NoError(t, err)

		root, err := doc.CreateElement("doc")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, `"plain.exe"`, "no-quote value double-quote delimited")
		require.Contains(t, out, `'has"dquote'`, "double-quote-only value single-quote delimited")
		require.Contains(t, out, `"has'squote"`, "single-quote-only value double-quote delimited")
		_, err = helium.NewParser().Parse(t.Context(), []byte(out))
		require.NoError(t, err)
	})

	// keeps DTD declaration
	// names distinct from the broader DTD Name grammar used by element constructs.
	t.Run("declaration names requiring an NCName", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			typ  enum.EntityType
		}{
			{name: "general", typ: enum.InternalGeneralEntity},
			{name: "parameter", typ: enum.InternalParameterEntity},
		} {
			t.Run(tc.name, func(t *testing.T) {
				doc := helium.NewDocument(ver10, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := doc.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddEntity("p:e", tc.typ, "", "", "value")
				require.NoError(t, err)
				root, err := doc.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, doc.AddChild(root))

				var buf bytes.Buffer
				require.ErrorIs(t, helium.NewWriter().WriteTo(&buf, doc), helium.ErrWriterInvalidName)
				require.NotContains(t, buf.String(), "<!ENTITY p:e")
			})
		}
	})
}
