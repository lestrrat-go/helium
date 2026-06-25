package stream_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium/stream"
	"github.com/stretchr/testify/require"
)

// TestWriteDTD exercises DTD writing branches.
func TestWriteDTD(t *testing.T) {
	t.Parallel()

	// dtdQuoteFor: a value containing the preferred quote should be wrapped in
	// the other quote. Default quoteChar is '"', so a sysid containing '"' must
	// be emitted with single quotes. Also exercise QuoteChar('\”) path.
	t.Run("dtd quote for alternate quote", func(t *testing.T) {
		t.Parallel()

		t.Run("double-quote default, sysid contains double quote", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", `a"b`))
			require.NoError(t, w.EndDTD())
			require.NoError(t, w.StartElement("doc"))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			require.Contains(t, buf.String(), `SYSTEM 'a"b'`)
		})

		t.Run("single-quote configured, sysid contains single quote", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf).QuoteChar('\'')
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", `a'b`))
			require.NoError(t, w.EndDTD())
			require.NoError(t, w.StartElement("doc"))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			require.Contains(t, buf.String(), `SYSTEM "a'b"`)
		})
	})

	// ensureDTDState: sticky-error path and outside-DTD path.
	t.Run("dtd outside state", func(t *testing.T) {
		t.Parallel()

		t.Run("WriteDTDElement outside DTD", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.Error(t, w.WriteDTDElement("e", "EMPTY"))
		})

		t.Run("WriteDTDAttlist outside DTD", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.Error(t, w.WriteDTDAttlist("e", "a CDATA #IMPLIED"))
		})

		t.Run("WriteDTDEntity outside DTD", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.Error(t, w.WriteDTDEntity(false, "e", "val"))
		})

		t.Run("WriteDTDExternalEntity outside DTD", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.Error(t, w.WriteDTDExternalEntity(false, "e", "", "sys", ""))
		})

		t.Run("WriteDTDNotation outside DTD", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.Error(t, w.WriteDTDNotation("n", "", "sys"))
		})
	})

	// Indented DTD output paths in StartDTD (PUBLIC and SYSTEM with indentation).
	t.Run("start dtd indented", func(t *testing.T) {
		t.Parallel()

		t.Run("indented PUBLIC", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf).Indent("  ")
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "-//X//DTD//EN", "http://example.com/x.dtd"))
			require.NoError(t, w.EndDTD())
			require.NoError(t, w.StartElement("doc"))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			out := buf.String()
			require.Contains(t, out, "PUBLIC")
			require.Contains(t, out, "-//X//DTD//EN")
		})

		t.Run("indented SYSTEM", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf).Indent("  ")
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", "http://example.com/x.dtd"))
			require.NoError(t, w.EndDTD())
			require.NoError(t, w.StartElement("doc"))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			require.Contains(t, buf.String(), "SYSTEM")
		})
	})

	// WriteDTDExternalEntity with PUBLIC id and NDATA notation exercises those
	// branches.
	t.Run("external entity variants", func(t *testing.T) {
		t.Parallel()

		t.Run("PUBLIC with NDATA", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.NoError(t, w.WriteDTDExternalEntity(false, "img", "-//X//PIC//EN", "pic.gif", "gif"))
			require.NoError(t, w.EndDTD())
			require.NoError(t, w.StartElement("doc"))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			out := buf.String()
			require.Contains(t, out, "PUBLIC")
			require.Contains(t, out, "NDATA gif")
		})

		t.Run("parameter entity SYSTEM", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.NoError(t, w.WriteDTDExternalEntity(true, "pe", "", "pe.dtd", ""))
			require.NoError(t, w.EndDTD())
			require.NoError(t, w.StartElement("doc"))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			out := buf.String()
			require.Contains(t, out, "<!ENTITY % pe")
			require.Contains(t, out, "SYSTEM")
		})

		t.Run("invalid pubid rejected", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.Error(t, w.WriteDTDExternalEntity(false, "e", "bad\x01pub", "sys", ""))
		})

		t.Run("invalid sysid rejected", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.Error(t, w.WriteDTDExternalEntity(false, "e", "", "a'b\"c", ""))
		})

		t.Run("invalid ndata rejected", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.Error(t, w.WriteDTDExternalEntity(false, "e", "", "sys", "1bad"))
		})
	})

	// WriteDTDNotation PUBLIC with both pubid and sysid.
	t.Run("notation variants", func(t *testing.T) {
		t.Parallel()

		t.Run("PUBLIC with sysid", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.NoError(t, w.WriteDTDNotation("n", "-//X//N//EN", "n.dtd"))
			require.NoError(t, w.EndDTD())
			require.NoError(t, w.StartElement("doc"))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			out := buf.String()
			require.Contains(t, out, "<!NOTATION n PUBLIC")
			require.Contains(t, out, "n.dtd")
		})

		t.Run("invalid name rejected", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.Error(t, w.WriteDTDNotation("1bad", "", "sys"))
		})

		t.Run("invalid pubid rejected", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.Error(t, w.WriteDTDNotation("n", "bad\x01", "sys"))
		})
	})

	// WriteDTD with a verbatim subset exercises the subset != "" branch.
	t.Run("with subset", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.WriteDTD("doc", "", "", "<!ELEMENT doc EMPTY>"))
		require.NoError(t, w.StartElement("doc"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndDocument())
		out := buf.String()
		require.Contains(t, out, "[<!ELEMENT doc EMPTY>]")
	})

	// validateSystemID: a value with both quotes is unquotable.
	t.Run("invalid sysid both quotes", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.Error(t, w.StartDTD("doc", "", `a'b"c`))
	})

	// validatePubid rejection in StartDTD.
	t.Run("invalid pubid", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.Error(t, w.StartDTD("doc", "bad\x01pub", "sys"))
	})

	// validateDTDFragment rejects markup delimiters in element content.
	t.Run("element fragment rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("doc", "", ""))
		require.Error(t, w.WriteDTDElement("e", "ANY><!ENTITY x"))
	})

	// validateDTDAttlistFragment: unterminated literal and bare '>' outside quotes.
	t.Run("attlist fragment rejected", func(t *testing.T) {
		t.Parallel()

		t.Run("unterminated literal", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.Error(t, w.WriteDTDAttlist("e", `a CDATA "unterminated`))
		})

		t.Run("bare gt outside quote", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.Error(t, w.WriteDTDAttlist("e", `a CDATA #IMPLIED>extra`))
		})

		t.Run("gt inside quote allowed", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.NoError(t, w.WriteDTDAttlist("e", `a CDATA "a>b"`))
		})
	})

	// WriteDTDEntity parameter entity success.
	t.Run("entity parameter", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("doc", "", ""))
		require.NoError(t, w.WriteDTDEntity(true, "pe", "value"))
		require.NoError(t, w.EndDTD())
		require.NoError(t, w.StartElement("doc"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndDocument())
		require.Contains(t, buf.String(), "<!ENTITY % pe")
	})
}

