package shim_test

import (
	stdxml "encoding/xml"
	"io"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/lestrrat-go/helium/shim"
	"github.com/stretchr/testify/require"
)

// Declaration and document fragments shared by the tests below.
const (
	declV10   = `<?xml version="1.0"?>`
	rootOnly  = `<root/>`
	itemOnly  = `<item/>`
	rootOpen  = `<root>`
	rootClose = `</root>`
)

// unmarshalDoc runs src through [shim.Unmarshal] and returns its verdict.
func unmarshalDoc(src string) error {
	var v struct{}
	return shim.Unmarshal([]byte(src), &v)
}

// drainTokens reads dec to EOF and returns the first error, so a Decoder's
// accept/reject verdict is comparable with unmarshalDoc's.
func drainTokens(dec *shim.Decoder) error {
	for {
		_, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// decodeDoc drains src through a reader-backed [shim.Decoder].
func decodeDoc(t *testing.T, src string) error {
	t.Helper()
	return drainTokens(shim.NewDecoder(t.Context(), strings.NewReader(src)))
}

// tokenDecodeDoc drains src through a TokenReader-backed [shim.Decoder] whose
// tokens come from an [encoding/xml] decoder — the canonical TokenReader
// composition. [shim.Decoder] applies one set of declaration rules whichever
// constructor built it, so this verdict must match decodeDoc's on every input,
// even though the wrapped stdlib decoder is itself permissive.
func tokenDecodeDoc(t *testing.T, src string) error {
	t.Helper()
	return drainTokens(shim.NewTokenDecoder(t.Context(), stdxml.NewDecoder(strings.NewReader(src))))
}

// requireAllReject asserts that every shim entry point rejects src.
func requireAllReject(t *testing.T, src string) {
	t.Helper()
	require.Error(t, unmarshalDoc(src), "Unmarshal must reject")
	require.Error(t, decodeDoc(t, src), "reader-backed Decoder must reject")
	require.Error(t, tokenDecodeDoc(t, src), "TokenReader-backed Decoder must reject")
}

// requireAllAccept asserts that every shim entry point accepts src.
func requireAllAccept(t *testing.T, src string) {
	t.Helper()
	require.NoError(t, unmarshalDoc(src), "Unmarshal must accept")
	require.NoError(t, decodeDoc(t, src), "reader-backed Decoder must accept")
	require.NoError(t, tokenDecodeDoc(t, src), "TokenReader-backed Decoder must accept")
}

// TestXMLDeclPosition covers where an XML declaration may appear. XMLDecl is
// only legal as the very first thing in a document — at position 0, with only a
// byte-order mark allowed ahead of it (leading whitespace displaces it) — and
// every entry point must agree on that.
func TestXMLDeclPosition(t *testing.T) {
	t.Run("declaration not at the start is rejected", func(t *testing.T) {
		for name, src := range map[string]string{
			"after an earlier declaration": declV10 + declV10 + rootOnly,
			"after a comment":              `<!-- c -->` + declV10 + rootOnly,
			"after a processing instruction": `<?xml-stylesheet href="a.xsl"?>` +
				declV10 + rootOnly,
			"after a doctype":         `<!DOCTYPE root>` + declV10 + rootOnly,
			"inside the root element": rootOpen + declV10 + rootClose,
		} {
			t.Run(name, func(t *testing.T) {
				requireAllReject(t, src)
			})
		}
	})

	// Leading whitespace ahead of a declaration is REJECTED by every entry
	// point: an XML declaration is legal only at document position 0
	// (prolog ::= XMLDecl? Misc* ...), so any whitespace ahead of it makes it
	// misplaced. shim's verdict is helium's, and helium rejects a declaration
	// not at the start of the document.
	t.Run("leading whitespace before a declaration is rejected", func(t *testing.T) {
		for name, src := range map[string]string{
			"space and newline": " \n" + declV10 + rootOnly,
			"a tab":             "\t" + declV10 + rootOnly,
			"a newline":         "\n" + declV10 + rootOnly,
			"spaces then a tab": "  \n\t" + declV10 + itemOnly,
		} {
			t.Run(name, func(t *testing.T) {
				requireAllReject(t, src)
			})
		}
	})

	// Leading whitespace ahead of the ROOT ELEMENT — with no declaration —
	// stays accepted: whitespace before an element is well-formed. Only
	// whitespace ahead of a DECLARATION rejects, because a declaration must be at
	// position 0 whereas an element need not.
	t.Run("leading whitespace before an element with no declaration is accepted", func(t *testing.T) {
		for name, src := range map[string]string{
			"two spaces":       "  " + itemOnly,
			"newline and tab":  "\n\t" + itemOnly,
			"no leading space": itemOnly,
		} {
			t.Run(name, func(t *testing.T) {
				requireAllAccept(t, src)
			})
		}
	})

	// An ordinary PI inside the root element is not a declaration and is
	// unaffected by the placement rule.
	t.Run("an ordinary PI inside the root element is accepted", func(t *testing.T) {
		requireAllAccept(t, rootOpen+`<?target data?>`+rootClose)
	})
}

// TestXMLDeclTargetBoundary covers processing instructions whose target merely
// begins with "xml". PITarget ::= Name - (('X'|'x')('M'|'m')('L'|'l')) subtracts
// only the exact name "xml", so "xmlversion" is a legal target and a document
// using one is well-formed. No entry point may read such a PI as a
// declaration.
func TestXMLDeclTargetBoundary(t *testing.T) {
	for name, src := range map[string]string{
		"xmlversion with a double-quoted value":          `<?xmlversion ="2.0"?>` + rootOnly,
		"xmlversion with a single-quoted value":          `<?xmlversion ='2.0'?>` + rootOnly,
		"xmlversion with spaces around =":                `<?xmlversion  =  '2.0'?>` + rootOnly,
		"xmlversion with no data":                        `<?xmlversion?>` + rootOnly,
		"xmlfoo":                                         `<?xmlfoo?>` + rootOnly,
		"xmlversion carrying a version pseudo-attribute": `<?xmlversion version="2.0"?>` + rootOnly,
		"xmlencoding":                                    `<?xmlencoding ="x"?>` + rootOnly,
	} {
		t.Run(name, func(t *testing.T) {
			requireAllAccept(t, src)
		})
	}
}

// TestXMLDeclReservedTargetCase covers the reserved processing-instruction
// target "xml" in non-lowercase casings. PITarget ::= Name -
// (('X'|'x')('M'|'m')('L'|'l')) reserves the name in ANY case, so <?XML?>,
// <?Xml?> and <?xMl?> are illegal targets wherever they appear and every entry
// point must reject them — while a longer xml-prefixed name stays a legal
// ordinary PI. This pins that the pre-parsed TokenReader path agrees with
// Unmarshal and the reader-backed Decoder rather than accepting a reserved-cased
// target as an ordinary PI.
func TestXMLDeclReservedTargetCase(t *testing.T) {
	t.Run("reserved-cased target is rejected everywhere", func(t *testing.T) {
		for name, src := range map[string]string{
			"upper as leading declaration":    `<?XML version="1.0"?>` + rootOnly,
			"title as leading declaration":    `<?Xml version="1.0"?>` + rootOnly,
			"mixed as leading declaration":    `<?xMl version="1.0"?>` + rootOnly,
			"upper with no pseudo-attributes": `<?XML?>` + rootOnly,
			"upper carrying a version 2.0":    `<?XML version="2.0"?>` + rootOnly,
			"mid-document reserved-cased PI":  rootOpen + `<?XML foo?>` + rootClose,
		} {
			t.Run(name, func(t *testing.T) {
				requireAllReject(t, src)
			})
		}
	})

	// A longer xml-prefixed target is not the reserved name (EqualFold matches
	// only equal-length strings), so it stays a legal ordinary PI. Pinning the
	// accept here guards against an over-eager case-fold that starts rejecting
	// well-formed documents.
	t.Run("longer xml-prefixed target stays accepted", func(t *testing.T) {
		for name, src := range map[string]string{
			"xmlfoo":            `<?xmlfoo?>` + rootOnly,
			"xml-stylesheet":    `<?xml-stylesheet href="a.xsl"?>` + rootOnly,
			"xmlversion":        `<?xmlversion ="2.0"?>` + rootOnly,
			"XMLfoo mixed case": `<?XMLfoo?>` + rootOnly,
		} {
			t.Run(name, func(t *testing.T) {
				requireAllAccept(t, src)
			})
		}
	})
}

// deliveredTokens is a permissive TokenReader that hands the shim a fixed token
// stream verbatim. It stands in for a TokenReader that delivers an XML
// declaration the shim then judges — unlike encoding/xml's own decoder, which
// rejects a version outside 1.0 during tokenization (both Token and RawToken
// error) and so never delivers a 1.1 declaration to the shim at all.
type deliveredTokens struct {
	toks []stdxml.Token
	i    int
}

func (d *deliveredTokens) Token() (stdxml.Token, error) {
	if d.i >= len(d.toks) {
		return nil, io.EOF
	}
	tok := d.toks[d.i]
	d.i++
	return tok, nil
}

// decodeDeliveredDecl drains a TokenReader-backed [shim.Decoder] fed the given
// leading tokens, then a declaration ProcInst carrying inst, then a minimal
// element. It exercises the shim's declaration decision on the TokenReader path
// without an encoding/xml decoder in the way.
func decodeDeliveredDecl(t *testing.T, leading []stdxml.Token, inst string) error {
	t.Helper()
	toks := append(append([]stdxml.Token(nil), leading...),
		stdxml.ProcInst{Target: "xml", Inst: []byte(inst)},
		stdxml.StartElement{Name: stdxml.Name{Local: "item"}},
		stdxml.EndElement{Name: stdxml.Name{Local: "item"}},
	)
	return drainTokens(shim.NewTokenDecoder(t.Context(), &deliveredTokens{toks: toks}))
}

// TestXMLDeclVersion11Accepted pins that shim accepts XML version 1.1: helium
// supports it, and shim's verdict is helium's, where encoding/xml rejects it.
// Unmarshal and the reader-backed Decoder see the bytes and accept. The
// TokenReader path defers to helium too, accepting a 1.1 declaration once it is
// delivered as a token — encoding/xml's own decoder cannot deliver one (it
// rejects 1.1 during tokenization), so that particular composition is the one
// place shim cannot exercise, a limitation of encoding/xml rather than of shim.
func TestXMLDeclVersion11Accepted(t *testing.T) {
	for name, src := range map[string]string{
		"version only":         `<?xml version="1.1"?>` + itemOnly,
		"version and encoding": `<?xml version="1.1" encoding="UTF-8"?>` + itemOnly,
	} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, unmarshalDoc(src), "Unmarshal must accept 1.1")
			require.NoError(t, decodeDoc(t, src), "reader-backed Decoder must accept 1.1")
		})
	}
	t.Run("TokenReader accepts a delivered 1.1 declaration", func(t *testing.T) {
		require.NoError(t, decodeDeliveredDecl(t, nil, `version="1.1"`))
	})
}

