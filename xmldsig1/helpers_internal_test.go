package xmldsig1

import (
	"bytes"
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/stretchr/testify/require"
)

// payloadFragment is the same-document fragment URI used by the internal test
// suite to reference the element carrying Id="payload".
const payloadFragment = "#payload"

// mustParse parses xml into a document, failing the test on error. Shared by the
// internal test suite.
func mustParse(t *testing.T, xml string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	return doc
}

// canonicalizeReference collects the octets writeReference streams for ref.
func canonicalizeReference(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem *helium.Element, ref parsedReference) (*helium.Element, []byte, bool, error) {
	var buf bytes.Buffer
	target, external, err := writeReference(ctx, cfg, doc, sigElem, ref, &buf)
	if err != nil {
		return nil, nil, false, err
	}
	return target, buf.Bytes(), external, nil
}

// signReferenceOctets collects the octets writeSignReference streams for ref.
func signReferenceOctets(ctx context.Context, cfg *signerConfig, doc *helium.Document, sigElem *helium.Element, ref ReferenceConfig, internalRoot *helium.Element) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeSignReference(ctx, cfg, doc, sigElem, ref, internalRoot, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// computeDigest hashes data with the digest algorithm algURI through the same
// sink signing and verification stream into.
func computeDigest(algURI string, data []byte, allowSHA1 bool) ([]byte, error) {
	sink, digester, err := newDigestSink(algURI, allowSHA1)
	if err != nil {
		return nil, err
	}
	if _, err := sink.Write(data); err != nil {
		return nil, err
	}
	return digester.sum()
}

// canonicalizeBytes collects what canonicalize writes.
func canonicalizeBytes(method string, doc *helium.Document, prefixes []string) ([]byte, error) {
	var buf bytes.Buffer
	if err := canonicalize(method, doc, prefixes, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// envelopedBytes collects what canonicalizeEnveloped writes.
func envelopedBytes(ctx context.Context, method string, doc *helium.Document, target, sigElem *helium.Element, wholeDoc bool, prefixes []string) ([]byte, error) {
	var buf bytes.Buffer
	if err := canonicalizeEnveloped(ctx, method, doc, target, sigElem, wholeDoc, prefixes, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// detachedSubtreeBytes collects what canonicalizeDetachedSubtree writes.
func detachedSubtreeBytes(ctx context.Context, method string, root, target *helium.Element, prefixes []string) ([]byte, error) {
	var buf bytes.Buffer
	if err := canonicalizeDetachedSubtree(ctx, method, root, target, prefixes, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// canonicalizeNodeSetMode canonicalizes nodes against doc by walking the whole
// document: the reference the subtree-start canonicalization is compared with.
func canonicalizeNodeSetMode(mode c14n.Mode, comments bool, nodes []helium.Node, doc *helium.Document, prefixes []string) ([]byte, error) {
	return newCanonicalizer(mode, comments, prefixes).NodeSet(nodes).CanonicalizeTo(doc)
}
