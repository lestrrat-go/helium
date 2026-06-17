package xmldsig1_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// TestDuplicateSignedInfoRejected mounts the duplicate-SignedInfo attack:
// the genuine, signed SignedInfo is the LAST child of the Signature (so it
// is the one canonicalized and checked against SignatureValue), while an
// attacker prepends a second, UNSIGNED SignedInfo carrying a Reference to
// attacker-controlled content with a self-consistent DigestValue. Verify
// must reject the document rather than report the injected reference as
// covered by the signature.
func TestDuplicateSignedInfoRejected(t *testing.T) {
	xml := `<root><data Id="payload">secret</data></root>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, xml)

	ref := xmldsig1.ReferenceConfig{
		URI:             "#payload",
		DigestAlgorithm: xmldsig1.DigestSHA256,
		Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
	}
	sigElem, err := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(ref).
		SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	signed, err := helium.WriteString(doc)
	require.NoError(t, err)

	// Extract the genuine SignedInfo block from the signed output.
	siStart := strings.Index(signed, "<ds:SignedInfo")
	require.GreaterOrEqual(t, siStart, 0)
	siEnd := strings.Index(signed, "</ds:SignedInfo>") + len("</ds:SignedInfo>")
	require.Greater(t, siEnd, siStart)
	genuineSI := signed[siStart:siEnd]

	// Build the injected, unsigned SignedInfo by computing a correct digest
	// for an evil payload. We obtain that digest by signing a throwaway doc
	// containing the evil element and lifting its SignedInfo verbatim — the
	// attacker controls the document, so this digest is self-consistent.
	evilDoc := mustParseXML(t, `<root><data Id="evil">attacker-controlled</data></root>`)
	evilSig, err := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#evil",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}).
		SignDetached(t.Context(), evilDoc, generateRSAKey(t))
	require.NoError(t, err)
	require.NoError(t, evilDoc.DocumentElement().AddChild(evilSig))
	evilSigned, err := helium.WriteString(evilDoc)
	require.NoError(t, err)
	eStart := strings.Index(evilSigned, "<ds:SignedInfo")
	eEnd := strings.Index(evilSigned, "</ds:SignedInfo>") + len("</ds:SignedInfo>")
	injectedSI := evilSigned[eStart:eEnd]

	// Inject: prepend the unsigned (evil) SignedInfo before the genuine one,
	// and inject the evil element into the document body. The genuine
	// SignedInfo remains the last SignedInfo child of the Signature, so it is
	// still the one whose canonical form matches SignatureValue.
	require.NotEqual(t, genuineSI, injectedSI)
	tampered := strings.Replace(signed, genuineSI, injectedSI+genuineSI, 1)
	tampered = strings.Replace(tampered, `<data Id="payload">secret</data>`,
		`<data Id="evil">attacker-controlled</data><data Id="payload">secret</data>`, 1)
	require.NotEqual(t, signed, tampered)

	doc2 := mustParseXML(t, tampered)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	res, err := verifier.Verify(t.Context(), doc2)

	// The document is structurally invalid (two SignedInfo elements in one
	// Signature). Verify must reject it outright.
	require.Error(t, err, "duplicate SignedInfo must be rejected")
	if res != nil {
		for _, r := range res.References {
			require.NotEqual(t, "#evil", r.URI,
				"injected, unsigned reference must never be reported as verified")
		}
	}
}

// TestDuplicateSignatureValueRejected guards against a Signature element
// carrying more than one SignatureValue child.
func TestDuplicateSignatureValueRejected(t *testing.T) {
	xml := `<root><data Id="payload">secret</data></root>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, xml)

	sigElem, err := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#payload",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}).
		SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	signed, err := helium.WriteString(doc)
	require.NoError(t, err)

	svStart := strings.Index(signed, "<ds:SignatureValue")
	require.GreaterOrEqual(t, svStart, 0)
	svEnd := strings.Index(signed, "</ds:SignatureValue>") + len("</ds:SignatureValue>")
	svBlock := signed[svStart:svEnd]
	tampered := strings.Replace(signed, svBlock, svBlock+svBlock, 1)
	require.NotEqual(t, signed, tampered)

	doc2 := mustParseXML(t, tampered)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc2)
	require.Error(t, err, "duplicate SignatureValue must be rejected")
}