// TestXMLDeclByteOrderMark pins that a leading UTF-8 byte-order mark does not
// displace an XML declaration from the start of the document — a BOM is not
// content.
func TestXMLDeclByteOrderMark(t *testing.T) {
	const bom = "\uFEFF"

	// A BOM ahead of a 1.0 declaration is accepted by every entry point,
	// including the encoding/xml-backed TokenReader (encoding/xml delivers a 1.0
	// declaration), which pins that a leading BOM token is not counted as content.
	t.Run("accepted ahead of a 1.0 declaration", func(t *testing.T) {
		requireAllAccept(t, bom+`<?xml version="1.0"?>`+itemOnly)
	})

	// A BOM ahead of an unsupported version is rejected by every entry point.
	t.Run("rejected ahead of an unsupported version", func(t *testing.T) {
		requireAllReject(t, bom+`<?xml version="2.0"?>`+itemOnly)
	})

	// A BOM ahead of a 1.1 declaration is accepted wherever the declaration is
	// seen: Unmarshal and the reader Decoder see the bytes, and the TokenReader
	// path accepts it once the BOM and declaration are delivered as tokens.
	// encoding/xml's own decoder again cannot deliver the 1.1 declaration.
	t.Run("accepted ahead of a 1.1 declaration where seen", func(t *testing.T) {
		src := bom + `<?xml version="1.1"?>` + itemOnly
		require.NoError(t, unmarshalDoc(src), "Unmarshal must accept BOM+1.1")
		require.NoError(t, decodeDoc(t, src), "reader-backed Decoder must accept BOM+1.1")
		require.NoError(t, decodeDeliveredDecl(t, []stdxml.Token{stdxml.CharData(bom)}, `version="1.1"`),
			"TokenReader must accept a delivered BOM+1.1")
	})
}

