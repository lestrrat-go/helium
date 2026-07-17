package shim_test

import (
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/shim"
	"github.com/stretchr/testify/require"
)

// Declaration and document fragments shared by the tests below.
const (
	declV10  = `<?xml version="1.0"?>`
	rootOnly = `<root/>`
	itemOnly = `<item/>`
)

// unmarshalDoc runs src through [shim.Unmarshal] and returns its verdict.
func unmarshalDoc(src string) error {
	var v struct{}
	return shim.Unmarshal([]byte(src), &v)
}

// decodeDoc drains src through a [shim.Decoder] to EOF and returns the first
// error, so its accept/reject verdict is comparable with unmarshalDoc's.
func decodeDoc(t *testing.T, src string) error {
	t.Helper()
	dec := shim.NewDecoder(t.Context(), strings.NewReader(src))
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

// TestXMLDeclPosition covers where an XML declaration may appear. XMLDecl is
// only legal as the very first thing in a document (leading whitespace aside),
// and both entry points must agree on that.
func TestXMLDeclPosition(t *testing.T) {
	t.Run("declaration not at the start is rejected", func(t *testing.T) {
		for name, src := range map[string]string{
			"after an earlier declaration": declV10 + declV10 + rootOnly,
			"after a comment":              `<!-- c -->` + declV10 + rootOnly,
			"after a processing instruction": `<?xml-stylesheet href="a.xsl"?>` +
				declV10 + rootOnly,
			"after a doctype": `<!DOCTYPE root>` + declV10 + rootOnly,
		} {
			t.Run(name, func(t *testing.T) {
				require.Error(t, unmarshalDoc(src), "Unmarshal must reject")
				require.Error(t, decodeDoc(t, src), "Decoder must reject")
			})
		}
	})

	// Leading whitespace ahead of a declaration is accepted by both entry
	// points: Unmarshal trims it before parsing, and the prolog scanner emits
	// it as CharData without treating it as content that displaces the
	// declaration from the start of the document.
	t.Run("leading whitespace before a declaration is accepted", func(t *testing.T) {
		src := "  \n\t" + declV10 + itemOnly
		require.NoError(t, unmarshalDoc(src))
		require.NoError(t, decodeDoc(t, src))
	})
}

// TestXMLDeclTargetBoundary covers processing instructions whose target merely
// begins with "xml". PITarget ::= Name - (('X'|'x')('M'|'m')('L'|'l')) subtracts
// only the exact name "xml", so "xmlversion" is a legal target and a document
// using one is well-formed. Neither entry point may read such a PI as a
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
			require.NoError(t, unmarshalDoc(src), "Unmarshal must accept a legal PI target")
			require.NoError(t, decodeDoc(t, src), "Decoder must accept a legal PI target")
		})
	}
}

// TestXMLDeclAgreement pins the accept/reject verdict of every declaration
// shape in the shim's bar, through both entry points, which must agree.
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
				require.NoError(t, unmarshalDoc(src), "Unmarshal must accept")
				require.NoError(t, decodeDoc(t, src), "Decoder must accept")
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
				require.Error(t, unmarshalDoc(src), "Unmarshal must reject")
				require.Error(t, decodeDoc(t, src), "Decoder must reject")
			})
		}
	})

	// A declared version outside 1.0 is named as unsupported by both entry
	// points, rather than reported as a bare grammar violation.
	t.Run("unsupported version is named by both entry points", func(t *testing.T) {
		src := `<?xml version="2.0"?>` + itemOnly

		uErr := unmarshalDoc(src)
		require.Error(t, uErr)
		require.Contains(t, uErr.Error(), `unsupported version "2.0"`)

		dErr := decodeDoc(t, src)
		require.Error(t, dErr)
		require.Contains(t, dErr.Error(), `unsupported version "2.0"`)
	})
}
