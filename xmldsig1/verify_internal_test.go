package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/stretchr/testify/require"
)

const dsigNS = NamespaceDSig

// findChild returns the first child element of parent with the given local
// name, failing the test if none is found.
func findChild(t *testing.T, parent *helium.Element, name string) *helium.Element {
	t.Helper()
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		if domutil.LocalName(e) == name {
			return e
		}
	}
	t.Fatalf("child %q not found", name)
	return nil
}

func countChildren(parent *helium.Element, name string) int {
	n := 0
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		if domutil.LocalName(e) == name {
			n++
		}
	}
	return n
}

// findFirstElement returns the first descendant Element matching localName.
func findFirstElement(t *testing.T, n helium.Node, name string) *helium.Element {
	t.Helper()
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if e, ok := helium.AsNode[*helium.Element](child); ok {
			if domutil.LocalName(e) == name {
				return e
			}
			if found := findFirstElement(t, e, name); found != nil {
				return found
			}
		}
	}
	return nil
}

// wrapText interleaves XML whitespace (CR, LF, tab, space) into s. Go's base64
// decoder skips CR/LF but rejects space and tab, so the space/tab variants are
// what exercise the strip-before-decode fix.
func wrapText(s string, col int) string {
	seps := []string{"\n", "  \t", "\r\n\t", " "}
	var out []byte
	sepIdx := 0
	for i := range len(s) {
		if i > 0 && i%col == 0 {
			out = append(out, seps[sepIdx%len(seps)]...)
			sepIdx++
		}
		out = append(out, s[i])
	}
	return string(out)
}

// setText removes existing text children and sets a single text node.
func setText(t *testing.T, e *helium.Element, text string) {
	t.Helper()
	for child := e.FirstChild(); child != nil; {
		next := child.NextSibling()
		if mn, ok := child.(helium.MutableNode); ok {
			helium.UnlinkNode(mn)
		}
		child = next
	}
	doc := e.OwnerDocument()
	require.NoError(t, e.AddChild(doc.CreateText([]byte(text))))
}

func TestDigestEqual(t *testing.T) {
	require.True(t, digestEqual([]byte{1, 2, 3}, []byte{1, 2, 3}))
	require.False(t, digestEqual([]byte{1, 2, 3}, []byte{1, 2}))    // length mismatch
	require.False(t, digestEqual([]byte{1, 2, 3}, []byte{1, 2, 4})) // content mismatch
}

func TestParseSignatureElement(t *testing.T) {
	t.Run("missing SignedInfo", func(t *testing.T) {
		doc := mustParse(t, `<ds:Signature xmlns:ds="`+dsigNS+`"><ds:SignatureValue xmlns:ds="`+dsigNS+`">AA==</ds:SignatureValue></ds:Signature>`)
		_, err := parseSignatureElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "missing SignedInfo")
	})

	t.Run("missing SignatureValue", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgRSASHA256 + `"/>` +
			`<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference></ds:SignedInfo>`
		doc := mustParse(t, `<ds:Signature xmlns:ds="`+dsigNS+`">`+si+`</ds:Signature>`)
		_, err := parseSignatureElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "missing SignatureValue")
	})

	t.Run("bad SignatureValue base64", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgRSASHA256 + `"/>` +
			`<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference></ds:SignedInfo>`
		doc := mustParse(t, `<ds:Signature xmlns:ds="`+dsigNS+`">`+si+
			`<ds:SignatureValue xmlns:ds="`+dsigNS+`">!!!bad</ds:SignatureValue></ds:Signature>`)
		_, err := parseSignatureElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
	})
}

