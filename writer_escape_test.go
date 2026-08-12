package helium_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/stretchr/testify/require"
)

func TestWriterRejectsInvalidChars(t *testing.T) {
	t.Parallel()

	t.Run("every context", func(t *testing.T) {
		// A C0 control character (U+0001) is invalid in XML 1.0. By default the
		// writer REJECTS it with ErrInvalidXMLChar (the SERE0006 serialization
		// error); RejectInvalidChars(false) opts into U+FFFD replacement.
		textDoc := func() *helium.Document {
			d := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
			r, err := d.CreateElement("r")
			require.NoError(t, err)
			require.NoError(t, d.AddChild(r))
			require.NoError(t, r.AppendText([]byte("a\x01b")))
			return d
		}
		attrDoc := func() *helium.Document {
			d := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
			r, err := d.CreateElement("r")
			require.NoError(t, err)
			require.NoError(t, d.AddChild(r))
			err = r.SetAttribute("v", "x\x01y")
			require.NoError(t, err)
			return d
		}

		// Default (zero-value Writer / NewWriter): an invalid char in text OR in an
		// attribute value is rejected, and the failure matches ErrInvalidXMLChar.
		_, err := helium.WriteString(textDoc())
		require.ErrorIs(t, err, helium.ErrInvalidXMLChar)
		var buf bytes.Buffer
		require.ErrorIs(t, helium.NewWriter().WriteTo(&buf, textDoc()), helium.ErrInvalidXMLChar)
		buf.Reset()
		require.ErrorIs(t, helium.NewWriter().WriteTo(&buf, attrDoc()), helium.ErrInvalidXMLChar)

		// EscapeNonASCII(false) still rejects by default \u2014 the check runs before any
		// char-reference/replacement branch, independent of the escaping setting.
		buf.Reset()
		require.ErrorIs(t, helium.NewWriter().EscapeNonASCII(false).WriteTo(&buf, textDoc()), helium.ErrInvalidXMLChar)

		// RejectInvalidChars(false) replaces the invalid char with U+FFFD. Under the
		// default EscapeNonASCII the replacement is the &#xFFFD; reference (matching
		// libxml2) \u2014 NEVER a bogus &#x1; reference for the out-of-range char.
		buf.Reset()
		require.NoError(t, helium.NewWriter().RejectInvalidChars(false).WriteTo(&buf, textDoc()))
		require.Contains(t, buf.String(), "&#xFFFD;")
		require.NotContains(t, buf.String(), "&#x1;")
		buf.Reset()
		require.NoError(t, helium.NewWriter().RejectInvalidChars(false).WriteTo(&buf, attrDoc()))
		require.Contains(t, buf.String(), "&#xFFFD;")
		require.NotContains(t, buf.String(), "&#x1;")

		// With EscapeNonASCII(false) the replacement is the raw U+FFFD character.
		buf.Reset()
		require.NoError(t, helium.NewWriter().EscapeNonASCII(false).RejectInvalidChars(false).WriteTo(&buf, textDoc()))
		require.Contains(t, buf.String(), "\uFFFD")
		require.NotContains(t, buf.String(), "&#x1;")
		buf.Reset()
		require.NoError(t, helium.NewWriter().EscapeNonASCII(false).RejectInvalidChars(false).WriteTo(&buf, attrDoc()))
		require.Contains(t, buf.String(), "\uFFFD")

		// RejectInvalidChars rejects the control char regardless of the
		// EscapeNonASCII setting (the check runs before char-reference escaping).
		buf.Reset()
		err = helium.NewWriter().RejectInvalidChars(true).WriteTo(&buf, textDoc())
		require.ErrorIs(t, err, helium.ErrInvalidXMLChar)
		buf.Reset()
		err = helium.NewWriter().EscapeNonASCII(false).RejectInvalidChars(true).WriteTo(&buf, textDoc())
		require.ErrorIs(t, err, helium.ErrInvalidXMLChar)

		// A control char in an attribute value is rejected too (escaping covers
		// attribute values, not only text nodes).
		explicitAttrDoc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		r, err := explicitAttrDoc.CreateElement("r")
		require.NoError(t, err)
		require.NoError(t, explicitAttrDoc.AddChild(r))
		err = r.SetAttribute("v", "x\x07y")
		require.NoError(t, err)
		buf.Reset()
		err = helium.NewWriter().RejectInvalidChars(true).WriteTo(&buf, explicitAttrDoc)
		require.ErrorIs(t, err, helium.ErrInvalidXMLChar)

		// A valid document still serializes cleanly under the default rejection.
		okDoc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		e, err := okDoc.CreateElement("r")
		require.NoError(t, err)
		require.NoError(t, okDoc.AddChild(e))
		require.NoError(t, e.AppendText([]byte("plain text\tok")))
		buf.Reset()
		require.NoError(t, helium.NewWriter().WriteTo(&buf, okDoc))
		require.Contains(t, buf.String(), "plain text\tok")
	})

	// the reference-less
	// serialization contexts — comment, PI data, and CDATA-section content — where a
	// character reference is not available. A C0 control (U+0001) is invalid in XML
	// 1.0 and restricted in XML 1.1. U+007F is valid in XML 1.0 but must be a
	// character reference in XML 1.1. Both are unserializable in these contexts. The
	// default policy rejects with ErrInvalidXMLChar; RejectInvalidChars(false)
	// substitutes a raw U+FFFD, never a character reference.
	t.Run("reference-free contexts", func(t *testing.T) {
		build := map[string]func(version, value string) *helium.Document{
			"comment-node": func(v, value string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				r, err := d.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(r))
				require.NoError(t, r.AddChild(d.CreateComment([]byte(value))))
				return d
			},
			"pi-data": func(v, value string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				r, err := d.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(r))
				require.NoError(t, r.AddChild(d.CreatePI("t", value)))
				return d
			},
			"cdata-section-node": func(v, value string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				r, err := d.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(r))
				require.NoError(t, r.AddChild(d.CreateCDATASection([]byte(value))))
				return d
			},
			"cdata-section-element": func(v, value string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				r, err := d.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(r))
				require.NoError(t, r.AppendText([]byte(value)))
				return d
			},
		}

		for _, tc := range []struct {
			name    string
			version string
			value   string
			ref     string
		}{
			{name: "xml10-c0", version: ver10, value: "a\x01b", ref: "&#1;"},
			{name: "xml11-c0", version: ver11, value: "a\x01b", ref: "&#1;"},
			{name: "xml11-restricted", version: ver11, value: "a\x7fb", ref: "&#127;"},
		} {
			for name, mk := range build {
				t.Run(tc.name+"/"+name, func(t *testing.T) {
					t.Parallel()
					w := helium.NewWriter()
					if name == "cdata-section-element" {
						w = w.CDATASectionElements(map[string]struct{}{"{}r": {}})
					}

					// Default policy rejects with ErrInvalidXMLChar.
					var buf bytes.Buffer
					require.ErrorIs(t, w.WriteTo(&buf, mk(tc.version, tc.value)), helium.ErrInvalidXMLChar)

					// Replacement mode substitutes a raw U+FFFD (a reference-less
					// context cannot emit a character reference).
					buf.Reset()
					require.NoError(t, w.RejectInvalidChars(false).WriteTo(&buf, mk(tc.version, tc.value)))
					require.Contains(t, buf.String(), "�")
					require.NotContains(t, buf.String(), tc.ref)
				})
			}
		}
	})

	// every DTD/entity-reference NAME
	// emission site. A name is written verbatim between markup delimiters and has no
	// character-reference (and no U+FFFD) form, so a name carrying a C0 control
	// (U+0001) is REJECTED with ErrWriterInvalidName in BOTH the default and the
	// RejectInvalidChars(false) mode, and in BOTH XML versions (the Name grammar
	// subsumes the character-range check and is version-independent) — mirroring how
	// element/attribute names already behave. These names never reach the writer from
	// a parsed document (the parser validates them); they are reachable only through
	// the tree-construction APIs, which check only for a stray colon.
	t.Run("verbatim names", func(t *testing.T) {
		// Each builder produces a document whose only defect is a control character in
		// one verbatim name; every other name is valid.
		build := map[string]func(t *testing.T, v string) *helium.Document{
			"doctype-name": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				_, err := d.CreateInternalSubset("r\x01", "", "")
				require.NoError(t, err)
				root, err := d.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				return d
			},
			"entity-name": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := d.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddEntity("e\x01", enum.InternalGeneralEntity, "", "", "v")
				require.NoError(t, err)
				root, err := d.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				return d
			},
			"notation-name": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := d.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddNotation("n\x01", "", "sys")
				require.NoError(t, err)
				root, err := d.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				return d
			},
			"ndata-notation-name": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := d.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddEntity("img", enum.ExternalGeneralUnparsedEntity, "", "pic.gif", "nota\x01")
				require.NoError(t, err)
				root, err := d.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				return d
			},
			"element-decl-name": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := d.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddElementDecl("e\x01", enum.EmptyElementType, nil)
				require.NoError(t, err)
				root, err := d.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				return d
			},
			"content-model-child-name": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := d.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				pcdata, err := d.CreateElementContent("", helium.ElementContentPCDATA)
				require.NoError(t, err)
				child, err := d.CreateElementContent("a\x01", helium.ElementContentElement)
				require.NoError(t, err)
				model, err := d.CreateElementContentChoice(pcdata, child, helium.ElementContentMult)
				require.NoError(t, err)
				_, err = dtd.AddElementDecl("m", enum.MixedElementType, model)
				require.NoError(t, err)
				root, err := d.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				return d
			},
			"attlist-element-name": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := d.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddAttributeDecl("el\x01", "a", enum.AttrCDATA, enum.AttrDefaultNone, "", nil)
				require.NoError(t, err)
				root, err := d.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				return d
			},
			"attlist-attr-name": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := d.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddAttributeDecl("el", "a\x01", enum.AttrCDATA, enum.AttrDefaultNone, "", nil)
				require.NoError(t, err)
				root, err := d.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				return d
			},
			"entity-reference-name": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				root, err := d.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				ref, err := d.CreateCharRef("e\x01")
				require.NoError(t, err)
				require.NoError(t, root.AddChild(ref))
				return d
			},
		}

		for _, version := range []string{ver10, ver11} {
			for name, mk := range build {
				t.Run(version+"/"+name, func(t *testing.T) {
					t.Parallel()
					// Default (reject) mode fails with ErrWriterInvalidName.
					var buf bytes.Buffer
					require.ErrorIs(t, helium.NewWriter().WriteTo(&buf, mk(t, version)), helium.ErrWriterInvalidName)
					// RejectInvalidChars(false) does NOT rescue a name (it has no U+FFFD
					// form): the same rejection stands.
					buf.Reset()
					require.ErrorIs(t, helium.NewWriter().RejectInvalidChars(false).WriteTo(&buf, mk(t, version)), helium.ErrWriterInvalidName)
				})
			}
		}
	})

	// the numeric character-reference
	// TARGET sites — an EntityRefNode "#N" (CreateCharRef) in element content and a
	// &#N; reference inside an entity value. Unlike a verbatim name, a character
	// reference carries its character as a target that must be serializable in the
	// target XML version, so the decision is VERSION-SENSITIVE: U+0001 is out of range
	// for XML 1.0 (rejected/replaced) but a RestrictedChar valid for XML 1.1 output
	// (emitted as &#1;). This is the &#1;-in-1.1 legality case.
	t.Run("char-ref targets", func(t *testing.T) {
		build := map[string]func(t *testing.T, v string) *helium.Document{
			"entity-ref-charref": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				root, err := d.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				ref, err := d.CreateCharRef("#1")
				require.NoError(t, err)
				require.NoError(t, root.AddChild(ref))
				return d
			},
			"entity-value-charref": func(t *testing.T, v string) *helium.Document {
				d := helium.NewDocument(v, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := d.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "&#1;")
				require.NoError(t, err)
				root, err := d.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, d.AddChild(root))
				return d
			},
		}

		for name, mk := range build {
			t.Run("xml10/"+name, func(t *testing.T) {
				t.Parallel()
				// XML 1.0: &#1; targets an out-of-range character.
				// Default mode rejects with ErrInvalidXMLChar (SERE0006).
				var buf bytes.Buffer
				require.ErrorIs(t, helium.NewWriter().WriteTo(&buf, mk(t, "1.0")), helium.ErrInvalidXMLChar)

				// Replacement mode substitutes the U+FFFD representation and NEVER emits a
				// bogus &#1;. With the default EscapeNonASCII it is the &#xFFFD; reference.
				buf.Reset()
				require.NoError(t, helium.NewWriter().RejectInvalidChars(false).WriteTo(&buf, mk(t, "1.0")))
				require.Contains(t, buf.String(), "&#xFFFD;")
				require.NotContains(t, buf.String(), "&#1;")

				// With EscapeNonASCII(false) the replacement is the raw U+FFFD character.
				buf.Reset()
				require.NoError(t, helium.NewWriter().EscapeNonASCII(false).RejectInvalidChars(false).WriteTo(&buf, mk(t, "1.0")))
				require.Contains(t, buf.String(), "�")
				require.NotContains(t, buf.String(), "&#1;")
			})

			t.Run("xml11/"+name, func(t *testing.T) {
				t.Parallel()
				// XML 1.1: &#1; targets a RestrictedChar that is legal as a character
				// reference, so it is emitted verbatim in BOTH modes — never rejected,
				// never replaced.
				var buf bytes.Buffer
				require.NoError(t, helium.NewWriter().WriteTo(&buf, mk(t, "1.1")))
				require.Contains(t, buf.String(), "&#1;")
				require.NotContains(t, buf.String(), "�")

				buf.Reset()
				require.NoError(t, helium.NewWriter().RejectInvalidChars(false).WriteTo(&buf, mk(t, "1.1")))
				require.Contains(t, buf.String(), "&#1;")
				require.NotContains(t, buf.String(), "�")
			})
		}
	})

	// keeps lexical failures separate from
	// target-range handling. EntityRefNode stores a body without delimiters, while
	// EntityValue carries complete reference markup in both content and orig paths.
	t.Run("malformed char-ref markup", func(t *testing.T) {
		newEntityDoc := func(t *testing.T, version, value string, orig bool) *helium.Document {
			t.Helper()
			doc := helium.NewDocument(version, "UTF-8", helium.StandaloneImplicitNo)
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

		for _, version := range []string{ver10, ver11} {
			t.Run(version+"/entity-ref-uppercase-hex-introducer", func(t *testing.T) {
				doc := helium.NewDocument(version, "UTF-8", helium.StandaloneImplicitNo)
				root, err := doc.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, doc.AddChild(root))
				ref, err := doc.CreateCharRef("#X41")
				require.NoError(t, err)
				require.NoError(t, root.AddChild(ref))

				var buf bytes.Buffer
				require.ErrorIs(t, helium.NewWriter().WriteTo(&buf, doc), helium.ErrWriterInvalidName)
				require.NotContains(t, buf.String(), "&#X41;")
			})

			for _, tc := range []struct {
				name  string
				value string
				orig  bool
			}{
				{name: "content-uppercase-hex-introducer", value: "&#X41;"},
				{name: "content-missing-digits", value: "&#x;"},
				{name: "content-bad-digit", value: "&#12z;"},
				{name: "content-missing-semicolon", value: "&#12"},
				{name: "content-overflow", value: "&#x110000;"},
				{name: "orig-uppercase-hex-introducer", value: "&#X41;", orig: true},
				{name: "orig-missing-digits", value: "&#x;", orig: true},
				{name: "orig-bad-digit", value: "&#12z;", orig: true},
				{name: "orig-missing-semicolon", value: "&#12", orig: true},
				{name: "orig-overflow", value: "&#x110000;", orig: true},
			} {
				t.Run(version+"/"+tc.name, func(t *testing.T) {
					var buf bytes.Buffer
					err := helium.NewWriter().RejectInvalidChars(false).WriteTo(&buf, newEntityDoc(t, version, tc.value, tc.orig))
					require.ErrorIs(t, err, helium.ErrWriterInvalidName)
					require.NotContains(t, buf.String(), tc.value)
				})
			}
		}
	})

	// both numeric-reference
	// emitters reject the complete surrogate block in XML 1.0 and XML 1.1. Replacement
	// remains available only because the numeric syntax itself is valid.
	t.Run("surrogate char-ref targets", func(t *testing.T) {
		build := map[string]func(t *testing.T, version string) *helium.Document{
			"entity-ref": func(t *testing.T, version string) *helium.Document {
				doc := helium.NewDocument(version, "UTF-8", helium.StandaloneImplicitNo)
				root, err := doc.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, doc.AddChild(root))
				ref, err := doc.CreateCharRef("#xD800")
				require.NoError(t, err)
				require.NoError(t, root.AddChild(ref))
				return doc
			},
			"entity-value-content": func(t *testing.T, version string) *helium.Document {
				doc := helium.NewDocument(version, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := doc.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "&#xD800;")
				require.NoError(t, err)
				root, err := doc.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, doc.AddChild(root))
				return doc
			},
			"entity-value-orig": func(t *testing.T, version string) *helium.Document {
				doc := helium.NewDocument(version, "UTF-8", helium.StandaloneImplicitNo)
				dtd, err := doc.CreateInternalSubset("doc", "", "")
				require.NoError(t, err)
				ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "plain")
				require.NoError(t, err)
				ent.SetOrig("&#xD800;")
				root, err := doc.CreateElement("doc")
				require.NoError(t, err)
				require.NoError(t, doc.AddChild(root))
				return doc
			},
		}

		for _, version := range []string{ver10, ver11} {
			for name, makeDoc := range build {
				t.Run(version+"/"+name, func(t *testing.T) {
					var buf bytes.Buffer
					require.ErrorIs(t, helium.NewWriter().WriteTo(&buf, makeDoc(t, version)), helium.ErrInvalidXMLChar)

					buf.Reset()
					require.NoError(t, helium.NewWriter().RejectInvalidChars(false).WriteTo(&buf, makeDoc(t, version)))
					require.NotContains(t, buf.String(), "&#xD800;")
					require.Contains(t, buf.String(), "&#xFFFD;")
				})
			}
		}
	})

	t.Run("XML 1.1 restricted characters", func(t *testing.T) {
		// U+0001 is a restricted-but-VALID character in XML 1.1, serialized as a
		// decimal character reference \u2014 never rejected and never replaced with
		// U+FFFD \u2014 regardless of the RejectInvalidChars setting.
		textDoc := func() *helium.Document {
			d := helium.NewDocument("1.1", "UTF-8", helium.StandaloneImplicitNo)
			r, err := d.CreateElement("r")
			require.NoError(t, err)
			require.NoError(t, d.AddChild(r))
			require.NoError(t, r.AppendText([]byte("a\x01b")))
			return d
		}
		attrDoc := func() *helium.Document {
			d := helium.NewDocument("1.1", "UTF-8", helium.StandaloneImplicitNo)
			r, err := d.CreateElement("r")
			require.NoError(t, err)
			require.NoError(t, d.AddChild(r))
			err = r.SetAttribute("v", "x\x01y")
			require.NoError(t, err)
			return d
		}

		// Default (rejection) mode: the restricted char serializes as &#1;.
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, textDoc()))
		require.Contains(t, buf.String(), "&#1;")
		require.NotContains(t, buf.String(), "\uFFFD")
		buf.Reset()
		require.NoError(t, helium.NewWriter().WriteTo(&buf, attrDoc()))
		require.Contains(t, buf.String(), "&#1;")

		// Replacement mode: same decimal character reference, unaffected.
		buf.Reset()
		require.NoError(t, helium.NewWriter().RejectInvalidChars(false).WriteTo(&buf, textDoc()))
		require.Contains(t, buf.String(), "&#1;")
		require.NotContains(t, buf.String(), "\uFFFD")
		buf.Reset()
		require.NoError(t, helium.NewWriter().EscapeNonASCII(false).RejectInvalidChars(false).WriteTo(&buf, attrDoc()))
		require.Contains(t, buf.String(), "&#1;")
	})
}