// utf16leBOM encodes s as little-endian UTF-16 with a leading byte-order mark.
// A declaration in such a document is not ASCII, so a byte-level scan cannot see
// it; the reader-backed Decoder's encoding decision must instead come from
// helium's decoded encoding, matching Unmarshal.
func utf16leBOM(s string) string {
	units := utf16.Encode([]rune(s))
	b := make([]byte, 0, 2+len(units)*2)
	b = append(b, 0xFF, 0xFE)
	for _, u := range units {
		b = append(b, byte(u), byte(u>>8))
	}
	return string(b)
}

// TestXMLDeclNonUTF8Encoding pins the shim's single encoding policy: a document
// declaring a non-UTF-8 encoding is rejected without a CharsetReader, and every
// entry point that sees the declaration must reach that verdict coherently — even
// when the declaration is itself in a fixed-width Unicode encoding that a
// byte-level scan cannot read.
func TestXMLDeclNonUTF8Encoding(t *testing.T) {
	t.Run("declared non-UTF-8 encoding is rejected without a CharsetReader", func(t *testing.T) {
		// UTF-16-encoded declarations are invisible to the byte-level prolog
		// scanner, so the reader-backed Decoder must reject them through helium's
		// decoded encoding, exactly as Unmarshal does. Version 2.0 is rejected for
		// its version as well; that is still a reject.
		for name, src := range map[string]string{
			"UTF-16 declaring 1.0": utf16leBOM(`<?xml version="1.0" encoding="UTF-16"?>` + rootOnly),
			"UTF-16 declaring 1.1": utf16leBOM(`<?xml version="1.1" encoding="UTF-16"?>` + rootOnly),
			"UTF-16 declaring 2.0": utf16leBOM(`<?xml version="2.0" encoding="UTF-16"?>` + rootOnly),
		} {
			t.Run(name, func(t *testing.T) {
				require.Error(t, unmarshalDoc(src), "Unmarshal must reject")
				require.Error(t, decodeDoc(t, src), "reader-backed Decoder must reject")
			})
		}

		// A UTF-8 document declaring a non-UTF-8 encoding is rejected on both
		// paths too. The byte scanner does see this declaration, so this guards
		// the existing behavior against regression.
		t.Run("ISO-8859-1 in a UTF-8 document", func(t *testing.T) {
			src := `<?xml version="1.0" encoding="ISO-8859-1"?>` + rootOnly
			require.Error(t, unmarshalDoc(src), "Unmarshal must reject")
			require.Error(t, decodeDoc(t, src), "reader-backed Decoder must reject")
		})
	})

	// A fixed-width Unicode document with no declared encoding names no encoding,
	// so both paths accept it. A byte-order-mark sniff would wrongly reject it,
	// which is why the policy is driven by helium's decoded encoding, not the BOM.
	t.Run("UTF-16 without a declared encoding is accepted", func(t *testing.T) {
		src := utf16leBOM(rootOnly)
		require.NoError(t, unmarshalDoc(src), "Unmarshal must accept")
		require.NoError(t, decodeDoc(t, src), "reader-backed Decoder must accept")
	})

	// With a CharsetReader set, a declared non-UTF-8 encoding is honored, so the
	// reader-backed Decoder accepts it (policy unchanged).
	t.Run("declared non-UTF-8 encoding is accepted with a CharsetReader", func(t *testing.T) {
		src := utf16leBOM(`<?xml version="1.0" encoding="UTF-16"?>` + rootOnly)
		dec := shim.NewDecoder(t.Context(), strings.NewReader(src))
		dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
		require.NoError(t, drainTokens(dec), "reader-backed Decoder must accept with a CharsetReader")
	})

	// Ordinary UTF-8 declarations, and a UTF-8 byte-order mark ahead of one, stay
	// accepted on every entry point — the fixed-width detection must not disturb
	// them.
	t.Run("UTF-8 declarations stay accepted", func(t *testing.T) {
		requireAllAccept(t, `<?xml version="1.0" encoding="UTF-8"?>`+itemOnly)
		requireAllAccept(t, declV10+itemOnly)
		requireAllAccept(t, "\uFEFF"+declV10+itemOnly)
	})
}