func TestParseSignedInfo(t *testing.T) {
	t.Run("missing C14N algorithm", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `"/>` +
			`</ds:SignedInfo>`
		doc := mustParse(t, si)
		var parsed parsedSignature
		err := parseSignedInfo(doc.DocumentElement(), &parsed)
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "CanonicalizationMethod missing Algorithm")
	})

	t.Run("missing SignatureMethod algorithm", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `"/>` +
			`</ds:SignedInfo>`
		doc := mustParse(t, si)
		var parsed parsedSignature
		err := parseSignedInfo(doc.DocumentElement(), &parsed)
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "SignatureMethod missing Algorithm")
	})

	// A SignedInfo with no CanonicalizationMethod is structurally invalid and
	// must be rejected up front rather than parsing OK and failing later as an
	// unsupported-algorithm error during canonicalization.
	t.Run("missing CanonicalizationMethod", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgRSASHA256 + `"/>` +
			`<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference></ds:SignedInfo>`
		doc := mustParse(t, si)
		var parsed parsedSignature
		err := parseSignedInfo(doc.DocumentElement(), &parsed)
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "missing CanonicalizationMethod")
	})

	// A SignedInfo with no SignatureMethod is structurally invalid and must be
	// rejected up front rather than parsing OK and failing later (possibly only
	// after key resolution) as an unsupported-algorithm error.
	t.Run("missing SignatureMethod", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
			`<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference></ds:SignedInfo>`
		doc := mustParse(t, si)
		var parsed parsedSignature
		err := parseSignedInfo(doc.DocumentElement(), &parsed)
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "missing SignatureMethod")
	})

	t.Run("no Reference", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgRSASHA256 + `"/>` +
			`</ds:SignedInfo>`
		doc := mustParse(t, si)
		var parsed parsedSignature
		err := parseSignedInfo(doc.DocumentElement(), &parsed)
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "no Reference")
	})

	// A second CanonicalizationMethod makes the c14n algorithm ambiguous; the
	// schema fixes its cardinality at one, so it must be rejected rather than
	// accepted last-one-wins.
	t.Run("duplicate CanonicalizationMethod", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + C14N10 + `"/>` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgRSASHA256 + `"/>` +
			`<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference></ds:SignedInfo>`
		doc := mustParse(t, si)
		var parsed parsedSignature
		err := parseSignedInfo(doc.DocumentElement(), &parsed)
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "multiple CanonicalizationMethod")
	})

	// A second SignatureMethod makes the signature algorithm ambiguous.
	t.Run("duplicate SignatureMethod", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgRSASHA256 + `"/>` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgECDSASHA256 + `"/>` +
			`<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference></ds:SignedInfo>`
		doc := mustParse(t, si)
		var parsed parsedSignature
		err := parseSignedInfo(doc.DocumentElement(), &parsed)
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "multiple SignatureMethod")
	})
}