func TestWriterCharMap(t *testing.T) {
	t.Parallel()

	// a character-map
	// replacement is emitted verbatim (never re-escaped, Unicode-normalized, or
	// rejected by the invalid-char policy) per XSLT/XQuery Serialization 3.1 §7, and
	// that this holds identically whether or not Normalization is active. The
	// surrounding content is still normalized; only the replacement span is inert.
	t.Run("normalization is verbatim", func(t *testing.T) {
		const decomposed = "é" // "e" + combining acute
		const composed = "é"    // U+00E9

		serialize := func(t *testing.T, w helium.Writer, src string) (string, error) {
			t.Helper()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)
			var buf strings.Builder
			err = w.WriteTo(&buf, doc)
			return buf.String(), err
		}

		// '@' maps to a replacement carrying XML markup characters; those must survive
		// verbatim (not become &lt;/&amp;) in both text and attribute value, and the
		// decomposed é around them must still compose under NFC.
		t.Run("markup chars in replacement", func(t *testing.T) {
			t.Parallel()
			m := map[rune]string{'@': "<b>&"}
			src := `<a x="` + decomposed + `@` + decomposed + `">` + decomposed + `@` + decomposed + `</a>`
			for _, form := range []string{"", "NFC"} {
				// The surrounding é composes under NFC; without normalization it stays
				// decomposed. The '@' replacement is verbatim in both.
				surround := decomposed
				if form == "NFC" {
					surround = composed
				}
				w := helium.NewWriter().XMLDeclaration(false).EscapeNonASCII(false).
					CharacterMap(m).Normalization(form)
				out, err := serialize(t, w, src)
				require.NoError(t, err, "form=%q", form)
				require.Contains(t, out, ">"+surround+"<b>&"+surround+"</a>",
					"text replacement verbatim, surround normalized (form=%q): %q", form, out)
				require.Contains(t, out, `x="`+surround+`<b>&`+surround+`"`,
					"attr replacement verbatim, surround normalized (form=%q): %q", form, out)
			}
		})

		// A replacement carrying a control character (an invalid XML char) is emitted
		// verbatim; RejectInvalidChars must not reject it, and Normalization must not
		// change that.
		t.Run("control char in replacement with RejectInvalidChars", func(t *testing.T) {
			t.Parallel()
			m := map[rune]string{'@': "x\x01y"}
			for _, form := range []string{"", "NFC"} {
				w := helium.NewWriter().XMLDeclaration(false).CharacterMap(m).
					RejectInvalidChars(true).Normalization(form)
				out, err := serialize(t, w, `<a x="@">@</a>`)
				require.NoError(t, err, "control-char replacement must not be rejected (form=%q)", form)
				require.Contains(t, out, ">x\x01y</a>", "text replacement verbatim (form=%q): %q", form, out)
				require.Contains(t, out, `x="x`+"\x01"+`y"`, "attr replacement verbatim (form=%q): %q", form, out)
			}
		})
	})

	// character-map matching
	// is decided on the PRE-normalization content (Serialization 3.1 §4: character
	// mapping — rule c — precedes Unicode normalization — rule d — and is never
	// re-applied): a mapped rune CREATED by NFC composition is emitted as that rune,
	// not as the replacement, so mapped-rune treatment is identical with and without
	// Normalization; a mapped rune PRESENT in the input still maps in both.
	t.Run("matching a created rune", func(t *testing.T) {
		const decomposed = "é" // "e" + combining acute
		const composed = "é"    // U+00E9
		m := map[rune]string{'é': "<mapped>"}

		serialize := func(t *testing.T, form, src string) string {
			t.Helper()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)
			var buf strings.Builder
			err = helium.NewWriter().XMLDeclaration(false).EscapeNonASCII(false).
				CharacterMap(m).Normalization(form).WriteTo(&buf, doc)
			require.NoError(t, err)
			return buf.String()
		}

		t.Run("NFC-composed rune is not newly matched", func(t *testing.T) {
			t.Parallel()
			src := `<a x="` + decomposed + `">` + decomposed + `</a>`
			// Without Normalization the decomposed pair matches nothing.
			out := serialize(t, "", src)
			require.Contains(t, out, ">"+decomposed+"</a>", "text unmapped without normalization: %q", out)
			require.Contains(t, out, `x="`+decomposed+`"`, "attr unmapped without normalization: %q", out)
			require.NotContains(t, out, "<mapped>", "no replacement without normalization: %q", out)
			// With NFC the pair composes to é — a rune CREATED by normalization — so
			// it is emitted as é, not substituted with the replacement.
			out = serialize(t, "NFC", src)
			require.Contains(t, out, ">"+composed+"</a>", "text composed, not mapped: %q", out)
			require.Contains(t, out, `x="`+composed+`"`, "attr composed, not mapped: %q", out)
			require.NotContains(t, out, "<mapped>", "no replacement for a normalization-created rune: %q", out)
		})

		t.Run("input-present mapped rune still maps under NFC", func(t *testing.T) {
			t.Parallel()
			// Text mixes an input-present é (must map in both configurations) with a
			// decomposed pair (must map in neither).
			src := `<a x="` + composed + `">` + composed + decomposed + `</a>`
			out := serialize(t, "", src)
			require.Contains(t, out, `x="<mapped>"`, "attr mapped without normalization: %q", out)
			require.Contains(t, out, "><mapped>"+decomposed+"</a>", "text: input é mapped, pair untouched: %q", out)
			out = serialize(t, "NFC", src)
			require.Contains(t, out, `x="<mapped>"`, "attr mapped under NFC: %q", out)
			require.Contains(t, out, "><mapped>"+composed+"</a>", "text: input é mapped, pair composed unmapped: %q", out)
		})
	})

	// that
	// pre-normalization character-map matching does not depend on finding a rune
	// absent from the content: with content containing EVERY Unicode private-use
	// rune, a mapped rune present in the input is replaced exactly once, and a
	// mapped rune CREATED by NFKC composition (fullwidth Ａ → A) still stays
	// literal.
	t.Run("private-use saturation", func(t *testing.T) {
		// All 137,468 private-use runes: the BMP Private Use Area plus both
		// supplementary private-use planes.
		var pu strings.Builder
		for _, rng := range [][2]rune{{0xE000, 0xF8FF}, {0xF0000, 0xFFFFD}, {0x100000, 0x10FFFD}} {
			for r := rng[0]; r <= rng[1]; r++ {
				pu.WriteRune(r)
			}
		}
		// Content: every private-use rune, then fullwidth Ａ (U+FF21, which NFKC
		// maps to ASCII A — must stay literal), then ASCII A (mapped — exactly one
		// replacement).
		src := "<a>" + pu.String() + "ＡA</a>"
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		var buf strings.Builder
		err = helium.NewWriter().XMLDeclaration(false).EscapeNonASCII(false).
			CharacterMap(map[rune]string{'A': "<mapped>"}).Normalization("NFKC").
			WriteTo(&buf, doc)
		require.NoError(t, err)
		out := buf.String()
		require.Equal(t, 1, strings.Count(out, "<mapped>"),
			"exactly one replacement: the input A maps, the NFKC-created A does not")
		require.Contains(t, out, "A<mapped></a>",
			"NFKC-composed A stays literal ahead of the replacement: %q", out[len(out)-40:])
	})

	// the value-style contract for every
	// map-taking Writer setter: the setter copies its input, so mutating the
	// caller's map AFTER the setter call does not change the configured Writer.
	// Without cloning the setters would retain the caller's map by reference, so a
	// post-set mutation (or a concurrent write) would silently change serialization.
	t.Run("map setters clone their input", func(t *testing.T) {
		serialize := func(t *testing.T, w helium.Writer, n helium.Node) string {
			t.Helper()
			var buf bytes.Buffer
			require.NoError(t, w.WriteTo(&buf, n))
			return buf.String()
		}

		parse := func(t *testing.T, src string) *helium.Document {
			t.Helper()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)
			return doc
		}

		t.Run("CharacterMap", func(t *testing.T) {
			t.Parallel()
			doc := parse(t, `<a>o</a>`)
			m := map[rune]string{}
			w := helium.NewWriter().CharacterMap(m)
			m['o'] = "0" // mutate after set — must not reach the Writer
			out := serialize(t, w, doc)
			require.Contains(t, out, ">o</a>", "unmutated char preserved: %q", out)
			require.NotContains(t, out, ">0</a>", "post-set mutation must not apply: %q", out)
		})

		t.Run("CDATASectionElements", func(t *testing.T) {
			t.Parallel()
			doc := parse(t, `<a>hi</a>`)
			m := map[string]struct{}{}
			w := helium.NewWriter().CDATASectionElements(m)
			m["{}a"] = struct{}{} // mutate after set
			out := serialize(t, w, doc)
			require.NotContains(t, out, "CDATA", "post-set mutation must not apply: %q", out)
		})

		t.Run("SuppressIndentElements", func(t *testing.T) {
			t.Parallel()
			doc := parse(t, `<a><b>x</b></a>`)
			m := map[string]struct{}{}
			w := helium.NewWriter().XMLDeclaration(false).Format(true).SuppressIndentElements(m)
			m["{}a"] = struct{}{} // mutate after set
			out := serialize(t, w, doc)
			// With no suppression the formatter indents <b> onto its own line.
			require.Contains(t, out, "\n", "output stays formatted: %q", out)
			require.NotContains(t, out, "<a><b>", "post-set suppression must not apply: %q", out)
		})

		t.Run("InheritedNamespaces", func(t *testing.T) {
			t.Parallel()
			firstElem := func(n helium.Node) helium.Node {
				for c := n.FirstChild(); c != nil; c = c.NextSibling() {
					if c.Type() == helium.ElementNode {
						return c
					}
				}
				return n
			}
			doc := parse(t, `<root xmlns:p="urn:p"><p:child>text</p:child></root>`)
			m := map[string]string{}
			w := helium.NewWriter().XMLDeclaration(false).InheritedNamespaces(m)
			m["p"] = "urn:p" // mutate after set — must not suppress the redeclaration
			out := serialize(t, w, firstElem(doc.DocumentElement()))
			require.Contains(t, out, `xmlns:p="urn:p"`, "post-set inherited ns must not apply: %q", out)
		})
	})
}

