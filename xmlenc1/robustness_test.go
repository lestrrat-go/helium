package xmlenc1_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// TestDecryptBogusFirstKeyTriesNext covers XENC-005: a candidate
// EncryptedKey that UNWRAPS/transports successfully but wraps the WRONG
// session key must not short-circuit the search. The decryptor must carry
// each candidate through block decryption (and plaintext validation),
// falling through to the next candidate on failure. Otherwise an attacker
// who prepends a valid RSA-OAEP EncryptedKey (for the recipient's public
// key) wrapping a random session key denies service to the legitimate key.
func TestDecryptBogusFirstKeyTriesNext(t *testing.T) {
	const blockAlg = xmlenc1.AES256GCM

	priv := generateRSAKey(t)
	pub := &priv.PublicKey

	wrapOAEP := func(t *testing.T, sessionKey []byte) []byte {
		t.Helper()
		// RSA-OAEP-MGF1P uses SHA-1 for both digest and MGF1.
		ct, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, pub, sessionKey, nil)
		require.NoError(t, err)
		return ct
	}

	realSessionKey := randKey(t, 32)
	cipher, err := xmlenc1.EncryptBytesForTest(blockAlg, realSessionKey, []byte("<x>secret</x>"))
	require.NoError(t, err)

	// First key is a perfectly valid RSA-OAEP transport of a RANDOM session
	// key: it unwraps cleanly under priv but cannot decrypt the ciphertext.
	// Second key transports the real session key.
	bogusSessionKey := randKey(t, 32)

	doc := mustParseXML(t, `<root/>`)
	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: blockAlg},
		EncryptedKeys: []*xmlenc1.EncryptedKey{
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
				CipherValue:      wrapOAEP(t, bogusSessionKey),
			},
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
				CipherValue:      wrapOAEP(t, realSessionKey),
			},
		},
		CipherValue: cipher,
	}
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	nodes, err := xmlenc1.NewDecryptor().PrivateKey(priv).Decrypt(t.Context(), elem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	s, err := helium.WriteString(nodes[0])
	require.NoError(t, err)
	require.Contains(t, s, "secret")
}

// TestMultiRecipientDecrypt covers XENC-002: an EncryptedData may carry
// several EncryptedKey candidates (one per recipient), and decryption must
// try each, committing to none of them first. This also makes a bogus
// EncryptedKey prepended to a legitimate one a non-issue instead of a DoS.
func TestMultiRecipientDecrypt(t *testing.T) {
	const algorithm = xmlenc1.AES256GCM

	newEncryptedData := func(t *testing.T, keys []*xmlenc1.EncryptedKey, cipher []byte) *helium.Element {
		t.Helper()
		doc := mustParseXML(t, `<root/>`)
		ed := &xmlenc1.EncryptedData{
			Type:             xmlenc1.TypeElement,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
			EncryptedKeys:    keys,
			CipherValue:      cipher,
		}
		elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)
		return elem
	}

	wrap := func(t *testing.T, kek, sessionKey []byte) []byte {
		t.Helper()
		wrapped, err := xmlenc1.AESKeyWrapForTest(kek, sessionKey)
		require.NoError(t, err)
		return wrapped
	}

	t.Run("second recipient matches", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("<x>secret</x>"))
		require.NoError(t, err)

		kekOther := randKey(t, 32)
		kekMine := randKey(t, 32)

		// Two legitimate recipients; only the second one's KEK is ours.
		elem := newEncryptedData(t, []*xmlenc1.EncryptedKey{
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
				CipherValue:      wrap(t, kekOther, sessionKey),
			},
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
				CipherValue:      wrap(t, kekMine, sessionKey),
			},
		}, cipher)

		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kekMine).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "secret")
	})

	t.Run("bogus first key tolerated", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("<x>secret</x>"))
		require.NoError(t, err)

		kekMine := randKey(t, 32)

		// A junk EncryptedKey is prepended ahead of the legitimate one.
		// Under the old "first key only" behavior this denied service to
		// the real recipient.
		elem := newEncryptedData(t, []*xmlenc1.EncryptedKey{
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
				CipherValue:      randKey(t, 40), // not a valid AES-wrap of any key
			},
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
				CipherValue:      wrap(t, kekMine, sessionKey),
			},
		}, cipher)

		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kekMine).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "secret")
	})

	t.Run("applicable failure not masked by later missing key", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("<x>secret</x>"))
		require.NoError(t, err)

		kekMine := randKey(t, 32)
		wrongSessionKey := randKey(t, 32)

		// First key is APPLICABLE (AES key-wrap, our KEK) but transports the
		// WRONG session key, so block decryption fails with a real error.
		// Second key is NON-APPLICABLE (RSA-OAEP, no private key supplied) and
		// only yields ErrMissingKey. The informative ErrDecryptionFailed must
		// surface, and the trailing ErrMissingKey must never overwrite it.
		elem := newEncryptedData(t, []*xmlenc1.EncryptedKey{
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
				CipherValue:      wrap(t, kekMine, wrongSessionKey),
			},
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
				CipherValue:      randKey(t, 256),
			},
		}, cipher)

		_, err = xmlenc1.NewDecryptor().KeyEncryptionKey(kekMine).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
		require.NotErrorIs(t, err, xmlenc1.ErrMissingKey)
	})
}