// TestWriteCDATA exercises CDATA writing branches.
func TestWriteCDATA(t *testing.T) {
	t.Parallel()

	// WriteCDATA with a "]]>" terminator must split into multiple sections.
	t.Run("split", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.WriteCDATA("a]]>b"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndDocument())
		out := buf.String()
		require.Contains(t, out, "<![CDATA[a]]]]><![CDATA[>b]]>")
	})

	// WriteCDATA validation error: invalid char must be rejected before any output.
	t.Run("invalid char", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		before := buf.Len()
		require.Error(t, w.WriteCDATA("bad\x00char"))
		require.Equal(t, before, buf.Len())
	})
}

// TestWriteNS exercises namespace-related writing branches.
func TestWriteNS(t *testing.T) {
	t.Parallel()

	// hasDefaultNSInScope is exercised via StartElementNS: an ancestor declares
	// a non-empty default namespace, then a child with no namespace must emit
	// xmlns="" to undeclare it.
	t.Run("default ns undeclare", func(t *testing.T) {
		t.Parallel()

		t.Run("undeclare default ns on child", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElementNS("", "root", "urn:default"))
			require.NoError(t, w.StartElementNS("", "child", ""))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			out := buf.String()
			require.Contains(t, out, `xmlns="urn:default"`)
			require.Contains(t, out, `xmlns=""`)
		})

		t.Run("no undeclare when no default ns in scope", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			// ancestor declares only a prefixed ns, default ns absent
			require.NoError(t, w.StartElementNS("p", "root", "urn:p"))
			require.NoError(t, w.StartElementNS("", "child", ""))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			require.NotContains(t, buf.String(), `xmlns=""`)
		})
	})

	// WriteElementNS / WriteAttributeNS: success path plus content validation
	// rejection (leaving writer unmutated).
	t.Run("conveniences", func(t *testing.T) {
		t.Parallel()

		t.Run("WriteElementNS success", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.WriteElementNS("p", "el", "urn:p", "hello"))
			require.NoError(t, w.EndDocument())
			out := buf.String()
			require.Contains(t, out, `<p:el xmlns:p="urn:p">hello</p:el>`)
		})

		t.Run("WriteElementNS invalid content rejected", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.Error(t, w.WriteElementNS("p", "el", "urn:p", "bad\x00"))
			require.Empty(t, buf.String())
		})

		t.Run("WriteElementNS invalid name rejected", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.Error(t, w.WriteElementNS("p", "1bad", "urn:p", "x"))
		})

		t.Run("WriteAttributeNS success", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.NoError(t, w.WriteAttributeNS("p", "a", "urn:p", "v"))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			out := buf.String()
			require.Contains(t, out, `xmlns:p="urn:p"`)
			require.Contains(t, out, `p:a="v"`)
		})

		t.Run("WriteAttributeNS invalid content rejected", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.Error(t, w.WriteAttributeNS("p", "a", "urn:p", "bad\x00"))
		})

		t.Run("WriteAttributeNS invalid prefix rejected", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.Error(t, w.WriteAttributeNS("1bad", "a", "urn:p", "v"))
		})
	})

	// StartElementNS invalid namespace URI rejected before any mutation.
	t.Run("start element ns invalid uri", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.Error(t, w.StartElementNS("p", "el", "bad\x00uri"))
		require.Empty(t, buf.String())
	})

	// StartAttributeNS state and URI validation paths.
	t.Run("start attribute ns edges", func(t *testing.T) {
		t.Parallel()

		t.Run("invalid uri rejected", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.Error(t, w.StartAttributeNS("p", "a", "bad\x00"))
		})

		t.Run("called outside opening tag", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.NoError(t, w.WriteString("text"))
			require.Error(t, w.StartAttributeNS("p", "a", "urn:p"))
		})
	})
}

