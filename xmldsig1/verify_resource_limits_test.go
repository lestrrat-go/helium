package xmldsig1_test

import (
	"crypto/rsa"
	"runtime"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// signTwoReferenceDoc signs samlAssertion with two whole-document enveloped
// References, producing a validly-signed document that carries more than one
// Reference so the MaxReferences cap can be exercised end to end.
func signTwoReferenceDoc(t *testing.T, key any) *helium.Document {
	t.Helper()
	doc := mustParseXML(t, samlAssertion)
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		Reference(xmldsig1.NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))
	return doc
}

// TestVerifyResourceLimits covers the opt-in parse-time resource caps that bound
// the decode/parse work an attacker-controlled Signature can force before the
// SignatureValue is checked.
func TestVerifyResourceLimits(t *testing.T) {
	// MaxReferences: a document declaring more References than the cap is
	// rejected with ErrResourceLimitExceeded before any Reference is digested,
	// while the same document verifies under the default (generous) cap.
	t.Run("max references", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := signTwoReferenceDoc(t, key)

		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			MaxReferences(1).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)

		// Default cap leaves the two-Reference document verifying normally.
		_, err = xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.NoError(t, err)

		// A negative cap disables the check entirely.
		_, err = xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			MaxReferences(-1).
			Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// MaxDecodedBytes: a running total of certificate and signature octets over
	// the cap is rejected. A one-byte cap trips on the first DigestValue decode;
	// the default cap is unaffected. The RetrievalMethod charge sites the same
	// cap covers are exercised in retrieval_method_internal_test.go.
	t.Run("max decoded bytes", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			MaxDecodedBytes(1).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)

		_, err = xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// MaxKeyInfoEntries: a KeyInfo carrying more entries than the cap is rejected.
	// A KeyInfo with two X509Certificate children exceeds a cap of 1; the default
	// cap leaves it verifying.
	t.Run("max key info entries", func(t *testing.T) {
		key := generateRSAKey(t)
		cert := generateSelfSignedCert(t, key)
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference()).
			KeyInfo(xmldsig1.X509DataKeyInfo(cert, cert))
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		_, err := xmldsig1.NewVerifier(xmldsig1.X509CertKeySource(cert)).
			MaxKeyInfoEntries(1).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)

		_, err = xmldsig1.NewVerifier(xmldsig1.X509CertKeySource(cert)).
			Verify(t.Context(), doc)
		require.NoError(t, err)
	})
}

// TestVerifyBase64ValueChildren pins which children of a ds: base64 value carry
// its characters. xs:base64Binary is a simple type, so only character
// information items belong to the value: text and CDATA children, and an entity
// reference, which stands for character data. A comment or processing
// instruction contributes none, and an element child is not permitted there at
// all.
func TestVerifyBase64ValueChildren(t *testing.T) {
	key := generateRSAKey(t)

	// A comment inside a signature value must leave the value it wraps
	// untouched. Splicing the comment's own text into the base64 stream would
	// reject a legitimately commented signature.
	t.Run("comment child leaves the value intact", func(t *testing.T) {
		doc := signedDocPaddedValue(t, key, "SignatureValue", "<!--QUJDREVG-->")
		result, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.NoError(t, err)
		require.Len(t, result.References, 1)
	})

	// An empty element child adds no characters, so a value that silently
	// accepted it would verify exactly as the unmodified one does. The value
	// must be refused for its shape instead.
	t.Run("element child is refused", func(t *testing.T) {
		doc := signedDocPaddedValue(t, key, "SignatureValue", "<x/>")
		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidSignature)
		require.ErrorContains(t, err, "SignatureValue")
	})

	// A conforming document may write a base64 value as an entity reference,
	// and both of these values are unchanged by it: the entity carries exactly
	// the characters the element held. Verification must therefore succeed
	// exactly as it does without the entity, at the DigestValue site (inside
	// ds:SignedInfo, so it is itself signed) and at the SignatureValue site.
	t.Run("entity-reference DigestValue verifies", func(t *testing.T) {
		doc := signedDocEntityValue(t, key, "DigestValue", "")
		result, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.NoError(t, err)
		require.Len(t, result.References, 1)
	})

	t.Run("entity-reference SignatureValue verifies", func(t *testing.T) {
		doc := signedDocEntityValue(t, key, "SignatureValue", "")
		result, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.NoError(t, err)
		require.Len(t, result.References, 1)
	})

	// The referenced entity's replacement text is read raw and one step deep.
	// A nested reference inside it stays the literal "&inner;" and markup stays
	// the literal "<x/>", so neither is expanded into the value and neither
	// decodes — the read never leaves the declared literal.
	t.Run("nested entity reference does not decode", func(t *testing.T) {
		doc := signedDocEntityValue(t, key, "SignatureValue", "&inner;")
		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidSignature)
		require.ErrorContains(t, err, "SignatureValue")
	})

	t.Run("entity holding markup does not decode", func(t *testing.T) {
		doc := signedDocEntityValue(t, key, "SignatureValue", "<x/>")
		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidSignature)
		require.ErrorContains(t, err, "SignatureValue")
	})
}