// secondKeyInfoDropsAgreementMethod builds an EncryptedKey carrying two
// ds:KeyInfo children: the first holds a full, cryptographically valid
// ECDH-ES AgreementMethod (AgreementMethod/KeyDerivationMethod/
// ConcatKDFParams/OriginatorKeyInfo), the second only a ds:KeyName hint.
// Before the KeyInfo cardinality guard exists, the EncryptedKey branch
// assigns ek.AgreementMethod on every ds:KeyInfo child and rejects
// no second one, so the KeyName-only KeyInfo silently overwrites the real
// AgreementMethod with nil.
func secondKeyInfoDropsAgreementMethod(t *testing.T) *helium.Element {
	t.Helper()
	point := ecPublicKeyPoint(t, ecdh.P256())
	encoded := base64.StdEncoding.EncodeToString(point)

	const xenc = `xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"`
	const ds = `xmlns:ds="http://www.w3.org/2000/09/xmldsig#"`
	const dsig11 = `xmlns:dsig11="http://www.w3.org/2009/xmldsig11#"`
	const xenc11 = `xmlns:xenc11="http://www.w3.org/2009/xmlenc11#"`

	xml := `<xenc:EncryptedData ` + xenc + ` ` + ds + ` ` + dsig11 + ` ` + xenc11 + `>` +
		`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
		`<ds:KeyInfo><xenc:EncryptedKey>` +
		`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES128KeyWrap + `"/>` +
		`<ds:KeyInfo><xenc:AgreementMethod Algorithm="` + xmlenc1.ECDHES + `">` +
		`<xenc11:KeyDerivationMethod Algorithm="` + xmlenc1.ConcatKDF + `">` +
		`<xenc11:ConcatKDFParams><ds:DigestMethod Algorithm="` + xmlenc1.DigestSHA256 + `"/></xenc11:ConcatKDFParams>` +
		`</xenc11:KeyDerivationMethod>` +
		`<xenc:OriginatorKeyInfo><ds:KeyValue><dsig11:ECKeyValue>` +
		`<dsig11:NamedCurve URI="` + ecCurveURIP256 + `"/>` +
		`<dsig11:PublicKey>` + encoded + `</dsig11:PublicKey>` +
		`</dsig11:ECKeyValue></ds:KeyValue></xenc:OriginatorKeyInfo>` +
		`</xenc:AgreementMethod></ds:KeyInfo>` +
		`<ds:KeyInfo><ds:KeyName>hint</ds:KeyName></ds:KeyInfo>` +
		`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
		`</xenc:EncryptedKey></ds:KeyInfo>` +
		`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
		`</xenc:EncryptedData>`
	return mustParseXML(t, xml).DocumentElement()
}

// duplicateNamedCurve builds an ECDH-ES EncryptedKey whose dsig11:ECKeyValue
// carries two dsig11:NamedCurve children (P-256 then P-521) ahead of one
// real P-256 point. Before the ECKeyValue cardinality guard exists, the
// second NamedCurve silently overrides curve selection, so the point is
// weighed against the wrong curve and rejected with an invalid-point
// message, and never a duplicate-NamedCurve one.
func duplicateNamedCurve(t *testing.T) *helium.Element {
	t.Helper()
	point := base64.StdEncoding.EncodeToString(ecPublicKeyPoint(t, ecdh.P256()))
	children := `<dsig11:NamedCurve URI="` + ecCurveURIP256 + `"/>` +
		`<dsig11:NamedCurve URI="` + ecCurveURIP521 + `"/>` +
		`<dsig11:PublicKey>` + point + `</dsig11:PublicKey>`
	return ecKeyValueEncryptedData(t, children)
}

// duplicatePublicKey builds an ECDH-ES EncryptedKey whose dsig11:ECKeyValue
// carries a NamedCurve followed by two independently generated, individually
// valid P-256 points. Before the ECKeyValue cardinality guard exists, the
// second PublicKey silently overwrites the first and is never rejected.
func duplicatePublicKey(t *testing.T) *helium.Element {
	t.Helper()
	first := base64.StdEncoding.EncodeToString(ecPublicKeyPoint(t, ecdh.P256()))
	second := base64.StdEncoding.EncodeToString(ecPublicKeyPoint(t, ecdh.P256()))
	children := `<dsig11:NamedCurve URI="` + ecCurveURIP256 + `"/>` +
		`<dsig11:PublicKey>` + first + `</dsig11:PublicKey>` +
		`<dsig11:PublicKey>` + second + `</dsig11:PublicKey>`
	return ecKeyValueEncryptedData(t, children)
}

