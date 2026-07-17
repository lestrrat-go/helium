package shim_test

import (
	"bytes"
	"context"
	stdxml "encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/shim"
	"github.com/stretchr/testify/require"
)

// byteOnlyReader implements io.Reader and io.ByteReader. When passed to
// ensureReader via CharsetReader, the decoder wraps it in a byteReaderWrapper
// that drives ReadByte, exercising byteReaderWrapper.Read.
type byteOnlyReader struct {
	r *bytes.Reader
}

func (b *byteOnlyReader) Read(p []byte) (int, error) { return b.r.Read(p) }
func (b *byteOnlyReader) ReadByte() (byte, error)    { return b.r.ReadByte() }

func TestDecoder(t *testing.T) {
	t.Run("charset-reader-byte-reader", func(t *testing.T) {
		input := `<?xml version="1.0" encoding="latin1"?><root>hi</root>`
		dec := shim.NewDecoder(context.Background(), strings.NewReader(input))
		dec.CharsetReader = func(charset string, in io.Reader) (io.Reader, error) {
			if charset != "latin1" {
				t.Errorf("unexpected charset %q", charset)
			}
			// Drain the converted input and return a ByteReader-only wrapper so
			// the decoder must adapt it via byteReaderWrapper.
			data, err := io.ReadAll(in)
			if err != nil {
				return nil, err
			}
			return &byteOnlyReader{r: bytes.NewReader(data)}, nil
		}

		var sawRoot bool
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Token error: %v", err)
			}
			if se, ok := tok.(shim.StartElement); ok && se.Name.Local == "root" {
				sawRoot = true
			}
		}
		if !sawRoot {
			t.Fatal("did not see <root> element")
		}
	})

	t.Run("close", func(t *testing.T) {
		dec := shim.NewDecoder(context.Background(), strings.NewReader(`<root><a/><b/></root>`))
		// Read one token to start the SAX goroutine, then Close to cancel it.
		if _, err := dec.Token(); err != nil {
			t.Fatalf("Token error: %v", err)
		}
		dec.Close()
		// Closing again must be safe (cancel is idempotent).
		dec.Close()
	})

	t.Run("close-before-read", func(t *testing.T) {
		dec := shim.NewDecoder(context.Background(), strings.NewReader(`<root/>`))
		// Close before any read: cancel is non-nil but goroutine never started.
		dec.Close()
	})

	t.Run("input-pos", func(t *testing.T) {
		dec := shim.NewDecoder(context.Background(), strings.NewReader(`<root>text</root>`))
		// Before reading, line is 1 -> returns (1,1).
		if l, c := dec.InputPos(); l != 1 || c != 1 {
			t.Fatalf("expected (1,1) before read, got (%d,%d)", l, c)
		}
		for {
			if _, err := dec.Token(); err != nil {
				break
			}
		}
		// After reading content, positions should be reported (line >= 1).
		if l, _ := dec.InputPos(); l < 1 {
			t.Fatalf("expected line >= 1, got %d", l)
		}
	})
}