// signedDocEntityValue builds the same detached signature signedDocPaddedValue
// does, then rewrites the named ds: element's whole content into a single
// reference to an internal-subset entity and reparses. helium's parser does not
// substitute entities, so the value reaches verification as one EntityRefNode
// child — the shape a document that writes its base64 through an entity
// actually arrives in.
//
// declared is what that entity declares. Empty means the element's own text
// verbatim, which is the conforming case: the value's characters do not change,
// so the signature is still the same signature. A non-empty declared
// substitutes other replacement text, for the shapes that must not decode.
func signedDocEntityValue(t *testing.T, key *rsa.PrivateKey, local, declared string) *helium.Document {
	t.Helper()
	serialized, err := helium.WriteString(signedDocPaddedValue(t, key, local, ""))
	require.NoError(t, err)

	open := "<ds:" + local + ">"
	openAt := strings.Index(serialized, open)
	require.Positive(t, openAt, "signed document has no ds:%s", local)
	start := openAt + len(open)
	end := strings.Index(serialized, "</ds:"+local+">")
	require.Greater(t, end, start, "ds:%s has no text", local)
	if declared == "" {
		declared = serialized[start:end]
	}

	// The internal subset goes before the document element, and "inner" is
	// declared so a declared value of "&inner;" is a reference to a real entity
	// rather than a parse error — what must not happen is its EXPANSION.
	rootAt := strings.Index(serialized, "<doc>")
	require.Positive(t, rootAt, "signed document has no doc element")
	return mustParseXML(t, serialized[:rootAt]+
		`<!DOCTYPE doc [<!ENTITY inner "QUJD"><!ENTITY dv "`+declared+`">]>`+
		serialized[rootAt:start]+"&dv;"+serialized[end:])
}

// splitIntoNodes lays out s as CDATA sections of chunk characters each, so the
// value reaches the parser as that many separate children. A per-node content
// cap bounds one indivisible run, and every section is its own run, so this is
// how lexical text evades that cap.
func splitIntoNodes(s string, chunk int) string {
	var b strings.Builder
	for i := 0; i < len(s); i += chunk {
		b.WriteString(`<![CDATA[` + s[i:min(i+chunk, len(s))] + `]]>`)
	}
	return b.String()
}

// splitIntoCDATA lays s out as parts CDATA sections.
func splitIntoCDATA(s string, parts int) string {
	return splitIntoNodes(s, (len(s)+parts-1)/parts)
}

// signedDocPaddedValue builds a detached signature over the #x subtree, then
// appends markup directly after the text of the named ds: element and reparses,
// so that element's value arrives as however many children the markup produces.
//
// The reference covers only the #x subtree, so no transform ever canonicalizes
// or copies the Signature itself: what the padding costs is then the base64
// handling alone, which is what these bounds are about. (A whole-document
// enveloped signature would additionally deep-copy the padding for the
// enveloped-signature transform, a document-sized cost unrelated to base64.)
//
// Padding ds:SignatureValue leaves a VALID signature: XML whitespace is part of
// the xs:base64Binary lexical space, so the value still decodes to the same
// octets, and SignatureValue sits outside both ds:SignedInfo and the referenced
// subtree, so nothing that is digested changes. Padding ds:DigestValue breaks
// the signature, which is why only over-budget (rejected) cases use it.
func signedDocPaddedValue(t *testing.T, key *rsa.PrivateKey, local, markup string) *helium.Document {
	t.Helper()
	doc := mustParseXML(t, `<doc><target Id="x"><v>signed content</v></target></doc>`)
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#x",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		})
	sigElem, err := signer.SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	serialized, err := helium.WriteString(doc)
	require.NoError(t, err)
	end := strings.Index(serialized, "</ds:"+local+">")
	require.Positive(t, end, "signed document has no ds:%s", local)
	return mustParseXML(t, serialized[:end]+markup+serialized[end:])
}