// TestParseRejectsDuplicateCardinality covers XENC-003: XML Encryption
// allows at most one EncryptionMethod and one CipherData per EncryptedData
// (and per EncryptedKey), at most one ds:KeyInfo per EncryptedType, and at
// most one NamedCurve/PublicKey per dsig11:ECKeyValue. Duplicates were
// previously accepted last-one-wins; they must now be rejected during
// parse.
func TestParseRejectsDuplicateCardinality(t *testing.T) {
	const xenc = `xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"`
	const ds = `xmlns:ds="http://www.w3.org/2000/09/xmldsig#"`

	for _, tc := range []struct {
		name    string
		xml     string
		build   func(t *testing.T) *helium.Element
		wantMsg string
	}{
		{
			name: "duplicate EncryptionMethod in EncryptedData",
			xml: `<xenc:EncryptedData ` + xenc + `>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES128GCM + `"/>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`,
		},
		{
			name: "duplicate CipherData in EncryptedData",
			xml: `<xenc:EncryptedData ` + xenc + `>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`<xenc:CipherData><xenc:CipherValue>BBBB</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`,
		},
		{
			name: "duplicate EncryptionMethod in EncryptedKey",
			xml: `<xenc:EncryptedData ` + xenc + ` ` + ds + `>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
				`<ds:KeyInfo><xenc:EncryptedKey>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP + `"/>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP11 + `"/>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedKey></ds:KeyInfo>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`,
		},
		{
			name: "duplicate CipherData in EncryptedKey",
			xml: `<xenc:EncryptedData ` + xenc + ` ` + ds + `>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
				`<ds:KeyInfo><xenc:EncryptedKey>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP + `"/>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`<xenc:CipherData><xenc:CipherValue>BBBB</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedKey></ds:KeyInfo>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`,
		},
		{
			name: "duplicate KeyInfo in EncryptedData",
			xml: `<xenc:EncryptedData ` + xenc + ` ` + ds + `>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
				`<ds:KeyInfo><xenc:EncryptedKey>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES128KeyWrap + `"/>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedKey></ds:KeyInfo>` +
				`<ds:KeyInfo><xenc:EncryptedKey>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES128KeyWrap + `"/>` +
				`<xenc:CipherData><xenc:CipherValue>BBBB</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedKey></ds:KeyInfo>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`,
		},
		{
			name: "duplicate KeyInfo in EncryptedKey",
			xml: `<xenc:EncryptedData ` + xenc + ` ` + ds + `>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
				`<ds:KeyInfo><xenc:EncryptedKey>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP + `"/>` +
				`<ds:KeyInfo/><ds:KeyInfo/>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedKey></ds:KeyInfo>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`,
		},
		{
			name:  "second KeyInfo in EncryptedKey drops AgreementMethod",
			build: secondKeyInfoDropsAgreementMethod,
		},
		{
			name:    "duplicate NamedCurve in ECKeyValue",
			build:   duplicateNamedCurve,
			wantMsg: "duplicate ECKeyValue NamedCurve",
		},
		{
			name:  "duplicate PublicKey in ECKeyValue",
			build: duplicatePublicKey,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			elem := tc.build
			var target *helium.Element
			if elem != nil {
				target = elem(t)
			} else {
				target = mustParseXML(t, tc.xml).DocumentElement()
			}
			_, err := xmlenc1.ParseEncryptedDataForTest(target)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
			if tc.wantMsg != "" {
				require.Contains(t, err.Error(), tc.wantMsg)
			}
		})
	}
}