// TestDecoderAndUnmarshalAgreeOnXMLDecl pins that the shim's two entry points
// reach the SAME accept/reject verdict on an XML declaration. A package that
// rejects a document through Unmarshal and accepts it through Decoder is
// incoherent, so the agreement itself is the assertion: each case runs both and
// requires they agree, and that the verdict matches the grammar (XML 1.0 §2.8):
//
//	XMLDecl      ::= '<?xml' VersionInfo EncodingDecl? SDDecl? S? '?>'
//	VersionInfo  ::= S 'version' Eq ("'" VersionNum "'" | '"' VersionNum '"')
//	EncodingDecl ::= S 'encoding' Eq ('"' EncName '"' | "'" EncName "'")
//	EncName      ::= [A-Za-z] ([A-Za-z0-9._] | '-')*
//	SDDecl       ::= S 'standalone' Eq (('"' ('yes'|'no') '"') | ("'" ('yes'|'no') "'"))
//	Eq           ::= S? '=' S?
//
// The version is mandatory and first, the three pseudo-attributes are the only
// ones admitted, each may appear once, and the order is fixed.
func TestDecoderAndUnmarshalAgreeOnXMLDecl(t *testing.T) {
	type item struct {
		Value string `xml:"value"`
	}

	const body = `<item><value>hi</value></item>`

	for _, tc := range []struct {
		name string
		xml  string
		// conforms is the XMLDecl verdict both entry points must reach.
		conforms bool
		// why states what qualifies or disqualifies the declaration.
		why string
	}{
		{
			name:     "version and encoding",
			xml:      `<?xml version="1.0" encoding="UTF-8"?>` + body,
			conforms: true,
			why:      "VersionInfo then EncodingDecl is the grammar's own order",
		},
		{
			name:     "no declaration at all",
			xml:      body,
			conforms: true,
			why:      "the XMLDecl is optional; a document may omit it entirely",
		},
		{
			name:     "version only",
			xml:      `<?xml version="1.0"?>` + body,
			conforms: true,
			why:      "EncodingDecl and SDDecl are both optional",
		},
		{
			name:     "version and standalone",
			xml:      `<?xml version="1.0" standalone="yes"?>` + body,
			conforms: true,
			why:      "SDDecl may follow VersionInfo with EncodingDecl absent",
		},
		{
			name:     "single quotes",
			xml:      `<?xml version='1.0' encoding='UTF-8'?>` + body,
			conforms: true,
			why:      "both quote characters are admitted for every value",
		},
		{
			name:     "extra whitespace",
			xml:      `<?xml  version="1.0"  ?>` + body,
			conforms: true,
			why:      "S is admitted before VersionInfo and before the closing ?>",
		},
		{
			name:     "spaces around eq",
			xml:      `<?xml version="1.0" encoding = "UTF-8" ?>` + body,
			conforms: true,
			why:      `Eq ::= S? '=' S? — the spaces around "=" are legal`,
		},
		{
			name:     "charset pseudo-attribute",
			xml:      `<?xml version="1.0" charset="UTF-8"?>` + body,
			conforms: false,
			why:      "charset is not one of the three admitted pseudo-attributes",
		},
		{
			name:     "empty decl",
			xml:      `<?xml?>` + body,
			conforms: false,
			why:      "VersionInfo is mandatory, and this declares nothing at all",
		},
		{
			name:     "no version",
			xml:      `<?xml encoding="UTF-8"?>` + body,
			conforms: false,
			why:      "VersionInfo is mandatory; an encoding alone does not satisfy it",
		},
		{
			name:     "version empty string",
			xml:      `<?xml version=""?>` + body,
			conforms: false,
			why:      "the empty string is not a VersionNum",
		},
		{
			name:     "encoding empty string",
			xml:      `<?xml version="1.0" encoding=""?>` + body,
			conforms: false,
			why:      "EncName must begin with a letter, so it is never empty",
		},
		{
			name:     "charset ahead of version",
			xml:      `<?xml charset version="2.0"?>` + body,
			conforms: false,
			why:      "charset is not admitted, and it is not even a pseudo-attribute",
		},
		{
			name:     "pseudo-attributes out of order",
			xml:      `<?xml encoding="UTF-8" version="1.0"?>` + body,
			conforms: false,
			why:      "XMLDecl fixes the order: version, then encoding, then standalone",
		},
		{
			name:     "unsupported version",
			xml:      `<?xml version="2.0"?>` + body,
			conforms: false,
			why:      `helium supports the 1.x family (1.0 and 1.1); a version outside it is rejected`,
		},
		{
			name:     "standalone not yes or no",
			xml:      `<?xml version="1.0" standalone="maybe"?>` + body,
			conforms: false,
			why:      `SDDecl admits only "yes" and "no"`,
		},
		{
			name:     "repeated version",
			xml:      `<?xml version="1.0" version="1.0"?>` + body,
			conforms: false,
			why:      "each pseudo-attribute may appear at most once",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out item
			unmarshalErr := shim.Unmarshal([]byte(tc.xml), &out)
			decodeErr := drainDecoder(t, tc.xml)

			// The agreement is the point: state it before the verdict, so a
			// divergence reports as a divergence rather than as one entry
			// point's failure.
			require.Equal(t, unmarshalErr != nil, decodeErr != nil,
				"Unmarshal and Decoder must agree on %q\n  Unmarshal: %v\n  Decoder:   %v",
				tc.xml, unmarshalErr, decodeErr)

			if tc.conforms {
				require.NoError(t, unmarshalErr, tc.why)
				require.NoError(t, decodeErr, tc.why)
				return
			}
			require.Error(t, unmarshalErr, tc.why)
			require.Error(t, decodeErr, tc.why)
		})
	}
}