// TestVerifyDecodedBytesAllocation pins what a base64 value inside a Signature
// may ALLOCATE, which the MaxDecodedBytes error assertions above cannot see.
// The cap governs decoded bytes, but the lexical text an attacker wraps around
// them is unbounded: xs:base64Binary permits XML whitespace between characters,
// and the value may be spread over as many text and CDATA children as the
// document likes. Joining that text into one string before the cap is charged
// makes the cap an accounting formality — the memory is allocated by the time
// the error is returned — and for a value the cap ACCEPTS it is never refused at
// all, so a test that only checks for the rejection would miss half of it.
//
// Each case reads the process-wide TotalAlloc delta across Verify, so these
// subtests must NOT run in parallel: a concurrent test's allocations would
// pollute the delta.
func TestVerifyDecodedBytesAllocation(t *testing.T) {
	// no t.Parallel(): isolated so each delta reflects only its own Verify.

	// whitespace is the padding the attacker writes around the value, and every
	// bound below is a multiple of it, because the defect being pinned is
	// exactly a cost that follows the lexical length. Only space and tab are
	// used: an XML parser folds CRLF to LF, which would make the text the DOM
	// holds shorter than the text written here and every multiple below harder
	// to read.
	const whitespace = 4 << 20
	padding := strings.Repeat(" \t", whitespace/2)

	// Reading each child's content is the floor a bound has to clear: a DOM
	// hands out a copy per node, and no value can be counted without looking at
	// it. A rejected value is looked at once, and an accepted one twice — once
	// to count it and once to build the characters the count approved.
	//
	// A child that contributes no character data — a comment, or the element
	// child xs:base64Binary does not admit at all — is never read, so nothing
	// scaling with its text is allocated. That bound is a small fraction of the
	// padding rather than a multiple of it.
	const (
		countOnly     = whitespace * 3 / 2
		countAndBuild = whitespace * 5 / 2
		neverRead     = whitespace / 4
	)

	key := generateRSAKey(t)

	for _, tc := range []struct {
		name string
		// local names the ds: element whose value carries the padding.
		local string
		// markup is appended after that element's text.
		markup string
		// maxDecoded is the MaxDecodedBytes cap; 0 leaves the default.
		maxDecoded int
		// wantErr is the sentinel Verify must report; nil means the padded
		// document must still verify.
		wantErr  error
		maxAlloc uint64
	}{
		{
			// A one-byte cap refuses the very first DigestValue, so the
			// whitespace after it is never legitimately needed.
			name:       "over-budget DigestValue with whitespace in one text node",
			local:      "DigestValue",
			markup:     padding,
			maxDecoded: 1,
			wantErr:    xmldsig1.ErrResourceLimitExceeded,
			maxAlloc:   countOnly,
		},
		{
			// CDATA is how the whitespace evades the parser's per-node content
			// cap: the cap bounds one indivisible run, and every section is its
			// own.
			name:       "over-budget DigestValue with whitespace split across CDATA sections",
			local:      "DigestValue",
			markup:     splitIntoCDATA(padding, 16),
			maxDecoded: 1,
			wantErr:    xmldsig1.ErrResourceLimitExceeded,
			maxAlloc:   countOnly,
		},
		{
			// Nothing here is ever refused: the signature is valid and its
			// decoded size is far under the default cap, while the whitespace
			// after it is not charged at all. Allocating for it would be
			// unbounded amplification with no error anywhere to notice it,
			// which a rejection-only test would never see.
			name:     "accepted SignatureValue with whitespace split across CDATA sections",
			local:    "SignatureValue",
			markup:   splitIntoCDATA(padding, 16),
			maxAlloc: countAndBuild,
		},
		{
			// An element child is where the lexical length stops being the
			// attacker's only lever: one element's content is the concatenation
			// of its whole subtree, so reading it aggregates that subtree into a
			// single buffer. It contributes no character data to an
			// xs:base64Binary value, so the value is refused without ever being
			// read and the padding costs nothing.
			name:     "SignatureValue with whitespace in an element child",
			local:    "SignatureValue",
			markup:   "<pad>" + padding + "</pad>",
			wantErr:  xmldsig1.ErrInvalidSignature,
			maxAlloc: neverRead,
		},
		{
			// The same aggregation reached through the DigestValue parse, which
			// is a different call site inside SignedInfo.
			name:     "DigestValue with whitespace in an element child",
			local:    "DigestValue",
			markup:   "<pad>" + padding + "</pad>",
			wantErr:  xmldsig1.ErrInvalidSignature,
			maxAlloc: neverRead,
		},
		{
			// A comment contributes no character data either, so the signature
			// stays valid and the comment's text is never read.
			name:     "accepted SignatureValue with whitespace in a comment child",
			local:    "SignatureValue",
			markup:   "<!--" + padding + "-->",
			maxAlloc: neverRead,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := signedDocPaddedValue(t, key, tc.local, tc.markup)
			verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
			if tc.maxDecoded != 0 {
				verifier = verifier.MaxDecodedBytes(tc.maxDecoded)
			}

			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			result, err := verifier.Verify(t.Context(), doc)
			runtime.ReadMemStats(&after)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			}
			if tc.wantErr == nil {
				// The padded value still verifies byte for byte, so the
				// whitespace is genuinely part of a legal signature and the
				// bound below is not measuring an early rejection.
				require.NoError(t, err)
				require.Len(t, result.References, 1)
			}

			allocated := after.TotalAlloc - before.TotalAlloc
			require.Less(t, allocated, tc.maxAlloc, "verifying %d lexical bytes of padding allocated %d bytes", len(tc.markup), allocated)
		})
	}
}
