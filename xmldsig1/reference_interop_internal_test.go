package xmldsig1

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// parseVectorSignature parses a W3C interop vector file, locates its
// ds:Signature, and returns the document and the parsed signature structure.
func parseVectorSignature(t *testing.T, name string) (*helium.Document, *parsedSignature) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "interop", name))
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	sig := findSig(doc.DocumentElement())
	require.NotNil(t, sig, "vector %s must contain a ds:Signature", name)
	parsed, err := parseSignatureElement(sig)
	require.NoError(t, err)
	return doc, parsed
}

// TestReferenceURIForm locks the fail-closed classification of same-document
// Reference URIs. Only the four supported forms resolve; every other URI —
// external references and unrecognized #xpointer(...) schemes — is rejected so
// a verifier never silently digests unintended bytes.
func TestReferenceURIForm(t *testing.T) {
	type want struct {
		id              string
		wholeDoc        bool
		includeComments bool
		ok              bool
	}
	cases := map[string]want{
		"":                        {"", true, false, true},
		"#e1ID":                   {"e1ID", false, false, true},
		"#xpointer(/)":            {"", true, true, true},
		"#xpointer(id('e1ID'))":   {"e1ID", false, true, true},
		`#xpointer(id("e1ID"))`:   {"e1ID", false, true, true},
		"#xpointer( id('e1ID') )": {"e1ID", false, true, true},
		// Fail-closed: external and unsupported schemes.
		"http://example.com/x.xml":  {"", false, false, false},
		"c14n11/xml-base-input.xml": {"", false, false, false},
		"#xpointer(//e1)":           {"", false, false, false},
		"#xpointer(count(//x))":     {"", false, false, false},
		"#xpointer(id(e1ID))":       {"", false, false, false}, // unquoted arg
		"#xpointer(id('a','b'))":    {"", false, false, false}, // second arg
		"#xpointer(id('e1ID')":      {"", false, false, false}, // unbalanced
		"#foo(bar)":                 {"", false, false, false}, // parens, not xpointer
	}
	for uri, exp := range cases {
		t.Run(uri, func(t *testing.T) {
			id, wholeDoc, includeComments, ok := referenceURIForm(uri)
			require.Equal(t, exp.ok, ok, "ok")
			require.Equal(t, exp.wholeDoc, wholeDoc, "wholeDoc")
			require.Equal(t, exp.includeComments, includeComments, "includeComments")
			require.Equal(t, exp.id, id, "id")
		})
	}
}

// TestXPointerReferenceDigests locks the same-document Reference URI forms and
// their comment-node node-set semantics (XMLDSig core §4.3.3.2-3) against the
// W3C "xpointer" interop vectors. For each Reference in each vector the digest
// recomputed over the resolved, transformed, canonicalized node-set MUST equal
// the DigestValue the signer recorded. The vectors cross the four URI forms
// (#xpointer(/), #xpointer(id('X')), "", #id) with C14N 1.1 WithComments: the
// #xpointer forms keep comment nodes; the bare "#id" / "" forms drop them.
func TestXPointerReferenceDigests(t *testing.T) {
	vectors := []string{
		"xpointer-1-SUN.xml", // #xpointer(/), enveloped, WithComments  → comments kept
		"xpointer-2-SUN.xml", // #xpointer(id('e1ID')), WithComments    → comments kept
		"xpointer-3-SUN.xml", // "", enveloped, WithComments            → comments dropped
		"xpointer-4-SUN.xml", // #e1ID, WithComments                    → comments dropped
		"xpointer-5-SUN.xml", // 3x #xpointer(id('X')), WithComments    → comments kept
		"xpointer-6-SUN.xml", // 3x #eID, WithComments                  → comments dropped
	}
	for _, name := range vectors {
		t.Run(name, func(t *testing.T) {
			doc, parsed := parseVectorSignature(t, name)
			require.NotEmpty(t, parsed.references)
			for i, ref := range parsed.references {
				_, canonical, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, findSig(doc.DocumentElement()), ref)
				require.NoErrorf(t, err, "reference %d (%q) canonicalization", i, ref.uri)
				// allowSHA1 is true here: these interop vectors predate the
				// SHA-1 deprecation, and the test asserts the digest bytes, not
				// the policy gate.
				computed, err := computeDigest(ref.digestAlgorithm, canonical, true)
				require.NoErrorf(t, err, "reference %d (%q) digest", i, ref.uri)
				require.Truef(t, digestEqual(computed, ref.digestValue),
					"reference %d (%q): digest mismatch\n got %s\nwant %s", i, ref.uri,
					base64.StdEncoding.EncodeToString(computed),
					base64.StdEncoding.EncodeToString(ref.digestValue))
			}
		})
	}
}
