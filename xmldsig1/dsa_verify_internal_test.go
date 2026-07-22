package xmldsig1

import (
	"context"
	"crypto/dsa" // verify-only legacy interop; see verifyDSA
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // DSA-SHA1 legacy interop digest, gated behind AllowSHA1
	"encoding/base64"
	"math/big"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/stretchr/testify/require"
)

// leftPad returns b left-padded with zero bytes to exactly size bytes.
func leftPad(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

// findByLocalName returns the first descendant-or-self element of n whose local
// name matches, depth-first.
func findByLocalName(n helium.Node, local string) *helium.Element {
	e, ok := helium.AsNode[*helium.Element](n)
	if !ok {
		return nil
	}
	if domutil.LocalName(e) == local {
		return e
	}
	for c := e.FirstChild(); c != nil; c = c.NextSibling() {
		if found := findByLocalName(c, local); found != nil {
			return found
		}
	}
	return nil
}

// TestDSASHA1VerifyEndToEnd builds a real DSA-SHA1 enveloping signature with a
// freshly generated DSA key — the library cannot sign DSA, so the digest and
// signature are computed here through the package's own canonicalizer — then
// verifies it end-to-end through the public API. The DSA public key is carried
// in the signature as a DSAKeyValue and rebuilt by the KeySource, exercising
// both DSAKeyValue parsing and the DSA-SHA1 verify path.
func TestDSASHA1VerifyEndToEnd(t *testing.T) {
	var params dsa.Parameters
	require.NoError(t, dsa.GenerateParameters(&params, rand.Reader, dsa.L1024N160))
	priv := &dsa.PrivateKey{PublicKey: dsa.PublicKey{Parameters: params}}
	require.NoError(t, dsa.GenerateKey(priv, rand.Reader))

	b64 := func(i *big.Int) string { return base64.StdEncoding.EncodeToString(i.Bytes()) }

	src := `<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">` +
		`<SignedInfo>` +
		`<CanonicalizationMethod Algorithm="` + C14N10 + `"/>` +
		`<SignatureMethod Algorithm="` + AlgDSASHA1 + `"/>` +
		`<Reference URI="#object"><DigestMethod Algorithm="` + DigestSHA1 + `"/><DigestValue></DigestValue></Reference>` +
		`</SignedInfo>` +
		`<SignatureValue></SignatureValue>` +
		`<KeyInfo><KeyValue><DSAKeyValue>` +
		`<P>` + b64(priv.P) + `</P>` +
		`<Q>` + b64(priv.Q) + `</Q>` +
		`<G>` + b64(priv.G) + `</G>` +
		`<Y>` + b64(priv.Y) + `</Y>` +
		`</DSAKeyValue></KeyValue></KeyInfo>` +
		`<Object Id="object">Approved</Object>` +
		`</Signature>`
	doc := mustParse(t, src)
	root := doc.DocumentElement()

	// Compute the reference digest over the Object subtree (inclusive Canonical
	// XML 1.0, the default when a Reference declares no c14n transform) and fill
	// in DigestValue before canonicalizing SignedInfo.
	object := findByLocalName(root, "Object")
	require.NotNil(t, object)
	objCanon, err := canonicalizeSubtree(C14N10, object, nil)
	require.NoError(t, err)
	objDigest := sha1.Sum(objCanon)
	digestValue := findByLocalName(root, "DigestValue")
	require.NotNil(t, digestValue)
	require.NoError(t, digestValue.AddChild(doc.CreateText([]byte(base64.StdEncoding.EncodeToString(objDigest[:])))))

	// Sign the canonicalized SignedInfo with DSA and encode as the XML-DSig
	// r||s concatenation (each integer left-padded to Q's byte length).
	signedInfo := findByLocalName(root, "SignedInfo")
	require.NotNil(t, signedInfo)
	siCanon, err := canonicalizeSubtree(C14N10, signedInfo, nil)
	require.NoError(t, err)
	siDigest := sha1.Sum(siCanon)
	r, s, err := dsa.Sign(rand.Reader, priv, siDigest[:])
	require.NoError(t, err)
	qLen := (priv.Q.BitLen() + 7) / 8
	rawSig := append(leftPad(r.Bytes(), qLen), leftPad(s.Bytes(), qLen)...)
	sigValue := findByLocalName(root, "SignatureValue")
	require.NotNil(t, sigValue)
	require.NoError(t, sigValue.AddChild(doc.CreateText([]byte(base64.StdEncoding.EncodeToString(rawSig)))))

	ks := KeySourceFunc(func(_ context.Context, ki *KeyInfoData, _ string) (any, error) {
		require.NotNil(t, ki.DSAKeyValue, "DSAKeyValue must be parsed into KeyInfoData")
		return &dsa.PublicKey{
			Parameters: dsa.Parameters{P: ki.DSAKeyValue.P, Q: ki.DSAKeyValue.Q, G: ki.DSAKeyValue.G},
			Y:          ki.DSAKeyValue.Y,
		}, nil
	})

	res, err := NewVerifier(ks).AllowSHA1(true).Verify(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.References, 1)
}