// TestXMLDeclAgreement pins the accept/reject verdict of every declaration
// shape in the shim's bar, through every entry point, which must agree.
func TestXMLDeclAgreement(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		for name, src := range map[string]string{
			"version and encoding":         `<?xml version="1.0" encoding="UTF-8"?>` + itemOnly,
			"no declaration":               itemOnly,
			"version only":                 declV10 + itemOnly,
			"version and standalone":       `<?xml version="1.0" standalone="yes"?>` + itemOnly,
			"single-quoted values":         `<?xml version='1.0' encoding='UTF-8'?>` + itemOnly,
			"extra whitespace":             `<?xml  version="1.0"  ?>` + itemOnly,
			"whitespace around =":          `<?xml version="1.0" encoding = "UTF-8" ?>` + itemOnly,
			"stylesheet PI without a decl": `<?xml-stylesheet type="text/xsl" href="a.xsl"?>` + itemOnly,
			"PI carrying a version pseudo": `<?xml-stylesheet charset="x" version="9"?>` +
				itemOnly,
			"declaration then a stylesheet PI": declV10 + `<?xml-stylesheet href="a.xsl"?>` + itemOnly,
		} {
			t.Run(name, func(t *testing.T) {
				requireAllAccept(t, src)
			})
		}
	})

	t.Run("rejected", func(t *testing.T) {
		for name, src := range map[string]string{
			"charset pseudo-attribute":  `<?xml version="1.0" charset="UTF-8"?>` + itemOnly,
			"no pseudo-attributes":      `<?xml?>` + itemOnly,
			"encoding without version":  `<?xml encoding="UTF-8"?>` + itemOnly,
			"empty version":             `<?xml version=""?>` + itemOnly,
			"empty encoding":            `<?xml version="1.0" encoding=""?>` + itemOnly,
			"bare charset then version": `<?xml charset version="2.0"?>` + itemOnly,
			"encoding before version":   `<?xml encoding="UTF-8" version="1.0"?>` + itemOnly,
		} {
			t.Run(name, func(t *testing.T) {
				requireAllReject(t, src)
			})
		}
	})

	// A declared version outside 1.0 is named as unsupported by both the
	// Unmarshal and reader-backed Decoder paths, rather than reported as a bare
	// grammar violation. The TokenReader path also rejects it (asserted through
	// requireAllReject) but is not pinned to the same wording here.
	t.Run("unsupported version is named by both entry points", func(t *testing.T) {
		src := `<?xml version="2.0"?>` + itemOnly

		uErr := unmarshalDoc(src)
		require.Error(t, uErr)
		require.Contains(t, uErr.Error(), `unsupported XML version "2.0"`)

		dErr := decodeDoc(t, src)
		require.Error(t, dErr)
		require.Contains(t, dErr.Error(), `unsupported XML version "2.0"`)

		requireAllReject(t, src)
	})
}