// TestWriteComment exercises comment writing branches.
func TestWriteComment(t *testing.T) {
	t.Parallel()

	// WriteComment validation errors: invalid char, contains --, ends with -.
	t.Run("validation", func(t *testing.T) {
		t.Parallel()

		t.Run("invalid char", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.Error(t, w.WriteComment("bad\x00"))
		})

		t.Run("contains double dash", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.Error(t, w.WriteComment("a--b"))
		})

		t.Run("ends with dash", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.Error(t, w.WriteComment("trailing-"))
		})
	})

	// EndComment with trailing dash in incremental writes rejected.
	t.Run("end comment trailing dash", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartComment())
		require.NoError(t, w.WriteString("text-"))
		require.Error(t, w.EndComment())
	})

	// WriteString in comment state where prior chunk ended with '-' and next
	// begins with '-' (forms '--').
	t.Run("write string comment dash split", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartComment())
		require.NoError(t, w.WriteString("a-"))
		require.Error(t, w.WriteString("-b"))
	})
}

// TestWritePI exercises processing-instruction writing branches.
func TestWritePI(t *testing.T) {
	t.Parallel()

	// WritePI content validation and the "?>" rejection.
	t.Run("validation", func(t *testing.T) {
		t.Parallel()

		t.Run("contains ?>", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.Error(t, w.WritePI("target", "data?>more"))
		})

		t.Run("invalid char", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.Error(t, w.WritePI("target", "bad\x00"))
		})
	})

	// WriteString in PI state with a question-mark suffix that would form "?>".
	t.Run("write string pi suffix", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartPI("t"))
		require.NoError(t, w.WriteString("data?"))
		// A following ">" would form "?>" and must be rejected.
		require.Error(t, w.WriteString(">more"))
	})
}

// TestEndElement exercises element-closing branches.
func TestEndElement(t *testing.T) {
	t.Parallel()

	// FullEndElement with children but no text triggers the writeEndIndent branch
	// when indentation is enabled.
	t.Run("full end element indented", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf).Indent("  ")
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartElement("child"))
		require.NoError(t, w.FullEndElement())
		require.NoError(t, w.FullEndElement())
		require.NoError(t, w.EndDocument())
		out := buf.String()
		require.Contains(t, out, "<child></child>")
	})

	// FullEndElement closing an element that still has an open attribute.
	t.Run("full end element with open attribute", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartAttribute("a"))
		require.NoError(t, w.WriteString("v"))
		require.NoError(t, w.FullEndElement())
		require.NoError(t, w.EndDocument())
		require.Contains(t, buf.String(), `<root a="v"></root>`)
	})

	// EndElement closing an element with an open attribute.
	t.Run("end element with open attribute", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartAttribute("a"))
		require.NoError(t, w.WriteString("v"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndDocument())
		require.Contains(t, buf.String(), `<root a="v"/>`)
	})

	// EndDocument auto-close branches for each open construct.
	t.Run("end document auto close", func(t *testing.T) {
		t.Parallel()

		t.Run("auto-close open PI", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.NoError(t, w.StartPI("php"))
			require.NoError(t, w.EndDocument())
			require.Contains(t, buf.String(), "?>")
		})

		t.Run("auto-close open CDATA", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.NoError(t, w.StartCDATA())
			require.NoError(t, w.EndDocument())
			require.Contains(t, buf.String(), "]]>")
		})

		t.Run("auto-close open comment", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.NoError(t, w.StartComment())
			require.NoError(t, w.EndDocument())
			require.Contains(t, buf.String(), "-->")
		})

		t.Run("auto-close open DTD", func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("doc", "", ""))
			require.NoError(t, w.EndDocument())
			require.Contains(t, buf.String(), "<!DOCTYPE doc>")
		})
	})
}