func TestParseReferenceElement(t *testing.T) {
	t.Run("missing DigestMethod algorithm", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `"/>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "DigestMethod missing Algorithm")
	})

	t.Run("bad DigestValue base64", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">!!!bad</ds:DigestValue>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
	})

	// A Reference with no DigestMethod is structurally invalid and must be
	// rejected up front rather than parsing OK and failing later as an
	// unsupported-digest error.
	t.Run("missing DigestMethod", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "missing DigestMethod")
	})

	// A Reference with no DigestValue is structurally invalid and must be
	// rejected up front rather than parsing OK and failing later as a digest
	// mismatch (the empty digest never matches the recomputed one).
	t.Run("missing DigestValue", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "missing DigestValue")
	})

	t.Run("with InclusiveNamespaces", func(t *testing.T) {
		const exc = ExcC14N10
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:Transforms xmlns:ds="` + dsigNS + `">` +
			`<ds:Transform xmlns:ds="` + dsigNS + `" Algorithm="` + exc + `">` +
			`<ec:InclusiveNamespaces xmlns:ec="` + exc + `" PrefixList="a b"/>` +
			`</ds:Transform></ds:Transforms>` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		ref, err := parseReferenceElement(doc.DocumentElement())
		require.NoError(t, err)
		require.Len(t, ref.transforms, 1)
		require.Equal(t, []string{"a", "b"}, ref.transforms[0].prefixes)
	})

	// ec:InclusiveNamespaces is an Exclusive C14N parameter; under a
	// non-exclusive Reference Transform (C14N10/C14N11/enveloped-signature) the
	// prefixes are silently dropped during canonicalization. Accepting it would
	// be fail-open, so it must be rejected — even an empty PrefixList.
	t.Run("InclusiveNamespaces on non-exclusive transform rejected", func(t *testing.T) {
		for _, alg := range []string{C14N10, C14N11URI, TransformEnvelopedSignature} {
			t.Run(alg, func(t *testing.T) {
				r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
					`<ds:Transforms xmlns:ds="` + dsigNS + `">` +
					`<ds:Transform xmlns:ds="` + dsigNS + `" Algorithm="` + alg + `">` +
					`<ec:InclusiveNamespaces xmlns:ec="` + ExcC14N10 + `" PrefixList="a b"/>` +
					`</ds:Transform></ds:Transforms>` +
					`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
					`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
					`</ds:Reference>`
				doc := mustParse(t, r)
				_, err := parseReferenceElement(doc.DocumentElement())
				require.ErrorIs(t, err, ErrUnsupportedTransform)
				require.Contains(t, err.Error(), "ec:InclusiveNamespaces")
			})
		}
	})

	// An empty PrefixList is still misplaced under a non-exclusive transform; the
	// boolean-tracked match must reject it rather than treat "no prefixes" as
	// harmless.
	t.Run("empty InclusiveNamespaces on non-exclusive transform rejected", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:Transforms xmlns:ds="` + dsigNS + `">` +
			`<ds:Transform xmlns:ds="` + dsigNS + `" Algorithm="` + C14N10 + `">` +
			`<ec:InclusiveNamespaces xmlns:ec="` + ExcC14N10 + `"/>` +
			`</ds:Transform></ds:Transforms>` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	// Accept arm: ec:InclusiveNamespaces on an exclusive c14n transform is valid
	// and its PrefixList is honored.
	t.Run("InclusiveNamespaces on exclusive transform accepted", func(t *testing.T) {
		for _, alg := range []string{ExcC14N10, ExcC14N10Comments} {
			t.Run(alg, func(t *testing.T) {
				r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
					`<ds:Transforms xmlns:ds="` + dsigNS + `">` +
					`<ds:Transform xmlns:ds="` + dsigNS + `" Algorithm="` + alg + `">` +
					`<ec:InclusiveNamespaces xmlns:ec="` + ExcC14N10 + `" PrefixList="a b"/>` +
					`</ds:Transform></ds:Transforms>` +
					`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
					`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
					`</ds:Reference>`
				doc := mustParse(t, r)
				ref, err := parseReferenceElement(doc.DocumentElement())
				require.NoError(t, err)
				require.Len(t, ref.transforms, 1)
				require.Equal(t, []string{"a", "b"}, ref.transforms[0].prefixes)
			})
		}
	})

	// An unknown child element under a Transform is an algorithm parameter we
	// cannot honor. Digesting as if it were absent would be fail-open, so any
	// unrecognized Transform child must be rejected — both under an exclusive
	// c14n transform (where ec:InclusiveNamespaces is the only valid child) and
	// under a non-exclusive one (where no child is valid at all).
	t.Run("unknown Transform child rejected", func(t *testing.T) {
		for _, alg := range []string{ExcC14N10, C14N10} {
			t.Run(alg, func(t *testing.T) {
				r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
					`<ds:Transforms xmlns:ds="` + dsigNS + `">` +
					`<ds:Transform xmlns:ds="` + dsigNS + `" Algorithm="` + alg + `">` +
					`<ds:XPath xmlns:ds="` + dsigNS + `">/foo</ds:XPath>` +
					`</ds:Transform></ds:Transforms>` +
					`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
					`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
					`</ds:Reference>`
				doc := mustParse(t, r)
				_, err := parseReferenceElement(doc.DocumentElement())
				require.ErrorIs(t, err, ErrUnsupportedTransform)
				require.Contains(t, err.Error(), "Transform parameter")
			})
		}
	})

	// A second ec:InclusiveNamespaces under an exclusive Transform must be
	// rejected rather than silently letting the last one win.
	t.Run("multiple InclusiveNamespaces rejected", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:Transforms xmlns:ds="` + dsigNS + `">` +
			`<ds:Transform xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `">` +
			`<ec:InclusiveNamespaces xmlns:ec="` + ExcC14N10 + `" PrefixList="a"/>` +
			`<ec:InclusiveNamespaces xmlns:ec="` + ExcC14N10 + `" PrefixList="b"/>` +
			`</ds:Transform></ds:Transforms>` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Contains(t, err.Error(), "multiple ec:InclusiveNamespaces")
	})

	// A second DigestMethod makes the digest algorithm ambiguous.
	t.Run("duplicate DigestMethod", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA512 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "multiple DigestMethod")
	})

	// Two DigestValue children (the core of DSIG-004): even when the second
	// matches the recomputed digest, the ambiguous schema-invalid Reference
	// must be rejected rather than accepted last-one-wins.
	t.Run("duplicate DigestValue", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">BB==</ds:DigestValue>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "multiple DigestValue")
	})

	// At most one Transforms element is permitted; two would let extra
	// transforms accumulate and silently change the canonical input.
	t.Run("duplicate Transforms", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:Transforms xmlns:ds="` + dsigNS + `">` +
			`<ds:Transform xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
			`</ds:Transforms>` +
			`<ds:Transforms xmlns:ds="` + dsigNS + `">` +
			`<ds:Transform xmlns:ds="` + dsigNS + `" Algorithm="` + C14N10 + `"/>` +
			`</ds:Transforms>` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "multiple Transforms")
	})
}

