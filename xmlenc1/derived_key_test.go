package xmlenc1_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// derivedKeyMarkup is the construct under test: an xenc11:DerivedKey naming a
// derivation this package implements nothing for.
const derivedKeyMarkup = `<xenc11:DerivedKey xmlns:xenc11="http://www.w3.org/2009/xmlenc11#"><xenc11:KeyDerivationMethod Algorithm="http://www.w3.org/2009/xmlenc11#ConcatKDF"/></xenc11:DerivedKey>`

// reparseWith serializes edElem, applies fn to the markup, and re-parses the
// result into a fresh EncryptedData element.
//
// The splice is done on TEXT because the DOM API offers no way to write an
// xenc11:DerivedKey the Encryptor never emits, and re-parsing is what puts the
// spliced element on the same footing as one a peer sent.
func reparseWith(t *testing.T, edElem *helium.Element, fn func(string) string) *helium.Element {
	t.Helper()
	src, err := helium.WriteString(edElem)
	require.NoError(t, err)

	doc := mustParseXML(t, fn(src))
	out := findEncryptedData(t, doc.DocumentElement())
	require.NotNil(t, out)
	return out
}

// insertDerivedKeyBefore puts a DerivedKey ahead of the EncryptedKey the
// Encryptor wrote, inside the ds:KeyInfo the two then share.
func insertDerivedKeyBefore(t *testing.T, edElem *helium.Element) *helium.Element {
	t.Helper()
	return reparseWith(t, edElem, func(src string) string {
		const open = `<ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">`
		require.Contains(t, src, open, "the Encryptor must have written a ds:KeyInfo to splice into")
		return strings.Replace(src, open, open+derivedKeyMarkup, 1)
	})
}

// insertDerivedKeyInfo adds a whole ds:KeyInfo holding only a DerivedKey. It is
// for a document the Encryptor wrote with no key protection at all (a bare
// SessionKey), which therefore carries no ds:KeyInfo of its own.
func insertDerivedKeyInfo(t *testing.T, edElem *helium.Element) *helium.Element {
	t.Helper()
	return reparseWith(t, edElem, func(src string) string {
		const cipherData = `<xenc:CipherData>`
		require.Contains(t, src, cipherData)
		keyInfo := `<ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` + derivedKeyMarkup + `</ds:KeyInfo>`
		return strings.Replace(src, cipherData, keyInfo+cipherData, 1)
	})
}