// TestDecryptType covers XENC-004: Decrypt parses the recovered plaintext as
// XML only when @Type names Element or Content. @Type sits outside the
// ciphertext and is unauthenticated even under AES-GCM, so an absent, empty,
// or unrecognized value is refused with ErrOpaquePayload, and never parsed:
// stripping the attribute must not reinterpret an authenticated opaque octet
// stream as an element the caller may graft into its tree. DecryptBytes is
// the octet path, and the error names it.
func TestDecryptType(t *testing.T) {
	const algorithm = xmlenc1.AES256GCM

	build := func(t *testing.T, typeURI string, cipher []byte) *helium.Element {
		t.Helper()
		doc := mustParseXML(t, `<root/>`)
		ed := &xmlenc1.EncryptedData{
			Type:             typeURI,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
			CipherValue:      cipher,
		}
		elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)
		return elem
	}

	// Each refusal builds an EncryptedData whose plaintext WOULD parse as one
	// element, so the verdict comes from @Type alone and not from the payload.
	for _, tc := range []struct {
		name         string
		typeURI      string
		explicitAttr bool
	}{
		{name: "absent Type refused"},
		{name: "explicit empty Type refused", explicitAttr: true},
		{name: "unknown Type refused", typeURI: "urn:example:bogus-type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sessionKey := randKey(t, 32)
			cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("<x>secret</x>"))
			require.NoError(t, err)

			elem := build(t, tc.typeURI, cipher)
			if tc.explicitAttr {
				require.NoError(t, elem.SetAttribute("Type", ""))
			}

			_, err = xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrOpaquePayload)
			// The payload is opaque, not malformed: the document is
			// well-formed and a caller must be able to tell the two apart.
			require.NotErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
			require.Contains(t, err.Error(), "DecryptBytes")
		})
	}

	t.Run("Element Type parses as XML", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("<x>secret</x>"))
		require.NoError(t, err)

		elem := build(t, xmlenc1.TypeElement, cipher)
		nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, helium.ElementNode, nodes[0].Type())
	})

	// The case the strict default is for: EncryptBytes emits no @Type, so an
	// opaque payload reaches Decrypt looking exactly like one whose @Type an
	// attacker stripped. Decrypt must refuse it by name instead of feeding the
	// octets to the XML parser and reporting whatever that parse decided.
	t.Run("opaque payload with no Type is refused, not parsed", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		payload := []byte{0x00, 0x01, 0xff, 0xfe, 'n', 'o', 't', ' ', 'x', 'm', 'l'}
		doc := mustParseXML(t, `<root/>`)

		elem, err := xmlenc1.NewEncryptor().SessionKey(sessionKey).EncryptBytes(t.Context(), doc, payload)
		require.NoError(t, err)

		_, err = xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrOpaquePayload)
		require.NotErrorIs(t, err, xmlenc1.ErrDecryptionFailed)

		// DecryptBytes is the path the error names, and it still works.
		got, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).DecryptBytes(t.Context(), elem)
		require.NoError(t, err)
		require.Equal(t, payload, got)
	})
}

// encryptedDataWithRawEncryptedKey builds an EncryptedData carrying real
// AES-256-GCM ciphertext plus one EncryptedKey spliced in verbatim, so a test
// can present an EncryptedKey the public marshaler would never emit.
func encryptedDataWithRawEncryptedKey(t *testing.T, sessionKey []byte, plaintext, rawEncryptedKey string) *helium.Element {
	t.Helper()
	cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte(plaintext))
	require.NoError(t, err)

	const xenc = `xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"`
	const ds = `xmlns:ds="http://www.w3.org/2000/09/xmldsig#"`
	xml := `<xenc:EncryptedData ` + xenc + ` ` + ds + ` Type="` + xmlenc1.TypeElement + `">` +
		`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
		`<ds:KeyInfo>` + rawEncryptedKey + `</ds:KeyInfo>` +
		`<xenc:CipherData><xenc:CipherValue>` + base64.StdEncoding.EncodeToString(cipher) +
		`</xenc:CipherValue></xenc:CipherData>` +
		`</xenc:EncryptedData>`
	return mustParseXML(t, xml).DocumentElement()
}

// TestSessionKeyBypassesSelectionNotParsing pins the documented scope of
// Decryptor.SessionKey: it is the sole key-resolution candidate, so no
// EncryptedKey is ever selected or decrypted, but decryptElement parses the
// whole EncryptedData first — including every EncryptedKey under ds:KeyInfo —
// so a malformed candidate still aborts the decrypt when it is within the
// MaxEncryptedKeys cap. An excess candidate is rejected before parsing;
// TestMaxEncryptedKeys covers that cap.
func TestSessionKeyBypassesSelectionNotParsing(t *testing.T) {
	t.Run("malformed EncryptedKey fails the decrypt", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		// No CipherData child: parseEncryptedKey rejects it before the
		// session-key branch is ever reached.
		raw := `<xenc:EncryptedKey>` +
			`<xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP + `"/>` +
			`</xenc:EncryptedKey>`
		elem := encryptedDataWithRawEncryptedKey(t, sessionKey, `<x>secret</x>`, raw)

		_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		// Pin the failing element: the EncryptedKey, not the EncryptedData.
		require.ErrorContains(t, err, "EncryptedKey missing CipherData/CipherValue")
	})

	t.Run("well-formed EncryptedKey is parsed but never selected", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		// A syntactically valid RSA-OAEP candidate whose CipherValue is
		// junk. Selection never runs, so the absent RSA private key and
		// the undecryptable candidate are both irrelevant.
		raw := `<xenc:EncryptedKey>` +
			`<xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP + `"/>` +
			`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
			`</xenc:EncryptedKey>`
		elem := encryptedDataWithRawEncryptedKey(t, sessionKey, `<x>secret</x>`, raw)

		nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "secret")
	})
}
