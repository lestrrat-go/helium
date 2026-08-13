package xmldsig1_test

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// xsltReferenceDoc is a document whose single ds:Reference declares the XSLT
// transform, so parsing the ds:Signature canonicalizes the ds:Transform's
// xsl:stylesheet subtree. The signature is not valid and does not need to be:
// the stylesheet is serialized while the signature element is parsed, which is
// before KeySource resolution and before the SignatureValue is checked.
const xsltReferenceDoc = `<root xmlns="urn:helium:test">` +
	`<payload Id="p1">covered</payload>` +
	`<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
	`<ds:SignedInfo>` +
	`<ds:CanonicalizationMethod Algorithm="` + xmldsig1.ExcC14N10 + `"/>` +
	`<ds:SignatureMethod Algorithm="` + xmldsig1.AlgRSASHA256 + `"/>` +
	`<ds:Reference URI="#p1">` +
	`<ds:Transforms>` +
	`<ds:Transform Algorithm="` + xmldsig1.TransformXSLT + `">` +
	`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">` +
	`<xsl:template match="/"><out/></xsl:template>` +
	`</xsl:stylesheet>` +
	`</ds:Transform>` +
	`</ds:Transforms>` +
	`<ds:DigestMethod Algorithm="` + xmldsig1.DigestSHA256 + `"/>` +
	`<ds:DigestValue>AA==</ds:DigestValue>` +
	`</ds:Reference>` +
	`</ds:SignedInfo>` +
	`<ds:SignatureValue>AA==</ds:SignatureValue>` +
	`</ds:Signature></root>`

// canonicalizeCancelContext reports a live context to every caller until the
// canonicalization walk itself polls it, and a cancelled one from that poll
// onwards. Verification polls ctx.Err() on entry and once per scanned child
// element, so a plainly cancelled context short-circuits well before a
// ds:Transform stylesheet is serialized; arming on the first poll made from the
// canonicalization file (xmldsig1/transforms.go) places the cancellation exactly
// where parseXSLTTransform consumes canonicalizeSubtree's error. Err is the only
// cancellation signal that walk consults, so Done never becomes ready and the
// value is standalone rather than derived from the test's context.
type canonicalizeCancelContext struct {
	armed atomic.Bool
}

func (c *canonicalizeCancelContext) Deadline() (time.Time, bool) { return time.Time{}, false }

func (c *canonicalizeCancelContext) Done() <-chan struct{} { return nil }

func (c *canonicalizeCancelContext) Value(any) any { return nil }

func (c *canonicalizeCancelContext) Err() error {
	if c.armed.Load() {
		return context.Canceled
	}
	_, file, _, ok := runtime.Caller(1)
	if !ok || !strings.HasSuffix(filepath.ToSlash(file), "/xmldsig1/transforms.go") {
		return nil
	}
	c.armed.Store(true)
	return context.Canceled
}

// TestVerifyXSLTTransformContextError locks the contract that a cancelled or
// expired context surfacing while a ds:Reference's XSLT stylesheet is serialized
// reaches the caller as a context error, not as a malformed-transform error. The
// route needs no caller opt-in: the stylesheet is recorded at parse time whether
// or not the XSLT transform can later run, so neither an XSLTTransformer nor a
// ReferenceResolver is configured here.
func TestVerifyXSLTTransformContextError(t *testing.T) {
	key := generateRSAKey(t)

	// The same document under a live context reaches the XSLT transform and
	// fails closed on it, which is what proves the cancelled run below really
	// exercises the stylesheet serialization rather than stopping earlier.
	t.Run("live context reaches the XSLT transform", func(t *testing.T) {
		doc := mustParseXML(t, xsltReferenceDoc)
		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrUnsupportedTransform)
	})

	t.Run("cancellation surfaces as a context error", func(t *testing.T) {
		doc := mustParseXML(t, xsltReferenceDoc)
		ctx := &canonicalizeCancelContext{}
		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(ctx, doc)
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, xmldsig1.ErrUnsupportedTransform,
			"a cancelled context is the caller's, not a malformed transform")
	})
}