// An xenc11:DerivedKey tells the recipient to derive the content key from
// master key material it already holds (xmlenc-core1 §3.5.2). This package
// implements no key derivation, so such a document cannot be decrypted here.
//
// What it must NOT report is ErrMissingKey. That sentinel says "no decryption
// key available", which sends a caller to audit the keys it configured — and
// nothing it configures can help, because the document asked for a facility
// this package does not have. ErrUnsupportedKeyDerivation names the real
// reason, so the caller can tell "I set this up wrong" from "helium cannot
// read this document at all".
func TestDerivedKeyUnsupported(t *testing.T) {
	// derivedKeyDoc carries a DerivedKey as the ONLY thing its ds:KeyInfo
	// offers, so key resolution finds no candidate and the refusal is
	// reached.
	const derivedKeyDoc = `<EncryptedData xmlns="http://www.w3.org/2001/04/xmlenc#" Type="http://www.w3.org/2001/04/xmlenc#Element">
  <EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/>
  <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <xenc11:DerivedKey xmlns:xenc11="http://www.w3.org/2009/xmlenc11#">
      <xenc11:KeyDerivationMethod Algorithm="http://www.w3.org/2009/xmlenc11#ConcatKDF"/>
    </xenc11:DerivedKey>
  </ds:KeyInfo>
  <CipherData><CipherValue>YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU=</CipherValue></CipherData>
</EncryptedData>`

	// referencedDerivedKeyDoc reaches the same construct the other way: a
	// ds:RetrievalMethod whose Type names a DerivedKey. The parse recognizes
	// that Type, so this path must refuse alike.
	const referencedDerivedKeyDoc = `<doc xmlns:xenc="http://www.w3.org/2001/04/xmlenc#" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xmlns:xenc11="http://www.w3.org/2009/xmlenc11#">
  <xenc11:DerivedKey Id="dk"/>
  <xenc:EncryptedData Type="http://www.w3.org/2001/04/xmlenc#Element">
    <xenc:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/>
    <ds:KeyInfo>
      <ds:RetrievalMethod Type="http://www.w3.org/2001/04/xmlenc#DerivedKey" URI="#dk"/>
    </ds:KeyInfo>
    <xenc:CipherData><xenc:CipherValue>YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU=</xenc:CipherValue></xenc:CipherData>
  </xenc:EncryptedData>
</doc>`

	for _, tc := range []struct {
		name string
		src  string
	}{
		{name: "inline", src: derivedKeyDoc},
		{name: "named by a RetrievalMethod", src: referencedDerivedKeyDoc},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := mustParseXML(t, tc.src)
			elem := findEncryptedData(t, doc.DocumentElement())
			require.NotNil(t, elem)

			_, err := xmlenc1.NewDecryptor().
				PrivateKey(generateRSAKey(t)).
				Decrypt(t.Context(), elem)
			require.Error(t, err)
			require.ErrorIs(t, err, xmlenc1.ErrUnsupportedKeyDerivation)

			// The point of the new sentinel: a caller testing for a
			// misconfigured key must NOT match this.
			require.NotErrorIs(t, err, xmlenc1.ErrMissingKey)
		})
	}

	// DecryptBytes shares the key resolution, so it must refuse alike.
	t.Run("DecryptBytes refuses alike", func(t *testing.T) {
		doc := mustParseXML(t, derivedKeyDoc)
		elem := findEncryptedData(t, doc.DocumentElement())

		_, err := xmlenc1.NewDecryptor().
			PrivateKey(generateRSAKey(t)).
			DecryptBytes(t.Context(), elem)
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrUnsupportedKeyDerivation)
		require.NotErrorIs(t, err, xmlenc1.ErrMissingKey)
	})

	// An EncryptedData carrying NO key material at all is still the plain
	// missing-key case. The new sentinel must not swallow it.
	t.Run("a keyless document still reports ErrMissingKey", func(t *testing.T) {
		const keyless = `<EncryptedData xmlns="http://www.w3.org/2001/04/xmlenc#" Type="http://www.w3.org/2001/04/xmlenc#Element">
  <EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/>
  <CipherData><CipherValue>YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU=</CipherValue></CipherData>
</EncryptedData>`

		doc := mustParseXML(t, keyless)
		elem := findEncryptedData(t, doc.DocumentElement())

		_, err := xmlenc1.NewDecryptor().
			PrivateKey(generateRSAKey(t)).
			Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
		require.NotErrorIs(t, err, xmlenc1.ErrUnsupportedKeyDerivation)
	})
}

// TestDerivedKeyAlongsideUsableKey pins that the refusal fires only when
// derivation was the ONLY thing on offer. A document that carries a DerivedKey
// AND a usable EncryptedKey still decrypts under the key it can actually use:
// helium ignores the construct it does not implement and takes the one it
// does, exactly as it did before the sentinel existed.
func TestDerivedKeyAlongsideUsableKey(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, `<Response><Assertion>secret</Assertion></Response>`)

	assertion, ok := helium.AsNode[*helium.Element](doc.DocumentElement().FirstChild())
	require.True(t, ok)

	edElem, err := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM11).
		KeyTransportAlgorithm(xmlenc1.RSAOAEP).
		RecipientPublicKey(&key.PublicKey).
		EncryptElement(t.Context(), assertion)
	require.NoError(t, err)

	// Splice a DerivedKey into the same ds:KeyInfo, ahead of the
	// EncryptedKey the encryptor wrote.
	spliced := insertDerivedKeyBefore(t, edElem)

	nodes, err := xmlenc1.NewDecryptor().PrivateKey(key).Decrypt(t.Context(), spliced)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	out, err := helium.WriteString(nodes[0])
	require.NoError(t, err)
	require.Contains(t, out, "secret")
}

// TestDerivedKeyPreSharedSessionKeyDecrypts pins that a caller holding the
// session key already is unaffected. Decryptor.SessionKey returns before key
// resolution runs at all, so a document whose ds:KeyInfo this package cannot
// read still decrypts for the caller who never needed it read.
func TestDerivedKeyPreSharedSessionKeyDecrypts(t *testing.T) {
	sessionKey := randKey(t, 32)
	doc := mustParseXML(t, `<Response><Assertion>secret</Assertion></Response>`)

	assertion, ok := helium.AsNode[*helium.Element](doc.DocumentElement().FirstChild())
	require.True(t, ok)

	edElem, err := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM11).
		SessionKey(sessionKey).
		EncryptElement(t.Context(), assertion)
	require.NoError(t, err)

	spliced := insertDerivedKeyInfo(t, edElem)

	nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), spliced)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
}