// TestXMLDeclInstTerminator pins that a TokenReader delivering an XML
// declaration whose Inst carries an embedded "?>" is rejected. A real
// declaration's pseudo-attribute region cannot contain "?>" — it is the
// declaration's own terminator — so an Inst that does is smuggling a second PI
// or arbitrary prolog markup past the declaration boundary. shim rejects it
// before reconstructing the declaration for helium, so the smuggled markup is
// never parsed as legitimate prolog nodes.
func TestXMLDeclInstTerminator(t *testing.T) {
	balanced := []stdxml.Token{
		stdxml.StartElement{Name: stdxml.Name{Local: "r"}},
		stdxml.EndElement{Name: stdxml.Name{Local: "r"}},
	}

	t.Run("an Inst with an embedded ?> is rejected", func(t *testing.T) {
		for name, inst := range map[string]string{
			"smuggling a second PI":   `version="1.0"?><?foo`,
			"smuggling a comment":     `version="1.0"?><!--x-->`,
			"smuggling a closing tag": `version="1.0"?></root><evil>`,
			"a bare terminator":       `version="1.0"?>`,
		} {
			t.Run(name, func(t *testing.T) {
				toks := append([]stdxml.Token{
					stdxml.ProcInst{Target: "xml", Inst: []byte(inst)},
				}, balanced...)
				err := drainTokens(shim.NewTokenDecoder(t.Context(), &deliveredTokens{toks: toks}))
				require.Error(t, err, "an Inst with an embedded ?> must be rejected")
			})
		}
	})

	// A well-formed declaration delivered the same way stays accepted — the
	// guard rejects only the "?>" terminator, never a valid pseudo-attribute
	// region.
	t.Run("a well-formed declaration stays accepted", func(t *testing.T) {
		for name, inst := range map[string]string{
			"version only":         `version="1.0"`,
			"version and encoding": `version="1.1" encoding="UTF-8"`,
		} {
			t.Run(name, func(t *testing.T) {
				require.NoError(t, decodeDeliveredDecl(t, nil, inst))
			})
		}
	})
}