// TestDecoderXMLDeclErrorMessages pins the wording the Decoder reports for a
// declaration it rejects. A version outside 1.0 keeps the unsupported-version
// message rather than being reported as a grammar violation, and a non-UTF-8
// encoding with no CharsetReader keeps its own message.
func TestDecoderXMLDeclErrorMessages(t *testing.T) {
	const body = `<item><value>hi</value></item>`

	t.Run("unsupported version", func(t *testing.T) {
		err := drainDecoder(t, `<?xml version="2.0"?>`+body)
		require.Error(t, err)
		// The Decoder defers to helium, so the verdict is helium's: it names the
		// version outside the 1.x family it rejected.
		require.Contains(t, err.Error(), `unsupported XML version "2.0"`)
	})

	t.Run("encoding without charset reader", func(t *testing.T) {
		err := drainDecoder(t, `<?xml version="1.0" encoding="ISO-8859-1"?>`+body)
		require.Error(t, err)
		require.Contains(t, err.Error(), `xml: encoding "ISO-8859-1" declared but Decoder.CharsetReader is nil`)
	})

	t.Run("malformed decl is a syntax error", func(t *testing.T) {
		err := drainDecoder(t, `<?xml version="1.0" charset="UTF-8"?>`+body)
		require.Error(t, err)
		var syntaxErr *stdxml.SyntaxError
		require.ErrorAs(t, err, &syntaxErr)
	})
}

// TestDecoderNonDeclProcInstUnaffected pins that the XMLDecl grammar is applied
// to the XML declaration ONLY. A processing instruction whose target merely
// starts with "xml" is an ordinary ProcInst, and its data is arbitrary — holding
// it to the declaration's grammar would reject real documents.
func TestDecoderNonDeclProcInstUnaffected(t *testing.T) {
	const body = `<item><value>hi</value></item>`

	for _, tc := range []struct {
		name string
		pi   string
	}{
		{"stylesheet", `<?xml-stylesheet href="a.xsl" type="text/xsl"?>`},
		{"stylesheet with charset", `<?xml-stylesheet charset="UTF-8"?>`},
		{"target starting with xml", `<?xmlfoo version="2.0"?>`},
		{"unrelated target", `<?php echo "x"; ?>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, drainDecoder(t, tc.pi+body),
				"a non-declaration PI carries arbitrary data")
			require.NoError(t, drainDecoder(t, `<?xml version="1.0"?>`+tc.pi+body),
				"a real declaration ahead of it does not change that")
		})
	}
}

// tokenSig renders a token as a stable, comparable signature string. Only the
// discriminating fields are included, which is enough to prove two token
// sequences match.
func tokenSig(tok stdxml.Token) string {
	switch v := tok.(type) {
	case stdxml.StartElement:
		return "Start(" + v.Name.Local + ")"
	case stdxml.EndElement:
		return "End(" + v.Name.Local + ")"
	case stdxml.CharData:
		return "Char(" + string(v) + ")"
	case stdxml.Comment:
		return "Comment(" + string(v) + ")"
	case stdxml.ProcInst:
		return "PI(" + v.Target + "|" + string(v.Inst) + ")"
	case stdxml.Directive:
		return "Directive(" + string(v) + ")"
	}
	return "?"
}

// shimTokenSigs drains a shim Decoder to EOF and returns each token's signature.
func shimTokenSigs(t *testing.T, xml string) []string {
	t.Helper()
	d := shim.NewDecoder(t.Context(), strings.NewReader(xml))
	defer d.Close()
	var sigs []string
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return sigs
		}
		require.NoError(t, err)
		sigs = append(sigs, tokenSig(tok))
	}
}

// stdlibTokenSigs drains an encoding/xml Decoder to EOF and returns each token's
// signature. It is the oracle the shim must match.
func stdlibTokenSigs(t *testing.T, xml string) []string {
	t.Helper()
	d := stdxml.NewDecoder(strings.NewReader(xml))
	var sigs []string
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return sigs
		}
		require.NoError(t, err)
		sigs = append(sigs, tokenSig(tok))
	}
}

// TestDecoderPrologNoDuplicate proves that non-declaration prolog tokens
// (comments, processing instructions) are emitted exactly once, matching
// encoding/xml. In-root comments/PIs must also stay single-emission.
func TestDecoderPrologNoDuplicate(t *testing.T) {
	inputs := []string{
		`<?xml version="1.0"?><!-- hi --><?pi data?><root/>`,
		`<!-- lead --><root/>`,
		`<?xml version="1.0"?><?xml-stylesheet href="a.xsl"?><root>text</root>`,
		`<!-- a --><!-- b --><?p1?><?p2?><root/>`,
		`<root><!-- inside --><?pi?></root>`,
		`<?xml version="1.0"?><root/>`,
	}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			want := stdlibTokenSigs(t, in)
			got := shimTokenSigs(t, in)
			require.Equal(t, want, got)
		})
	}
}

// drainDecoder reads every token of xml through a shim Decoder and returns the
// first error that is not the clean io.EOF ending the stream.
func drainDecoder(t *testing.T, xml string) error {
	t.Helper()
	d := shim.NewDecoder(t.Context(), strings.NewReader(xml))
	defer d.Close()
	for {
		_, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