func TestWriterNormalization(t *testing.T) {
	t.Parallel()

	// Writer.Normalization: Unicode normalization is
	// scoped to text-node and attribute-value character content (Serialization 3.1
	// §4). Element/attribute names, comments, and PIs are never normalized.
	t.Run("forms", func(t *testing.T) {
		const decomposed = "e\u0301" // "e" + combining acute
		const composed = "\u00e9"    // U+00E9
		src := "<caf" + decomposed + " at" + decomposed + "=\"" + decomposed + "\">" +
			"<!--" + decomposed + "--><?p " + decomposed + "?>" + decomposed +
			"</caf" + decomposed + ">"
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		var buf strings.Builder
		// EscapeNonASCII(false) so the composed é appears literally, isolating the
		// normalization effect from the writer's numeric-reference escaping.
		err = helium.NewWriter().XMLDeclaration(false).EscapeNonASCII(false).
			Normalization("NFC").WriteTo(&buf, doc)
		require.NoError(t, err)
		out := buf.String()

		// Text and attribute value are composed; names, comment, and PI stay decomposed.
		require.Contains(t, out, ">"+composed+"</caf"+decomposed+">", "text normalized: %q", out)
		require.Contains(t, out, "at"+decomposed+"=\""+composed+"\"", "attr value normalized, name not: %q", out)
		require.Contains(t, out, "<caf"+decomposed, "element name not normalized: %q", out)
		require.Contains(t, out, "<!--"+decomposed+"-->", "comment not normalized: %q", out)
		require.Contains(t, out, "<?p "+decomposed+"?>", "PI not normalized: %q", out)
		require.NotContains(t, out, "caf"+composed, "name must stay decomposed: %q", out)

		// Without normalization the output is byte-identical to the source content.
		var raw strings.Builder
		err = helium.NewWriter().XMLDeclaration(false).EscapeNonASCII(false).WriteTo(&raw, doc)
		require.NoError(t, err)
		require.NotContains(t, raw.String(), composed, "no normalization by default: %q", raw.String())
	})

	// an unrecognized
	// normalization-form value is observable (ErrUnsupportedNormalizationForm) rather
	// than silently disabling normalization, while the supported forms and the
	// disabling values ("", "none") are accepted.
	t.Run("invalid form rejected", func(t *testing.T) {
		newDoc := func(t *testing.T) *helium.Document {
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.AppendText([]byte("x")))
			return doc
		}

		for _, form := range []string{"NFCC", "nfc", "fully-normalized", "NONE", "garbage"} {
			t.Run("rejects "+form, func(t *testing.T) {
				t.Parallel()
				var buf strings.Builder
				err := helium.NewWriter().Normalization(form).WriteTo(&buf, newDoc(t))
				require.Error(t, err, "unrecognized normalization form must be rejected")
				require.ErrorIs(t, err, helium.ErrUnsupportedNormalizationForm)
				require.Empty(t, buf.String(), "no output byte before the rejection")
			})
		}

		for _, form := range []string{"", "none", "NFC", "NFD", "NFKC", "NFKD"} {
			t.Run("accepts "+form, func(t *testing.T) {
				t.Parallel()
				var buf strings.Builder
				err := helium.NewWriter().Normalization(form).WriteTo(&buf, newDoc(t))
				require.NoError(t, err, "supported normalization form must serialize")
			})
		}

		// A bare element (non-Document) path is also validated.
		t.Run("bare element path", func(t *testing.T) {
			t.Parallel()
			doc := newDoc(t)
			var buf strings.Builder
			err := helium.NewWriter().Normalization("bogus").WriteTo(&buf, doc.DocumentElement())
			require.ErrorIs(t, err, helium.ErrUnsupportedNormalizationForm)
		})
	})
}