// TestEmptyReferencesRejected guards against a SignedInfo that carries zero
// Reference children. XML-Signature requires at least one Reference; a
// SignatureValue computed over a reference-free SignedInfo cryptographically
// verifies yet covers no document content, so the signature attests to
// nothing. Verify must reject such a structure rather than report success.
//
// The attack is constructed with full key control (an attacker controlling the
// signing key is the worst case): produce a genuine signature, strip the
// Reference element, then recompute a valid SignatureValue over the now-empty
// SignedInfo. The resulting document is a perfectly valid empty-reference
// signature.
func TestEmptyReferencesRejected(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
	require.NoError(t, err)

	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             payloadFragment,
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{ExcC14NTransform()},
		}).
		SignDetached(context.Background(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	// Locate SignedInfo and remove its single Reference child so the
	// SignedInfo now has zero references.
	signedInfo := findChild(t, sigElem, "SignedInfo")
	ref := findChild(t, signedInfo, "Reference")
	helium.UnlinkNode(ref)

	// Recompute a valid SignatureValue over the reference-free SignedInfo.
	canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
	require.NoError(t, err)
	sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
	require.NoError(t, err)

	sigValueElem := findChild(t, sigElem, "SignatureValue")
	for c := sigValueElem.FirstChild(); c != nil; c = sigValueElem.FirstChild() {
		mc, ok := c.(helium.MutableNode)
		require.True(t, ok)
		helium.UnlinkNode(mc)
	}
	require.NoError(t, sigValueElem.AddChild(
		doc.CreateText([]byte(base64.StdEncoding.EncodeToString(sigBytes)))))

	// Sanity: the signature value itself is cryptographically valid over the
	// empty SignedInfo, so the only thing standing between this document and a
	// false "verified" result is the empty-reference check.
	require.Equal(t, 0, countChildren(signedInfo, "Reference"))

	verifier := NewVerifier(StaticKey(&key.PublicKey))
	_, err = verifier.Verify(context.Background(), doc)
	require.Error(t, err, "signature with zero references must be rejected")
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.True(t, strings.Contains(err.Error(), "Reference"),
		"error should mention the missing Reference: %v", err)
}

// TestDuplicateDigestValueRejected is the end-to-end form of DSIG-004: a
// SignedInfo whose Reference carries two DigestValue children must be rejected
// even when the document is otherwise a perfectly valid signature. The attack
// is built with full key control: sign normally, append a second (matching)
// DigestValue to the Reference, then recompute a valid SignatureValue over the
// now-ambiguous SignedInfo. Last-one-wins parsing would accept it; a conforming
// verifier must reject the schema-invalid duplicate.
func TestDuplicateDigestValueRejected(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
	require.NoError(t, err)

	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             payloadFragment,
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{ExcC14NTransform()},
		}).
		SignDetached(context.Background(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	// Sanity: the freshly produced signature verifies before tampering.
	_, err = NewVerifier(StaticKey(&key.PublicKey)).Verify(context.Background(), doc)
	require.NoError(t, err)

	signedInfo := findChild(t, sigElem, "SignedInfo")
	reference := findChild(t, signedInfo, "Reference")
	digestValue := findChild(t, reference, "DigestValue")

	// Append an identical second DigestValue so the Reference now has two.
	dup, err := helium.CopyNode(digestValue, doc)
	require.NoError(t, err)
	dupElem, ok := helium.AsNode[*helium.Element](dup)
	require.True(t, ok)
	require.NoError(t, reference.AddChild(dupElem))
	require.Equal(t, 2, countChildren(reference, "DigestValue"))

	// Recompute a valid SignatureValue over the now-ambiguous SignedInfo.
	canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
	require.NoError(t, err)
	sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
	require.NoError(t, err)

	sigValueElem := findChild(t, sigElem, "SignatureValue")
	setText(t, sigValueElem, base64.StdEncoding.EncodeToString(sigBytes))

	_, err = NewVerifier(StaticKey(&key.PublicKey)).Verify(context.Background(), doc)
	require.Error(t, err, "signature with duplicate DigestValue must be rejected")
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.Contains(t, err.Error(), "multiple DigestValue")
}

// TestVerifyLineWrappedDigestValue ensures a DigestValue carrying embedded
// whitespace (line-wrapped base64) still verifies. Because DigestValue lives
// inside SignedInfo, the wrapping must be present when SignedInfo is signed —
// c14n preserves the whitespace in the signed bytes, and the verifier must
// strip it before base64-decoding to recompute the reference digest.
func TestVerifyLineWrappedDigestValue(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	const samlAssertion = `<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_abc123" IssueInstant="2024-01-01T00:00:00Z" Version="2.0"><saml:Issuer>https://idp.example.com</saml:Issuer></saml:Assertion>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(samlAssertion))
	require.NoError(t, err)

	signer := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	sigElem := findFirstElement(t, doc, "Signature")
	require.NotNil(t, sigElem)
	signedInfo := findFirstElement(t, sigElem, "SignedInfo")
	require.NotNil(t, signedInfo)
	digestValue := findFirstElement(t, signedInfo, "DigestValue")
	require.NotNil(t, digestValue)
	sigValue := findFirstElement(t, sigElem, "SignatureValue")
	require.NotNil(t, sigValue)

	// Line-wrap the DigestValue text in place, then re-sign SignedInfo so the
	// SignatureValue covers the wrapped DigestValue.
	wrapped := wrapText(domutil.TextContent(digestValue), 16)
	require.Contains(t, wrapped, " ")
	setText(t, digestValue, wrapped)

	canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
	require.NoError(t, err)
	sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
	require.NoError(t, err)
	setText(t, sigValue, base64.StdEncoding.EncodeToString(sigBytes))

	verifier := NewVerifier(StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

// TestReferenceNamespace guards the namespace-confusion bypasses of the
// at-least-one-Reference rule: only core ds:Reference elements count.
func TestReferenceNamespace(t *testing.T) {
	// foreign namespace rejected guards against a namespace-confusion bypass of
	// the empty-reference check. The "at least one Reference" rule must count only
	// ds:Reference elements in the XML-Signature namespace. If parseSignedInfo
	// matched on local name alone, a SignedInfo carrying zero genuine ds:Reference
	// children plus a single foreign-namespace <evil:Reference> would satisfy
	// len(references) > 0 and verify against a recomputed SignatureValue while
	// covering no document content — re-opening the no-content-signature bypass.
	t.Run("foreign namespace rejected", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
		require.NoError(t, err)

		sigElem, err := NewSigner().
			SignatureAlgorithm(AlgRSASHA256).
			Reference(ReferenceConfig{
				URI:             payloadFragment,
				DigestAlgorithm: DigestSHA256,
				Transforms:      []Transform{ExcC14NTransform()},
			}).
			SignDetached(context.Background(), doc, key)
		require.NoError(t, err)
		require.NoError(t, doc.DocumentElement().AddChild(sigElem))

		signedInfo := findChild(t, sigElem, "SignedInfo")
		ref := findChild(t, signedInfo, "Reference")

		// Move the genuine ds:Reference into a foreign namespace so that, by local
		// name alone, the SignedInfo still appears to contain a Reference while it
		// no longer carries any ds:Reference.
		const evilNS = "urn:example:evil"
		require.NoError(t, ref.DeclareNamespace("evil", evilNS))
		require.NoError(t, ref.SetActiveNamespace("evil", evilNS))
		require.Equal(t, evilNS, elementNamespaceURI(ref))

		// Recompute a valid SignatureValue over the mutated SignedInfo so the only
		// thing standing between this document and a false "verified" result is the
		// namespace check on the Reference count.
		canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
		require.NoError(t, err)
		sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
		require.NoError(t, err)

		sigValueElem := findChild(t, sigElem, "SignatureValue")
		for c := sigValueElem.FirstChild(); c != nil; c = sigValueElem.FirstChild() {
			mc, ok := c.(helium.MutableNode)
			require.True(t, ok)
			helium.UnlinkNode(mc)
		}
		require.NoError(t, sigValueElem.AddChild(
			doc.CreateText([]byte(base64.StdEncoding.EncodeToString(sigBytes)))))

		verifier := NewVerifier(StaticKey(&key.PublicKey))
		_, err = verifier.Verify(context.Background(), doc)
		require.Error(t, err, "signature whose only Reference is in a foreign namespace must be rejected")
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.True(t, strings.Contains(err.Error(), "Reference"),
			"error should mention the missing Reference: %v", err)
	})

	// dsig11 namespace rejected guards the core/1.1 namespace split. The
	// XML-Signature 1.1 namespace (http://www.w3.org/2009/xmldsig11#) is only for
	// new 1.1-specific elements (e.g. ECKeyValue), NOT an alternate spelling of the
	// core Reference. A dsig11:Reference therefore must not count toward the
	// at-least-one-Reference rule, so mutating the only ds:Reference into the 1.1
	// namespace must make verification fail just like a foreign-namespace Reference.
	t.Run("dsig11 namespace rejected", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
		require.NoError(t, err)

		sigElem, err := NewSigner().
			SignatureAlgorithm(AlgRSASHA256).
			Reference(ReferenceConfig{
				URI:             payloadFragment,
				DigestAlgorithm: DigestSHA256,
				Transforms:      []Transform{ExcC14NTransform()},
			}).
			SignDetached(context.Background(), doc, key)
		require.NoError(t, err)
		require.NoError(t, doc.DocumentElement().AddChild(sigElem))

		signedInfo := findChild(t, sigElem, "SignedInfo")
		ref := findChild(t, signedInfo, "Reference")

		// Move the genuine ds:Reference into the XML-Signature 1.1 namespace. By
		// local name alone the SignedInfo still appears to contain a Reference, but
		// xmldsig11# is not the core namespace, so it must not be accepted as one.
		require.NoError(t, ref.DeclareNamespace("dsig11", NamespaceDSig11))
		require.NoError(t, ref.SetActiveNamespace("dsig11", NamespaceDSig11))
		require.Equal(t, NamespaceDSig11, elementNamespaceURI(ref))

		// Recompute a valid SignatureValue over the mutated SignedInfo so the only
		// thing standing between this document and a false "verified" result is the
		// core-namespace check on the Reference count.
		canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
		require.NoError(t, err)
		sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
		require.NoError(t, err)

		sigValueElem := findChild(t, sigElem, "SignatureValue")
		for c := sigValueElem.FirstChild(); c != nil; c = sigValueElem.FirstChild() {
			mc, ok := c.(helium.MutableNode)
			require.True(t, ok)
			helium.UnlinkNode(mc)
		}
		require.NoError(t, sigValueElem.AddChild(
			doc.CreateText([]byte(base64.StdEncoding.EncodeToString(sigBytes)))))

		verifier := NewVerifier(StaticKey(&key.PublicKey))
		_, err = verifier.Verify(context.Background(), doc)
		require.Error(t, err, "signature whose only Reference is in the xmldsig11# namespace must be rejected")
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.True(t, strings.Contains(err.Error(), "Reference"),
			"error should mention the missing Reference: %v", err)
	})

	// genuine reference still verifies is the positive control: a normal
	// ds:Reference signature must continue to verify after the namespace guard is
	// added, so the fix rejects only foreign-namespace References.
	t.Run("genuine reference still verifies", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
		require.NoError(t, err)

		sigElem, err := NewSigner().
			SignatureAlgorithm(AlgRSASHA256).
			Reference(ReferenceConfig{
				URI:             payloadFragment,
				DigestAlgorithm: DigestSHA256,
				Transforms:      []Transform{ExcC14NTransform()},
			}).
			SignDetached(context.Background(), doc, key)
		require.NoError(t, err)
		require.NoError(t, doc.DocumentElement().AddChild(sigElem))

		verifier := NewVerifier(StaticKey(&key.PublicKey))
		_, err = verifier.Verify(context.Background(), doc)
		require.NoError(t, err, "a genuine ds:Reference signature must still verify")
	})
}

// TestCloneNilConfig covers the cfg == nil branches of the builder clone paths.
func TestCloneNilConfig(t *testing.T) {
	// signer covers the s.cfg == nil branch of Signer.clone by invoking a builder
	// method on a zero-value Signer.
	t.Run("signer", func(t *testing.T) {
		var s Signer // zero value: cfg is nil
		s2 := s.SignatureAlgorithm(AlgRSASHA256)
		require.NotNil(t, s2.cfg)
		require.Equal(t, AlgRSASHA256, s2.cfg.signatureAlgorithm)
		require.Equal(t, ExcC14N10, s2.cfg.c14nMethod)
	})

	// verifier covers the v.cfg == nil branch of Verifier.clone.
	t.Run("verifier", func(t *testing.T) {
		var v Verifier // zero value: cfg is nil
		v2 := v.AllowSHA1(true)
		require.NotNil(t, v2.cfg)
		require.True(t, v2.cfg.allowSHA1)
	})
}

// reSignSignedInfo recomputes a valid SignatureValue over the (possibly
// tampered) SignedInfo using Exclusive C14N (the signer's SignedInfo default)
// and the given inclusive prefixes, replacing the existing SignatureValue text.
// Tests use this after mutating SignedInfo so that verification reaches the
// per-Reference transform handling rather than failing earlier on the
// SignatureValue check.
func reSignSignedInfo(t *testing.T, doc *helium.Document, sigElem, signedInfo *helium.Element, prefixes []string, key *rsa.PrivateKey) {
	t.Helper()
	canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, prefixes)
	require.NoError(t, err)
	sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
	require.NoError(t, err)

	sigValueElem := findChild(t, sigElem, "SignatureValue")
	for c := sigValueElem.FirstChild(); c != nil; c = sigValueElem.FirstChild() {
		mc, ok := c.(helium.MutableNode)
		require.True(t, ok)
		helium.UnlinkNode(mc)
	}
	require.NoError(t, sigValueElem.AddChild(
		doc.CreateText([]byte(base64.StdEncoding.EncodeToString(sigBytes)))))
}

// TestVerifyUnsupportedTransform drives verifyReference's fail-closed rejection
// of an unsupported transform URI. The signer now refuses to emit an unknown
// transform, so the malicious SignedInfo is built by signing a valid
// Enveloped+ExcC14N reference, rewriting the c14n Transform's Algorithm to an
// unsupported URI, and recomputing a valid SignatureValue over the tampered
// SignedInfo. The verifier must reject the unknown transform before digesting.
func TestVerifyUnsupportedTransform(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
	require.NoError(t, err)

	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             payloadFragment,
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{Enveloped(), ExcC14NTransform()},
		}).
		SignDetached(context.Background(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	// Rewrite the Exclusive C14N Transform to an unsupported transform URI,
	// leaving the pipeline as [enveloped, xpath].
	signedInfo := findChild(t, sigElem, "SignedInfo")
	excTransform := findTransformByAlgorithm(t, signedInfo, ExcC14N10)
	require.NoError(t, excTransform.SetAttribute("Algorithm", TransformXPath))

	reSignSignedInfo(t, doc, sigElem, signedInfo, nil, key)

	_, err = NewVerifier(StaticKey(&key.PublicKey)).Verify(context.Background(), doc)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
}

// TestVerifyTransformAfterCanonicalization guards DSIG-001's ordered-pipeline
// rule: an octet-producing c14n transform ends the pipeline, so any transform
// ordered after it (here a second, enveloped transform) cannot be applied and
// the Reference must be rejected fail-closed.
func TestVerifyTransformAfterCanonicalization(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
	require.NoError(t, err)

	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             payloadFragment,
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{ExcC14NTransform()},
		}).
		SignDetached(context.Background(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	// Append an enveloped-signature Transform AFTER the c14n transform.
	signedInfo := findChild(t, sigElem, "SignedInfo")
	refElem := findChild(t, signedInfo, "Reference")
	transforms := findChild(t, refElem, "Transforms")
	extra, err := doc.CreateElement("Transform")
	require.NoError(t, err)
	require.NoError(t, extra.SetActiveNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, extra.SetAttribute("Algorithm", TransformEnvelopedSignature))
	require.NoError(t, transforms.AddChild(extra))

	reSignSignedInfo(t, doc, sigElem, signedInfo, nil, key)

	_, err = NewVerifier(StaticKey(&key.PublicKey)).Verify(context.Background(), doc)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	require.Contains(t, err.Error(), "ordered after canonicalization")
}

// TestVerifyOmittedTransformDefaultsToC14N10 guards DSIG-001's default: a
// Reference with no transform must convert the node-set to octets with inclusive
// Canonical XML 1.0, NOT Exclusive C14N. The document carries an unused ancestor
// namespace, which inclusive C14N carries into the #payload subtree but
// Exclusive C14N drops, so the digest matches only under the correct default.
func TestVerifyOmittedTransformDefaultsToC14N10(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(),
		[]byte(`<doc xmlns:unused="urn:unused"><data Id="payload"><child>v</child></data></doc>`))
	require.NoError(t, err)

	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             payloadFragment,
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{C14NTransform(C14N10)},
		}).
		SignDetached(context.Background(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	// Drop the Transforms element so the Reference declares no transform; its
	// DigestValue was computed with inclusive Canonical XML 1.0.
	signedInfo := findChild(t, sigElem, "SignedInfo")
	refElem := findChild(t, signedInfo, "Reference")
	helium.UnlinkNode(findChild(t, refElem, "Transforms"))

	reSignSignedInfo(t, doc, sigElem, signedInfo, nil, key)

	_, err = NewVerifier(StaticKey(&key.PublicKey)).Verify(context.Background(), doc)
	require.NoError(t, err, "omitted transform must default to inclusive Canonical XML 1.0")
}

// TestVerifySignedInfoInclusiveNamespaces guards DSIG-002: an
// ec:InclusiveNamespaces PrefixList declared on SignedInfo's
// CanonicalizationMethod must be honored when canonicalizing SignedInfo. The
// root declares an unused namespace prefix; Exclusive C14N drops it unless the
// PrefixList forces its inclusion, so the SignatureValue (computed WITH the
// prefix list) verifies only when the verifier threads that PrefixList through.
func TestVerifySignedInfoInclusiveNamespaces(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(),
		[]byte(`<doc xmlns:extra="urn:extra"><data Id="payload">v</data></doc>`))
	require.NoError(t, err)

	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             payloadFragment,
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{ExcC14NTransform()},
		}).
		SignDetached(context.Background(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	// Declare an ec:InclusiveNamespaces PrefixList on the CanonicalizationMethod.
	signedInfo := findChild(t, sigElem, "SignedInfo")
	c14nMethodElem := findChild(t, signedInfo, "CanonicalizationMethod")
	inc, err := doc.CreateElement("InclusiveNamespaces")
	require.NoError(t, err)
	require.NoError(t, inc.DeclareNamespace("ec", ExcC14N10))
	require.NoError(t, inc.SetActiveNamespace("ec", ExcC14N10))
	require.NoError(t, inc.SetAttribute("PrefixList", "extra"))
	require.NoError(t, c14nMethodElem.AddChild(inc))

	// Recompute the SignatureValue using the declared exclusive c14n + PrefixList.
	reSignSignedInfo(t, doc, sigElem, signedInfo, []string{"extra"}, key)

	_, err = NewVerifier(StaticKey(&key.PublicKey)).Verify(context.Background(), doc)
	require.NoError(t, err, "SignedInfo ec:InclusiveNamespaces PrefixList must be honored")
}

// TestVerifyRejectsSignatureMethodParameter guards DSIG-003: any child parameter
// of SignatureMethod (e.g. ds:HMACOutputLength, which requests a truncated HMAC)
// is unsupported and must be rejected fail-closed rather than silently ignored.
func TestVerifyRejectsSignatureMethodParameter(t *testing.T) {
	si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
		`<ds:CanonicalizationMethod Algorithm="` + ExcC14N10 + `"/>` +
		`<ds:SignatureMethod Algorithm="` + AlgHMACSHA256 + `">` +
		`<ds:HMACOutputLength>128</ds:HMACOutputLength>` +
		`</ds:SignatureMethod>` +
		`<ds:Reference URI="">` +
		`<ds:DigestMethod Algorithm="` + DigestSHA256 + `"/>` +
		`<ds:DigestValue>AA==</ds:DigestValue>` +
		`</ds:Reference>` +
		`</ds:SignedInfo>`
	doc := mustParse(t, si)
	var parsed parsedSignature
	err := parseSignedInfo(doc.DocumentElement(), &parsed)
	require.ErrorIs(t, err, ErrUnsupportedAlgorithm)
	require.Contains(t, err.Error(), "SignatureMethod parameter")
}

// TestVerifyRejectsCanonicalizationMethodParameter guards DSIG-002's
// fail-closed arm: an unrecognized child parameter of CanonicalizationMethod
// must be rejected rather than silently ignored.
func TestVerifyRejectsCanonicalizationMethodParameter(t *testing.T) {
	si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
		`<ds:CanonicalizationMethod Algorithm="` + ExcC14N10 + `">` +
		`<ds:BogusParameter/>` +
		`</ds:CanonicalizationMethod>` +
		`<ds:SignatureMethod Algorithm="` + AlgRSASHA256 + `"/>` +
		`<ds:Reference URI="">` +
		`<ds:DigestMethod Algorithm="` + DigestSHA256 + `"/>` +
		`<ds:DigestValue>AA==</ds:DigestValue>` +
		`</ds:Reference>` +
		`</ds:SignedInfo>`
	doc := mustParse(t, si)
	var parsed parsedSignature
	err := parseSignedInfo(doc.DocumentElement(), &parsed)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	require.Contains(t, err.Error(), "CanonicalizationMethod parameter")
}

// TestVerifyRejectsInclusiveNamespacesOnNonExclusiveC14N guards the fail-closed
// parameter handling: ec:InclusiveNamespaces is an Exclusive XML Canonicalization
// parameter and canonicalize() only honors its PrefixList for exclusive modes. A
// non-exclusive CanonicalizationMethod (C14N 1.0 / C14N 1.1) declaring an
// ec:InclusiveNamespaces parameter would have it silently ignored, so it must be
// rejected rather than accepted.
func TestVerifyRejectsInclusiveNamespacesOnNonExclusiveC14N(t *testing.T) {
	for _, alg := range []string{C14N10, C14N10Comments, C14N11URI, C14N11Comments} {
		t.Run(alg, func(t *testing.T) {
			si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
				`<ds:CanonicalizationMethod Algorithm="` + alg + `">` +
				`<ec:InclusiveNamespaces xmlns:ec="` + ExcC14N10 + `" PrefixList="extra"/>` +
				`</ds:CanonicalizationMethod>` +
				`<ds:SignatureMethod Algorithm="` + AlgRSASHA256 + `"/>` +
				`<ds:Reference URI="">` +
				`<ds:DigestMethod Algorithm="` + DigestSHA256 + `"/>` +
				`<ds:DigestValue>AA==</ds:DigestValue>` +
				`</ds:Reference>` +
				`</ds:SignedInfo>`
			doc := mustParse(t, si)
			var parsed parsedSignature
			err := parseSignedInfo(doc.DocumentElement(), &parsed)
			require.ErrorIs(t, err, ErrUnsupportedTransform)
			require.Contains(t, err.Error(), "ec:InclusiveNamespaces")
		})
	}
}

// TestVerifyAcceptsInclusiveNamespacesOnExclusiveC14N is the accept arm: an
// ec:InclusiveNamespaces PrefixList declared on an exclusive CanonicalizationMethod
// is honored and its prefixes threaded through to canonicalization.
func TestVerifyAcceptsInclusiveNamespacesOnExclusiveC14N(t *testing.T) {
	for _, alg := range []string{ExcC14N10, ExcC14N10Comments} {
		t.Run(alg, func(t *testing.T) {
			si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
				`<ds:CanonicalizationMethod Algorithm="` + alg + `">` +
				`<ec:InclusiveNamespaces xmlns:ec="` + ExcC14N10 + `" PrefixList="extra ns2"/>` +
				`</ds:CanonicalizationMethod>` +
				`<ds:SignatureMethod Algorithm="` + AlgRSASHA256 + `"/>` +
				`<ds:Reference URI="">` +
				`<ds:DigestMethod Algorithm="` + DigestSHA256 + `"/>` +
				`<ds:DigestValue>AA==</ds:DigestValue>` +
				`</ds:Reference>` +
				`</ds:SignedInfo>`
			doc := mustParse(t, si)
			var parsed parsedSignature
			require.NoError(t, parseSignedInfo(doc.DocumentElement(), &parsed))
			require.Equal(t, []string{"extra", "ns2"}, parsed.c14nPrefixes)
		})
	}
}
