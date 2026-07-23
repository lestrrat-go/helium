// Package transform provides plug-in transform implementations for the xmldsig1
// verifier's injected seams.
//
// XML Signature (xmldsig-core §4.3.3.4) lets a ds:Reference chain a pipeline of
// ds:Transform algorithms — canonicalization, base64, the XPath filter, the
// enveloped-signature transform, and XSLT. helium's xmldsig1 runs all of those
// NATIVELY except one: XSLT is a full transformation language, so honoring it
// means running an attacker-supplied stylesheet through a complete XSLT engine.
// xmldsig1 refuses to pull that engine into the security-critical signature
// package or to execute attacker XSLT by default; instead it exposes an injected
// [github.com/lestrrat-go/helium/xmldsig1.XSLTTransformer] seam that fails closed
// unless the embedder supplies an implementation.
//
// This package is that implementation, kept out of xmldsig1 so the core keeps its
// minimal dependency set. It holds ONLY [XSLT]: because every other transform is
// native, there is no transform.C14N or transform.Base64 — the package name names
// the domain (injected xmldsig1 transforms), not a family that will grow.
//
// # Security
//
// On the verify path BOTH the stylesheet and its input are attacker-controlled.
// Enabling XSLT means accepting that risk: a malicious signature can carry a
// stylesheet that consumes unbounded CPU/memory or produces enormous output.
// [XSLT] parses with helium's locked-down parser (external entities/DTD/network
// blocked) and honors context cancellation, so bound the context with a deadline
// and treat wiring it in as the explicit point where you accept running
// attacker-controlled XSLT.
package transform

import (
	"bytes"
	"context"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/lestrrat-go/helium/xslt3"
)

// XSLT is an [xmldsig1.XSLTTransformer] backed by helium's xslt3 engine. Pass it
// to [xmldsig1.Verifier.XSLTTransformer] to honor an XSLT transform on a
// ds:Reference; xmldsig1 never runs XSLT unless you do.
//
// The zero value is ready to use. It parses the stylesheet and input with
// helium's default (locked-down) parser and applies the stylesheet with xslt3's
// hardened defaults, relying on the caller's context deadline for a time bound.
// See the package documentation for the security responsibilities this carries.
type XSLT struct{}

var _ xmldsig1.XSLTTransformer = XSLT{}

// TransformXSLT compiles stylesheet and applies it to input, returning the
// transform output to the ordered Reference pipeline. xslt3 removes
// helium.Writer's document terminator on the writer path that creates it, so
// every trailing newline reaching this adapter is result content and is
// preserved. Both arguments are the current pipeline octets xmldsig1 hands to
// the seam; neither is trusted, so ctx should carry a deadline. One Reference may
// invoke this method multiple times.
func (XSLT) TransformXSLT(ctx context.Context, stylesheet, input []byte) ([]byte, error) {
	ssDoc, err := helium.NewParser().Parse(ctx, stylesheet)
	if err != nil {
		return nil, err
	}
	ss, err := xslt3.CompileStylesheet(ctx, ssDoc)
	if err != nil {
		return nil, err
	}
	srcDoc, err := helium.NewParser().Parse(ctx, input)
	if err != nil {
		return nil, err
	}
	invocation := ss.Transform(srcDoc)
	resultDoc, err := invocation.Do(ctx)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := xslt3.SerializeResult(&buf, resultDoc, invocation.ResolvedOutputDef()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