func TestWriterRejectsInjection(t *testing.T) {
	t.Parallel()

	t.Run("injected names", func(t *testing.T) {
		t.Run("element name injection", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement(`root injected="1"`)
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))

			_, err = helium.WriteString(doc)
			require.Error(t, err, "injected element name must not serialize")
		})

		t.Run("attribute name injection", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			// SetAttribute only rejects colons, so a space-bearing name slips
			// through and would inject a second attribute on serialization.
			err = root.SetAttribute(`x onmouseover`, "1")
			require.NoError(t, err)

			_, err = helium.WriteString(doc)
			require.Error(t, err, "injected attribute name must not serialize")
		})

		t.Run("reserved xmlns attribute name rejected", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			// "xmlns" is a valid NCName, but a normal attribute named "xmlns"
			// would be emitted as a namespace declaration that never went through
			// DeclareNamespace.
			err = root.SetAttribute("xmlns", "urn:evil")
			require.NoError(t, err)

			_, err = helium.WriteString(doc)
			require.Error(t, err, "reserved xmlns attribute name must not serialize")
		})

		t.Run("reserved xmlns-prefixed attribute name rejected", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			// An attribute whose QName prefix is "xmlns" (e.g. "xmlns:foo") is a
			// namespace declaration and must not be emitted as a normal attribute.
			ns, err := doc.CreateNamespace("xmlns", "urn:x")
			require.NoError(t, err)
			err = root.SetAttributeNS("foo", "v", ns)
			require.NoError(t, err)

			_, err = helium.WriteString(doc)
			require.Error(t, err, "reserved xmlns-prefixed attribute name must not serialize")
		})

		t.Run("valid element name serializes", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Contains(t, str, "<root/>")
		})

		t.Run("valid namespaced name serializes", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.SetActiveNamespace("p", "urn:example"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Contains(t, str, "<p:root")
		})
	})

	t.Run("injected namespace prefix", func(t *testing.T) {
		t.Run("namespace prefix injection", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			// DeclareNamespace does not validate the prefix, so a crafted prefix
			// would inject raw markup into the start tag on serialization.
			require.NoError(t, root.DeclareNamespace(`p injected="1`, "urn"))

			_, err = helium.WriteString(doc)
			require.Error(t, err, "injected namespace prefix must not serialize")
		})

		t.Run("valid prefix serializes", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.DeclareNamespace("p", "urn:example"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Contains(t, str, `xmlns:p="urn:example"`)
		})

		t.Run("default namespace serializes", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.DeclareNamespace("", "urn:default"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Contains(t, str, `xmlns="urn:default"`)
		})

		t.Run("reserved xml prefix serializes", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.DeclareNamespace("xml", lexicon.NamespaceXML))

			_, err = helium.WriteString(doc)
			require.NoError(t, err, "reserved xml prefix must still serialize")
		})

		t.Run("reserved xmlns prefix rejected", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			// Namespaces-in-XML forbids declaring the xmlns prefix; the serializer
			// must not emit xmlns:xmlns="...".
			require.NoError(t, root.DeclareNamespace("xmlns", "urn"))

			_, err = helium.WriteString(doc)
			require.Error(t, err, "reserved xmlns prefix must not serialize")
		})
	})

	t.Run("malformed comment or PI", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)

		var sb strings.Builder
		require.Error(t, helium.Write(&sb, doc.CreateComment([]byte("a--b"))),
			"comment containing -- must be rejected")
		sb.Reset()
		require.Error(t, helium.Write(&sb, doc.CreateComment([]byte("a-"))),
			"comment ending in - must be rejected")
		sb.Reset()
		require.Error(t, helium.Write(&sb, doc.CreateComment([]byte("-"))),
			"single-dash comment must be rejected")
		sb.Reset()
		require.Error(t, helium.Write(&sb, doc.CreatePI("t", "a?>b")),
			"PI content containing ?> must be rejected")

		// Valid comment/PI still serialize.
		sb.Reset()
		require.NoError(t, helium.Write(&sb, doc.CreateComment([]byte(" ok "))))
		sb.Reset()
		require.NoError(t, helium.Write(&sb, doc.CreateComment([]byte(""))),
			"empty comment must serialize without an out-of-range panic")
		sb.Reset()
		require.NoError(t, helium.Write(&sb, doc.CreatePI("php", "echo 1")))
	})

	// ensures that an invalid PI target — in
	// particular one that injects markup — is rejected before being emitted, so
	// the serialized output never contains the injection.
	t.Run("malformed PI target", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			target string
		}{
			{name: "injection", target: "x?><evil/><?x"},
			{name: "empty", target: ""},
			{name: "starts-digit", target: "1bad"},
			{name: "has-space", target: "a b"},
			{name: "reserved-xml", target: "xml"},
			{name: "invalid-utf8", target: "\xff\xfe"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				root, err := doc.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, doc.SetDocumentElement(root))
				require.NoError(t, root.AddChild(doc.CreatePI(tc.target, "")))

				var sb strings.Builder
				err = helium.Write(&sb, doc)
				require.Error(t, err, "invalid PI target must be rejected")
				require.NotContains(t, sb.String(), "<evil/>",
					"injection must not be emitted")
			})
		}

		// A valid target still serializes.
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("r")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.AddChild(doc.CreatePI("php", "echo 1")))
		var sb strings.Builder
		require.NoError(t, helium.Write(&sb, doc))
		require.Contains(t, sb.String(), "<?php echo 1?>")
	})

	// Finding 2: an element whose QName
	// prefix is the reserved "xmlns" prefix must not serialize, even when an active
	// namespace bypasses dumpNs.
	t.Run("xmlns element name", func(t *testing.T) {
		t.Run("xmlns-prefixed element name rejected", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			// SetActiveNamespace sets the node's active namespace directly, so the
			// "xmlns" prefix is emitted as the element QName prefix (<xmlns:root/>),
			// which Namespaces-in-XML forbids.
			require.NoError(t, root.SetActiveNamespace("xmlns", "urn:evil"))

			_, err = helium.WriteString(doc)
			require.Error(t, err, "xmlns-prefixed element name must not serialize")
		})

		t.Run("valid namespaced element name serializes", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.SetActiveNamespace("p", "urn:example"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Contains(t, str, "<p:root")
		})

		t.Run("bare xmlns element name serializes", func(t *testing.T) {
			t.Parallel()
			// "xmlns" is a valid element name: <xmlns>...</xmlns> is well-formed XML.
			// It is reserved only as an attribute name (default-namespace decl), so
			// an element literally named "xmlns" must serialize without error. This
			// is the regression case from xslt3 test si-element-261.
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("xmlns")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))

			str, err := helium.WriteString(doc)
			require.NoError(t, err, "bare xmlns element name must serialize")
			require.Contains(t, str, "<xmlns")
		})
	})

	// checkNamespaceBinding treats
	// the reserved "xml" prefix as bound even when the namespace object carries an
	// empty href: Namespaces in XML binds "xml" by definition to the XML namespace,
	// so "xml:local" reparses without any in-scope declaration (the parser resolves
	// it in lookupNamespace). The empty-href case arises from
	// CreateNamespace("xml", "") and from html.Parse building colon names. A
	// non-xml prefix with an empty href must still be rejected.
	t.Run("implicit xml prefix accepted", func(t *testing.T) {
		t.Run("empty-URI xml attribute serializes and reparses", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			ns, err := doc.CreateNamespace("xml", "")
			require.NoError(t, err)
			require.NoError(t, root.SetAttributeNS("lang", "en", ns))

			str, err := helium.WriteString(doc)
			require.NoError(t, err, "xml:lang with an empty-href xml namespace must serialize")
			require.Contains(t, str, `xml:lang="en"`)

			reparsed, err := helium.NewParser().Parse(t.Context(), []byte(str))
			require.NoError(t, err, "writer output must reparse")
			require.NotNil(t, reparsed)
		})

		t.Run("xml-prefixed element serializes and reparses", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			ns, err := doc.CreateNamespace("xml", "")
			require.NoError(t, err)
			root, err := doc.CreateElementNS("foo", ns)
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))

			str, err := helium.WriteString(doc)
			require.NoError(t, err, "an xml:-prefixed element must serialize")
			require.Contains(t, str, `<xml:foo`)

			reparsed, err := helium.NewParser().Parse(t.Context(), []byte(str))
			require.NoError(t, err, "writer output must reparse")
			require.NotNil(t, reparsed)
		})

		t.Run("XHTML path accepts empty-URI xml attribute", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			_, err := doc.CreateInternalSubset(
				"html",
				"-//W3C//DTD XHTML 1.0 Strict//EN",
				"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd",
			)
			require.NoError(t, err)
			root, err := doc.CreateElement("html")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			ns, err := doc.CreateNamespace("xml", "")
			require.NoError(t, err)
			require.NoError(t, root.SetAttributeNS("lang", "en", ns))

			str, err := helium.WriteString(doc)
			require.NoError(t, err, "XHTML serializer must accept the implicit xml prefix")
			require.Contains(t, str, `xml:lang="en"`)
		})

		t.Run("non-xml prefix with empty URI still rejected", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			ns, err := doc.CreateNamespace("foo", "")
			require.NoError(t, err)
			require.NoError(t, root.SetAttributeNS("lang", "en", ns))

			_, err = helium.WriteString(doc)
			require.Error(t, err, "a non-xml prefix bound to an empty URI must not serialize")
			require.ErrorIs(t, err, helium.ErrWriterUnboundNamespacePrefix)
		})
	})
}
