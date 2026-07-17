package shim_test

import (
	stdxml "encoding/xml"
	"io"
	"strings"
	"testing"

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
// only legal as the very first thing in a document (leading whitespace aside),
// and every entry point must agree on that.
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

	// Leading whitespace ahead of a declaration is accepted by every entry
	// point: Unmarshal trims it before parsing, the prolog scanner emits it as
	// CharData without treating it as content that displaces the declaration
	// from the start of the document, and a TokenReader's leading whitespace
	// CharData is held to the same rule.
	t.Run("leading whitespace before a declaration is accepted", func(t *testing.T) {
		requireAllAccept(t, "  \n\t"+declV10+itemOnly)
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
		require.Contains(t, uErr.Error(), `unsupported version "2.0"`)

		dErr := decodeDoc(t, src)
		require.Error(t, dErr)
		require.Contains(t, dErr.Error(), `unsupported version "2.0"`)

		requireAllReject(t, src)
	})
}