// TestStickyErrorEdges exercises sticky-error propagation and nil-writer paths.
func TestStickyErrorEdges(t *testing.T) {
	t.Parallel()

	// propagation: once a write fails, subsequent ops are no-ops and the
	// relevant guard branches (w.err != nil) are taken.
	t.Run("propagation", func(t *testing.T) {
		t.Parallel()
		fw := &failWriter{failAfter: 3}
		w := stream.NewWriter(fw)
		// First successful-ish write then failure on a longer write.
		_ = w.StartElement("rootlong")
		require.Error(t, w.Error())
		// All subsequent calls should observe the sticky error.
		require.Error(t, w.StartElement("x"))
		require.Error(t, w.StartElementNS("p", "x", "urn:p"))
		require.Error(t, w.EndElement())
		require.Error(t, w.FullEndElement())
		require.Error(t, w.WriteElement("a", "b"))
		require.Error(t, w.WriteElementNS("p", "a", "urn:p", "b"))
		require.Error(t, w.StartAttribute("a"))
		require.Error(t, w.StartAttributeNS("p", "a", "urn:p"))
		require.Error(t, w.EndAttribute())
		require.Error(t, w.WriteAttribute("a", "b"))
		require.Error(t, w.WriteAttributeNS("p", "a", "urn:p", "b"))
		require.Error(t, w.WriteString("x"))
		require.Error(t, w.WriteRaw("x"))
		require.Error(t, w.StartComment())
		require.Error(t, w.EndComment())
		require.Error(t, w.WriteComment("x"))
		require.Error(t, w.StartPI("t"))
		require.Error(t, w.EndPI())
		require.Error(t, w.WritePI("t", "x"))
		require.Error(t, w.StartCDATA())
		require.Error(t, w.EndCDATA())
		require.Error(t, w.WriteCDATA("x"))
		require.Error(t, w.StartDTD("d", "", ""))
		require.Error(t, w.EndDTD())
		require.Error(t, w.WriteDTDElement("e", "EMPTY"))
		require.Error(t, w.WriteDTDAttlist("e", "a CDATA #IMPLIED"))
		require.Error(t, w.WriteDTDEntity(false, "e", "v"))
		require.Error(t, w.WriteDTDExternalEntity(false, "e", "", "s", ""))
		require.Error(t, w.WriteDTDNotation("n", "", "s"))
		require.Error(t, w.StartDocument("", "", ""))
		require.Error(t, w.EndDocument())
		require.Error(t, w.Flush())
	})

	t.Run("nil writer error", func(t *testing.T) {
		t.Parallel()
		w := stream.NewWriter(nil)
		err := w.StartElement("root")
		require.Error(t, err)
		require.Contains(t, err.Error(), "output writer is nil")
	})
}

// TestWriteRawStates covers WriteRaw in attribute state, and WriteRaw invalid
// state.
func TestWriteRawStates(t *testing.T) {
	t.Parallel()

	t.Run("WriteRaw in attribute", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartAttribute("a"))
		require.NoError(t, w.WriteRaw("raw&value"))
		require.NoError(t, w.EndAttribute())
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndDocument())
		require.Contains(t, buf.String(), `a="raw&value"`)
	})

	t.Run("WriteRaw invalid state inside comment", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartComment())
		require.Error(t, w.WriteRaw("x"))
	})
}

// TestWriteEscapedBranches covers writeEscaped: the '\n' and '\t' raw-byte (text
// mode) and the quote rawByte branch where quoteChar differs.
func TestWriteEscapedBranches(t *testing.T) {
	t.Parallel()

	t.Run("tab and newline preserved in text", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.WriteString("a\tb\nc"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndDocument())
		require.Contains(t, buf.String(), "a\tb\nc")
	})

	t.Run("attribute with single quote when double-quoted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.WriteAttribute("a", "it's"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndDocument())
		require.Contains(t, buf.String(), `a="it's"`)
	})

	t.Run("attribute newline and tab escaped", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.WriteAttribute("a", "x\ny\tz"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndDocument())
		out := buf.String()
		require.Contains(t, out, "&#10;")
		require.Contains(t, out, "&#9;")
	})
}

// TestVersionValidation covers isValidXMLVersion rejections via StartDocument:
// empty fractional, non-1. form.
func TestVersionValidation(t *testing.T) {
	t.Parallel()

	t.Run("trailing dot only", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.Error(t, w.StartDocument("1.", "", ""))
	})

	t.Run("non-numeric fraction", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.Error(t, w.StartDocument("1.x", "", ""))
	})

	t.Run("valid 1.1", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("1.1", "", ""))
	})
}

// TestEndAttributeOutside covers EndAttribute outside attribute returns an error.
func TestEndAttributeOutside(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.Error(t, w.EndAttribute())
}
